package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aiden/mutcheck/internal/domain"
	"github.com/aiden/mutcheck/internal/execution"
	"github.com/aiden/mutcheck/internal/mutation"
)

type Config struct {
	ProjectRoot    string
	SourcePath     string
	PackagePattern string
	Timeout        time.Duration
}

type App struct {
	runner execution.GoTestRunner
}

func New() App { return App{runner: execution.GoTestRunner{}} }

func (app App) Run(ctx context.Context, config Config) (domain.Summary, error) {
	source := filepath.Join(config.ProjectRoot, config.SourcePath)
	mutants, err := mutation.Discover(source)
	if err != nil {
		return domain.Summary{}, err
	}
	if err := app.runBaseline(ctx, config); err != nil {
		return domain.Summary{}, err
	}
	for index := range mutants {
		mutants[index].SourcePath = filepath.ToSlash(config.SourcePath)
	}

	summary := domain.Summary{ProjectRoot: config.ProjectRoot, SourcePath: filepath.ToSlash(config.SourcePath), PackagePattern: config.PackagePattern, Timeout: config.Timeout.String()}
	for _, mutant := range mutants {
		summary.Results = append(summary.Results, app.runMutant(ctx, config, mutant))
	}
	return summary, nil
}

func (app App) runBaseline(parent context.Context, config Config) error {
	workspace, cleanup, err := execution.CopyProject(config.ProjectRoot)
	if err != nil {
		return err
	}
	defer cleanup()
	ctx, cancel := execution.WithTimeout(parent, config.Timeout)
	defer cancel()
	run, err := app.runner.Run(ctx, workspace, config.PackagePattern)
	if err != nil {
		return fmt.Errorf("baseline test execution failed: %w", err)
	}
	if run.TimedOut {
		return fmt.Errorf("baseline tests exceeded the %s timeout", config.Timeout)
	}
	if run.ExitCode != 0 {
		return errorsForBaseline(run)
	}
	return nil
}

func errorsForBaseline(run execution.TestRun) error {
	if run.BuildFailed {
		return fmt.Errorf("baseline tests could not compile")
	}
	return fmt.Errorf("baseline tests failed; mutation score is not meaningful until they pass")
}

func (app App) runMutant(parent context.Context, config Config, mutant domain.Mutant) domain.Result {
	started := time.Now()
	result := domain.Result{Mutant: mutant}
	workspace, cleanup, err := execution.CopyProject(config.ProjectRoot)
	if err != nil {
		result.Status, result.Detail = domain.StatusError, "unable to create isolated workspace"
		result.Duration = time.Since(started)
		return result
	}
	defer cleanup()

	mutated, err := mutation.Apply(filepath.Join(workspace, config.SourcePath), mutant.CandidateIndex)
	if err != nil {
		result.Status, result.Detail = domain.StatusInvalid, "unable to generate valid mutated source"
		result.Duration = time.Since(started)
		return result
	}
	if err := os.WriteFile(filepath.Join(workspace, config.SourcePath), mutated, 0o644); err != nil {
		result.Status, result.Detail = domain.StatusError, "unable to write mutated source in workspace"
		result.Duration = time.Since(started)
		return result
	}

	ctx, cancel := execution.WithTimeout(parent, config.Timeout)
	defer cancel()
	run, err := app.runner.Run(ctx, workspace, config.PackagePattern)
	result.Duration = time.Since(started)
	if err != nil {
		result.Status, result.Detail = domain.StatusError, "unable to execute go test"
		return result
	}
	if run.TimedOut {
		result.Status, result.Detail = domain.StatusTimeout, "go test exceeded timeout"
		return result
	}
	if run.ExitCode == 0 {
		result.Status = domain.StatusSurvived
		return result
	}
	if run.TestFailed {
		result.Status = domain.StatusKilled
		return result
	}
	if run.BuildFailed {
		result.Status, result.Detail = domain.StatusInvalid, "mutated project did not compile"
		return result
	}
	result.Status, result.Detail = domain.StatusError, "go test failed before reporting a test failure"
	return result
}

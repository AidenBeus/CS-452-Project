package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aiden/mutcheck/internal/app"
	"github.com/aiden/mutcheck/internal/report"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, "usage: mutcheck run --source <file.go> [--project <dir>] [--packages <pattern>] [--timeout <duration>] [--report <file>]")
		os.Exit(2)
	}

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	project := flags.String("project", ".", "Go module root")
	source := flags.String("source", "", "Go source file relative to --project (required)")
	packages := flags.String("packages", "./...", "package pattern passed to go test")
	timeout := flags.Duration("timeout", 30*time.Second, "timeout for each go test invocation")
	reportPath := flags.String("report", "", "optional path for a JSON report")
	verbose := flags.Bool("verbose", false, "show every mutant instead of survivors only")
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	config, err := buildConfig(*project, *source, *packages, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mutcheck:", err)
		os.Exit(2)
	}

	summary, err := app.New().Run(context.Background(), config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mutcheck:", err)
		os.Exit(1)
	}

	report.PrintTerminal(os.Stdout, summary, report.Options{Verbose: *verbose, Color: report.SupportsColor(os.Stdout)})
	if *reportPath != "" {
		if err := report.WriteJSON(*reportPath, summary); err != nil {
			fmt.Fprintln(os.Stderr, "mutcheck: write report:", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "JSON report:", *reportPath)
	}
}

func buildConfig(project, source, packages string, timeout time.Duration) (app.Config, error) {
	if source == "" {
		return app.Config{}, errors.New("--source is required")
	}
	if timeout <= 0 {
		return app.Config{}, errors.New("--timeout must be greater than zero")
	}
	root, err := filepath.Abs(project)
	if err != nil {
		return app.Config{}, fmt.Errorf("resolve project path: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return app.Config{}, errors.New("--project must be an existing directory")
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return app.Config{}, errors.New("--project must contain a go.mod file")
	}
	var relative string
	if filepath.IsAbs(source) {
		relative, err = filepath.Rel(root, source)
	} else {
		relative = filepath.Clean(source)
	}
	if err != nil || relative == "." || relative == ".." || (len(relative) >= 3 && relative[:3] == ".."+string(filepath.Separator)) {
		return app.Config{}, errors.New("--source must be inside --project")
	}
	if filepath.Ext(relative) != ".go" || filepath.Base(relative) == "" {
		return app.Config{}, errors.New("--source must name a .go file")
	}
	if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
		return app.Config{}, errors.New("--source must be an existing file")
	}
	if packages == "" {
		return app.Config{}, errors.New("--packages cannot be empty")
	}
	return app.Config{ProjectRoot: root, SourcePath: relative, PackagePattern: packages, Timeout: timeout}, nil
}

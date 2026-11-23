package report

import (
	"strings"
	"testing"

	"github.com/aiden/mutcheck/internal/domain"
)

func TestPrintTerminalShowsScoreBarAndSurvivorsByDefault(t *testing.T) {
	summary := sampleSummary()
	var output strings.Builder
	PrintTerminal(&output, summary, Options{})
	text := output.String()
	for _, expected := range []string{"Mutation score:  50.0% [##########----------]", "Outcome", "SURVIVED", "M002", "age.go:4:9", ">= -> >"} {
		if !strings.Contains(text, expected) {
			t.Errorf("output does not contain %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "M001") {
		t.Errorf("default report included killed mutant:\n%s", text)
	}
}

func TestPrintTerminalVerboseIncludesEveryMutant(t *testing.T) {
	var output strings.Builder
	PrintTerminal(&output, sampleSummary(), Options{Verbose: true})
	text := output.String()
	for _, expected := range []string{"All mutations", "M001", "M002", "KILLED", "SURVIVED"} {
		if !strings.Contains(text, expected) {
			t.Errorf("verbose output does not contain %q:\n%s", expected, text)
		}
	}
}

func TestScoreBarBoundsScore(t *testing.T) {
	if scoreBar(-1) != "--------------------" {
		t.Fatalf("negative score bar = %q", scoreBar(-1))
	}
	if scoreBar(101) != "####################" {
		t.Fatalf("large score bar = %q", scoreBar(101))
	}
}

func sampleSummary() domain.Summary {
	return domain.Summary{
		SourcePath:     "age.go",
		PackagePattern: "./...",
		Results: []domain.Result{
			{Mutant: domain.Mutant{ID: "M001", SourcePath: "age.go", Line: 8, Column: 9, Original: "==", Replacement: "!="}, Status: domain.StatusKilled},
			{Mutant: domain.Mutant{ID: "M002", SourcePath: "age.go", Line: 4, Column: 9, Original: ">=", Replacement: ">"}, Status: domain.StatusSurvived},
		},
	}
}

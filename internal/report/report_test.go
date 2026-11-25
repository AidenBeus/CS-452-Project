package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aiden/mutcheck/internal/domain"
)

func TestPrintTerminalShowsScoreBarAndSurvivorsByDefault(t *testing.T) {
	summary := sampleSummary()
	var output strings.Builder
	PrintTerminal(&output, summary, Options{})
	text := output.String()
	for _, expected := range []string{"Mutation score:  50.0% [##########----------]", "Outcome", "SURVIVED", "M002", "age.go:4:12", ">= -> >", "Source context", "- return age >= 18", "+ return age > 18"} {
		if !strings.Contains(text, expected) {
			t.Errorf("output does not contain %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "M001") {
		t.Errorf("default report included killed mutant:\n%s", text)
	}
}

func TestSourceContextIsExcludedFromJSON(t *testing.T) {
	encoded, err := json.Marshal(sampleSummary())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "return age >= 18") {
		t.Fatalf("JSON report includes source context: %s", encoded)
	}
}

func TestWriteGitHubSummaryIsBoundedAndExcludesSourceContext(t *testing.T) {
	var output strings.Builder
	WriteGitHubSummary(&output, sampleSummary())
	text := output.String()
	for _, expected := range []string{"## Mutation Testing", "**Mutation score:** 50.0%", "| 1 | 1 | 0 | 0 | 0 |", "<code>M002</code>", "<code>age.go:4</code>", "<code>&gt;=</code> → <code>&gt;</code>"} {
		if !strings.Contains(text, expected) {
			t.Errorf("GitHub summary does not contain %q:\n%s", expected, text)
		}
	}
	for _, unexpected := range []string{"M001", "return age >= 18"} {
		if strings.Contains(text, unexpected) {
			t.Errorf("GitHub summary includes %q:\n%s", unexpected, text)
		}
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
			{Mutant: domain.Mutant{ID: "M002", SourcePath: "age.go", Line: 4, Column: 12, Original: ">=", Replacement: ">", SourceLine: "return age >= 18", MutatedLine: "return age > 18"}, Status: domain.StatusSurvived},
		},
	}
}

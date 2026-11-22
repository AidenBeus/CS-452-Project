package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aiden/mutcheck/internal/domain"
)

func PrintTerminal(writer io.Writer, summary domain.Summary) {
	fmt.Fprintln(writer, "Mutation Testing Results")
	fmt.Fprintln(writer, "------------------------")
	fmt.Fprintln(writer, "Total mutants:", len(summary.Results))
	fmt.Fprintln(writer, "Killed:", summary.Count(domain.StatusKilled))
	fmt.Fprintln(writer, "Survived:", summary.Count(domain.StatusSurvived))
	fmt.Fprintln(writer, "Timeout:", summary.Count(domain.StatusTimeout))
	fmt.Fprintln(writer, "Invalid:", summary.Count(domain.StatusInvalid))
	fmt.Fprintln(writer, "Error:", summary.Count(domain.StatusError))
	if score, ok := summary.Score(); ok {
		fmt.Fprintf(writer, "Mutation score: %.1f%%\n", score)
	} else {
		fmt.Fprintln(writer, "Mutation score: N/A (no killed or survived mutants)")
	}
	for _, result := range summary.Results {
		if result.Status == domain.StatusSurvived {
			fmt.Fprintf(writer, "Survived: %s %s:%d:%d %s -> %s\n", result.Mutant.ID, result.Mutant.SourcePath, result.Mutant.Line, result.Mutant.Column, result.Mutant.Original, result.Mutant.Replacement)
		}
	}
}

func WriteJSON(path string, summary domain.Summary) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

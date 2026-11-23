package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aiden/mutcheck/internal/domain"
)

const scoreBarWidth = 20

type Options struct {
	Verbose bool
	Color   bool
}

// SupportsColor enables ANSI colors only for an interactive terminal that has not opted out.
func SupportsColor(file *os.File) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func PrintTerminal(writer io.Writer, summary domain.Summary, options Options) {
	fmt.Fprintln(writer, "Mutation Testing Results")
	fmt.Fprintln(writer, strings.Repeat("=", 24))
	fmt.Fprintf(writer, "Source: %s    Packages: %s\n\n", summary.SourcePath, summary.PackagePattern)
	if score, ok := summary.Score(); ok {
		fmt.Fprintf(writer, "Mutation score: %5.1f%% [%s]\n", score, scoreBar(score))
	} else {
		fmt.Fprintln(writer, "Mutation score: N/A (no killed or survived mutants)")
	}
	fmt.Fprintf(writer, "Mutants: %d\n", len(summary.Results))
	printOutcomeTable(writer, summary, options.Color)

	results := displayResults(summary.Results, options.Verbose)
	if len(results) == 0 {
		fmt.Fprintln(writer, "\nSurviving mutations: none")
		return
	}
	if options.Verbose {
		fmt.Fprintln(writer, "\nAll mutations")
		printMutantTable(writer, results, true, options.Color)
		return
	}
	fmt.Fprintln(writer, "\nSurviving mutations")
	printMutantTable(writer, results, false, options.Color)
}

func scoreBar(score float64) string {
	filled := int(score/100*scoreBarWidth + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > scoreBarWidth {
		filled = scoreBarWidth
	}
	return strings.Repeat("#", filled) + strings.Repeat("-", scoreBarWidth-filled)
}

func printOutcomeTable(writer io.Writer, summary domain.Summary, color bool) {
	statuses := []domain.Status{domain.StatusKilled, domain.StatusSurvived, domain.StatusTimeout, domain.StatusInvalid, domain.StatusError}
	rows := make([][]string, 0, len(statuses))
	for _, status := range statuses {
		rows = append(rows, []string{strings.ToUpper(string(status)), fmt.Sprint(summary.Count(status))})
	}
	printTable(writer, []string{"Outcome", "Count"}, rows, color, 0)
}

func displayResults(results []domain.Result, verbose bool) []domain.Result {
	if verbose {
		return results
	}
	survivors := make([]domain.Result, 0)
	for _, result := range results {
		if result.Status == domain.StatusSurvived {
			survivors = append(survivors, result)
		}
	}
	return survivors
}

func printMutantTable(writer io.Writer, results []domain.Result, includeStatus, color bool) {
	headers := []string{"ID", "Location", "Change"}
	rows := make([][]string, 0, len(results))
	if includeStatus {
		headers = append(headers, "Status")
	}
	for _, result := range results {
		row := []string{
			result.Mutant.ID,
			fmt.Sprintf("%s:%d:%d", result.Mutant.SourcePath, result.Mutant.Line, result.Mutant.Column),
			result.Mutant.Original + " -> " + result.Mutant.Replacement,
		}
		if includeStatus {
			row = append(row, strings.ToUpper(string(result.Status)))
		}
		rows = append(rows, row)
	}
	statusColumn := -1
	if includeStatus {
		statusColumn = len(headers) - 1
	}
	printTable(writer, headers, rows, color, statusColumn)
}

func printTable(writer io.Writer, headers []string, rows [][]string, color bool, statusColumn int) {
	widths := make([]int, len(headers))
	for index, header := range headers {
		widths[index] = len(header)
	}
	for _, row := range rows {
		for index, value := range row {
			if len(value) > widths[index] {
				widths[index] = len(value)
			}
		}
	}
	writeRow(writer, headers, widths, false, -1)
	divider := make([]string, len(widths))
	for index, width := range widths {
		divider[index] = strings.Repeat("-", width)
	}
	writeRow(writer, divider, widths, false, -1)
	for _, row := range rows {
		writeRow(writer, row, widths, color, statusColumn)
	}
}

func writeRow(writer io.Writer, values []string, widths []int, color bool, statusColumn int) {
	formatted := make([]string, len(values))
	for index, value := range values {
		formatted[index] = fmt.Sprintf("%-*s", widths[index], value)
		if color && index == statusColumn {
			formatted[index] = colorStatus(value, formatted[index])
		}
	}
	fmt.Fprintln(writer, strings.Join(formatted, "  "))
}

func colorStatus(status, value string) string {
	code := ""
	switch strings.ToLower(status) {
	case string(domain.StatusKilled):
		code = "32" // green
	case string(domain.StatusSurvived):
		code = "33" // yellow
	case string(domain.StatusTimeout), string(domain.StatusInvalid), string(domain.StatusError):
		code = "31" // red
	}
	if code == "" {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
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

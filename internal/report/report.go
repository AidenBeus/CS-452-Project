package report

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aiden/mutcheck/internal/domain"
)

const (
	scoreBarWidth       = 20
	maxSummarySurvivors = 10
)

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
	} else {
		fmt.Fprintln(writer, "\nSurviving mutations")
		printMutantTable(writer, results, false, options.Color)
	}
	printSourceContext(writer, results)
}

func printSourceContext(writer io.Writer, results []domain.Result) {
	printedHeader := false
	for _, result := range results {
		if result.Mutant.SourceLine == "" || result.Mutant.MutatedLine == "" {
			continue
		}
		if !printedHeader {
			fmt.Fprintln(writer, "\nSource context")
			printedHeader = true
		}
		location := fmt.Sprintf("%s:%d", result.Mutant.SourcePath, result.Mutant.Line)
		fmt.Fprintf(writer, "%s %s\n", result.Mutant.ID, location)
		fmt.Fprintf(writer, "  - %s\n", safeSourceLine(result.Mutant.SourceLine))
		fmt.Fprintf(writer, "  + %s\n", safeSourceLine(result.Mutant.MutatedLine))
	}
}

func safeSourceLine(line string) string {
	line = strings.ReplaceAll(line, "\t", "    ")
	line = strings.Map(func(character rune) rune {
		if character < 32 || character == 127 {
			return -1
		}
		return character
	}, line)
	const maxLength = 200
	if len(line) > maxLength {
		return line[:maxLength-3] + "..."
	}
	return line
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

// ReadJSON reads a report previously written by WriteJSON.
func ReadJSON(path string) (domain.Summary, error) {
	file, err := os.Open(path)
	if err != nil {
		return domain.Summary{}, err
	}
	defer file.Close()
	var summary domain.Summary
	if err := json.NewDecoder(file).Decode(&summary); err != nil {
		return domain.Summary{}, err
	}
	return summary, nil
}

// WriteGitHubSummary renders a bounded, source-free GitHub-flavored Markdown summary.
func WriteGitHubSummary(writer io.Writer, summary domain.Summary) {
	fmt.Fprintln(writer, "## Mutation Testing")
	if score, ok := summary.Score(); ok {
		fmt.Fprintf(writer, "**Mutation score:** %.1f%%\n\n", score)
	} else {
		fmt.Fprintln(writer, "**Mutation score:** N/A (no killed or survived mutants)")
		fmt.Fprintln(writer)
	}
	fmt.Fprintln(writer, "| Killed | Survived | Timeout | Invalid | Error |")
	fmt.Fprintln(writer, "| ---: | ---: | ---: | ---: | ---: |")
	fmt.Fprintf(writer, "| %d | %d | %d | %d | %d |\n", summary.Count(domain.StatusKilled), summary.Count(domain.StatusSurvived), summary.Count(domain.StatusTimeout), summary.Count(domain.StatusInvalid), summary.Count(domain.StatusError))

	survivors := displayResults(summary.Results, false)
	if len(survivors) == 0 {
		fmt.Fprintln(writer, "\n### Surviving mutations\n\nNone.")
		return
	}
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "### Surviving mutations")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "| ID | Location | Change |")
	fmt.Fprintln(writer, "| --- | --- | --- |")
	limit := len(survivors)
	if limit > maxSummarySurvivors {
		limit = maxSummarySurvivors
	}
	for _, result := range survivors[:limit] {
		location := fmt.Sprintf("%s:%d", result.Mutant.SourcePath, result.Mutant.Line)
		fmt.Fprintf(writer, "| %s | %s | %s → %s |\n", markdownCode(result.Mutant.ID), markdownCode(location), markdownCode(result.Mutant.Original), markdownCode(result.Mutant.Replacement))
	}
	if remaining := len(survivors) - limit; remaining > 0 {
		fmt.Fprintf(writer, "\n_And %d additional surviving mutation(s); see the JSON artifact for the complete list._\n", remaining)
	}
}

func markdownCode(value string) string {
	value = strings.Map(func(character rune) rune {
		if character < 32 || character == 127 {
			return ' '
		}
		return character
	}, value)
	value = html.EscapeString(value)
	value = strings.ReplaceAll(value, "|", "&#124;")
	return "<code>" + value + "</code>"
}

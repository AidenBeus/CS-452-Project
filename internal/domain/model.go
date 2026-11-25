package domain

import "time"

// Status is the result of evaluating one mutant.
type Status string

const (
	StatusKilled   Status = "killed"
	StatusSurvived Status = "survived"
	StatusTimeout  Status = "timeout"
	StatusInvalid  Status = "invalid"
	StatusError    Status = "error"
)

// Mutant describes one deliberate, single source-code change.
type Mutant struct {
	ID             string `json:"id"`
	CandidateIndex int    `json:"candidate_index"`
	SourcePath     string `json:"source_path"`
	Line           int    `json:"line"`
	Column         int    `json:"column"`
	Operator       string `json:"operator"`
	Original       string `json:"original"`
	Replacement    string `json:"replacement"`
	// SourceLine and MutatedLine are terminal-only context. They are excluded from
	// JSON reports so reports do not include target-project source code.
	SourceLine  string `json:"-"`
	MutatedLine string `json:"-"`
}

type Result struct {
	Mutant   Mutant        `json:"mutant"`
	Status   Status        `json:"status"`
	Duration time.Duration `json:"duration_nanoseconds"`
	Detail   string        `json:"detail,omitempty"`
}

type Summary struct {
	ProjectRoot    string   `json:"project_root"`
	SourcePath     string   `json:"source_path"`
	PackagePattern string   `json:"package_pattern"`
	Timeout        string   `json:"timeout"`
	Results        []Result `json:"results"`
}

func (s Summary) Count(status Status) int {
	count := 0
	for _, result := range s.Results {
		if result.Status == status {
			count++
		}
	}
	return count
}

func (s Summary) Score() (float64, bool) {
	killed, survived := s.Count(StatusKilled), s.Count(StatusSurvived)
	if killed+survived == 0 {
		return 0, false
	}
	return float64(killed) / float64(killed+survived) * 100, true
}

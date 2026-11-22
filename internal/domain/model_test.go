package domain

import (
	"math"
	"testing"
)

func TestScoreExcludesNonTestOutcomes(t *testing.T) {
	summary := Summary{Results: []Result{
		{Status: StatusKilled}, {Status: StatusKilled}, {Status: StatusSurvived},
		{Status: StatusTimeout}, {Status: StatusInvalid}, {Status: StatusError},
	}}
	score, ok := summary.Score()
	if !ok || math.Abs(score-200.0/3.0) > 0.000001 {
		t.Fatalf("Score() = %v, %v; want 66.666... and true", score, ok)
	}
}

func TestScoreIsUnavailableWithoutKilledOrSurvivedMutants(t *testing.T) {
	score, ok := (Summary{Results: []Result{{Status: StatusTimeout}}}).Score()
	if ok || score != 0 {
		t.Fatalf("Score() = %v, %v; want 0, false", score, ok)
	}
}

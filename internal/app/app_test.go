package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aiden/mutcheck/internal/domain"
)

func TestRunClassifiesKilledAndSurvivingMutants(t *testing.T) {
	project := filepath.Join("testdata", "sample")
	sourcePath := filepath.Join(project, "age.go")
	before, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := New().Run(context.Background(), Config{
		ProjectRoot: project, SourcePath: "age.go", PackagePattern: "./...", Timeout: 20 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if summary.Count(domain.StatusKilled) != 1 {
		t.Fatalf("killed = %d, want 1; results: %#v", summary.Count(domain.StatusKilled), summary.Results)
	}
	if summary.Count(domain.StatusSurvived) != 2 {
		t.Fatalf("survived = %d, want 2; results: %#v", summary.Count(domain.StatusSurvived), summary.Results)
	}
	after, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("mutation run changed the original project source")
	}
}

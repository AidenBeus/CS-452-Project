package mutation

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const allOperatorsSource = `package sample

func operators(a, b int, left, right bool) bool {
	_ = a > b
	_ = a >= b
	_ = a < b
	_ = a <= b
	_ = a == b
	_ = a != b
	_ = a + b
	_ = a - b
	_ = left && right
	_ = left || right
	return true && !false
}
`

func TestDiscoverFindsEachSupportedOperator(t *testing.T) {
	path := writeSource(t, allOperatorsSource)
	mutants, err := Discover(path)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(mutants) != 13 {
		t.Fatalf("Discover() returned %d mutants; want 13", len(mutants))
	}
	if mutants[0].Original != ">" || mutants[0].Replacement != ">=" {
		t.Fatalf("first mutant = %#v; want > to >=", mutants[0])
	}
	if mutants[0].SourceLine != "\t_ = a > b" || mutants[0].MutatedLine != "\t_ = a >= b" {
		t.Fatalf("first mutant source context = %#v", mutants[0])
	}
	if mutants[11].Original != "true" || mutants[11].Replacement != "false" {
		t.Fatalf("true mutant = %#v", mutants[11])
	}
}

func TestApplyChangesExactlyOneCandidateAndEmitsParsableGo(t *testing.T) {
	path := writeSource(t, allOperatorsSource)
	mutated, err := Apply(path, 1)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if strings.Contains(string(mutated), "a > b") || !strings.Contains(string(mutated), "a >= b") {
		t.Fatalf("first mutation not applied:\n%s", mutated)
	}
	if !strings.Contains(string(mutated), "a == b") {
		t.Fatalf("unexpected second mutation:\n%s", mutated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), path, mutated, 0); err != nil {
		t.Fatalf("mutated source did not parse: %v", err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(original) != allOperatorsSource {
		t.Fatal("Apply() changed original source")
	}
}

func writeSource(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.go")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

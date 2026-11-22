package execution

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyProjectExcludesSecretsAndPreservesOriginal(t *testing.T) {
	project := t.TempDir()
	mustWrite(t, filepath.Join(project, "go.mod"), "module example.com/sample\n")
	mustWrite(t, filepath.Join(project, "main.go"), "package sample\n")
	mustWrite(t, filepath.Join(project, ".env"), "TOKEN=do-not-copy\n")
	mustWrite(t, filepath.Join(project, "key.pem"), "do-not-copy\n")
	mustWrite(t, filepath.Join(project, ".git", "config"), "do-not-copy\n")

	workspace, cleanup, err := CopyProject(project)
	if err != nil {
		t.Fatalf("CopyProject() error = %v", err)
	}
	defer cleanup()
	if _, err := os.Stat(filepath.Join(workspace, "main.go")); err != nil {
		t.Fatalf("expected normal source in workspace: %v", err)
	}
	for _, name := range []string{".env", "key.pem", ".git"} {
		if _, err := os.Stat(filepath.Join(workspace, name)); !os.IsNotExist(err) {
			t.Fatalf("excluded path %q exists or could not be checked: %v", name, err)
		}
	}
	contents, err := os.ReadFile(filepath.Join(project, "main.go"))
	if err != nil || string(contents) != "package sample\n" {
		t.Fatalf("original changed: %q, %v", contents, err)
	}
}

func TestInspectEventsClassifiesTestAndBuildFailures(t *testing.T) {
	testFailed, buildFailed := inspectEvents(`{"Action":"fail","Test":"TestExample"}`)
	if !testFailed || buildFailed {
		t.Fatalf("test event = testFailed:%v buildFailed:%v", testFailed, buildFailed)
	}
	testFailed, buildFailed = inspectEvents(`{"Action":"output","Output":"# example [example.test]\\n"}
{"Action":"output","Output":"FAIL\\texample [build failed]\\n"}`)
	if testFailed || !buildFailed {
		t.Fatalf("build event = testFailed:%v buildFailed:%v", testFailed, buildFailed)
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

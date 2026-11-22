package execution

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CopyProject creates an isolated, disposable copy of a trusted Go project.
func CopyProject(projectRoot string) (string, func(), error) {
	workspace, err := os.MkdirTemp("", "mutcheck-")
	if err != nil {
		return "", nil, fmt.Errorf("create workspace: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(workspace) }
	if err := filepath.WalkDir(projectRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if shouldExclude(relative, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not supported: %s", relative)
		}
		destination := filepath.Join(workspace, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, destination, info)
	}); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("copy project: %w", err)
	}
	return workspace, cleanup, nil
}

func shouldExclude(relative string, entry fs.DirEntry) bool {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for _, part := range parts {
		if part == ".git" {
			return true
		}
	}
	name := entry.Name()
	if name == ".env" || strings.HasPrefix(name, ".env.") {
		return true
	}
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key")
}

func copyFile(source, destination string, info fs.FileInfo) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

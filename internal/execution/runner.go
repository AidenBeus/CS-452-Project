package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const maxOutputBytes = 1 << 20

type TestRun struct {
	ExitCode        int
	TimedOut        bool
	TestFailed      bool
	BuildFailed     bool
	OutputTruncated bool
}

// GoTestRunner invokes go test without a shell and captures only bounded output.
type GoTestRunner struct{}

func (GoTestRunner) Run(ctx context.Context, directory, packagePattern string) (TestRun, error) {
	command := exec.CommandContext(ctx, "go", "test", "-json", packagePattern)
	command.Dir = directory
	buffer := &limitedBuffer{limit: maxOutputBytes}
	command.Stdout = buffer
	command.Stderr = buffer
	err := command.Run()
	run := TestRun{OutputTruncated: buffer.truncated}
	if ctx.Err() == context.DeadlineExceeded {
		run.TimedOut = true
		return run, nil
	}
	if err == nil {
		return run, nil
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return run, fmt.Errorf("start go test: %w", err)
	}
	run.ExitCode = exitError.ExitCode()
	run.TestFailed, run.BuildFailed = inspectEvents(buffer.String())
	return run, nil
}

func inspectEvents(output string) (testFailed, buildFailed bool) {
	for _, line := range strings.Split(output, "\n") {
		var event struct {
			Action string
			Test   string
			Output string
		}
		if json.Unmarshal([]byte(line), &event) == nil {
			if event.Action == "fail" && event.Test != "" {
				testFailed = true
			}
			if strings.Contains(event.Output, "[build failed]") || strings.Contains(event.Output, "build failed") {
				buildFailed = true
			}
			continue
		}
		if strings.Contains(line, "[build failed]") || strings.Contains(line, "build failed") {
			buildFailed = true
		}
	}
	return testFailed, buildFailed
}

type limitedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *limitedBuffer) Write(value []byte) (int, error) {
	if buffer.Len() >= buffer.limit {
		buffer.truncated = true
		return len(value), nil
	}
	remaining := buffer.limit - buffer.Len()
	if len(value) > remaining {
		_, _ = buffer.Buffer.Write(value[:remaining])
		buffer.truncated = true
		return len(value), nil
	}
	return buffer.Buffer.Write(value)
}

func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

package shell

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Minute

// Result holds the output of a shell command execution.
type Result struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// IsSuccess returns true if the command exited with code 0.
func (r *Result) IsSuccess() bool {
	return r.ExitCode == 0
}

// Run executes a command in the given working directory with a timeout.
// The command is split by spaces; for complex commands use RunShell.
func Run(ctx context.Context, workDir, name string, args ...string) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Command: name + " " + strings.Join(args, " "),
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			result.ExitCode = -1
			result.Stderr = result.Stderr + "\n[TIMEOUT] Command exceeded " + defaultTimeout.String()
		} else {
			return nil, fmt.Errorf("failed to execute %q: %w", name, err)
		}
	}

	return result, nil
}

// RunAll executes multiple commands sequentially, stopping on first failure.
// Returns all results collected so far.
func RunAll(ctx context.Context, workDir string, commands [][]string) ([]*Result, error) {
	var results []*Result
	for _, cmd := range commands {
		if len(cmd) == 0 {
			continue
		}
		result, err := Run(ctx, workDir, cmd[0], cmd[1:]...)
		if err != nil {
			return results, err
		}
		results = append(results, result)
		if !result.IsSuccess() {
			// Stop on first failure but return all results so far.
			return results, nil
		}
	}
	return results, nil
}

// FormatResults formats a slice of Results into a readable report.
func FormatResults(results []*Result) string {
	var b strings.Builder
	for i, r := range results {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		status := "PASS"
		if !r.IsSuccess() {
			status = fmt.Sprintf("FAIL (exit %d)", r.ExitCode)
		}
		b.WriteString(fmt.Sprintf("### `%s` — %s\n\n", r.Command, status))
		if r.Stdout != "" {
			b.WriteString("**stdout:**\n```\n")
			b.WriteString(strings.TrimRight(r.Stdout, "\n"))
			b.WriteString("\n```\n\n")
		}
		if r.Stderr != "" {
			b.WriteString("**stderr:**\n```\n")
			b.WriteString(strings.TrimRight(r.Stderr, "\n"))
			b.WriteString("\n```\n\n")
		}
	}
	return b.String()
}

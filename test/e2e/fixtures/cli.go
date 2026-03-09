package fixtures

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CLIRunner executes the rosactl binary for testing
type CLIRunner struct {
	BinaryPath string
}

// NewCLIRunner creates a new CLI runner
func NewCLIRunner(binaryPath string) *CLIRunner {
	return &CLIRunner{
		BinaryPath: binaryPath,
	}
}

// Run executes the rosactl binary with the given arguments
// Returns stdout, stderr, and error
func (r *CLIRunner) Run(args ...string) (string, string, error) {
	cmd := exec.Command(r.BinaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

// ExpectSuccess runs the command and expects it to succeed
// Returns stdout or fails the test
func (r *CLIRunner) ExpectSuccess(args ...string) (string, error) {
	stdout, stderr, err := r.Run(args...)

	if err != nil {
		return "", fmt.Errorf("command failed: %w\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	return stdout, nil
}

// ExpectFailure runs the command and expects it to fail
// Returns stderr and error
func (r *CLIRunner) ExpectFailure(args ...string) (string, error) {
	stdout, stderr, err := r.Run(args...)

	if err == nil {
		return "", fmt.Errorf("expected command to fail but it succeeded\nStdout: %s", stdout)
	}

	return stderr, err
}

// RunWithInput executes the rosactl binary with stdin input
func (r *CLIRunner) RunWithInput(input string, args ...string) (string, string, error) {
	cmd := exec.Command(r.BinaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(input)

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

package workspace

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ResolveBranchName determines the branch name for a task's worktree.
//
// If scriptPath is non-empty, the script is executed with taskJSON on
// stdin and the branch name is read from stdout.
// Otherwise, defaultBranch is returned.
func ResolveBranchName(scriptPath string, taskJSON []byte, defaultBranch string) (string, error) {
	if scriptPath == "" {
		return defaultBranch, nil
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Stdin = bytes.NewReader(taskJSON)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("branch name script: %w (stderr: %s)", err, stderr.String())
	}

	branch := strings.TrimSpace(stdout.String())
	if branch == "" {
		return defaultBranch, nil
	}
	return branch, nil
}

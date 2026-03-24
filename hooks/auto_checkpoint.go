package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/e1sidy/slate"
)

// AutoCheckpoint captures the current state of a task's worktrees
// and creates a checkpoint in Slate. Designed to fire on PreCompact.
func AutoCheckpoint(ctx context.Context, store *slate.Store, taskID, taskDir string) error {
	// Find all worktree symlinks in the task directory.
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return fmt.Errorf("read task dir: %w", err)
	}

	var summaryParts []string
	var allFiles []string

	for _, e := range entries {
		if e.Name() == "CLAUDE.md" || e.Name() == ".claude" {
			continue
		}

		full := filepath.Join(taskDir, e.Name())
		target, err := os.Readlink(full)
		if err != nil {
			continue // not a symlink
		}

		// Run git status in the worktree.
		status, modified := gitWorktreeStatus(target)
		if len(modified) > 0 {
			summaryParts = append(summaryParts,
				fmt.Sprintf("%d files modified in %s", len(modified), e.Name()))
			for _, f := range modified {
				allFiles = append(allFiles, filepath.Join(e.Name(), f))
			}
		}

		// Run git diff --stat for change summary.
		diffStat := gitDiffStat(target)
		if diffStat != "" && status != "" {
			summaryParts = append(summaryParts,
				fmt.Sprintf("%s: %s", e.Name(), diffStat))
		}
	}

	if len(summaryParts) == 0 {
		// No changes to checkpoint.
		return nil
	}

	done := fmt.Sprintf("Auto-checkpoint: %s", strings.Join(summaryParts[:min(len(summaryParts), 3)], ", "))

	_, err = store.AddCheckpoint(ctx, taskID, "roland-auto", slate.CheckpointParams{
		Done:  done,
		Files: allFiles,
	})
	if err != nil {
		return fmt.Errorf("add checkpoint: %w", err)
	}

	return nil
}

// gitWorktreeStatus runs git status --porcelain and returns the raw output
// and a list of modified file paths.
func gitWorktreeStatus(dir string) (string, []string) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", nil
	}

	var files []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[2:]))
		}
	}
	return output, files
}

// gitDiffStat runs git diff --stat and returns a one-line summary.
func gitDiffStat(dir string) string {
	cmd := exec.Command("git", "diff", "--stat", "--shortstat")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Last line is the summary.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

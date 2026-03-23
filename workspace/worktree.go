package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/e1sidy/roland/internal/gitutil"
)

// WorktreeOpts configures a new worktree.
type WorktreeOpts struct {
	Home       string // ROLAND_HOME path.
	RepoName   string // Short name of the registered repo.
	Branch     string // Branch to create (typically the task ID).
	BaseBranch string // Base branch to create from (e.g., "origin/main").
	TaskDir    string // Task workspace directory to symlink into.
	PostSetup  string // Optional script to run after creation.
}

// WorktreeAdd creates a git worktree for a repo and symlinks it into the task directory.
//
// Steps:
//  1. Fetch remote in the repo
//  2. Create worktree with a new branch from the base
//  3. Write .roland-base file (records PR target branch)
//  4. Run post-setup script if configured
//  5. Symlink worktree into the task directory
func WorktreeAdd(opts WorktreeOpts) (string, error) {
	repoDir := filepath.Join(opts.Home, "repos", opts.RepoName)
	if _, err := os.Stat(repoDir); err != nil {
		return "", fmt.Errorf("repo %q not found at %s: %w", opts.RepoName, repoDir, err)
	}

	wtDir := filepath.Join(opts.Home, "worktrees", opts.RepoName, opts.Branch)

	// Check if worktree already exists.
	if _, err := os.Stat(wtDir); err == nil {
		// Worktree exists — just symlink into the task dir.
		if err := symlinkIntoTask(opts.TaskDir, opts.RepoName, wtDir); err != nil {
			return wtDir, fmt.Errorf("symlink existing worktree: %w", err)
		}
		return wtDir, nil
	}

	// Ensure worktrees parent dir exists.
	if err := os.MkdirAll(filepath.Dir(wtDir), 0o755); err != nil {
		return "", fmt.Errorf("create worktree parent: %w", err)
	}

	// Fetch remote before branching (if base is a remote branch).
	if strings.HasPrefix(opts.BaseBranch, "origin/") {
		_ = gitutil.Run(repoDir, "fetch", "origin")
	}

	// Create worktree with new branch.
	err := gitutil.Run(repoDir, "worktree", "add", "-b", opts.Branch, wtDir, opts.BaseBranch)
	if err != nil {
		// Branch might already exist — try without -b.
		err2 := gitutil.Run(repoDir, "worktree", "add", wtDir, opts.Branch)
		if err2 != nil {
			return "", fmt.Errorf("worktree add: %w (also tried existing branch: %v)", err, err2)
		}
	}

	// Write .roland-base file.
	writeBaseBranch(wtDir, opts.BaseBranch)

	// Run post-setup script if provided.
	if opts.PostSetup != "" {
		if err := runPostSetup(opts.PostSetup, repoDir, wtDir); err != nil {
			// Post-setup failure is non-fatal — warn but continue.
			fmt.Fprintf(os.Stderr, "⚠ post-setup for %s failed: %v\n", opts.RepoName, err)
		}
	}

	// Symlink into task directory.
	if err := symlinkIntoTask(opts.TaskDir, opts.RepoName, wtDir); err != nil {
		return wtDir, fmt.Errorf("symlink worktree: %w", err)
	}

	return wtDir, nil
}

// WorktreeRemove removes a git worktree.
func WorktreeRemove(home, repoName, branch string) error {
	repoDir := filepath.Join(home, "repos", repoName)
	wtDir := filepath.Join(home, "worktrees", repoName, branch)

	// Try git worktree remove first.
	if err := gitutil.Run(repoDir, "worktree", "remove", wtDir, "--force"); err != nil {
		// Fallback to manual removal.
		if err2 := os.RemoveAll(wtDir); err2 != nil {
			return fmt.Errorf("remove worktree dir: %w", err2)
		}
	}
	return nil
}

// WorktreeList lists all worktree branches for a repo.
func WorktreeList(home, repoName string) ([]string, error) {
	wtBase := filepath.Join(home, "worktrees", repoName)
	entries, err := os.ReadDir(wtBase)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read worktree dir: %w", err)
	}

	var branches []string
	for _, e := range entries {
		if e.IsDir() {
			branches = append(branches, e.Name())
		}
	}
	return branches, nil
}

// ReadBaseBranch reads the .roland-base file from a worktree directory.
// Returns "origin/main" if the file doesn't exist.
func ReadBaseBranch(wtDir string) string {
	data, err := os.ReadFile(filepath.Join(wtDir, ".roland-base"))
	if err != nil {
		return "origin/main"
	}
	base := strings.TrimSpace(string(data))
	if base == "" {
		return "origin/main"
	}
	return base
}

// writeBaseBranch writes the PR target branch to .roland-base.
func writeBaseBranch(wtDir, baseBranch string) {
	_ = os.WriteFile(filepath.Join(wtDir, ".roland-base"), []byte(baseBranch+"\n"), 0o644)
}

// symlinkIntoTask creates a symlink from the task directory to the worktree.
// e.g., tasks/st-ab12-fix-auth/backend → ../../worktrees/backend/st-ab12
func symlinkIntoTask(taskDir, repoName, wtDir string) error {
	link := filepath.Join(taskDir, repoName)

	// If symlink already exists and points to the right target, skip.
	if gitutil.IsSymlink(link) {
		target, err := os.Readlink(link)
		if err == nil && target == wtDir {
			return nil
		}
		// Wrong target — remove and recreate.
		os.Remove(link)
	}

	return os.Symlink(wtDir, link)
}

// runPostSetup executes a post-setup script in the worktree directory.
func runPostSetup(script, srcDir, destDir string) error {
	// Resolve script path relative to the repo source directory.
	scriptPath := script
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(srcDir, scriptPath)
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = destDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

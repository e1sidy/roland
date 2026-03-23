package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadBaseBranch_Default(t *testing.T) {
	dir := t.TempDir()
	got := ReadBaseBranch(dir)
	if got != "origin/main" {
		t.Errorf("ReadBaseBranch = %q, want %q", got, "origin/main")
	}
}

func TestReadBaseBranch_Custom(t *testing.T) {
	dir := t.TempDir()
	writeBaseBranch(dir, "origin/develop")

	got := ReadBaseBranch(dir)
	if got != "origin/develop" {
		t.Errorf("ReadBaseBranch = %q, want %q", got, "origin/develop")
	}
}

func TestReadBaseBranch_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".roland-base"), []byte(""), 0o644)

	got := ReadBaseBranch(dir)
	if got != "origin/main" {
		t.Errorf("ReadBaseBranch = %q, want %q (empty should default)", got, "origin/main")
	}
}

func TestSymlinkIntoTask(t *testing.T) {
	taskDir := t.TempDir()
	wtDir := t.TempDir()

	if err := symlinkIntoTask(taskDir, "backend", wtDir); err != nil {
		t.Fatalf("symlinkIntoTask: %v", err)
	}

	link := filepath.Join(taskDir, "backend")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if target != wtDir {
		t.Errorf("symlink target = %q, want %q", target, wtDir)
	}
}

func TestSymlinkIntoTask_Idempotent(t *testing.T) {
	taskDir := t.TempDir()
	wtDir := t.TempDir()

	// First call.
	if err := symlinkIntoTask(taskDir, "backend", wtDir); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call — should be idempotent.
	if err := symlinkIntoTask(taskDir, "backend", wtDir); err != nil {
		t.Fatalf("second call: %v", err)
	}

	link := filepath.Join(taskDir, "backend")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if target != wtDir {
		t.Errorf("symlink target = %q, want %q", target, wtDir)
	}
}

func TestSymlinkIntoTask_UpdateTarget(t *testing.T) {
	taskDir := t.TempDir()
	wtDir1 := t.TempDir()
	wtDir2 := t.TempDir()

	// Point to first target.
	symlinkIntoTask(taskDir, "backend", wtDir1)
	// Update to second target.
	symlinkIntoTask(taskDir, "backend", wtDir2)

	link := filepath.Join(taskDir, "backend")
	target, _ := os.Readlink(link)
	if target != wtDir2 {
		t.Errorf("symlink target = %q, want %q (updated)", target, wtDir2)
	}
}

func TestResolveBranchName_NoScript(t *testing.T) {
	got, err := ResolveBranchName("", nil, "st-ab12")
	if err != nil {
		t.Fatalf("ResolveBranchName: %v", err)
	}
	if got != "st-ab12" {
		t.Errorf("branch = %q, want %q", got, "st-ab12")
	}
}

func TestWorktreeList_Empty(t *testing.T) {
	home := t.TempDir()
	branches, err := WorktreeList(home, "backend")
	if err != nil {
		t.Fatalf("WorktreeList: %v", err)
	}
	if len(branches) != 0 {
		t.Errorf("WorktreeList = %d, want 0", len(branches))
	}
}

func TestWorktreeList_WithEntries(t *testing.T) {
	home := t.TempDir()
	wtBase := filepath.Join(home, "worktrees", "backend")
	os.MkdirAll(filepath.Join(wtBase, "st-ab12"), 0o755)
	os.MkdirAll(filepath.Join(wtBase, "st-cd34"), 0o755)

	branches, err := WorktreeList(home, "backend")
	if err != nil {
		t.Fatalf("WorktreeList: %v", err)
	}
	if len(branches) != 2 {
		t.Errorf("WorktreeList = %d, want 2", len(branches))
	}
}

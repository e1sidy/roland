// Package workspace manages ephemeral task directories and git worktrees.
//
// Each active task gets a directory under ROLAND_HOME/tasks/<slug>/ that
// contains symlinks to git worktrees stored under ROLAND_HOME/worktrees/.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// TaskDir represents an active task workspace.
type TaskDir struct {
	// Path is the absolute filesystem path to the task directory.
	Path string

	// TaskID is the Slate task ID (e.g., "st-ab12" or "st-ab12.1").
	TaskID string

	// Slug is the directory name (e.g., "st-ab12-fix-auth-bug").
	Slug string
}

// Create creates a new task workspace directory.
// The directory name is derived from the task ID and title.
func Create(home, taskID, title string) (*TaskDir, error) {
	slug := makeSlug(taskID, title)
	dir := filepath.Join(home, "tasks", slug)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create task dir: %w", err)
	}

	return &TaskDir{
		Path:   dir,
		TaskID: taskID,
		Slug:   slug,
	}, nil
}

// Open finds an existing task workspace by task ID prefix match.
// The task ID can be a prefix (e.g., "st-ab12" matches "st-ab12-fix-auth").
func Open(home, taskID string) (*TaskDir, error) {
	tasksDir := filepath.Join(home, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task %q not found: tasks dir does not exist", taskID)
		}
		return nil, fmt.Errorf("read tasks dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		extracted := ExtractTaskID(entry.Name())
		if extracted == taskID {
			dir := filepath.Join(tasksDir, entry.Name())
			return &TaskDir{
				Path:   dir,
				TaskID: extracted,
				Slug:   entry.Name(),
			}, nil
		}
	}

	return nil, fmt.Errorf("task %q not found", taskID)
}

// Remove deletes a task workspace directory.
func Remove(home, taskID string) error {
	td, err := Open(home, taskID)
	if err != nil {
		return err
	}
	return os.RemoveAll(td.Path)
}

// List returns all task workspaces, sorted by slug.
func List(home string) ([]*TaskDir, error) {
	tasksDir := filepath.Join(home, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tasks dir: %w", err)
	}

	var dirs []*TaskDir
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := ExtractTaskID(entry.Name())
		if id == "" {
			continue
		}
		dirs = append(dirs, &TaskDir{
			Path:   filepath.Join(tasksDir, entry.Name()),
			TaskID: id,
			Slug:   entry.Name(),
		})
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Slug < dirs[j].Slug
	})
	return dirs, nil
}

// taskIDPattern matches Slate IDs like "st-ab12" or "st-ab12.1.2".
var taskIDPattern = regexp.MustCompile(`^(st-[a-z0-9]+(?:\.[0-9]+)*)`)

// ExtractTaskID extracts the Slate task ID from a slug or path.
// Returns empty string if no task ID is found.
func ExtractTaskID(slugOrPath string) string {
	// Use the base name if it looks like a path.
	base := filepath.Base(slugOrPath)
	m := taskIDPattern.FindString(base)
	return m
}

// makeSlug generates a filesystem-safe directory name from a task ID and title.
// Example: makeSlug("st-ab12", "Fix Auth Bug!") → "st-ab12-fix-auth-bug"
func makeSlug(taskID, title string) string {
	if title == "" {
		return taskID
	}

	// Lowercase and replace non-alphanumeric with dashes.
	var b strings.Builder
	b.WriteString(taskID)
	b.WriteByte('-')

	prevDash := true // Avoid leading dash after taskID-.
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}

	slug := strings.TrimRight(b.String(), "-")

	// Truncate to 50 chars.
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

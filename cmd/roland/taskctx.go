package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/roland/workspace"
	"github.com/e1sidy/slate"
)

// resolveTaskID determines the task ID and task directory from args or cwd.
//
// If args contains a task ID, it is used directly.
// Otherwise, the current working directory is inspected for a task ID prefix.
func resolveTaskID(home string, args []string) (taskID string, taskDir string, err error) {
	if len(args) > 0 && args[0] != "" {
		taskID = args[0]
	} else {
		// Try to detect from cwd.
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return "", "", fmt.Errorf("get cwd: %w", cwdErr)
		}
		taskID = workspace.ExtractTaskID(cwd)
		if taskID == "" {
			// Check if we're inside a task subdirectory.
			taskID = workspace.ExtractTaskID(filepath.Base(cwd))
		}
		if taskID == "" {
			return "", "", fmt.Errorf("no task ID provided and could not detect from cwd %q", cwd)
		}
	}

	// Find the task directory.
	td, err := workspace.Open(home, taskID)
	if err != nil {
		return taskID, "", err
	}
	return td.TaskID, td.Path, nil
}

// openSlateStore opens a connection to the Slate database using config.
func openSlateStore(cfg *roland.Config) (*slate.Store, error) {
	dbPath := slate.DefaultDBPath()
	if cfg.SlateHome != "" {
		dbPath = filepath.Join(cfg.SlateHome, "slate.db")
	}

	store, err := slate.Open(context.Background(), dbPath, slate.WithPrefix("st"))
	if err != nil {
		return nil, fmt.Errorf("open slate store: %w", err)
	}
	return store, nil
}

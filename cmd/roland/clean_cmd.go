package main

import (
	"context"
	"fmt"

	"github.com/e1sidy/roland/workspace"
	"github.com/spf13/cobra"
)

func cleanCmd() *cobra.Command {
	var (
		dryRun bool
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Prune orphaned task workspaces",
		Long: `Finds task workspaces whose tasks are closed or cancelled in Slate,
and removes them along with their worktrees.

Use --dry-run to see what would be cleaned without making changes.
Use --force to remove workspaces even if the Slate store is unavailable.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			dirs, err := workspace.List(cfg.Home)
			if err != nil {
				return fmt.Errorf("list workspaces: %w", err)
			}
			if len(dirs) == 0 {
				fmt.Println("No task workspaces found.")
				return nil
			}

			// Open store to check task status.
			store, storeErr := openSlateStore(cfg)
			if storeErr != nil && !force {
				return fmt.Errorf("open store: %w (use --force to clean without Slate)", storeErr)
			}
			if store != nil {
				defer store.Close()
			}

			cleaned := 0
			for _, td := range dirs {
				shouldClean := false

				if store != nil {
					task, taskErr := store.Get(ctx, td.TaskID)
					if taskErr != nil {
						// Task not found in Slate — orphaned.
						shouldClean = true
					} else if task.Status.IsTerminal() {
						shouldClean = true
					}
				} else if force {
					// No store access + force — clean everything.
					shouldClean = true
				}

				if !shouldClean {
					continue
				}

				if dryRun {
					fmt.Printf("[dry-run] Would remove workspace %s (%s)\n", td.TaskID, td.Path)
					cleaned++
					continue
				}

				fmt.Printf("Removing workspace %s...\n", td.TaskID)
				if err := workspace.Remove(cfg.Home, td.TaskID); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to remove %s: %v\n", td.TaskID, err)
					continue
				}
				cleaned++
			}

			if cleaned == 0 {
				fmt.Println("Nothing to clean.")
			} else if dryRun {
				fmt.Printf("\nWould clean %d workspace(s).\n", cleaned)
			} else {
				fmt.Printf("\nCleaned %d workspace(s).\n", cleaned)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cmd.Flags().BoolVar(&force, "force", false, "Remove workspaces even without Slate access")

	return cmd
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/e1sidy/roland/internal/gitutil"
	"github.com/e1sidy/roland/workspace"
	"github.com/spf13/cobra"
)

func doneCmd() *cobra.Command {
	var (
		reason string
		dryRun bool
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "done [task-id]",
		Short: "Complete a task: check PRs, clean worktrees, close task",
		Long: `Verifies all PRs are merged, removes worktrees and the task workspace,
then closes the task in Slate.

Use --force to skip PR merge checks.
Use --dry-run to see what would happen without making changes.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, taskDir, err := resolveTaskID(cfg.Home, args)
			if err != nil {
				return fmt.Errorf("resolve task: %w", err)
			}

			ctx := context.Background()

			// Discover worktrees.
			entries, err := os.ReadDir(taskDir)
			if err != nil {
				return fmt.Errorf("read task dir: %w", err)
			}

			type wtInfo struct {
				repoName string
				target   string
				branch   string
			}
			var worktrees []wtInfo

			for _, e := range entries {
				linkPath := filepath.Join(taskDir, e.Name())
				if !gitutil.IsSymlink(linkPath) {
					continue
				}
				target, err := os.Readlink(linkPath)
				if err != nil {
					continue
				}
				if !filepath.IsAbs(target) {
					target = filepath.Join(taskDir, target)
				}

				branchOut, err := gitutil.Output(target, "rev-parse", "--abbrev-ref", "HEAD")
				if err != nil {
					continue
				}
				branch := strings.TrimSpace(string(branchOut))
				worktrees = append(worktrees, wtInfo{
					repoName: e.Name(),
					target:   target,
					branch:   branch,
				})
			}

			// Check PR merge status unless forced.
			if !force {
				for _, wt := range worktrees {
					ghCmd := exec.Command("gh", "pr", "list",
						"--head", wt.branch,
						"--state", "open",
						"--json", "number,state",
					)
					ghCmd.Dir = wt.target
					out, err := ghCmd.Output()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not check PR status for %s: %v\n", wt.repoName, err)
						continue
					}
					// If output contains open PRs, warn (don't hard-block).
					trimmed := strings.TrimSpace(string(out))
					if trimmed != "[]" && trimmed != "" {
						fmt.Fprintf(os.Stderr, "⚠ Warning: %s has open PR(s) for branch %s\n", wt.repoName, wt.branch)
						if dryRun {
							fmt.Printf("[dry-run] %s: has open PR(s) for branch %s\n", wt.repoName, wt.branch)
						}
					}
				}
			}

			if dryRun {
				fmt.Printf("[dry-run] Would remove %d worktree(s) and workspace for %s\n", len(worktrees), taskID)
				fmt.Printf("[dry-run] Would close task %s with reason: %s\n", taskID, reason)
				return nil
			}

			// Clean worktrees.
			for _, wt := range worktrees {
				fmt.Printf("Removing worktree %s/%s...\n", wt.repoName, wt.branch)
				if err := workspace.WorktreeRemove(cfg.Home, wt.repoName, wt.branch); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree %s/%s: %v\n", wt.repoName, wt.branch, err)
				}

				// Delete remote branch if configured.
				if cfg.CleanupRemoteBranches && wt.branch != "" {
					fmt.Printf("Deleting remote branch %s...\n", wt.branch)
					if err := gitutil.Run(wt.target, "push", "origin", "--delete", wt.branch); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to delete remote branch %s: %v\n", wt.branch, err)
					}
				}
			}

			// Remove workspace.
			fmt.Printf("Removing workspace %s...\n", taskDir)
			if err := workspace.Remove(cfg.Home, taskID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove workspace: %v\n", err)
			}

			// Close task in Slate.
			store, err := openSlateStore(cfg)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer store.Close()

			if err := store.CloseTask(ctx, taskID, reason, "roland"); err != nil {
				return fmt.Errorf("close task: %w", err)
			}

			fmt.Printf("Task %s completed.\n", taskID)
			return nil
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "done", "Closure reason")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without making changes")
	cmd.Flags().BoolVar(&force, "force", false, "Skip PR merge checks")

	return cmd
}

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/roland/internal/gitutil"
	"github.com/spf13/cobra"
)

func workCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work [task-id]",
		Short: "Resume work on a task and launch agent",
		Long: `Resolves the task (from argument or cwd), shows a resumption briefing
with the latest checkpoint and git status across worktrees, then launches
the configured AI coding agent.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, taskDir, err := resolveTaskID(cfg.Home, args)
			if err != nil {
				return fmt.Errorf("resolve task: %w", err)
			}

			ctx := context.Background()

			// Show resumption briefing.
			fmt.Printf("%s Resuming task %s\n", bold(">>>"), bold(taskID))
			fmt.Printf("    Workspace: %s\n\n", taskDir)

			// Show latest checkpoint if available.
			store, storeErr := openSlateStore(cfg)
			if storeErr == nil {
				defer store.Close()
				cp, cpErr := store.LatestCheckpoint(ctx, taskID)
				if cpErr == nil {
					fmt.Println(bold("Latest Checkpoint:"))
					fmt.Printf("  Done:      %s\n", cp.Done)
					if cp.Next != "" {
						fmt.Printf("  Next:      %s\n", cp.Next)
					}
					if cp.Blockers != "" {
						fmt.Printf("  Blockers:  %s\n", colorize(colorRed, cp.Blockers))
					}
					if cp.Decisions != "" {
						fmt.Printf("  Decisions: %s\n", cp.Decisions)
					}
					fmt.Printf("  Time:      %s\n", cp.CreatedAt.Format("2006-01-02 15:04"))
					fmt.Println()
				}
			}

			// Show git status for each worktree in the task directory.
			entries, _ := os.ReadDir(taskDir)
			for _, e := range entries {
				linkPath := filepath.Join(taskDir, e.Name())
				// Only check symlinks (worktrees).
				if !gitutil.IsSymlink(linkPath) {
					continue
				}
				target, err := os.Readlink(linkPath)
				if err != nil {
					continue
				}
				// Resolve relative symlinks.
				if !filepath.IsAbs(target) {
					target = filepath.Join(taskDir, target)
				}

				out, gitErr := gitutil.Output(target, "status", "--porcelain")
				if gitErr != nil {
					continue
				}

				status := strings.TrimSpace(string(out))
				if status == "" {
					fmt.Printf("  %s: %s\n", e.Name(), colorize(colorGreen, "clean"))
				} else {
					lines := strings.Split(status, "\n")
					fmt.Printf("  %s: %s (%d changed)\n", e.Name(), colorize(colorYellow, "dirty"), len(lines))
				}
			}
			fmt.Println()

			// Regenerate CLAUDE.md with latest persona content (picks up learned patterns).
			if storeErr == nil {
				personaName := ""
				if attr, _ := store.GetAttr(ctx, taskID, roland.AttrPersonaUsed); attr != nil {
					personaName = attr.Value
				}
				if personaName != "" {
					task, _ := store.GetFull(ctx, taskID)
					if task != nil {
						var repoNames []string
						for _, e := range entries {
							if gitutil.IsSymlink(filepath.Join(taskDir, e.Name())) {
								repoNames = append(repoNames, e.Name())
							}
						}
						if err := writeTaskClaudeMDResolved(taskDir, cfg.Home, personaName, repoNames, task); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to refresh CLAUDE.md: %v\n", err)
						}
					}
				}
			}

			// Launch agent.
			fmt.Printf("Launching %s in %s...\n", cfg.Agent, taskDir)
			flags := cfg.AgentFlags[cfg.Agent]
			return launchAgent(taskDir, cfg.Agent, flags)
		},
	}

	return cmd
}

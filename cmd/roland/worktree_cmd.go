package main

import (
	"fmt"

	"github.com/e1sidy/roland/workspace"
	"github.com/spf13/cobra"
)

func worktreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "Manage git worktrees",
	}

	cmd.AddCommand(
		worktreeAddCmd(),
		worktreeRemoveCmd(),
		worktreeListCmd(),
	)

	return cmd
}

func worktreeAddCmd() *cobra.Command {
	var (
		baseBranch string
		taskArg    string
	)

	cmd := &cobra.Command{
		Use:   "add <repo-name> <branch>",
		Short: "Add a git worktree for a repo",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoName := args[0]
			branch := args[1]

			// Validate repo exists.
			rc, ok := cfg.Repos[repoName]
			if !ok {
				return fmt.Errorf("repo %q not registered", repoName)
			}

			base := baseBranch
			if base == "" {
				base = rc.BaseBranch
				if base == "" {
					base = "origin/main"
				}
			}

			// Resolve task dir if provided.
			taskDir := ""
			if taskArg != "" {
				_, td, err := resolveTaskID(cfg.Home, []string{taskArg})
				if err != nil {
					return fmt.Errorf("resolve task: %w", err)
				}
				taskDir = td
			}

			wtDir, err := workspace.WorktreeAdd(workspace.WorktreeOpts{
				Home:       cfg.Home,
				RepoName:   repoName,
				Branch:     branch,
				BaseBranch: base,
				TaskDir:    taskDir,
				PostSetup:  rc.PostSetup,
			})
			if err != nil {
				return fmt.Errorf("add worktree: %w", err)
			}

			fmt.Printf("Worktree created at %s\n", wtDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&baseBranch, "base", "", "Base branch (default: repo's base_branch or origin/main)")
	cmd.Flags().StringVar(&taskArg, "task", "", "Task ID to symlink worktree into")

	return cmd
}

func worktreeRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <repo-name> <branch>",
		Aliases: []string{"remove"},
		Short:   "Remove a git worktree",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := workspace.WorktreeRemove(cfg.Home, args[0], args[1]); err != nil {
				return fmt.Errorf("remove worktree: %w", err)
			}
			fmt.Printf("Removed worktree %s/%s\n", args[0], args[1])
			return nil
		},
	}
}

func worktreeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo-name>",
		Short: "List worktree branches for a repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branches, err := workspace.WorktreeList(cfg.Home, args[0])
			if err != nil {
				return fmt.Errorf("list worktrees: %w", err)
			}
			if len(branches) == 0 {
				fmt.Printf("No worktrees for %s\n", args[0])
				return nil
			}
			for _, b := range branches {
				fmt.Printf("  %s\n", b)
			}
			return nil
		},
	}
}

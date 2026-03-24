package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/roland/persona"
	"github.com/e1sidy/roland/workspace"
	"github.com/e1sidy/slate"
	"github.com/spf13/cobra"
)

func handoffCmd() *cobra.Command {
	var toPersona string

	cmd := &cobra.Command{
		Use:   "handoff <task-id>",
		Short: "Transfer a task to a different persona",
		Long: `Hands off a task to a new persona by:
  1. Auto-checkpointing current state
  2. Removing old workspace (preserving worktrees)
  3. Updating persona_used attribute
  4. Creating new workspace with new persona CLAUDE.md
  5. Re-symlinking worktrees
  6. Launching agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			ctx := context.Background()

			if !persona.IsValid(cfg.Home, toPersona) {
				return fmt.Errorf("persona %q not found", toPersona)
			}

			store, err := openSlateStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := roland.EnsureAttrs(ctx, store); err != nil {
				return err
			}

			task, err := store.GetFull(ctx, taskID)
			if err != nil {
				return fmt.Errorf("get task: %w", err)
			}

			// Step 1: Auto-checkpoint before handoff.
			_, err = store.AddCheckpoint(ctx, taskID, "roland", slate.CheckpointParams{
				Done: fmt.Sprintf("Handing off to %s", toPersona),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: auto-checkpoint failed: %v\n", err)
			} else {
				fmt.Fprintln(os.Stderr, "Step 1: Auto-checkpoint saved.")
			}

			// Step 2: Get current worktree symlinks before removing workspace.
			td, err := workspace.Open(cfg.Home, taskID)
			if err != nil {
				return fmt.Errorf("open workspace: %w", err)
			}
			worktreeLinks := getWorktreeSymlinks(td.Path)

			// Step 3: Remove old workspace (keeps worktree targets on disk).
			if err := workspace.Remove(cfg.Home, taskID); err != nil {
				return fmt.Errorf("remove workspace: %w", err)
			}
			fmt.Fprintln(os.Stderr, "Step 2: Old workspace removed (worktrees preserved).")

			// Step 4: Update persona_used attr.
			if err := store.SetAttr(ctx, taskID, roland.AttrPersonaUsed, toPersona); err != nil {
				return fmt.Errorf("set persona attr: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Step 3: Persona updated to %s.\n", toPersona)

			// Step 5: Create new workspace.
			newTD, err := workspace.Create(cfg.Home, taskID, task.Title)
			if err != nil {
				return fmt.Errorf("create new workspace: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Step 4: New workspace created at %s\n", newTD.Path)

			// Step 6: Re-symlink worktrees into new workspace.
			for repoName, target := range worktreeLinks {
				linkPath := filepath.Join(newTD.Path, repoName)
				if err := os.Symlink(target, linkPath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to symlink %s: %v\n", repoName, err)
				}
			}
			fmt.Fprintln(os.Stderr, "Step 5: Worktrees re-linked.")

			// Step 7: Generate CLAUDE.md with new persona.
			var repoNames []string
			for name := range worktreeLinks {
				repoNames = append(repoNames, name)
			}
			if err := writeTaskClaudeMDResolved(newTD.Path, cfg.Home, toPersona, repoNames, task); err != nil {
				return fmt.Errorf("write CLAUDE.md: %w", err)
			}
			fmt.Fprintln(os.Stderr, "Step 6: CLAUDE.md generated with new persona.")

			// Step 8: Install hooks + skills with new persona context.
			installHooksAndSkills(newTD.Path, cfg, toPersona, string(task.Type))
			fmt.Fprintln(os.Stderr, "Step 7: Hooks and skills installed.")

			// Launch agent.
			fmt.Fprintf(os.Stderr, "\nLaunching %s in %s...\n", cfg.Agent, newTD.Path)
			flags := cfg.AgentFlags[cfg.Agent]
			return launchAgent(newTD.Path, cfg.Agent, flags)
		},
	}

	cmd.Flags().StringVar(&toPersona, "to", "", "Target persona (required)")
	cmd.MarkFlagRequired("to")
	return cmd
}

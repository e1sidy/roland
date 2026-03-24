package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/roland/persona"
	"github.com/e1sidy/slate"
	"github.com/spf13/cobra"
)

func delegateCmd() *cobra.Command {
	var (
		personaName string
		repos       string
	)

	cmd := &cobra.Command{
		Use:   "delegate <subtask-id>",
		Short: "Delegate a subtask to a persona and launch agent in background",
		Long: `Creates workspace for a subtask and launches an agent in background.
If the subtask doesn't exist, creates it as a child of the current task.
Reuses the same setup flow as pickup but launches in background.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			subtaskID := args[0]
			ctx := context.Background()

			store, err := openSlateStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := roland.EnsureAttrs(ctx, store); err != nil {
				return err
			}

			// Check if subtask exists; if not, create it.
			_, err = store.Get(ctx, subtaskID)
			if err != nil {
				// Try to detect parent from current task context.
				parentID, _, _ := resolveTaskID(cfg.Home, nil)
				if parentID == "" {
					return fmt.Errorf("subtask %q not found and no parent context to create it", subtaskID)
				}
				// Create the subtask.
				created, err := store.Create(ctx, slate.CreateParams{
					Title:    subtaskID, // use ID as title placeholder
					ParentID: parentID,
				})
				if err != nil {
					return fmt.Errorf("create subtask: %w", err)
				}
				subtaskID = created.ID
				fmt.Fprintf(cmd.ErrOrStderr(), "Created subtask %s under %s\n", subtaskID, parentID)
			}

			// Parse repos.
			var repoNames []string
			if repos != "" {
				repoNames = strings.Split(repos, ",")
			} else {
				for name := range cfg.Repos {
					repoNames = append(repoNames, name)
				}
			}
			for _, name := range repoNames {
				if _, ok := cfg.Repos[name]; !ok {
					return fmt.Errorf("repo %q not registered", name)
				}
			}

			if !persona.IsValid(cfg.Home, personaName) {
				return fmt.Errorf("persona %q not found", personaName)
			}

			// Setup workspace (shared with pickup).
			result, err := setupWorkspace(ctx, setupOpts{
				Store:       store,
				Cfg:         cfg,
				TaskID:      subtaskID,
				PersonaName: personaName,
				RepoNames:   repoNames,
			})
			if err != nil {
				return fmt.Errorf("setup workspace: %w", err)
			}

			// Launch agent in BACKGROUND (not exec replace).
			flags := cfg.AgentFlags[cfg.Agent]
			pid, err := launchAgentBackground(result.TaskDir.Path, cfg.Agent, flags)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: agent launch failed: %v\n", err)
				fmt.Fprintf(cmd.ErrOrStderr(), "Workspace is ready at %s — use 'roland work %s' to retry\n", result.TaskDir.Path, subtaskID)
				return nil
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Delegated %s to %s (PID %d)\n", subtaskID, personaName, pid)
			fmt.Fprintf(cmd.ErrOrStderr(), "Use 'roland watch' to monitor progress\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&personaName, "persona", "builder", "Persona for the delegated agent")
	cmd.Flags().StringVar(&repos, "repos", "", "Comma-separated repo names (default: all)")
	return cmd
}

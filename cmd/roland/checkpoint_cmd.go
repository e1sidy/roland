package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/e1sidy/slate"
	"github.com/spf13/cobra"
)

func checkpointCmd() *cobra.Command {
	var (
		done      string
		decisions string
		next      string
		blockers  string
		files     string
	)

	cmd := &cobra.Command{
		Use:   "checkpoint [task-id]",
		Short: "Record a progress checkpoint",
		Long: `Adds a structured checkpoint to the task in Slate.

The --done flag is required and describes what was accomplished.
Other fields are optional but recommended.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, _, err := resolveTaskID(cfg.Home, args)
			if err != nil {
				return fmt.Errorf("resolve task: %w", err)
			}

			ctx := context.Background()
			store, err := openSlateStore(cfg)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer store.Close()

			var fileList []string
			if files != "" {
				fileList = strings.Split(files, ",")
			}

			cp, err := store.AddCheckpoint(ctx, taskID, "roland", slate.CheckpointParams{
				Done:      done,
				Decisions: decisions,
				Next:      next,
				Blockers:  blockers,
				Files:     fileList,
			})
			if err != nil {
				return fmt.Errorf("add checkpoint: %w", err)
			}

			fmt.Printf("Checkpoint added for %s at %s\n", taskID, cp.CreatedAt.Format("15:04:05"))
			if next != "" {
				fmt.Printf("  Next: %s\n", next)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&done, "done", "", "What was accomplished (required)")
	cmd.Flags().StringVar(&decisions, "decisions", "", "Key decisions made")
	cmd.Flags().StringVar(&next, "next", "", "What should happen next")
	cmd.Flags().StringVar(&blockers, "blockers", "", "Current blockers")
	cmd.Flags().StringVar(&files, "files", "", "Comma-separated file paths touched")
	_ = cmd.MarkFlagRequired("done")

	return cmd
}

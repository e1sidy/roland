package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/slate"
	"github.com/spf13/cobra"
)

func watchCmd() *cobra.Command {
	var (
		interval      string
		staleMinutes  int
	)

	cmd := &cobra.Command{
		Use:   "watch [task-id]",
		Short: "Monitor child task status changes",
		Long: `Polls Slate for status changes on child tasks of the given parent.
Reports when children close, alerts on stale in_progress tasks,
and notifies when all children are done.

Press Ctrl+C to stop.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve parent task.
			parentID, _, err := resolveTaskID(cfg.Home, args)
			if err != nil {
				return fmt.Errorf("resolve task: %w", err)
			}

			dur, err := time.ParseDuration(interval)
			if err != nil {
				return fmt.Errorf("invalid interval %q: %w", interval, err)
			}

			store, err := openSlateStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			if err := roland.EnsureAttrs(ctx, store); err != nil {
				return err
			}

			return runWatch(ctx, store, parentID, dur, time.Duration(staleMinutes)*time.Minute)
		},
	}

	cmd.Flags().StringVar(&interval, "interval", "5m", "Polling interval (e.g. 30s, 5m)")
	cmd.Flags().IntVar(&staleMinutes, "stale", 30, "Alert if in_progress task has no checkpoint for this many minutes")
	return cmd
}

// statusSnapshot holds the last-known status of each child task.
type statusSnapshot map[string]slate.Status

func runWatch(ctx context.Context, store *slate.Store, parentID string, interval, staleThreshold time.Duration) error {
	parent, err := store.Get(ctx, parentID)
	if err != nil {
		return fmt.Errorf("get parent: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Watching children of %s — %s (interval: %s)\n\n", parent.ID, parent.Title, interval)

	// Initial snapshot.
	snapshot := make(statusSnapshot)
	children, err := store.Children(ctx, parentID)
	if err != nil {
		return fmt.Errorf("get children: %w", err)
	}
	for _, c := range children {
		snapshot[c.ID] = c.Status
		persona := getAttrValue(ctx, store, c.ID, roland.AttrPersonaUsed)
		fmt.Fprintf(os.Stderr, "  %s [%s] %s (persona: %s)\n", c.ID, c.Status, c.Title, persona)
	}
	fmt.Fprintln(os.Stderr)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "\nWatch stopped.")
			return nil
		case <-ticker.C:
			done, err := pollChildren(ctx, store, parentID, snapshot, staleThreshold)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Poll error: %v\n", err)
				continue
			}
			if done {
				fmt.Fprintf(os.Stderr, "\n✓ All subtasks done — parent %s is ready to close.\n", parentID)
				return nil
			}
		}
	}
}

func pollChildren(ctx context.Context, store *slate.Store, parentID string, snapshot statusSnapshot, staleThreshold time.Duration) (bool, error) {
	children, err := store.Children(ctx, parentID)
	if err != nil {
		return false, err
	}

	allTerminal := true
	now := time.Now()

	for _, c := range children {
		oldStatus, known := snapshot[c.ID]

		if !known {
			// New child appeared.
			fmt.Fprintf(os.Stderr, "[%s] New child: %s — %s (%s)\n",
				now.Format("15:04:05"), c.ID, c.Title, c.Status)
			snapshot[c.ID] = c.Status
		} else if c.Status != oldStatus {
			// Status changed.
			persona := getAttrValue(ctx, store, c.ID, roland.AttrPersonaUsed)
			fmt.Fprintf(os.Stderr, "[%s] %s: %s → %s (persona: %s)\n",
				now.Format("15:04:05"), c.ID, oldStatus, c.Status, persona)
			snapshot[c.ID] = c.Status
		}

		// Check for stale in_progress tasks.
		if c.Status == slate.StatusInProgress && staleThreshold > 0 {
			cp, _ := store.LatestCheckpoint(ctx, c.ID)
			if cp != nil {
				age := now.Sub(cp.CreatedAt)
				if age > staleThreshold {
					fmt.Fprintf(os.Stderr, "[%s] ⚠ %s: in_progress with no checkpoint for %s\n",
						now.Format("15:04:05"), c.ID, age.Round(time.Minute))
				}
			} else {
				// No checkpoints at all — check task updated_at.
				if now.Sub(c.UpdatedAt) > staleThreshold {
					fmt.Fprintf(os.Stderr, "[%s] ⚠ %s: in_progress with no checkpoint for %s\n",
						now.Format("15:04:05"), c.ID, now.Sub(c.UpdatedAt).Round(time.Minute))
				}
			}
		}

		if !c.Status.IsTerminal() {
			allTerminal = false
		}
	}

	return allTerminal && len(children) > 0, nil
}

func getAttrValue(ctx context.Context, store *slate.Store, taskID, key string) string {
	attr, err := store.GetAttr(ctx, taskID, key)
	if err != nil || attr == nil {
		return ""
	}
	return attr.Value
}

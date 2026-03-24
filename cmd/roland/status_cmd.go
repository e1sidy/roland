package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/roland/internal/gitutil"
	"github.com/e1sidy/roland/workspace"
	"github.com/e1sidy/slate"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	var (
		showAll  bool
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all task workspaces",
		Long: `Lists all active task workspaces with their status, worktree state
(dirty/clean), and latest checkpoint.

Use --all to include completed tasks.
Use --json for machine-readable output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			dirs, err := workspace.List(cfg.Home)
			if err != nil {
				return fmt.Errorf("list workspaces: %w", err)
			}
			if len(dirs) == 0 {
				if !jsonOut {
					fmt.Println("No active task workspaces.")
				} else {
					fmt.Println("[]")
				}
				return nil
			}

			// Open store for task info.
			store, storeErr := openSlateStore(cfg)
			if storeErr != nil && !jsonOut {
				fmt.Fprintf(os.Stderr, "Warning: could not open Slate store: %v\n", storeErr)
			}
			if store != nil {
				defer store.Close()
			}

			type cpJSON struct {
				Done      string `json:"done"`
				Next      string `json:"next,omitempty"`
				Blockers  string `json:"blockers,omitempty"`
				CreatedAt string `json:"created_at"`
			}

			type worktreeStatus struct {
				Repo   string `json:"repo"`
				Branch string `json:"branch"`
				Dirty  bool   `json:"dirty"`
			}

			type taskStatus struct {
				TaskID           string           `json:"taskID"`
				Title            string           `json:"title,omitempty"`
				Status           string           `json:"status,omitempty"`
				Persona          string           `json:"persona,omitempty"`
				LatestCheckpoint *cpJSON          `json:"latestCheckpoint,omitempty"`
				Worktrees        []worktreeStatus `json:"worktrees"`
			}

			var statuses []taskStatus

			for _, td := range dirs {
				ts := taskStatus{
					TaskID: td.TaskID,
				}

				// Get task info from Slate.
				if store != nil {
					task, taskErr := store.GetFull(ctx, td.TaskID)
					if taskErr == nil {
						ts.Title = task.Title
						ts.Status = string(task.Status)

						// Filter out terminal tasks unless --all.
						if !showAll && task.Status.IsTerminal() {
							continue
						}

						// Get persona.
						if personaAttr, ok := task.Attrs[roland.AttrPersonaUsed]; ok {
							ts.Persona = personaAttr
						}
					}

					// Get latest checkpoint.
					cp, cpErr := store.LatestCheckpoint(ctx, td.TaskID)
					if cpErr == nil {
						ts.LatestCheckpoint = &cpJSON{
							Done:      cp.Done,
							Next:      cp.Next,
							Blockers:  cp.Blockers,
							CreatedAt: cp.CreatedAt.Format("2006-01-02 15:04"),
						}
					}
				}

				// Discover worktrees.
				entries, _ := os.ReadDir(td.Path)
				for _, e := range entries {
					linkPath := filepath.Join(td.Path, e.Name())
					if !gitutil.IsSymlink(linkPath) {
						continue
					}
					target, err := os.Readlink(linkPath)
					if err != nil {
						continue
					}
					if !filepath.IsAbs(target) {
						target = filepath.Join(td.Path, target)
					}

					ws := worktreeStatus{Repo: e.Name()}

					branchOut, err := gitutil.Output(target, "rev-parse", "--abbrev-ref", "HEAD")
					if err == nil {
						ws.Branch = strings.TrimSpace(string(branchOut))
					}

					statusOut, err := gitutil.Output(target, "status", "--porcelain")
					if err == nil {
						ws.Dirty = strings.TrimSpace(string(statusOut)) != ""
					}

					ts.Worktrees = append(ts.Worktrees, ws)
				}

				statuses = append(statuses, ts)
			}

			if jsonOut {
				data, err := json.MarshalIndent(statuses, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal json: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			// Pretty print.
			for i, ts := range statuses {
				if i > 0 {
					fmt.Println()
				}

				statusStr := ""
				if ts.Status != "" {
					statusStr = " " + colorStatus(slate.Status(ts.Status))
				}
				personaStr := ""
				if ts.Persona != "" {
					personaStr = fmt.Sprintf(" (%s)", ts.Persona)
				}

				titleStr := ""
				if ts.Title != "" {
					titleStr = " — " + ts.Title
				}

				fmt.Printf("%s%s%s%s\n", bold(ts.TaskID), statusStr, personaStr, titleStr)

				// Show worktrees.
				for _, wt := range ts.Worktrees {
					dirtyStr := colorize(colorGreen, "clean")
					if wt.Dirty {
						dirtyStr = colorize(colorYellow, "dirty")
					}
					branchStr := ""
					if wt.Branch != "" {
						branchStr = fmt.Sprintf(" (%s)", wt.Branch)
					}
					fmt.Printf("  %-20s %s%s\n", wt.Repo, dirtyStr, branchStr)
				}

				// Show latest checkpoint.
				if ts.LatestCheckpoint != nil {
					fmt.Printf("  %s %s\n", colorize(colorGray, "checkpoint:"), ts.LatestCheckpoint.Done)
					if ts.LatestCheckpoint.Next != "" {
						fmt.Printf("  %s %s\n", colorize(colorGray, "next:"), ts.LatestCheckpoint.Next)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "Include completed/cancelled tasks")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

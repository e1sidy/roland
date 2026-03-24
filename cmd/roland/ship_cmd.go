package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/roland/internal/gitutil"
	"github.com/e1sidy/roland/workspace"
	"github.com/spf13/cobra"
)

func shipCmd() *cobra.Command {
	var (
		repoFlag      string
		draft         bool
		dryRun        bool
		title         string
		body          string
		requireReview bool
	)

	cmd := &cobra.Command{
		Use:   "ship [task-id]",
		Short: "Push branches and create pull requests",
		Long: `Discovers worktrees in the task workspace, pushes branches to remote,
and creates pull requests via 'gh pr create'.

Use --repo to ship only a specific repo's worktree.
Use --dry-run to see what would happen without making changes.
Use --require-review to block shipping if review_status is not approved.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, taskDir, err := resolveTaskID(cfg.Home, args)
			if err != nil {
				return fmt.Errorf("resolve task: %w", err)
			}

			ctx := context.Background()

			// Quality gate: check review_status.
			// Always warn if changes_requested; hard-block if --require-review and not approved.
			store, storeErr := openSlateStore(cfg)
			if storeErr == nil {
				attr, attrErr := store.GetAttr(ctx, taskID, roland.AttrReviewStatus)
				store.Close()
				if attrErr == nil {
					if attr.Value == "changes_requested" {
						fmt.Fprintf(os.Stderr, "⚠ Warning: review status is %q for task %s\n", attr.Value, taskID)
					}
					if requireReview && attr.Value != "approved" {
						return fmt.Errorf("review status is %q (not approved); shipping blocked by --require-review", attr.Value)
					}
				} else if requireReview {
					return fmt.Errorf("review status not set; shipping blocked by --require-review")
				}
			} else if requireReview {
				return fmt.Errorf("open store for review check: %w", storeErr)
			}

			// Discover worktrees in task directory.
			entries, err := os.ReadDir(taskDir)
			if err != nil {
				return fmt.Errorf("read task dir: %w", err)
			}

			shipped := 0
			for _, e := range entries {
				linkPath := filepath.Join(taskDir, e.Name())
				if !gitutil.IsSymlink(linkPath) {
					continue
				}

				repoName := e.Name()
				if repoFlag != "" && repoName != repoFlag {
					continue
				}

				target, err := os.Readlink(linkPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not read symlink %s: %v\n", linkPath, err)
					continue
				}
				// Resolve relative symlinks.
				if !filepath.IsAbs(target) {
					target = filepath.Join(taskDir, target)
				}

				// Get current branch.
				branchOut, err := gitutil.Output(target, "rev-parse", "--abbrev-ref", "HEAD")
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not get branch for %s: %v\n", repoName, err)
					continue
				}
				branch := strings.TrimSpace(string(branchOut))

				// Get base branch for PR target.
				baseBranch := workspace.ReadBaseBranch(target)
				// Strip origin/ prefix for PR base.
				prBase := strings.TrimPrefix(baseBranch, "origin/")

				if dryRun {
					fmt.Printf("[dry-run] %s: would push %s and create PR against %s\n", repoName, branch, prBase)
					shipped++
					continue
				}

				// Push branch.
				fmt.Printf("Pushing %s/%s...\n", repoName, branch)
				if err := gitutil.Run(target, "push", "-u", "origin", branch); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: push failed for %s: %v\n", repoName, err)
					continue
				}

				// Create PR via gh.
				prTitle := title
				if prTitle == "" {
					prTitle = fmt.Sprintf("[%s] %s", taskID, branch)
				}
				prBody := body
				if prBody == "" {
					// Try PR template from repo config.
					rc := cfg.Repos[repoName]
					if rc != nil && rc.PRTemplate != "" {
						prBody = renderPRTemplate(rc.PRTemplate, taskID, branch)
					}
				}
				if prBody == "" {
					prBody = fmt.Sprintf("Task: %s\nBranch: %s\n\nCreated by Roland.", taskID, branch)
				}

				ghArgs := []string{"pr", "create",
					"--base", prBase,
					"--head", branch,
					"--title", prTitle,
					"--body", prBody,
				}
				if draft {
					ghArgs = append(ghArgs, "--draft")
				}

				ghCmd := exec.Command("gh", ghArgs...)
				ghCmd.Dir = target
				ghCmd.Stdout = os.Stdout
				ghCmd.Stderr = os.Stderr
				if err := ghCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: PR creation failed for %s: %v\n", repoName, err)
					continue
				}

				shipped++
			}

			if repoFlag != "" && shipped == 0 {
				return fmt.Errorf("repo %q not found in task workspace", repoFlag)
			}

			if shipped == 0 {
				fmt.Println("No worktrees found to ship.")
			} else {
				fmt.Printf("\nShipped %d repo(s) for task %s.\n", shipped, taskID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repoFlag, "repo", "", "Ship only this repo")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create PRs as draft")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without making changes")
	cmd.Flags().StringVar(&title, "title", "", "PR title (default: auto-generated)")
	cmd.Flags().StringVar(&body, "body", "", "PR body (default: auto-generated)")
	cmd.Flags().BoolVar(&requireReview, "require-review", false, "Block shipping unless review_status is approved")

	return cmd
}

// renderPRTemplate loads and renders a PR template file with task variables.
func renderPRTemplate(templatePath, taskID, branch string) string {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "" // fallback to default
	}

	// Fetch task details if store is available.
	taskTitle, taskDesc, taskPriority, cpText := taskID, "", "", ""
	store, err := openSlateStore(cfg)
	if err == nil {
		ctx := context.Background()
		task, terr := store.GetFull(ctx, taskID)
		if terr == nil {
			taskTitle = task.Title
			taskDesc = task.Description
			taskPriority = task.Priority.String()
		}
		cp, cerr := store.LatestCheckpoint(ctx, taskID)
		if cerr == nil && cp != nil {
			cpText = cp.Done
		}
		store.Close()
	}

	r := strings.NewReplacer(
		"{id}", taskID,
		"{title}", taskTitle,
		"{description}", taskDesc,
		"{checkpoint}", cpText,
		"{branch}", branch,
		"{priority}", taskPriority,
	)
	return r.Replace(string(data))
}

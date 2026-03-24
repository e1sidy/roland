package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/e1sidy/roland"
	rolandembed "github.com/e1sidy/roland/embed"
	"github.com/e1sidy/roland/hooks"
	"github.com/e1sidy/roland/persona"
	"github.com/e1sidy/roland/skill"
	"github.com/e1sidy/roland/workspace"
	"github.com/e1sidy/slate"
	"github.com/spf13/cobra"
)

func pickupCmd() *cobra.Command {
	var (
		personaName string
		repos       string
	)

	cmd := &cobra.Command{
		Use:   "pickup <task-id>",
		Short: "Claim a task, create workspace, and launch agent",
		Long: `Orchestrates the full pickup flow:

  1. Open Slate store and ensure Roland attributes exist
  2. Get task details from Slate
  3. Claim the task (atomic, prevents double-claim)
  4. Create task workspace directory
  5. Set repos and persona_used attributes
  6. Create git worktrees for each repo
  7. Generate CLAUDE.md with persona + task context
  8. Install hooks and inject matching skills

Then launches the configured AI coding agent in the workspace.

Steps 1-5 use strict rollback on failure.
Steps 6-7 warn on failure but continue (non-critical).
Agent launch failure does not rollback.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			ctx := context.Background()

			// Parse repos flag.
			var repoNames []string
			if repos != "" {
				repoNames = strings.Split(repos, ",")
			} else {
				// Default: all registered repos.
				for name := range cfg.Repos {
					repoNames = append(repoNames, name)
				}
			}

			// Validate repos exist in config.
			for _, name := range repoNames {
				if _, ok := cfg.Repos[name]; !ok {
					return fmt.Errorf("repo %q not registered; use 'roland repo add' first", name)
				}
			}

			// Validate persona.
			if !persona.IsValid(cfg.Home, personaName) {
				return fmt.Errorf("persona %q not found; use 'roland persona list' to see available personas", personaName)
			}

			// === STEP 1: Open Slate store + EnsureAttrs ===
			store, err := openSlateStore(cfg)
			if err != nil {
				return fmt.Errorf("step 1 (open store): %w", err)
			}
			defer store.Close()

			if err := roland.EnsureAttrs(ctx, store); err != nil {
				return fmt.Errorf("step 1 (ensure attrs): %w", err)
			}
			fmt.Println("Step 1: Slate store opened, attributes ensured.")

			// === STEP 2: Get task from Slate ===
			task, err := store.GetFull(ctx, taskID)
			if err != nil {
				return fmt.Errorf("step 2 (get task): %w", err)
			}
			fmt.Printf("Step 2: Task %s — %s\n", task.ID, task.Title)

			// === STEP 3: Claim task ===
			claimResult, err := store.Claim(ctx, taskID, "roland")
			if err != nil {
				return fmt.Errorf("step 3 (claim): %w", err)
			}
			if claimResult.ParentProgressed {
				fmt.Printf("  Parent %s auto-progressed to in_progress\n", claimResult.ParentID)
			}
			fmt.Println("Step 3: Task claimed.")

			// From here, errors need rollback of the claim.
			rollbackClaim := func() {
				if releaseErr := store.ReleaseClaim(ctx, taskID, "roland"); releaseErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to release claim: %v\n", releaseErr)
				}
			}

			// === STEP 4: Create workspace ===
			td, err := workspace.Create(cfg.Home, taskID, task.Title)
			if err != nil {
				rollbackClaim()
				return fmt.Errorf("step 4 (create workspace): %w", err)
			}
			fmt.Printf("Step 4: Workspace created at %s\n", td.Path)

			// From here, errors also need workspace cleanup.
			rollbackWorkspace := func() {
				if rmErr := workspace.Remove(cfg.Home, taskID); rmErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove workspace: %v\n", rmErr)
				}
				rollbackClaim()
			}

			// === STEP 5: Set repos + persona_used attrs ===
			repoJSON, err := json.Marshal(repoNames)
			if err != nil {
				rollbackWorkspace()
				return fmt.Errorf("step 5 (marshal repos): %w", err)
			}
			if err := store.SetAttr(ctx, taskID, roland.AttrRepos, string(repoJSON)); err != nil {
				rollbackWorkspace()
				return fmt.Errorf("step 5 (set repos attr): %w", err)
			}
			if err := store.SetAttr(ctx, taskID, roland.AttrPersonaUsed, personaName); err != nil {
				rollbackWorkspace()
				return fmt.Errorf("step 5 (set persona attr): %w", err)
			}
			fmt.Println("Step 5: Attributes set.")

			// === STEP 6: Create worktrees ===
			// Track created worktrees for rollback.
			var createdWorktrees []struct{ repo, branch string }

			for _, repoName := range repoNames {
				rc := cfg.Repos[repoName]
				baseBranch := rc.BaseBranch
				if baseBranch == "" {
					baseBranch = "origin/main"
				}

				// Resolve branch name (use script if configured, else taskID).
				taskJSON, _ := json.Marshal(task)
				branch, err := workspace.ResolveBranchName(rc.BranchName, taskJSON, taskID)
				if err != nil {
					// Rollback created worktrees.
					for _, wt := range createdWorktrees {
						if wtErr := workspace.WorktreeRemove(cfg.Home, wt.repo, wt.branch); wtErr != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree %s/%s: %v\n", wt.repo, wt.branch, wtErr)
						}
					}
					rollbackWorkspace()
					return fmt.Errorf("step 6 (resolve branch for %s): %w", repoName, err)
				}

				wtDir, err := workspace.WorktreeAdd(workspace.WorktreeOpts{
					Home:       cfg.Home,
					RepoName:   repoName,
					Branch:     branch,
					BaseBranch: baseBranch,
					TaskDir:    td.Path,
					PostSetup:  rc.PostSetup,
				})
				if err != nil {
					// Rollback created worktrees.
					for _, wt := range createdWorktrees {
						if wtErr := workspace.WorktreeRemove(cfg.Home, wt.repo, wt.branch); wtErr != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree %s/%s: %v\n", wt.repo, wt.branch, wtErr)
						}
					}
					rollbackWorkspace()
					return fmt.Errorf("step 6 (create worktree for %s): %w", repoName, err)
				}
				createdWorktrees = append(createdWorktrees, struct{ repo, branch string }{repoName, branch})
				fmt.Printf("Step 6: Worktree for %s at %s\n", repoName, wtDir)
			}

			// === STEP 7 (CLAUDE.md): Strict — part of core setup ===
			if err := writeTaskClaudeMD(td.Path, cfg.Home, personaName, task); err != nil {
				// Rollback worktrees + workspace + claim.
				for _, wt := range createdWorktrees {
					if wtErr := workspace.WorktreeRemove(cfg.Home, wt.repo, wt.branch); wtErr != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree %s/%s: %v\n", wt.repo, wt.branch, wtErr)
					}
				}
				rollbackWorkspace()
				return fmt.Errorf("step 7 (write CLAUDE.md): %w", err)
			}
			fmt.Println("Step 7: CLAUDE.md generated.")

			// === STEP 8 (hooks + skills): Warn and continue on failure ===
			reg := hooks.DefaultRegistry()
			mgr := hooks.NewManager(reg)
			enabled := hooks.EnabledForSource(cfg.Hooks, hooks.SourceTask, reg)
			hctx := hooks.HookContext{
				RolandHome: cfg.Home,
				TargetDir:  td.Path,
				SlateHome:  cfg.SlateHome,
			}
			if err := mgr.Sync(td.Path, enabled, cfg.Agent, hctx); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: hook installation failed: %v\n", err)
			} else {
				fmt.Println("Step 8: Hooks installed.")
			}

			// Inject matching skills.
			matchCtx := &skill.MatchContext{
				Persona:  personaName,
				TaskType: string(task.Type),
				Labels:   task.Labels,
			}
			injected, err := skill.InjectMatching(cfg.Home, td.Path, matchCtx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: skill injection failed: %v\n", err)
			} else if len(injected) > 0 {
				fmt.Printf("Step 8: Skills injected: %s\n", strings.Join(injected, ", "))
			}

			// === Launch agent ===
			fmt.Printf("\nLaunching %s in %s...\n", cfg.Agent, td.Path)
			flags := cfg.AgentFlags[cfg.Agent]
			return launchAgent(td.Path, cfg.Agent, flags)
		},
	}

	cmd.Flags().StringVar(&personaName, "persona", "builder", "Persona to use for the agent")
	cmd.Flags().StringVar(&repos, "repos", "", "Comma-separated repo names (default: all registered)")

	return cmd
}

// writeTaskClaudeMD generates a CLAUDE.md combining persona content with task context.
func writeTaskClaudeMD(taskDir, home, personaName string, task *slate.Task) error {
	// Get persona content (with repo-aware resolution).
	// Determine primary repo for persona override lookup.
	primaryRepo := ""
	entries2, _ := os.ReadDir(taskDir)
	for _, e := range entries2 {
		if e.Name() != "CLAUDE.md" && e.Name() != ".claude" {
			primaryRepo = e.Name()
			break
		}
	}
	personaContent, err := persona.ResolvePersona(home, personaName, primaryRepo)
	if err != nil {
		return fmt.Errorf("get persona %q: %w", personaName, err)
	}

	// Build workspace layout by listing symlinks in task dir.
	var layout strings.Builder
	entries, _ := os.ReadDir(taskDir)
	for _, e := range entries {
		if e.Name() == "CLAUDE.md" || e.Name() == ".claude" {
			continue
		}
		layout.WriteString(fmt.Sprintf("  %s/\n", e.Name()))
	}

	// Build labels string.
	labelsStr := "none"
	if len(task.Labels) > 0 {
		labelsStr = strings.Join(task.Labels, ", ")
	}

	// Compose the CLAUDE.md content.
	var buf strings.Builder
	buf.WriteString(personaContent)
	buf.WriteString("\n\n---\n\n")
	buf.WriteString("# Task Context\n\n")
	buf.WriteString(fmt.Sprintf("- **Task ID**: %s\n", task.ID))
	buf.WriteString(fmt.Sprintf("- **Title**: %s\n", task.Title))
	if task.Description != "" {
		buf.WriteString(fmt.Sprintf("- **Description**: %s\n", task.Description))
	}
	buf.WriteString(fmt.Sprintf("- **Priority**: %s\n", task.Priority.String()))
	buf.WriteString(fmt.Sprintf("- **Type**: %s\n", task.Type))
	buf.WriteString(fmt.Sprintf("- **Labels**: %s\n", labelsStr))
	buf.WriteString(fmt.Sprintf("\n## Workspace Layout\n\n```\n%s/\n%s```\n", workspace.ExtractTaskID(taskDir), layout.String()))

	// Also write the embedded base CLAUDE.md content.
	buf.WriteString("\n---\n\n")
	buf.WriteString(rolandembed.DefaultClaudeMD)

	dest := fmt.Sprintf("%s/CLAUDE.md", taskDir)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return fmt.Errorf("create task dir: %w", err)
	}
	return os.WriteFile(dest, []byte(buf.String()), 0o644)
}

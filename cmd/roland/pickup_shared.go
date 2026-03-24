package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/e1sidy/roland"
	rolandembed "github.com/e1sidy/roland/embed"
	"github.com/e1sidy/roland/hooks"
	"github.com/e1sidy/roland/persona"
	"github.com/e1sidy/roland/skill"
	"github.com/e1sidy/roland/workspace"
	"github.com/e1sidy/slate"
)

// setupOpts holds parameters for the shared pickup/delegate setup flow.
type setupOpts struct {
	Store       *slate.Store
	Cfg         *roland.Config
	TaskID      string
	PersonaName string
	RepoNames   []string
	Quiet       bool // suppress step output
}

// setupResult holds the outcome of a setup flow.
type setupResult struct {
	TaskDir *workspace.TaskDir
	Task    *slate.Task
}

// setupWorkspace performs the common pickup/delegate setup:
// claim → workspace → attrs → worktrees → CLAUDE.md → hooks → skills.
// Returns the created workspace and task. On failure, rolls back completed steps.
func setupWorkspace(ctx context.Context, opts setupOpts) (*setupResult, error) {
	store := opts.Store
	taskID := opts.TaskID

	// Get task.
	task, err := store.GetFull(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task %s: %w", taskID, err)
	}

	// Claim.
	_, err = store.Claim(ctx, taskID, "roland")
	if err != nil {
		return nil, fmt.Errorf("claim %s: %w", taskID, err)
	}
	rollbackClaim := func() {
		store.ReleaseClaim(ctx, taskID, "roland")
	}

	// Create workspace.
	td, err := workspace.Create(opts.Cfg.Home, taskID, task.Title)
	if err != nil {
		rollbackClaim()
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	rollbackWorkspace := func() {
		workspace.Remove(opts.Cfg.Home, taskID)
		rollbackClaim()
	}

	// Set attrs.
	repoJSON, _ := json.Marshal(opts.RepoNames)
	if err := store.SetAttr(ctx, taskID, roland.AttrRepos, string(repoJSON)); err != nil {
		rollbackWorkspace()
		return nil, fmt.Errorf("set repos attr: %w", err)
	}
	if err := store.SetAttr(ctx, taskID, roland.AttrPersonaUsed, opts.PersonaName); err != nil {
		rollbackWorkspace()
		return nil, fmt.Errorf("set persona attr: %w", err)
	}

	// Create worktrees.
	var createdWorktrees []struct{ repo, branch string }
	rollbackWorktrees := func() {
		for _, wt := range createdWorktrees {
			workspace.WorktreeRemove(opts.Cfg.Home, wt.repo, wt.branch)
		}
		rollbackWorkspace()
	}

	taskJSON, _ := json.Marshal(task)
	for _, repoName := range opts.RepoNames {
		rc := opts.Cfg.Repos[repoName]
		if rc == nil {
			continue
		}
		baseBranch := rc.BaseBranch
		if baseBranch == "" {
			baseBranch = "origin/main"
		}
		branch, err := workspace.ResolveBranchName(rc.BranchName, taskJSON, taskID)
		if err != nil {
			rollbackWorktrees()
			return nil, fmt.Errorf("resolve branch for %s: %w", repoName, err)
		}
		_, err = workspace.WorktreeAdd(workspace.WorktreeOpts{
			Home:       opts.Cfg.Home,
			RepoName:   repoName,
			Branch:     branch,
			BaseBranch: baseBranch,
			TaskDir:    td.Path,
			PostSetup:  rc.PostSetup,
		})
		if err != nil {
			rollbackWorktrees()
			return nil, fmt.Errorf("worktree for %s: %w", repoName, err)
		}
		createdWorktrees = append(createdWorktrees, struct{ repo, branch string }{repoName, branch})
	}

	// Generate CLAUDE.md.
	if err := writeTaskClaudeMDResolved(td.Path, opts.Cfg.Home, opts.PersonaName, opts.RepoNames, task); err != nil {
		rollbackWorktrees()
		return nil, fmt.Errorf("write CLAUDE.md: %w", err)
	}

	// Install hooks + skills (warn on failure).
	installHooksAndSkills(td.Path, opts.Cfg, opts.PersonaName, string(task.Type))

	return &setupResult{TaskDir: td, Task: task}, nil
}

// writeTaskClaudeMDResolved generates CLAUDE.md using ResolvePersona for repo-aware persona.
func writeTaskClaudeMDResolved(taskDir, home, personaName string, repos []string, task *slate.Task) error {
	// Determine primary repo for persona resolution.
	primaryRepo := ""
	if len(repos) > 0 {
		primaryRepo = repos[0]
	}

	personaContent, err := persona.ResolvePersona(home, personaName, primaryRepo)
	if err != nil {
		return fmt.Errorf("resolve persona %q: %w", personaName, err)
	}

	var layout strings.Builder
	entries, _ := os.ReadDir(taskDir)
	for _, e := range entries {
		if e.Name() == "CLAUDE.md" || e.Name() == ".claude" {
			continue
		}
		layout.WriteString(fmt.Sprintf("  %s/\n", e.Name()))
	}

	labelsStr := "none"
	if len(task.Labels) > 0 {
		labelsStr = strings.Join(task.Labels, ", ")
	}

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
	buf.WriteString("\n---\n\n")
	buf.WriteString(rolandembed.DefaultClaudeMD)

	dest := fmt.Sprintf("%s/CLAUDE.md", taskDir)
	return os.WriteFile(dest, []byte(buf.String()), 0o644)
}

// installHooksAndSkills installs hooks and injects skills (warn on failure).
func installHooksAndSkills(taskDir string, cfg *roland.Config, personaName, taskType string) {
	reg := hooks.DefaultRegistry()
	mgr := hooks.NewManager(reg)
	enabled := hooks.EnabledForSource(cfg.Hooks, hooks.SourceTask, reg)
	hctx := hooks.HookContext{
		RolandHome: cfg.Home,
		TargetDir:  taskDir,
		SlateHome:  cfg.SlateHome,
	}
	if err := mgr.Sync(taskDir, enabled, cfg.Agent, hctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: hook installation failed: %v\n", err)
	}

	matchCtx := &skill.MatchContext{
		Persona:  personaName,
		TaskType: taskType,
	}
	if _, err := skill.InjectMatching(cfg.Home, taskDir, matchCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: skill injection failed: %v\n", err)
	}
}

// launchAgentBackground launches the agent as a background process (for delegate).
// Returns the PID of the launched process.
func launchAgentBackground(dir string, agent roland.AgentTool, flags []string) (int, error) {
	binary, err := exec.LookPath(agent.Command())
	if err != nil {
		return 0, fmt.Errorf("agent %q not found in PATH: %w", agent.Command(), err)
	}

	argv := []string{binary}
	argv = append(argv, flags...)
	switch agent {
	case roland.AgentClaude:
		argv = append(argv, "--dir", dir)
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start agent: %w", err)
	}

	// Release the process so it doesn't become a zombie.
	go cmd.Wait()

	return cmd.Process.Pid, nil
}

// getWorktreeSymlinks reads worktree symlinks from a task directory.
func getWorktreeSymlinks(taskDir string) map[string]string {
	links := make(map[string]string) // repoName → target path
	entries, _ := os.ReadDir(taskDir)
	for _, e := range entries {
		if e.Name() == "CLAUDE.md" || e.Name() == ".claude" {
			continue
		}
		full := taskDir + "/" + e.Name()
		target, err := os.Readlink(full)
		if err == nil {
			links[e.Name()] = target
		}
	}
	return links
}

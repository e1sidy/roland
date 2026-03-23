# Architecture

## System Overview

Roland is a workspace orchestration layer that sits between AI coding agents and the Slate task management system. It manages the full lifecycle of agent-driven development through 6 subsystems.

```
┌─────────────────────────────────────────────────────┐
│                    AI Coding Agent                    │
│               (Claude Code / OpenCode)                │
└────────────────────────┬────────────────────────────┘
                         │ exec / launch
┌────────────────────────▼────────────────────────────┐
│                      Roland CLI                      │
│  pickup · work · checkpoint · ship · done · status   │
├──────────────────────────────────────────────────────┤
│  ┌──────────┐ ┌──────────┐ ┌───────┐ ┌──────────┐  │
│  │ Workspace│ │  Persona │ │ Hooks │ │  Skills  │  │
│  │ Manager  │ │  System  │ │ System│ │  System  │  │
│  └────┬─────┘ └──────────┘ └───────┘ └──────────┘  │
│       │                                              │
│  ┌────▼─────┐                                        │
│  │ Worktree │                                        │
│  │ Manager  │                                        │
│  └────┬─────┘                                        │
├───────┼──────────────────────────────────────────────┤
│       │         Slate SDK (Go library)               │
│       │    Store · Claim · Checkpoint · Attrs        │
│       │                                              │
│  ┌────▼─────┐                                        │
│  │  SQLite  │                                        │
│  │ (slate.db)│                                        │
│  └──────────┘                                        │
└──────────────────────────────────────────────────────┘
```

## Subsystems

### 1. Workspace Manager (`workspace/`)

Manages ephemeral task directories under `ROLAND_HOME/tasks/`. Each task gets a directory named with a slug derived from the task ID and title (e.g., `st-a1b2-fix-auth-bug/`). The directory contains symlinks to git worktrees.

Key operations:
- `Create(home, taskID, title)` — create a task directory
- `Open(home, taskID)` — find an existing task directory by ID prefix match
- `Remove(home, taskID)` — delete a task directory
- `List(home)` — enumerate all task workspaces
- `ExtractTaskID(slug)` — parse the Slate task ID from a directory name

### 2. Worktree Manager (`workspace/`)

Manages git worktrees under `ROLAND_HOME/worktrees/`. Each repo gets a subdirectory, and each task branch gets its own worktree within that. Worktrees are symlinked into task directories.

Key operations:
- `WorktreeAdd(opts)` — fetch, create worktree, run post-setup, symlink
- `WorktreeRemove(home, repo, branch)` — remove a worktree
- `WorktreeList(home, repo)` — list worktree branches for a repo
- `ResolveBranchName(script, taskJSON, default)` — determine branch name from script or fallback
- `ReadBaseBranch(wtDir)` — read the PR target branch from `.roland-base`

### 3. Persona System (`persona/`)

Provides behavior templates for AI agents. Roland ships 4 built-in personas embedded in the binary:

| Persona      | Role |
|-------------|------|
| `builder`    | Implementation-focused, writes code and tests |
| `researcher` | Investigation-focused, explores and documents |
| `reviewer`   | Quality-focused, reviews code and suggests fixes |
| `planner`    | Planning-focused, breaks work into tasks |

Custom personas are stored as Markdown files in `ROLAND_HOME/personas/`. Custom personas with the same name as a built-in take precedence.

### 4. Hook System (`hooks/`)

Injects context into AI agent sessions through a 4-layer architecture:

1. **Hook definitions** — types, content generators, event/matcher config
2. **Registry** — collection of hooks, queried by name or source
3. **Manager** — orchestrates install/uninstall/sync across delivery targets
4. **Delivery** — per-agent: Claude Code bash scripts + settings.json, OpenCode JS plugin

Built-in hooks (5):

| Hook | Source | Purpose |
|------|--------|---------|
| `slate-instructions` | home | Slate CLI cheatsheet |
| `slate-ready-tasks` | home | Output of `slate ready` |
| `roland-instructions` | home | Roland CLI cheatsheet |
| `roland-repos` | home | Registered repo list |
| `roland-task-context` | task | Current task + latest checkpoint |

### 5. Skill System (`skill/`)

Manages reusable context directories that are auto-injected into task workspaces. Skills are registered in `ROLAND_HOME/.skills/skills.json` and injected via symlinks into `.claude/skills/` within task directories.

Matching uses **OR logic** across three dimensions:
- **Personas** — skill matches if the active persona is in the skill's persona list
- **Task types** — skill matches if the task type is in the skill's type list
- **Tags/labels** — skill matches if any task label overlaps with the skill's tag list

If all dimensions are empty, the skill is manual-only (never auto-injected).

### 6. Configuration (`config.go`, `roland.yaml`)

Manages Roland settings: default agent, IDE preference, per-agent flags, repo registry, and hook enable/disable state. Configuration is loaded from `ROLAND_HOME/roland.yaml`.

Home resolution cascade:
1. `ROLAND_HOME` environment variable
2. `~/.config/roland/home` pointer file
3. `~/.roland/` (default)

## Data Flow: Pickup to Done

### `roland pickup st-a1b2`

```
Step 1: Open Slate store → EnsureAttrs (idempotent)
Step 2: GetFull(taskID) → retrieve task details
Step 3: Claim(taskID, "roland") → atomic claim, prevents double-pickup
Step 4: workspace.Create(home, taskID, title) → create task directory
Step 5: SetAttr(repos, persona_used) → store metadata in Slate
Step 6: For each repo:
        ├── Fetch remote
        ├── Create worktree with new branch from base
        ├── Write .roland-base file
        ├── Run post-setup script (if configured)
        └── Symlink into task directory
Step 7: Write CLAUDE.md (persona content + task context)
Step 8: Install hooks + inject matching skills (warn on failure)
  ──→   Launch agent (exec replaces the Roland process)
```

Rollback: Steps 1-5 have strict rollback. If step 4 fails, the claim from step 3 is released. If step 6 fails, previously created worktrees are removed, the workspace is deleted, and the claim is released.

### `roland work st-a1b2`

```
1. Resolve task ID (from argument or cwd)
2. Show resumption briefing:
   ├── Latest checkpoint (done, next, blockers)
   └── Git status per worktree (clean/dirty)
3. Launch agent in task directory
```

### `roland ship`

```
1. Resolve task ID
2. (Optional) Check review_status if --require-review
3. For each worktree in task directory:
   ├── Read current branch
   ├── Read base branch from .roland-base
   ├── Push branch to origin
   └── Create PR via `gh pr create`
```

### `roland done st-a1b2`

```
1. Resolve task ID
2. Discover worktrees in task directory
3. Check PR merge status (unless --force)
4. Remove all worktrees
5. Remove task workspace directory
6. Close task in Slate
```

## ROLAND_HOME Filesystem Layout

```
~/.roland/                     # ROLAND_HOME (default)
├── roland.yaml                # Configuration
├── CLAUDE.md                  # Home-level agent instructions
├── .claude/
│   └── settings.json          # Claude Code settings (hooks wired here)
├── hooks/                     # Hook script files (.sh)
├── repos/                     # Cloned repositories
│   ├── backend/               # git clone of backend repo
│   └── frontend/              # git clone of frontend repo
├── worktrees/                 # Git worktrees (per-repo, per-branch)
│   ├── backend/
│   │   └── st-a1b2/           # Worktree for task st-a1b2
│   └── frontend/
│       └── st-a1b2/
├── tasks/                     # Ephemeral task workspaces
│   └── st-a1b2-fix-auth-bug/  # Task directory
│       ├── CLAUDE.md           # Persona + task context
│       ├── .claude/
│       │   └── skills/         # Symlinked skill directories
│       ├── backend → ../../worktrees/backend/st-a1b2
│       └── frontend → ../../worktrees/frontend/st-a1b2
├── personas/                  # Custom persona files (.md)
└── .skills/                   # Skill registry
    ├── skills.json            # Skill metadata
    └── <skill-name>/          # Copied skill directories
```

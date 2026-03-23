# Getting Started

This guide walks through installing Roland, setting up your first repository, and running a full task lifecycle.

## Prerequisites

- **Go 1.25+** — [golang.org/dl](https://go.dev/dl/)
- **Git** — any recent version
- **Slate** — installed and initialized (`go install github.com/e1sidy/slate/cmd/slate@latest && slate init`)
- **An AI coding agent** — [Claude Code](https://docs.anthropic.com/en/docs/claude-code) or [OpenCode](https://opencode.ai)
- **GitHub CLI** (`gh`) — required for `roland ship` to create pull requests

## Installation

### From source

```bash
# Clone both repos (Roland depends on Slate via replace directive)
git clone https://github.com/e1sidy/slate.git
git clone https://github.com/e1sidy/roland.git
cd roland
go build -o roland ./cmd/roland

# Move to a directory in your PATH
mv roland /usr/local/bin/
```

### Via `go install`

```bash
go install github.com/e1sidy/roland/cmd/roland@latest
```

### Verify

```bash
roland version
# roland dev
```

## Initialize Roland

```bash
roland init
```

This creates:
- `~/.roland/` — the home directory (or wherever `ROLAND_HOME` points)
- `~/.roland/roland.yaml` — configuration file
- `~/.roland/repos/` — where repos are cloned
- `~/.roland/tasks/` — where task workspaces live
- `~/.roland/worktrees/` — where git worktrees are stored
- `~/.roland/personas/` — where custom personas go
- `~/.config/roland/home` — pointer file so Roland can find its home

To use the current directory instead:

```bash
roland init --here
```

## Register a Repository

```bash
roland repo add https://github.com/your-org/backend.git
# Added repo "backend" at /home/user/.roland/repos/backend
```

Roland clones the repo and registers it with the short name `backend` (derived from the URL). To use a custom name:

```bash
roland repo add https://github.com/your-org/backend.git --name api
```

List registered repos:

```bash
roland repo list
# backend              https://github.com/your-org/backend.git  (base: origin/main)
```

## Full Task Lifecycle

### Step 1: Create a Task in Slate

```bash
slate create "Fix authentication bug in login endpoint" --type bug --priority 2
# Created: st-x7k9
```

### Step 2: Pick Up the Task

```bash
roland pickup st-x7k9
```

This orchestrates 8 steps:
1. Opens the Slate store and ensures Roland attributes exist
2. Retrieves task details from Slate
3. Claims the task atomically (prevents double-pickup)
4. Creates a task workspace at `~/.roland/tasks/st-x7k9-fix-authentication-bug/`
5. Sets `repos` and `persona_used` attributes in Slate
6. Creates a git worktree for each registered repo
7. Generates a `CLAUDE.md` with persona instructions and task context
8. Installs hooks and injects matching skills

Then it launches the configured AI agent (Claude Code by default) in the workspace.

To use a specific persona:

```bash
roland pickup st-x7k9 --persona researcher
```

To use only specific repos:

```bash
roland pickup st-x7k9 --repos backend,frontend
```

### Step 3: Work on the Task

The agent is launched automatically after pickup. To resume later:

```bash
roland work st-x7k9
```

This shows a resumption briefing with the latest checkpoint and git status across worktrees, then re-launches the agent.

If you are already inside the task directory, the task ID is detected automatically:

```bash
cd ~/.roland/tasks/st-x7k9-fix-authentication-bug
roland work
```

### Step 4: Record Checkpoints

At any point during work, record progress:

```bash
roland checkpoint --done "Implemented auth middleware" --next "Add unit tests" --blockers "Need API key for external service"
```

Checkpoints are stored in Slate and shown during `roland work` resumption.

### Step 5: Ship Pull Requests

When the work is ready:

```bash
roland ship
# Pushing backend/st-x7k9...
# Creating PR...
# Shipped 1 repo(s) for task st-x7k9.
```

Preview what would happen without making changes:

```bash
roland ship --dry-run
```

Ship only one repo:

```bash
roland ship --repo backend
```

Create draft PRs:

```bash
roland ship --draft
```

### Step 6: Complete the Task

After PRs are merged:

```bash
roland done st-x7k9
```

This:
1. Verifies all PRs are merged (use `--force` to skip)
2. Removes all git worktrees for the task
3. Removes the task workspace directory
4. Closes the task in Slate

## What's Next

- [CLI Reference](cli-reference.md) — detailed docs for every command
- [Configuration](configuration.md) — customize agent, IDE, hooks, repo settings
- [Concepts](concepts.md) — deep dive into workspaces, worktrees, personas, hooks, skills
- [Architecture](architecture.md) — system design and data flow
- [SDK Reference](sdk-reference.md) — Go API documentation

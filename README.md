# Roland

Workspace orchestration for AI coding agents.

Roland manages the full lifecycle of agent-driven development: **pickup** a task, **work** on it with isolated git worktrees, **checkpoint** progress, **ship** pull requests, and mark it **done**. It integrates with [Slate](https://github.com/e1sidy/slate) for task state management and provides personas, hooks, skills, and workspace isolation.

## Features

- **Task lifecycle** — `pickup` claims a Slate task, creates a workspace, launches an agent; `done` cleans up and closes the task
- **Git worktree isolation** — each task gets its own branches, symlinked into an ephemeral workspace directory
- **Persona system** — 4 built-in personas (builder, researcher, reviewer, planner) plus custom persona support
- **Hook system** — context injection into Claude Code and OpenCode via a registry/manager/delivery architecture
- **Skill system** — reusable context directories auto-injected based on persona, task type, or labels (OR matching)
- **Atomic pickup with rollback** — if any step fails during pickup, all previous steps are rolled back
- **Session continuity** — `work` command shows latest checkpoint and git status for seamless resumption
- **Multi-agent support** — Claude Code and OpenCode, with per-agent flags
- **Quality gates** — optional `--require-review` blocks shipping until review is approved

## Quick Start

### Install

```bash
go install github.com/e1sidy/roland/cmd/roland@latest
```

### Initialize

```bash
roland init
```

This creates the `ROLAND_HOME` directory structure (default: `~/.roland/`), writes default configuration, installs hooks, and stores a pointer file at `~/.config/roland/home`.

### Register a Repository

```bash
roland repo add https://github.com/your-org/backend.git
```

### Full Lifecycle

```bash
# 1. Pick up a task from Slate
roland pickup st-a1b2

# 2. Work is done by the launched agent. Resume later:
roland work st-a1b2

# 3. Record progress at any time
roland checkpoint --done "Implemented auth middleware" --next "Add tests"

# 4. Push branches and create PRs
roland ship

# 5. After PRs are merged, close the task
roland done st-a1b2
```

### Other Useful Commands

```bash
roland status                  # See all active task workspaces
roland status --json           # Machine-readable output
roland open st-a1b2            # Open workspace in your IDE
roland config agent opencode   # Switch to OpenCode
roland persona list            # List available personas
roland skill list              # List registered skills
roland clean --dry-run         # Preview orphaned workspace cleanup
```

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/architecture.md) | System overview, data flow, filesystem layout |
| [Getting Started](docs/getting-started.md) | Installation and first-run walkthrough |
| [CLI Reference](docs/cli-reference.md) | Man-page style docs for every command |
| [Configuration](docs/configuration.md) | `roland.yaml`, environment variables, agent flags |
| [Concepts](docs/concepts.md) | Workspaces, worktrees, personas, hooks, skills |
| [SDK Reference](docs/sdk-reference.md) | Public Go API documentation |

## Build from Source

```bash
git clone https://github.com/e1sidy/roland.git
cd roland

# Slate must be cloned as a sibling (go.mod uses replace directive)
git clone https://github.com/e1sidy/slate.git ../slate

go build -o roland ./cmd/roland
go test ./... -count=1 -timeout 120s
```

### Version Injection

```bash
go build -ldflags "-X github.com/e1sidy/roland.Version=v0.1.0" -o roland ./cmd/roland
```

## Related Projects

- **[Slate](https://github.com/e1sidy/slate)** — Task management SDK and CLI. Roland uses Slate as its task layer for state tracking, checkpoints, dependencies, and custom attributes.

## License

See [LICENSE](LICENSE).

# Roland — AI Agent Workspace Orchestrator

## Module

```
github.com/e1sidy/roland
```

Go 1.25+. Depends on `github.com/e1sidy/slate` (task layer) via `replace` directive for local development.

## File Layout

```
roland/
├── cmd/roland/          # CLI — thin Cobra wrappers only
├── embed/               # Embedded assets (CLAUDE.md template, settings)
├── hooks/               # Hook system (registry, manager, delivery)
│   └── scripts/         # Embedded bash hook scripts
├── internal/
│   ├── gitutil/         # Git CLI wrapper helpers
│   └── testutil/        # Test helpers (TempHome, TempSlateStore)
├── persona/             # Persona system (built-in + custom)
│   └── templates/       # Embedded persona markdown files
├── schemas/             # JSON schemas (skills.json)
├── skill/               # Skill system (registry, matching, injection)
├── workspace/           # Task dirs + git worktree management
├── roland.go            # AgentTool, IDE types, Version
├── config.go            # Config, ResolveHome, LoadConfig, SaveConfig
├── repo.go              # AddRepo, RemoveRepo, SyncRepo
├── attrs.go             # EnsureAttrs (Roland custom attributes in Slate)
└── go.mod
```

## Build & Test

```bash
go build ./...                          # Build everything
go vet ./...                            # Static analysis
go test ./... -count=1 -timeout 120s    # Run all tests
go test -coverprofile=c.out ./          # SDK coverage
go build -o roland ./cmd/roland         # Build the binary
```

## Key Types

| Type              | Package   | Purpose                                   |
|-------------------|-----------|-------------------------------------------|
| `Config`          | roland    | Loaded from `roland.yaml`                 |
| `RepoConfig`      | roland    | Per-repo settings (URL, base branch, etc) |
| `AgentTool`       | roland    | Enum: `claude`, `opencode`                |
| `IDE`             | roland    | Enum: `vscode`, `cursor`, `windsurf`, `nvim` |
| `Repo`            | roland    | Registered codebase                       |
| `TaskDir`         | workspace | Active task workspace                     |
| `WorktreeOpts`    | workspace | Parameters for worktree creation          |
| `Hook`            | hooks     | Context injection definition              |
| `Registry`        | hooks     | Collection of hooks                       |
| `Manager`         | hooks     | Install/uninstall/sync orchestrator       |
| `HookContext`     | hooks     | Data for hook content generators          |
| `PersonaInfo`     | persona   | Persona metadata (name, source)           |
| `SkillEntry`      | skill     | Registered skill metadata                 |
| `SkillConfig`     | skill     | Skill registry (`skills.json`)            |
| `MatchContext`    | skill     | Context for skill auto-matching           |

## Conventions

1. **SDK-first**: All business logic lives in the root package or sub-packages (`workspace/`, `hooks/`, `persona/`, `skill/`). The CLI (`cmd/roland/`) is a thin Cobra wrapper — never put business logic there.

2. **`context.Context`**: First parameter on every public SDK method that touches Slate or performs I/O.

3. **Error wrapping**: Always `fmt.Errorf("context: %w", err)`. Never swallow errors.

4. **Rollback on failure**: The `pickup` flow uses explicit rollback functions. If step N fails, undo steps 1 through N-1.

5. **Idempotent operations**: `EnsureAttrs`, `Inject`, hook `Sync` — all safe to call repeatedly.

6. **Table-driven tests**: Use `tt := []struct{...}` pattern. Test names: `TestCreate_Basic`, `TestMatch_PersonaOnly`.

7. **Test helpers**: Use `testutil.TempHome(t)` and `testutil.TempSlateStore(t)` for isolated tests.

## What NOT to Do

- Do not put business logic in `cmd/roland/` — keep it in SDK packages.
- Do not use `fmt.Sprintf` for SQL values — Slate handles all SQL internally.
- Do not hardcode paths — use `ResolveHome()`, `ConfigPath()`, `ReposDir()`, etc.
- Do not skip error wrapping — every returned error must include context.
- Do not create files unnecessarily — prefer editing existing ones.

## Status Model (Slate)

Tasks flow through these statuses:

```
open → in_progress → closed
  ↓        ↓
  └→ cancelled ←─┘
  ↓        ↓
  └→ deferred ←──┘
  ↓
  └→ blocked
```

Roland uses `Claim` (atomic, prevents double-claim) to move a task to `in_progress`, and `CloseTask` to move it to `closed`. The `pickup` command orchestrates this full lifecycle.

## Custom Attributes

Roland defines these in Slate via `EnsureAttrs`:

| Key              | Type   | Description                          |
|------------------|--------|--------------------------------------|
| `repos`          | object | JSON array of repo names             |
| `persona_used`   | string | Persona assigned to the task         |
| `review_status`  | string | `pending`, `approved`, `changes_requested` |
| `session_count`  | string | Number of agent sessions             |

# Roland

You are operating inside a Roland-managed workspace.

## Home

Roland home: {{.Home}}

## Key Commands

```bash
# Progress tracking
roland checkpoint --done "what you accomplished" --next "what's next"

# Worktree management
roland worktree add <repo> <branch>   # Add a worktree
roland worktree list <repo>           # List worktrees

# Shipping
roland ship [--dry-run]               # Push branches + create PRs
roland ship --repo <name>             # Ship specific repo only

# Status
roland status                         # See all active tasks
```

## Rules

1. Always write checkpoints before stopping work.
2. Commit frequently with clear messages.
3. Run tests before considering work done.
4. If blocked, add a checkpoint with blockers and stop.

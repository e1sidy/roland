# Multi-Agent Coordination

Roland supports delegating work between personas and monitoring progress.

## Delegate

Create a workspace for a subtask and launch an agent in background:

```bash
roland delegate <subtask-id> --persona researcher --repos backend
```

If the subtask doesn't exist in Slate, it's created as a child of your current task.

The agent launches in background — you keep your terminal. Use `roland watch` to monitor.

## Watch

Monitor child task status changes:

```bash
roland watch                          # watch current task's children
roland watch st-ab12                  # watch specific parent
roland watch --interval 30s           # custom poll interval (default: 5m)
roland watch --stale 15               # alert after 15min without checkpoint
```

Reports:
- Status changes: "st-ab12.1: open → in_progress (persona: builder)"
- Stale alerts: "st-ab12.2: in_progress with no checkpoint for 30m"
- Completion: "All subtasks done — parent st-ab12 is ready to close"

Press Ctrl+C to stop.

## Handoff

Transfer a task to a different persona:

```bash
roland handoff st-ab12 --to reviewer
```

What happens:
1. Auto-checkpoints current state ("Handing off to reviewer")
2. Removes old workspace (preserves git worktrees)
3. Updates `persona_used` attribute in Slate
4. Creates new workspace with reviewer persona CLAUDE.md
5. Re-links existing worktrees
6. Launches agent with new persona

## Workflow Example

```bash
# Planner breaks down an epic
roland pickup st-epic1 --persona planner
# Planner creates subtasks, then delegates:
roland delegate st-epic1.1 --persona researcher --repos backend
roland delegate st-epic1.2 --persona builder --repos frontend

# Planner monitors progress
roland watch st-epic1

# When researcher finishes, hand off to builder
roland handoff st-epic1.1 --to builder

# When all subtasks close
roland done st-epic1
```

# Learning Engine

Roland analyzes completed task data to improve future agent sessions.

## How It Works

1. **Data sources**: Checkpoint decisions, blockers, file co-changes, close reasons
2. **Pattern extraction**: Find phrases recurring in 3+ tasks
3. **Grouping**: Patterns assigned to the persona that produced them
4. **User review**: Interactive approve/reject before applying
5. **Storage**: Approved patterns appended to persona files as `## Learned Patterns`

## Commands

### Analyze and Learn

```bash
roland learn                          # analyze recent completed tasks
roland learn --since 2026-03-01       # filter by date
roland learn --persona builder        # filter by persona
```

Interactive flow: each pattern is shown with its source tasks. Approve (Y) or skip (n).

### View Learnings

```bash
roland learn --show builder           # show current learned patterns
```

### Reset Learnings

```bash
roland learn --reset builder          # remove ## Learned Patterns section
```

## Estimation Calibration

Compare task estimates vs actual cycle times:

```bash
roland learn --calibration
roland learn --calibration --type bug
roland learn --calibration --since 2026-01-01
```

Output: median estimate vs actual per task type and persona, with ratio and sample size.

## Decision Library

Searchable index of all decisions from checkpoint `decisions` fields:

```bash
roland decisions                      # list recent
roland decisions --search "auth"      # keyword search
roland decisions --persona builder    # filter by persona
roland decisions --task st-ab12       # decisions from specific task
roland decisions --rebuild            # rebuild index from checkpoints
```

## Per-Project Persona Overrides

Persona resolution is repo-aware:

```
~/.roland/personas/builder.md           ← base
~/.roland/personas/backend-builder.md   ← auto-selected for backend repo
```

Create an override:
```bash
roland persona create backend-builder --from builder
roland persona edit backend-builder
```

When `roland pickup --persona builder --repos backend` runs, it automatically selects `backend-builder.md` if it exists.

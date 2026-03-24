# Pipeline Mode

Auto-sequenced agent workflows: completing one step triggers the next.

## Usage

```bash
roland pipeline feature --var title="Dark mode"
```

Applies the template, creates tasks, then auto-delegates and monitors:
1. Delegates `on_create` steps immediately
2. Polls for completions
3. When a step closes, auto-delegates the next `on_dep_done` step
4. Repeats until all steps complete

## Pipeline YAML

Add a `pipeline:` section to any template:

```yaml
name: feature-pipeline
description: Auto-sequenced feature development
vars:
  - name: title
    required: true
tasks:
  - id: research
    title: "Research: <<.title>>"
    persona: researcher
  - id: implement
    title: "Implement: <<.title>>"
    persona: builder
    deps: [research]
  - id: review
    title: "Review: <<.title>>"
    persona: reviewer
    deps: [implement]
pipeline:
  - step: research
    trigger: on_create
  - step: implement
    trigger: on_dep_done
  - step: review
    trigger: on_dep_done
```

## Triggers

| Trigger | When |
|---------|------|
| `on_create` | Immediately after template apply |
| `on_dep_done` | When all blocking dependencies close |

## Options

```bash
roland pipeline feature --var title="X" --interval 30s
```

| Flag | Default | Description |
|------|---------|-------------|
| `--var` | — | Template variables (repeatable) |
| `--interval` | 30s | Polling interval |

Press Ctrl+C to stop monitoring (tasks continue running).

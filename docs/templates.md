# Task Templates

Templates define reusable task structures with dependencies, personas, and variable substitution.

## Built-in Templates

| Template | Tasks | Flow |
|----------|-------|------|
| `bug-fix` | 4 | reproduce → fix → test → review |
| `feature` | 4 | research → implement → review → ship |
| `refactor` | 4 | analyze → refactor → test → review |
| `code-review` | 3 | review → feedback → verify |
| `incident-response` | 4 | triage → fix → RCA → preventive |

## Commands

### List Templates

```bash
roland template list
```

### Show Template Structure

```bash
roland template show feature
```

### Apply a Template

```bash
roland template apply feature --var title="Dark mode"
```

Creates a parent epic + child tasks with dependencies in Slate. Each task gets `persona_used` and `repos` attributes set for `roland pickup`.

### Create Template from Epic

```bash
roland template create st-epic1
```

Reverse-engineers a template from a completed epic's structure. Output as YAML — save to `~/.roland/templates/` to reuse.

### Decompose (AI-Suggested Structure)

```bash
roland template decompose st-task1
```

Finds similar completed epics and suggests a subtask structure based on patterns.

## Template YAML Format

Templates use `<<.var>>` delimiters (not `{{}}`) to avoid markdown conflicts:

```yaml
name: feature
description: Standard feature development
vars:
  - name: title
    required: true
  - name: scope
    default: all
tasks:
  - id: research
    title: "Research: <<.title>>"
    type: task
    persona: researcher
  - id: implement
    title: "Implement: <<.title>>"
    type: feature
    persona: builder
    deps: [research]
    labels: [<<.scope>>]
  - id: review
    title: "Review: <<.title>>"
    type: task
    persona: reviewer
    deps: [implement]
```

## Custom Templates

Save YAML files to `~/.roland/templates/`:

```bash
cp my-template.yaml ~/.roland/templates/
roland template list  # now shows your template
```

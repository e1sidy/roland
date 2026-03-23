# Configuration

## roland.yaml

Roland's configuration lives at `ROLAND_HOME/roland.yaml`. It is created by `roland init` and can be edited directly or through `roland config` commands.

### Full Example

```yaml
# Default AI coding agent: claude or opencode
agent: claude

# Per-agent CLI flags
agent_flags:
  claude:
    - "--dangerously-skip-permissions"
  opencode: []

# Preferred code editor: vscode, cursor, windsurf, nvim
ide: cursor

# Override Slate home directory (empty = use Slate default)
slate_home: ""

# Registered repositories
repos:
  backend:
    url: https://github.com/your-org/backend.git
    base_branch: origin/main
    branch_name: ""
    post_setup: ""
  frontend:
    url: https://github.com/your-org/frontend.git
    base_branch: origin/main
    branch_name: ./scripts/branch-name.sh
    post_setup: ./scripts/post-setup.sh

# Hook enable/disable overrides (true = enabled, false = disabled)
# Hooks not listed here default to enabled.
hooks:
  slate-instructions: true
  slate-ready-tasks: true
  roland-instructions: true
  roland-repos: true
  roland-task-context: true
```

### Field Reference

#### `agent`

| | |
|---|---|
| Type | string |
| Default | `claude` |
| Valid values | `claude`, `opencode` |
| CLI | `roland config agent [name]` |

The default AI coding agent launched by `roland pickup` and `roland work`.

#### `agent_flags`

| | |
|---|---|
| Type | map of string to string array |
| Default | `{claude: ["--dangerously-skip-permissions"], opencode: []}` |
| CLI | Edit `roland.yaml` directly |

Per-agent CLI flags passed when launching the agent. The key is the agent name, the value is an array of flag strings.

Example with custom flags:

```yaml
agent_flags:
  claude:
    - "--dangerously-skip-permissions"
    - "--model"
    - "claude-sonnet-4-20250514"
  opencode:
    - "--provider"
    - "anthropic"
```

#### `ide`

| | |
|---|---|
| Type | string |
| Default | `cursor` |
| Valid values | `vscode`, `cursor`, `windsurf`, `nvim` |
| CLI | `roland config ide [name]` |

The preferred code editor, used by `roland open`.

IDE to command mapping:

| IDE | Command |
|-----|---------|
| `vscode` | `code` |
| `cursor` | `cursor` |
| `windsurf` | `windsurf` |
| `nvim` | `nvim` |

#### `slate_home`

| | |
|---|---|
| Type | string |
| Default | `""` (empty = use Slate's default) |
| CLI | Edit `roland.yaml` directly |

Overrides the path to the Slate home directory. When empty, Roland uses Slate's default database path (`~/.config/slate/slate.db` or `SLATE_HOME`).

#### `repos`

| | |
|---|---|
| Type | map of string to RepoConfig |
| Default | `{}` |
| CLI | `roland repo add`, `roland repo remove`, `roland repo post-setup` |

Registered codebases. Each key is the short name (e.g., `backend`), and the value is a `RepoConfig` object.

##### RepoConfig fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | string | | Git clone URL |
| `base_branch` | string | `origin/main` | Default branch for worktree creation |
| `branch_name` | string | `""` | Script path for custom branch name generation |
| `post_setup` | string | `""` | Script path to run after worktree creation |

**`branch_name`**: If set, this script is executed with the task JSON on stdin. It should output the branch name to stdout. If the script outputs nothing, the task ID is used as the branch name.

**`post_setup`**: This script runs in the worktree directory after creation. Useful for dependency installation (`npm install`, `poetry install`, etc.). Relative paths are resolved from the repo source directory. Post-setup failures are non-fatal (warn and continue).

#### `hooks`

| | |
|---|---|
| Type | map of string to boolean |
| Default | `{}` (all hooks default to enabled) |
| CLI | `roland config hooks enable/disable`, `roland hook add/remove` |

Controls which hooks are installed. Hooks not present in this map default to enabled.

## Environment Variables

### ROLAND_HOME

| | |
|---|---|
| Default | Resolved via cascade (see below) |
| Example | `export ROLAND_HOME=/home/user/my-roland` |

The root directory for all Roland data. If not set, Roland uses the home resolution cascade.

### SLATE_HOME

| | |
|---|---|
| Default | Slate's own default (`~/.config/slate/`) |
| Example | `export SLATE_HOME=/home/user/.slate` |

Overrides the Slate home directory. Roland uses this to locate the `slate.db` file. The `slate_home` config field takes precedence over this variable.

### NO_COLOR

| | |
|---|---|
| Default | unset |
| Example | `export NO_COLOR=1` |

When set (to any value), disables ANSI color output in Roland CLI. Follows the [no-color.org](https://no-color.org) convention. Color is also disabled automatically when stdout is not a terminal.

### EDITOR

| | |
|---|---|
| Default | `vi` |
| Example | `export EDITOR=nvim` |

Used by `roland persona edit` to open persona files for editing.

## Home Resolution Cascade

Roland resolves its home directory in this order:

1. **`ROLAND_HOME` environment variable** — if set, used directly
2. **`~/.config/roland/home` pointer file** — if it exists and contains a non-empty path, that path is used
3. **`~/.roland/`** — default fallback

The pointer file is written by `roland init` and survives shell changes and cache clears. You can manually write it:

```bash
mkdir -p ~/.config/roland
echo "/path/to/my/roland" > ~/.config/roland/home
```

## Agent Flags

Agent flags are passed directly to the agent CLI when launching. They are configured per-agent in `agent_flags`.

### Claude Code

Default flags: `["--dangerously-skip-permissions"]`

Claude Code is launched with `--dir <task-dir>` appended automatically. Example effective command:

```bash
claude --dangerously-skip-permissions --dir /home/user/.roland/tasks/st-a1b2-fix-auth
```

### OpenCode

Default flags: `[]` (empty)

OpenCode is launched with the task directory as the working directory.

## Repo Configuration Examples

### Basic repo

```yaml
repos:
  backend:
    url: https://github.com/org/backend.git
    base_branch: origin/main
```

### Repo with custom branch naming

```yaml
repos:
  backend:
    url: https://github.com/org/backend.git
    base_branch: origin/main
    branch_name: ./scripts/branch-name.sh
```

The `branch-name.sh` script receives task JSON on stdin:

```bash
#!/bin/bash
# Read task JSON from stdin, output branch name
jq -r '"feat/" + .id + "-" + (.title | gsub("[^a-zA-Z0-9]"; "-") | ascii_downcase)'
```

### Repo with post-setup

```yaml
repos:
  frontend:
    url: https://github.com/org/frontend.git
    base_branch: origin/main
    post_setup: ./scripts/post-setup.sh
```

The `post-setup.sh` script runs in the worktree directory:

```bash
#!/bin/bash
npm install
cp .env.example .env
```

### Repo with develop as base

```yaml
repos:
  legacy-api:
    url: https://github.com/org/legacy-api.git
    base_branch: origin/develop
```

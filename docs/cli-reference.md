# CLI Reference

Man-page style reference for all Roland commands. Commands are grouped by category.

---

## roland

```
roland [command]
```

Workspace orchestration for AI coding agents.

Roland manages the full lifecycle of agent-driven development: pickup, work, checkpoint, ship, done. It integrates with Slate for task management and provides personas, hooks, skills, and workspace isolation via git worktrees.

**Global behavior:** Every subcommand (except `init`, `completion`, `version`, and `help`) loads configuration from `ROLAND_HOME/roland.yaml` on startup. If the config cannot be loaded, the command fails.

---

## Lifecycle Commands

### roland pickup

```
roland pickup <task-id> [flags]
```

Claim a task, create workspace, and launch agent.

Orchestrates the full pickup flow:

1. Open Slate store and ensure Roland attributes exist
2. Get task details from Slate
3. Claim the task (atomic, prevents double-claim)
4. Create task workspace directory
5. Set `repos` and `persona_used` attributes
6. Create git worktrees for each repo
7. Generate CLAUDE.md with persona + task context
8. Install hooks and inject matching skills
9. Launch the configured AI coding agent

Steps 1-7 use strict rollback on failure. Step 8 warns on failure but continues. Agent launch failure does not rollback.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--persona` | `builder` | Persona to use for the agent |
| `--repos` | all registered | Comma-separated repo names |

**Examples:**

```bash
roland pickup st-a1b2
roland pickup st-a1b2 --persona researcher
roland pickup st-a1b2 --repos backend,frontend
```

**Side effects:**
- Claims the task in Slate (sets assignee to "roland")
- Creates `ROLAND_HOME/tasks/<slug>/` directory
- Creates git worktrees under `ROLAND_HOME/worktrees/`
- Sets `repos`, `persona_used` attributes in Slate
- Writes CLAUDE.md, hooks, and skills into the task directory
- Replaces the current process with the agent (`exec`)

---

### roland work

```
roland work [task-id]
```

Resume work on a task and launch agent.

Resolves the task (from argument or current working directory), shows a resumption briefing with the latest checkpoint and git status across worktrees, then launches the configured AI coding agent.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `task-id` | No | Slate task ID. If omitted, detected from cwd. |

**Examples:**

```bash
roland work st-a1b2
roland work                    # Detects task from cwd
cd ~/.roland/tasks/st-a1b2-fix-auth && roland work
```

**Side effects:**
- Replaces the current process with the agent (`exec`)

---

### roland checkpoint

```
roland checkpoint [task-id] [flags]
```

Record a progress checkpoint.

Adds a structured checkpoint to the task in Slate. The `--done` flag is required and describes what was accomplished. Other fields are optional but recommended.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--done` | **(required)** | What was accomplished |
| `--decisions` | | Key decisions made |
| `--next` | | What should happen next |
| `--blockers` | | Current blockers |
| `--files` | | Comma-separated file paths touched |

**Examples:**

```bash
roland checkpoint --done "Implemented auth middleware" --next "Add tests"
roland checkpoint st-a1b2 --done "Fixed login bug" --blockers "Need API key"
roland checkpoint --done "Refactored DB layer" --files "db.go,db_test.go"
```

**Side effects:**
- Writes a checkpoint record to Slate

---

### roland ship

```
roland ship [task-id] [flags]
```

Push branches and create pull requests.

Discovers worktrees in the task workspace, pushes branches to remote, and creates pull requests via `gh pr create`.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | | Ship only this repo |
| `--draft` | `false` | Create PRs as draft |
| `--dry-run` | `false` | Show what would happen without making changes |
| `--title` | auto-generated | PR title |
| `--body` | auto-generated | PR body |
| `--require-review` | `false` | Block shipping unless `review_status` is `approved` |

**Examples:**

```bash
roland ship
roland ship --dry-run
roland ship --repo backend --draft
roland ship --title "Fix auth bug" --body "Closes #123"
roland ship --require-review
```

**Side effects:**
- Pushes branches to remote (`git push -u origin <branch>`)
- Creates pull requests via GitHub CLI

---

### roland done

```
roland done [task-id] [flags]
```

Complete a task: check PRs, clean worktrees, close task.

Verifies all PRs are merged, removes worktrees and the task workspace, then closes the task in Slate.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--reason` | `done` | Closure reason |
| `--dry-run` | `false` | Show what would happen without making changes |
| `--force` | `false` | Skip PR merge checks |

**Examples:**

```bash
roland done st-a1b2
roland done --force
roland done --dry-run
roland done st-a1b2 --reason "Duplicate of st-c3d4"
```

**Side effects:**
- Removes git worktrees (`git worktree remove`)
- Deletes the task workspace directory
- Closes the task in Slate

---

## Configuration Commands

### roland config agent

```
roland config agent [name]
```

Get or set the default AI coding agent.

Without arguments, prints the current agent. With an argument, sets it. Valid agents: `claude`, `opencode`.

**Examples:**

```bash
roland config agent            # Print current agent
roland config agent opencode   # Switch to OpenCode
```

**Side effects (when setting):**
- Writes to `roland.yaml`

---

### roland config ide

```
roland config ide [name]
```

Get or set the preferred IDE.

Without arguments, prints the current IDE. With an argument, sets it. Valid IDEs: `vscode`, `cursor`, `windsurf`, `nvim`.

**Examples:**

```bash
roland config ide
roland config ide cursor
```

**Side effects (when setting):**
- Writes to `roland.yaml`

---

### roland config hooks list

```
roland config hooks list
```

List all hooks and their enabled/disabled status.

---

### roland config hooks enable

```
roland config hooks enable <hook-name>
```

Enable a hook.

**Side effects:**
- Writes to `roland.yaml`

---

### roland config hooks disable

```
roland config hooks disable <hook-name>
```

Disable a hook.

**Side effects:**
- Writes to `roland.yaml`

---

### roland config hooks sync

```
roland config hooks sync
```

Sync hook installations to match config. Installs missing hooks, removes extra ones.

**Side effects:**
- Writes/removes hook script files
- Updates `.claude/settings.json`

---

### roland config reset-claude-md

```
roland config reset-claude-md
```

Reset CLAUDE.md in ROLAND_HOME to the default template.

**Side effects:**
- Overwrites `ROLAND_HOME/CLAUDE.md`

---

## Repository Commands

### roland repo add

```
roland repo add <url> [flags]
```

Clone and register a repository.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | derived from URL | Short name for the repo |

**Examples:**

```bash
roland repo add https://github.com/org/backend.git
roland repo add https://github.com/org/backend.git --name api
roland repo add git@github.com:org/frontend.git
```

**Side effects:**
- Clones the repo to `ROLAND_HOME/repos/<name>/`
- Writes to `roland.yaml`

---

### roland repo list

```
roland repo list
```

List registered repositories with their URLs and base branches.

---

### roland repo remove

```
roland repo remove <name> [flags]
```

Unregister a repository.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--delete-files` | `false` | Also delete the repo directory on disk |

**Examples:**

```bash
roland repo remove backend
roland repo remove backend --delete-files
```

**Side effects:**
- Removes the repo from `roland.yaml`
- Optionally deletes the repo directory

---

### roland repo sync

```
roland repo sync [name]
```

Fetch and fast-forward repos.

Without arguments, syncs all repos. With a name, syncs only that repo. Performs `git fetch --prune` followed by `git merge --ff-only` on the current branch.

**Examples:**

```bash
roland repo sync            # Sync all repos
roland repo sync backend    # Sync only backend
```

**Side effects:**
- Runs git fetch and merge on repo directories

---

### roland repo post-setup

```
roland repo post-setup <repo-name> <script-path>
```

Set the post-setup script for a repo. This script runs after every worktree is created for this repo (e.g., `npm install`, `poetry install`).

**Examples:**

```bash
roland repo post-setup frontend ./scripts/post-setup.sh
```

**Side effects:**
- Writes to `roland.yaml`

---

## Worktree Commands

### roland worktree add

```
roland worktree add <repo-name> <branch> [flags]
```

Add a git worktree for a repo.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--base` | repo's `base_branch` or `origin/main` | Base branch to create from |
| `--task` | | Task ID to symlink worktree into |

**Examples:**

```bash
roland worktree add backend fix-auth
roland worktree add backend fix-auth --base origin/develop
roland worktree add backend fix-auth --task st-a1b2
```

**Side effects:**
- Runs `git worktree add` on the repo
- Creates directory under `ROLAND_HOME/worktrees/`
- Writes `.roland-base` file in the worktree
- Creates symlink in the task directory (if `--task` provided)
- Runs post-setup script (if configured)

---

### roland worktree rm

```
roland worktree rm <repo-name> <branch>
```

Remove a git worktree. Aliases: `remove`.

**Examples:**

```bash
roland worktree rm backend fix-auth
roland worktree remove backend fix-auth
```

**Side effects:**
- Runs `git worktree remove` (with fallback to manual deletion)

---

### roland worktree list

```
roland worktree list <repo-name>
```

List worktree branches for a repo.

**Examples:**

```bash
roland worktree list backend
```

---

## Persona Commands

### roland persona list

```
roland persona list
```

List all available personas with their source (builtin or custom).

---

### roland persona show

```
roland persona show <name>
```

Display a persona's full content.

**Examples:**

```bash
roland persona show builder
```

---

### roland persona create

```
roland persona create <name> [flags]
```

Create a new custom persona.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | | Base persona to copy from |

**Examples:**

```bash
roland persona create devops
roland persona create devops --from builder
```

**Side effects:**
- Creates `ROLAND_HOME/personas/<name>.md`

---

### roland persona edit

```
roland persona edit <name>
```

Open a persona for editing in `$EDITOR`. For built-in personas, creates a custom copy first.

**Side effects:**
- May create `ROLAND_HOME/personas/<name>.md` (copy of built-in)
- Opens `$EDITOR` (default: `vi`)

---

### roland persona delete

```
roland persona delete <name>
```

Delete a custom persona. Built-in personas cannot be deleted unless a custom override exists (in which case the override is removed, reverting to built-in).

**Side effects:**
- Removes `ROLAND_HOME/personas/<name>.md`

---

## Skill Commands

### roland skill add

```
roland skill add <path> --name <name> [flags]
```

Register a skill from a directory. The directory must contain a `SKILL.md` file.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | **(required)** | Skill name |
| `--external` | `false` | Keep skill at its current location instead of copying |

**Examples:**

```bash
roland skill add ./my-skill --name testing
roland skill add /path/to/skill --name testing --external
```

**Side effects:**
- Copies skill directory into `ROLAND_HOME/.skills/` (unless `--external`)
- Writes to `ROLAND_HOME/.skills/skills.json`

---

### roland skill list

```
roland skill list
```

List registered skills with their matching criteria.

---

### roland skill tag

```
roland skill tag <skill-name> [flags]
```

Set matching criteria for a skill. These determine when the skill is auto-injected during pickup.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--personas` | | Comma-separated persona names |
| `--types` | | Comma-separated task types |
| `--tags` | | Comma-separated tags/labels |

**Examples:**

```bash
roland skill tag my-skill --personas builder,researcher
roland skill tag my-skill --types feature,bug
roland skill tag my-skill --tags backend,api
```

**Side effects:**
- Writes to `ROLAND_HOME/.skills/skills.json`

---

### roland skill inject

```
roland skill inject <skill-name> [task-id]
```

Manually inject a skill into a task workspace.

**Examples:**

```bash
roland skill inject testing st-a1b2
roland skill inject testing           # Uses cwd to detect task
```

**Side effects:**
- Creates a symlink in the task's `.claude/skills/` directory

---

### roland skill eject

```
roland skill eject <skill-name> [task-id]
```

Remove a skill from a task workspace.

**Examples:**

```bash
roland skill eject testing st-a1b2
```

**Side effects:**
- Removes the symlink from the task's `.claude/skills/` directory

---

## Hook Commands

### roland hook list

```
roland hook list
```

List all hooks with their enabled/disabled status and installation state.

---

### roland hook add

```
roland hook add <hook-name>
```

Enable and install a hook.

**Side effects:**
- Sets the hook to enabled in `roland.yaml`
- Installs the hook script and wires it into agent settings

---

### roland hook remove

```
roland hook remove <hook-name>
```

Disable and uninstall a hook.

**Side effects:**
- Sets the hook to disabled in `roland.yaml`
- Removes the hook script and settings entry

---

### roland hook sync

```
roland hook sync
```

Sync hook installations to match config. Installs missing enabled hooks, removes disabled ones.

**Side effects:**
- Writes/removes hook script files
- Updates agent settings files

---

## Utility Commands

### roland status

```
roland status [flags]
```

Show status of all task workspaces.

Lists all active task workspaces with their status, worktree state (dirty/clean), persona, and latest checkpoint.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | `false` | Include completed/cancelled tasks |
| `--json` | `false` | Output as JSON |

**Examples:**

```bash
roland status
roland status --all
roland status --json
```

---

### roland clean

```
roland clean [flags]
```

Prune orphaned task workspaces.

Finds task workspaces whose tasks are closed/cancelled in Slate (or not found), and removes them along with their worktrees.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show what would be cleaned without making changes |
| `--force` | `false` | Remove workspaces even without Slate access |

**Examples:**

```bash
roland clean --dry-run
roland clean
roland clean --force
```

**Side effects:**
- Removes task workspace directories
- Removes associated worktrees

---

### roland open

```
roland open [task-id]
```

Open a task workspace in the configured IDE.

**Examples:**

```bash
roland open st-a1b2
roland open                    # Detects task from cwd
```

**Side effects:**
- Launches the IDE process (non-blocking)

---

### roland init

```
roland init [flags]
```

Initialize a Roland home directory.

Creates the directory structure, writes default config, installs hooks, writes the home pointer file, and generates CLAUDE.md and settings.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--here` | `false` | Use current directory as ROLAND_HOME |
| `--update` | `false` | Update existing installation (overwrite config, reinstall hooks) |

**Examples:**

```bash
roland init
roland init --here
roland init --update
```

**Side effects:**
- Creates `ROLAND_HOME/` directory structure
- Writes `roland.yaml`, `CLAUDE.md`, `.claude/settings.json`
- Installs hooks
- Writes `~/.config/roland/home` pointer file

---

### roland version

```
roland version
```

Print the Roland version.

---

### roland completion

```
roland completion [bash|zsh|fish]
```

Generate shell completion scripts.

**Examples:**

```bash
source <(roland completion bash)
roland completion zsh > "${fpath[1]}/_roland"
roland completion fish | source
```

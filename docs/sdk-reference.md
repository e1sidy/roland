# SDK Reference

Go API documentation for all public functions and types in Roland.

---

## Root Package (`github.com/e1sidy/roland`)

The root package provides configuration, repository management, agent/IDE enums, and Slate attribute management.

### Types

#### AgentTool

```go
type AgentTool string

const (
    AgentClaude   AgentTool = "claude"
    AgentOpenCode AgentTool = "opencode"
)
```

Identifies a supported AI coding agent.

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `IsValid` | `(a AgentTool) IsValid() bool` | Returns true if the agent is recognized |
| `Command` | `(a AgentTool) Command() string` | Returns the CLI command name (e.g., `"claude"`, `"opencode"`) |

#### IDE

```go
type IDE string

const (
    IDEVSCode  IDE = "vscode"
    IDECursor  IDE = "cursor"
    IDEWindsurf IDE = "windsurf"
    IDENvim    IDE = "nvim"
)
```

Identifies a supported code editor / IDE.

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `IsValid` | `(i IDE) IsValid() bool` | Returns true if the IDE is recognized |
| `Command` | `(i IDE) Command() string` | Returns the CLI command (e.g., `"code"`, `"cursor"`, `"nvim"`) |

#### Config

```go
type Config struct {
    Agent      AgentTool                  `yaml:"agent"`
    AgentFlags map[AgentTool][]string     `yaml:"agent_flags"`
    IDE        IDE                        `yaml:"ide"`
    SlateHome  string                     `yaml:"slate_home"`
    Repos      map[string]*RepoConfig     `yaml:"repos"`
    Hooks      map[string]bool            `yaml:"hooks"`
    Home       string                     `yaml:"-"`
}
```

Holds Roland's configuration, loaded from `roland.yaml`.

#### RepoConfig

```go
type RepoConfig struct {
    URL        string `yaml:"url"`
    BaseBranch string `yaml:"base_branch"`
    BranchName string `yaml:"branch_name"`
    PostSetup  string `yaml:"post_setup"`
}
```

Holds per-repository configuration.

#### Repo

```go
type Repo struct {
    Name string
    URL  string
    Path string
}
```

Represents a registered codebase.

#### SyncResult

```go
type SyncResult struct {
    Name    string
    Updated bool
    Error   error
}
```

Holds the result of syncing a repo.

### Functions

#### ValidAgentNames

```go
func ValidAgentNames() []string
```

Returns all valid agent tool names (`["claude", "opencode"]`).

#### ValidIDENames

```go
func ValidIDENames() []string
```

Returns all valid IDE names (`["vscode", "cursor", "windsurf", "nvim"]`).

#### UserAgent

```go
func UserAgent() string
```

Returns the user agent string (e.g., `"roland/dev"`).

#### DefaultConfig

```go
func DefaultConfig() *Config
```

Returns a Config with sensible defaults: agent=claude, IDE=cursor, empty repos and hooks.

#### ResolveHome

```go
func ResolveHome() (string, error)
```

Determines the ROLAND_HOME directory using the resolution cascade: `ROLAND_HOME` env var, `~/.config/roland/home` pointer file, `~/.roland/`.

#### WriteHomePointer

```go
func WriteHomePointer(rolandHome string) error
```

Writes the given path to `~/.config/roland/home`.

#### ConfigPath

```go
func ConfigPath(home string) string
```

Returns the path to `roland.yaml` within the given home directory.

#### LoadConfig

```go
func LoadConfig(home string) (*Config, error)
```

Reads `roland.yaml` from the given home directory. If the file does not exist, a default config is returned. Invalid agents are reset to `claude`.

#### SaveConfig

```go
func SaveConfig(cfg *Config) error
```

Writes the config to `roland.yaml`. Creates the home directory if needed.

#### ReposDir

```go
func ReposDir(home string) string
```

Returns the path where repos are cloned (`<home>/repos`).

#### TasksDir

```go
func TasksDir(home string) string
```

Returns the path where task workspaces live (`<home>/tasks`).

#### WorktreesDir

```go
func WorktreesDir(home string) string
```

Returns the path where git worktrees live (`<home>/worktrees`).

#### AddRepo

```go
func AddRepo(cfg *Config, url, name string) (*Repo, error)
```

Clones a repo and registers it in the config. If name is empty, it is derived from the URL. Writes updated config.

#### RemoveRepo

```go
func RemoveRepo(cfg *Config, name string, deleteFiles bool) error
```

Unregisters a repo and optionally deletes its files. Writes updated config.

#### ListRepos

```go
func ListRepos(cfg *Config) []*Repo
```

Returns all registered repos, sorted by name.

#### SyncRepo

```go
func SyncRepo(cfg *Config, name string) (*SyncResult, error)
```

Fetches and fast-forwards a registered repo. Performs `git fetch --prune` followed by `git merge --ff-only` on the current branch.

#### SyncAllRepos

```go
func SyncAllRepos(cfg *Config) []*SyncResult
```

Syncs every registered repo.

#### SetPostSetup

```go
func SetPostSetup(cfg *Config, repoName, scriptPath string) error
```

Sets the post-setup script for a repo. Writes updated config.

#### EnsureAttrs

```go
func EnsureAttrs(ctx context.Context, store *slate.Store) error
```

Defines all Roland-specific custom attributes in Slate. Idempotent -- safe to call on every pickup or init. Defines: `repos`, `persona_used`, `review_status`, `session_count`.

### Constants

```go
const (
    AttrRepos         = "repos"
    AttrPersonaUsed   = "persona_used"
    AttrReviewStatus  = "review_status"
    AttrSessionCount  = "session_count"
)
```

Custom attribute keys used by Roland in Slate.

### Variables

```go
var Version = "dev"
```

Set at build time via `-ldflags "-X github.com/e1sidy/roland.Version=..."`.

---

## Package `workspace`

Manages ephemeral task directories and git worktrees.

### Types

#### TaskDir

```go
type TaskDir struct {
    Path   string  // Absolute filesystem path
    TaskID string  // Slate task ID (e.g., "st-a1b2")
    Slug   string  // Directory name (e.g., "st-a1b2-fix-auth-bug")
}
```

Represents an active task workspace.

#### WorktreeOpts

```go
type WorktreeOpts struct {
    Home       string  // ROLAND_HOME path
    RepoName   string  // Short name of the registered repo
    Branch     string  // Branch to create
    BaseBranch string  // Base branch (e.g., "origin/main")
    TaskDir    string  // Task workspace directory to symlink into
    PostSetup  string  // Optional script to run after creation
}
```

Configures a new worktree.

### Functions

#### Create

```go
func Create(home, taskID, title string) (*TaskDir, error)
```

Creates a new task workspace directory. The directory name is derived from the task ID and title as a slug.

#### Open

```go
func Open(home, taskID string) (*TaskDir, error)
```

Finds an existing task workspace by task ID. The task ID is matched against directory names via prefix extraction.

#### Remove

```go
func Remove(home, taskID string) error
```

Deletes a task workspace directory.

#### List

```go
func List(home string) ([]*TaskDir, error)
```

Returns all task workspaces, sorted by slug.

#### ExtractTaskID

```go
func ExtractTaskID(slugOrPath string) string
```

Extracts the Slate task ID from a slug or path. Returns empty string if no task ID is found. Matches the pattern `st-[a-z0-9]+(\.[0-9]+)*`.

#### WorktreeAdd

```go
func WorktreeAdd(opts WorktreeOpts) (string, error)
```

Creates a git worktree for a repo and symlinks it into the task directory. Steps: fetch remote, create worktree with new branch, write `.roland-base` file, run post-setup script, create symlink. Returns the worktree directory path. Idempotent: if the worktree already exists, it is symlinked without recreation.

#### WorktreeRemove

```go
func WorktreeRemove(home, repoName, branch string) error
```

Removes a git worktree. Uses `git worktree remove --force` with fallback to manual deletion.

#### WorktreeList

```go
func WorktreeList(home, repoName string) ([]string, error)
```

Lists all worktree branch directories for a repo.

#### ReadBaseBranch

```go
func ReadBaseBranch(wtDir string) string
```

Reads the `.roland-base` file from a worktree directory. Returns `"origin/main"` if the file does not exist.

#### ResolveBranchName

```go
func ResolveBranchName(scriptPath string, taskJSON []byte, defaultBranch string) (string, error)
```

Determines the branch name for a task's worktree. If scriptPath is non-empty, the script is executed with taskJSON on stdin and the branch name is read from stdout. Otherwise, defaultBranch is returned.

---

## Package `persona`

Manages agent behavior templates.

### Types

#### PersonaInfo

```go
type PersonaInfo struct {
    Name   string  // e.g., "builder"
    Source string  // "builtin" or "custom"
    Path   string  // Filesystem path (empty for built-in)
}
```

Describes a persona and its source.

### Functions

#### Names

```go
func Names() []string
```

Returns the names of all built-in personas.

#### Get

```go
func Get(home, name string) (string, error)
```

Returns the content of a persona by name. Custom personas (in `ROLAND_HOME/personas/`) take precedence over built-in.

#### IsValid

```go
func IsValid(home, name string) bool
```

Returns true if the persona exists (built-in or custom).

#### Create

```go
func Create(home, name, fromBase string) error
```

Creates a new custom persona. If fromBase is non-empty, copies from the named persona. Otherwise creates an empty template.

#### Edit

```go
func Edit(home, name string) (string, error)
```

Returns the filesystem path to a persona file for editing. For built-in personas, creates a custom copy first.

#### Delete

```go
func Delete(home, name string) error
```

Deletes a custom persona file. Built-in personas cannot be deleted unless a custom override exists.

#### ListAll

```go
func ListAll(home string) ([]PersonaInfo, error)
```

Returns all personas (built-in + custom), sorted by name. Custom personas that override built-in ones are listed as "custom".

---

## Package `hooks`

Manages context injection into AI coding agents.

### Types

#### Source

```go
type Source string

const (
    SourceHome Source = "home"  // Installed to ROLAND_HOME
    SourceTask Source = "task"  // Installed to task directory
)
```

Indicates where a hook should be installed.

#### Hook

```go
type Hook struct {
    Name            string
    Source          Source
    Event           string
    Matcher         string
    Timeout         int
    ClaudeScript    func(HookContext) string
    OpenCodeSnippet func(HookContext) string
}
```

Defines a context injection hook.

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `TimeoutOrDefault` | `(h *Hook) TimeoutOrDefault() int` | Returns the timeout, or 10 if not set |

#### HookContext

```go
type HookContext struct {
    RolandHome string  // Path to ROLAND_HOME
    TargetDir  string  // Path where hooks are being installed
    SlateHome  string  // Path to SLATE_HOME
}
```

Provides data to hook content generators.

#### Registry

```go
type Registry struct { /* unexported fields */ }
```

Holds a collection of hooks.

#### Manager

```go
type Manager struct { /* unexported fields */ }
```

Orchestrates hook installation, uninstallation, and syncing.

### Functions

#### NewRegistry

```go
func NewRegistry() *Registry
```

Creates an empty hook registry.

#### DefaultRegistry

```go
func DefaultRegistry() *Registry
```

Returns a registry pre-loaded with the 5 built-in hooks.

#### Registry Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Register` | `(r *Registry) Register(h *Hook)` | Adds a hook. Panics on duplicate names. |
| `Get` | `(r *Registry) Get(name string) *Hook` | Returns a hook by name, or nil. |
| `All` | `(r *Registry) All() []*Hook` | Returns all hooks, sorted by name. |
| `BySource` | `(r *Registry) BySource(s Source) []*Hook` | Returns hooks with the given source. |
| `Names` | `(r *Registry) Names() []string` | Returns all hook names, sorted. |

#### NewManager

```go
func NewManager(r *Registry) *Manager
```

Creates a manager backed by the given registry.

#### Manager Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Registry` | `(m *Manager) Registry() *Registry` | Returns the underlying registry. |
| `Install` | `(m *Manager) Install(name, targetDir string, agent AgentTool, ctx HookContext) error` | Installs a single hook. |
| `Uninstall` | `(m *Manager) Uninstall(name, targetDir string, agent AgentTool) error` | Removes a single hook. |
| `Sync` | `(m *Manager) Sync(targetDir string, enabled map[string]bool, agent AgentTool, ctx HookContext) error` | Ensures exactly the enabled hooks are installed. Idempotent. |
| `Installed` | `(m *Manager) Installed(targetDir string) []string` | Returns names of installed hooks. |

#### EnabledForSource

```go
func EnabledForSource(cfg map[string]bool, source Source, reg *Registry) map[string]bool
```

Filters a hook enable map to only include hooks matching the given source type. Hooks not present in the config default to enabled.

#### MaterializeScripts

```go
func MaterializeScripts(targetDir string) error
```

Writes embedded hook scripts to the target directory's `hooks/` subdirectory.

---

## Package `skill`

Manages reusable context directories for auto-injection.

### Types

#### SkillEntry

```go
type SkillEntry struct {
    Location  string   `json:"location"`
    Personas  []string `json:"personas,omitempty"`
    TaskTypes []string `json:"task_types,omitempty"`
    Tags      []string `json:"tags,omitempty"`
    Version   string   `json:"version,omitempty"`
}
```

Holds metadata about a registered skill.

#### SkillConfig

```go
type SkillConfig struct {
    Schema string                  `json:"$schema,omitempty"`
    Skills map[string]*SkillEntry  `json:"skills"`
}
```

Holds the skill registry (serialized to `skills.json`).

#### SkillInfo

```go
type SkillInfo struct {
    Name  string
    Entry *SkillEntry
}
```

Combines a name with its entry for listing.

#### MatchContext

```go
type MatchContext struct {
    Persona  string
    TaskType string
    Labels   []string
}
```

Provides the current task context for skill matching.

### Functions

#### SkillsPath

```go
func SkillsPath(home string) string
```

Returns the path to `skills.json` (`<home>/.skills/skills.json`).

#### DefaultSkillsDir

```go
func DefaultSkillsDir(home string) string
```

Returns the default directory for skill storage (`<home>/.skills`).

#### LoadSkills

```go
func LoadSkills(home string) (*SkillConfig, error)
```

Reads `skills.json`. Returns an empty config if the file does not exist.

#### SaveSkills

```go
func SaveSkills(home string, sc *SkillConfig) error
```

Writes `skills.json`.

#### Add

```go
func Add(home, skillPath, name string, external bool) (*SkillEntry, error)
```

Registers a new skill. If external is false, the skill directory is copied into the skills directory. The directory must contain a `SKILL.md` file.

#### Remove

```go
func Remove(home, name string, deleteFiles bool) error
```

Unregisters a skill and optionally deletes its files.

#### SetTags

```go
func SetTags(home, name string, personas, taskTypes, tags []string) error
```

Updates the matching criteria for a skill.

#### List

```go
func List(home string) ([]*SkillInfo, error)
```

Returns all registered skills, sorted by name.

#### Get

```go
func Get(home, name string) (*SkillEntry, error)
```

Returns a single skill entry by name.

#### Match

```go
func Match(entry *SkillEntry, ctx *MatchContext) bool
```

Returns true if the skill should be auto-injected for the given context. Uses OR logic across personas, task types, and tags. Returns false if all dimensions are empty.

#### MatchAll

```go
func MatchAll(sc *SkillConfig, ctx *MatchContext) []string
```

Returns the names of all skills that match the given context.

#### Inject

```go
func Inject(skillName, skillLocation, taskDir string) error
```

Creates a symlink from the task directory's `.claude/skills/` to the skill location. Idempotent.

#### Eject

```go
func Eject(skillName, taskDir string) error
```

Removes a skill symlink from the task directory.

#### Injected

```go
func Injected(taskDir string) ([]string, error)
```

Returns the names of skills currently injected into the task directory.

#### InjectMatching

```go
func InjectMatching(home, taskDir string, ctx *MatchContext) ([]string, error)
```

Loads the skill registry, finds matching skills, and injects them all. Returns the list of injected skill names.

---

## Package `embed`

Provides embedded assets for Roland.

### Variables

```go
var DefaultClaudeMD string
```

The default CLAUDE.md template content (embedded at build time).

```go
var DefaultClaudeSettings string
```

The default `claude-settings.json` content (embedded at build time).

### Functions

#### WriteClaudeMD

```go
func WriteClaudeMD(dir, homePath string, force bool) error
```

Writes the default CLAUDE.md to the given directory. Substitutes `{{.Home}}` with homePath. If force is false, an existing file is not overwritten.

#### WriteClaudeSettings

```go
func WriteClaudeSettings(dir string) error
```

Writes the default `claude-settings.json` to `.claude/settings.json` within the given directory. Does not overwrite existing files.

---

## Package `internal/gitutil`

Thin wrappers around git CLI commands.

### Functions

#### Run

```go
func Run(dir string, args ...string) error
```

Executes a git command in the given directory, inheriting stdout/stderr.

#### Output

```go
func Output(dir string, args ...string) ([]byte, error)
```

Executes a git command and captures stdout.

#### IsSymlink

```go
func IsSymlink(path string) bool
```

Returns true if the path is a symbolic link.

---

## Package `internal/testutil`

Test helpers for Roland.

### Functions

#### TempHome

```go
func TempHome(t *testing.T) string
```

Creates a temporary directory suitable for use as `ROLAND_HOME`. Automatically cleaned up when the test finishes.

#### TempSlateStore

```go
func TempSlateStore(t *testing.T) (*slate.Store, context.Context)
```

Opens an in-memory Slate store for testing. Automatically closed when the test finishes.

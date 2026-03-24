package roland

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds Roland's configuration, loaded from roland.yaml.
type Config struct {
	// Agent is the default AI coding agent to launch.
	Agent AgentTool `yaml:"agent"`

	// AgentFlags holds per-agent CLI flags. Key is agent name.
	AgentFlags map[AgentTool][]string `yaml:"agent_flags"`

	// IDE is the preferred code editor.
	IDE IDE `yaml:"ide"`

	// SlateHome overrides Slate's home directory. Empty = use Slate default.
	SlateHome string `yaml:"slate_home"`

	// Repos is the set of registered codebases, keyed by short name.
	Repos map[string]*RepoConfig `yaml:"repos"`

	// Hooks controls which hooks are enabled. Key is hook name.
	Hooks map[string]bool `yaml:"hooks"`

	// CustomHooks holds user-defined hooks. Key is hook name.
	CustomHooks map[string]*CustomHookDef `yaml:"custom_hooks"`

	// ContextBudget is the max token budget for context injection (default: 4096).
	ContextBudget int `yaml:"context_budget"`

	// AutoCheckpoint controls auto-checkpoint on PreCompact (default: true).
	AutoCheckpoint *bool `yaml:"auto_checkpoint"`

	// CleanupRemoteBranches deletes remote branches on `roland done` (default: false).
	CleanupRemoteBranches bool `yaml:"cleanup_remote_branches"`

	// ConfigVersion tracks the config schema version for migration (default: 1).
	ConfigVersion int `yaml:"config_version"`

	// Home is the resolved ROLAND_HOME path. Not persisted to YAML.
	Home string `yaml:"-"`
}

// IsAutoCheckpointEnabled returns whether auto-checkpoint is enabled (default: true).
func (c *Config) IsAutoCheckpointEnabled() bool {
	if c.AutoCheckpoint == nil {
		return true
	}
	return *c.AutoCheckpoint
}

// CustomHookDef defines a user-created hook stored in roland.yaml.
type CustomHookDef struct {
	// Event is the agent event trigger (e.g., "SessionStart", "PreToolUse").
	Event string `yaml:"event"`

	// Script is the path to the bash script to execute.
	Script string `yaml:"script"`

	// Matcher filters which tool triggers this hook ("*" = all).
	Matcher string `yaml:"matcher"`

	// Timeout in seconds. 0 = default (10s).
	Timeout int `yaml:"timeout"`
}

// RepoConfig holds per-repository configuration.
type RepoConfig struct {
	// URL is the git clone URL.
	URL string `yaml:"url"`

	// BaseBranch is the default branch to create worktrees from (e.g., "origin/main").
	BaseBranch string `yaml:"base_branch"`

	// BranchName is an optional script path that generates branch names.
	// The script receives task JSON on stdin and outputs the branch name.
	BranchName string `yaml:"branch_name"`

	// PostSetup is an optional script to run after worktree creation
	// (e.g., npm install, poetry install).
	PostSetup string `yaml:"post_setup"`

	// PRTemplate is the path to a PR body template file.
	// Template variables: {id}, {title}, {description}, {checkpoint}, {branch}, {priority}.
	PRTemplate string `yaml:"pr_template"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentClaude,
		AgentFlags: map[AgentTool][]string{
			AgentClaude:   {"--dangerously-skip-permissions"},
			AgentOpenCode: {},
			AgentCodex:    {},
			AgentGemini:   {},
		},
		IDE:         IDECursor,
		Repos:       make(map[string]*RepoConfig),
		Hooks:         make(map[string]bool),
		CustomHooks:   make(map[string]*CustomHookDef),
		ContextBudget: 4096,
		ConfigVersion: 1,
	}
}

// ResolveHome determines the ROLAND_HOME directory.
//
// Resolution order:
//  1. ROLAND_HOME environment variable
//  2. ~/.config/roland/home pointer file
//  3. ~/.roland/ (default)
func ResolveHome() (string, error) {
	if env := os.Getenv("ROLAND_HOME"); env != "" {
		return env, nil
	}

	ptr, err := globalPointerPath()
	if err == nil {
		data, err := os.ReadFile(ptr)
		if err == nil {
			p := strings.TrimSpace(string(data))
			if p != "" {
				return p, nil
			}
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".roland"), nil
}

// WriteHomePointer writes the given path to ~/.config/roland/home.
// This allows Roland to find its home directory even if ROLAND_HOME
// is not set — the pointer file survives cache clears and shell changes.
func WriteHomePointer(rolandHome string) error {
	ptr, err := globalPointerPath()
	if err != nil {
		return fmt.Errorf("pointer path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(ptr), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return os.WriteFile(ptr, []byte(rolandHome+"\n"), 0o644)
}

// globalPointerPath returns ~/.config/roland/home.
func globalPointerPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "roland", "home"), nil
}

// ConfigPath returns the path to roland.yaml within the given home.
func ConfigPath(home string) string {
	return filepath.Join(home, "roland.yaml")
}

// LoadConfig reads roland.yaml from the given home directory.
// If the file does not exist, a default config is returned.
func LoadConfig(home string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.Home = home

	data, err := os.ReadFile(ConfigPath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.Home = home

	// Ensure AgentFlags map exists with defaults for missing agents.
	if cfg.AgentFlags == nil {
		cfg.AgentFlags = map[AgentTool][]string{
			AgentClaude:   {"--dangerously-skip-permissions"},
			AgentOpenCode: {},
		}
	}
	if _, ok := cfg.AgentFlags[AgentClaude]; !ok {
		cfg.AgentFlags[AgentClaude] = []string{"--dangerously-skip-permissions"}
	}

	// Validate agent.
	if !cfg.Agent.IsValid() {
		cfg.Agent = AgentClaude
	}

	// Ensure maps are non-nil.
	if cfg.Repos == nil {
		cfg.Repos = make(map[string]*RepoConfig)
	}
	if cfg.Hooks == nil {
		cfg.Hooks = make(map[string]bool)
	}
	if cfg.CustomHooks == nil {
		cfg.CustomHooks = make(map[string]*CustomHookDef)
	}

	// Config migration: upgrade old configs to current version.
	if cfg.ConfigVersion < 1 {
		migrateConfig(cfg)
	}

	return cfg, nil
}

// migrateConfig applies migration steps to bring old configs up to date.
func migrateConfig(cfg *Config) {
	// v0 → v1: add new agent flags, context_budget default.
	if _, ok := cfg.AgentFlags[AgentCodex]; !ok {
		cfg.AgentFlags[AgentCodex] = []string{}
	}
	if _, ok := cfg.AgentFlags[AgentGemini]; !ok {
		cfg.AgentFlags[AgentGemini] = []string{}
	}
	if cfg.ContextBudget <= 0 {
		cfg.ContextBudget = 4096
	}
	cfg.ConfigVersion = 1

	// Auto-save migrated config.
	SaveConfig(cfg)
}

// SaveConfig writes the config to roland.yaml.
func SaveConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(cfg.Home, 0o755); err != nil {
		return fmt.Errorf("create home dir: %w", err)
	}
	return os.WriteFile(ConfigPath(cfg.Home), data, 0o644)
}

// ReposDir returns the path where repos are cloned.
func ReposDir(home string) string {
	return filepath.Join(home, "repos")
}

// TasksDir returns the path where task workspaces live.
func TasksDir(home string) string {
	return filepath.Join(home, "tasks")
}

// WorktreesDir returns the path where git worktrees live.
func WorktreesDir(home string) string {
	return filepath.Join(home, "worktrees")
}

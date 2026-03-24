package roland

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Agent != AgentClaude {
		t.Errorf("default agent = %q, want %q", cfg.Agent, AgentClaude)
	}
	if cfg.IDE != IDECursor {
		t.Errorf("default IDE = %q, want %q", cfg.IDE, IDECursor)
	}
	if cfg.Repos == nil {
		t.Error("repos map is nil")
	}
	if cfg.Hooks == nil {
		t.Error("hooks map is nil")
	}
	// Default Claude flags should include --dangerously-skip-permissions.
	flags := cfg.AgentFlags[AgentClaude]
	if len(flags) != 1 || flags[0] != "--dangerously-skip-permissions" {
		t.Errorf("default claude flags = %v, want [--dangerously-skip-permissions]", flags)
	}
}

func TestLoadConfig_Missing(t *testing.T) {
	home := t.TempDir()
	cfg, err := LoadConfig(home)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Agent != AgentClaude {
		t.Errorf("agent = %q, want %q", cfg.Agent, AgentClaude)
	}
	if cfg.Home != home {
		t.Errorf("home = %q, want %q", cfg.Home, home)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	home := t.TempDir()
	cfg := DefaultConfig()
	cfg.Home = home
	cfg.Agent = AgentOpenCode
	cfg.IDE = IDENvim
	cfg.Repos["backend"] = &RepoConfig{URL: "https://github.com/org/backend.git", BaseBranch: "origin/main"}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig(home)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Agent != AgentOpenCode {
		t.Errorf("agent = %q, want %q", loaded.Agent, AgentOpenCode)
	}
	if loaded.IDE != IDENvim {
		t.Errorf("IDE = %q, want %q", loaded.IDE, IDENvim)
	}
	rc, ok := loaded.Repos["backend"]
	if !ok {
		t.Fatal("backend repo not in loaded config")
	}
	if rc.URL != "https://github.com/org/backend.git" {
		t.Errorf("backend URL = %q", rc.URL)
	}
}

func TestSaveAndLoadConfig_CustomHooks(t *testing.T) {
	home := t.TempDir()
	cfg := DefaultConfig()
	cfg.Home = home
	cfg.CustomHooks["my-hook"] = &CustomHookDef{
		Event:   "SessionStart",
		Script:  "/path/to/my-hook.sh",
		Matcher: "Bash",
		Timeout: 5,
	}
	cfg.Hooks["my-hook"] = true

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig(home)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	ch, ok := loaded.CustomHooks["my-hook"]
	if !ok {
		t.Fatal("custom hook not in loaded config")
	}
	if ch.Event != "SessionStart" {
		t.Errorf("event = %q, want SessionStart", ch.Event)
	}
	if ch.Script != "/path/to/my-hook.sh" {
		t.Errorf("script = %q", ch.Script)
	}
	if ch.Matcher != "Bash" {
		t.Errorf("matcher = %q", ch.Matcher)
	}
	if ch.Timeout != 5 {
		t.Errorf("timeout = %d, want 5", ch.Timeout)
	}
	if !loaded.Hooks["my-hook"] {
		t.Error("custom hook should be enabled")
	}
}

func TestLoadConfig_InvalidAgent(t *testing.T) {
	home := t.TempDir()
	// Write config with invalid agent.
	content := []byte("agent: badagent\n")
	if err := os.WriteFile(filepath.Join(home, "roland.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(home)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	// Should fallback to claude.
	if cfg.Agent != AgentClaude {
		t.Errorf("agent = %q, want %q (fallback)", cfg.Agent, AgentClaude)
	}
}

func TestResolveHome_EnvVar(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("ROLAND_HOME", custom)

	home, err := ResolveHome()
	if err != nil {
		t.Fatalf("ResolveHome: %v", err)
	}
	if home != custom {
		t.Errorf("home = %q, want %q", home, custom)
	}
}

func TestWriteAndResolveHome_Pointer(t *testing.T) {
	// Clear env var so it doesn't interfere.
	t.Setenv("ROLAND_HOME", "")

	custom := t.TempDir()
	if err := WriteHomePointer(custom); err != nil {
		t.Fatalf("WriteHomePointer: %v", err)
	}

	home, err := ResolveHome()
	if err != nil {
		t.Fatalf("ResolveHome: %v", err)
	}
	if home != custom {
		t.Errorf("home = %q, want %q", home, custom)
	}
}

func TestAgentTool_Command(t *testing.T) {
	tt := []struct {
		agent AgentTool
		want  string
	}{
		{AgentClaude, "claude"},
		{AgentOpenCode, "opencode"},
		{AgentTool("unknown"), "unknown"},
	}
	for _, tc := range tt {
		if got := tc.agent.Command(); got != tc.want {
			t.Errorf("%q.Command() = %q, want %q", tc.agent, got, tc.want)
		}
	}
}

func TestAgentTool_IsValid(t *testing.T) {
	tt := []struct {
		agent AgentTool
		want  bool
	}{
		{AgentClaude, true},
		{AgentOpenCode, true},
		{AgentTool("unknown"), false},
		{AgentTool(""), false},
	}
	for _, tc := range tt {
		if got := tc.agent.IsValid(); got != tc.want {
			t.Errorf("%q.IsValid() = %v, want %v", tc.agent, got, tc.want)
		}
	}
}

func TestIDE_Command(t *testing.T) {
	tt := []struct {
		ide  IDE
		want string
	}{
		{IDEVSCode, "code"},
		{IDECursor, "cursor"},
		{IDEWindsurf, "windsurf"},
		{IDENvim, "nvim"},
		{IDE("unknown"), "unknown"},
	}
	for _, tc := range tt {
		if got := tc.ide.Command(); got != tc.want {
			t.Errorf("%q.Command() = %q, want %q", tc.ide, got, tc.want)
		}
	}
}

func TestIDE_IsValid(t *testing.T) {
	tt := []struct {
		ide  IDE
		want bool
	}{
		{IDEVSCode, true},
		{IDECursor, true},
		{IDEWindsurf, true},
		{IDENvim, true},
		{IDE("unknown"), false},
	}
	for _, tc := range tt {
		if got := tc.ide.IsValid(); got != tc.want {
			t.Errorf("%q.IsValid() = %v, want %v", tc.ide, got, tc.want)
		}
	}
}

func TestValidAgentNames(t *testing.T) {
	names := ValidAgentNames()
	if len(names) != 4 {
		t.Errorf("len(ValidAgentNames()) = %d, want 4", len(names))
	}
}

func TestValidIDENames(t *testing.T) {
	names := ValidIDENames()
	if len(names) != 4 {
		t.Errorf("len(ValidIDENames()) = %d, want 4", len(names))
	}
}

func TestConfigPath(t *testing.T) {
	got := ConfigPath("/home/user/.roland")
	want := "/home/user/.roland/roland.yaml"
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}

func TestDirPaths(t *testing.T) {
	home := "/home/user/.roland"
	if got := ReposDir(home); got != "/home/user/.roland/repos" {
		t.Errorf("ReposDir = %q", got)
	}
	if got := TasksDir(home); got != "/home/user/.roland/tasks" {
		t.Errorf("TasksDir = %q", got)
	}
	if got := WorktreesDir(home); got != "/home/user/.roland/worktrees" {
		t.Errorf("WorktreesDir = %q", got)
	}
}

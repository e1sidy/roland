package hooks

import (
	"testing"

	"github.com/e1sidy/roland"
)

func TestSync_InstallsEnabled(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Hook{
		Name:    "test-a",
		Source:  SourceHome,
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return "#!/bin/bash\necho a\n"
		},
	})
	reg.Register(&Hook{
		Name:    "test-b",
		Source:  SourceHome,
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return "#!/bin/bash\necho b\n"
		},
	})

	mgr := NewManager(reg)
	targetDir := t.TempDir()
	ctx := HookContext{RolandHome: "/tmp/roland"}
	enabled := map[string]bool{"test-a": true, "test-b": true}

	if err := mgr.Sync(targetDir, enabled, roland.AgentClaude, ctx); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	installed := mgr.Installed(targetDir)
	if len(installed) != 2 {
		t.Errorf("Installed = %d, want 2: %v", len(installed), installed)
	}
}

func TestSync_RemovesDisabled(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Hook{
		Name:    "test-a",
		Source:  SourceHome,
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return "#!/bin/bash\necho a\n"
		},
	})

	mgr := NewManager(reg)
	targetDir := t.TempDir()
	ctx := HookContext{RolandHome: "/tmp/roland"}

	// First: install.
	mgr.Sync(targetDir, map[string]bool{"test-a": true}, roland.AgentClaude, ctx)

	// Second: disable.
	mgr.Sync(targetDir, map[string]bool{"test-a": false}, roland.AgentClaude, ctx)

	installed := mgr.Installed(targetDir)
	if len(installed) != 0 {
		t.Errorf("Installed = %d after disable, want 0: %v", len(installed), installed)
	}
}

func TestInstall_NotFound(t *testing.T) {
	mgr := NewManager(NewRegistry())
	ctx := HookContext{}

	err := mgr.Install("nonexistent", t.TempDir(), roland.AgentClaude, ctx)
	if err == nil {
		t.Error("Install should fail for unknown hook")
	}
}

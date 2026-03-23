package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallClaude(t *testing.T) {
	targetDir := t.TempDir()
	h := &Hook{
		Name:    "test-hook",
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return "#!/bin/bash\necho hello\n"
		},
	}
	ctx := HookContext{RolandHome: "/tmp/roland", TargetDir: targetDir}

	if err := installClaude(h, targetDir, ctx); err != nil {
		t.Fatalf("installClaude: %v", err)
	}

	// Script file should exist.
	scriptPath := filepath.Join(targetDir, "hooks", "test-hook.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Errorf("script not created: %v", err)
	}

	// Settings should be updated.
	settings, err := readSettings(targetDir)
	if err != nil {
		t.Fatalf("readSettings: %v", err)
	}
	hooksList, _ := settings["hooks"].([]any)
	if len(hooksList) == 0 {
		t.Error("hooks list is empty in settings")
	}
}

func TestUninstallClaude(t *testing.T) {
	targetDir := t.TempDir()
	h := &Hook{
		Name:    "test-hook",
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return "#!/bin/bash\necho hello\n"
		},
	}
	ctx := HookContext{RolandHome: "/tmp/roland"}

	installClaude(h, targetDir, ctx)
	if err := uninstallClaude("test-hook", targetDir); err != nil {
		t.Fatalf("uninstallClaude: %v", err)
	}

	// Script should be gone.
	scriptPath := filepath.Join(targetDir, "hooks", "test-hook.sh")
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Error("script file still exists after uninstall")
	}
}

func TestReadSettings_Missing(t *testing.T) {
	targetDir := t.TempDir()
	settings, err := readSettings(targetDir)
	if err != nil {
		t.Fatalf("readSettings: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("expected empty settings, got %v", settings)
	}
}

func TestReadSettings_Invalid(t *testing.T) {
	targetDir := t.TempDir()
	claudeDir := filepath.Join(targetDir, ".claude")
	os.MkdirAll(claudeDir, 0o755)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("not json"), 0o644)

	settings, err := readSettings(targetDir)
	if err != nil {
		t.Fatalf("readSettings: %v", err)
	}
	// Should return empty map on parse error.
	if len(settings) != 0 {
		t.Errorf("expected empty settings on parse error, got %v", settings)
	}
}

func TestAddHookToSettings_Idempotent(t *testing.T) {
	settings := map[string]any{}

	addHookToSettings(settings, "SessionStart", "*", "/path/to/test-hook.sh", 10)
	addHookToSettings(settings, "SessionStart", "*", "/path/to/test-hook.sh", 10) // Second add.

	hooksList := settings["hooks"].([]any)
	if len(hooksList) != 1 {
		t.Errorf("expected 1 hook entry after idempotent add, got %d", len(hooksList))
	}
}

func TestRemoveHookFromSettings(t *testing.T) {
	settings := map[string]any{}
	addHookToSettings(settings, "SessionStart", "*", "/path/to/test-hook.sh", 10)
	addHookToSettings(settings, "SessionStart", "*", "/path/to/other-hook.sh", 10)

	removeHookFromSettings(settings, "test-hook")

	hooksList := settings["hooks"].([]any)
	if len(hooksList) != 1 {
		t.Errorf("expected 1 hook after remove, got %d", len(hooksList))
	}
	// Remaining should be other-hook.
	entry := hooksList[0].(map[string]any)
	cmd := entry["command"].(string)
	if cmd != "bash /path/to/other-hook.sh" {
		t.Errorf("remaining hook cmd = %q", cmd)
	}
}

func TestInstalledClaudeHooks(t *testing.T) {
	targetDir := t.TempDir()
	dir := filepath.Join(targetDir, "hooks")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "hook-a.sh"), []byte("#!/bin/bash"), 0o755)
	os.WriteFile(filepath.Join(dir, "hook-b.sh"), []byte("#!/bin/bash"), 0o755)
	os.WriteFile(filepath.Join(dir, "not-a-hook.txt"), []byte("text"), 0o644)

	names := installedClaudeHooks(targetDir)
	if len(names) != 2 {
		t.Errorf("installedClaudeHooks = %d, want 2: %v", len(names), names)
	}
}

func TestWriteAndReadSettings(t *testing.T) {
	targetDir := t.TempDir()
	settings := map[string]any{
		"key": "value",
	}

	if err := writeSettings(targetDir, settings); err != nil {
		t.Fatalf("writeSettings: %v", err)
	}

	loaded, err := readSettings(targetDir)
	if err != nil {
		t.Fatalf("readSettings: %v", err)
	}

	data, _ := json.Marshal(loaded)
	if v, _ := loaded["key"].(string); v != "value" {
		t.Errorf("loaded settings = %s", string(data))
	}
}

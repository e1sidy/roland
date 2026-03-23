package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hooksDir returns the path to the hooks scripts directory.
func hooksDir(targetDir string) string {
	return filepath.Join(targetDir, "hooks")
}

// settingsPath returns the path to .claude/settings.json.
func settingsPath(targetDir string) string {
	return filepath.Join(targetDir, ".claude", "settings.json")
}

// installClaude writes a bash script and wires it into settings.json.
func installClaude(h *Hook, targetDir string, ctx HookContext) error {
	// Write the script file.
	dir := hooksDir(targetDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}

	scriptPath := filepath.Join(dir, h.Name+".sh")
	content := h.ClaudeScript(ctx)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write script: %w", err)
	}

	// Wire into settings.json.
	settings, err := readSettings(targetDir)
	if err != nil {
		return err
	}

	addHookToSettings(settings, h.Event, h.Matcher, scriptPath, h.TimeoutOrDefault())

	return writeSettings(targetDir, settings)
}

// uninstallClaude removes a hook's script and settings entry.
func uninstallClaude(name, targetDir string) error {
	// Remove script file.
	scriptPath := filepath.Join(hooksDir(targetDir), name+".sh")
	os.Remove(scriptPath)

	// Remove from settings.json.
	settings, err := readSettings(targetDir)
	if err != nil {
		return err
	}

	removeHookFromSettings(settings, name)

	return writeSettings(targetDir, settings)
}

// readSettings loads .claude/settings.json, or returns an empty map.
func readSettings(targetDir string) (map[string]any, error) {
	path := settingsPath(targetDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return map[string]any{}, nil // Start fresh on parse error.
	}
	return settings, nil
}

// writeSettings writes the settings map to .claude/settings.json.
func writeSettings(targetDir string, settings map[string]any) error {
	dir := filepath.Join(targetDir, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .claude dir: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(settingsPath(targetDir), data, 0o644)
}

// addHookToSettings adds a hook entry to the settings hooks array.
// The entry is identified by the script path. Idempotent: replaces existing.
func addHookToSettings(settings map[string]any, event, matcher, scriptPath string, timeout int) {
	hooksVal, _ := settings["hooks"]
	hooksList, ok := hooksVal.([]any)
	if !ok {
		hooksList = []any{}
	}

	entry := map[string]any{
		"type":    event,
		"command": "bash " + scriptPath,
	}
	if matcher != "*" && matcher != "" {
		entry["matcher"] = matcher
	}
	if timeout > 0 {
		entry["timeout"] = timeout
	}

	// Remove any existing entry for this script.
	var filtered []any
	for _, item := range hooksList {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		cmd, _ := m["command"].(string)
		if !strings.Contains(cmd, filepath.Base(scriptPath)) {
			filtered = append(filtered, item)
		}
	}
	filtered = append(filtered, entry)

	settings["hooks"] = filtered
}

// removeHookFromSettings removes all hook entries for a given hook name.
func removeHookFromSettings(settings map[string]any, hookName string) {
	hooksVal, _ := settings["hooks"]
	hooksList, ok := hooksVal.([]any)
	if !ok {
		return
	}

	var filtered []any
	scriptFile := hookName + ".sh"
	for _, item := range hooksList {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		cmd, _ := m["command"].(string)
		if !strings.Contains(cmd, scriptFile) {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) > 0 {
		settings["hooks"] = filtered
	} else {
		delete(settings, "hooks")
	}
}

// installedClaudeHooks returns names of hooks with script files present.
func installedClaudeHooks(targetDir string) []string {
	dir := hooksDir(targetDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
			names = append(names, strings.TrimSuffix(e.Name(), ".sh"))
		}
	}
	return names
}

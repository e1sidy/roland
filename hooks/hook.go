// Package hooks manages context injection into AI coding agents.
//
// The hook system follows a 4-layer architecture:
//  1. Hook definitions (types and content generators)
//  2. Registry (collection of hooks)
//  3. Manager (install/uninstall/sync orchestration)
//  4. Delivery (per-agent: Claude Code bash scripts, OpenCode JS plugin)
package hooks

import "github.com/e1sidy/roland"

// Source indicates where a hook should be installed.
type Source string

const (
	// SourceHome installs to ROLAND_HOME (applies to all tasks).
	SourceHome Source = "home"

	// SourceTask installs to a specific task directory.
	SourceTask Source = "task"
)

// Hook defines a context injection hook.
type Hook struct {
	// Name uniquely identifies this hook (e.g., "slate-instructions").
	Name string

	// Source indicates where this hook is installed (home or task).
	Source Source

	// Event is the agent event that triggers this hook
	// (e.g., "SessionStart", "PreCompact", "PostToolUse").
	Event string

	// Matcher filters which tool triggers this hook.
	// "*" matches all tools; "Bash" matches only Bash.
	Matcher string

	// Timeout in seconds before the hook is killed. 0 = use default (10s).
	Timeout int

	// ClaudeScript returns the bash script content for Claude Code delivery.
	// The function receives context so it can generate dynamic content.
	ClaudeScript func(HookContext) string

	// OpenCodeSnippet returns the JS snippet for OpenCode delivery.
	OpenCodeSnippet func(HookContext) string
}

// HookContext provides data to hook content generators.
type HookContext struct {
	RolandHome string // Path to ROLAND_HOME.
	TargetDir  string // Path where hooks are being installed.
	SlateHome  string // Path to SLATE_HOME (for Slate CLI).
}

// TimeoutOrDefault returns the hook timeout, or 10 if not set.
func (h *Hook) TimeoutOrDefault() int {
	if h.Timeout > 0 {
		return h.Timeout
	}
	return 10
}

// EnabledForSource filters a hook enable map to only include hooks
// matching the given source type from the registry.
func EnabledForSource(cfg map[string]bool, source Source, reg *Registry) map[string]bool {
	result := make(map[string]bool)
	for _, h := range reg.All() {
		if h.Source != source {
			continue
		}
		// If hook is in the config, use that value.
		if enabled, exists := cfg[h.Name]; exists {
			result[h.Name] = enabled
		} else {
			// Default: enabled (new hooks work out of the box).
			result[h.Name] = true
		}
	}
	return result
}

// AgentTool re-exports for delivery selection.
type AgentTool = roland.AgentTool

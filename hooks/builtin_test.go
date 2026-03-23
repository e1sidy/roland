package hooks

import (
	"strings"
	"testing"
)

func TestBuiltinHooks_Count(t *testing.T) {
	hooks := builtinHooks()
	if len(hooks) != 5 {
		t.Fatalf("builtinHooks() = %d, want 5", len(hooks))
	}
}

func TestSlateInstructions_Content(t *testing.T) {
	reg := DefaultRegistry()
	h := reg.Get("slate-instructions")
	if h == nil {
		t.Fatal("slate-instructions not found")
	}

	ctx := HookContext{RolandHome: "/tmp/roland"}
	content := h.ClaudeScript(ctx)
	if !strings.Contains(content, "slate show") {
		t.Error("slate-instructions should contain 'slate show'")
	}
	if !strings.Contains(content, "slate ready") {
		t.Error("slate-instructions should contain 'slate ready'")
	}
}

func TestRolandInstructions_Content(t *testing.T) {
	reg := DefaultRegistry()
	h := reg.Get("roland-instructions")
	if h == nil {
		t.Fatal("roland-instructions not found")
	}

	ctx := HookContext{RolandHome: "/tmp/roland"}
	content := h.ClaudeScript(ctx)
	if !strings.Contains(content, "roland checkpoint") {
		t.Error("roland-instructions should contain 'roland checkpoint'")
	}
	if !strings.Contains(content, "roland ship") {
		t.Error("roland-instructions should contain 'roland ship'")
	}
}

func TestRolandTaskContext_Content(t *testing.T) {
	reg := DefaultRegistry()
	h := reg.Get("roland-task-context")
	if h == nil {
		t.Fatal("roland-task-context not found")
	}

	ctx := HookContext{RolandHome: "/tmp/roland"}
	content := h.ClaudeScript(ctx)
	if !strings.Contains(content, "TASK_ID") {
		t.Error("roland-task-context should detect task ID")
	}
	if !strings.Contains(content, "slate show") {
		t.Error("roland-task-context should show task")
	}
	if !strings.Contains(content, "slate checkpoints") {
		t.Error("roland-task-context should show checkpoints")
	}
}

func TestBuiltinHooks_AllHaveClaudeScript(t *testing.T) {
	ctx := HookContext{RolandHome: "/tmp/roland"}
	for _, h := range builtinHooks() {
		if h.ClaudeScript == nil {
			t.Errorf("hook %q has nil ClaudeScript", h.Name)
			continue
		}
		content := h.ClaudeScript(ctx)
		if content == "" {
			t.Errorf("hook %q ClaudeScript returned empty", h.Name)
		}
	}
}

func TestBuiltinHooks_AllHaveOpenCodeSnippet(t *testing.T) {
	ctx := HookContext{RolandHome: "/tmp/roland"}
	for _, h := range builtinHooks() {
		if h.OpenCodeSnippet == nil {
			t.Errorf("hook %q has nil OpenCodeSnippet", h.Name)
			continue
		}
		content := h.OpenCodeSnippet(ctx)
		if content == "" {
			t.Errorf("hook %q OpenCodeSnippet returned empty", h.Name)
		}
	}
}

func TestEnabledForSource(t *testing.T) {
	reg := DefaultRegistry()
	cfg := map[string]bool{
		"slate-instructions": true,
		"roland-repos":       false,
	}

	enabled := EnabledForSource(cfg, SourceHome, reg)

	// slate-instructions: explicitly true.
	if !enabled["slate-instructions"] {
		t.Error("slate-instructions should be enabled")
	}
	// roland-repos: explicitly false.
	if enabled["roland-repos"] {
		t.Error("roland-repos should be disabled")
	}
	// slate-ready-tasks: not in config, defaults to enabled.
	if !enabled["slate-ready-tasks"] {
		t.Error("slate-ready-tasks should default to enabled")
	}
	// roland-task-context: SourceTask, should not appear in SourceHome results.
	if _, exists := enabled["roland-task-context"]; exists {
		t.Error("roland-task-context (SourceTask) should not be in SourceHome results")
	}
}

package hooks

import (
	"testing"
)

func TestRegister_Duplicate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register duplicate should panic")
		}
	}()

	reg := NewRegistry()
	h := &Hook{Name: "test"}
	reg.Register(h)
	reg.Register(h) // Should panic.
}

func TestGet_Found(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Hook{Name: "test"})

	h := reg.Get("test")
	if h == nil {
		t.Error("Get should return hook")
	}
}

func TestGet_NotFound(t *testing.T) {
	reg := NewRegistry()
	if reg.Get("missing") != nil {
		t.Error("Get should return nil for missing hook")
	}
}

func TestAll_Sorted(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Hook{Name: "zebra"})
	reg.Register(&Hook{Name: "alpha"})

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("All() = %d, want 2", len(all))
	}
	if all[0].Name != "alpha" {
		t.Errorf("first = %q, want %q", all[0].Name, "alpha")
	}
}

func TestBySource(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Hook{Name: "home1", Source: SourceHome})
	reg.Register(&Hook{Name: "task1", Source: SourceTask})
	reg.Register(&Hook{Name: "home2", Source: SourceHome})

	homeHooks := reg.BySource(SourceHome)
	if len(homeHooks) != 2 {
		t.Errorf("BySource(Home) = %d, want 2", len(homeHooks))
	}

	taskHooks := reg.BySource(SourceTask)
	if len(taskHooks) != 1 {
		t.Errorf("BySource(Task) = %d, want 1", len(taskHooks))
	}
}

func TestNames(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Hook{Name: "beta"})
	reg.Register(&Hook{Name: "alpha"})

	names := reg.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("Names() = %v, want [alpha beta]", names)
	}
}

func TestRegisterCustom_OverridesBuiltin(t *testing.T) {
	reg := DefaultRegistry()
	// Override a built-in hook.
	custom := &Hook{Name: "slate-instructions", Source: SourceHome, Event: "SessionStart"}
	reg.RegisterCustom(custom)

	h := reg.Get("slate-instructions")
	if h != custom {
		t.Error("RegisterCustom should override existing hook")
	}
	// Count should remain the same (replaced, not added).
	if len(reg.All()) != 5 {
		t.Errorf("All() = %d after override, want 5", len(reg.All()))
	}
}

func TestRegisterCustomHooks(t *testing.T) {
	reg := DefaultRegistry()
	customDefs := map[string]*CustomHookDef{
		"my-hook": {
			Event:   "SessionStart",
			Script:  "/path/to/my-hook.sh",
			Matcher: "Bash",
			Timeout: 5,
		},
		"another": {
			Event:  "PreCompact",
			Script: "/path/to/another.sh",
		},
	}
	reg.RegisterCustomHooks(customDefs)

	// Should have 5 built-in + 2 custom = 7.
	if len(reg.All()) != 7 {
		t.Fatalf("All() = %d, want 7", len(reg.All()))
	}

	h := reg.Get("my-hook")
	if h == nil {
		t.Fatal("my-hook not found")
	}
	if h.Event != "SessionStart" {
		t.Errorf("event = %q, want SessionStart", h.Event)
	}
	if h.Matcher != "Bash" {
		t.Errorf("matcher = %q, want Bash", h.Matcher)
	}
	if h.Timeout != 5 {
		t.Errorf("timeout = %d, want 5", h.Timeout)
	}

	// Verify script content.
	ctx := HookContext{RolandHome: "/tmp"}
	content := h.ClaudeScript(ctx)
	if content == "" {
		t.Error("ClaudeScript returned empty")
	}

	// Default matcher for another.
	h2 := reg.Get("another")
	if h2.Matcher != "*" {
		t.Errorf("default matcher = %q, want *", h2.Matcher)
	}
}

func TestDefaultRegistry(t *testing.T) {
	reg := DefaultRegistry()
	all := reg.All()
	if len(all) != 5 {
		t.Fatalf("DefaultRegistry has %d hooks, want 5", len(all))
	}

	// Verify specific hooks exist.
	expected := []string{
		"roland-instructions",
		"roland-repos",
		"roland-task-context",
		"slate-instructions",
		"slate-ready-tasks",
	}
	names := reg.Names()
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want)
		}
	}
}

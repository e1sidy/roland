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

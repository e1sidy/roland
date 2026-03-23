package persona

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != 4 {
		t.Fatalf("Names() = %d names, want 4: %v", len(names), names)
	}
	expected := []string{"builder", "planner", "researcher", "reviewer"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestGet_Builtin(t *testing.T) {
	home := t.TempDir()
	for _, name := range Names() {
		content, err := Get(home, name)
		if err != nil {
			t.Errorf("Get(%q): %v", name, err)
			continue
		}
		if content == "" {
			t.Errorf("Get(%q) returned empty content", name)
		}
	}
}

func TestGet_NotFound(t *testing.T) {
	home := t.TempDir()
	_, err := Get(home, "nonexistent")
	if err == nil {
		t.Error("Get should fail for nonexistent persona")
	}
}

func TestIsValid_Builtin(t *testing.T) {
	home := t.TempDir()
	if !IsValid(home, "builder") {
		t.Error("IsValid('builder') should be true")
	}
	if IsValid(home, "nonexistent") {
		t.Error("IsValid('nonexistent') should be false")
	}
}

func TestGet_CustomOverride(t *testing.T) {
	home := t.TempDir()
	personasDir := filepath.Join(home, "personas")
	os.MkdirAll(personasDir, 0o755)

	customContent := "# Custom Builder\n\nMy custom builder persona.\n"
	os.WriteFile(filepath.Join(personasDir, "builder.md"), []byte(customContent), 0o644)

	content, err := Get(home, "builder")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if content != customContent {
		t.Errorf("Get returned built-in instead of custom override")
	}
}

func TestCreate_FromBase(t *testing.T) {
	home := t.TempDir()

	if err := Create(home, "devops", "builder"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	content, err := Get(home, "devops")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if content == "" {
		t.Error("created persona has empty content")
	}
}

func TestCreate_Empty(t *testing.T) {
	home := t.TempDir()

	if err := Create(home, "custom", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	content, err := Get(home, "custom")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if content == "" {
		t.Error("created persona has empty content")
	}
}

func TestCreate_AlreadyExists(t *testing.T) {
	home := t.TempDir()
	Create(home, "myp", "")

	err := Create(home, "myp", "")
	if err == nil {
		t.Error("Create should fail if persona already exists")
	}
}

func TestDelete_Custom(t *testing.T) {
	home := t.TempDir()
	Create(home, "custom", "")

	if err := Delete(home, "custom"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if IsValid(home, "custom") {
		t.Error("persona should not exist after delete")
	}
}

func TestDelete_Builtin(t *testing.T) {
	home := t.TempDir()
	// No custom override — should fail.
	err := Delete(home, "builder")
	if err == nil {
		t.Error("Delete should fail for built-in persona without custom override")
	}
}

func TestDelete_BuiltinWithOverride(t *testing.T) {
	home := t.TempDir()
	// Create custom override of built-in.
	Create(home, "builder", "builder")

	// Should succeed — removes override, reverts to built-in.
	if err := Delete(home, "builder"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Built-in should still work.
	if !IsValid(home, "builder") {
		t.Error("built-in should still be available after deleting override")
	}
}

func TestListAll_BuiltinOnly(t *testing.T) {
	home := t.TempDir()
	personas, err := ListAll(home)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(personas) != 4 {
		t.Fatalf("ListAll = %d, want 4", len(personas))
	}
	for _, p := range personas {
		if p.Source != "builtin" {
			t.Errorf("%q source = %q, want %q", p.Name, p.Source, "builtin")
		}
	}
}

func TestListAll_MixedSources(t *testing.T) {
	home := t.TempDir()
	Create(home, "devops", "")
	Create(home, "builder", "builder") // Override built-in.

	personas, err := ListAll(home)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	// 4 built-in + 1 new custom = 5 total.
	if len(personas) != 5 {
		t.Fatalf("ListAll = %d, want 5", len(personas))
	}

	// Builder should show as custom (override).
	for _, p := range personas {
		if p.Name == "builder" && p.Source != "custom" {
			t.Errorf("builder source = %q, want %q (custom override)", p.Source, "custom")
		}
		if p.Name == "devops" && p.Source != "custom" {
			t.Errorf("devops source = %q, want %q", p.Source, "custom")
		}
	}
}

func TestEdit_Builtin(t *testing.T) {
	home := t.TempDir()
	path, err := Edit(home, "builder")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	// Should create a custom copy.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("custom copy not created at %s", path)
	}
}

func TestEdit_Custom(t *testing.T) {
	home := t.TempDir()
	Create(home, "devops", "")

	path, err := Edit(home, "devops")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("custom file not found at %s", path)
	}
}

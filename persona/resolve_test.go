package persona

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePersona_ProjectOverride(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)

	// Create base and override.
	os.WriteFile(filepath.Join(personaDir, "builder.md"), []byte("base builder"), 0o644)
	os.WriteFile(filepath.Join(personaDir, "backend-builder.md"), []byte("backend-specific builder"), 0o644)

	// With repo "backend" → should find override.
	content, err := ResolvePersona(home, "builder", "backend")
	if err != nil {
		t.Fatalf("ResolvePersona: %v", err)
	}
	if content != "backend-specific builder" {
		t.Errorf("content = %q, want backend-specific override", content)
	}
}

func TestResolvePersona_FallbackToBase(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)

	// Only base exists.
	os.WriteFile(filepath.Join(personaDir, "builder.md"), []byte("base builder"), 0o644)

	// With repo "frontend" → no override exists, falls back to base.
	content, err := ResolvePersona(home, "builder", "frontend")
	if err != nil {
		t.Fatalf("ResolvePersona: %v", err)
	}
	if content != "base builder" {
		t.Errorf("content = %q, want base builder", content)
	}
}

func TestResolvePersona_EmptyRepo(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)
	os.WriteFile(filepath.Join(personaDir, "builder.md"), []byte("base builder"), 0o644)

	// Empty repo → standard Get() behavior.
	content, err := ResolvePersona(home, "builder", "")
	if err != nil {
		t.Fatalf("ResolvePersona: %v", err)
	}
	if content != "base builder" {
		t.Errorf("content = %q, want base builder", content)
	}
}

func TestResolvePersona_BuiltinFallback(t *testing.T) {
	home := t.TempDir()
	// No custom personas — falls back to embedded built-in.
	content, err := ResolvePersona(home, "builder", "backend")
	if err != nil {
		t.Fatalf("ResolvePersona: %v", err)
	}
	if content == "" {
		t.Error("should fall back to embedded builder persona")
	}
}

func TestListOverrides(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)

	// Create some files.
	os.WriteFile(filepath.Join(personaDir, "builder.md"), []byte("base"), 0o644)
	os.WriteFile(filepath.Join(personaDir, "backend-builder.md"), []byte("override"), 0o644)
	os.WriteFile(filepath.Join(personaDir, "frontend-builder.md"), []byte("override"), 0o644)
	os.WriteFile(filepath.Join(personaDir, "reviewer.md"), []byte("base"), 0o644)

	overrides, err := ListOverrides(home)
	if err != nil {
		t.Fatal(err)
	}

	if len(overrides) != 2 {
		t.Errorf("overrides = %d, want 2 (backend-builder, frontend-builder)", len(overrides))
	}
}

func TestListOverrides_Empty(t *testing.T) {
	home := t.TempDir()
	overrides, err := ListOverrides(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(overrides) != 0 {
		t.Errorf("overrides = %d, want 0", len(overrides))
	}
}

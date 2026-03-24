package learning

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnrichPersona(t *testing.T) {
	home := t.TempDir()

	// Create a base persona file.
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)
	os.WriteFile(filepath.Join(personaDir, "builder.md"), []byte("# Builder\n\nYou are the builder.\n"), 0o644)

	patterns := []Pattern{
		{Text: "use jwt over sessions", TaskIDs: []string{"st-1", "st-2", "st-3"}},
		{Text: "add rate limiting", TaskIDs: []string{"st-4", "st-5", "st-6"}},
	}

	if err := EnrichPersona(home, "builder", patterns); err != nil {
		t.Fatalf("EnrichPersona: %v", err)
	}

	// Read the file back.
	data, err := os.ReadFile(filepath.Join(personaDir, "builder.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "## Learned Patterns") {
		t.Error("missing ## Learned Patterns header")
	}
	if !strings.Contains(content, "use jwt over sessions") {
		t.Error("missing pattern 1")
	}
	if !strings.Contains(content, "add rate limiting") {
		t.Error("missing pattern 2")
	}
	if !strings.Contains(content, "# Builder") {
		t.Error("original content should be preserved")
	}
}

func TestEnrichPersona_ReplaceExisting(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)
	os.WriteFile(filepath.Join(personaDir, "builder.md"), []byte("# Builder\n\n## Learned Patterns\n\n- old pattern\n"), 0o644)

	patterns := []Pattern{
		{Text: "new pattern", TaskIDs: []string{"st-1", "st-2", "st-3"}},
	}

	if err := EnrichPersona(home, "builder", patterns); err != nil {
		t.Fatalf("EnrichPersona: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(personaDir, "builder.md"))
	content := string(data)

	if strings.Contains(content, "old pattern") {
		t.Error("old pattern should be replaced")
	}
	if !strings.Contains(content, "new pattern") {
		t.Error("new pattern should be present")
	}
}

func TestShowLearnings(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)
	os.WriteFile(filepath.Join(personaDir, "builder.md"),
		[]byte("# Builder\n\n## Learned Patterns\n\n- pattern A\n- pattern B\n"), 0o644)

	content, err := ShowLearnings(home, "builder")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "pattern A") {
		t.Error("missing pattern A")
	}
	if !strings.Contains(content, "pattern B") {
		t.Error("missing pattern B")
	}
}

func TestShowLearnings_NoLearnings(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)
	os.WriteFile(filepath.Join(personaDir, "builder.md"),
		[]byte("# Builder\n\nYou are the builder.\n"), 0o644)

	content, err := ShowLearnings(home, "builder")
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		t.Errorf("expected empty, got %q", content)
	}
}

func TestResetLearnings(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)
	os.WriteFile(filepath.Join(personaDir, "builder.md"),
		[]byte("# Builder\n\nContent here.\n\n## Learned Patterns\n\n- pattern A\n"), 0o644)

	if err := ResetLearnings(home, "builder"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(personaDir, "builder.md"))
	content := string(data)

	if strings.Contains(content, "## Learned Patterns") {
		t.Error("learned section should be removed")
	}
	if !strings.Contains(content, "Content here") {
		t.Error("original content should be preserved")
	}
}

func TestResetLearnings_NoSection(t *testing.T) {
	home := t.TempDir()
	personaDir := filepath.Join(home, "personas")
	os.MkdirAll(personaDir, 0o755)
	os.WriteFile(filepath.Join(personaDir, "builder.md"),
		[]byte("# Builder\n"), 0o644)

	// Should not error when there's nothing to reset.
	if err := ResetLearnings(home, "builder"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

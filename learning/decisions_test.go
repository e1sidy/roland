package learning

import (
	"context"
	"testing"

	"github.com/e1sidy/slate"

	"github.com/e1sidy/roland/internal/testutil"
)

func TestIndexDecisions(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	// Create tasks with checkpoints containing decisions.
	t1, _ := store.Create(ctx, createParams("Task 1"))
	store.AddCheckpoint(ctx, t1.ID, "user", checkpointParams("done 1", "Used JWT over sessions"))
	store.AddCheckpoint(ctx, t1.ID, "user", checkpointParams("done 2", "Chose PostgreSQL"))

	t2, _ := store.Create(ctx, createParams("Task 2"))
	store.AddCheckpoint(ctx, t2.ID, "user", checkpointParams("done", "Used API versioning via URL path"))

	entries, err := IndexDecisions(ctx, store, AnalyzeParams{})
	if err != nil {
		t.Fatalf("IndexDecisions: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("entries = %d, want 3", len(entries))
	}
}

func TestSearchDecisions(t *testing.T) {
	entries := []DecisionEntry{
		{Text: "Used JWT over sessions", TaskID: "t1"},
		{Text: "Chose PostgreSQL pooling", TaskID: "t2"},
		{Text: "API versioning via URL", TaskID: "t3"},
	}

	results := SearchDecisions(entries, "jwt")
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].TaskID != "t1" {
		t.Errorf("taskID = %q, want t1", results[0].TaskID)
	}

	// Case insensitive.
	results2 := SearchDecisions(entries, "POSTGRESQL")
	if len(results2) != 1 {
		t.Errorf("case insensitive: %d, want 1", len(results2))
	}

	// Empty query returns all.
	all := SearchDecisions(entries, "")
	if len(all) != 3 {
		t.Errorf("empty query: %d, want 3", len(all))
	}
}

func TestFilterByTask(t *testing.T) {
	entries := []DecisionEntry{
		{Text: "A", TaskID: "t1"},
		{Text: "B", TaskID: "t2"},
		{Text: "C", TaskID: "t1"},
	}
	result := FilterByTask(entries, "t1")
	if len(result) != 2 {
		t.Errorf("filtered = %d, want 2", len(result))
	}
}

func TestFilterByPersona(t *testing.T) {
	entries := []DecisionEntry{
		{Text: "A", Persona: "builder"},
		{Text: "B", Persona: "reviewer"},
		{Text: "C", Persona: "builder"},
	}
	result := FilterByPersona(entries, "builder")
	if len(result) != 2 {
		t.Errorf("filtered = %d, want 2", len(result))
	}
}

func TestDecisionIndex_SaveLoad(t *testing.T) {
	home := t.TempDir()
	entries := []DecisionEntry{
		{Text: "Decision 1", TaskID: "t1", Persona: "builder"},
		{Text: "Decision 2", TaskID: "t2", Persona: "reviewer"},
	}

	if err := SaveDecisionIndex(home, entries); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadDecisionIndex(home)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("loaded = %d, want 2", len(loaded))
	}
	if loaded[0].Text != "Decision 1" {
		t.Errorf("text = %q", loaded[0].Text)
	}
}

func TestLoadDecisionIndex_Missing(t *testing.T) {
	home := t.TempDir()
	entries, err := LoadDecisionIndex(home)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if entries != nil {
		t.Error("should be nil for missing file")
	}
}

// --- Helpers ---

func createParams(title string) slate.CreateParams {
	return slate.CreateParams{Title: title}
}

func checkpointParams(done, decisions string) slate.CheckpointParams {
	return slate.CheckpointParams{Done: done, Decisions: decisions}
}

// Ensure we import the right slate package.
var _ = context.Background

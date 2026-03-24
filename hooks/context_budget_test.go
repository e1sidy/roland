package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/e1sidy/roland/internal/testutil"
	"github.com/e1sidy/slate"
)

func TestEstimateTokens(t *testing.T) {
	tt := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"abcd", 1},
		{"hello world", 3}, // 11 chars / 4 ≈ 3
		{"a", 1},
	}
	for _, tc := range tt {
		got := EstimateTokens(tc.input)
		if got != tc.want {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestSummarizeCheckpoints(t *testing.T) {
	cps := []*slate.Checkpoint{
		{Done: "First checkpoint", CreatedAt: time.Now().Add(-3 * time.Hour)},
		{Done: "Second checkpoint", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{Done: "Third checkpoint", CreatedAt: time.Now().Add(-1 * time.Hour)},
	}

	// Large budget — all fit.
	result := SummarizeCheckpoints(cps, 1000)
	if result == "" {
		t.Error("should produce output with large budget")
	}

	// Tiny budget — should truncate.
	result2 := SummarizeCheckpoints(cps, 10)
	if result2 == "" {
		t.Error("should produce at least header with tiny budget")
	}
}

func TestSummarizeCheckpoints_Empty(t *testing.T) {
	result := SummarizeCheckpoints(nil, 100)
	if result != "" {
		t.Error("empty checkpoints should produce empty string")
	}
}

func TestPruneStaleReady(t *testing.T) {
	now := time.Now()
	tasks := []*slate.Task{
		{ID: "t1", CreatedAt: now.Add(-1 * time.Hour)},      // fresh
		{ID: "t2", CreatedAt: now.Add(-48 * time.Hour)},     // 2 days old
		{ID: "t3", CreatedAt: now.Add(-1000 * time.Hour)},   // 41+ days old
	}

	// Prune tasks older than 30 days.
	fresh := PruneStaleReady(tasks, 30*24*time.Hour)
	if len(fresh) != 2 {
		t.Errorf("fresh = %d, want 2 (t3 pruned)", len(fresh))
	}
}

func TestPruneStaleReady_AllFresh(t *testing.T) {
	now := time.Now()
	tasks := []*slate.Task{
		{ID: "t1", CreatedAt: now},
		{ID: "t2", CreatedAt: now.Add(-1 * time.Hour)},
	}
	fresh := PruneStaleReady(tasks, 24*time.Hour)
	if len(fresh) != 2 {
		t.Errorf("fresh = %d, want 2", len(fresh))
	}
}

func TestBuildContext(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	task, _ := store.Create(ctx, slate.CreateParams{Title: "Test task", Description: "A test"})
	store.AddCheckpoint(ctx, task.ID, "user", slate.CheckpointParams{
		Done:      "Implemented auth",
		Next:      "Add tests",
		Decisions: "Used JWT",
	})

	result, err := BuildContext(ctx, store, task.ID, 4096)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if result == "" {
		t.Error("context should not be empty")
	}
	// Should contain task title.
	if !containsStr(result, "Test task") {
		t.Error("context should contain task title")
	}
	// Should contain checkpoint.
	if !containsStr(result, "Implemented auth") {
		t.Error("context should contain latest checkpoint")
	}
}

func TestBuildContext_SmallBudget(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	task, _ := store.Create(ctx, slate.CreateParams{Title: "Test"})

	// Small budget — should still include task section.
	result, err := BuildContext(ctx, store, task.ID, 50)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if result == "" {
		t.Error("even with small budget, task section should be included")
	}
}

func TestBuildContext_WithDeps(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	blocker, _ := store.Create(ctx, slate.CreateParams{Title: "Blocker task"})
	blocked, _ := store.Create(ctx, slate.CreateParams{Title: "Blocked task"})
	store.AddDependency(ctx, blocker.ID, blocked.ID, slate.Blocks)

	result, err := BuildContext(ctx, store, blocked.ID, 4096)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if !containsStr(result, "Blocker task") {
		t.Error("context should include blocking dependency")
	}
}

func TestBuildContext_DefaultBudget(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)
	task, _ := store.Create(ctx, slate.CreateParams{Title: "Test"})

	// Budget 0 should use default.
	result, err := BuildContext(ctx, store, task.ID, 0)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if result == "" {
		t.Error("default budget should produce output")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && context.Background() != nil && findSubstr(s, sub)
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

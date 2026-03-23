package roland

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/e1sidy/roland/internal/testutil"
	"github.com/e1sidy/roland/persona"
	"github.com/e1sidy/roland/skill"
	"github.com/e1sidy/roland/workspace"
	"github.com/e1sidy/slate"
)

// TestLifecycle_PickupWorkShipDone exercises the full Roland lifecycle at the
// SDK level: pickup (claim, workspace, attrs, persona, skills) → work
// (checkpoint, latest) → ship (review status) → done (close).
//
// This doesn't launch agents or create real git repos, but verifies the SDK
// orchestration that the CLI commands wrap.
func TestLifecycle_PickupWorkShipDone(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)
	home := testutil.TempHome(t)

	// --- Setup: ensure attrs and create a task ---

	if err := EnsureAttrs(ctx, store); err != nil {
		t.Fatalf("EnsureAttrs: %v", err)
	}

	task, err := store.Create(ctx, slate.CreateParams{
		Title:    "Implement auth",
		Type:     slate.TypeFeature,
		Priority: slate.P1,
	})
	if err != nil {
		t.Fatalf("Create task: %v", err)
	}

	// --- PICKUP ---

	// Step 1: Claim task.
	_, err = store.Claim(ctx, task.ID, "roland")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// Verify task is now in_progress.
	claimed, _ := store.Get(ctx, task.ID)
	if claimed.Status != slate.StatusInProgress {
		t.Errorf("status after claim = %q, want %q", claimed.Status, slate.StatusInProgress)
	}
	if claimed.Assignee != "roland" {
		t.Errorf("assignee = %q, want %q", claimed.Assignee, "roland")
	}

	// Step 2: Create workspace.
	td, err := workspace.Create(home, task.ID, task.Title)
	if err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	if td.TaskID != task.ID {
		t.Errorf("TaskDir.TaskID = %q, want %q", td.TaskID, task.ID)
	}

	// Step 3: Set attrs.
	reposJSON, _ := json.Marshal([]string{"backend"})
	if err := store.SetAttr(ctx, task.ID, AttrRepos, string(reposJSON)); err != nil {
		t.Fatalf("SetAttr repos: %v", err)
	}
	if err := store.SetAttr(ctx, task.ID, AttrPersonaUsed, "builder"); err != nil {
		t.Fatalf("SetAttr persona: %v", err)
	}
	if err := store.SetAttr(ctx, task.ID, AttrSessionCount, "1"); err != nil {
		t.Fatalf("SetAttr session_count: %v", err)
	}

	// Step 4: Verify persona exists.
	if !persona.IsValid(home, "builder") {
		t.Fatal("builder persona not valid")
	}
	content, err := persona.Get(home, "builder")
	if err != nil {
		t.Fatalf("Get persona: %v", err)
	}
	if content == "" {
		t.Error("persona content empty")
	}

	// Step 5: Verify skill matching (empty registry = no skills injected).
	matchCtx := &skill.MatchContext{
		Persona:  "builder",
		TaskType: string(task.Type),
		Labels:   task.Labels,
	}
	injected, err := skill.InjectMatching(home, td.Path, matchCtx)
	if err != nil {
		t.Fatalf("InjectMatching: %v", err)
	}
	// No skills registered, so none injected.
	if len(injected) != 0 {
		t.Errorf("injected %d skills, want 0 (none registered)", len(injected))
	}

	// --- WORK ---

	// Add a checkpoint (simulates agent progress).
	cp, err := store.AddCheckpoint(ctx, task.ID, "roland", slate.CheckpointParams{
		Done:      "Implemented auth middleware",
		Decisions: "Used JWT over sessions",
		Next:      "Write tests",
		Blockers:  "",
		Files:     []string{"auth.go", "auth_test.go"},
	})
	if err != nil {
		t.Fatalf("AddCheckpoint: %v", err)
	}
	if cp.Done != "Implemented auth middleware" {
		t.Errorf("checkpoint.Done = %q", cp.Done)
	}

	// Verify latest checkpoint (resumption briefing uses this).
	latest, err := store.LatestCheckpoint(ctx, task.ID)
	if err != nil {
		t.Fatalf("LatestCheckpoint: %v", err)
	}
	if latest.Done != "Implemented auth middleware" {
		t.Errorf("latest.Done = %q", latest.Done)
	}
	if latest.Decisions != "Used JWT over sessions" {
		t.Errorf("latest.Decisions = %q", latest.Decisions)
	}

	// Increment session count (as work_cmd does).
	if err := store.SetAttr(ctx, task.ID, AttrSessionCount, "2"); err != nil {
		t.Fatalf("increment session_count: %v", err)
	}

	// --- SHIP (quality gate check) ---

	// Set review_status to changes_requested.
	if err := store.SetAttr(ctx, task.ID, AttrReviewStatus, "changes_requested"); err != nil {
		t.Fatalf("SetAttr review_status: %v", err)
	}
	attr, _ := store.GetAttr(ctx, task.ID, AttrReviewStatus)
	if attr.Value != "changes_requested" {
		t.Errorf("review_status = %q, want changes_requested", attr.Value)
	}

	// Update to approved.
	if err := store.SetAttr(ctx, task.ID, AttrReviewStatus, "approved"); err != nil {
		t.Fatalf("SetAttr review_status approved: %v", err)
	}
	attr, _ = store.GetAttr(ctx, task.ID, AttrReviewStatus)
	if attr.Value != "approved" {
		t.Errorf("review_status = %q, want approved", attr.Value)
	}

	// --- DONE ---

	// Verify we can read repos attr for cleanup.
	reposAttr, err := store.GetAttr(ctx, task.ID, AttrRepos)
	if err != nil {
		t.Fatalf("GetAttr repos: %v", err)
	}
	var repos []string
	if err := json.Unmarshal([]byte(reposAttr.Value), &repos); err != nil {
		t.Fatalf("unmarshal repos: %v", err)
	}
	if len(repos) != 1 || repos[0] != "backend" {
		t.Errorf("repos = %v, want [backend]", repos)
	}

	// Remove workspace (simulates done cleanup).
	if err := workspace.Remove(home, task.ID); err != nil {
		t.Fatalf("Remove workspace: %v", err)
	}

	// Verify workspace is gone.
	_, err = workspace.Open(home, task.ID)
	if err == nil {
		t.Error("workspace should not exist after Remove")
	}

	// Close task.
	if err := store.CloseTask(ctx, task.ID, "done", "roland"); err != nil {
		t.Fatalf("CloseTask: %v", err)
	}

	// Verify task is closed.
	closed, _ := store.Get(ctx, task.ID)
	if !closed.Status.IsTerminal() {
		t.Errorf("status after close = %q, want terminal", closed.Status)
	}
	if closed.CloseReason != "done" {
		t.Errorf("close_reason = %q, want %q", closed.CloseReason, "done")
	}
}

// TestLifecycle_PickupRollback verifies that a failed pickup step
// cleans up all previously created resources.
func TestLifecycle_PickupRollback(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)
	home := testutil.TempHome(t)

	EnsureAttrs(ctx, store)

	task, _ := store.Create(ctx, slate.CreateParams{Title: "Rollback test"})

	// Simulate pickup steps 1-2, then simulate failure.
	store.Claim(ctx, task.ID, "roland")
	td, _ := workspace.Create(home, task.ID, task.Title)

	// Verify state exists.
	claimed, _ := store.Get(ctx, task.ID)
	if claimed.Status != slate.StatusInProgress {
		t.Fatal("task should be in_progress")
	}
	if td == nil {
		t.Fatal("workspace should exist")
	}

	// Simulate rollback (as pickup_cmd.go does on failure).
	workspace.Remove(home, task.ID)
	store.ReleaseClaim(context.Background(), task.ID, "roland")

	// Verify clean state.
	released, _ := store.Get(ctx, task.ID)
	if released.Status != slate.StatusOpen {
		t.Errorf("status after rollback = %q, want %q", released.Status, slate.StatusOpen)
	}
	if released.Assignee != "" {
		t.Errorf("assignee after rollback = %q, want empty", released.Assignee)
	}
	_, err := workspace.Open(home, task.ID)
	if err == nil {
		t.Error("workspace should not exist after rollback")
	}
}

// TestLifecycle_CleanPrunesOrphans verifies that orphaned workspaces
// (for terminal tasks) are detected.
func TestLifecycle_CleanPrunesOrphans(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)
	home := testutil.TempHome(t)

	EnsureAttrs(ctx, store)

	// Create and close a task, but leave the workspace.
	task, _ := store.Create(ctx, slate.CreateParams{Title: "To be orphaned"})
	workspace.Create(home, task.ID, task.Title)
	store.CloseTask(ctx, task.ID, "done", "test")

	// List workspaces — the orphan should be found.
	dirs, err := workspace.List(home)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(dirs) != 1 {
		t.Fatalf("List = %d, want 1", len(dirs))
	}

	// Check task is terminal (what clean_cmd does).
	got, _ := store.Get(ctx, dirs[0].TaskID)
	if !got.Status.IsTerminal() {
		t.Error("task should be terminal (orphaned)")
	}

	// Remove orphan (what clean_cmd does).
	workspace.Remove(home, dirs[0].TaskID)

	// Verify gone.
	after, _ := workspace.List(home)
	if len(after) != 0 {
		t.Error("orphaned workspace should be removed")
	}
}

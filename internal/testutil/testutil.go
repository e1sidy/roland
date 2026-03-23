// Package testutil provides helpers for Roland tests.
package testutil

import (
	"context"
	"testing"

	"github.com/e1sidy/slate"
)

// TempHome creates a temporary directory suitable for use as ROLAND_HOME.
// It is automatically cleaned up when the test finishes.
func TempHome(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// TempSlateStore opens an in-memory Slate store for testing.
// The store is automatically closed when the test finishes.
func TempSlateStore(t *testing.T) (*slate.Store, context.Context) {
	t.Helper()
	ctx := context.Background()
	dbPath := t.TempDir() + "/test.db"
	store, err := slate.Open(ctx, dbPath, slate.WithPrefix("st"))
	if err != nil {
		t.Fatalf("open slate store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store, ctx
}

package hooks

import (
	"os"
	"strings"
	"testing"
)

func TestSyncOpenCode_Creates(t *testing.T) {
	targetDir := t.TempDir()
	ctx := HookContext{RolandHome: "/tmp/roland"}

	hooks := []*Hook{
		{
			Name: "test-hook",
			OpenCodeSnippet: func(ctx HookContext) string {
				return `return "hello from test";`
			},
		},
	}

	if err := syncOpenCode(hooks, targetDir, ctx); err != nil {
		t.Fatalf("syncOpenCode: %v", err)
	}

	path := openCodePluginPath(targetDir)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("plugin file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "hello from test") {
		t.Error("plugin should contain hook snippet")
	}
	if !strings.Contains(content, "module.exports") {
		t.Error("plugin should be a valid JS module")
	}
}

func TestSyncOpenCode_NoHooks(t *testing.T) {
	targetDir := t.TempDir()
	ctx := HookContext{RolandHome: "/tmp/roland"}

	// Create a plugin first.
	syncOpenCode([]*Hook{
		{Name: "test", OpenCodeSnippet: func(ctx HookContext) string { return `return "x";` }},
	}, targetDir, ctx)

	// Sync with no hooks — should remove plugin.
	if err := syncOpenCode(nil, targetDir, ctx); err != nil {
		t.Fatalf("syncOpenCode: %v", err)
	}

	path := openCodePluginPath(targetDir)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("plugin file should be removed when no hooks enabled")
	}
}

func TestGenerateOpenCodePlugin(t *testing.T) {
	snippets := []string{
		`return "snippet1";`,
		`return "snippet2";`,
	}
	content := generateOpenCodePlugin(snippets)

	if !strings.Contains(content, "module.exports") {
		t.Error("should contain module.exports")
	}
	if !strings.Contains(content, "snippet1") {
		t.Error("should contain snippet1")
	}
	if !strings.Contains(content, "snippet2") {
		t.Error("should contain snippet2")
	}
}

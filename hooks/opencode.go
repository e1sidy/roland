package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// openCodePluginPath returns the path to the combined Roland hooks JS plugin.
func openCodePluginPath(targetDir string) string {
	return filepath.Join(targetDir, ".opencode", "plugins", "roland-hooks.js")
}

// syncOpenCode generates a single JS plugin combining all enabled hooks.
// If no hooks are enabled, removes the plugin file.
func syncOpenCode(enabledHooks []*Hook, targetDir string, ctx HookContext) error {
	pluginPath := openCodePluginPath(targetDir)

	if len(enabledHooks) == 0 {
		os.Remove(pluginPath)
		return nil
	}

	// Sort hooks for deterministic output.
	sort.Slice(enabledHooks, func(i, j int) bool {
		return enabledHooks[i].Name < enabledHooks[j].Name
	})

	// Collect JS snippets.
	var snippets []string
	for _, h := range enabledHooks {
		if h.OpenCodeSnippet != nil {
			snippet := h.OpenCodeSnippet(ctx)
			if snippet != "" {
				snippets = append(snippets, snippet)
			}
		}
	}

	if len(snippets) == 0 {
		os.Remove(pluginPath)
		return nil
	}

	// Generate combined JS plugin.
	content := generateOpenCodePlugin(snippets)

	// Write plugin file.
	dir := filepath.Dir(pluginPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create opencode plugins dir: %w", err)
	}
	return os.WriteFile(pluginPath, []byte(content), 0o644)
}

// generateOpenCodePlugin creates a JS file that registers all snippets.
func generateOpenCodePlugin(snippets []string) string {
	var b strings.Builder
	b.WriteString("// Roland hooks — auto-generated. Do not edit.\n")
	b.WriteString("module.exports = function(api) {\n")
	b.WriteString("  api.on('session.created', function() {\n")
	b.WriteString("    var parts = [];\n")
	for i, s := range snippets {
		b.WriteString(fmt.Sprintf("    // Hook %d\n", i+1))
		b.WriteString(fmt.Sprintf("    parts.push((function() { %s })());\n", s))
	}
	b.WriteString("    return parts.join('\\n');\n")
	b.WriteString("  });\n")
	b.WriteString("};\n")
	return b.String()
}

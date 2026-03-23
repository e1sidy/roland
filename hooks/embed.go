package hooks

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed scripts/*.sh
var scriptsFS embed.FS

// MaterializeScripts writes embedded hook scripts to the target directory.
// This is called during `roland init` to make scripts available on disk.
func MaterializeScripts(targetDir string) error {
	entries, err := scriptsFS.ReadDir("scripts")
	if err != nil {
		return fmt.Errorf("read embedded scripts: %w", err)
	}

	dir := filepath.Join(targetDir, "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := scriptsFS.ReadFile("scripts/" + e.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Name(), err)
		}
		dest := filepath.Join(dir, e.Name())
		if err := os.WriteFile(dest, data, 0o755); err != nil {
			return fmt.Errorf("write %s: %w", e.Name(), err)
		}
	}
	return nil
}

package hooks

import (
	"fmt"

	"github.com/e1sidy/roland"
)

// Manager orchestrates hook installation, uninstallation, and syncing.
type Manager struct {
	registry *Registry
}

// NewManager creates a manager backed by the given registry.
func NewManager(r *Registry) *Manager {
	return &Manager{registry: r}
}

// Registry returns the underlying hook registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}

// Install installs a single hook into the target directory.
func (m *Manager) Install(name, targetDir string, agent roland.AgentTool, ctx HookContext) error {
	h := m.registry.Get(name)
	if h == nil {
		return fmt.Errorf("hook %q not found in registry", name)
	}

	switch agent {
	case roland.AgentClaude:
		return installClaude(h, targetDir, ctx)
	case roland.AgentOpenCode:
		// OpenCode hooks are synced in bulk, not individually.
		return nil
	default:
		return fmt.Errorf("unsupported agent %q for hook install", agent)
	}
}

// Uninstall removes a single hook from the target directory.
func (m *Manager) Uninstall(name, targetDir string, agent roland.AgentTool) error {
	switch agent {
	case roland.AgentClaude:
		return uninstallClaude(name, targetDir)
	case roland.AgentOpenCode:
		// OpenCode hooks are synced in bulk.
		return nil
	default:
		return fmt.Errorf("unsupported agent %q for hook uninstall", agent)
	}
}

// Sync ensures the target directory has exactly the enabled hooks installed.
// Installs missing hooks, removes extra ones. Idempotent.
func (m *Manager) Sync(targetDir string, enabled map[string]bool, agent roland.AgentTool, ctx HookContext) error {
	// For Claude: sync individual hooks.
	if agent == roland.AgentClaude {
		installed := make(map[string]bool)
		for _, name := range m.Installed(targetDir) {
			installed[name] = true
		}

		// Install or update all enabled hooks.
		for name, isEnabled := range enabled {
			if !isEnabled {
				continue
			}
			if err := m.Install(name, targetDir, agent, ctx); err != nil {
				return fmt.Errorf("install %q: %w", name, err)
			}
		}

		// Uninstall hooks that are installed but not enabled.
		for name := range installed {
			if !enabled[name] && m.registry.Get(name) != nil {
				if err := m.Uninstall(name, targetDir, agent); err != nil {
					return fmt.Errorf("uninstall %q: %w", name, err)
				}
			}
		}
		return nil
	}

	// For OpenCode: sync all enabled hooks in one JS plugin.
	if agent == roland.AgentOpenCode {
		var enabledHooks []*Hook
		for name, isEnabled := range enabled {
			if isEnabled {
				if h := m.registry.Get(name); h != nil {
					enabledHooks = append(enabledHooks, h)
				}
			}
		}
		return syncOpenCode(enabledHooks, targetDir, ctx)
	}

	return nil
}

// Installed returns the names of hooks currently installed in the target directory.
func (m *Manager) Installed(targetDir string) []string {
	return installedClaudeHooks(targetDir)
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/roland/hooks"
	"github.com/spf13/cobra"
)

func hookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage context injection hooks",
	}

	cmd.AddCommand(
		hookListCmd(),
		hookAddCmd(),
		hookRemoveCmd(),
		hookSyncCmd(),
	)

	return cmd
}

// loadRegistryWithCustom returns a registry containing built-in + custom hooks.
func loadRegistryWithCustom() *hooks.Registry {
	reg := hooks.DefaultRegistry()
	// Convert config custom hooks to the hooks package type.
	customDefs := make(map[string]*hooks.CustomHookDef, len(cfg.CustomHooks))
	for name, def := range cfg.CustomHooks {
		customDefs[name] = &hooks.CustomHookDef{
			Event:   def.Event,
			Script:  def.Script,
			Matcher: def.Matcher,
			Timeout: def.Timeout,
		}
	}
	reg.RegisterCustomHooks(customDefs)
	return reg
}

func hookListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all hooks and their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := loadRegistryWithCustom()
			installed := hooks.NewManager(reg).Installed(cfg.Home)
			installedSet := make(map[string]bool)
			for _, name := range installed {
				installedSet[name] = true
			}

			for _, h := range reg.All() {
				enabled := true
				if v, ok := cfg.Hooks[h.Name]; ok {
					enabled = v
				}

				enabledStr := colorize(colorGreen, "enabled")
				if !enabled {
					enabledStr = colorize(colorRed, "disabled")
				}

				installedStr := ""
				if installedSet[h.Name] {
					installedStr = colorize(colorGreen, " (installed)")
				}

				source := string(h.Source)
				// Mark custom hooks.
				if _, isCustom := cfg.CustomHooks[h.Name]; isCustom {
					source = "custom"
				}

				fmt.Printf("  %-30s %-10s %s%s\n", h.Name, enabledStr, source, installedStr)
			}
			return nil
		},
	}
}

func hookAddCmd() *cobra.Command {
	var (
		event   string
		script  string
		matcher string
		timeout int
	)

	cmd := &cobra.Command{
		Use:   "add <hook-name>",
		Short: "Add a hook (built-in or custom with --event/--script)",
		Long: `Enable a built-in hook by name, or create a custom hook with --event and --script.

Examples:
  roland hook add slate-instructions          # Enable built-in hook
  roland hook add my-hook --event SessionStart --script ./hooks/my-hook.sh
  roland hook add pre-ship --event PreToolUse --matcher Bash --script ./validate.sh`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if script != "" {
				// Custom hook: requires --event and --script.
				if event == "" {
					return fmt.Errorf("--event is required for custom hooks (e.g., SessionStart, PreToolUse)")
				}

				// Resolve script path to absolute.
				absScript, err := filepath.Abs(script)
				if err != nil {
					return fmt.Errorf("resolve script path: %w", err)
				}
				if _, err := os.Stat(absScript); err != nil {
					return fmt.Errorf("script not found: %s", absScript)
				}

				if matcher == "" {
					matcher = "*"
				}

				// Store in config.
				cfg.CustomHooks[name] = &roland.CustomHookDef{
					Event:   event,
					Script:  absScript,
					Matcher: matcher,
					Timeout: timeout,
				}
				cfg.Hooks[name] = true
				if err := roland.SaveConfig(cfg); err != nil {
					return fmt.Errorf("save config: %w", err)
				}

				// Install via registry.
				reg := loadRegistryWithCustom()
				mgr := hooks.NewManager(reg)
				hctx := hooks.HookContext{
					RolandHome: cfg.Home,
					TargetDir:  cfg.Home,
					SlateHome:  cfg.SlateHome,
				}
				if err := mgr.Install(name, cfg.Home, cfg.Agent, hctx); err != nil {
					return fmt.Errorf("install hook: %w", err)
				}

				fmt.Printf("Added custom hook %q (event=%s, script=%s)\n", name, event, absScript)
				return nil
			}

			// Built-in hook: just enable it.
			reg := hooks.DefaultRegistry()
			h := reg.Get(name)
			if h == nil {
				return fmt.Errorf("hook %q not found; use --event and --script for custom hooks, or choose from: %s", name, strings.Join(reg.Names(), ", "))
			}

			cfg.Hooks[name] = true
			if err := roland.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			mgr := hooks.NewManager(reg)
			hctx := hooks.HookContext{
				RolandHome: cfg.Home,
				TargetDir:  cfg.Home,
				SlateHome:  cfg.SlateHome,
			}
			if err := mgr.Install(name, cfg.Home, cfg.Agent, hctx); err != nil {
				return fmt.Errorf("install hook: %w", err)
			}

			fmt.Println("Enabled and installed hook:", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&event, "event", "", "Agent event trigger (e.g., SessionStart, PreToolUse, PreCompact)")
	cmd.Flags().StringVar(&script, "script", "", "Path to bash script for the hook")
	cmd.Flags().StringVar(&matcher, "matcher", "*", "Tool matcher filter ('*' = all tools, 'Bash' = Bash only)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Hook timeout in seconds (0 = default 10s)")

	return cmd
}

func hookRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <hook-name>",
		Short: "Disable and uninstall a hook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Disable in config.
			cfg.Hooks[name] = false
			// Remove custom definition if it exists.
			delete(cfg.CustomHooks, name)
			if err := roland.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			// Uninstall.
			reg := loadRegistryWithCustom()
			mgr := hooks.NewManager(reg)
			if err := mgr.Uninstall(name, cfg.Home, cfg.Agent); err != nil {
				return fmt.Errorf("uninstall hook: %w", err)
			}

			fmt.Println("Removed hook:", name)
			return nil
		},
	}
}

func hookSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync hook installations to match config",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := loadRegistryWithCustom()
			mgr := hooks.NewManager(reg)
			enabled := hooks.EnabledForSource(cfg.Hooks, hooks.SourceHome, reg)
			hctx := hooks.HookContext{
				RolandHome: cfg.Home,
				TargetDir:  cfg.Home,
				SlateHome:  cfg.SlateHome,
			}
			if err := mgr.Sync(cfg.Home, enabled, cfg.Agent, hctx); err != nil {
				return fmt.Errorf("sync hooks: %w", err)
			}
			fmt.Println("Hooks synced.")
			return nil
		},
	}
}

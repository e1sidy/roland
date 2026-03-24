package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/e1sidy/roland"
	rolandembed "github.com/e1sidy/roland/embed"
	"github.com/e1sidy/roland/hooks"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	var (
		here   bool
		update bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a Roland home directory",
		Long: `Creates the ROLAND_HOME directory structure, writes default config,
installs hooks, and writes the home pointer file.

With --here, uses the current directory as ROLAND_HOME.
With --update, overwrites existing config and reinstalls hooks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var home string
			if here {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("get cwd: %w", err)
				}
				home = cwd
			} else {
				h, err := roland.ResolveHome()
				if err != nil {
					return fmt.Errorf("resolve home: %w", err)
				}
				home = h
			}

			// Check if already initialized.
			configPath := roland.ConfigPath(home)
			if _, err := os.Stat(configPath); err == nil && !update {
				fmt.Println("Roland is already initialized at", home)
				fmt.Println("Use --update to reinstall hooks and refresh config.")
				return nil
			}

			// Create directory structure.
			dirs := []string{
				home,
				roland.ReposDir(home),
				roland.TasksDir(home),
				roland.WorktreesDir(home),
				filepath.Join(home, "personas"),
				filepath.Join(home, "exports"),
				filepath.Join(home, "scripts"),
				filepath.Join(home, ".skills"),
				filepath.Join(home, ".claude"),
				filepath.Join(home, ".opencode"),
				filepath.Join(home, "hooks"),
			}
			for _, d := range dirs {
				if err := os.MkdirAll(d, 0o755); err != nil {
					return fmt.Errorf("create dir %s: %w", d, err)
				}
			}

			// Write or update config.
			c, err := roland.LoadConfig(home)
			if err != nil || update {
				c = roland.DefaultConfig()
				c.Home = home
			}
			if err := roland.SaveConfig(c); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			// Write CLAUDE.md and settings.
			if err := rolandembed.WriteClaudeMD(home, home, update); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write CLAUDE.md: %v\n", err)
			}
			if err := rolandembed.WriteClaudeSettings(home); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write settings: %v\n", err)
			}

			// Install hooks.
			reg := hooks.DefaultRegistry()
			mgr := hooks.NewManager(reg)
			enabled := hooks.EnabledForSource(c.Hooks, hooks.SourceHome, reg)
			hctx := hooks.HookContext{
				RolandHome: home,
				TargetDir:  home,
				SlateHome:  c.SlateHome,
			}
			if err := mgr.Sync(home, enabled, c.Agent, hctx); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: hook sync failed: %v\n", err)
			}

			// Write home pointer.
			if err := roland.WriteHomePointer(home); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write home pointer: %v\n", err)
			}

			fmt.Println("Initialized Roland at", home)
			return nil
		},
	}

	cmd.Flags().BoolVar(&here, "here", false, "Use current directory as ROLAND_HOME")
	cmd.Flags().BoolVar(&update, "update", false, "Update existing installation (overwrite config, reinstall hooks)")

	return cmd
}

package main

import (
	"fmt"
	"strings"

	"github.com/e1sidy/roland"
	rolandembed "github.com/e1sidy/roland/embed"
	"github.com/e1sidy/roland/hooks"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Roland configuration",
		Long:  "View and modify Roland settings: agent, IDE, hooks.",
	}

	cmd.AddCommand(
		configAgentCmd(),
		configIDECmd(),
		configHooksCmd(),
		configResetClaudeMDCmd(),
	)

	return cmd
}

func configAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent [name]",
		Short: "Get or set the default AI coding agent",
		Long:  "Without arguments, shows the current agent. With an argument, sets it.\nValid agents: " + strings.Join(roland.ValidAgentNames(), ", "),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Println(cfg.Agent)
				return nil
			}
			agent := roland.AgentTool(args[0])
			if !agent.IsValid() {
				return fmt.Errorf("invalid agent %q; valid agents: %s", args[0], strings.Join(roland.ValidAgentNames(), ", "))
			}
			cfg.Agent = agent
			if err := roland.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("Agent set to", agent)
			return nil
		},
	}
}

func configIDECmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ide [name]",
		Short: "Get or set the preferred IDE",
		Long:  "Without arguments, shows the current IDE. With an argument, sets it.\nValid IDEs: " + strings.Join(roland.ValidIDENames(), ", "),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Println(cfg.IDE)
				return nil
			}
			ide := roland.IDE(args[0])
			if !ide.IsValid() {
				return fmt.Errorf("invalid IDE %q; valid IDEs: %s", args[0], strings.Join(roland.ValidIDENames(), ", "))
			}
			cfg.IDE = ide
			if err := roland.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("IDE set to", ide)
			return nil
		},
	}
}

func configHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage hook configuration",
	}

	cmd.AddCommand(
		configHooksListCmd(),
		configHooksEnableCmd(),
		configHooksDisableCmd(),
		configHooksSyncCmd(),
	)

	return cmd
}

func configHooksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all hooks and their enabled status",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := hooks.DefaultRegistry()
			for _, h := range reg.All() {
				enabled := true
				if v, ok := cfg.Hooks[h.Name]; ok {
					enabled = v
				}
				status := colorize(colorGreen, "enabled")
				if !enabled {
					status = colorize(colorRed, "disabled")
				}
				fmt.Printf("  %-30s %s  [%s]\n", h.Name, status, h.Source)
			}
			return nil
		},
	}
}

func configHooksEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <hook-name>",
		Short: "Enable a hook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			reg := hooks.DefaultRegistry()
			if reg.Get(name) == nil {
				return fmt.Errorf("hook %q not found; available: %s", name, strings.Join(reg.Names(), ", "))
			}
			cfg.Hooks[name] = true
			if err := roland.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("Enabled hook:", name)
			return nil
		},
	}
}

func configHooksDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <hook-name>",
		Short: "Disable a hook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			reg := hooks.DefaultRegistry()
			if reg.Get(name) == nil {
				return fmt.Errorf("hook %q not found; available: %s", name, strings.Join(reg.Names(), ", "))
			}
			cfg.Hooks[name] = false
			if err := roland.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("Disabled hook:", name)
			return nil
		},
	}
}

func configHooksSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync hook installations to match config",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := hooks.DefaultRegistry()
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

func configResetClaudeMDCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-claude-md",
		Short: "Reset CLAUDE.md to the default template",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rolandembed.WriteClaudeMD(cfg.Home, cfg.Home, true); err != nil {
				return fmt.Errorf("write CLAUDE.md: %w", err)
			}
			fmt.Println("CLAUDE.md reset to default.")
			return nil
		},
	}
}

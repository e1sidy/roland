package main

import (
	"fmt"

	"github.com/e1sidy/roland"
	"github.com/spf13/cobra"
)

func repoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage registered repositories",
	}

	cmd.AddCommand(
		repoAddCmd(),
		repoListCmd(),
		repoRemoveCmd(),
		repoSyncCmd(),
		repoPostSetupCmd(),
	)

	return cmd
}

func repoAddCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "add <url>",
		Short: "Clone and register a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := roland.AddRepo(cfg, args[0], name)
			if err != nil {
				return fmt.Errorf("add repo: %w", err)
			}
			fmt.Printf("Added repo %q at %s\n", repo.Name, repo.Path)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Short name for the repo (derived from URL if omitted)")

	return cmd
}

func repoListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos := roland.ListRepos(cfg)
			if len(repos) == 0 {
				fmt.Println("No repos registered. Use 'roland repo add <url>' to add one.")
				return nil
			}
			for _, r := range repos {
				rc := cfg.Repos[r.Name]
				base := ""
				if rc != nil {
					base = rc.BaseBranch
				}
				fmt.Printf("  %-20s %s  (base: %s)\n", r.Name, r.URL, base)
			}
			return nil
		},
	}
}

func repoRemoveCmd() *cobra.Command {
	var deleteFiles bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Unregister a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := roland.RemoveRepo(cfg, args[0], deleteFiles); err != nil {
				return fmt.Errorf("remove repo: %w", err)
			}
			fmt.Printf("Removed repo %q\n", args[0])
			if deleteFiles {
				fmt.Println("Files deleted.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&deleteFiles, "delete-files", false, "Also delete the repo directory on disk")

	return cmd
}

func repoSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [name]",
		Short: "Fetch and fast-forward repos",
		Long:  "Syncs a single repo by name, or all repos if no name is given.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				result, err := roland.SyncRepo(cfg, args[0])
				if err != nil {
					return fmt.Errorf("sync repo: %w", err)
				}
				if result.Updated {
					fmt.Printf("Synced %s (updated)\n", result.Name)
				} else {
					fmt.Printf("Synced %s (up to date)\n", result.Name)
				}
				return nil
			}

			results := roland.SyncAllRepos(cfg)
			for _, r := range results {
				if r.Error != nil {
					fmt.Printf("  %-20s %s\n", r.Name, colorize(colorRed, r.Error.Error()))
				} else if r.Updated {
					fmt.Printf("  %-20s %s\n", r.Name, colorize(colorGreen, "updated"))
				} else {
					fmt.Printf("  %-20s %s\n", r.Name, "up to date")
				}
			}
			return nil
		},
	}
}

func repoPostSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "post-setup <repo-name> <script-path>",
		Short: "Set the post-setup script for a repo",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := roland.SetPostSetup(cfg, args[0], args[1]); err != nil {
				return fmt.Errorf("set post-setup: %w", err)
			}
			fmt.Printf("Post-setup for %q set to %q\n", args[0], args[1])
			return nil
		},
	}
}

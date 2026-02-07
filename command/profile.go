package command

import (
	"fmt"
	"os"
	"sort"

	"github.com/Koutaro-Hanabusa/mangrove"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  "List and inspect profiles defined in the configuration.",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		names := cfg.ProfileNames()
		if len(names) == 0 {
			fmt.Fprintln(os.Stderr, "No profiles defined.")
			return nil
		}

		sort.Strings(names)

		fmt.Fprintln(os.Stderr)
		for _, name := range names {
			marker := " "
			if name == cfg.DefaultProfile {
				marker = "*"
			}
			profile := cfg.Profiles[name]
			repoCount := len(profile.Repos)
			fmt.Fprintf(os.Stderr, " %s %s  (%d repos)\n",
				marker,
				mangrove.ProfileNameStyle.Render(name),
				repoCount,
			)
		}
		fmt.Fprintln(os.Stderr)
		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show <profile-name>",
	Short: "Show profile details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		profile, ok := cfg.Profiles[name]
		if !ok {
			return fmt.Errorf("profile %q not found", name)
		}

		fmt.Fprintf(os.Stderr, "\n%s", mangrove.ProfileNameStyle.Render(name))
		if name == cfg.DefaultProfile {
			fmt.Fprintf(os.Stderr, " %s", mangrove.DimStyle.Render("(default)"))
		}
		fmt.Fprintln(os.Stderr)

		fmt.Fprintf(os.Stderr, "\n  %s\n", mangrove.HeaderStyle.Render("Repositories"))
		for _, repo := range profile.Repos {
			defaultBase := repo.GetDefaultBase()
			fmt.Fprintf(os.Stderr, "    %s\n", mangrove.RepoNameStyle.Render(repo.Name))
			fmt.Fprintf(os.Stderr, "      path:         %s\n", repo.Path)
			fmt.Fprintf(os.Stderr, "      default_base: %s\n", mangrove.BranchNameStyle.Render(defaultBase))
		}

		if len(profile.Hooks.PostCreate) > 0 {
			fmt.Fprintf(os.Stderr, "\n  %s\n", mangrove.HeaderStyle.Render("Hooks (post_create)"))
			for _, hook := range profile.Hooks.PostCreate {
				fmt.Fprintf(os.Stderr, "    %s: %s\n",
					mangrove.RepoNameStyle.Render(hook.Repo),
					mangrove.DimStyle.Render(hook.Run),
				)
			}
		}

		fmt.Fprintln(os.Stderr)
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	rootCmd.AddCommand(profileCmd)
}

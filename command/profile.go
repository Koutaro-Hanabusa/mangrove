package command

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

var profileAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		// Prompt for profile name
		var profileName string
		for profileName == "" {
			profileName = promptInput(reader, "Profile name", "")
			if profileName == "" {
				fmt.Fprintln(os.Stderr, "  Profile name is required.")
			}
		}

		// Check if already exists
		if _, ok := cfg.Profiles[profileName]; ok {
			return fmt.Errorf("profile %q already exists", profileName)
		}

		if !mangrove.IsFzfAvailable() {
			return fmt.Errorf("fzf is required for repository selection. Install it with: brew install fzf")
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		// Loop to add repos
		var repos []mangrove.Repo
		for {
			fmt.Fprintln(os.Stderr, "? Select repository directory (Esc to finish):")
			repoPath, err := mangrove.SelectDirectory("Repository path:", home)
			if err != nil {
				if strings.Contains(err.Error(), "cancelled") {
					if len(repos) == 0 {
						fmt.Fprintln(os.Stderr, "  At least one repository is required.")
						continue
					}
					break
				}
				return fmt.Errorf("directory selection failed: %w", err)
			}

			expandedPath := mangrove.ExpandPath(repoPath)
			if !isGitRepo(expandedPath) {
				fmt.Fprintf(os.Stderr, "  %s is not a valid git repository.\n", expandedPath)
				continue
			}

			repoName := filepath.Base(expandedPath)
			detectedBranch := mangrove.DetectDefaultBranch(expandedPath)
			fmt.Fprintf(os.Stderr, "  -> Detected: %s (branch: %s)\n", repoName, detectedBranch)

			defaultBase := promptInput(reader, "Default base branch", detectedBranch)
			if defaultBase == "" {
				defaultBase = detectedBranch
			}

			repos = append(repos, mangrove.Repo{
				Name:        repoName,
				Path:        expandedPath,
				DefaultBase: defaultBase,
			})

			fmt.Fprint(os.Stderr, "? Add another repository? (Y/n): ")
			if !promptYesNo(reader, true) {
				break
			}
		}

		if len(repos) == 0 {
			return fmt.Errorf("no repositories added, aborting")
		}

		profile := mangrove.Profile{Repos: repos}
		if err := cfg.AddProfile(profileName, profile); err != nil {
			return err
		}

		// Ask to set as default
		fmt.Fprint(os.Stderr, "? Set as default profile? (y/N): ")
		if promptYesNo(reader, false) {
			cfg.DefaultProfile = profileName
		}

		if err := mangrove.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		mangrove.PrintSuccess("Profile %q added with %d repos", profileName, len(repos))
		return nil
	},
}

var profileAddRepoCmd = &cobra.Command{
	Use:   "add-repo [profile-name]",
	Short: "Add a repository to a profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		var profileName string
		if len(args) > 0 {
			profileName = args[0]
		} else {
			_, name, err := resolveProfile(true)
			if err != nil {
				return err
			}
			profileName = name
		}

		// Verify profile exists
		if _, ok := cfg.Profiles[profileName]; !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}

		if !mangrove.IsFzfAvailable() {
			return fmt.Errorf("fzf is required. Install it with: brew install fzf")
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		fmt.Fprintln(os.Stderr, "? Select repository directory:")
		repoPath, err := mangrove.SelectDirectory("Repository path:", home)
		if err != nil {
			return fmt.Errorf("directory selection failed: %w", err)
		}

		expandedPath := mangrove.ExpandPath(repoPath)
		if !isGitRepo(expandedPath) {
			return fmt.Errorf("%s is not a valid git repository", expandedPath)
		}

		repoName := filepath.Base(expandedPath)
		detectedBranch := mangrove.DetectDefaultBranch(expandedPath)
		fmt.Fprintf(os.Stderr, "  -> Detected: %s (branch: %s)\n", repoName, detectedBranch)

		defaultBase := promptInput(reader, "Default base branch", detectedBranch)
		if defaultBase == "" {
			defaultBase = detectedBranch
		}

		repo := mangrove.Repo{
			Name:        repoName,
			Path:        expandedPath,
			DefaultBase: defaultBase,
		}

		if err := cfg.AddRepoToProfile(profileName, repo); err != nil {
			return err
		}

		if err := mangrove.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		mangrove.PrintSuccess("Added %s to profile %q", repoName, profileName)
		return nil
	},
}

var profileRemoveRepoCmd = &cobra.Command{
	Use:   "remove-repo [profile-name] [repo-name]",
	Short: "Remove a repository from a profile",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var profileName, repoName string

		if len(args) == 2 {
			profileName = args[0]
			repoName = args[1]
		} else if len(args) == 1 {
			if profileFlag != "" {
				profileName = profileFlag
				repoName = args[0]
			} else {
				_, name, err := resolveProfile(true)
				if err != nil {
					return err
				}
				profileName = name
				repoName = args[0]
			}
		}

		if err := cfg.RemoveRepoFromProfile(profileName, repoName); err != nil {
			return err
		}

		if err := mangrove.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		mangrove.PrintSuccess("Removed %s from profile %q", repoName, profileName)
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileAddRepoCmd)
	profileCmd.AddCommand(profileRemoveRepoCmd)
	rootCmd.AddCommand(profileCmd)
}

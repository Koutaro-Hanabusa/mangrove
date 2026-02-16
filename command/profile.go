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
	Long:  "Interactively create a new profile with repositories.",
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

		// Check if profile already exists
		if _, exists := cfg.Profiles[profileName]; exists {
			return fmt.Errorf("profile %q already exists", profileName)
		}

		// Check fzf availability
		if !mangrove.IsFzfAvailable() {
			return fmt.Errorf("fzf is required for repository selection. Install it with: brew install fzf")
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		// Loop to add repositories
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

			if !isGitRepoRoot(expandedPath) {
				fmt.Fprintf(os.Stderr, "  %s is not a git repository root.\n", expandedPath)
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

		profile := mangrove.Profile{
			Repos: repos,
		}

		if err := cfg.AddProfile(profileName, profile); err != nil {
			return err
		}

		// Ask to set as default profile
		if cfg.DefaultProfile == "" {
			fmt.Fprint(os.Stderr, "? Set as default profile? (Y/n): ")
			if promptYesNo(reader, true) {
				cfg.DefaultProfile = profileName
			}
		}

		if err := mangrove.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		mangrove.PrintSuccess("Added profile %q with %d repo(s)", profileName, len(repos))
		return nil
	},
}

var profileAddRepoCmd = &cobra.Command{
	Use:   "add-repo [profile-name]",
	Short: "Add a repository to an existing profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		// Resolve profile name
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

		if _, ok := cfg.Profiles[profileName]; !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}

		// Check fzf availability
		if !mangrove.IsFzfAvailable() {
			return fmt.Errorf("fzf is required for repository selection. Install it with: brew install fzf")
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		// Select repository directory
		fmt.Fprintln(os.Stderr, "? Select repository directory:")
		repoPath, err := mangrove.SelectDirectory("Repository path:", home)
		if err != nil {
			return fmt.Errorf("directory selection failed: %w", err)
		}

		expandedPath := mangrove.ExpandPath(repoPath)

		if !isGitRepoRoot(expandedPath) {
			return fmt.Errorf("%s is not a git repository root", expandedPath)
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

		mangrove.PrintSuccess("Added repository %q to profile %q", repoName, profileName)
		return nil
	},
}

var profileRemoveRepoCmd = &cobra.Command{
	Use:   "remove-repo [profile-name] [repo-name]",
	Short: "Remove a repository from a profile",
	Args:  cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve profile name
		var profileName string
		if len(args) >= 1 {
			profileName = args[0]
		} else {
			_, name, err := resolveProfile(true)
			if err != nil {
				return err
			}
			profileName = name
		}

		profile, ok := cfg.Profiles[profileName]
		if !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}

		// Resolve repo name
		var repoName string
		if len(args) >= 2 {
			repoName = args[1]
		} else {
			// Let user select from existing repos
			repoNames := make([]string, len(profile.Repos))
			for i, r := range profile.Repos {
				repoNames[i] = r.Name
			}
			if len(repoNames) == 0 {
				return fmt.Errorf("profile %q has no repositories", profileName)
			}
			selected, err := mangrove.SelectWithFzf(repoNames, "Remove repo:", "Select repository to remove")
			if err != nil {
				return err
			}
			repoName = selected
		}

		if err := cfg.RemoveRepoFromProfile(profileName, repoName); err != nil {
			return err
		}

		if err := mangrove.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		mangrove.PrintSuccess("Removed repository %q from profile %q", repoName, profileName)
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

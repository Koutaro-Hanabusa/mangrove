package command

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Koutaro-Hanabusa/mangrove"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new mgv configuration",
	Long:  "Interactively create a new ~/.config/mgv/config.yaml configuration file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		// Check if config already exists
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		configPath := filepath.Join(home, ".config", "mgv", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Fprintf(os.Stderr, "? Config already exists at %s. Overwrite? (y/N): ", configPath)
			if !promptYesNo(reader, false) {
				fmt.Fprintln(os.Stderr, "  Aborted.")
				return nil
			}
		}

		// Prompt for base_dir
		baseDir := promptInput(reader, "Base directory", "~/mgv-workspaces")
		if baseDir == "" {
			baseDir = "~/mgv-workspaces"
		}

		// Prompt for profile name (required)
		var profileName string
		for profileName == "" {
			profileName = promptInput(reader, "Profile name", "")
			if profileName == "" {
				fmt.Fprintln(os.Stderr, "  Profile name is required.")
			}
		}

		// Check fzf availability for directory selection
		if !mangrove.IsFzfAvailable() {
			return fmt.Errorf("fzf is required for repository selection. Install it with: brew install fzf")
		}

		// Loop to add repositories
		var repos []mangrove.Repo
		for {
			fmt.Fprintln(os.Stderr, "? Select repository directory (Esc to finish):")
			repoPath, err := mangrove.SelectGitRepository("Repository path:", home)
			if err != nil {
				// User cancelled with Esc
				if strings.Contains(err.Error(), "cancelled") {
					break
				}
				return fmt.Errorf("directory selection failed: %w", err)
			}

			// Expand path for validation
			expandedPath := mangrove.ExpandPath(repoPath)

			// Validate it's a git repo
			if !isGitRepoRoot(expandedPath) {
				fmt.Fprintf(os.Stderr, "  %s is not a git repository root.\n", expandedPath)
				continue
			}

			// Auto-detect repo name from directory name
			repoName := filepath.Base(expandedPath)

			// Auto-detect default branch
			detectedBranch := mangrove.DetectDefaultBranch(expandedPath)

			// Show detected info
			fmt.Fprintf(os.Stderr, "  -> Detected: %s (branch: %s)\n", repoName, detectedBranch)

			// Prompt for default base branch
			defaultBase := promptInput(reader, "Default base branch", detectedBranch)
			if defaultBase == "" {
				defaultBase = detectedBranch
			}

			repos = append(repos, mangrove.Repo{
				Name:        repoName,
				Path:        expandedPath,
				DefaultBase: defaultBase,
			})

			// Ask to add another
			fmt.Fprint(os.Stderr, "? Add another repository? (Y/n): ")
			if !promptYesNo(reader, true) {
				break
			}
		}

		if len(repos) == 0 {
			return fmt.Errorf("no repositories added, aborting")
		}

		// Ask to set as default profile
		fmt.Fprint(os.Stderr, "? Set as default profile? (Y/n): ")
		isDefault := promptYesNo(reader, true)

		// Build Config
		newCfg := &mangrove.Config{
			BaseDir:  mangrove.ExpandPath(baseDir),
			Profiles: map[string]mangrove.Profile{},
		}
		if isDefault {
			newCfg.DefaultProfile = profileName
		}
		newCfg.Profiles[profileName] = mangrove.Profile{
			Repos: repos,
		}

		// Save config
		if err := mangrove.SaveConfig(newCfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		mangrove.PrintSuccess("Created %s", configPath)
		return nil
	},
}

// promptInput prints a prompt and reads a line of input.
// If defaultVal is non-empty, it is shown in parentheses and used when input is empty.
func promptInput(reader *bufio.Reader, prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Fprintf(os.Stderr, "? %s (%s): ", prompt, defaultVal)
	} else {
		fmt.Fprintf(os.Stderr, "? %s: ", prompt)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// promptYesNo reads a yes/no response from the reader.
// The prompt should already be printed before calling this function.
// defaultYes determines the behavior when the user presses Enter with no input.
func promptYesNo(reader *bufio.Reader, defaultYes bool) bool {
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return defaultYes
	}
}

// isGitRepoRoot checks if the given path is the root of a git repository.
func isGitRepoRoot(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	toplevel := strings.TrimSpace(string(out))
	return toplevel == path
}

func init() {
	rootCmd.AddCommand(initCmd)
}

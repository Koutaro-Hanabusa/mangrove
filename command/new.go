package command

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/1126buri/mangrove"
	"github.com/spf13/cobra"
)

var (
	newYes  bool
	newBase string
)

var newCmd = &cobra.Command{
	Use:   "new [workspace-name]",
	Short: "Create a new workspace",
	Long: `Create a new workspace with worktrees for all repos in the selected profile.

Interactive mode: prompts for profile, workspace name, and base branch for each repo.
Non-interactive mode (--yes): uses default_profile and default_base for each repo.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive := !newYes

		// Resolve profile
		profile, profileName, err := resolveProfile(interactive)
		if err != nil {
			return err
		}

		// Get workspace name
		var wsName string
		if len(args) > 0 {
			wsName = args[0]
		} else if interactive {
			fmt.Fprint(os.Stderr, "? Workspace name: ")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read workspace name: %w", err)
			}
			wsName = strings.TrimSpace(input)
		}

		if wsName == "" {
			return fmt.Errorf("workspace name is required")
		}

		// Determine base branches for each repo
		baseBranches := make(map[string]string)

		if interactive {
			// Check fzf availability
			if !mangrove.IsFzfAvailable() {
				return fmt.Errorf("fzf is required for interactive mode. Install with: brew install fzf\nOr use --yes flag for non-interactive mode")
			}

			for _, repo := range profile.Repos {
				prompt := fmt.Sprintf("[%s] Base branch:", repo.Name)
				branch, err := mangrove.SelectBranch(repo.Path, prompt, repo.GetDefaultBase())
				if err != nil {
					return fmt.Errorf("branch selection for %s failed: %w", repo.Name, err)
				}
				baseBranches[repo.Name] = branch
			}
		} else {
			// Non-interactive: use --base or default_base for each repo
			for _, repo := range profile.Repos {
				if newBase != "" {
					baseBranches[repo.Name] = newBase
				} else {
					baseBranches[repo.Name] = repo.GetDefaultBase()
				}
			}
		}

		return mangrove.CreateWorkspace(cfg, profile, profileName, wsName, baseBranches)
	},
}

func init() {
	newCmd.Flags().BoolVarP(&newYes, "yes", "y", false, "non-interactive mode (use defaults)")
	newCmd.Flags().StringVarP(&newBase, "base", "b", "", "common base branch for all repos")
	rootCmd.AddCommand(newCmd)
}

package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/1126buri/mangrove"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [workspace-name]",
	Short: "Show detailed git status for a workspace",
	Long: `Show detailed git status for each repo in a workspace.

Displays branch name, clean/changed status, and ahead/behind counts.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var profileName, wsName string

		if len(args) > 0 {
			wsName = args[0]
			_, pName, err := resolveProfile(profileFlag == "")
			if err != nil {
				return err
			}
			profileName = pName
		} else {
			// Interactive workspace selection
			if !mangrove.IsFzfAvailable() {
				return fmt.Errorf("fzf is required for interactive mode. Install with: brew install fzf")
			}

			workspaces, err := mangrove.ListWorkspaces(cfg, profileFlag)
			if err != nil {
				return err
			}
			if len(workspaces) == 0 {
				return fmt.Errorf("no workspaces found")
			}

			labels := mangrove.WorkspaceLabels(workspaces)
			selected, err := mangrove.SelectWorkspace(labels)
			if err != nil {
				return err
			}

			pName, wName, err := mangrove.ParseWorkspaceLabel(selected)
			if err != nil {
				return err
			}
			profileName = pName
			wsName = wName
		}

		profile, _, err := cfg.GetProfile(profileName)
		if err != nil {
			return err
		}

		wsPath := mangrove.GetWorkspacePath(cfg, profileName, wsName)

		fmt.Fprintf(os.Stderr, "\n%s/%s:\n",
			mangrove.ProfileNameStyle.Render(profileName),
			mangrove.RepoNameStyle.Render(wsName),
		)

		for _, repo := range profile.Repos {
			repoDir := filepath.Join(wsPath, repo.Name)

			if _, err := os.Stat(repoDir); os.IsNotExist(err) {
				mangrove.PrintWarning("%s: worktree not found", repo.Name)
				continue
			}

			branch, err := mangrove.CurrentBranch(repoDir)
			if err != nil {
				mangrove.PrintError("%s: failed to get branch: %v", repo.Name, err)
				continue
			}

			changedCount, err := mangrove.StatusChangedCount(repoDir)
			if err != nil {
				mangrove.PrintError("%s: failed to get status: %v", repo.Name, err)
				continue
			}

			ahead, behind, err := mangrove.AheadBehind(repo.Path, repo.GetDefaultBase(), branch)
			if err != nil {
				// Non-fatal: ahead/behind may not be available
				ahead, behind = 0, 0
			}

			mangrove.PrintRepoStatus(repo.Name, branch, changedCount, ahead, behind, repo.GetDefaultBase())
		}

		fmt.Fprintln(os.Stderr)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

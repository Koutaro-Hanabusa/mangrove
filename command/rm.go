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
	rmYes        bool
	rmWithBranch bool
	rmForce      bool
)

var rmCmd = &cobra.Command{
	Use:   "rm [workspace-name]",
	Short: "Remove a workspace",
	Long: `Remove a workspace and its worktrees.

Interactive mode: presents a list of workspaces to choose from.
Use --with-branch to also delete the local branches.
Use --force to remove workspaces with uncommitted changes.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive := !rmYes

		var profileName, wsName string

		if len(args) > 0 {
			wsName = args[0]
			// Resolve profile
			_, pName, err := resolveProfile(interactive)
			if err != nil {
				return err
			}
			profileName = pName
		} else if interactive {
			// Interactive workspace selection
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
		} else {
			return fmt.Errorf("workspace name is required in non-interactive mode")
		}

		profile, profileName, err := cfg.GetProfile(profileName)
		if err != nil {
			return err
		}

		// Check for uncommitted changes and warn
		if !rmForce && !rmYes {
			wsPath := mangrove.GetWorkspacePath(cfg, profileName, wsName)
			for _, repo := range profile.Repos {
				repoDir := wsPath + "/" + repo.Name
				if _, err := os.Stat(repoDir); os.IsNotExist(err) {
					continue
				}
				count, err := mangrove.StatusChangedCount(repoDir)
				if err != nil {
					continue
				}
				if count > 0 {
					mangrove.PrintWarning("%s has uncommitted changes (%d files)", repo.Name, count)
					fmt.Fprint(os.Stderr, "? Force remove anyway? (y/N): ")
					reader := bufio.NewReader(os.Stdin)
					input, _ := reader.ReadString('\n')
					if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
						return fmt.Errorf("aborted")
					}
					rmForce = true
					break
				}
			}
		}

		// Ask about branch deletion in interactive mode
		if interactive && !rmWithBranch {
			fmt.Fprint(os.Stderr, "? Also delete local branches? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
				rmWithBranch = true
			}
		}

		return mangrove.RemoveWorkspace(cfg, profile, profileName, wsName, rmWithBranch, rmForce)
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&rmYes, "yes", "y", false, "non-interactive mode (skip confirmations)")
	rmCmd.Flags().BoolVar(&rmWithBranch, "with-branch", false, "also delete local branches")
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "force remove even with uncommitted changes")
	rootCmd.AddCommand(rmCmd)
}

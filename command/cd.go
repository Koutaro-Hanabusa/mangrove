package command

import (
	"fmt"

	"github.com/Koutaro-Hanabusa/mangrove"
	"github.com/spf13/cobra"
)

var cdCmd = &cobra.Command{
	Use:   "cd [workspace-name]",
	Short: "Output workspace path for cd",
	Long: `Output the path of a workspace to stdout for use with cd.

Usage: cd $(mgv cd)
       cd $(mgv cd feature-login)
       cd $(mgv cd feature-login --profile project-a)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var profileName, wsName string

		if len(args) > 0 {
			// Direct workspace name specified
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

		path := mangrove.GetWorkspacePath(cfg, profileName, wsName)

		// Output path to stdout (not stderr) so cd $(mgv cd) works
		fmt.Println(path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cdCmd)
}

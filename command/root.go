package command

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/Koutaro-Hanabusa/mangrove"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via ldflags. Falls back to debug.BuildInfo or "dev".
	Version = "dev"

	// profileFlag is the global --profile flag.
	profileFlag string

	// cfg holds the loaded configuration.
	cfg *mangrove.Config
)

func resolveVersion() string {
	// If ldflags set a non-default version, use it.
	if Version != "dev" {
		return Version
	}
	// Try to get version from module info (populated by go install).
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:     "mgv",
	Short:   "mangrove - Multi-repo worktree manager",
	Long:    "mangrove (mgv) manages workspaces across multiple git repositories using git worktree.",
	Version: resolveVersion(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for completion commands
		if cmd.Name() == "completion" || cmd.Name() == "help" || cmd.Name() == "init" {
			return nil
		}

		var err error
		cfg, err = mangrove.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "profile name (overrides default_profile)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, mangrove.ErrCancelled) {
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// resolveProfile resolves the profile from the flag or config default.
func resolveProfile(interactive bool) (*mangrove.Profile, string, error) {
	if profileFlag != "" {
		return cfg.GetProfile(profileFlag)
	}

	if interactive && profileFlag == "" && cfg.DefaultProfile == "" {
		names := cfg.ProfileNames()
		if len(names) == 0 {
			return nil, "", fmt.Errorf("no profiles defined in config")
		}
		if len(names) == 1 {
			return cfg.GetProfile(names[0])
		}
		selected, err := mangrove.SelectProfile(names)
		if err != nil {
			return nil, "", err
		}
		return cfg.GetProfile(selected)
	}

	return cfg.GetProfile(profileFlag)
}

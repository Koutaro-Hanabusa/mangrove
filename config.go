package mangrove

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Hook represents a post-create hook to run after workspace creation.
type Hook struct {
	Repo string `mapstructure:"repo" yaml:"repo"`
	Run  string `mapstructure:"run"  yaml:"run"`
}

// Hooks holds the different hook stages.
type Hooks struct {
	PostCreate []Hook `mapstructure:"post_create" yaml:"post_create"`
}

// Repo represents a single git repository within a profile.
type Repo struct {
	Name        string `mapstructure:"name"         yaml:"name"`
	Path        string `mapstructure:"path"         yaml:"path"`
	DefaultBase string `mapstructure:"default_base" yaml:"default_base"`
}

// Profile represents a named collection of repositories and their hooks.
type Profile struct {
	Repos []Repo `mapstructure:"repos" yaml:"repos"`
	Hooks Hooks  `mapstructure:"hooks" yaml:"hooks"`
}

// Config is the top-level configuration structure.
type Config struct {
	BaseDir        string             `mapstructure:"base_dir"        yaml:"base_dir"`
	DefaultProfile string             `mapstructure:"default_profile" yaml:"default_profile"`
	Profiles       map[string]Profile `mapstructure:"profiles"        yaml:"profiles"`
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// CollapsePath replaces the home directory prefix with ~/ for portable storage.
func CollapsePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home+"/") {
		return "~/" + path[len(home)+1:]
	}
	if path == home {
		return "~"
	}
	return path
}

// SaveConfig writes the Config struct to ~/.config/mgv/config.yaml.
func SaveConfig(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "mgv")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Collapse paths to ~/ form for portable storage
	saveCfg := Config{
		BaseDir:        CollapsePath(cfg.BaseDir),
		DefaultProfile: cfg.DefaultProfile,
		Profiles:       make(map[string]Profile, len(cfg.Profiles)),
	}
	for profileName, profile := range cfg.Profiles {
		repos := make([]Repo, len(profile.Repos))
		for i, repo := range profile.Repos {
			repos[i] = Repo{
				Name:        repo.Name,
				Path:        CollapsePath(repo.Path),
				DefaultBase: repo.DefaultBase,
			}
		}
		saveCfg.Profiles[profileName] = Profile{
			Repos: repos,
			Hooks: profile.Hooks,
		}
	}

	data, err := yaml.Marshal(&saveCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DetectDefaultBranch detects the default branch of a remote repository.
// Falls back to "main" on error.
func DetectDefaultBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "main"
	}

	// Parse "refs/remotes/origin/main" -> "main"
	ref := strings.TrimSpace(string(output))
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "main"
}

// LoadConfig reads the configuration from ~/.config/mgv/config.yaml.
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "mgv")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Set defaults
	viper.SetDefault("base_dir", "~/mgv-workspaces")
	viper.SetDefault("default_profile", "")
	viper.SetDefault("profiles", map[string]Profile{})

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found at %s/config.yaml: %w", configDir, err)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand paths
	cfg.BaseDir = ExpandPath(cfg.BaseDir)
	for profileName, profile := range cfg.Profiles {
		for i := range profile.Repos {
			profile.Repos[i].Path = ExpandPath(profile.Repos[i].Path)
		}
		cfg.Profiles[profileName] = profile
	}

	return &cfg, nil
}

// GetProfile returns the named profile from the config.
// If name is empty, returns the default profile.
func (c *Config) GetProfile(name string) (*Profile, string, error) {
	if name == "" {
		name = c.DefaultProfile
	}
	if name == "" {
		return nil, "", fmt.Errorf("no profile specified and no default_profile set in config")
	}
	profile, ok := c.Profiles[name]
	if !ok {
		return nil, "", fmt.Errorf("profile %q not found in config", name)
	}
	return &profile, name, nil
}

// ProfileNames returns a sorted list of profile names.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

// AddProfile adds a new profile to the config.
// Returns an error if a profile with the same name already exists.
func (c *Config) AddProfile(name string, profile Profile) error {
	if _, ok := c.Profiles[name]; ok {
		return fmt.Errorf("profile %q already exists", name)
	}
	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}
	c.Profiles[name] = profile
	return nil
}

// AddRepoToProfile adds a repository to an existing profile.
// Returns an error if the profile does not exist or a repo with the same name already exists.
func (c *Config) AddRepoToProfile(profileName string, repo Repo) error {
	profile, ok := c.Profiles[profileName]
	if !ok {
		return fmt.Errorf("profile %q not found", profileName)
	}
	for _, r := range profile.Repos {
		if r.Name == repo.Name {
			return fmt.Errorf("repository %q already exists in profile %q", repo.Name, profileName)
		}
	}
	profile.Repos = append(profile.Repos, repo)
	c.Profiles[profileName] = profile
	return nil
}

// RemoveRepoFromProfile removes a repository from an existing profile.
// Returns an error if the profile or repository does not exist.
func (c *Config) RemoveRepoFromProfile(profileName, repoName string) error {
	profile, ok := c.Profiles[profileName]
	if !ok {
		return fmt.Errorf("profile %q not found", profileName)
	}
	idx := -1
	for i, r := range profile.Repos {
		if r.Name == repoName {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("repository %q not found in profile %q", repoName, profileName)
	}
	profile.Repos = append(profile.Repos[:idx], profile.Repos[idx+1:]...)
	c.Profiles[profileName] = profile
	return nil
}

// GetRepoDefaultBase returns the default base branch for a repo,
// falling back to "main" if not set.
func (r *Repo) GetDefaultBase() string {
	if r.DefaultBase != "" {
		return r.DefaultBase
	}
	return "main"
}

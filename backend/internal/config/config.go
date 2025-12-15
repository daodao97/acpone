package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed acpone.config.example.json
var exampleConfigData []byte

// WorkspaceConfig defines a workspace
type WorkspaceConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// AgentConfig defines an ACP agent
type AgentConfig struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Command        string            `json:"command"`
	Args           []string          `json:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Prestart       bool              `json:"prestart,omitempty"`
	PermissionMode string            `json:"permissionMode,omitempty"`
}

// RoutingConfig defines routing rules
type RoutingConfig struct {
	Keywords map[string]string `json:"keywords,omitempty"`
	Meta     bool              `json:"meta,omitempty"`
}

// Config is the main acpone configuration
type Config struct {
	Agents           []AgentConfig     `json:"agents"`
	DefaultAgent     string            `json:"defaultAgent"`
	Routing          *RoutingConfig    `json:"routing,omitempty"`
	Workspaces       []WorkspaceConfig `json:"workspaces,omitempty"`
	DefaultWorkspace string            `json:"defaultWorkspace,omitempty"`
}

// rawConfig supports legacy field names
type rawConfig struct {
	Agents           []AgentConfig     `json:"agents,omitempty"`
	Backends         []AgentConfig     `json:"backends,omitempty"`
	DefaultAgent     string            `json:"defaultAgent,omitempty"`
	DefaultBackend   string            `json:"defaultBackend,omitempty"`
	Routing          *RoutingConfig    `json:"routing,omitempty"`
	Workspaces       []WorkspaceConfig `json:"workspaces,omitempty"`
	DefaultWorkspace string            `json:"defaultWorkspace,omitempty"`
}

func (r *rawConfig) normalize() *Config {
	agents := r.Agents
	if len(agents) == 0 {
		agents = r.Backends
	}
	defaultAgent := r.DefaultAgent
	if defaultAgent == "" {
		defaultAgent = r.DefaultBackend
	}
	return &Config{
		Agents:           agents,
		DefaultAgent:     defaultAgent,
		Routing:          r.Routing,
		Workspaces:       r.Workspaces,
		DefaultWorkspace: r.DefaultWorkspace,
	}
}

// LoadedConfigPath stores the path of loaded config file
var LoadedConfigPath string

// Load loads configuration from file or defaults
func Load(configPath string) (*Config, error) {
	if configPath != "" {
		fullPath, err := filepath.Abs(configPath)
		if err != nil {
			return nil, err
		}
		LoadedConfigPath = fullPath
		return loadFromFile(fullPath)
	}

	// Try default locations
	paths := defaultPaths()
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			LoadedConfigPath = absPath
			return loadFromFile(p)
		}
	}

	LoadedConfigPath = ""
	return DefaultConfig(), nil
}

func loadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return raw.normalize(), nil
}

func defaultPaths() []string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	return []string{
		"./acpone.config.json",
		"./acpone.json",
		filepath.Join(home, ".acpone", "acpone.config.json"),
		filepath.Join(home, ".config", "acpone", "config.json"),
	}
}

// userConfigPath returns the user config path ~/.acpone/acpone.config.json
func userConfigPath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	return filepath.Join(home, ".acpone", "acpone.config.json")
}

// EnsureConfigExists creates config from example if it doesn't exist
func EnsureConfigExists() error {
	configPath := userConfigPath()

	// Check if any config file exists
	for _, p := range defaultPaths() {
		if _, err := os.Stat(p); err == nil {
			return nil // Config exists, nothing to do
		}
	}

	// No config found, create from example
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, exampleConfigData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("📝 Created config file: %s\n", configPath)
	return nil
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	cwd, _ := os.Getwd()
	return &Config{
		Agents: []AgentConfig{
			{
				ID:      "claude",
				Name:    "Claude Code",
				Command: "npx",
				Args:    []string{"@anthropics/claude-code", "--acp"},
			},
		},
		DefaultAgent: "claude",
		Routing:      &RoutingConfig{Meta: true},
		Workspaces: []WorkspaceConfig{
			{ID: "default", Name: "Default", Path: cwd},
		},
		DefaultWorkspace: "default",
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Agents) == 0 {
		return errors.New("at least one agent must be configured")
	}

	ids := make(map[string]bool)
	for _, agent := range c.Agents {
		if agent.ID == "" || agent.Command == "" {
			return errors.New("agent must have id and command")
		}
		if ids[agent.ID] {
			return fmt.Errorf("duplicate agent id: %s", agent.ID)
		}
		ids[agent.ID] = true
	}

	if !ids[c.DefaultAgent] {
		return fmt.Errorf("default agent not found: %s", c.DefaultAgent)
	}

	return nil
}

// FindAgent returns agent config by ID
func (c *Config) FindAgent(id string) *AgentConfig {
	for i := range c.Agents {
		if c.Agents[i].ID == id {
			return &c.Agents[i]
		}
	}
	return nil
}

// FindWorkspace returns workspace config by ID
func (c *Config) FindWorkspace(id string) *WorkspaceConfig {
	for i := range c.Workspaces {
		if c.Workspaces[i].ID == id {
			return &c.Workspaces[i]
		}
	}
	return nil
}

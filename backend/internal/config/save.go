package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// FindConfigPath finds existing config file path
func FindConfigPath() string {
	for _, p := range defaultPaths() {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

// Save saves configuration to file
func (c *Config) Save(configPath string) error {
	targetPath := configPath
	if targetPath == "" {
		targetPath = FindConfigPath()
	}
	if targetPath == "" {
		return errors.New("no config file found to save to")
	}

	// Read existing to preserve extra fields
	existing := make(map[string]any)
	if data, err := os.ReadFile(targetPath); err == nil {
		json.Unmarshal(data, &existing)
	}

	// Merge agents
	mergedAgents := c.mergeAgents(existing)

	output := map[string]any{
		"agents":       mergedAgents,
		"defaultAgent": c.DefaultAgent,
	}
	if c.Routing != nil {
		output["routing"] = c.Routing
	}
	if len(c.Workspaces) > 0 {
		output["workspaces"] = c.Workspaces
	}
	if c.DefaultWorkspace != "" {
		output["defaultWorkspace"] = c.DefaultWorkspace
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(targetPath, append(data, '\n'), 0644)
}

func (c *Config) mergeAgents(existing map[string]any) []map[string]any {
	existingAgents := make(map[string]map[string]any)
	if agents, ok := existing["agents"].([]any); ok {
		for _, a := range agents {
			if agent, ok := a.(map[string]any); ok {
				if id, ok := agent["id"].(string); ok {
					existingAgents[id] = agent
				}
			}
		}
	}

	result := make([]map[string]any, 0, len(c.Agents))
	for _, agent := range c.Agents {
		merged := make(map[string]any)

		// Copy existing fields first
		if ea, ok := existingAgents[agent.ID]; ok {
			for k, v := range ea {
				merged[k] = v
			}
		}

		// Override with new values
		merged["id"] = agent.ID
		merged["name"] = agent.Name
		merged["command"] = agent.Command
		if len(agent.Args) > 0 {
			merged["args"] = agent.Args
		}
		if len(agent.Env) > 0 {
			merged["env"] = agent.Env
		}
		if agent.PermissionMode != "" {
			merged["permissionMode"] = agent.PermissionMode
		}

		result = append(result, merged)
	}

	return result
}

// AddWorkspace adds a workspace to config
func (c *Config) AddWorkspace(ws WorkspaceConfig) {
	c.Workspaces = append(c.Workspaces, ws)
}

package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/daodao97/acpone/internal/config"
)

// WorkspaceStore manages workspace persistence
type WorkspaceStore struct {
	filePath string
}

// NewWorkspaceStore creates a new workspace store
func NewWorkspaceStore(filePath string) *WorkspaceStore {
	if filePath == "" {
		filePath = defaultWorkspacePath()
	}
	dir := filepath.Dir(filePath)
	os.MkdirAll(dir, 0755)
	return &WorkspaceStore{filePath: filePath}
}

func defaultWorkspacePath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".acpone", "workspaces.json")
}

// workspaceFile matches TypeScript format: {"workspaces": [...]}
type workspaceFile struct {
	Workspaces []config.WorkspaceConfig `json:"workspaces"`
}

// Load loads all workspaces
func (s *WorkspaceStore) Load() []config.WorkspaceConfig {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil
	}

	var file workspaceFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil
	}

	return file.Workspaces
}

// Save saves all workspaces
func (s *WorkspaceStore) Save(workspaces []config.WorkspaceConfig) error {
	file := workspaceFile{Workspaces: workspaces}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

// Add adds a workspace
func (s *WorkspaceStore) Add(ws config.WorkspaceConfig) error {
	workspaces := s.Load()
	workspaces = append(workspaces, ws)
	return s.Save(workspaces)
}

// Remove removes a workspace by ID
func (s *WorkspaceStore) Remove(id string) error {
	workspaces := s.Load()
	filtered := make([]config.WorkspaceConfig, 0, len(workspaces))
	for _, ws := range workspaces {
		if ws.ID != id {
			filtered = append(filtered, ws)
		}
	}
	return s.Save(filtered)
}

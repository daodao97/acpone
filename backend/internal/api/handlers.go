package api

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/anthropics/acpone/internal/config"
)

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	s.agentCommandsMu.RLock()
	defer s.agentCommandsMu.RUnlock()

	agents := make([]map[string]any, 0, len(s.config.Agents))
	for _, a := range s.config.Agents {
		agentData := map[string]any{
			"id":             a.ID,
			"name":           a.Name,
			"permissionMode": a.PermissionMode,
			"command":        a.Command,
			"args":           a.Args,
			"env":            a.Env,
		}
		// Include cached commands if available
		if cmds, ok := s.agentCommands[a.ID]; ok {
			agentData["commands"] = cmds
		}
		agents = append(agents, agentData)
	}

	writeJSON(w, map[string]any{
		"agents":  agents,
		"default": s.config.DefaultAgent,
	})
}

func (s *Server) handleAgentUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		AgentID        string            `json:"agentId"`
		PermissionMode string            `json:"permissionMode,omitempty"`
		Env            map[string]string `json:"env,omitempty"`
		UpdateEnv      bool              `json:"updateEnv,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	agent := s.config.FindAgent(data.AgentID)
	if agent == nil {
		writeError(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Update permission mode if provided
	if data.PermissionMode != "" {
		agent.PermissionMode = data.PermissionMode
	}

	// Update env if requested
	if data.UpdateEnv {
		agent.Env = data.Env
	}

	if err := s.config.Save(""); err != nil {
		writeError(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	// Stop the agent process so it will be recreated with new config on next request
	_ = s.agents.Stop(data.AgentID)

	// Clear agent initialization state so it will re-initialize
	delete(s.initialized, data.AgentID)

	// Clear all session mappings for this agent
	for convID, sessions := range s.agentSessions {
		if _, ok := sessions[data.AgentID]; ok {
			delete(s.agentSessions[convID], data.AgentID)
		}
	}

	writeJSON(w, map[string]any{"success": true, "agent": agent})
}

func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listWorkspaces(w, r)
	case "POST":
		s.createWorkspace(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces := make([]map[string]any, 0, len(s.config.Workspaces))
	for _, ws := range s.config.Workspaces {
		workspaces = append(workspaces, map[string]any{
			"id":   ws.ID,
			"name": ws.Name,
			"path": ws.Path,
		})
	}

	writeJSON(w, map[string]any{
		"workspaces": workspaces,
		"default":    s.config.DefaultWorkspace,
	})
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if data.Name == "" || data.Path == "" {
		writeError(w, "name and path are required", http.StatusBadRequest)
		return
	}

	// Validate path format and existence
	if err := validateWorkspacePath(data.Path); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate ID from name
	id := strings.ToLower(data.Name)
	id = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")

	// Check duplicate
	for _, ws := range s.config.Workspaces {
		if ws.ID == id {
			writeError(w, "Workspace with this name already exists", http.StatusBadRequest)
			return
		}
	}

	ws := config.WorkspaceConfig{ID: id, Name: data.Name, Path: data.Path}
	s.config.Workspaces = append(s.config.Workspaces, ws)
	s.workspaceStore.Add(ws)

	writeJSON(w, map[string]any{"workspace": ws})
}

// validateWorkspacePath checks if the path exists and has valid format for the current OS
func validateWorkspacePath(path string) error {
	// On Windows, check for Git Bash style paths (e.g., /c/Users/...)
	if runtime.GOOS == "windows" {
		// Git Bash paths start with /drive/ (e.g., /c/, /d/)
		if len(path) >= 3 && path[0] == '/' && path[2] == '/' {
			return &pathError{
				msg: "Invalid path format for Windows. Use Windows-style path (e.g., C:\\Users\\... or C:/Users/...) instead of Git Bash path (" + path + ")",
			}
		}
		// Also check for simple Unix paths that won't work on Windows
		if len(path) > 0 && path[0] == '/' && (len(path) < 2 || path[1] != '/') {
			return &pathError{
				msg: "Invalid path format for Windows. Path appears to be Unix-style: " + path,
			}
		}
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &pathError{msg: "Path does not exist: " + path}
		}
		return &pathError{msg: "Cannot access path: " + path + " (" + err.Error() + ")"}
	}

	// Check if it's a directory
	if !info.IsDir() {
		return &pathError{msg: "Path is not a directory: " + path}
	}

	return nil
}

type pathError struct {
	msg string
}

func (e *pathError) Error() string {
	return e.msg
}

func (s *Server) handlePermissionConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		AgentID    string `json:"agentId"`
		ToolCallID string `json:"toolCallId"`
		OptionID   string `json:"optionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	agent, err := s.agents.Get(data.AgentID)
	if err != nil {
		writeError(w, "Agent not found", http.StatusNotFound)
		return
	}

	agent.ConfirmPermission(data.ToolCallID, data.OptionID)
	writeJSON(w, map[string]any{"success": true})
}

func (s *Server) resolveWorkspacePath(workspaceID string) string {
	if workspaceID != "" {
		if ws := s.config.FindWorkspace(workspaceID); ws != nil {
			return ws.Path
		}
	}

	if s.config.DefaultWorkspace != "" {
		if ws := s.config.FindWorkspace(s.config.DefaultWorkspace); ws != nil {
			return ws.Path
		}
	}

	if len(s.config.Workspaces) > 0 {
		return s.config.Workspaces[0].Path
	}

	return "."
}


package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/daodao97/acpone/internal/agent"
	"github.com/daodao97/acpone/internal/conversation"
	"github.com/daodao97/acpone/internal/jsonrpc"
)

type chatRequest struct {
	Message        string   `json:"message"`
	ConversationID string   `json:"conversationId"`
	WorkspaceID    string   `json:"workspaceId"`
	Files          []string `json:"files"` // Uploaded file paths
}

type streamItem struct {
	Type string
	Text string
	Tool *conversation.ToolCallInfo
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	sendEvent := func(event string, data any) {
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
		flusher.Flush()
	}

	// Get or create conversation
	convID, isNew := s.getOrCreateConversation(req)
	conv := s.conversations.Get(convID)

	// Determine agent
	mentionedAgent := s.router.DetectMention(req.Message)
	previousAgent := conv.ActiveAgent
	agentID := previousAgent

	if mentionedAgent != "" {
		agentID = mentionedAgent
		if agentID != previousAgent {
			s.conversations.SetActiveAgent(convID, agentID)
			log.Printf("Agent switched via @mention: %s -> %s", previousAgent, agentID)
		}
	}

	agentChanged := previousAgent != agentID && len(conv.Messages) > 0

	// Initialize agent if needed
	if !s.initialized[agentID] {
		sendEvent("status", map[string]string{"message": fmt.Sprintf("Initializing %s...", agentID)})
		if err := s.initializeAgent(agentID); err != nil {
			sendEvent("error", map[string]string{"message": err.Error()})
			return
		}
		s.initialized[agentID] = true
	}

	// Get or create agent session
	// Get agent process and set up handlers early (before session/new)
	// This ensures we capture available_commands_update sent after session/new
	agentProc, _ := s.agents.Get(agentID)
	agentProc.SetWorkingDir(s.resolveWorkspacePath(req.WorkspaceID))

	streamItems := make([]streamItem, 0)
	currentText := ""
	toolCallMap := make(map[string]int)

	agentProc.OnNotification(func(msg *jsonrpc.Message) {
		s.handleNotification(msg, sendEvent, &streamItems, &currentText, toolCallMap, agentID)
	})

	agentProc.OnPermission(func(req *agent.PermissionRequest) {
		sendEvent("permission_request", req)
	})

	sessionsMap := s.agentSessions[convID]
	if sessionsMap == nil {
		sessionsMap = make(map[string]string)
		s.agentSessions[convID] = sessionsMap
	}

	sessionID := sessionsMap[agentID]
	if sessionID == "" {
		cwd := s.resolveWorkspacePath(req.WorkspaceID)
		var err error
		sessionID, err = s.createAgentSession(agentID, cwd)
		if err != nil {
			sendEvent("error", map[string]string{"message": err.Error()})
			return
		}
		sessionsMap[agentID] = sessionID
	}

	s.conversations.SetSessionID(convID, sessionID)

	// Build prompt with context if agent changed
	promptText := req.Message

	// Add file references to prompt
	if len(req.Files) > 0 {
		promptText = formatFileReferences(req.Files) + " " + promptText
	}

	if agentChanged {
		context := s.conversations.GetContextSummary(convID, 10)
		if context != "" {
			promptText = context + "User: " + promptText
			sendEvent("status", map[string]string{"message": fmt.Sprintf("Switching to %s with context...", agentID)})
		}
	}

	s.conversations.AddUserMessage(convID, req.Message)

	sendEvent("session", map[string]any{
		"conversationId": convID,
		"sessionId":      sessionID,
		"agent":          agentID,
		"isNew":          isNew,
	})
	sendEvent("status", map[string]string{"message": "Processing..."})

	// Call session/prompt
	response, err := agentProc.Request("session/prompt", map[string]any{
		"sessionId": sessionID,
		"prompt":    []map[string]string{{"type": "text", "text": promptText}},
	})

	if err != nil {
		sendEvent("error", map[string]string{"message": err.Error()})
		return
	}

	// Finalize stream items
	if currentText != "" {
		streamItems = append(streamItems, streamItem{Type: "text", Text: currentText})
	}

	for _, item := range streamItems {
		if item.Type == "text" {
			s.conversations.AddAssistantMessage(convID, item.Text, agentID)
		} else if item.Tool != nil {
			s.conversations.AddToolCall(convID, item.Tool, agentID)
		}
	}

	s.persistConversation(convID)

	// Send done
	var result map[string]any
	response.ParseResult(&result)
	if result == nil {
		result = make(map[string]any)
	}
	if result["stopReason"] == nil {
		result["stopReason"] = "end_turn"
	}
	sendEvent("done", result)
}

func (s *Server) getOrCreateConversation(req chatRequest) (string, bool) {
	if req.ConversationID != "" && s.conversations.Has(req.ConversationID) {
		return req.ConversationID, false
	}

	if req.ConversationID != "" {
		stored, err := s.sessionStore.Load(req.ConversationID)
		if err == nil {
			s.restoreConversation(stored)
			return req.ConversationID, false
		}
	}

	// Create new conversation
	convID := generateUUID()
	workspaceID := req.WorkspaceID
	if workspaceID == "" {
		workspaceID = s.config.DefaultWorkspace
	}
	s.conversations.Create(convID, s.config.DefaultAgent, workspaceID)
	s.agentSessions[convID] = make(map[string]string)
	return convID, true
}

func (s *Server) initializeAgent(agentID string) error {
	_, err := s.agents.Request(agentID, "initialize", map[string]any{
		"protocolVersion": 1,
		"clientCapabilities": map[string]any{
			"fs": map[string]bool{"readTextFile": true, "writeTextFile": true},
		},
		"clientInfo": map[string]string{"name": "acpone-go", "version": "0.1.0"},
	})
	return err
}

func (s *Server) createAgentSession(agentID, cwd string) (string, error) {
	result, err := s.agents.Request(agentID, "session/new", map[string]any{
		"cwd":        cwd,
		"mcpServers": []any{},
	})
	if err != nil {
		return "", err
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response")
	}

	sessionID, _ := resultMap["sessionId"].(string)
	if sessionID == "" {
		return "", fmt.Errorf("no sessionId in response")
	}

	// Set permission mode
	agentConfig := s.config.FindAgent(agentID)
	if agentConfig != nil && agentConfig.PermissionMode == "bypass" {
		modeID := "bypassPermissions"
		if agentID == "codex" {
			modeID = "auto"
		}
		s.agents.Request(agentID, "session/set_mode", map[string]any{
			"sessionId": sessionID,
			"modeId":    modeID,
		})
	}

	return sessionID, nil
}

// formatFileReferences formats file paths as @filename references for the prompt
func formatFileReferences(files []string) string {
	if len(files) == 0 {
		return ""
	}

	refs := make([]string, 0, len(files))
	for _, path := range files {
		// Extract filename from full path
		filename := filepath.Base(path)
		refs = append(refs, "@"+filename)
	}
	return strings.Join(refs, " ")
}

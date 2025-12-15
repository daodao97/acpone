package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/acpone/internal/storage"
)

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions := s.sessionStore.List()
	writeJSON(w, map[string]any{"sessions": sessions})
}

func (s *Server) handleSessionNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		WorkspaceID string `json:"workspaceId"`
	}
	json.NewDecoder(r.Body).Decode(&data)

	id := generateUUID()
	workspaceID := data.WorkspaceID
	if workspaceID == "" {
		workspaceID = s.config.DefaultWorkspace
	}

	session := storage.CreateSession(id, s.config.DefaultAgent, workspaceID)
	s.sessionStore.Save(session)
	s.conversations.Create(id, s.config.DefaultAgent, workspaceID)
	s.agentSessions[id] = make(map[string]string)

	writeJSON(w, map[string]any{
		"session": map[string]any{
			"id":           session.ID,
			"title":        session.Title,
			"activeAgent":  session.ActiveAgent,
			"workspaceId":  session.WorkspaceID,
			"messageCount": 0,
			"createdAt":    session.CreatedAt,
			"updatedAt":    session.UpdatedAt,
		},
	})
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if id == "" {
		writeError(w, "Session ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		session, err := s.sessionStore.Load(id)
		if err != nil {
			writeError(w, "Session not found", http.StatusNotFound)
			return
		}
		s.restoreConversation(session)
		writeJSON(w, map[string]any{"session": session})

	case "DELETE":
		s.sessionStore.Delete(id)
		s.conversations.Delete(id)
		delete(s.agentSessions, id)
		writeJSON(w, map[string]any{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) restoreConversation(session *storage.StoredSession) {
	s.conversations.Create(session.ID, session.ActiveAgent, session.WorkspaceID)
	for _, msg := range session.Messages {
		if msg.Role == "user" {
			s.conversations.AddUserMessage(session.ID, msg.Content)
		} else {
			s.conversations.AddAssistantMessage(session.ID, msg.Content, msg.Agent)
		}
	}
	s.agentSessions[session.ID] = make(map[string]string)
}

func (s *Server) persistConversation(convID string) {
	conv := s.conversations.Get(convID)
	if conv == nil {
		return
	}

	session := &storage.StoredSession{
		ID:          convID,
		Title:       storage.GenerateTitle(conv.Messages),
		Messages:    conv.Messages,
		ActiveAgent: conv.ActiveAgent,
		WorkspaceID: conv.WorkspaceID,
		CreatedAt:   conv.CreatedAt,
		UpdatedAt:   time.Now().UnixMilli(),
	}

	s.sessionStore.Save(session)
}

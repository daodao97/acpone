package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/daodao97/acpone/internal/conversation"
)

const defaultWorkspace = "_default"

// StoredSession represents a persisted session
type StoredSession struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Messages    []conversation.Message `json:"messages"`
	ActiveAgent string                 `json:"activeAgent"`
	WorkspaceID string                 `json:"workspaceId,omitempty"`
	CreatedAt   int64                  `json:"createdAt"`
	UpdatedAt   int64                  `json:"updatedAt"`
}

// SessionMeta is metadata for listing
type SessionMeta struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ActiveAgent  string `json:"activeAgent"`
	WorkspaceID  string `json:"workspaceId,omitempty"`
	MessageCount int    `json:"messageCount"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
}

// SessionStore manages session persistence
type SessionStore struct {
	baseDir string
}

// NewSessionStore creates a new session store
func NewSessionStore(baseDir string) *SessionStore {
	if baseDir == "" {
		baseDir = defaultBaseDir()
	}
	os.MkdirAll(baseDir, 0755)
	return &SessionStore{baseDir: baseDir}
}

func defaultBaseDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".acpone", "sessions")
}

func (s *SessionStore) workspaceDir(workspaceID string) string {
	if workspaceID == "" {
		workspaceID = defaultWorkspace
	}
	return filepath.Join(s.baseDir, workspaceID)
}

func (s *SessionStore) filePath(id, workspaceID string) string {
	wsDir := s.workspaceDir(workspaceID)
	os.MkdirAll(wsDir, 0755)
	return filepath.Join(wsDir, id+".json")
}

func (s *SessionStore) findFile(id string) (string, string) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return "", ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		filePath := filepath.Join(s.baseDir, entry.Name(), id+".json")
		if _, err := os.Stat(filePath); err == nil {
			wsID := entry.Name()
			if wsID == defaultWorkspace {
				wsID = ""
			}
			return filePath, wsID
		}
	}
	return "", ""
}

// Save saves a session
func (s *SessionStore) Save(session *StoredSession) error {
	filePath := s.filePath(session.ID, session.WorkspaceID)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// Load loads a session by ID
func (s *SessionStore) Load(id string) (*StoredSession, error) {
	filePath, _ := s.findFile(id)
	if filePath == "" {
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var session StoredSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// Delete deletes a session
func (s *SessionStore) Delete(id string) error {
	filePath, _ := s.findFile(id)
	if filePath == "" {
		return nil
	}
	return os.Remove(filePath)
}

// List returns all session metadata
func (s *SessionStore) List() []SessionMeta {
	var sessions []SessionMeta

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return sessions
	}

	for _, wsEntry := range entries {
		if !wsEntry.IsDir() {
			continue
		}

		wsDir := filepath.Join(s.baseDir, wsEntry.Name())
		files, err := os.ReadDir(wsDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if filepath.Ext(file.Name()) != ".json" {
				continue
			}

			filePath := filepath.Join(wsDir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			var session StoredSession
			if err := json.Unmarshal(data, &session); err != nil {
				continue
			}

			wsID := session.WorkspaceID
			if wsID == "" && wsEntry.Name() != defaultWorkspace {
				wsID = wsEntry.Name()
			}

			sessions = append(sessions, SessionMeta{
				ID:           session.ID,
				Title:        session.Title,
				ActiveAgent:  session.ActiveAgent,
				WorkspaceID:  wsID,
				MessageCount: len(session.Messages),
				CreatedAt:    session.CreatedAt,
				UpdatedAt:    session.UpdatedAt,
			})
		}
	}

	// Sort by updatedAt descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})

	return sessions
}

// GenerateTitle generates title from first user message
func GenerateTitle(messages []conversation.Message) string {
	for _, msg := range messages {
		if msg.Role == "user" {
			text := msg.Content
			if len(text) > 50 {
				return text[:50] + "..."
			}
			return text
		}
	}
	return "New Chat"
}

// CreateSession creates a new stored session
func CreateSession(id, activeAgent, workspaceID string) *StoredSession {
	now := time.Now().UnixMilli()
	return &StoredSession{
		ID:          id,
		Title:       "New Chat",
		Messages:    []conversation.Message{},
		ActiveAgent: activeAgent,
		WorkspaceID: workspaceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

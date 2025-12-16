package conversation

import (
	"fmt"
	"sync"
	"time"
)

// ToolCallInfo represents tool call information
type ToolCallInfo struct {
	ToolCallID  string `json:"toolCallId"`
	ToolName    string `json:"toolName"`
	Kind        string `json:"kind,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"` // pending, completed, error
	Input       string `json:"input,omitempty"`
	RawInput    string `json:"rawInput,omitempty"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
}

// MessageFile represents a file attached to a message
type MessageFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Message in conversation history
type Message struct {
	Role      string        `json:"role"` // user, assistant
	Content   string        `json:"content"`
	Agent     string        `json:"agent,omitempty"`
	ToolCall  *ToolCallInfo `json:"toolCall,omitempty"`
	Files     []MessageFile `json:"files,omitempty"`
	Timestamp int64         `json:"timestamp"`
}

// Conversation with full history
type Conversation struct {
	ID               string    `json:"id"`
	Messages         []Message `json:"messages"`
	ActiveAgent      string    `json:"activeAgent"`
	CurrentSessionID string    `json:"currentSessionId,omitempty"`
	WorkspaceID      string    `json:"workspaceId,omitempty"`
	CreatedAt        int64     `json:"createdAt"`
}

// Manager manages conversations
type Manager struct {
	conversations map[string]*Conversation
	mu            sync.RWMutex
}

// NewManager creates a new conversation manager
func NewManager() *Manager {
	return &Manager{
		conversations: make(map[string]*Conversation),
	}
}

// Create creates a new conversation
func (m *Manager) Create(id, defaultAgent, workspaceID string) *Conversation {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv := &Conversation{
		ID:          id,
		Messages:    []Message{},
		ActiveAgent: defaultAgent,
		WorkspaceID: workspaceID,
		CreatedAt:   time.Now().UnixMilli(),
	}
	m.conversations[id] = conv
	return conv
}

// Get returns a conversation by ID
func (m *Manager) Get(id string) *Conversation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conversations[id]
}

// Has checks if conversation exists
func (m *Manager) Has(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.conversations[id]
	return ok
}

// SetWorkspace sets workspace ID
func (m *Manager) SetWorkspace(id, workspaceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv, ok := m.conversations[id]; ok {
		conv.WorkspaceID = workspaceID
	}
}

// AddUserMessage adds a user message with optional files
func (m *Manager) AddUserMessage(id, content string, files []MessageFile) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv, ok := m.conversations[id]; ok {
		conv.Messages = append(conv.Messages, Message{
			Role:      "user",
			Content:   content,
			Files:     files,
			Timestamp: time.Now().UnixMilli(),
		})
	}
}

// AddAssistantMessage adds an assistant message
func (m *Manager) AddAssistantMessage(id, content, agent string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv, ok := m.conversations[id]; ok {
		conv.Messages = append(conv.Messages, Message{
			Role:      "assistant",
			Content:   content,
			Agent:     agent,
			Timestamp: time.Now().UnixMilli(),
		})
	}
}

// AddToolCall adds a tool call message
func (m *Manager) AddToolCall(id string, toolCall *ToolCallInfo, agent string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv, ok := m.conversations[id]; ok {
		conv.Messages = append(conv.Messages, Message{
			Role:      "assistant",
			Content:   "",
			Agent:     agent,
			ToolCall:  toolCall,
			Timestamp: time.Now().UnixMilli(),
		})
	}
}

// SetActiveAgent sets the active agent
func (m *Manager) SetActiveAgent(id, agent string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv, ok := m.conversations[id]; ok {
		conv.ActiveAgent = agent
	}
}

// SetSessionID sets the current session ID
func (m *Manager) SetSessionID(id, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv, ok := m.conversations[id]; ok {
		conv.CurrentSessionID = sessionID
	}
}

// GetContextSummary returns context summary for agent switching
func (m *Manager) GetContextSummary(id string, maxMessages int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conv, ok := m.conversations[id]
	if !ok || len(conv.Messages) == 0 {
		return ""
	}

	if maxMessages <= 0 {
		maxMessages = 10
	}

	start := 0
	if len(conv.Messages) > maxMessages {
		start = len(conv.Messages) - maxMessages
	}

	recent := conv.Messages[start:]
	lines := []string{"[Previous conversation context]"}

	for _, msg := range recent {
		prefix := "User"
		if msg.Role == "assistant" {
			prefix = fmt.Sprintf("Assistant (%s)", msg.Agent)
		}
		content := msg.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		lines = append(lines, fmt.Sprintf("%s: %s", prefix, content))
	}

	lines = append(lines, "[End of context]\n")

	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}

// Delete removes a conversation
func (m *Manager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.conversations, id)
}

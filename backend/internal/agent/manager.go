package agent

import (
	"fmt"
	"sync"

	"github.com/anthropics/acpone/internal/config"
)

// NotificationHandler handles agent notifications
type NotificationHandler func(agentID string, msg any)

// Manager manages agent lifecycle
type Manager struct {
	agents       map[string]*Process
	defaultAgent string
	mu           sync.RWMutex
	handlers     []NotificationHandler
}

// NewManager creates a new agent manager
func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		agents:       make(map[string]*Process),
		defaultAgent: cfg.DefaultAgent,
	}

	for i := range cfg.Agents {
		agent := &cfg.Agents[i]
		m.agents[agent.ID] = NewProcess(agent)
	}

	return m
}

// DefaultID returns the default agent ID
func (m *Manager) DefaultID() string {
	return m.defaultAgent
}

// Get returns an agent by ID
func (m *Manager) Get(id string) (*Process, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if id == "" {
		id = m.defaultAgent
	}

	agent, ok := m.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return agent, nil
}

// Has checks if agent exists
func (m *Manager) Has(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.agents[id]
	return ok
}

// IDs returns all agent IDs
func (m *Manager) IDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.agents))
	for id := range m.agents {
		ids = append(ids, id)
	}
	return ids
}

// Start starts an agent
func (m *Manager) Start(id string) (*Process, error) {
	agent, err := m.Get(id)
	if err != nil {
		return nil, err
	}

	if agent.Status() == StatusRunning {
		return agent, nil
	}

	if err := agent.Start(); err != nil {
		return nil, err
	}

	return agent, nil
}

// OnNotification registers a notification handler
func (m *Manager) OnNotification(handler NotificationHandler) {
	m.mu.Lock()
	m.handlers = append(m.handlers, handler)
	m.mu.Unlock()
}

// Request sends a request to an agent
func (m *Manager) Request(agentID, method string, params any) (any, error) {
	agent, err := m.Start(agentID)
	if err != nil {
		return nil, err
	}

	msg, err := agent.Request(method, params)
	if err != nil {
		return nil, err
	}

	var result any
	if err := msg.ParseResult(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// Shutdown stops all agents
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, agent := range m.agents {
		agent.Stop()
	}
	return nil
}

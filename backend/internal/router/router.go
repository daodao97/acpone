package router

import (
	"regexp"

	"github.com/daodao97/acpone/internal/config"
)

var mentionRegex = regexp.MustCompile(`@(\w+)`)

// RouteContext provides context for routing decisions
type RouteContext struct {
	PromptText string
	SessionID  string
	Meta       map[string]string
}

// Strategy defines a routing strategy
type Strategy interface {
	Route(ctx RouteContext) string
}

// Router routes requests to agents
type Router struct {
	strategies      []Strategy
	defaultAgent    string
	availableAgents map[string]bool
}

// New creates a new router
func New(cfg *config.Config) *Router {
	agents := make(map[string]bool)
	for _, a := range cfg.Agents {
		agents[a.ID] = true
	}

	strategies := buildStrategies(cfg.Routing, agents)

	return &Router{
		strategies:      strategies,
		defaultAgent:    cfg.DefaultAgent,
		availableAgents: agents,
	}
}

// DetectMention detects @mention in prompt text
func (r *Router) DetectMention(text string) string {
	matches := mentionRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		agentID := matches[1]
		if r.availableAgents[agentID] {
			return agentID
		}
	}
	return ""
}

// Route routes a request to an agent
func (r *Router) Route(ctx RouteContext) string {
	for _, s := range r.strategies {
		agentID := s.Route(ctx)
		if agentID != "" && r.availableAgents[agentID] {
			return agentID
		}
	}
	return r.defaultAgent
}

// DefaultAgent returns the default agent ID
func (r *Router) DefaultAgent() string {
	return r.defaultAgent
}

// HasAgent checks if agent exists
func (r *Router) HasAgent(id string) bool {
	return r.availableAgents[id]
}

func buildStrategies(routing *config.RoutingConfig, agents map[string]bool) []Strategy {
	var strategies []Strategy

	if routing == nil {
		return strategies
	}

	// Mention strategy (always first)
	strategies = append(strategies, &MentionStrategy{agents: agents})

	// Keyword strategy
	if len(routing.Keywords) > 0 {
		strategies = append(strategies, &KeywordStrategy{keywords: routing.Keywords})
	}

	// Meta strategy
	if routing.Meta {
		strategies = append(strategies, &MetaStrategy{})
	}

	return strategies
}

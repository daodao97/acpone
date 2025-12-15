package router

import (
	"strings"
)

// MentionStrategy routes by @mention
type MentionStrategy struct {
	agents map[string]bool
}

func (s *MentionStrategy) Route(ctx RouteContext) string {
	matches := mentionRegex.FindStringSubmatch(ctx.PromptText)
	if len(matches) > 1 {
		agentID := matches[1]
		if s.agents[agentID] {
			return agentID
		}
	}
	return ""
}

// KeywordStrategy routes by keywords in prompt
type KeywordStrategy struct {
	keywords map[string]string
}

func (s *KeywordStrategy) Route(ctx RouteContext) string {
	text := strings.ToLower(ctx.PromptText)
	for keyword, agentID := range s.keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return agentID
		}
	}
	return ""
}

// MetaStrategy routes by session metadata
type MetaStrategy struct{}

func (s *MetaStrategy) Route(ctx RouteContext) string {
	if ctx.Meta == nil {
		return ""
	}
	return ctx.Meta["agent"]
}

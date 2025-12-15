package api

import (
	"encoding/json"

	"github.com/daodao97/acpone/internal/conversation"
	"github.com/daodao97/acpone/internal/jsonrpc"
)

type sessionUpdate struct {
	SessionUpdate     string             `json:"sessionUpdate"`
	Content           any                `json:"content,omitempty"` // Can be object or array
	ToolCallID        string             `json:"toolCallId,omitempty"`
	Title             string             `json:"title,omitempty"`
	Status            string             `json:"status,omitempty"`
	Kind              string             `json:"kind,omitempty"`
	RawInput          map[string]any     `json:"rawInput,omitempty"`
	Error             string             `json:"error,omitempty"`
	Meta              *sessionUpdateMeta `json:"_meta,omitempty"`
	AvailableCommands []SlashCommand     `json:"availableCommands,omitempty"`
}

// SlashCommand represents an available slash command
type SlashCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Input       *struct {
		Hint string `json:"hint,omitempty"`
	} `json:"input,omitempty"`
}

type sessionUpdateMeta struct {
	ClaudeCode *struct {
		ToolName     string `json:"toolName,omitempty"`
		ToolResponse *struct {
			Stdout string `json:"stdout,omitempty"`
			Stderr string `json:"stderr,omitempty"`
			Type   string `json:"type,omitempty"`
			File   *struct {
				FilePath string `json:"filePath"`
				Content  string `json:"content"`
			} `json:"file,omitempty"`
		} `json:"toolResponse,omitempty"`
		Error string `json:"error,omitempty"`
	} `json:"claudeCode,omitempty"`
}

func (s *Server) handleNotification(
	msg *jsonrpc.Message,
	sendEvent func(string, any),
	streamItems *[]streamItem,
	currentText *string,
	toolCallMap map[string]int,
	agentID string,
) {
	if msg.Method != "session/update" {
		return
	}

	var params struct {
		Update sessionUpdate `json:"update"`
	}
	if err := msg.ParseParams(&params); err != nil {
		return
	}

	update := params.Update

	switch update.SessionUpdate {
	case "agent_message_chunk", "agent_thought_chunk":
		if text := extractTextContent(update.Content); text != "" {
			*currentText += text
		}
		// Forward text chunks to frontend
		sendEvent("update", params)
		return

	case "available_commands_update":
		if len(update.AvailableCommands) > 0 {
			// Store commands for this agent
			s.agentCommandsMu.Lock()
			s.agentCommands[agentID] = update.AvailableCommands
			s.agentCommandsMu.Unlock()

			sendEvent("commands", map[string]any{
				"agent":    agentID,
				"commands": update.AvailableCommands,
			})
		}
		return // Don't forward raw update for commands

	case "tool_call", "tool_call_update":
		// Flush current text
		if *currentText != "" {
			*streamItems = append(*streamItems, streamItem{Type: "text", Text: *currentText})
			*currentText = ""
		}

		toolID := update.ToolCallID
		if toolID == "" {
			return
		}

		toolName := update.Kind
		if update.Meta != nil && update.Meta.ClaudeCode != nil && update.Meta.ClaudeCode.ToolName != "" {
			toolName = update.Meta.ClaudeCode.ToolName
		}

		title := update.Title
		if title == "" {
			title = toolID
		}

		status := "pending"
		hasError := update.Error != "" || (update.Meta != nil && update.Meta.ClaudeCode != nil && update.Meta.ClaudeCode.Error != "")
		if hasError {
			status = "error"
		} else if update.Status == "completed" {
			status = "completed"
		}

		input := extractInput(update.RawInput)
		output, errMsg := extractOutput(update)
		// Only extract description if status is not completed (completed content is output, not description)
		var description string
		if update.Status != "completed" {
			description = extractDescription(update.Content)
		}

		// Build rawInput JSON string for display
		var rawInputJSON string
		if len(update.RawInput) > 0 {
			if data, err := json.Marshal(update.RawInput); err == nil {
				rawInputJSON = string(data)
			}
		}

		toolCall := &conversation.ToolCallInfo{
			ToolCallID:  toolID,
			ToolName:    toolName,
			Kind:        update.Kind,
			Title:       title,
			Description: description,
			Status:      status,
			Input:       input,
			RawInput:    rawInputJSON,
			Output:      output,
			Error:       errMsg,
		}

		if idx, ok := toolCallMap[toolID]; ok {
			existing := (*streamItems)[idx]
			if existing.Tool != nil {
				// Preserve fields from previous updates if current is empty
				if input == "" {
					toolCall.Input = existing.Tool.Input
					input = existing.Tool.Input
				}
				if rawInputJSON == "" {
					toolCall.RawInput = existing.Tool.RawInput
					rawInputJSON = existing.Tool.RawInput
				}
				if title == toolID && existing.Tool.Title != toolID {
					toolCall.Title = existing.Tool.Title
					title = existing.Tool.Title
				}
				if description == "" {
					toolCall.Description = existing.Tool.Description
					description = existing.Tool.Description
				}
				if output == "" {
					toolCall.Output = existing.Tool.Output
					output = existing.Tool.Output
				}
				if errMsg == "" {
					toolCall.Error = existing.Tool.Error
					errMsg = existing.Tool.Error
				}
			}
			(*streamItems)[idx] = streamItem{Type: "tool", Tool: toolCall}
		} else {
			toolCallMap[toolID] = len(*streamItems)
			*streamItems = append(*streamItems, streamItem{Type: "tool", Tool: toolCall})
		}

		// Send enriched tool call event with all details
		sendEvent("tool_call", map[string]any{
			"toolCallId":    toolID,
			"toolName":      toolName,
			"kind":          update.Kind,
			"title":         title,
			"description":   description,
			"status":        status,
			"input":         input,
			"rawInput":      rawInputJSON,
			"output":        output,
			"error":         errMsg,
			"sessionUpdate": update.SessionUpdate,
		})
		return // Don't send raw params for tool calls

	default:
		// Forward other updates to frontend
		sendEvent("update", params)
	}
}

// extractTextContent extracts text from content field
// Content can be: {"type":"text","text":"..."} or other formats
func extractTextContent(content any) string {
	if content == nil {
		return ""
	}

	// Try as map (object)
	m, ok := content.(map[string]any)
	if !ok {
		return ""
	}

	// Check type is text
	if t, _ := m["type"].(string); t == "text" {
		if text, _ := m["text"].(string); text != "" {
			return text
		}
	}

	return ""
}

func extractInput(rawInput map[string]any) string {
	if rawInput == nil {
		return ""
	}

	if cmd, ok := rawInput["command"].(string); ok {
		return cmd
	}
	if path, ok := rawInput["file_path"].(string); ok {
		return path
	}
	if pattern, ok := rawInput["pattern"].(string); ok {
		return pattern
	}
	if old, ok := rawInput["old_string"].(string); ok {
		return "old_string: " + old
	}

	data, _ := json.MarshalIndent(rawInput, "", "  ")
	return string(data)
}

func extractOutput(update sessionUpdate) (output, errMsg string) {
	if update.Meta != nil && update.Meta.ClaudeCode != nil {
		cc := update.Meta.ClaudeCode
		if cc.ToolResponse != nil {
			resp := cc.ToolResponse
			if resp.Stdout != "" {
				output = resp.Stdout
			}
			if resp.Stderr != "" {
				errMsg = resp.Stderr
			}
			if resp.Type == "text" && resp.File != nil {
				output = "File: " + resp.File.FilePath + "\n" + resp.File.Content
			}
		}
		if cc.Error != "" {
			errMsg = cc.Error
		}
	}

	if update.Error != "" {
		errMsg = update.Error
	}

	return
}

// extractDescription extracts description text from content array
// Content format: [{"type":"content","content":{"type":"text","text":"description"}}]
func extractDescription(content any) string {
	if content == nil {
		return ""
	}

	// Try as array
	arr, ok := content.([]any)
	if !ok || len(arr) == 0 {
		return ""
	}

	// Get first item
	item, ok := arr[0].(map[string]any)
	if !ok {
		return ""
	}

	// Get nested content object
	innerContent, ok := item["content"].(map[string]any)
	if !ok {
		return ""
	}

	// Get text
	text, _ := innerContent["text"].(string)
	return text
}

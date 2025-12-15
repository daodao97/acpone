package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/acpone/internal/jsonrpc"
)

// Request sends a JSON-RPC request and waits for response
func (p *Process) Request(method string, params any) (*jsonrpc.Message, error) {
	if p.Status() != StatusRunning {
		if err := p.Start(); err != nil {
			return nil, err
		}
	}

	p.mu.Lock()
	p.requestID++
	id := p.requestID
	resultCh := make(chan *jsonrpc.Message, 1)
	p.pending[id] = &PendingRequest{Result: resultCh, Method: method}
	p.mu.Unlock()

	req := jsonrpc.NewRequest(id, method, params)
	if err := p.write(req); err != nil {
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
		return nil, err
	}

	// Wait for response (no timeout - agent may take long)
	msg, ok := <-resultCh
	if !ok {
		return nil, fmt.Errorf("request cancelled")
	}

	if msg.Error != nil {
		return nil, msg.Error
	}

	return msg, nil
}

// ConfirmPermission responds to a permission request
func (p *Process) ConfirmPermission(toolCallID, optionID string) {
	p.mu.Lock()
	perm, ok := p.permissions[toolCallID]
	if ok {
		delete(p.permissions, toolCallID)
	}
	p.mu.Unlock()

	if ok && perm.Response != nil {
		perm.Response <- optionID
	}
}

// Notify sends a notification (no response)
func (p *Process) Notify(method string, params any) error {
	if p.Status() != StatusRunning {
		return nil
	}
	notif := jsonrpc.NewNotification(method, params)
	return p.write(notif)
}

func (p *Process) write(v any) error {
	p.mu.Lock()
	stdin := p.stdin
	p.mu.Unlock()

	if stdin == nil {
		return fmt.Errorf("process not running")
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	fmt.Printf(">>> [%s] %s\n", p.ID, string(data))
	_, err = fmt.Fprintf(stdin, "%s\n", data)
	return err
}

func (p *Process) readLoop() {
	// Capture current stdout to detect if this loop belongs to current process
	p.mu.Lock()
	currentStdout := p.stdout
	p.mu.Unlock()

	if currentStdout == nil {
		return
	}

	scanner := bufio.NewScanner(currentStdout)
	// Increase buffer for large responses
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		lineStr := string(line)
		fmt.Printf("<<< [%s] %s\n", p.ID, lineStr)

		var msg jsonrpc.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		p.handleMessage(&msg)
	}

	// Only set status if this is still the active process
	p.mu.Lock()
	if p.stdout == currentStdout || p.stdout == nil {
		p.status = StatusStopped
	}
	p.mu.Unlock()
}

func (p *Process) handleMessage(msg *jsonrpc.Message) {
	// Response to our request
	if msg.IsResponse() && msg.ID != nil {
		p.mu.Lock()
		req, ok := p.pending[*msg.ID]
		if ok {
			delete(p.pending, *msg.ID)
		}
		p.mu.Unlock()

		if ok {
			req.Result <- msg
		}
		return
	}

	// Request from agent
	if msg.IsRequest() {
		p.handleRequest(msg)
		return
	}

	// Notification from agent
	if msg.IsNotification() {
		if p.onNotification != nil {
			p.onNotification(msg)
		}
	}
}

func (p *Process) handleRequest(msg *jsonrpc.Message) {
	switch msg.Method {
	case "session/request_permission":
		p.handlePermissionRequest(msg)

	case "fs/read_text_file":
		p.handleReadFile(msg)

	case "fs/write_text_file":
		p.handleWriteFile(msg)

	default:
		if msg.ID != nil {
			p.sendError(*msg.ID, jsonrpc.MethodNotFound, "Method not found: "+msg.Method)
		}
	}
}

func (p *Process) handlePermissionRequest(msg *jsonrpc.Message) {
	var req PermissionRequest
	if err := msg.ParseParams(&req); err != nil {
		return
	}

	toolCallID := req.ToolCall.ToolCallID
	if toolCallID == "" {
		toolCallID = fmt.Sprintf("perm-%d", time.Now().UnixMilli())
	}

	// Emit permission request for frontend
	if p.onPermission != nil {
		p.onPermission(&req)
	}

	// Wait for response
	respCh := make(chan string, 1)
	p.mu.Lock()
	p.permissions[toolCallID] = &PendingPermission{
		RequestID: *msg.ID,
		Response:  respCh,
	}
	p.mu.Unlock()

	optionID := <-respCh

	outcome := "selected"
	if len(optionID) > 6 && optionID[:6] == "reject" {
		outcome = "rejected"
	}

	if msg.ID != nil {
		p.sendResponse(*msg.ID, map[string]any{
			"outcome": map[string]any{
				"outcome":  outcome,
				"optionId": optionID,
			},
		})
	}
}

func (p *Process) sendResponse(id int, result any) {
	resp := jsonrpc.NewResponse(id, result)
	p.write(resp)
}

func (p *Process) sendError(id int, code int, message string) {
	resp := jsonrpc.NewErrorResponse(id, code, message)
	p.write(resp)
}

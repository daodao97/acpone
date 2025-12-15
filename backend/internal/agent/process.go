package agent

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/anthropics/acpone/internal/config"
	"github.com/anthropics/acpone/internal/jsonrpc"
)

// Status represents agent process status
type Status string

const (
	StatusIdle     Status = "idle"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusError    Status = "error"
	StatusStopped  Status = "stopped"
)

// PermissionRequest from agent
type PermissionRequest struct {
	SessionID string `json:"sessionId"`
	Options   []struct {
		OptionID string `json:"optionId"`
		Name     string `json:"name"`
		Kind     string `json:"kind"`
	} `json:"options"`
	ToolCall struct {
		ToolCallID string         `json:"toolCallId"`
		RawInput   map[string]any `json:"rawInput,omitempty"`
		Status     string         `json:"status,omitempty"`
		Title      string         `json:"title,omitempty"`
		Kind       string         `json:"kind,omitempty"`
	} `json:"toolCall"`
}

// PendingRequest tracks an in-flight request
type PendingRequest struct {
	Result chan *jsonrpc.Message
	Method string
}

// PendingPermission tracks permission request
type PendingPermission struct {
	RequestID int
	Response  chan string // optionId
}

// Process wraps a backend ACP process
type Process struct {
	ID         string
	Name       string
	config     *config.AgentConfig
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	status     Status
	requestID  int
	workingDir string

	pending     map[int]*PendingRequest
	permissions map[string]*PendingPermission
	mu          sync.Mutex

	// Event handlers
	onNotification func(msg *jsonrpc.Message)
	onPermission   func(req *PermissionRequest)
}

// NewProcess creates a new agent process
func NewProcess(cfg *config.AgentConfig) *Process {
	cwd, _ := os.Getwd()
	return &Process{
		ID:          cfg.ID,
		Name:        cfg.Name,
		config:      cfg,
		status:      StatusIdle,
		workingDir:  cwd,
		pending:     make(map[int]*PendingRequest),
		permissions: make(map[string]*PendingPermission),
	}
}

// Status returns current status
func (p *Process) Status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

// SetWorkingDir sets the working directory
func (p *Process) SetWorkingDir(dir string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workingDir = dir
}

// OnNotification sets notification handler
func (p *Process) OnNotification(fn func(*jsonrpc.Message)) {
	p.onNotification = fn
}

// OnPermission sets permission request handler
func (p *Process) OnPermission(fn func(*PermissionRequest)) {
	p.onPermission = fn
}

// Start starts the agent process
func (p *Process) Start() error {
	p.mu.Lock()
	if p.status == StatusRunning {
		p.mu.Unlock()
		return nil
	}
	p.status = StatusStarting
	p.mu.Unlock()

	cmd := exec.Command(p.config.Command, p.config.Args...)
	cmd.Env = os.Environ()
	for k, v := range p.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Windows: 隐藏控制台窗口
	hideWindow(cmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		p.setStatus(StatusError)
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.setStatus(StatusError)
		return err
	}

	// Capture stderr (on Windows without console, os.Stderr doesn't work)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		p.setStatus(StatusError)
		return err
	}

	if err := cmd.Start(); err != nil {
		p.setStatus(StatusError)
		return err
	}

	p.mu.Lock()
	p.cmd = cmd
	p.stdin = stdin
	p.stdout = stdout
	p.stderr = stderr
	p.status = StatusRunning
	p.mu.Unlock()

	go p.readLoop()
	go p.readStderr()
	return nil
}

// readStderr reads and logs stderr output
func (p *Process) readStderr() {
	p.mu.Lock()
	stderr := p.stderr
	p.mu.Unlock()

	if stderr == nil {
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			fmt.Printf("!!! [%s] stderr: %s", p.ID, string(buf[:n]))
		}
		if err != nil {
			break
		}
	}
}

// Stop stops the agent process and waits for it to exit
func (p *Process) Stop() error {
	p.mu.Lock()
	if p.cmd == nil {
		p.mu.Unlock()
		return nil
	}

	cmd := p.cmd
	stdin := p.stdin

	// Clear all state
	p.cmd = nil
	p.stdin = nil
	p.stdout = nil
	p.status = StatusStopped

	// Reject pending requests
	for id, req := range p.pending {
		close(req.Result)
		delete(p.pending, id)
	}
	p.mu.Unlock()

	// Close stdin to signal the process
	if stdin != nil {
		stdin.Close()
	}

	// Send interrupt signal
	_ = cmd.Process.Signal(os.Interrupt)

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited normally
	case <-time.After(3 * time.Second):
		// Force kill if not responding
		_ = cmd.Process.Kill()
		<-done
	}

	return nil
}

func (p *Process) setStatus(s Status) {
	p.mu.Lock()
	p.status = s
	p.mu.Unlock()
}

package agent

import (
	"os"
	"path/filepath"

	"github.com/daodao97/acpone/internal/jsonrpc"
)

func (p *Process) handleReadFile(msg *jsonrpc.Message) {
	var params struct {
		Path string `json:"path"`
	}
	if err := msg.ParseParams(&params); err != nil {
		if msg.ID != nil {
			p.sendError(*msg.ID, jsonrpc.InvalidParams, "Invalid params")
		}
		return
	}

	filePath := p.resolvePath(params.Path)
	content, err := os.ReadFile(filePath)
	if err != nil {
		if msg.ID != nil {
			p.sendError(*msg.ID, jsonrpc.InternalError, err.Error())
		}
		return
	}

	if msg.ID != nil {
		p.sendResponse(*msg.ID, map[string]string{"content": string(content)})
	}
}

func (p *Process) handleWriteFile(msg *jsonrpc.Message) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := msg.ParseParams(&params); err != nil {
		if msg.ID != nil {
			p.sendError(*msg.ID, jsonrpc.InvalidParams, "Invalid params")
		}
		return
	}

	filePath := p.resolvePath(params.Path)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		if msg.ID != nil {
			p.sendError(*msg.ID, jsonrpc.InternalError, err.Error())
		}
		return
	}

	if err := os.WriteFile(filePath, []byte(params.Content), 0644); err != nil {
		if msg.ID != nil {
			p.sendError(*msg.ID, jsonrpc.InternalError, err.Error())
		}
		return
	}

	if msg.ID != nil {
		p.sendResponse(*msg.ID, nil)
	}
}

func (p *Process) resolvePath(targetPath string) string {
	if targetPath == "" {
		return p.workingDir
	}
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	return filepath.Join(p.workingDir, targetPath)
}

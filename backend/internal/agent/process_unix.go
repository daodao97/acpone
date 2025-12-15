//go:build !windows

package agent

import "os/exec"

// hideWindow 在非 Windows 系统上不需要任何操作
func hideWindow(cmd *exec.Cmd) {
	// Unix 系统不需要隐藏窗口
}

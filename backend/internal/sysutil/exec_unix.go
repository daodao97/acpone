//go:build !windows

package sysutil

import "os/exec"

// HideWindow 在非 Windows 系统上不需要任何操作
func HideWindow(cmd *exec.Cmd) {
	// Unix 系统不需要隐藏窗口
}

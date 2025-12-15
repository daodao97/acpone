//go:build windows

package sysutil

import (
	"os/exec"
	"syscall"
)

// HideWindow 在 Windows 上隐藏子进程的控制台窗口
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

//go:build windows

package sysutil

import (
	"os/exec"
	"syscall"
)

// HideWindow 在 Windows 上隐藏子进程的控制台窗口
// 只使用 CREATE_NO_WINDOW，不设置 HideWindow 以保持 stdio 正常工作
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

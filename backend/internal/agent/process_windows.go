//go:build windows

package agent

import (
	"os/exec"

	"github.com/daodao97/acpone/internal/sysutil"
)

// hideWindow 在 Windows 上隐藏子进程的控制台窗口
func hideWindow(cmd *exec.Cmd) {
	sysutil.HideWindow(cmd)
}

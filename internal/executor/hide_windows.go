// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

//go:build windows

package executor

import (
	"os/exec"
	"syscall"
)

// HideWindow 在 Windows 下以无控制台窗口方式启动子进程，避免黑框闪烁。
func HideWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

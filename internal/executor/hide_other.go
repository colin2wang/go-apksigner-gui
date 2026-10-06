//go:build !windows

package executor

import "os/exec"

// HideWindow 非 Windows 平台无需隐藏窗口，留空实现。
func HideWindow(cmd *exec.Cmd) {}

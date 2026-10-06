// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

//go:build !windows

package executor

import "os/exec"

// HideWindow 非 Windows 平台无需隐藏窗口，留空实现。
func HideWindow(cmd *exec.Cmd) {}

//go:build !windows

package secure

import "errors"

// 非 Windows 平台无 DPAPI，主密钥保持原样落盘（与旧版行为一致）。
// ErrNotProtected 供统一调用方判断。
var ErrNotProtected = errors.New("secure: 数据未经保护")

// ProtectData 非 Windows 平台为恒等操作。
func ProtectData(plain string) (string, error) {
	return plain, nil
}

// UnprotectData 非 Windows 平台为恒等操作。
func UnprotectData(data string) (string, error) {
	return data, nil
}

// IsProtected 非 Windows 平台恒为 false。
func IsProtected(data string) bool {
	return false
}

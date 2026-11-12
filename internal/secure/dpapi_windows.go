//go:build windows

package secure

import (
	"encoding/base64"
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// ErrNotProtected 与 dpapi_other.go 中语义一致，供统一调用方判断。
var ErrNotProtected = errors.New("secure: 数据未经保护")

// 主密钥经 Windows DPAPI（CryptProtectData）保护后落盘，密文绑定当前用户，
// 拿到 config.json 的其他用户/其他机器无法解密。

// dpapiPrefix 标记 CryptoKey 字段中存放的是 DPAPI 保护块。
const dpapiPrefix = "dpapi:"

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

// ProtectData 用 DPAPI 保护字符串，返回 "dpapi:" + base64(blob)。
func ProtectData(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	blobIn := dataBlob{
		cbData: uint32(len(plain)),
		pbData: (*byte)(unsafe.Pointer(unsafe.StringData(plain))),
	}
	var blobOut dataBlob
	r1, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&blobIn)),
		0,       // p szDescription
		0, 0, 0, // 未使用：entropy / reserved / prompt
		0, // dwFlags
		uintptr(unsafe.Pointer(&blobOut)),
	)
	if r1 == 0 {
		return "", fmt.Errorf("secure: DPAPI 保护失败: %v", err)
	}
	defer localFree(blobOut.pbData)
	out := dpapiPrefix + base64.StdEncoding.EncodeToString(
		unsafe.Slice(blobOut.pbData, blobOut.cbData))
	return out, nil
}

// UnprotectData 还原 DPAPI 保护块；输入不是 dpapi: 前缀时返回 ErrNotProtected，
// 由调用方决定按旧版明文处理。
func UnprotectData(data string) (string, error) {
	if !IsProtected(data) {
		return "", ErrNotProtected
	}
	raw, err := base64.StdEncoding.DecodeString(data[len(dpapiPrefix):])
	if err != nil {
		return "", err
	}
	blobIn := dataBlob{
		cbData: uint32(len(raw)),
		pbData: (*byte)(unsafe.Pointer(unsafe.SliceData(raw))),
	}
	var blobOut dataBlob
	r1, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&blobIn)),
		0, // p szDescription
		0, 0, 0,
		0,
		uintptr(unsafe.Pointer(&blobOut)),
	)
	if r1 == 0 {
		return "", fmt.Errorf("secure: DPAPI 解保护失败: %v", err)
	}
	defer localFree(blobOut.pbData)
	return string(unsafe.Slice(blobOut.pbData, blobOut.cbData)), nil
}

// IsProtected 判断数据是否为 DPAPI 保护块。
func IsProtected(data string) bool {
	return len(data) > len(dpapiPrefix) && data[:len(dpapiPrefix)] == dpapiPrefix
}

func localFree(p *byte) {
	if p != nil {
		syscall.LocalFree(syscall.Handle(uintptr(unsafe.Pointer(p))))
	}
}

// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

// Package secure 提供基于 AES-256-GCM 的轻量对称加密，用于在本机配置文件中以密文形式保存密钥库密码。
// 主密钥随配置文件一同生成并存储（文件权限 0600），仅在本机、本用户下可用。
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encrypt 使用 base64 编码的 32 字节主密钥加密明文，返回 base64( nonce || ciphertext )。
func Encrypt(plain, keyB64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil || len(key) != 32 {
		return "", errors.New("secure: 无效的主密钥")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解析 base64( nonce || ciphertext ) 并用主密钥还原明文。
func Decrypt(cipherB64, keyB64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil || len(key) != 32 {
		return "", errors.New("secure: 无效的主密钥")
	}
	data, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", errors.New("secure: 密文长度不足")
	}
	nonce, ct := data[:ns], data[ns:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

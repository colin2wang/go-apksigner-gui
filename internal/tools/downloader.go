// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/i18n"
)

// ProgressFunc 下载过程回调。
type ProgressFunc func(dto.Progress)

// bufferSize 下载缓冲区，恒定为 32KB，避免大文件占用过多内存。
const bufferSize = 32 * 1024

// progressInterval 进度回调的最小间隔，避免高频刷新界面。
const progressInterval = 120 * time.Millisecond

// Download 下载 build-tools 压缩包：支持断点续传、失败退避重试、流式 SHA256 校验。
// 返回落盘路径。
func Download(ctx context.Context, client *http.Client, v dto.BuildToolVersion, destDir string, on ProgressFunc) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("tools.err.mkDownloadDir"), err)
	}
	dest := filepath.Join(destDir, filepath.Base(v.FileName))
	if dest == destDir || filepath.Base(v.FileName) == "" {
		return "", fmt.Errorf("%s", i18n.T("tools.err.badURL", v.URL))
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		ok, err := downloadOnce(ctx, client, v, dest, on)
		if err == nil && ok {
			return dest, nil
		}
		lastErr = err
		if attempt < 2 {
			backoff := time.Duration(1<<attempt) * time.Second
			emit(on, dto.Progress{FileName: v.FileName, Stage: "downloading", Message: i18n.T("tools.msg.retry", backoff, attempt+1)})
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("%s", i18n.T("tools.err.downloadIncomplete"))
	}
	return "", lastErr
}

// downloadOnce 执行单次（可续传）下载，返回是否完成。
func downloadOnce(ctx context.Context, client *http.Client, v dto.BuildToolVersion, dest string, on ProgressFunc) (bool, error) {
	var offset int64
	if fi, err := os.Stat(dest); err == nil {
		if v.Size > 0 && fi.Size() > v.Size {
			_ = os.Remove(dest) // 目标文件异常，重新开始
		} else {
			offset = fi.Size()
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.URL, nil)
	if err != nil {
		return false, fmt.Errorf("%s: %w", i18n.T("tools.err.makeReq"), err)
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("%s: %w", i18n.T("tools.err.reqFailed", v.URL), err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		offset = 0 // 服务端不支持续传，从头下载
	case http.StatusPartialContent:
	default:
		return false, fmt.Errorf("%s", i18n.T("tools.err.httpStatus", resp.StatusCode, v.URL))
	}

	total := v.Size
	if resp.ContentLength > 0 {
		if resp.StatusCode == http.StatusPartialContent {
			total = offset + resp.ContentLength
		} else {
			total = resp.ContentLength
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 && resp.StatusCode == http.StatusPartialContent {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(dest, flags, 0o644)
	if err != nil {
		return false, fmt.Errorf("%s: %w", i18n.T("tools.err.createFile", dest), err)
	}
	defer file.Close()

	// 续传时先把已有内容喂给 hasher，保证校验覆盖完整文件
	hasher := sha256.New()
	if offset > 0 {
		existing, err := os.Open(dest)
		if err != nil {
			return false, err
		}
		if _, err := io.CopyN(hasher, existing, offset); err != nil {
			existing.Close()
			return false, err
		}
		existing.Close()
	}

	writer := io.MultiWriter(file, hasher)
	buf := make([]byte, bufferSize)
	downloaded := offset
	lastEmit := time.Now()

	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := writer.Write(buf[:n]); werr != nil {
				return false, fmt.Errorf("%s: %w", i18n.T("tools.err.writeFile"), werr)
			}
			downloaded += int64(n)
			if time.Since(lastEmit) >= progressInterval {
				lastEmit = time.Now()
				emitProgress(on, v.FileName, downloaded, total, "downloading", "")
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return false, fmt.Errorf("%s: %w", i18n.T("tools.err.readData"), readErr)
		}
	}
	emitProgress(on, v.FileName, downloaded, total, "downloading", "")

	fi, err := os.Stat(dest)
	if err != nil {
		return false, err
	}
	if v.Size > 0 && fi.Size() != v.Size {
		return false, fmt.Errorf("%s", i18n.T("tools.err.sizeMismatch", fi.Size(), v.Size))
	}

	emitProgress(on, v.FileName, downloaded, total, "verifying", "正在校验文件完整性")
	if v.SHA256 != "" {
		actual := strings.ToLower(hex.EncodeToString(hasher.Sum(nil)))
		if !strings.EqualFold(actual, strings.ToLower(v.SHA256)) {
			_ = os.Remove(dest)
			return false, fmt.Errorf("%s", i18n.T("tools.err.sha256Mismatch", v.SHA256, actual))
		}
	}
	return true, nil
}

// emitProgress 计算百分比并触发回调。
func emitProgress(on ProgressFunc, fileName string, downloaded, total int64, stage, message string) {
	p := dto.Progress{
		FileName:   fileName,
		Downloaded: downloaded,
		Total:      total,
		Stage:      stage,
		Message:    message,
	}
	if total > 0 {
		p.Percent = float64(downloaded) / float64(total) * 100
		if p.Percent > 100 {
			p.Percent = 100
		}
	}
	emit(on, p)
}

// emit 安全调用回调。
func emit(on ProgressFunc, p dto.Progress) {
	if on != nil {
		on(p)
	}
}

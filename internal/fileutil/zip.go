// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

// Package fileutil 提供 ZIP 相关工具：安全解压、过滤重打包、目录操作。
package fileutil

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go-apksigner-gui/internal/i18n"
)

// ErrZipSlip 表示压缩包内存在越界路径，已被拒绝解压。
var ErrZipSlip = fmt.Errorf("%s", i18n.T("fileutil.err.zipSlip"))

// ExtractOptions 解压选项。
type ExtractOptions struct {
	// OnFile 每解压一个文件后回调，传入目标路径。
	OnFile func(path string)
	// Skip 返回 true 时跳过该条目。
	Skip func(name string) bool
}

// Extract 安全解压 zip 到 dest，做 ZIP Slip 校验。
func Extract(zipPath, dest string, opt ExtractOptions) (int, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", i18n.T("fileutil.err.openArchive"), err)
	}
	defer r.Close()

	cleanDest := filepath.Clean(dest)
	if err := os.MkdirAll(cleanDest, 0o755); err != nil {
		return 0, err
	}

	count := 0
	for _, f := range r.File {
		if opt.Skip != nil && opt.Skip(f.Name) {
			continue
		}
		target, err := safeJoin(cleanDest, f.Name)
		if err != nil {
			return count, err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return count, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return count, err
		}
		if err := writeEntry(f, target); err != nil {
			return count, err
		}
		count++
		if opt.OnFile != nil {
			opt.OnFile(target)
		}
	}
	return count, nil
}

// writeEntry 写入单个条目，保持原始文件权限。
func writeEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("fileutil.err.readEntry", f.Name), err)
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, entryPerm(f))
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return fmt.Errorf("%s: %w", i18n.T("fileutil.err.write", target), err)
	}
	return out.Close()
}

// entryPerm 计算落盘权限：目录 755，其余保留可执行位后取 644/755。
func entryPerm(f *zip.File) os.FileMode {
	mode := f.Mode()
	if mode == 0 {
		return 0o644
	}
	if mode.IsDir() {
		return 0o755
	}
	if mode&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

// safeJoin 确保解压目标位于 dest 之内。
func safeJoin(dest, name string) (string, error) {
	name = filepath.FromSlash(name)
	target := filepath.Clean(filepath.Join(dest, name))
	rel, err := filepath.Rel(dest, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: %s", ErrZipSlip, name)
	}
	return target, nil
}

// RemoveEntries 重新打包 ZIP，排除指定前缀（大小写不敏感），用于移除 META-INF 旧签名。
// 返回被排除的条目数量。
func RemoveEntries(src, dst string, prefixes []string) (int, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", i18n.T("fileutil.err.openFile", src), err)
	}
	defer r.Close()

	tmp := dst + ".tmp"
	outFile, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	w := zip.NewWriter(outFile)

	removed := 0
	for _, f := range r.File {
		if matchesPrefix(f.Name, prefixes) {
			removed++
			continue
		}
		if err := copyEntry(w, f); err != nil {
			w.Close()
			outFile.Close()
			os.Remove(tmp)
			return removed, err
		}
	}
	if err := w.Close(); err != nil {
		outFile.Close()
		os.Remove(tmp)
		return removed, err
	}
	if err := outFile.Close(); err != nil {
		os.Remove(tmp)
		return removed, err
	}
	if err := os.Rename(tmp, dst); err != nil {
		// 跨卷或占用时回退为复制
		if cerr := copyFile(tmp, dst); cerr != nil {
			return removed, cerr
		}
		os.Remove(tmp)
	}
	return removed, nil
}

// copyEntry 优先使用原始压缩数据直拷，失败时回退到重新压缩，保证兼容性与体积。
func copyEntry(w *zip.Writer, f *zip.File) error {
	if err := w.Copy(f); err == nil {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("fileutil.err.readEntry", f.Name), err)
	}
	defer rc.Close()

	header := f.FileHeader
	header.CompressedSize64 = 0
	header.UncompressedSize64 = 0
	header.CRC32 = 0
	dst, err := w.CreateHeader(&header)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, rc)
	return err
}

// matchesPrefix 判断是否命中需要剔除的前缀。
func matchesPrefix(name string, prefixes []string) bool {
	lower := strings.ToLower(name)
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// copyFile 复制文件内容。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// EnsureDir 确保目录存在。
func EnsureDir(path string) error { return os.MkdirAll(path, 0o755) }

// FileSize 返回文件大小，不存在时返回 0。
func FileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}

// Exists 判断路径是否存在。
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

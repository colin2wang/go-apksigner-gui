// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

package fileutil

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// buildZip 构造包含若干条目的测试压缩包。
func buildZip(t *testing.T, path string, names []string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for _, name := range names {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte("data:" + name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "evil.zip")
	buildZip(t, src, []string{"good/apksigner", "../escaped.txt"})

	_, err := Extract(src, filepath.Join(dir, "out"), ExtractOptions{})
	if err == nil {
		t.Fatal("包含越界路径时应当解压失败")
	}
	if !errors.Is(err, ErrZipSlip) {
		t.Fatalf("应返回 ErrZipSlip，实际: %v", err)
	}
	if Exists(filepath.Join(dir, "escaped.txt")) {
		t.Fatal("不应在目标目录之外创建文件")
	}
}

func TestExtractKeepsPermissions(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "ok.zip")
	buildZip(t, src, []string{"build-tools/34.0.0/lib/apksigner.jar", "build-tools/34.0.0/apksigner"})

	count, err := Extract(src, filepath.Join(dir, "out"), ExtractOptions{})
	if err != nil {
		t.Fatalf("解压失败: %v", err)
	}
	if count != 2 {
		t.Fatalf("期望解压 2 个文件，实际 %d", count)
	}
	if !Exists(filepath.Join(dir, "out", "build-tools", "34.0.0", "apksigner")) {
		t.Fatal("apksigner 未解压到预期路径")
	}
}

func TestRemoveEntriesDropsMetaInf(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "signed.apk")
	buildZip(t, src, []string{
		"AndroidManifest.xml",
		"classes.dex",
		"META-INF/MANIFEST.MF",
		"META-INF/CERT.RSA",
		"meta-inf/lower-case.txt",
	})

	dst := filepath.Join(dir, "unsigned.apk")
	removed, err := RemoveEntries(src, dst, []string{"META-INF/"})
	if err != nil {
		t.Fatalf("重打包失败: %v", err)
	}
	if removed != 3 {
		t.Fatalf("期望剔除 3 个 META-INF 条目，实际 %d", removed)
	}

	r, err := zip.OpenReader(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if len(r.File) != 2 {
		t.Fatalf("剩余条目数不符，实际 %d", len(r.File))
	}
	for _, f := range r.File {
		if len(f.Name) >= 8 && f.Name[:8] == "META-INF" {
			t.Fatalf("仍有签名文件残留: %s", f.Name)
		}
	}
}

func TestFileSizeAndExists(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if FileSize(p) != 5 {
		t.Fatalf("文件大小不符: %d", FileSize(p))
	}
	if FileSize(filepath.Join(dir, "missing")) != 0 {
		t.Fatal("缺失文件应返回 0")
	}
	if !Exists(p) || Exists(filepath.Join(dir, "missing")) {
		t.Fatal("Exists 判断错误")
	}
}

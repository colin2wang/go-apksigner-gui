package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return New(filepath.Join(dir, "config.json"))
}

func TestStoreDefaults(t *testing.T) {
	s := newTestStore(t)
	got := s.Snapshot()
	if got.ToolsDir == "" {
		t.Fatal("默认工具目录不应为空")
	}
	if !got.DefaultV1 || !got.DefaultV2 || !got.DefaultV3 {
		t.Fatal("默认应启用 V1/V2/V3 签名方案")
	}
	if got.RecentFiles == nil {
		t.Fatal("最近文件列表应初始化为空切片，避免 JSON null")
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	s := newTestStore(t)
	if err := s.Update(func(cfg *Settings) {
		cfg.LastAlias = "test-key"
		cfg.LastKeystore = filepath.Join("tmp", "demo.jks")
		cfg.DefaultV4 = true
	}); err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	data, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatalf("配置文件未生成: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("配置文件不是合法 JSON: %v", err)
	}

	reloaded := New(s.Path())
	got := reloaded.Snapshot()
	if got.LastAlias != "test-key" || got.LastKeystore != filepath.Join("tmp", "demo.jks") {
		t.Fatalf("配置往返读取不一致: %+v", got)
	}
	if !got.DefaultV4 {
		t.Fatal("DefaultV4 未持久化")
	}
}

func TestStoreIgnoresBrokenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(path)
	if s.Snapshot().ToolsDir == "" {
		t.Fatal("配置损坏时应回退到默认值")
	}
}

func TestApplyDefaultsFillsMissing(t *testing.T) {
	var s Settings
	applyDefaults(&s)
	if s.ToolsDir == "" {
		t.Fatal("应补齐工具目录")
	}
	if s.Mirror.BaseURL == "" {
		t.Fatal("应补齐默认镜像源")
	}
}

func TestAddRecent(t *testing.T) {
	list := AddRecent(nil, "a.apk")
	list = AddRecent(list, "b.apk")
	list = AddRecent(list, "a.apk")
	if list[0] != "a.apk" {
		t.Fatalf("最近使用的文件应排在首位: %v", list)
	}
	if len(list) != 2 {
		t.Fatalf("重复项应被去重: %v", list)
	}
	for i := 0; i < 20; i++ {
		list = AddRecent(list, filepath.Join("dir", string(rune('a'+i))+".apk"))
	}
	if len(list) > RecentLimit {
		t.Fatalf("最近文件数量应受限，实际 %d", len(list))
	}
}

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go-apksigner-gui/internal/i18n"
)

// Store 线程安全的配置存储器。
type Store struct {
	path string
	mu   sync.RWMutex
	data Settings
}

// New 创建存储器并加载已有配置（文件不存在时使用默认值）。
func New(path string) *Store {
	if path == "" {
		path = filepath.Join(Dir(), "config.json")
	}
	s := &Store{path: path, data: Defaults()}
	if err := s.Load(); err != nil {
		// 配置损坏不阻塞启动，使用默认值并记录到标准错误
		fmt.Fprintf(os.Stderr, "[config] 读取配置失败: %v\n", err)
	}
	return s
}

// Dir 返回配置目录：优先 os.UserConfigDir()，回退到主目录。
func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		home, herr := os.UserHomeDir()
		if herr != nil || home == "" {
			return ".go-apksigner-gui"
		}
		base = home
	}
	return filepath.Join(base, "go-apksigner-gui")
}

// Path 返回配置文件路径。
func (s *Store) Path() string { return s.path }

// Load 从磁盘加载配置。
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.data = Defaults()
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var v Settings
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("config.err.parse", s.path), err)
	}
	applyDefaults(&v)
	s.data = v
	return nil
}

// Save 原子写入配置（临时文件 + rename）。
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		// Windows 上目标被占用时 rename 可能失败，回退为直接写入
		return os.WriteFile(s.path, data, 0o600)
	}
	return nil
}

// Update 在锁保护下修改配置并落盘。
func (s *Store) Update(fn func(*Settings)) error {
	if fn == nil {
		return nil
	}
	s.mu.Lock()
	fn(&s.data)
	s.mu.Unlock()
	return s.Save()
}

// Snapshot 返回配置副本。
func (s *Store) Snapshot() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

// Get 返回可变引用的访问函数（内部更新后立即落盘）。
func (s *Store) Get() *Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := s.data
	return &cp
}

// applyDefaults 补齐缺失字段的默认值。
func applyDefaults(s *Settings) {
	if s.ToolsDir == "" {
		s.ToolsDir = DefaultToolsDir()
	}
	if s.Mirror.BaseURL == "" {
		s.Mirror = Defaults().Mirror
	}
	if s.RecentFiles == nil {
		s.RecentFiles = []string{}
	}
	if s.RecentAlias == nil {
		s.RecentAlias = []string{}
	}
	if s.CryptoKey == "" {
		s.CryptoKey = generateCryptoKey()
	}
	if s.SavedPasswords == nil {
		s.SavedPasswords = map[string]string{}
	}
	if s.Language == "" {
		s.Language = "zh-CN"
	}
	if !s.DefaultV1 && !s.DefaultV2 && !s.DefaultV3 {
		s.DefaultV1 = true
		s.DefaultV2 = true
		s.DefaultV3 = true
	}
}

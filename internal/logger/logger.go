// Package logger 提供结构化文件日志、输出脱敏与内存环形缓冲。
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ringSize 内存中保留的最大日志条数（供前端回看）。
const ringSize = 500

// Entry 一条结构化日志。
// Time 使用 Unix 毫秒时间戳（而非 time.Time），保证 Wails 绑定生成的 TS 类型简洁可用。
type Entry struct {
	Time    int64  `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Attrs   string `json:"attrs,omitempty"`
}

var (
	mu       sync.RWMutex
	entries  []Entry
	filePath string
)

// secretPatterns 需要脱敏的常见密码片段：
//  1. pass:xxx / password=xxx 形式的命令行参数（apksigner 的 --ks-pass pass:xxx）
//  2. -storepass xxx / -keypass xxx 形式的 keytool 参数
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(pass(word)?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s]+)`),
	regexp.MustCompile(`(?i)(\s-(?:storepass|keypass|srcstorepass|srckeypass|deststorepass|destkeypass)\s+)("[^"]*"|'[^']*'|[^\s]+)`),
}

// Mask 对文本做脱敏处理：隐藏显式给出的密文与命令行中的密码片段。
func Mask(text string, secrets ...string) string {
	masked := text
	for _, p := range secretPatterns {
		masked = p.ReplaceAllString(masked, "${1}******")
	}
	for _, s := range secrets {
		if len(s) >= 3 {
			masked = strings.ReplaceAll(masked, s, "******")
		}
	}
	return masked
}

// Init 初始化文件日志；dir 为空时仅输出到标准错误。
func Init(dir string) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "apksigner-gui.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	filePath = path
	handler := slog.NewTextHandler(io.MultiWriter(os.Stderr, f), &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
	return nil
}

// record 记录一条日志到环形缓冲。
func record(level string, msg string, attrs string) {
	mu.Lock()
	defer mu.Unlock()
	entries = append(entries, Entry{Time: time.Now().UnixMilli(), Level: level, Message: Mask(msg), Attrs: attrs})
	if len(entries) > ringSize {
		entries = entries[len(entries)-ringSize:]
	}
}

// Info 记录信息日志（同步写入环形缓冲）。
func Info(msg string, keyvals ...any) {
	record("INFO", msg, formatAttrs(keyvals...))
	slog.Info(msg, keyvals...)
}

// Warn 记录警告日志。
func Warn(msg string, keyvals ...any) {
	record("WARN", msg, formatAttrs(keyvals...))
	slog.Warn(msg, keyvals...)
}

// Error 记录错误日志。
func Error(msg string, keyvals ...any) {
	record("ERROR", msg, formatAttrs(keyvals...))
	slog.Error(msg, keyvals...)
}

// Debug 记录调试日志。
func Debug(msg string, keyvals ...any) {
	record("DEBUG", msg, formatAttrs(keyvals...))
	slog.Debug(msg, keyvals...)
}

// Recent 返回最近 n 条日志。
func Recent(n int) []Entry {
	mu.RLock()
	defer mu.RUnlock()
	if n <= 0 || n > len(entries) {
		n = len(entries)
	}
	out := make([]Entry, 0, n)
	copy(out, entries[len(entries)-n:])
	return append([]Entry{}, entries[len(entries)-n:]...)
}

// FilePath 返回日志文件路径（可能为空）。
func FilePath() string { return filePath }

func formatAttrs(keyvals ...any) string {
	if len(keyvals) == 0 {
		return ""
	}
	var sb strings.Builder
	for i := 0; i+1 < len(keyvals); i += 2 {
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(Mask(slog.Attr{Key: toString(keyvals[i]), Value: slog.AnyValue(keyvals[i+1])}.String()))
	}
	return sb.String()
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "attr"
}

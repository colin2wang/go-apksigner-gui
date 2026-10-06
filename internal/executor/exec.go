// Package executor 封装外部进程调用：统一超时控制、跨平台可执行文件后缀、
// stdout/stderr 分流捕获与节流的实时行回调。所有参数以 []string 传递，不经 shell，
// 天然免疫命令注入（Windows 批处理除外，内部已做转义）。
package executor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Result 描述一次外部进程的完整结果。
type Result struct {
	ExitCode int           `json:"exitCode"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Combined string        `json:"combined"`
	Duration time.Duration `json:"durationMs"`
	TimedOut bool          `json:"timedOut"`
	Err      error         `json:"-"`
	StartErr error         `json:"-"`
}

// OK 判断命令是否成功退出（输出到 stderr 但退出码为 0 也算成功，如 java 的告警提示）。
func (r *Result) OK() bool { return r != nil && r.StartErr == nil && r.ExitCode == 0 }

// ErrorText 返回便于展示的错误描述（优先 stderr，其次 stdout）。
func (r *Result) ErrorText() string {
	if r == nil {
		return ""
	}
	if strings.TrimSpace(r.Stderr) != "" {
		return r.Stderr
	}
	if strings.TrimSpace(r.Stdout) != "" {
		return r.Stdout
	}
	if r.Err != nil {
		return r.Err.Error()
	}
	return ""
}

// Output 返回合并后的输出。
func (r *Result) Output() string {
	if r == nil {
		return ""
	}
	if r.Combined != "" {
		return r.Combined
	}
	return strings.TrimSpace(r.Stdout + "\n" + r.Stderr)
}

// Options 进程调用的可选项。
type Options struct {
	Timeout  time.Duration             // 0 表示 5 分钟默认超时
	WorkDir  string                    // 工作目录
	Env      map[string]string         // 追加的环境变量
	OnLine   func(stream, text string) // 实时行回调（stream: stdout/stderr）
	Throttle time.Duration             // 回调节流间隔，默认 100ms
}

// DefaultTimeout 未指定超时时的默认值。
const DefaultTimeout = 5 * time.Minute

// ErrEmptyCommand 未提供可执行文件时返回。
var ErrEmptyCommand = errors.New("executor: 未提供可执行文件")

// windowsSuffix 记录 Windows 平台下常见工具的可执行后缀。
var windowsSuffix = map[string]string{
	"apksigner": ".bat",
	"apktool":   ".bat",
	"zipalign":  ".exe",
	"aapt2":     ".exe",
	"aapt":      ".exe",
	"keytool":   ".exe",
	"java":      ".exe",
	"jarsigner": ".exe",
}

// Normalize 在 Windows 上为无后缀的工具名补全后缀（如 apksigner -> apksigner.bat）。
// 其它平台原样返回；已带后缀的路径不会被改动。
func Normalize(name string) string {
	if runtime.GOOS != "windows" || name == "" {
		return name
	}
	base := filepath.Base(name)
	if filepath.Ext(base) != "" {
		return name
	}
	if suffix, ok := windowsSuffix[strings.ToLower(base)]; ok {
		return name + suffix
	}
	return name
}

// IsBatch 判断 Windows 下是否为批处理脚本（需要 cmd /c 执行）。
func IsBatch(name string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".bat" || ext == ".cmd"
}

// Run 执行外部命令并返回结构化结果。ctx 取消或超时会终止子进程。
func Run(ctx context.Context, name string, args []string, opt Options) *Result {
	if strings.TrimSpace(name) == "" {
		return &Result{ExitCode: -1, StartErr: ErrEmptyCommand, Err: ErrEmptyCommand}
	}
	if opt.Timeout <= 0 {
		opt.Timeout = DefaultTimeout
	}
	if opt.Throttle <= 0 {
		opt.Throttle = 100 * time.Millisecond
	}
	if ctx == nil {
		ctx = context.Background()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()

	cmd := buildCommand(timeoutCtx, name, args)
	if opt.WorkDir != "" {
		cmd.Dir = opt.WorkDir
	}
	if len(opt.Env) > 0 {
		env := os.Environ()
		for k, v := range opt.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return &Result{ExitCode: -1, StartErr: err, Err: err}
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return &Result{ExitCode: -1, StartErr: err, Err: err}
	}

	emitter := newLineEmitter(opt.Throttle, opt.OnLine)

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return &Result{ExitCode: -1, StartErr: fmt.Errorf("启动 %s 失败: %w", name, err), Err: err}
	}

	var mu sync.Mutex
	var outBuf, errBuf strings.Builder

	var wg sync.WaitGroup
	wg.Add(2)
	go pump(&wg, stdoutPipe, "stdout", &mu, &outBuf, emitter)
	go pump(&wg, stderrPipe, "stderr", &mu, &errBuf, emitter)
	wg.Wait()

	waitErr := cmd.Wait()
	emitter.flush()

	res := &Result{
		Stdout:   outBuf.String(),
		Stderr:   errBuf.String(),
		Combined: outBuf.String() + errBuf.String(),
		Duration: time.Since(start),
	}
	if waitErr != nil {
		res.Err = waitErr
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
	}
	if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
		res.TimedOut = true
	}
	return res
}

// RunSimple 便捷方法：不带回调地执行命令，返回输出文本与错误。
func RunSimple(ctx context.Context, name string, args []string, timeout time.Duration) (string, error) {
	res := Run(ctx, name, args, Options{Timeout: timeout})
	output := res.Output()
	if res.StartErr != nil {
		return output, res.StartErr
	}
	if res.TimedOut {
		return output, fmt.Errorf("命令执行超时 (%v): %s", timeout, name)
	}
	if res.ExitCode != 0 {
		return output, fmt.Errorf("%s 退出码 %d: %s", filepath.Base(name), res.ExitCode, res.ErrorText())
	}
	return output, nil
}

// buildCommand 构造命令：Windows 下批处理脚本通过 cmd /c 执行。
func buildCommand(ctx context.Context, name string, args []string) *exec.Cmd {
	var cmd *exec.Cmd
	if IsBatch(name) {
		cmd = exec.CommandContext(ctx, "cmd", "/c", buildBatchCommandLine(name, args))
	} else {
		cmd = exec.CommandContext(ctx, name, args...)
	}
	HideWindow(cmd)
	return cmd
}

// buildBatchCommandLine 拼接 cmd 可识别的单行命令串。
func buildBatchCommandLine(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteArg(name))
	for _, a := range args {
		parts = append(parts, quoteArg(a))
	}
	return strings.Join(parts, " ")
}

// quoteArg 需要时对 Windows 参数加引号并转义内部引号。
// cmd.exe 中双引号内的嵌入引号须写作 ""（而非 \"），否则会破坏引号配平导致解析失败。
func quoteArg(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, " \t\"&|<>()^!%,;=") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// pump 按行读取进程输出，写入缓冲区并触发回调。
func pump(wg *sync.WaitGroup, r io.Reader, stream string, mu *sync.Mutex, buf *strings.Builder, em *lineEmitter) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		mu.Lock()
		buf.WriteString(line)
		buf.WriteString("\n")
		mu.Unlock()
		em.add(stream, line)
	}
	// 忽略读取错误：进程被 kill 时管道关闭属正常情况
}

// lineEmitter 对回调做节流，避免高频输出打满事件通道。
type lineEmitter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
	buf      []string
	timer    *time.Timer
	emit     func(stream, text string)
	stream   string
}

func newLineEmitter(interval time.Duration, emit func(stream, text string)) *lineEmitter {
	return &lineEmitter{interval: interval, emit: emit}
}

func (e *lineEmitter) add(stream, line string) {
	if e.emit == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stream = stream
	e.buf = append(e.buf, line)
	if e.interval <= 0 || time.Since(e.last) >= e.interval {
		e.flushLocked()
		return
	}
	if e.timer == nil {
		wait := e.interval - time.Since(e.last)
		e.timer = time.AfterFunc(wait, e.flush)
	}
}

func (e *lineEmitter) flush() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.flushLocked()
}

func (e *lineEmitter) flushLocked() {
	if len(e.buf) == 0 {
		return
	}
	text := strings.Join(e.buf, "\n")
	e.buf = nil
	e.last = time.Now()
	if e.timer != nil {
		e.timer.Stop()
		e.timer = nil
	}
	stream := e.stream
	if e.emit != nil {
		go e.emit(stream, text)
	}
}

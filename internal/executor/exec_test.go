package executor

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBuildBatchCommandLine_QuotesWhenNeeded(t *testing.T) {
	got := buildBatchCommandLine(`C:\Program Files\Android\build-tools\apksigner.bat`, []string{"--ks", "C:\\my key.jks", "--ks-pass", "pass:ab cd"})
	for _, want := range []string{`"C:\Program Files\Android\build-tools\apksigner.bat"`, `"C:\my key.jks"`, `"pass:ab cd"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("期望包含 %q，实际: %s", want, got)
		}
	}
}

func TestQuoteArg_EmptyAndPlain(t *testing.T) {
	if got := quoteArg(""); got != `""` {
		t.Fatalf("空参数应输出 \"\"，实际 %q", got)
	}
	if got := quoteArg("apksigner"); got != "apksigner" {
		t.Fatalf("普通参数不应加引号，实际 %q", got)
	}
}

func TestNormalize(t *testing.T) {
	if runtime.GOOS != "windows" {
		if got := Normalize("/usr/bin/apksigner"); got != "/usr/bin/apksigner" {
			t.Fatalf("非 Windows 平台不应改写路径，实际 %s", got)
		}
		return
	}
	cases := map[string]string{
		`C:\tools\apksigner`:                 `C:\tools\apksigner.bat`,
		`C:\tools\zipalign`:                  `C:\tools\zipalign.exe`,
		`C:\Program Files\jdk\keytool`:       `C:\Program Files\jdk\keytool.exe`,
		`C:\tools\apksigner.bat`:             `C:\tools\apksigner.bat`,
		`C:\tools\build-tools\apksigner.jar`: `C:\tools\build-tools\apksigner.jar`,
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Fatalf("Normalize(%s) = %s，期望 %s", in, got, want)
		}
	}
}

func TestIsBatch(t *testing.T) {
	if runtime.GOOS != "windows" {
		if IsBatch("apksigner.bat") {
			t.Fatal("非 Windows 平台不应判定为批处理")
		}
		return
	}
	if !IsBatch(`C:\tools\apksigner.BAT`) {
		t.Fatal("Windows 下 .BAT 应判定为批处理")
	}
	if IsBatch(`C:\tools\zipalign.exe`) {
		t.Fatal(".exe 不应判定为批处理")
	}
}

func TestRunEmptyCommand(t *testing.T) {
	res := Run(context.Background(), "  ", nil, Options{})
	if res.StartErr == nil {
		t.Fatal("空命令应当返回启动错误")
	}
	if res.OK() {
		t.Fatal("空命令结果不应为成功")
	}
}

// TestHelperProcess 作为超时测试的子进程入口。
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_EXECUTOR_TEST_SLEEP") == "1" {
		time.Sleep(3 * time.Second)
		os.Exit(0)
	}
}

func TestRunTimeout(t *testing.T) {
	helper := os.Args[0]
	if _, err := os.Stat(helper); err != nil {
		t.Skip("无法访问测试可执行文件")
	}
	res := Run(context.Background(), helper, []string{"-test.run=TestHelperProcess"},
		Options{Timeout: 400 * time.Millisecond, Env: map[string]string{"GO_EXECUTOR_TEST_SLEEP": "1"}})
	if !res.TimedOut {
		t.Fatalf("期望超时标记为真，实际: timedOut=%v exit=%d", res.TimedOut, res.ExitCode)
	}
}

func TestRunOutputCapture(t *testing.T) {
	bin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("环境未安装 go，跳过")
	}
	var streamed []string
	res := Run(context.Background(), bin, []string{"version"}, Options{
		Timeout:  30 * time.Second,
		Throttle: 10 * time.Millisecond,
		OnLine: func(stream, text string) {
			streamed = append(streamed, stream+":"+text)
		},
	})
	if !res.OK() {
		t.Fatalf("执行 go version 失败: %v / %s", res.Err, res.ErrorText())
	}
	if !strings.Contains(res.Stdout, "go version") {
		t.Fatalf("未捕获到版本号输出: %q", res.Stdout)
	}
	if len(streamed) == 0 {
		t.Fatal("期望收到实时输出回调")
	}
}

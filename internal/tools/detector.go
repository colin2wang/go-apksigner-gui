package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"go-apksigner-gui/internal/executor"
)

// Tool 描述一个可用工具的调用方式。
// Mode = exe 表示可执行文件直接调用；Mode = jar 表示需通过 java -jar 调用。
type Tool struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

// Command 返回该工具的被调用形式：可执行文件路径与完整参数。
func (t Tool) Command(args []string) (string, []string) {
	if t.Mode == "jar" {
		return "java", append([]string{"-jar", t.Path}, args...)
	}
	return t.Path, args
}

// Available 判断工具路径是否有效。
func (t Tool) Available() bool { return t.Path != "" }

// spec 定义工具的候选文件名与版本查询方式。
type spec struct {
	name       string
	required   bool
	winFiles   []string
	unixFiles  []string
	versionArg []string
	jarRel     string // 相对工具目录的 jar 位置，用于 java -jar 兜底
}

var specs = []spec{
	{
		name: "apksigner", required: true,
		winFiles:   []string{"apksigner.bat", "apksigner.exe", "apksigner"},
		unixFiles:  []string{"apksigner"},
		versionArg: []string{"--version"},
		jarRel:     filepath.Join("lib", "apksigner.jar"),
	},
	{
		name: "zipalign", required: true,
		winFiles:   []string{"zipalign.exe", "zipalign.bat", "zipalign"},
		unixFiles:  []string{"zipalign"},
		versionArg: []string{"-h"},
	},
	{
		name: "aapt2", required: false,
		winFiles:   []string{"aapt2.exe", "aapt2"},
		unixFiles:  []string{"aapt2"},
		versionArg: []string{"version"},
	},
	{
		name: "keytool", required: true,
		winFiles:   []string{"keytool.exe", "keytool"},
		unixFiles:  []string{"keytool"},
		versionArg: []string{"-help"},
	},
	{
		name: "sdkmanager", required: false,
		winFiles:   []string{"sdkmanager.bat", "sdkmanager"},
		unixFiles:  []string{"sdkmanager"},
		versionArg: []string{"--version"},
	},
}

var versionPattern = regexp.MustCompile(`\d+(?:\.\d+){1,3}`)

// DetectOptions 探测所需上下文。
type DetectOptions struct {
	ToolsDir string            // 本地工具目录（优先搜索）
	Override map[string]string // 用户手工指定的工具路径
	SdkPath  string            // 用户配置的 Android SDK 根目录
}

// Detect 返回全部工具的可用情况。
func Detect(opt DetectOptions) map[string]Tool {
	result := make(map[string]Tool, len(specs))
	for _, s := range specs {
		t := Tool{Name: s.name, Mode: "exe"}

		// 1. 用户指定路径
		if p, ok := opt.Override[s.name]; ok && p != "" {
			if fileExists(p) {
				t.Path = p
				t.Source = "custom"
			}
		}
		// 2. 本地工具目录（最近安装的优先）
		if t.Path == "" && opt.ToolsDir != "" {
			if p := findInToolsDir(opt.ToolsDir, s); p != "" {
				t.Path = p
				t.Source = "tools"
			}
		}
		// 2.5 用户配置的 Android SDK 路径
		if t.Path == "" && opt.SdkPath != "" {
			if p := findInSdkDir(opt.SdkPath, s); p != "" {
				t.Path = p
				t.Source = "sdk"
			}
		}
		// 3. Java 环境变量 / 常见安装路径（keytool 专用）
		if t.Path == "" && s.name == "keytool" {
			if p := findKeytool(); p != "" {
				t.Path = p
				t.Source = "path"
			}
		}
		// 4. 系统 PATH
		if t.Path == "" {
			if p := findInPath(s); p != "" {
				t.Path = p
				t.Source = "path"
			}
		}
		// 4.5 优先以 jar 模式调用：避免 Windows 下 cmd /c 重新解析含特殊字符
		// （如密码中的 " % & | ( ) 等）的参数而报“命令语法不正确”。
		if t.Path != "" && s.jarRel != "" {
			if jar := filepath.Join(filepath.Dir(t.Path), s.jarRel); fileExists(jar) && javaAvailable() {
				t.Path = jar
				t.Mode = "jar"
			}
		}
		// 5. jar 兜底（apksigner 可用 java -jar 调用）
		if t.Path == "" && s.jarRel != "" {
			if p := findJar(opt.ToolsDir, s.jarRel); p != "" {
				t.Path = p
				t.Mode = "jar"
				t.Source = "tools"
			}
		}
		if t.Path != "" {
			t.Path = executor.Normalize(t.Path)
			t.Version = versionOf(t)
		}
		result[s.name] = t
	}
	return result
}

// versionOf 查询工具版本，失败时返回 detected。
func versionOf(t Tool) string {
	s := specOf(t.Name)
	if s == nil || len(s.versionArg) == 0 {
		return "detected"
	}
	bin, args := t.Command(s.versionArg)
	out, err := executor.RunSimple(context.Background(), bin, args, 20*time.Second)
	if err != nil && strings.TrimSpace(out) == "" {
		return "detected"
	}
	if v := versionPattern.FindString(out); v != "" {
		return v
	}
	return "detected"
}

func specOf(name string) *spec {
	for i := range specs {
		if specs[i].name == name {
			return &specs[i]
		}
	}
	return nil
}

// candidates 返回当前系统下的候选文件名。
func (s spec) candidates() []string {
	if runtime.GOOS == "windows" {
		return s.winFiles
	}
	return s.unixFiles
}

// findInToolsDir 在工具目录中递归查找，命中多版本时取版本号最大的目录。
func findInToolsDir(dir string, s spec) string {
	if dir == "" {
		return ""
	}
	type hit struct {
		path string
		key  string
	}
	var hits []hit
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		for _, name := range s.candidates() {
			if strings.EqualFold(info.Name(), name) {
				hits = append(hits, hit{path: path, key: buildToolsVersionOf(dir, path)})
				return nil
			}
		}
		return nil
	})
	if len(hits) == 0 {
		return ""
	}
	best := hits[0]
	for _, h := range hits[1:] {
		if compareBuildToolsVersion(h.key, best.key) > 0 {
			best = h
		}
	}
	return best.path
}

// findInSdkDir 在用户配置的 Android SDK 根目录下查找工具：
// sdkmanager 位于 cmdline-tools/latest/bin，其余工具位于 build-tools/<ver>/。
func findInSdkDir(sdkPath string, s spec) string {
	if sdkPath == "" {
		return ""
	}
	if s.name == "sdkmanager" {
		bin := filepath.Join(sdkPath, "cmdline-tools", "latest", "bin")
		for _, name := range s.candidates() {
			p := filepath.Join(bin, name)
			if fileExists(p) {
				return p
			}
		}
		return ""
	}
	return findInToolsDir(filepath.Join(sdkPath, "build-tools"), s)
}

// buildToolsVersionOf 从路径中提取 build-tools 版本片段（如 .../build-tools/34.0.0/apksigner）。
func buildToolsVersionOf(dir, path string) string {
	rel := strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(dir))
	parts := strings.Split(strings.Trim(rel, "/"), "/")
	for i, p := range parts {
		if p == "build-tools" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// compareBuildToolsVersion 版本号比较：>0 表示 a 更新。
func compareBuildToolsVersion(a, b string) int {
	parse := func(v string) [3]int {
		var out [3]int
		nums := strings.Split(v, ".")
		for i := 0; i < len(nums) && i < 3; i++ {
			n := 0
			for _, r := range nums[i] {
				if r < '0' || r > '9' {
					break
				}
				n = n*10 + int(r-'0')
			}
			out[i] = n
		}
		return out
	}
	x, y := parse(a), parse(b)
	for i := 0; i < 3; i++ {
		if x[i] != y[i] {
			if x[i] > y[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// findKeytool 从 JAVA_HOME 与常见安装目录查找 keytool。
func findKeytool() string {
	exe := "keytool"
	if runtime.GOOS == "windows" {
		exe = "keytool.exe"
	}
	candidates := []string{}
	if jh := os.Getenv("JAVA_HOME"); jh != "" {
		candidates = append(candidates, filepath.Join(jh, "bin", exe))
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if root := os.Getenv(env); root != "" {
			matches, _ := filepath.Glob(filepath.Join(root, "Java", "*", "bin", exe))
			candidates = append(candidates, matches...)
			matches, _ = filepath.Glob(filepath.Join(root, "Android", "*", "jbr", "bin", exe))
			candidates = append(candidates, matches...)
		}
	}
	if runtime.GOOS == "darwin" {
		matches, _ := filepath.Glob("/Library/Java/JavaVirtualMachines/*/Contents/Home/bin/keytool")
		candidates = append(candidates, matches...)
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

// findInPath 在系统 PATH 中查找。
func findInPath(s spec) string {
	for _, name := range s.candidates() {
		if p, err := exec.LookPath(name); err == nil && p != "" {
			return p
		}
	}
	return ""
}

// findJar 查找 jar 兜底路径。
func findJar(dir, rel string) string {
	if dir == "" {
		return ""
	}
	var found string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(filepath.ToSlash(path), filepath.ToSlash(rel)) {
			found = path
		}
		return nil
	})
	return found
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

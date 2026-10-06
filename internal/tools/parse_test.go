package tools

import (
	"strings"
	"testing"

	"go-apksigner-gui/internal/dto"
)

const sampleIndex = `<?xml version="1.0" encoding="utf-8"?>
<sdk-repository xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <license id="android-sdk-license" type="text">Terms and Conditions</license>
  <remotePackage path="build-tools;35.0.0">
    <type-details xsi:type="genericDetailsType"/>
    <revision><major>35</major><minor>0</minor><micro>0</micro></revision>
    <display-name>Android SDK Build-Tools 35</display-name>
    <archives>
      <archive os="linux"><size>59684244</size><checksum>aaaaaa</checksum><url>build-tools_r35-linux.zip</url></archive>
      <archive os="windows"><size>60123456</size><checksum>bbbbbb</checksum><url>build-tools_r35-windows.zip</url></archive>
    </archives>
  </remotePackage>
  <remotePackage path="build-tools;34.0.0">
    <revision><major>34</major><minor>0</minor><micro>0</micro></revision>
    <archives>
      <archive host-os="macosx"><size>1</size><checksum>cccccc</checksum><url>build-tools_r34-macosx.zip</url></archive>
      <archive host-os="windows"><size>2</size><checksum>dddddd</checksum><url>build-tools_r34-windows.zip</url></archive>
    </archives>
  </remotePackage>
  <remotePackage path="build-tools;36.1">
    <revision><major>36</major><minor>1</minor><micro>0</micro></revision>
    <archives>
      <archive os="any"><complete><size>3</size><checksum>eeeeee</checksum><url>build-tools_r36.1-any.zip</url></complete></archive>
    </archives>
  </remotePackage>
  <remotePackage path="platforms;android-35">
    <revision><major>35</major></revision>
    <archives>
      <archive os="windows"><size>9</size><checksum>ffffff</checksum><url>android-14.zip</url></archive>
    </archives>
  </remotePackage>
</sdk-repository>`

func TestParseRepositoryWindows(t *testing.T) {
	versions, err := ParseRepository([]byte(sampleIndex), "https://mirror.example.com/AndroidSDK/", "windows")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("期望 3 个 build-tools 版本，实际 %d: %+v", len(versions), versions)
	}
	if versions[0].Version != "36.1.0" {
		t.Fatalf("首个版本应为 36.1.0（含 any 归档且版本由 revision 补全），实际 %s", versions[0].Version)
	}
	found := map[string]dto.BuildToolVersion{}
	for _, v := range versions {
		found[v.Version] = v
	}
	if v := found["35.0.0"]; v.URL != "https://mirror.example.com/AndroidSDK/build-tools_r35-windows.zip" {
		t.Fatalf("下载地址拼接错误: %s", v.URL)
	}
	if v := found["34.0.0"]; v.SHA256 != "dddddd" {
		t.Fatalf("校验值解析错误: %s", v.SHA256)
	}
}

func TestParseRepositoryUnknownOS(t *testing.T) {
	// 未知系统只应命中 os="any" 的通用归档；平台专属归档必须被过滤
	versions, err := ParseRepository([]byte(sampleIndex), "https://mirror.example.com/", "plan9")
	if err != nil {
		t.Fatalf("存在通用归档时不该报错: %v", err)
	}
	if len(versions) != 1 || versions[0].Version != "36.1.0" {
		t.Fatalf("仅应保留通用归档版本，实际 %+v", versions)
	}
}

func TestParseRepositoryBroken(t *testing.T) {
	if _, err := ParseRepository([]byte("<sdk-repository><oops"), "https://x/", "windows"); err == nil {
		t.Fatal("残缺 XML 应返回错误")
	}
}

func TestResolveURL(t *testing.T) {
	cases := map[string]string{
		"build-tools_r35-windows.zip": "https://m/android/build-tools_r35-windows.zip",
		"/absolute/x.zip":             "https://m/android/absolute/x.zip",
		"https://other/x.zip":         "https://other/x.zip",
	}
	for in, want := range cases {
		if got := resolveURL("https://m/android", in); got != want {
			t.Fatalf("resolveURL(%q) = %s，期望 %s", in, got, want)
		}
	}
}

func TestMatchesOS(t *testing.T) {
	if !matchesOS("macosx", "macosx") {
		t.Fatal("相同系统应匹配")
	}
	if matchesOS("macosx", "windows") {
		t.Fatal("不同系统不应匹配")
	}
	if !matchesOS("any", "windows") {
		t.Fatal("any 应视为通用归档")
	}
	if matchesOS("", "linux") {
		t.Fatal("缺少系统标识的归档不应被匹配")
	}
	if !matchesOS("linux,macosx", "linux") {
		t.Fatal("逗号分隔的多系统列表应匹配")
	}
}

func TestStaticVersions(t *testing.T) {
	list := StaticVersions("windows", "https://m/")
	if len(list) == 0 {
		t.Fatal("静态清单不应为空")
	}
	if !strings.HasSuffix(list[0].URL, "-windows.zip") || list[0].FileName != "build-tools_r"+list[0].Version+"-windows.zip" {
		t.Fatalf("静态清单文件名不符合命名规则: %+v", list[0])
	}
	if !SortVersionsSortedDesc(list) {
		t.Fatal("静态清单应按版本号倒序")
	}
}

// SortVersionsSortedDesc 校验列表是否已按版本倒序。
func SortVersionsSortedDesc(list []dto.BuildToolVersion) bool {
	for i := 1; i < len(list); i++ {
		if compareBuildToolsVersion(list[i-1].Version, list[i].Version) < 0 {
			return false
		}
	}
	return true
}

func TestCompareBuildToolsVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"35.0.0", "34.0.0", 1},
		{"34.0.1", "34.0.0", 1},
		{"30.0.3", "31.0.0", -1},
		{"35.0.0", "35.0.0", 0},
		{"", "0.0.0", 0},
	}
	for _, c := range cases {
		if got := compareBuildToolsVersion(c.a, c.b); got != c.want {
			t.Fatalf("compare(%s,%s) = %d，期望 %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIndexURL(t *testing.T) {
	got := IndexURL("https://dl.google.com/android/repository")
	if got != "https://dl.google.com/android/repository/repository2-3.xml" {
		t.Fatalf("索引地址拼接错误: %s", got)
	}
}

func TestMirrorsNotEmpty(t *testing.T) {
	if len(Mirrors) < 2 {
		t.Fatal("应提供多个镜像源")
	}
	for _, m := range Mirrors {
		if !strings.HasPrefix(m.BaseURL, "http") {
			t.Fatalf("镜像源地址非法: %+v", m)
		}
	}
	if m := FindMirror("https://unknown.example.com/sdk/"); m.Name != "自定义" {
		t.Fatalf("未知地址应回退为自定义镜像源，实际 %+v", m)
	}
}

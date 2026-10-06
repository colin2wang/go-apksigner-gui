package apkinfo

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

const badgingOutput = `package: name='com.example.demo' versionCode='123' versionName='1.2.3' compileSdkVersion='33' compileSdkVersionCodename='13'
sdkVersion:'24'
targetSdkVersion:'33'
application-label:'Demo App'
application-label-zh-CN:'示例应用'
application-icon-160:'res/mipmap-anydpi-v26/ic_launcher.xml'
launchable-activity: name='com.example.demo.MainActivity'  label='' icon=''
uses-permission: name='android.permission.INTERNET'`

func TestParseBadging(t *testing.T) {
	info := ParseBadging(badgingOutput)
	if info.PackageName != "com.example.demo" {
		t.Fatalf("包名解析错误: %s", info.PackageName)
	}
	if info.VersionCode != "123" || info.VersionName != "1.2.3" {
		t.Fatalf("版本解析错误: %s / %s", info.VersionCode, info.VersionName)
	}
	if info.MinSDK != "24" || info.TargetSDK != "33" {
		t.Fatalf("SDK 解析错误: %s / %s", info.MinSDK, info.TargetSDK)
	}
	if info.Label != "Demo App" {
		t.Fatalf("应用名解析错误: %s", info.Label)
	}
}

func TestParseBadgingEmpty(t *testing.T) {
	info := ParseBadging("nothing useful")
	if info.PackageName != "" || info.MinSDK != "" {
		t.Fatalf("无匹配内容时不应填充字段: %+v", info)
	}
}

// attrFixture 描述构造 START_TAG 时的一个属性。
type attrFixture struct {
	nameIdx  uint32
	rawIdx   uint32
	dataType byte
	data     uint32
}

// buildFileHeader 构造二进制 XML 文件头。
func buildFileHeader(total int) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint16(buf[0:2], 0x0003)
	binary.LittleEndian.PutUint16(buf[2:4], 8)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(total))
	return buf
}

// buildStringPool 构造 UTF-8 字符串池 chunk。
func buildStringPool(strs []string) []byte {
	headerSize := 28
	stringsStart := headerSize + len(strs)*4

	data := []byte{}
	offsets := []int{}
	for _, s := range strs {
		offsets = append(offsets, len(data))
		data = append(data, byte(len([]rune(s)))) // 字符数（单字节变长）
		data = append(data, byte(len(s)))         // 字节数
		data = append(data, []byte(s)...)
	}
	buf := make([]byte, 0, stringsStart+len(data))
	head := make([]byte, headerSize)
	binary.LittleEndian.PutUint16(head[0:2], resStringPoolType)
	binary.LittleEndian.PutUint16(head[2:4], uint16(headerSize))
	binary.LittleEndian.PutUint32(head[4:8], uint32(stringsStart+len(data)))
	binary.LittleEndian.PutUint32(head[8:12], uint32(len(strs)))
	binary.LittleEndian.PutUint32(head[12:16], 0)
	binary.LittleEndian.PutUint32(head[16:20], 0x100) // UTF-8 编码
	binary.LittleEndian.PutUint32(head[20:24], uint32(stringsStart))
	binary.LittleEndian.PutUint32(head[24:28], 0)
	buf = append(buf, head...)
	for _, off := range offsets {
		o := make([]byte, 4)
		binary.LittleEndian.PutUint32(o, uint32(off))
		buf = append(buf, o...)
	}
	buf = append(buf, data...)
	return buf
}

// buildStartTag 构造一个 START_TAG chunk（headerSize=16，attrExt=20）。
func buildStartTag(nameIdx uint32, attrs []attrFixture) []byte {
	headerSize := 16
	attrExtSize := 20
	size := headerSize + attrExtSize + len(attrs)*20
	buf := make([]byte, size)
	binary.LittleEndian.PutUint16(buf[0:2], resXMLStartTagType)
	binary.LittleEndian.PutUint16(buf[2:4], uint16(headerSize))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(size))
	binary.LittleEndian.PutUint32(buf[8:12], 1)  // lineNumber
	binary.LittleEndian.PutUint32(buf[12:16], 0) // comment
	binary.LittleEndian.PutUint32(buf[16:20], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(buf[20:24], nameIdx) // name
	binary.LittleEndian.PutUint16(buf[24:26], uint16(attrExtSize))
	binary.LittleEndian.PutUint16(buf[26:28], 20)
	binary.LittleEndian.PutUint16(buf[28:30], uint16(len(attrs)))
	for i, a := range attrs {
		base := headerSize + attrExtSize + i*20
		binary.LittleEndian.PutUint32(buf[base:base+4], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(buf[base+4:base+8], a.nameIdx)
		binary.LittleEndian.PutUint32(buf[base+8:base+12], a.rawIdx)
		binary.LittleEndian.PutUint16(buf[base+12:base+14], 8)
		buf[base+14] = 0
		buf[base+15] = a.dataType
		binary.LittleEndian.PutUint32(buf[base+16:base+20], a.data)
	}
	return buf
}

// buildManifestFixture 构造一个最小可解析的二进制 manifest。
func buildManifestFixture() []byte {
	pool := []string{
		"",                 // 0
		"manifest",         // 1
		"package",          // 2
		"versionCode",      // 3
		"versionName",      // 4
		"minSdkVersion",    // 5
		"targetSdkVersion", // 6
		"com.demo.app",     // 7
		"1.0.0",            // 8
		"application",      // 9
		"label",            // 10
		"Demo",             // 11
	}
	strPool := buildStringPool(pool)
	manifestTag := buildStartTag(1, []attrFixture{
		{nameIdx: 2, rawIdx: 7, dataType: typeStringData, data: 7}, // package
		{nameIdx: 3, rawIdx: 0, dataType: typeIntData, data: 42},   // versionCode
		{nameIdx: 4, rawIdx: 8, dataType: typeStringData, data: 8}, // versionName
		{nameIdx: 5, rawIdx: 0, dataType: typeIntData, data: 26},   // minSdkVersion
		{nameIdx: 6, rawIdx: 0, dataType: typeIntData, data: 34},   // targetSdkVersion
	})
	applicationTag := buildStartTag(9, []attrFixture{
		{nameIdx: 10, rawIdx: 11, dataType: typeStringData, data: 11}, // label
	})
	total := 8 + len(strPool) + len(manifestTag) + len(applicationTag)
	out := buildFileHeader(total)
	out = append(out, strPool...)
	out = append(out, manifestTag...)
	out = append(out, applicationTag...)
	return out
}

func TestParseAXML(t *testing.T) {
	info, err := ParseAXML(buildManifestFixture())
	if err != nil {
		t.Fatalf("解析 AXML 失败: %v", err)
	}
	if info.PackageName != "com.demo.app" {
		t.Fatalf("包名解析错误: %s", info.PackageName)
	}
	if info.VersionCode != "42" {
		t.Fatalf("versionCode 解析错误: %s", info.VersionCode)
	}
	if info.VersionName != "1.0.0" {
		t.Fatalf("versionName 解析错误: %s", info.VersionName)
	}
	if info.MinSDK != "26" || info.TargetSDK != "34" {
		t.Fatalf("SDK 解析错误: %s / %s", info.MinSDK, info.TargetSDK)
	}
	if info.Label != "Demo" {
		t.Fatalf("应用名解析错误: %s", info.Label)
	}
}

func TestParseAXMLInvalid(t *testing.T) {
	if _, err := ParseAXML([]byte("not xml at all")); err == nil {
		t.Fatal("非二进制 XML 应返回错误")
	}
	if _, err := ParseAXML([]byte{}); err == nil {
		t.Fatal("空数据应返回错误")
	}
}

func TestParseManifestFromAPK(t *testing.T) {
	dir := t.TempDir()
	apk := filepath.Join(dir, "sample.apk")
	f, err := os.Create(apk)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	fw, err := w.Create("AndroidManifest.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(buildManifestFixture()); err != nil {
		t.Fatal(err)
	}
	fw2, err := w.Create("classes.dex")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw2.Write([]byte("dex"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	info, err := ParseManifest(apk)
	if err != nil {
		t.Fatalf("从 APK 解析 manifest 失败: %v", err)
	}
	if info.PackageName != "com.demo.app" || info.Source != "axml" {
		t.Fatalf("解析结果错误: %+v", info)
	}
}

func TestParseManifestMissing(t *testing.T) {
	dir := t.TempDir()
	apk := filepath.Join(dir, "empty.apk")
	if err := os.WriteFile(apk, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseManifest(apk); err == nil {
		t.Fatal("非法 APK 应返回错误")
	}
}

func TestStringPoolDecodeSafety(t *testing.T) {
	// 越界索引不应 panic
	if got := stringByIndex([]string{"a"}, 99); got != "" {
		t.Fatalf("越界索引应返回空串，实际 %q", got)
	}
}

func TestReadLength16(t *testing.T) {
	data := []byte{0x05, 0x81, 0x02}
	if n, p, ok := readLength16(data, 0); !ok || n != 5 || p != 1 {
		t.Fatalf("单字节长度解析错误: %d %d %v", n, p, ok)
	}
	if n, p, ok := readLength16(data, 1); !ok || n != 0x102 || p != 3 {
		t.Fatalf("双字节长度解析错误: %d %d %v", n, p, ok)
	}
	if _, _, ok := readLength16(data, 5); ok {
		t.Fatal("越界读取应返回 false")
	}
}

func TestParseAttributesStopsAtBoundary(t *testing.T) {
	// 属性区截断时不应 panic 或死循环
	chunk := bytes.Repeat([]byte{0}, 40)
	if attrs := parseAttributes(chunk, 16, nil); attrs != nil {
		t.Fatalf("异常 chunk 不应解析出属性: %+v", attrs)
	}
}

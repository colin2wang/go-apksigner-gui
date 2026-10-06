// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

package apkinfo

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/i18n"
)

// 二进制 XML 中的 chunk 类型。
const (
	resStringPoolType  = 0x0001
	resXMLStartTagType = 0x0102
	typeStringData     = 0x03
	typeIntData        = 0x10
	typeBooleanData    = 0x12
)

// maxManifestSize AndroidManifest.xml 的最大读取上限，防止异常包导致内存暴涨。
const maxManifestSize = 8 << 20

// ErrManifestMissing APK 中没有 AndroidManifest.xml。
var ErrManifestMissing = errors.New(i18n.T("apkinfo.err.manifestMissing"))

// manifestAttr 记录从 AXML 中解析出的属性。
type manifestAttr struct {
	name  string
	value string
}

// ParseManifest 从 APK 中读取并解析 AndroidManifest.xml。
func ParseManifest(apkPath string) (dto.APKInfo, error) {
	data, err := readManifest(apkPath)
	if err != nil {
		return dto.APKInfo{Path: apkPath}, err
	}
	info, err := ParseAXML(data)
	info.Path = apkPath
	info.Source = "axml"
	return info, err
}

// readManifest 从 APK(ZIP) 中取出二进制 manifest 数据。
func readManifest(apkPath string) ([]byte, error) {
	r, err := zip.OpenReader(apkPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("apkinfo.err.openApk"), err)
	}
	defer r.Close()

	for _, f := range r.File {
		if !strings.EqualFold(f.Name, "AndroidManifest.xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", i18n.T("apkinfo.err.readEntry", f.Name), err)
		}
		defer rc.Close()
		limit := f.UncompressedSize64
		if limit > maxManifestSize {
			limit = maxManifestSize
		}
		data, rerr := io.ReadAll(io.LimitReader(rc, int64(limit)))
		if rerr != nil {
			return nil, rerr
		}
		return data, nil
	}
	return nil, ErrManifestMissing
}

// ParseAXML 解析二进制 AndroidManifest.xml，提取包名、版本与 SDK 信息。
// 只处理字符串池与 START_TAG，遇到异常结构时返回已解析到的部分结果。
func ParseAXML(data []byte) (dto.APKInfo, error) {
	info := dto.APKInfo{}
	if len(data) < 8 {
		return info, errors.New(i18n.T("apkinfo.err.manifestShort"))
	}
	if binary.LittleEndian.Uint16(data[0:2]) != 0x0003 {
		return info, errors.New(i18n.T("apkinfo.err.notBinaryXML"))
	}
	total := int(binary.LittleEndian.Uint32(data[4:8]))
	if total < 8 || total > len(data) {
		total = len(data)
	}

	pool := []string{}
	attrs := []manifestAttr{}
	inManifest := false
	inApplication := false

	for off := 8; off+8 <= total; {
		chunkType := binary.LittleEndian.Uint16(data[off : off+2])
		headerSize := int(binary.LittleEndian.Uint16(data[off+2 : off+4]))
		chunkSize := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		if chunkSize < headerSize || chunkSize <= 0 || off+chunkSize > total {
			break
		}
		switch chunkType {
		case resStringPoolType:
			pool, _ = parseStringPool(data[off : off+chunkSize])
		case resXMLStartTagType:
			name := stringByIndex(pool, binary.LittleEndian.Uint32(data[off+headerSize+4:off+headerSize+8]))
			switch name {
			case "manifest":
				inManifest = true
			case "application":
				if inManifest {
					inApplication = true
				}
			}
			if inManifest {
				attrs = append(attrs, parseAttributes(data[off:off+chunkSize], headerSize, pool)...)
			}
		}
		off += chunkSize
		if inApplication {
			break
		}
	}

	for _, a := range attrs {
		switch a.name {
		case "package":
			info.PackageName = a.value
		case "versionCode":
			info.VersionCode = a.value
		case "versionName":
			info.VersionName = a.value
		case "minSdkVersion":
			info.MinSDK = a.value
		case "targetSdkVersion":
			info.TargetSDK = a.value
		case "label":
			if info.Label == "" {
				info.Label = a.value
			}
		}
	}
	if info.PackageName == "" {
		return info, errors.New(i18n.T("apkinfo.err.noPackage"))
	}
	return info, nil
}

// parseAttributes 解析 START_TAG 中的属性列表。
func parseAttributes(chunk []byte, headerSize int, pool []string) []manifestAttr {
	if len(chunk) < headerSize+36 {
		return nil
	}
	ext := headerSize
	attributeStart := int(binary.LittleEndian.Uint16(chunk[ext+8 : ext+10]))
	attributeSize := int(binary.LittleEndian.Uint16(chunk[ext+10 : ext+12]))
	count := int(binary.LittleEndian.Uint16(chunk[ext+12 : ext+14]))
	if attributeSize < 20 {
		attributeSize = 20
	}
	attrs := make([]manifestAttr, 0, count)
	for i := 0; i < count; i++ {
		base := ext + attributeStart + i*attributeSize
		if base+20 > len(chunk) {
			break
		}
		nameIdx := binary.LittleEndian.Uint32(chunk[base+4 : base+8])
		rawIdx := binary.LittleEndian.Uint32(chunk[base+8 : base+12])
		dataType := chunk[base+15]
		data := binary.LittleEndian.Uint32(chunk[base+16 : base+20])

		name := stringByIndex(pool, nameIdx)
		if name == "" {
			continue
		}
		value := ""
		switch dataType {
		case typeStringData:
			value = stringByIndex(pool, data)
		case typeIntData:
			value = fmt.Sprintf("%d", int32(data))
		case typeBooleanData:
			if data == 0 {
				value = "false"
			} else {
				value = "true"
			}
		default:
			value = stringByIndex(pool, rawIdx)
		}
		if value == "" {
			value = stringByIndex(pool, rawIdx)
		}
		if value != "" {
			attrs = append(attrs, manifestAttr{name: strings.TrimSuffix(name, "Attr"), value: value})
		}
	}
	return attrs
}

// parseStringPool 解析字符串池 chunk。
func parseStringPool(chunk []byte) ([]string, error) {
	if len(chunk) < 28 {
		return nil, errors.New(i18n.T("apkinfo.err.stringPoolShort"))
	}
	// headerSize := binary.LittleEndian.Uint16(chunk[2:4])
	stringCount := int(binary.LittleEndian.Uint32(chunk[8:12]))
	// styleCount := binary.LittleEndian.Uint32(chunk[12:16])
	flags := binary.LittleEndian.Uint32(chunk[16:20])
	stringsStart := int(binary.LittleEndian.Uint32(chunk[20:24]))
	utf8 := flags&0x100 != 0

	pool := make([]string, 0, stringCount)
	for i := 0; i < stringCount && 28+i*4+4 <= len(chunk); i++ {
		rel := int(binary.LittleEndian.Uint32(chunk[28+i*4 : 28+i*4+4]))
		pos := stringsStart + rel
		if pos < 0 || pos >= len(chunk) {
			pool = append(pool, "")
			continue
		}
		var value string
		if utf8 {
			value, _ = decodeUTF8String(chunk, pos)
		} else {
			value, _ = decodeUTF16String(chunk, pos)
		}
		pool = append(pool, value)
	}
	return pool, nil
}

// readLength16 读取 AXML 中长度占用的变长 u16（1 或 2 字节）。
func readLength16(data []byte, pos int) (int, int, bool) {
	if pos >= len(data) {
		return 0, pos, false
	}
	b := int(data[pos])
	if b&0x80 == 0 {
		return b, pos + 1, true
	}
	if pos+1 >= len(data) {
		return 0, pos, false
	}
	return ((b & 0x7f) << 8) | int(data[pos+1]), pos + 2, true
}

// decodeUTF8String 解码 UTF-8 编码的字符串条目（先字符数再字节数）。
func decodeUTF8String(data []byte, pos int) (string, bool) {
	_, p, ok := readLength16(data, pos)
	if !ok {
		return "", false
	}
	byteLen, p, ok := readLength16(data, p)
	if !ok {
		return "", false
	}
	if byteLen <= 0 || p+byteLen > len(data) {
		return "", false
	}
	return string(data[p : p+byteLen]), true
}

// decodeUTF16String 解码 UTF-16LE 编码的字符串条目。
func decodeUTF16String(data []byte, pos int) (string, bool) {
	charLen, p, ok := readLength16(data, pos)
	if !ok {
		return "", false
	}
	byteLen := charLen * 2
	if byteLen <= 0 || p+byteLen > len(data) {
		return "", false
	}
	u16 := make([]uint16, charLen)
	for i := 0; i < charLen; i++ {
		u16[i] = binary.LittleEndian.Uint16(data[p+i*2 : p+i*2+2])
	}
	return string(runeSliceToString(u16)), true
}

// runeSliceToString 将 UTF-16 单元转为字符串。
func runeSliceToString(units []uint16) []rune {
	out := make([]rune, 0, len(units))
	for _, u := range units {
		out = append(out, rune(u))
	}
	return out
}

// stringByIndex 安全获取字符串池中的值。
func stringByIndex(pool []string, idx uint32) string {
	if int(idx) < len(pool) {
		return pool[idx]
	}
	return ""
}

// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

package tools

import (
	"encoding/xml"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"

	"go-apksigner-gui/internal/dto"
	"go-apksigner-gui/internal/i18n"
)

// IndexFileName Google SDK 仓库索引文件名。
const IndexFileName = "repository2-3.xml"

// HostOS 返回当前系统在仓库索引中的标识。
func HostOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "macosx"
	case "linux":
		return "linux"
	default:
		return runtime.GOOS
	}
}

// IndexURL 拼接仓库索引地址，保证以单个斜杠结尾。
func IndexURL(baseURL string) string {
	if baseURL == "" {
		baseURL = DefaultMirror().BaseURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL + IndexFileName
}

// xmlPackage 对应索引中的 remotePackage 节点。
type xmlPackage struct {
	Path     string `xml:"path,attr"`
	Revision struct {
		Major int `xml:"major"`
		Minor int `xml:"minor"`
		Micro int `xml:"micro"`
	} `xml:"revision"`
	Archives xmlArchives `xml:"archives"`
}

// xmlArchives 归档列表。
type xmlArchives struct {
	Archive []xmlArchive `xml:"archive"`
}

// xmlArchive 单个平台归档，兼容属性式与元素式写法。
type xmlArchive struct {
	OSAttr   string `xml:"os,attr"`
	HostOS   string `xml:"host-os,attr"`
	OSElem   string `xml:"os"`
	Size     int64  `xml:"size"`
	Checksum string `xml:"checksum"`
	URL      string `xml:"url"`
	Complete *struct {
		Size     int64  `xml:"size"`
		Checksum string `xml:"checksum"`
		URL      string `xml:"url"`
	} `xml:"complete"`
}

// os 返回归档对应的系统标识。
func (a xmlArchive) os() string {
	if a.HostOS != "" {
		return a.HostOS
	}
	if a.OSAttr != "" {
		return a.OSAttr
	}
	return a.OSElem
}

// ParseRepository 解析仓库索引 XML，返回指定 hostOS 的 build-tools 版本清单。
// 采用流式解码，任何层级的 remotePackage 节点都会被收集。
func ParseRepository(data []byte, baseURL string, hostOS string) ([]dto.BuildToolVersion, error) {
	if hostOS == "" {
		hostOS = HostOS()
	}
	versions := []dto.BuildToolVersion{}
	dec := xml.NewDecoder(strings.NewReader(strings.TrimSpace(string(data))))
	seen := map[string]bool{}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if len(versions) > 0 {
				return versions, nil // 容忍索引中的局部错误
			}
			return nil, fmt.Errorf("%s: %w", i18n.T("tools.err.parseIndex"), err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "remotePackage" {
			continue
		}
		var pkg xmlPackage
		if err := dec.DecodeElement(&pkg, &start); err != nil {
			continue
		}
		if pkg.Path == "" || !strings.HasPrefix(pkg.Path, "build-tools;") {
			continue
		}
		versionName := strings.TrimPrefix(pkg.Path, "build-tools;")
		// 路径版本号不完整时（如 build-tools;36.1）用 revision 补全
		if strings.Count(versionName, ".") < 2 && pkg.Revision.Major > 0 {
			versionName = fmt.Sprintf("%d.%d.%d", pkg.Revision.Major, pkg.Revision.Minor, pkg.Revision.Micro)
		}
		if versionName == "" || seen[versionName] {
			continue
		}
		for _, archive := range pkg.Archives.Archive {
			if !matchesOS(archive.os(), hostOS) {
				continue
			}
			size, checksum, url := archive.Size, archive.Checksum, archive.URL
			if archive.Complete != nil {
				if archive.Complete.Size > 0 {
					size = archive.Complete.Size
				}
				if archive.Complete.Checksum != "" {
					checksum = archive.Complete.Checksum
				}
				if archive.Complete.URL != "" {
					url = archive.Complete.URL
				}
			}
			if url == "" {
				continue
			}
			seen[versionName] = true
			versions = append(versions, dto.BuildToolVersion{
				Version:  versionName,
				FileName: url,
				URL:      resolveURL(baseURL, url),
				SHA256:   checksum,
				Size:     size,
			})
			break
		}
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("tools.err.noBuildToolsInIndex", hostOS))
	}
	SortVersions(versions)
	return versions, nil
}

// matchesOS 判断归档是否适用于目标系统。
func matchesOS(archiveOS, hostOS string) bool {
	archiveOS = strings.ToLower(strings.TrimSpace(archiveOS))
	hostOS = strings.ToLower(strings.TrimSpace(hostOS))
	if archiveOS == "any" {
		return true
	}
	if archiveOS == "" {
		return false // 缺少系统标识的归档无法确认适用性，跳过
	}
	return strings.Contains(archiveOS, hostOS)
}

// resolveURL 拼接下载地址：相对路径拼到镜像根路径上。
func resolveURL(baseURL, file string) string {
	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		return file
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL + strings.TrimPrefix(file, "/")
}

// StaticVersions 索引不可用时的内置版本清单（文件名遵循 Google SDK 命名规则）。
func StaticVersions(hostOS, baseURL string) []dto.BuildToolVersion {
	if hostOS == "" {
		hostOS = HostOS()
	}
	fileOS := hostOS
	if fileOS == "macosx" {
		fileOS = "macosx"
	}
	versions := []string{
		"36.1.0", "36.0.0", "35.0.1", "35.0.0", "34.0.0",
		"33.0.2", "33.0.1", "33.0.0", "32.0.0", "31.0.0", "30.0.3",
	}
	out := make([]dto.BuildToolVersion, 0, len(versions))
	for _, v := range versions {
		name := fmt.Sprintf("build-tools_r%s-%s.zip", v, fileOS)
		out = append(out, dto.BuildToolVersion{
			Version:  v,
			FileName: name,
			URL:      resolveURL(baseURL, name),
		})
	}
	return out
}

// SortVersions 按版本号倒序排列。
func SortVersions(versions []dto.BuildToolVersion) {
	sort.SliceStable(versions, func(i, j int) bool {
		return compareBuildToolsVersion(versions[i].Version, versions[j].Version) > 0
	})
}

package certificate

import (
	"regexp"
	"strings"

	"go-apksigner-gui/internal/dto"
)

// 以下正则同时兼容 keytool 的英文与中文（本地化）输出。
var (
	storeTypePattern = regexp.MustCompile(`(?im)^\s*(?:keystore\s*type|密钥库类型)\s*[:：]\s*(.+?)\s*$`)
	aliasPattern     = regexp.MustCompile(`(?i)^\s*(?:alias\s*name|别名)\s*[:：]\s*(.+?)\s*$`)
	ownerPattern     = regexp.MustCompile(`(?i)^\s*(?:owner|所有者)\s*[:：]\s*(.+?)\s*$`)
	issuerPattern    = regexp.MustCompile(`(?i)^\s*(?:issuer|签发人|发布者)\s*[:：]\s*(.+?)\s*$`)
	serialPattern    = regexp.MustCompile(`(?i)^\s*(?:serial\s*number|序列号)\s*[:：]\s*(.+?)\s*$`)
	createPattern    = regexp.MustCompile(`(?i)^\s*(?:creation\s*date|创建日期)\s*[:：]\s*(.+?)\s*$`)
	entryTypePattern = regexp.MustCompile(`(?i)^\s*(?:entry\s*type|条目类型)\s*[:：]\s*(.+?)\s*$`)
	keyAlgoPattern   = regexp.MustCompile(`(?i)^\s*(?:subject\s*public\s*key\s*algorithm|主体公共密钥算法)\s*[:：]\s*(.+?)\s*$`)
	fingerPattern    = regexp.MustCompile(`(?i)^\s*(MD5|SHA1|SHA256|SHA-1|SHA-256|SHA-512)\s*[:：]\s*(.+?)\s*$`)
	validLinePattern = regexp.MustCompile(`(?i)(valid\s*from|有效期)`)
	// validPrefixPattern 去除有效期行中的引导词与冒号。
	validPrefixPattern = regexp.MustCompile(`(?i)^\s*(valid\s*from|有效期)\s*(为)?\s*[:：]?\s*(从|自)?\s*`)
	countPattern       = regexp.MustCompile(`(?im)(?:keystore\s*contains|您的密钥库包含)\s*(\d+)`)
)

// ParseKeytoolList 解析 keytool -list -v 的输出，兼容中英文。
func ParseKeytoolList(output string) dto.KeystoreInfo {
	info := dto.KeystoreInfo{Entries: []dto.KeystoreEntry{}}
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")

	if m := storeTypePattern.FindStringSubmatch(output); len(m) == 2 {
		info.Type = m[1]
	}

	var current *dto.KeystoreEntry
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if m := aliasPattern.FindStringSubmatch(line); len(m) == 2 {
			if current != nil {
				info.Entries = append(info.Entries, *current)
			}
			current = &dto.KeystoreEntry{Alias: m[1]}
			continue
		}
		if current == nil {
			continue
		}
		switch {
		case ownerPattern.MatchString(line):
			current.Owner = fieldValue(line, ownerPattern)
		case issuerPattern.MatchString(line):
			current.Issuer = fieldValue(line, issuerPattern)
		case serialPattern.MatchString(line):
			current.Serial = fieldValue(line, serialPattern)
		case createPattern.MatchString(line):
			current.CreateDate = fieldValue(line, createPattern)
		case entryTypePattern.MatchString(line):
			current.EntryType = fieldValue(line, entryTypePattern)
		case keyAlgoPattern.MatchString(line):
			current.KeyAlgo = fieldValue(line, keyAlgoPattern)
		case fingerPattern.MatchString(line):
			m := fingerPattern.FindStringSubmatch(line)
			digest := strings.ReplaceAll(strings.ToUpper(m[1]), "-", "")
			value := strings.TrimSpace(m[2])
			switch digest {
			case "MD5":
				current.MD5 = value
			case "SHA1":
				current.SHA1 = value
			case "SHA256":
				current.SHA256 = value
			}
		case validLinePattern.MatchString(line):
			from, to := parseValidity(line)
			if from != "" {
				current.ValidFrom = from
			}
			if to != "" {
				current.ValidTo = to
			}
		}
	}
	if current != nil {
		info.Entries = append(info.Entries, *current)
	}
	return info
}

// fieldValue 提取键值行中的值。
func fieldValue(line string, pattern *regexp.Regexp) string {
	m := pattern.FindStringSubmatch(line)
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// parseValidity 从有效期行中提取起止时间，支持英文 until 与中文「至」两种分隔。
func parseValidity(line string) (string, string) {
	value := line
	if idx := strings.IndexAny(line, ":"); idx >= 0 && strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "valid") {
		value = strings.TrimSpace(line[idx+1:])
	}
	sepIdx := -1
	for _, sep := range []string{" until ", " 至 ", "至", "until"} {
		if idx := strings.Index(strings.ToLower(value), strings.ToLower(sep)); idx > 0 {
			sepIdx = idx
			break
		}
	}
	if sepIdx < 0 {
		return strings.TrimSpace(value), ""
	}
	sep := value[sepIdx:]
	return cleanValidPart(value[:sepIdx], true), cleanValidPart(sep, false)
}

// cleanValidPart 清理日期片段中的引导词（英文 Valid from / 中文 有效期…从）。
func cleanValidPart(s string, leading bool) string {
	s = strings.TrimSpace(s)
	if leading {
		s = validPrefixPattern.ReplaceAllString(s, "")
	}
	if !leading {
		s = strings.TrimSpace(s)
		// 反复去掉分隔符本身（英文 until / 中文 至、到）与冒号
		cuts := []string{"until", "至", "到", ":", "：", "从", "自"}
		for changed := true; changed && s != ""; {
			changed = false
			s = strings.TrimSpace(s)
			for _, cut := range cuts {
				if strings.HasPrefix(strings.ToLower(s), cut) {
					s = strings.TrimSpace(s[len(cut):])
					changed = true
					break
				}
			}
		}
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), ":"))
}

// CountEntries 提取 "keystore contains N entries" 中的条目数，用于校验解析完整性。
func CountEntries(output string) int {
	m := countPattern.FindStringSubmatch(output)
	if len(m) != 2 {
		return -1
	}
	n := 0
	for _, r := range m[1] {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	return n
}

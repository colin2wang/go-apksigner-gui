// Package dto 定义前后端统一的数据契约（JSON 序列化的唯一来源）。
package dto

// TaskResult 通用任务结果，前端仅依据 Success 分支渲染，Detail 用于日志抽屉展开。
type TaskResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// ToolStatus 外部工具（apksigner / zipalign / aapt2 / keytool / apktool）的可用状态。
type ToolStatus struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Path      string `json:"path"`
	Installed bool   `json:"installed"`
	Required  bool   `json:"required"`
	Source    string `json:"source"` // tools（工具目录）| path（系统 PATH）| missing
}

// Progress 长时间任务（下载 / 校验 / 解压）的进度事件。
type Progress struct {
	FileName   string  `json:"fileName"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Percent    float64 `json:"percent"`
	Stage      string  `json:"stage"` // downloading|verifying|extracting|done|error
	Message    string  `json:"message,omitempty"`
}

// Step 流水线步骤状态事件。
type Step struct {
	Name   string `json:"name"`
	Status string `json:"status"` // pending|running|success|error
	Detail string `json:"detail,omitempty"`
}

// MirrorSource 可选的 build-tools 下载镜像源。
type MirrorSource struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Note    string `json:"note"`
}

// ProxyConfig 下载所用代理配置。
type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"` // http|https|socks5
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// BuildToolVersion 可下载的 build-tools 版本。
type BuildToolVersion struct {
	Version  string `json:"version"`
	FileName string `json:"fileName"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

// SignerCert 签名者证书摘要信息。
type SignerCert struct {
	DN     string `json:"dn"`
	MD5    string `json:"md5"`
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
}

// KeystoreEntry 密钥库中单个条目的证书信息。
type KeystoreEntry struct {
	Alias      string `json:"alias"`
	Owner      string `json:"owner"`
	Issuer     string `json:"issuer"`
	Serial     string `json:"serial"`
	ValidFrom  string `json:"validFrom"`
	ValidTo    string `json:"validTo"`
	MD5        string `json:"md5"`
	SHA1       string `json:"sha1"`
	SHA256     string `json:"sha256"`
	KeyAlgo    string `json:"keyAlgo"`
	StoreType  string `json:"storeType"`
	EntryType  string `json:"entryType"`
	CreateDate string `json:"createDate"`
}

// KeystoreInfo 密钥库整体信息。
type KeystoreInfo struct {
	Path    string          `json:"path"`
	Type    string          `json:"type"`
	Entries []KeystoreEntry `json:"entries"`
	Raw     string          `json:"raw,omitempty"`
}

// CertInfo 生成密钥库时填写的证书信息。
type CertInfo struct {
	Alias        string `json:"alias"`
	CommonName   string `json:"commonName"`
	Organization string `json:"organization"`
	OrgUnit      string `json:"orgUnit"`
	Locality     string `json:"locality"`
	State        string `json:"state"`
	Country      string `json:"country"`
	Validity     int    `json:"validity"`
	KeyAlgorithm string `json:"keyAlgorithm"` // RSA|EC|DSA
	KeySize      int    `json:"keySize"`
}

// KeystoreRequest 生成密钥库的完整入参。
type KeystoreRequest struct {
	OutputPath string   `json:"outputPath"`
	StoreType  string   `json:"storeType"` // PKCS12|JKS
	StorePass  string   `json:"storePass"`
	KeyPass    string   `json:"keyPass"`
	Cert       CertInfo `json:"cert"`
}

// SignOptions 签名选项。
type SignOptions struct {
	InputAPK          string `json:"inputApk"`
	OutputAPK         string `json:"outputApk"`
	Keystore          string `json:"keystore"`
	Alias             string `json:"alias"`
	StorePass         string `json:"storePass"`
	KeyPass           string `json:"keyPass"`
	V1Enabled         bool   `json:"v1Enabled"`
	V2Enabled         bool   `json:"v2Enabled"`
	V3Enabled         bool   `json:"v3Enabled"`
	V4Enabled         bool   `json:"v4Enabled"`
	ZipAlignFirst     bool   `json:"zipAlignFirst"`
	RemoveOldSign     bool   `json:"removeOldSign"`
	MinSDK            int    `json:"minSdk"`
	MaxSDK            int    `json:"maxSdk"`
	DebuggableApkPerm bool   `json:"debuggableApkPerm"`
}

// VerifyResult apksigner verify 的结构化结果。
type VerifyResult struct {
	Verified  bool         `json:"verified"`
	V1Signed  bool         `json:"v1Signed"`
	V2Signed  bool         `json:"v2Signed"`
	V3Signed  bool         `json:"v3Signed"`
	V4Signed  bool         `json:"v4Signed"`
	Certs     []SignerCert `json:"certs"`
	RawOutput string       `json:"rawOutput"`
	Warnings  []string     `json:"warnings"`
}

// SignResult 签名流水线结果。
type SignResult struct {
	Success     bool          `json:"success"`
	OutputPath  string        `json:"outputPath"`
	Message     string        `json:"message"`
	SignVersion string        `json:"signVersion"`
	FileSize    int64         `json:"fileSize"`
	Verify      *VerifyResult `json:"verify,omitempty"`
}

// APKInfo APK 基础信息。
type APKInfo struct {
	Path        string `json:"path"`
	PackageName string `json:"packageName"`
	VersionName string `json:"versionName"`
	VersionCode string `json:"versionCode"`
	MinSDK      string `json:"minSdk"`
	TargetSDK   string `json:"targetSdk"`
	Label       string `json:"label"`
	FileSize    int64  `json:"fileSize"`
	IsSigned    bool   `json:"isSigned"`
	Source      string `json:"source"` // aapt2|axml|unknown
	Message     string `json:"message,omitempty"`
	RawOutput   string `json:"rawOutput,omitempty"`
}

// KeystoreFile 密钥库目录中的文件摘要（不含密码与内容）。
type KeystoreFile struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"` // Unix 毫秒
}

// AppSettings 暴露给前端的全局设置。
type AppSettings struct {
	ToolsDir     string            `json:"toolsDir"`
	ConfigPath   string            `json:"configPath"`
	LastKeystore string            `json:"lastKeystore"`
	LastAlias    string            `json:"lastAlias"`
	LastApk      string            `json:"lastApk"`
	DefaultV1    bool              `json:"defaultV1"`
	DefaultV2    bool              `json:"defaultV2"`
	DefaultV3    bool              `json:"defaultV3"`
	DefaultV4    bool              `json:"defaultV4"`
	RecentFiles  []string          `json:"recentFiles"`
	RecentAlias  []string          `json:"recentAliases"`
	Mirror       MirrorSource      `json:"mirror"`
	Proxy        ProxyConfig       `json:"proxy"`
	AndroidSdk   string            `json:"androidSdk"`
	Language     string            `json:"language"`
	Tools        map[string]string `json:"tools,omitempty"`
}

// AppMeta 应用元信息。
type AppMeta struct {
	Name                     string `json:"name"`
	Version                  string `json:"version"`
	GoVersion                string `json:"goVersion"`
	Platform                 string `json:"platform"`
	ConfigPath               string `json:"configPath"`
	ToolsDir                 string `json:"toolsDir"`
	DefaultBuildToolsVersion string `json:"defaultBuildToolsVersion"`
}

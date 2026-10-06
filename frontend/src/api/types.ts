// 前后端契约类型：与 internal/dto 保持一致（JSON 字段名一致即可）
export interface TaskResult {
  success: boolean
  message: string
  detail?: string
}

export interface ToolStatus {
  name: string
  version: string
  path: string
  installed: boolean
  required: boolean
  source: string
}

export interface Progress {
  fileName: string
  downloaded: number
  total: number
  percent: number
  stage: string
  message?: string
}

export interface Step {
  name: string
  status: 'pending' | 'running' | 'success' | 'error'
  detail?: string
}

export interface MirrorSource {
  name: string
  baseUrl: string
  note?: string
}

export interface ProxyConfig {
  enabled: boolean
  type: string
  host: string
  port: number
  username: string
  password: string
}

export interface BuildToolVersion {
  version: string
  fileName: string
  url: string
  sha256: string
  size: number
}

export interface SignerCert {
  dn: string
  md5: string
  sha1: string
  sha256: string
}

export interface KeystoreEntry {
  alias: string
  owner: string
  issuer: string
  serial: string
  validFrom: string
  validTo: string
  md5: string
  sha1: string
  sha256: string
  keyAlgo: string
  storeType: string
  entryType: string
  createDate: string
}

export interface KeystoreInfo {
  path: string
  type: string
  entries: KeystoreEntry[]
  raw?: string
}

export interface KeystoreFile {
  path: string
  name: string
  size: number
  modTime: number
}

export interface CertInfo {
  alias: string
  commonName: string
  organization: string
  orgUnit: string
  locality: string
  state: string
  country: string
  validity: number
  keyAlgorithm: string
  keySize: number
}

export interface KeystoreRequest {
  outputPath: string
  storeType: string
  storePass: string
  keyPass: string
  cert: CertInfo
}

export interface SignOptions {
  inputApk: string
  outputApk: string
  keystore: string
  alias: string
  storePass: string
  keyPass: string
  v1Enabled: boolean
  v2Enabled: boolean
  v3Enabled: boolean
  v4Enabled: boolean
  zipAlignFirst: boolean
  removeOldSign: boolean
  minSdk: number
  maxSdk: number
  debuggableApkPerm: boolean
}

export interface VerifyResult {
  verified: boolean
  v1Signed: boolean
  v2Signed: boolean
  v3Signed: boolean
  v4Signed: boolean
  certs: SignerCert[]
  rawOutput: string
  warnings: string[]
}

export interface SignResult {
  success: boolean
  outputPath: string
  message: string
  signVersion: string
  fileSize: number
  verify?: VerifyResult
}

export interface APKInfo {
  path: string
  packageName: string
  versionName: string
  versionCode: string
  minSdk: string
  targetSdk: string
  label: string
  fileSize: number
  isSigned: boolean
  source: string
  message?: string
}

export interface AppSettings {
  toolsDir: string
  configPath: string
  lastKeystore: string
  lastAlias: string
  lastApk: string
  defaultV1: boolean
  defaultV2: boolean
  defaultV3: boolean
  defaultV4: boolean
  recentFiles: string[]
  recentAliases: string[]
  mirror: MirrorSource
  proxy: ProxyConfig
  androidSdk: string
  language?: string
  tools?: Record<string, string>
}

export interface AppMeta {
  name: string
  version: string
  goVersion: string
  platform: string
  configPath: string
  toolsDir: string
  defaultBuildToolsVersion: string
}

export interface LogEntry {
  time: number
  level: string
  message: string
  attrs?: string
}

export interface LogLine {
  time: number
  level: 'INFO' | 'WARN' | 'ERROR' | 'DEBUG' | 'STDOUT' | 'STDERR'
  text: string
}

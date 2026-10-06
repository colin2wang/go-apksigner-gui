import * as bindings from '../../wailsjs/go/main/App'
import { EventsOff, EventsOn } from '../../wailsjs/runtime'
import type {
  AppMeta,
  AppSettings,
  APKInfo,
  BuildToolVersion,
  KeystoreFile,
  KeystoreInfo,
  KeystoreRequest,
  MirrorSource,
  ProxyConfig,
  SignOptions,
  SignResult,
  TaskResult,
  ToolStatus,
  VerifyResult,
  LogEntry,
} from './types'

// 事件名常量：与 app.go 中的 EventsEmit 保持一致
export const EventNames = {
  downloadProgress: 'download:progress',
  taskStep: 'task:step',
  taskLog: 'task:log',
  toolsChanged: 'tools:changed',
}

export const api = {
  getAppMeta: () => bindings.GetAppMeta() as unknown as Promise<AppMeta>,
  getTools: () => bindings.GetTools() as unknown as Promise<ToolStatus[]>,
  getMirrors: () => bindings.GetMirrors() as unknown as Promise<MirrorSource[]>,
  getSettings: () => bindings.GetSettings() as unknown as Promise<AppSettings>,
  saveSettings: (settings: AppSettings) => bindings.SaveSettings(settings as never) as unknown as Promise<void>,
  recentLogs: (n: number) => bindings.RecentLogs(n) as unknown as Promise<LogEntry[]>,

  selectOpenFile: (filter?: string) => bindings.SelectOpenFile(filter ?? ''),
  selectSaveFile: (defaultName: string, filter?: string) => bindings.SelectSaveFile(defaultName, filter ?? ''),
  selectDirectory: (title: string) => bindings.SelectDirectory(title),

  fetchVersions: () => bindings.FetchBuildToolVersions() as unknown as Promise<BuildToolVersion[]>,
  installBuildTools: (version: string) => bindings.InstallBuildTools(version) as unknown as Promise<TaskResult>,
  cancelInstall: (version: string) => bindings.CancelInstall(version),
  cleanTemp: () => bindings.CleanTemp() as unknown as Promise<TaskResult>,

  // 通过当前（未保存也可）代理配置测试到 Google 主页的连通性。
  testProxy: (proxy: ProxyConfig) => bindings.TestProxy(proxy as never) as unknown as Promise<TaskResult>,

  generateKeystore: (req: KeystoreRequest) => bindings.GenerateKeystore(req as never) as unknown as Promise<TaskResult>,
  listKeystore: (path: string, pass: string) => bindings.ListKeystore(path, pass) as unknown as Promise<KeystoreInfo>,
  listAliases: (path: string, pass: string) => bindings.ListAliases(path, pass) as Promise<string[]>,
  exportCertificate: (ks: string, pass: string, alias: string, out: string) =>
    bindings.ExportCertificate(ks, pass, alias, out) as unknown as Promise<TaskResult>,
  convertKeystore: (src: string, srcPass: string, dst: string, dstPass: string, type: string) =>
    bindings.ConvertKeystore(src, srcPass, dst, dstPass, type) as unknown as Promise<TaskResult>,
  deleteAlias: (path: string, pass: string, alias: string) =>
    bindings.DeleteAlias(path, pass, alias) as unknown as Promise<TaskResult>,

  keystoreDir: () => bindings.KeystoreDir() as unknown as Promise<string>,
  listKeystoreDir: () => bindings.ListKeystoreDir() as unknown as Promise<KeystoreFile[]>,
  importKeystore: (src: string, pass: string) =>
    bindings.ImportKeystore(src, pass) as unknown as Promise<TaskResult>,
  getSavedPassword: (path: string) => bindings.GetSavedPassword(path) as unknown as Promise<string>,
  savePassword: (path: string, pass: string) => bindings.SavePassword(path, pass) as unknown as Promise<void>,
  forgetPassword: (path: string) => bindings.ForgetPassword(path) as unknown as Promise<void>,

  signAPK: (opts: SignOptions) => bindings.SignAPK(opts as never) as unknown as Promise<SignResult>,
  verifyAPK: (path: string) => bindings.VerifyAPK(path) as unknown as Promise<VerifyResult>,
  zipAlign: (input: string, output: string) => bindings.ZipAlign(input, output) as unknown as Promise<TaskResult>,

  parseAPK: (path: string) => bindings.ParseAPK(path) as unknown as Promise<APKInfo>,
}

// on 订阅后端事件，返回取消订阅函数。
export function on<T>(name: string, callback: (data: T) => void): () => void {
  EventsOn(name, callback as never)
  return () => EventsOff(name)
}

// pickFile 选择文件并在用户取消时返回空串。
export async function pickFile(filter?: string): Promise<string> {
  try {
    const path = await api.selectOpenFile(filter)
    return path ?? ''
  } catch {
    return ''
  }
}

import { reactive, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { api, EventNames, on } from '../api'
import type {
  AppMeta,
  AppSettings,
  BuildToolVersion,
  LogLine,
  MirrorSource,
  Progress,
  Step,
  ToolStatus,
} from '../api/types'

const emptySettings: AppSettings = {
  toolsDir: '',
  configPath: '',
  lastKeystore: '',
  lastAlias: '',
  lastApk: '',
  defaultV1: true,
  defaultV2: true,
  defaultV3: true,
  defaultV4: false,
  recentFiles: [],
  recentAliases: [],
  mirror: { name: '', baseUrl: '' },
  proxy: { enabled: false, type: 'http', host: '', port: 0, username: '', password: '' },
  androidSdk: '',
  tools: {},
}

export const state = reactive({
  meta: null as AppMeta | null,
  tools: [] as ToolStatus[],
  mirrors: [] as MirrorSource[],
  versions: [] as BuildToolVersion[],
  settings: emptySettings as AppSettings,
  loadingTools: false,
  loadingVersions: false,
  installing: false,
  signing: false,
  progress: null as Progress | null,
  steps: [] as Step[],
  logs: [] as LogLine[],
  busy: false,
})

export const missingTools = computed(() => state.tools.filter((t) => !t.installed))
export const requiredMissing = computed(() => state.tools.filter((t) => t.required && !t.installed))

export function log(level: LogLine['level'], text: string) {
  state.logs.push({ time: Date.now(), level, text })
  if (state.logs.length > 800) {
    state.logs.splice(0, state.logs.length - 800)
  }
}

export function notify(type: 'success' | 'error' | 'warning' | 'info', message: string) {
  MessagePlugin[type]({ content: message, placement: 'top-center', duration: type === 'error' ? 6000 : 3000 })
  log(type === 'error' ? 'ERROR' : type === 'warning' ? 'WARN' : 'INFO', message)
}

export function clearLogs() {
  state.logs.splice(0, state.logs.length)
}

export async function refreshTools() {
  state.loadingTools = true
  try {
    state.tools = await api.getTools()
  } catch (err) {
    notify('error', `工具检测失败: ${err}`)
  } finally {
    state.loadingTools = false
  }
}

export async function loadVersions() {
  state.loadingVersions = true
  try {
    state.versions = await api.fetchVersions()
  } catch (err) {
    notify('error', `获取版本列表失败: ${err}`)
  } finally {
    state.loadingVersions = false
  }
}

export async function initStore() {
  try {
    state.meta = await api.getAppMeta()
  } catch (err) {
    console.error(err)
  }
  state.settings = { ...emptySettings, ...(await api.getSettings()) }
  state.mirrors = await api.getMirrors()
  if (!state.settings.recentFiles) state.settings.recentFiles = []
  if (!state.settings.recentAliases) state.settings.recentAliases = []
  await refreshTools()

  subscribeEvents()
  const entries = await api.recentLogs(200).catch(() => [])
  entries.forEach((e) => log((e.level as LogLine['level']) ?? 'INFO', e.attrs ? `${e.message} ${e.attrs}` : e.message))
}

// subscribeEvents 订阅后端的进度 / 步骤 / 日志推送。
function subscribeEvents() {
  on<Progress>(EventNames.downloadProgress, (p) => {
    state.progress = p
    if (p.message) log('INFO', p.message)
  })
  on<Step>(EventNames.taskStep, (s) => {
    const exist = state.steps.findIndex((item) => item.name === s.name)
    if (exist >= 0) state.steps.splice(exist, 1, s)
    else state.steps.push(s)
    if (s.detail) log(s.status === 'error' ? 'ERROR' : 'INFO', `${s.name}: ${s.detail}`)
  })
  on<{ stream: string; text: string }>(EventNames.taskLog, (payload) => {
    payload.text.split('\n').forEach((line) => {
      if (line.trim()) log(payload.stream === 'stderr' ? 'STDERR' : 'STDOUT', line)
    })
  })
  on<ToolStatus[]>(EventNames.toolsChanged, (list) => {
    state.tools = list
  })
}

export async function saveSettings() {
  await api.saveSettings(state.settings)
  notify('success', '设置已保存')
  await refreshTools()
}

// formatSize 格式化文件体积。
export function formatSize(bytes: number): string {
  if (!bytes) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let value = bytes
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  return `${value.toFixed(value >= 10 || i === 0 ? 0 : 1)} ${units[i]}`
}

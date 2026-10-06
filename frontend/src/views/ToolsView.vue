<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '../api'
import { formatSize, loadVersions, notify, refreshTools, state } from '../stores/app'

const selectedVersion = ref('')
const installing = computed(() => state.installing)
const progress = computed(() => state.progress)
const stageText = computed(() => {
  switch (state.progress?.stage) {
    case 'downloading':
      return '下载中'
    case 'verifying':
      return '完整性校验'
    case 'extracting':
      return '解压安装'
    case 'done':
      return '完成'
    case 'error':
      return '失败'
    default:
      return '待开始'
  }
})

const toolIcons: Record<string, string> = {
  apksigner: '签名',
  zipalign: '对齐',
  aapt2: '解析',
  keytool: '证书',
}

async function refresh() {
  await loadVersions()
  if (!selectedVersion.value && state.meta?.defaultBuildToolsVersion) {
    selectedVersion.value = state.meta.defaultBuildToolsVersion
  }
  notify('info', `已获取 ${state.versions.length} 个可用版本`)
}

async function install(version: string) {
  if (!version) {
    notify('warning', '请先选择一个版本')
    return
  }
  state.installing = true
  state.progress = { fileName: version, downloaded: 0, total: 0, percent: 0, stage: 'downloading', message: '准备下载' }
  try {
    const res = await api.installBuildTools(version)
    if (res.success) {
      notify('success', res.message)
      await refreshTools()
    } else {
      notify('error', res.message)
    }
  } catch (err) {
    notify('error', `安装失败: ${err}`)
  } finally {
    state.installing = false
  }
}

function cancel() {
  if (!selectedVersion.value) return
  api.cancelInstall(selectedVersion.value)
  notify('warning', '已发送取消请求')
}
</script>

<template>
  <div class="space-y-5 animate-fade-up">
    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        工具状态
        <span class="ml-auto text-xs font-normal text-muted">
          {{ state.loadingTools ? '检测中…' : `已就绪 ${state.tools.filter((t) => t.installed).length}/${state.tools.length}` }}
        </span>
      </div>

      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="tool in state.tools"
          :key="tool.name"
          class="rounded-xl border p-4 transition"
          :class="tool.installed ? 'border-brand/25 bg-brand/5' : 'border-white/5 bg-white/[0.02]'"
        >
          <div class="mb-2 flex items-center justify-between">
            <span class="font-mono text-sm text-white">{{ tool.name }}</span>
            <span
              class="chip"
              :class="tool.installed ? 'bg-brand/15 text-brand-light' : 'bg-red-500/15 text-red-300'"
            >
              {{ tool.installed ? '可用' : '缺失' }}
            </span>
          </div>
          <p class="text-[11px] uppercase tracking-wider text-muted">{{ toolIcons[tool.name] ?? '工具' }}</p>
          <p class="mt-2 truncate text-[11px] text-slate-400" :title="tool.path">{{ tool.path || '未找到可执行文件' }}</p>
          <p class="mt-1 text-[11px] text-slate-500">
            {{ tool.version || '-' }} · 来源 {{ tool.source || '-' }}
          </p>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-sky-400"></span>
        下载 build-tools
        <div class="ml-auto flex items-center gap-2">
          <button class="btn-ghost !py-1.5 !text-xs" :disabled="state.loadingVersions" @click="refresh">
            {{ state.loadingVersions ? '获取中…' : '刷新版本列表' }}
          </button>
        </div>
      </div>

      <p class="mb-3 text-xs text-muted">
        当前镜像源：
        <span class="text-brand-light">{{ state.settings.mirror?.name || '未设置' }}</span>
        （可在「设置」页切换国内镜像与代理）
      </p>

      <div class="max-h-72 overflow-y-auto rounded-xl border border-white/5">
        <table class="w-full text-sm">
          <thead class="sticky top-0 bg-ink-900/90 text-xs uppercase tracking-wider text-muted backdrop-blur">
            <tr>
              <th class="px-4 py-2 text-left">版本</th>
              <th class="px-4 py-2 text-left">文件名</th>
              <th class="px-4 py-2 text-right">体积</th>
              <th class="px-4 py-2 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="v in state.versions"
              :key="v.version"
              class="border-t border-white/5 transition hover:bg-white/[0.03]"
            >
              <td class="px-4 py-2 font-mono text-slate-100">{{ v.version }}</td>
              <td class="px-4 py-2 text-xs text-slate-400">{{ v.fileName }}</td>
              <td class="px-4 py-2 text-right text-xs text-slate-400">{{ v.size ? formatSize(v.size) : '-' }}</td>
              <td class="px-4 py-2 text-right">
                <button
                  class="btn-primary !py-1 !text-xs"
                  :disabled="installing"
                  @click="
                    selectedVersion = v.version;
                    install(v.version)
                  "
                >
                  安装
                </button>
              </td>
            </tr>
            <tr v-if="!state.versions.length">
              <td colspan="4" class="px-4 py-6 text-center text-xs text-muted">
                暂无数据，点击「刷新版本列表」获取
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="progress" class="mt-4 rounded-xl border border-white/5 bg-ink-900/50 p-4">
        <div class="mb-2 flex items-center justify-between text-xs">
          <span class="text-slate-300">
            {{ stageText }} · {{ progress.fileName }}
            <span v-if="progress.message" class="ml-2 text-muted">{{ progress.message }}</span>
          </span>
          <span class="font-mono text-brand-light">{{ progress.percent.toFixed(1) }}%</span>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-white/5">
          <div
            class="h-full rounded-full bg-gradient-to-r from-brand to-brand-light transition-all duration-300"
            :style="{ width: `${Math.min(100, Math.max(2, progress.percent))}%` }"
            :class="progress.stage === 'error' ? 'from-red-500 to-red-400' : ''"
          ></div>
        </div>
        <div class="mt-2 flex items-center justify-between text-[11px] text-muted">
          <span>{{ formatSize(progress.downloaded) }} / {{ formatSize(progress.total) }}</span>
          <button v-if="installing" class="btn-ghost !py-1 !text-[11px]" @click="cancel">取消下载</button>
        </div>
      </div>
    </section>
  </div>
</template>

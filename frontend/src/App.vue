<!--
  Author: colin2wang (colin2wang@gmail.com)
  Date: 2026-10-06
-->

<script setup lang="ts">
import { computed, nextTick, ref, watch, type Component } from 'vue'
import ToolsView from './views/ToolsView.vue'
import CertView from './views/CertView.vue'
import SignView from './views/SignView.vue'
import ApkInfoView from './views/ApkInfoView.vue'
import SettingsView from './views/SettingsView.vue'
import { clearLogs, missingTools, notify, state } from './stores/app'
import { t } from './i18n'

type TabKey = 'tools' | 'cert' | 'sign' | 'apk' | 'settings'

interface NavItem {
  key: TabKey
  label: string
  desc: string
  icon: string
}

const navItems = computed<NavItem[]>(() => [
  { key: 'tools', label: t('nav.tools'), desc: 'build-tools', icon: 'M12 3l8 4.5v9L12 21l-8-4.5v-9L12 3z' },
  { key: 'cert', label: t('nav.cert'), desc: 'keytool', icon: 'M7 4h10v16H7zM9 8h6M9 12h6' },
  { key: 'sign', label: t('nav.sign'), desc: 'apksigner', icon: 'M5 12l5 5L19 7' },
  { key: 'apk', label: t('nav.apk'), desc: 'aapt2', icon: 'M6 3h9l4 4v14H6zM15 3v4h4' },
  { key: 'settings', label: t('nav.settings'), desc: 'preferences', icon: 'M12 15a3 3 0 100-6 3 3 0 000 6zM19 12a7 7 0 00-.1-1.2l2-1.5-2-3.4-2.3 1a7 7 0 00-2-1.2L14.2 3H9.8l-.4 2.7a7 7 0 00-2 1.2l-2.3-1-2 3.4 2 1.5a7 7 0 000 2.4l-2 1.5 2 3.4 2.3-1a7 7 0 002 1.2l.4 2.7h4.4l.4-2.7a7 7 0 002-1.2l2.3 1 2-3.4-2-1.5c.06-.4.1-.8.1-1.2z' },
])

const views: Record<TabKey, Component> = {
  tools: ToolsView,
  cert: CertView,
  sign: SignView,
  apk: ApkInfoView,
  settings: SettingsView,
}

const active = ref<TabKey>('tools')
const currentView = computed(() => views[active.value])
const logOpen = ref(true)
const autoScroll = ref(true)
const logBody = ref<HTMLElement | null>(null)

const running = computed(() => state.signing || state.installing || state.busy)

watch(
  () => state.logs.length,
  async () => {
    if (!autoScroll.value || !logOpen.value) return
    await nextTick()
    if (logBody.value) logBody.value.scrollTop = logBody.value.scrollHeight
  },
)

function levelClass(level: string) {
  switch (level) {
    case 'ERROR':
      return 'text-red-400'
    case 'WARN':
      return 'text-amber-300'
    case 'DEBUG':
      return 'text-sky-300'
    case 'STDERR':
      return 'text-orange-300'
    default:
      return 'text-emerald-300'
  }
}

function timeText(ts: number) {
  const d = new Date(ts)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function handleClear() {
  clearLogs()
  notify('info', t('app.logsCleared'))
}
</script>

<template>
  <div class="flex h-screen flex-col bg-transparent">
    <header
      class="flex items-center justify-between border-b border-white/5 bg-ink-900/50 px-6 py-3 backdrop-blur-xl"
    >
      <div class="flex items-center gap-3">
        <div class="grid h-9 w-9 place-items-center rounded-xl bg-gradient-to-br from-brand to-brand-dark shadow-glow">
          <svg viewBox="0 0 24 24" class="h-5 w-5 text-ink-900" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M5 12l5 5L19 7" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </div>
        <div>
          <h1 class="text-lg font-semibold tracking-wide text-white">APK Signer Studio</h1>
          <p class="text-xs text-muted">
            {{ state.meta?.platform }} · {{ state.meta?.version }} · {{ state.meta?.goVersion }}
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <span v-if="running" class="chip bg-brand/15 text-brand-light">
          <span class="h-2 w-2 animate-ping rounded-full bg-brand"></span> {{ t('app.taskRunning') }}
        </span>
        <span v-if="missingTools.length" class="chip bg-amber-400/15 text-amber-300">
          {{ t('app.missingTools', missingTools.length) }}
        </span>
        <span v-else class="chip bg-brand/15 text-brand-light">{{ t('app.toolsReady') }}</span>
      </div>
    </header>

    <div class="flex min-h-0 flex-1">
      <nav class="w-60 shrink-0 space-y-1 border-r border-white/5 bg-ink-900/30 p-3">
        <button
          v-for="item in navItems"
          :key="item.key"
          class="nav-item w-full"
          :class="active === item.key && 'nav-item-active'"
          @click="active = item.key"
        >
          <svg viewBox="0 0 24 24" class="h-4 w-4 opacity-80" fill="none" stroke="currentColor" stroke-width="1.6">
            <path :d="item.icon" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span class="flex-1 text-left">{{ item.label }}</span>
          <span class="text-[10px] uppercase tracking-wider text-muted">{{ item.desc }}</span>
        </button>

        <div class="mt-6 rounded-xl border border-white/5 bg-white/[0.03] p-3 text-xs text-muted">
          <p class="mb-2 font-medium text-slate-200">{{ t('app.toolsDir') }}</p>
          <p class="kbd-text">{{ state.meta?.toolsDir }}</p>
        </div>
      </nav>

      <main class="min-w-0 flex-1 overflow-y-auto px-6 py-5">
        <Transition name="fade-slide" mode="out-in">
          <component :is="currentView" :key="active" />
        </Transition>
      </main>
    </div>

    <footer class="border-t border-white/5 bg-ink-900/60 backdrop-blur-xl">
      <div class="flex items-center justify-between px-6 py-2">
        <button class="btn-ghost !py-1 !text-xs" @click="logOpen = !logOpen">
          <span>{{ logOpen ? t('app.collapseLog') : t('app.expandLog') }}</span>
          <span class="text-[10px] text-muted">{{ state.logs.length }} {{ t('app.logLines') }}</span>
        </button>
        <div class="flex items-center gap-3 text-xs text-muted">
          <label class="flex items-center gap-1.5">
            <input v-model="autoScroll" type="checkbox" class="accent-brand" />
            {{ t('app.autoScroll') }}
          </label>
          <button class="rounded-lg px-2 py-1 transition hover:bg-white/5" @click="handleClear">{{ t('app.clear') }}</button>
        </div>
      </div>
      <div
        v-show="logOpen"
        ref="logBody"
        class="h-44 space-y-0.5 overflow-y-auto border-t border-white/5 bg-black/30 px-6 py-3 font-mono text-[11px] leading-relaxed"
      >
        <p v-for="(line, idx) in state.logs" :key="idx" class="flex gap-3">
          <span class="shrink-0 text-slate-500">{{ timeText(line.time) }}</span>
          <span class="w-16 shrink-0" :class="levelClass(line.level)">{{ line.level }}</span>
          <span class="whitespace-pre-wrap break-all text-slate-300">{{ line.text }}</span>
        </p>
        <p v-if="!state.logs.length" class="text-slate-500">{{ t('app.noLogs') }}</p>
      </div>
    </footer>
  </div>
</template>

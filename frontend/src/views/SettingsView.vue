<script setup lang="ts">
import { reactive, ref } from 'vue'
import { api } from '../api'
import { t, setLocale, locales, type Locale } from '../i18n'
import { notify, refreshTools, saveSettings, state } from '../stores/app'
import type { ToolStatus } from '../api/types'

const dirty = ref(false)
const toolPaths = reactive<Record<string, string>>(state.settings.tools ?? {})

function markDirty() {
  dirty.value = true
}

// 默认签名方案：用显式读写方法避免模板中出现类型断言
const schemeKeys = ['defaultV1', 'defaultV2', 'defaultV3', 'defaultV4'] as const
type SchemeKey = (typeof schemeKeys)[number]
const schemeItems: { key: SchemeKey; label: string }[] = [
  { key: 'defaultV1', label: 'V1' },
  { key: 'defaultV2', label: 'V2' },
  { key: 'defaultV3', label: 'V3' },
  { key: 'defaultV4', label: 'V4' },
]
const defaultScheme = (key: SchemeKey) => state.settings[key]
function setDefaultScheme(key: SchemeKey, value: boolean) {
  state.settings[key] = value
  markDirty()
}

async function chooseToolsDir() {
  try {
    const dir = await api.selectDirectory(t('settings.toolsDirTitle'))
    if (dir) {
      state.settings.toolsDir = dir
      markDirty()
    }
  } catch {
    /* 取消 */
  }
}

async function chooseSdkDir() {
  try {
    const dir = await api.selectDirectory(t('settings.sdkDirTitle'))
    if (dir) {
      state.settings.androidSdk = dir
      markDirty()
    }
  } catch {
    /* 取消 */
  }
}

async function pickToolPath(tool: ToolStatus) {
  try {
    const path = await api.selectOpenFile(tool.name === 'keytool' ? '*.exe;*' : '*')
    if (path) {
      toolPaths[tool.name] = path
      state.settings.tools = { ...toolPaths }
      markDirty()
    }
  } catch {
    /* 取消 */
  }
}

async function apply() {
  state.settings.tools = { ...toolPaths }
  await saveSettings()
  dirty.value = false
}

async function cleanTemp() {
  const res = await api.cleanTemp()
  notify(res.success ? 'success' : 'error', res.message)
}

// 测试代理连通性（直接对当前表单值，无需先保存）
const testingProxy = ref(false)
const proxyTest = ref<{ ok: boolean; msg: string } | null>(null)

async function testProxy() {
  proxyTest.value = null
  testingProxy.value = true
  try {
    const res = await api.testProxy(state.settings.proxy)
    proxyTest.value = { ok: res.success, msg: res.message }
    notify(res.success ? 'success' : 'error', res.message)
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err)
    proxyTest.value = { ok: false, msg }
    notify('error', t('settings.err.proxyTest', msg))
  } finally {
    testingProxy.value = false
  }
}

// changeLanguage 切换界面语言：立即更新前端 locale 并通知后端 SetLanguage 持久化。
function changeLanguage() {
  const value = (state.settings.language || 'zh-CN') as Locale
  setLocale(value)
  api.setLanguage(value)
}
</script>

<template>
  <div class="space-y-5 animate-fade-up">
    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        {{ t('settings.section.path') }}
      </div>
      <div class="grid gap-4">
        <div>
          <label class="field-label">{{ t('settings.field.toolsDir') }}</label>
          <div class="flex gap-2">
            <input :value="state.settings.toolsDir" readonly class="field-input cursor-default opacity-80" />
            <button class="btn-ghost shrink-0" @click="chooseToolsDir">{{ t('settings.change') }}</button>
          </div>
          <p class="mt-1 text-[11px] text-muted">{{ t('settings.configFile') }}{{ state.settings.configPath }}</p>
        </div>

        <div>
          <label class="field-label">{{ t('settings.field.manualTools') }}</label>
          <div class="grid gap-2 md:grid-cols-2">
            <div v-for="tool in state.tools" :key="tool.name" class="flex items-center gap-2">
              <span class="w-24 shrink-0 font-mono text-xs text-slate-300">{{ tool.name }}</span>
              <input
                :value="toolPaths[tool.name] ?? ''"
                class="field-input flex-1"
                :placeholder="tool.path || t('settings.autoDetect')"
                @input="markDirty"
                @change="(e) => (toolPaths[tool.name] = (e.target as HTMLInputElement).value)"
              />
              <button class="btn-ghost shrink-0 !px-2 !text-xs" @click="pickToolPath(tool)">{{ t('common.browse') }}</button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        {{ t('settings.section.language') }}
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">{{ t('settings.field.language') }}</label>
          <select v-model="state.settings.language" class="field-input" @change="changeLanguage">
            <option v-for="opt in locales" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
          </select>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand-light"></span>
        {{ t('settings.section.sdk') }}
      </div>
      <div class="grid gap-4">
        <div>
          <label class="field-label">{{ t('settings.field.sdkPath') }}</label>
          <div class="flex gap-2">
            <input
              :value="state.settings.androidSdk"
              readonly
              class="field-input cursor-default opacity-80"
              :placeholder="t('settings.sdkPlaceholder')"
            />
            <button class="btn-ghost shrink-0" @click="chooseSdkDir">{{ t('settings.select') }}</button>
          </div>
          <p class="mt-1 text-[11px] text-muted">
            {{ t('settings.sdkHint') }}
          </p>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-sky-400"></span>
        {{ t('settings.section.mirror') }}
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">{{ t('settings.field.mirror') }}</label>
          <select v-model="state.settings.mirror" class="field-input" @change="markDirty">
            <option v-for="m in state.mirrors" :key="m.baseUrl" :value="m">{{ m.name }} - {{ m.note }}</option>
          </select>
        </div>
        <div class="flex items-end">
          <label class="flex cursor-pointer items-center gap-2 pb-2.5 text-sm text-slate-200">
            <input v-model="state.settings.proxy.enabled" type="checkbox" class="accent-brand" @change="markDirty" />
            {{ t('settings.enableProxy') }}
          </label>
        </div>
        <div>
          <label class="field-label">{{ t('settings.field.proxyType') }}</label>
          <select v-model="state.settings.proxy.type" class="field-input" @change="markDirty">
            <option value="http">HTTP</option>
            <option value="https">HTTPS</option>
            <option value="socks5">SOCKS5</option>
          </select>
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div>
            <label class="field-label">{{ t('settings.field.proxyHost') }}</label>
            <input v-model="state.settings.proxy.host" class="field-input" placeholder="127.0.0.1" @input="markDirty" />
          </div>
          <div>
            <label class="field-label">{{ t('settings.field.proxyPort') }}</label>
            <input v-model.number="state.settings.proxy.port" type="number" class="field-input" @input="markDirty" />
          </div>
        </div>
        <div>
          <label class="field-label">{{ t('settings.field.proxyUser') }}</label>
          <input v-model="state.settings.proxy.username" class="field-input" @input="markDirty" />
        </div>
        <div>
          <label class="field-label">{{ t('settings.field.proxyPass') }}</label>
          <input v-model="state.settings.proxy.password" type="password" class="field-input" @input="markDirty" />
        </div>

        <div class="md:col-span-2 flex flex-wrap items-center gap-3">
          <button class="btn-ghost" :disabled="testingProxy" @click="testProxy">
            <span
              v-if="testingProxy"
              class="mr-1 inline-block h-3 w-3 animate-spin rounded-full border-2 border-current border-t-transparent align-[-1px]"
            />
            {{ testingProxy ? t('settings.testing') : t('settings.testProxy') }}
          </button>
          <span
            v-if="proxyTest"
            :class="proxyTest.ok ? 'text-emerald-400' : 'text-rose-400'"
            class="text-xs"
            >{{ proxyTest.msg }}</span
          >
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand-light"></span>
        {{ t('settings.section.scheme') }}
      </div>
      <div class="grid gap-3 md:grid-cols-4">
        <label
          v-for="item in schemeItems"
          :key="item.key"
          class="flex cursor-pointer items-center gap-2 rounded-xl border border-white/5 bg-white/[0.02] px-3 py-2.5 text-sm transition hover:border-brand/30"
        >
          <input
            type="checkbox"
            class="accent-brand"
            :checked="defaultScheme(item.key)"
            @change="(e) => setDefaultScheme(item.key, (e.target as HTMLInputElement).checked)"
          />
          {{ item.label }}
        </label>
      </div>
    </section>

    <div class="flex flex-wrap items-center gap-3">
      <button class="btn-primary" @click="apply">{{ t('settings.save') }}</button>
      <button class="btn-ghost" @click="refreshTools">{{ t('settings.redetect') }}</button>
      <button class="btn-ghost" @click="cleanTemp">{{ t('settings.cleanTemp') }}</button>
      <span v-if="dirty" class="text-xs text-amber-300">{{ t('settings.unsaved') }}</span>
    </div>
  </div>
</template>

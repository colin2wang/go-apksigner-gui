<script setup lang="ts">
import { reactive, ref } from 'vue'
import { api } from '../api'
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
    const dir = await api.selectDirectory('选择工具存放目录')
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
    const dir = await api.selectDirectory('选择 Android SDK 根目录')
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
    notify('error', `代理测试失败: ${msg}`)
  } finally {
    testingProxy.value = false
  }
}
</script>

<template>
  <div class="space-y-5 animate-fade-up">
    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        路径
      </div>
      <div class="grid gap-4">
        <div>
          <label class="field-label">工具存放目录</label>
          <div class="flex gap-2">
            <input :value="state.settings.toolsDir" readonly class="field-input cursor-default opacity-80" />
            <button class="btn-ghost shrink-0" @click="chooseToolsDir">更改</button>
          </div>
          <p class="mt-1 text-[11px] text-muted">配置文件：{{ state.settings.configPath }}</p>
        </div>

        <div>
          <label class="field-label">手工指定工具路径（留空则自动探测）</label>
          <div class="grid gap-2 md:grid-cols-2">
            <div v-for="tool in state.tools" :key="tool.name" class="flex items-center gap-2">
              <span class="w-24 shrink-0 font-mono text-xs text-slate-300">{{ tool.name }}</span>
              <input
                :value="toolPaths[tool.name] ?? ''"
                class="field-input flex-1"
                :placeholder="tool.path || '自动探测'"
                @input="markDirty"
                @change="(e) => (toolPaths[tool.name] = (e.target as HTMLInputElement).value)"
              />
              <button class="btn-ghost shrink-0 !px-2 !text-xs" @click="pickToolPath(tool)">浏览</button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand-light"></span>
        Android SDK
      </div>
      <div class="grid gap-4">
        <div>
          <label class="field-label">Android SDK 路径</label>
          <div class="flex gap-2">
            <input
              :value="state.settings.androidSdk"
              readonly
              class="field-input cursor-default opacity-80"
              placeholder="留空则由应用自动下载 cmdline-tools 安装"
            />
            <button class="btn-ghost shrink-0" @click="chooseSdkDir">选择</button>
          </div>
          <p class="mt-1 text-[11px] text-muted">
            已手动安装 Android SDK（含 build-tools）时填写其根目录，应用将直接从此目录探测 apksigner / zipalign / aapt2；
            留空则使用工具目录下的托管 SDK，首次安装时自动下载 command-line tools 并通过 sdkmanager 安装 build-tools。
          </p>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-sky-400"></span>
        镜像源与代理
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">下载镜像源</label>
          <select v-model="state.settings.mirror" class="field-input" @change="markDirty">
            <option v-for="m in state.mirrors" :key="m.baseUrl" :value="m">{{ m.name }} - {{ m.note }}</option>
          </select>
        </div>
        <div class="flex items-end">
          <label class="flex cursor-pointer items-center gap-2 pb-2.5 text-sm text-slate-200">
            <input v-model="state.settings.proxy.enabled" type="checkbox" class="accent-brand" @change="markDirty" />
            启用下载代理
          </label>
        </div>
        <div>
          <label class="field-label">代理类型</label>
          <select v-model="state.settings.proxy.type" class="field-input" @change="markDirty">
            <option value="http">HTTP</option>
            <option value="https">HTTPS</option>
            <option value="socks5">SOCKS5</option>
          </select>
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div>
            <label class="field-label">主机</label>
            <input v-model="state.settings.proxy.host" class="field-input" placeholder="127.0.0.1" @input="markDirty" />
          </div>
          <div>
            <label class="field-label">端口</label>
            <input v-model.number="state.settings.proxy.port" type="number" class="field-input" @input="markDirty" />
          </div>
        </div>
        <div>
          <label class="field-label">代理用户名（可选）</label>
          <input v-model="state.settings.proxy.username" class="field-input" @input="markDirty" />
        </div>
        <div>
          <label class="field-label">代理密码（可选）</label>
          <input v-model="state.settings.proxy.password" type="password" class="field-input" @input="markDirty" />
        </div>

        <div class="md:col-span-2 flex flex-wrap items-center gap-3">
          <button class="btn-ghost" :disabled="testingProxy" @click="testProxy">
            <span
              v-if="testingProxy"
              class="mr-1 inline-block h-3 w-3 animate-spin rounded-full border-2 border-current border-t-transparent align-[-1px]"
            />
            {{ testingProxy ? '测试中…' : '测试代理连通性' }}
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
        默认签名方案
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
      <button class="btn-primary" @click="apply">保存设置</button>
      <button class="btn-ghost" @click="refreshTools">重新检测工具</button>
      <button class="btn-ghost" @click="cleanTemp">清理临时文件</button>
      <span v-if="dirty" class="text-xs text-amber-300">有未保存的修改</span>
    </div>
  </div>
</template>

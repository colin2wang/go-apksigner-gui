<script setup lang="ts">
import { ref } from 'vue'
import { api, pickFile } from '../api'
import { formatSize, notify, state } from '../stores/app'
import type { APKInfo, VerifyResult } from '../api/types'

const path = ref(state.settings.lastApk || '')
const info = ref<APKInfo | null>(null)
const verify = ref<VerifyResult | null>(null)
const busy = ref(false)

async function choose() {
  const file = await pickFile('*.apk')
  if (file) {
    path.value = file
    await parse()
  }
}

async function parse() {
  if (!path.value) return notify('warning', '请选择 APK 文件')
  busy.value = true
  verify.value = null
  try {
    info.value = await api.parseAPK(path.value)
    if (info.value.message) notify('warning', info.value.message)
  } catch (err) {
    notify('error', `解析失败: ${err}`)
  } finally {
    busy.value = false
  }
}

async function verifySign() {
  if (!path.value) return notify('warning', '请选择 APK 文件')
  busy.value = true
  try {
    verify.value = await api.verifyAPK(path.value)
  } catch (err) {
    notify('error', `验证失败: ${err}`)
  } finally {
    busy.value = false
  }
}

const sourceText = (source?: string) =>
  source === 'aapt2' ? 'aapt2 dump badging' : source === 'axml' ? '内置 AXML 解析（aapt2 不可用）' : '未知'
</script>

<template>
  <div class="space-y-5 animate-fade-up">
    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        选择 APK
      </div>
      <div class="flex flex-wrap gap-2">
        <select v-model="path" class="field-input max-w-2xl">
          <option value="">-- 选择最近文件或使用浏览 --</option>
          <option v-for="file in state.settings.recentFiles ?? []" :key="file" :value="file">{{ file }}</option>
        </select>
        <button class="btn-ghost" @click="choose">浏览</button>
        <button class="btn-primary" :disabled="busy" @click="parse">{{ busy ? '解析中…' : '解析信息' }}</button>
        <button class="btn-ghost" :disabled="busy" @click="verifySign">验证签名</button>
      </div>
    </section>

    <section v-if="info" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-sky-400"></span>
        基础信息
        <span class="ml-auto text-[11px] font-normal text-muted">{{ sourceText(info.source) }}</span>
      </div>
      <dl class="grid gap-4 md:grid-cols-3">
        <div>
          <dt class="text-muted text-xs">包名</dt>
          <dd class="stat-value break-all">{{ info.packageName || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted text-xs">应用名</dt>
          <dd class="stat-value">{{ info.label || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted text-xs">版本</dt>
          <dd class="stat-value">{{ info.versionName || '-' }} ({{ info.versionCode || '-' }})</dd>
        </div>
        <div>
          <dt class="text-muted text-xs">minSdk</dt>
          <dd class="stat-value">{{ info.minSdk || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted text-xs">targetSdk</dt>
          <dd class="stat-value">{{ info.targetSdk || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted text-xs">文件大小</dt>
          <dd class="stat-value">{{ formatSize(info.fileSize) }}</dd>
        </div>
        <div class="md:col-span-3">
          <dt class="text-muted text-xs">路径</dt>
          <dd class="kbd-text">{{ info.path }}</dd>
        </div>
      </dl>
    </section>

    <section v-if="verify" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full" :class="verify.verified ? 'bg-brand' : 'bg-red-400'"></span>
        签名详情
      </div>
      <div class="mb-3 flex flex-wrap gap-2 text-xs">
        <span class="chip" :class="verify.verified ? 'bg-brand/15 text-brand-light' : 'bg-red-500/15 text-red-300'">
          {{ verify.verified ? '签名有效' : '签名无效或缺失' }}
        </span>
        <span
          v-for="item in [
            { label: 'V1', value: verify.v1Signed },
            { label: 'V2', value: verify.v2Signed },
            { label: 'V3', value: verify.v3Signed },
            { label: 'V4', value: verify.v4Signed },
          ]"
          :key="item.label"
          class="chip"
          :class="item.value ? 'bg-white/10 text-slate-200' : 'bg-white/5 text-slate-500'"
        >
          {{ item.label }}: {{ item.value ? '已签名' : '未签名' }}
        </span>
      </div>
      <div v-for="(cert, idx) in verify.certs ?? []" :key="idx" class="rounded-xl border border-white/5 bg-ink-900/50 p-3 text-xs">
        <p class="text-slate-300">DN: <span class="stat-value break-all">{{ cert.dn }}</span></p>
        <p class="mt-1 text-slate-400">SHA-256: <span class="kbd-text">{{ cert.sha256 }}</span></p>
      </div>
      <details v-if="verify.rawOutput" class="mt-3 text-xs text-muted">
        <summary class="cursor-pointer">查看原始输出</summary>
        <pre class="mt-2 max-h-60 overflow-auto whitespace-pre-wrap font-mono text-[11px] text-slate-300">{{ verify.rawOutput }}</pre>
      </details>
    </section>
  </div>
</template>

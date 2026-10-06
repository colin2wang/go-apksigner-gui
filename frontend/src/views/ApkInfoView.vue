<!--
  Author: colin2wang (colin2wang@gmail.com)
  Date: 2026-10-06
-->

<script setup lang="ts">
import { ref } from 'vue'
import { api, pickFile } from '../api'
import { t } from '../i18n'
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
  if (!path.value) return notify('warning', t('apk.warn.choose'))
  busy.value = true
  verify.value = null
  try {
    info.value = await api.parseAPK(path.value)
    if (info.value.message) notify('warning', info.value.message)
  } catch (err) {
    notify('error', t('apk.err.parse', String(err)))
  } finally {
    busy.value = false
  }
}

async function verifySign() {
  if (!path.value) return notify('warning', t('apk.warn.choose'))
  busy.value = true
  try {
    verify.value = await api.verifyAPK(path.value)
  } catch (err) {
    notify('error', t('apk.err.verify', String(err)))
  } finally {
    busy.value = false
  }
}

const sourceText = (source?: string) =>
  source === 'aapt2' ? t('apk.source.aapt2') : source === 'axml' ? t('apk.source.axml') : t('apk.source.unknown')
</script>

<template>
  <div class="space-y-5 animate-fade-up">
    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        {{ t('apk.chooseTitle') }}
      </div>
      <div class="flex flex-wrap gap-2">
        <select v-model="path" class="field-input max-w-2xl">
          <option value="">{{ t('apk.recentPlaceholder') }}</option>
          <option v-for="file in state.settings.recentFiles ?? []" :key="file" :value="file">{{ file }}</option>
        </select>
        <button class="btn-ghost" @click="choose">{{ t('common.browse') }}</button>
        <button class="btn-primary" :disabled="busy" @click="parse">{{ busy ? t('apk.parsing') : t('apk.parseBtn') }}</button>
        <button class="btn-ghost" :disabled="busy" @click="verifySign">{{ t('apk.verifyBtn') }}</button>
      </div>
    </section>

    <section v-if="info" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-sky-400"></span>
        {{ t('apk.basicTitle') }}
        <span class="ml-auto text-[11px] font-normal text-muted">{{ sourceText(info.source) }}</span>
      </div>
      <dl class="grid gap-4 md:grid-cols-3">
        <div>
          <dt class="text-muted text-xs">{{ t('apk.pkg') }}</dt>
          <dd class="stat-value break-all">{{ info.packageName || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted text-xs">{{ t('apk.appName') }}</dt>
          <dd class="stat-value">{{ info.label || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted text-xs">{{ t('apk.version') }}</dt>
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
          <dt class="text-muted text-xs">{{ t('apk.fileSize') }}</dt>
          <dd class="stat-value">{{ formatSize(info.fileSize) }}</dd>
        </div>
        <div class="md:col-span-3">
          <dt class="text-muted text-xs">{{ t('apk.path') }}</dt>
          <dd class="kbd-text">{{ info.path }}</dd>
        </div>
      </dl>
    </section>

    <section v-if="verify" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full" :class="verify.verified ? 'bg-brand' : 'bg-red-400'"></span>
        {{ t('apk.signDetailTitle') }}
      </div>
      <div class="mb-3 flex flex-wrap gap-2 text-xs">
        <span class="chip" :class="verify.verified ? 'bg-brand/15 text-brand-light' : 'bg-red-500/15 text-red-300'">
          {{ verify.verified ? t('apk.sigValid') : t('apk.sigInvalid') }}
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
          {{ item.label }}: {{ item.value ? t('apk.signed') : t('apk.unsigned') }}
        </span>
      </div>
      <div v-for="(cert, idx) in verify.certs ?? []" :key="idx" class="rounded-xl border border-white/5 bg-ink-900/50 p-3 text-xs">
        <p class="text-slate-300">DN: <span class="stat-value break-all">{{ cert.dn }}</span></p>
        <p class="mt-1 text-slate-400">SHA-256: <span class="kbd-text">{{ cert.sha256 }}</span></p>
      </div>
      <details v-if="verify.rawOutput" class="mt-3 text-xs text-muted">
        <summary class="cursor-pointer">{{ t('apk.viewRaw') }}</summary>
        <pre class="mt-2 max-h-60 overflow-auto whitespace-pre-wrap font-mono text-[11px] text-slate-300">{{ verify.rawOutput }}</pre>
      </details>
    </section>
  </div>
</template>

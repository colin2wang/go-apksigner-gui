<!--
  Author: colin2wang (colin2wang@gmail.com)
  Date: 2026-10-06
-->

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, pickFile } from '../api'
import { formatSize, notify, state } from '../stores/app'
import { t } from '../i18n'
import type { APKInfo, KeystoreFile, SignOptions, VerifyResult } from '../api/types'

const form = ref({
  inputApk: state.settings.lastApk || '',
  outputApk: '',
  keystore: state.settings.lastKeystore || '',
  alias: state.settings.lastAlias || '',
  storePass: '',
  keyPass: '',
  v1Enabled: state.settings.defaultV1,
  v2Enabled: state.settings.defaultV2,
  v3Enabled: state.settings.defaultV3,
  v4Enabled: state.settings.defaultV4,
  zipAlignFirst: true,
  removeOldSign: false,
  minSdk: 0,
  maxSdk: 0,
})

const apkInfo = ref<APKInfo | null>(null)
const verifyResult = ref<VerifyResult | null>(null)
const resultMessage = ref('')
const aliases = ref<string[]>([])
const aliasLoading = ref(false)
const ksFiles = ref<KeystoreFile[]>([])
const ksPass = ref('')
const rememberPass = ref(true)
const savedPass = ref(true)
const passResolving = ref(false)

const stepOrder = computed(() => [
  t('sign.step.remove'),
  t('sign.step.zipalign'),
  t('sign.step.sign'),
  t('sign.step.verify'),
])
const visibleSteps = computed(() => stepOrder.value.filter((name) => state.steps.some((s) => s.name === name)))
const stepMap = computed(() => {
  const map: Record<string, string> = {}
  state.steps.forEach((s) => {
    map[s.name] = s.status
  })
  return map
})

function stepClass(status?: string) {
  switch (status) {
    case 'success':
      return 'border-brand/60 bg-brand/10 text-brand-light'
    case 'running':
      return 'border-sky-400/60 bg-sky-400/10 text-sky-200 animate-breathe'
    case 'error':
      return 'border-red-500/60 bg-red-500/10 text-red-300'
    default:
      return 'border-white/10 bg-white/[0.02] text-slate-400'
  }
}

async function chooseApk() {
  const path = await pickFile('*.apk')
  if (!path) return
  form.value.inputApk = path
  apkInfo.value = null
  try {
    apkInfo.value = await api.parseAPK(path)
    form.value.minSdk = 0
  } catch (err) {
    notify('error', t('sign.apkParseFailed', String(err)))
  }
}

async function chooseKeystore() {
  const path = await pickFile('*.jks;*.keystore;*.p12;*.pfx')
  if (!path) return
  form.value.keystore = path
  await onSelectKs()
}

async function loadKsList() {
  try {
    ksFiles.value = await api.listKeystoreDir()
  } catch {
    ksFiles.value = []
  }
}

async function onSelectKs() {
  form.value.alias = ''
  aliases.value = []
  ksPass.value = ''
  const path = form.value.keystore
  if (!path) {
    savedPass.value = false
    return
  }
  try {
    const sp = await api.getSavedPassword(path)
    if (sp) {
      form.value.storePass = sp
      savedPass.value = true
      await autoLoadAliases()
    } else {
      form.value.storePass = ''
      savedPass.value = false
    }
  } catch {
    form.value.storePass = ''
    savedPass.value = false
  }
}

async function confirmInlinePass() {
  if (!ksPass.value) return notify('warning', t('sign.fillStorePass'))
  passResolving.value = true
  try {
    aliases.value = await api.listAliases(form.value.keystore, ksPass.value)
    form.value.storePass = ksPass.value
    if (rememberPass.value) await api.savePassword(form.value.keystore, ksPass.value)
    savedPass.value = true
    if (!aliases.value.length) notify('warning', t('sign.noAliasLoaded'))
  } catch (err) {
    notify('error', t('sign.verifyPassFailed', String(err)))
  } finally {
    passResolving.value = false
  }
}

async function autoLoadAliases() {
  if (!form.value.keystore || !form.value.storePass) {
    notify('warning', t('sign.selectKsAndPass'))
    return
  }
  aliasLoading.value = true
  try {
    aliases.value = await api.listAliases(form.value.keystore, form.value.storePass)
    if (!aliases.value.length) notify('warning', t('sign.noAliasLoaded'))
  } catch (err) {
    notify('error', t('sign.loadAliasFailed', String(err)))
  } finally {
    aliasLoading.value = false
  }
}

async function chooseOutput() {
  try {
    const path = await api.selectSaveFile('signed.apk', '*.apk')
    if (path) form.value.outputApk = path
  } catch {
    /* 用户取消 */
  }
}


async function sign() {
  if (!form.value.inputApk) return notify('warning', t('sign.chooseApk'))
  if (!form.value.keystore) return notify('warning', t('sign.chooseKeystore'))
  if (!form.value.storePass) return notify('warning', t('sign.fillStorePass'))

  state.signing = true
  state.steps.splice(0, state.steps.length)
  verifyResult.value = null
  resultMessage.value = ''

  const payload: SignOptions = {
    inputApk: form.value.inputApk,
    outputApk: form.value.outputApk,
    keystore: form.value.keystore,
    alias: form.value.alias,
    storePass: form.value.storePass,
    keyPass: form.value.keyPass,
    v1Enabled: form.value.v1Enabled,
    v2Enabled: form.value.v2Enabled,
    v3Enabled: form.value.v3Enabled,
    v4Enabled: form.value.v4Enabled,
    zipAlignFirst: form.value.zipAlignFirst,
    removeOldSign: form.value.removeOldSign,
    minSdk: Number(form.value.minSdk) || 0,
    maxSdk: Number(form.value.maxSdk) || 0,
    debuggableApkPerm: false,
  }

  try {
    const res = await api.signAPK(payload)
    verifyResult.value = res.verify ?? null
    resultMessage.value = res.message
    if (res.success) {
      notify('success', res.message)
      form.value.outputApk = res.outputPath
    } else {
      notify('error', res.message)
    }
  } catch (err) {
    notify('error', t('sign.signFailed', String(err)))
  } finally {
    state.signing = false
  }
}

async function verify() {
  const target = form.value.outputApk || form.value.inputApk
  if (!target) return notify('warning', t('sign.chooseVerifyApk'))
  try {
    verifyResult.value = await api.verifyAPK(target)
    notify(verifyResult.value.verified ? 'success' : 'warning', t(verifyResult.value.verified ? 'sign.verifyPass' : 'sign.verifyFail'))
  } catch (err) {
    notify('error', t('sign.signFailed', String(err)))
  }
}

const schemes = computed(() => [
  { key: 'v1Enabled' as const, label: 'V1 (JAR)', hint: t('sign.scheme.v1Hint') },
  { key: 'v2Enabled' as const, label: 'V2', hint: t('sign.scheme.v2Hint') },
  { key: 'v3Enabled' as const, label: 'V3', hint: t('sign.scheme.v3Hint') },
  { key: 'v4Enabled' as const, label: 'V4', hint: t('sign.scheme.v4Hint') },
])

onMounted(async () => {
  await loadKsList()
  if (form.value.keystore) await onSelectKs()
})
</script>

<template>
  <div class="space-y-5 animate-fade-up">
    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        {{ t('sign.sectionFiles') }}
      </div>

      <div class="grid gap-4">
        <div>
          <label class="field-label">{{ t('sign.inputApk') }}</label>
          <div class="flex gap-2">
            <input v-model="form.inputApk" class="field-input" :placeholder="t('sign.inputApkPlaceholder')" />
            <button class="btn-ghost shrink-0" @click="chooseApk">{{ t('sign.btnChooseApk') }}</button>
          </div>
        </div>

        <div>
          <label class="field-label">{{ t('sign.outputApkLabel') }}</label>
          <div class="flex gap-2">
            <input v-model="form.outputApk" class="field-input" :placeholder="t('sign.outputApkPlaceholder')" />
            <button class="btn-ghost shrink-0" @click="chooseOutput">{{ t('sign.btnChoosePath') }}</button>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="field-label">{{ t('sign.keystoreLabel') }}</label>
            <div class="flex gap-2">
              <select v-model="form.keystore" class="field-input" :disabled="!ksFiles.length" @change="onSelectKs">
                <option v-for="f in ksFiles" :key="f.path" :value="f.path" :title="f.path">{{ f.name }}</option>
                <option v-if="form.keystore && !ksFiles.some((f) => f.path === form.keystore)" :value="form.keystore">
                  {{ form.keystore.split(/[\\/]/).pop() }} {{ t('sign.external') }}
                </option>
                <option v-if="!ksFiles.length" value="" disabled>{{ t('sign.keystoresEmpty') }}</option>
              </select>
              <button class="btn-ghost shrink-0" @click="chooseKeystore">{{ t('common.browse') }}</button>
            </div>
            <p v-if="!ksFiles.length" class="mt-1 text-xs text-muted">{{ t('sign.goToCert') }}</p>
          </div>
          <div>
            <label class="field-label">{{ t('sign.alias') }}</label>
            <div class="flex gap-2">
              <select v-model="form.alias" class="field-input" :disabled="!aliases.length">
                <option v-for="a in aliases" :key="a" :value="a">{{ a }}</option>
                <option v-if="!aliases.length" value="" disabled>{{ aliasLoading ? t('sign.aliasLoading') : t('sign.aliasSelectFirst') }}</option>
              </select>
              <button class="btn-ghost shrink-0" :disabled="aliasLoading || !form.keystore" @click="autoLoadAliases">
                {{ aliasLoading ? t('sign.aliasLoading') : t('common.refresh') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="form.keystore && !savedPass" class="grid items-end gap-4 md:grid-cols-2">
          <div>
            <label class="field-label">{{ t('sign.keystorePass') }}</label>
            <input v-model="ksPass" type="password" class="field-input" :placeholder="t('sign.keystorePassPlaceholder')" autocomplete="off" @keyup.enter="confirmInlinePass" />
          </div>
          <div class="flex items-center gap-3 pb-1">
            <label class="flex items-center gap-2 text-sm text-slate-200">
              <input v-model="rememberPass" type="checkbox" class="accent-brand" /> {{ t('sign.rememberPass') }}
            </label>
            <button class="btn-ghost shrink-0" :disabled="passResolving" @click="confirmInlinePass">
              {{ passResolving ? t('sign.verifying') : t('common.confirm') }}
            </button>
          </div>
        </div>

        <p v-if="form.keystore && savedPass" class="text-xs text-muted">{{ t('sign.usingSavedPass') }}</p>

        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="field-label">{{ t('sign.keyPassLabel') }}</label>
            <input v-model="form.keyPass" type="password" class="field-input" autocomplete="off" :placeholder="form.storePass ? t('sign.keyPassPlaceholder') : ''" />
          </div>
        </div>
      </div>
    </section>

    <section v-if="apkInfo" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-sky-400"></span>
        {{ t('sign.apkSummary') }}
        <span class="ml-auto text-[11px] font-normal text-muted">{{ t('sign.source') }}：{{ apkInfo.source }}</span>
      </div>
      <dl class="grid gap-3 text-xs md:grid-cols-3">
        <div>
          <dt class="text-muted">{{ t('sign.pkg') }}</dt>
          <dd class="stat-value">{{ apkInfo.packageName || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted">{{ t('sign.version') }}</dt>
          <dd class="stat-value">{{ apkInfo.versionName || '-' }} ({{ apkInfo.versionCode || '-' }})</dd>
        </div>
        <div>
          <dt class="text-muted">{{ t('sign.size') }}</dt>
          <dd class="stat-value">{{ formatSize(apkInfo.fileSize) }}</dd>
        </div>
        <div>
          <dt class="text-muted">{{ t('sign.minSdk') }}</dt>
          <dd class="stat-value">{{ apkInfo.minSdk || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted">{{ t('sign.targetSdk') }}</dt>
          <dd class="stat-value">{{ apkInfo.targetSdk || '-' }}</dd>
        </div>
        <div>
          <dt class="text-muted">{{ t('sign.signStatus') }}</dt>
          <dd>
            <span class="chip" :class="apkInfo.isSigned ? 'bg-sky-400/15 text-sky-300' : 'bg-white/10 text-slate-300'">
              {{ apkInfo.isSigned ? t('sign.signed') : t('sign.unsigned') }}
            </span>
          </dd>
        </div>
      </dl>
    </section>

    <section class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand-light"></span>
        {{ t('sign.params') }}
      </div>

      <div class="mb-4 grid gap-3 md:grid-cols-4">
        <label
          v-for="s in schemes"
          :key="s.key"
          class="flex cursor-pointer items-center gap-3 rounded-xl border px-3 py-2.5 transition"
          :class="form[s.key] ? 'border-brand/40 bg-brand/5' : 'border-white/5 bg-white/[0.02]'"
        >
          <input v-model="form[s.key]" type="checkbox" class="accent-brand" />
          <span>
            <span class="block text-sm text-white">{{ s.label }}</span>
            <span class="block text-[11px] text-muted">{{ s.hint }}</span>
          </span>
        </label>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">{{ t('sign.minSdkLabel') }}</label>
          <input v-model.number="form.minSdk" type="number" min="0" class="field-input" />
        </div>
        <div>
          <label class="field-label">{{ t('sign.maxSdkLabel') }}</label>
          <input v-model.number="form.maxSdk" type="number" min="0" class="field-input" />
        </div>
      </div>

      <div class="mt-4 flex flex-wrap gap-6">
        <label class="flex cursor-pointer items-center gap-2 text-sm text-slate-200">
          <input v-model="form.zipAlignFirst" type="checkbox" class="accent-brand" />
          {{ t('sign.zipAlignFirst') }}
        </label>
        <label class="flex cursor-pointer items-center gap-2 text-sm text-slate-200">
          <input v-model="form.removeOldSign" type="checkbox" class="accent-brand" />
          {{ t('sign.removeOld') }}
        </label>
      </div>

      <div class="mt-5 flex flex-wrap gap-3">
        <button class="btn-primary" :disabled="state.signing" @click="sign">
          {{ state.signing ? t('sign.signing') : t('sign.start') }}
        </button>
        <button class="btn-ghost" :disabled="state.signing" @click="verify">{{ t('sign.verifyBtn') }}</button>
      </div>
    </section>

    <section v-if="visibleSteps.length" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        {{ t('sign.pipeline') }}
      </div>
      <div class="flex flex-wrap gap-3">
        <div v-for="name in visibleSteps" :key="name" class="rounded-xl border px-3 py-2 text-xs" :class="stepClass(stepMap[name])">
          <span class="font-medium">{{ name }}</span>
          <span class="ml-2 uppercase tracking-wider opacity-70">{{ stepMap[name] }}</span>
        </div>
      </div>
    </section>

    <section v-if="resultMessage || verifyResult" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full" :class="verifyResult?.verified ? 'bg-brand' : 'bg-red-400'"></span>
        {{ t('sign.verifyResult') }}
      </div>
      <p class="mb-3 text-sm" :class="verifyResult?.verified ? 'text-brand-light' : 'text-red-300'">
        {{ resultMessage }}
      </p>
      <div v-if="verifyResult" class="space-y-3">
        <div class="flex flex-wrap gap-2 text-xs">
          <span v-for="item in [
            { label: 'V1', value: verifyResult.v1Signed },
            { label: 'V2', value: verifyResult.v2Signed },
            { label: 'V3', value: verifyResult.v3Signed },
            { label: 'V4', value: verifyResult.v4Signed },
          ]" :key="item.label" class="chip" :class="item.value ? 'bg-brand/15 text-brand-light' : 'bg-white/10 text-slate-400'">
            {{ item.label }}: {{ item.value ? t('sign.signed') : t('sign.unsigned') }}
          </span>
        </div>
        <div v-for="(cert, idx) in verifyResult.certs ?? []" :key="idx" class="rounded-xl border border-white/5 bg-ink-900/50 p-3 text-xs">
          <p class="text-slate-300">DN: <span class="stat-value">{{ cert.dn }}</span></p>
          <p class="mt-1 text-slate-400">SHA-256: <span class="kbd-text">{{ cert.sha256 }}</span></p>
          <p class="mt-1 text-slate-400">SHA-1: <span class="kbd-text">{{ cert.sha1 }}</span></p>
        </div>
        <p v-if="verifyResult.warnings?.length" class="text-xs text-amber-300">
          {{ t('sign.warnings') }}：{{ verifyResult.warnings.join(' / ') }}
        </p>
      </div>
    </section>
  </div>
</template>

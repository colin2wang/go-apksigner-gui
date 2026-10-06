<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, reactive } from 'vue'
import { api } from '../api'
import { notify, state } from '../stores/app'
import { DialogPlugin } from 'tdesign-vue-next'
import type { KeystoreInfo, KeystoreRequest, KeystoreFile } from '../api/types'

type TabKey = 'home' | 'import' | 'generate' | 'view' | 'convert'
const tab = ref<TabKey>('home')
const tabs: { key: TabKey; label: string }[] = [
  { key: 'home', label: '首页' },
  { key: 'import', label: '导入' },
  { key: 'generate', label: '重新生成' },
  { key: 'view', label: '查看·导出' },
  { key: 'convert', label: '格式转换' },
]

const form = ref<KeystoreRequest>({
  outputPath: 'release.jks',
  storeType: 'JKS',
  storePass: '',
  keyPass: '',
  cert: {
    alias: state.settings.lastAlias || 'android-key',
    commonName: '',
    organization: '',
    orgUnit: 'Dev',
    locality: 'Beijing',
    state: 'Beijing',
    country: 'CN',
    validity: 9125,
    keyAlgorithm: 'RSA',
    keySize: 2048,
  },
})

const info = ref<KeystoreInfo | null>(null)
const viewPath = ref(state.settings.lastKeystore || '')
const viewPass = ref('')
const exportTarget = ref({ alias: '', output: 'certificate.pem' })
const convert = ref({ src: '', srcPass: '', dst: '', dstPass: '', type: 'PKCS12' })
const detail = ref('')
const busy = ref(false)
// viewPassInput 查看页密码输入框引用，用于选择密钥库后自动聚焦。
const viewPassInput = ref<HTMLInputElement | null>(null)

// ---- 首页：keystores 目录列表
const homeList = ref<KeystoreFile[]>([])
async function loadHome() {
  try {
    homeList.value = await api.listKeystoreDir()
  } catch (e) {
    notify('error', `读取密钥库目录失败: ${e}`)
  }
}

// ---- 首页：点击卡片后输入密码弹窗
const homeDialog = reactive({ open: false, path: '', name: '', pass: '', error: '', remember: false, busy: false })
const homePassInput = ref<HTMLInputElement | null>(null)
async function openHome(f: KeystoreFile) {
  homeDialog.path = f.path
  homeDialog.name = f.name
  homeDialog.pass = ''
  homeDialog.error = ''
  homeDialog.remember = false
  homeDialog.open = true
  const saved = await api.getSavedPassword(f.path).catch(() => '')
  if (saved) {
    homeDialog.pass = saved
    homeDialog.remember = true
  }
  nextTick(() => homePassInput.value?.focus())
}
async function homeSubmit() {
  if (!homeDialog.pass) {
    homeDialog.error = '请输入密钥库密码'
    return
  }
  homeDialog.busy = true
  homeDialog.error = ''
  try {
    await api.listKeystore(homeDialog.path, homeDialog.pass)
    if (homeDialog.remember) await api.savePassword(homeDialog.path, homeDialog.pass).catch(() => {})
    else await api.forgetPassword(homeDialog.path).catch(() => {})
    viewPath.value = homeDialog.path
    viewPass.value = homeDialog.pass
    tab.value = 'view'
    homeDialog.open = false
    await view()
  } catch (e) {
    homeDialog.error = `校验失败: ${e}`
  } finally {
    homeDialog.busy = false
  }
}

// ---- 导入：选文件 + 密码，校验后复制到 keystores 目录
const importForm = reactive({ src: '', pass: '', info: null as KeystoreInfo | null, busy: false })
async function importChoose() {
  try {
    const p = await api.selectOpenFile('*.jks;*.keystore;*.p12;*.pfx')
    if (p) {
      importForm.src = p
      importForm.info = null
    }
  } catch {
    /* 取消 */
  }
}
async function importLoad() {
  if (!importForm.src) return notify('warning', '请先选择文件')
  if (!importForm.pass) return notify('warning', '请输入密钥库密码')
  importForm.busy = true
  try {
    importForm.info = await api.listKeystore(importForm.src, importForm.pass)
  } catch (e) {
    importForm.info = null
    notify('error', `校验失败: ${e}`)
  } finally {
    importForm.busy = false
  }
}
async function importDo() {
  if (!importForm.src || !importForm.pass) return notify('warning', '请选择文件并输入密码')
  importForm.busy = true
  try {
    const res = await api.importKeystore(importForm.src, importForm.pass)
    notify(res.success ? 'success' : 'error', res.message)
    if (res.success) {
      importForm.src = ''
      importForm.pass = ''
      importForm.info = null
      await loadHome()
      tab.value = 'home'
    }
  } catch (e) {
    notify('error', `导入失败: ${e}`)
  } finally {
    importForm.busy = false
  }
}

async function chooseSave() {
  try {
    const path = await api.selectSaveFile('release.jks', '*.jks;*.keystore;*.p12')
    if (path) form.value.outputPath = path
  } catch {
    /* 取消 */
  }
}

async function chooseExisting() {
  try {
    const path = await api.selectOpenFile('*.jks;*.keystore;*.p12;*.pfx')
    if (!path) return
    viewPath.value = path
    if (viewPass.value) view(true).catch(() => {})
    else nextTick(() => viewPassInput.value?.focus())
  } catch {
    /* 取消 */
  }
}

// ksExts 密钥库可识别的扩展名。
const ksExts = ['.jks', '.keystore', '.p12', '.pfx']
// basename 取路径中的文件名，用于最近列表展示。
function basename(p: string): string {
  return p.split(/[\\/]/).pop() || p
}
// recentKeystores 取配置中最近使用的密钥库（过滤非密钥库文件）。
const recentKeystores = computed(() =>
  (state.settings.recentFiles || [])
    .filter((p) => ksExts.some((e) => p.toLowerCase().endsWith(e)))
    .slice(0, 6),
)

// openRecent 从最近列表点选：填充路径，已有密码则自动校验，否则聚焦密码框。
async function openRecent(path: string) {
  viewPath.value = path
  if (viewPass.value) view(true).catch(() => {})
  else nextTick(() => viewPassInput.value?.focus())
}

// chooseExistingTarget 生成页选择已有密钥库作为追加目标。
async function chooseExistingTarget() {
  try {
    const path = await api.selectOpenFile('*.jks;*.keystore;*.p12;*.pfx')
    if (path) form.value.outputPath = path
  } catch {
    /* 取消 */
  }
}

async function generate() {
  if (!form.value.outputPath) return notify('warning', '请填写保存路径')
  if (!form.value.cert.alias) return notify('warning', '请填写别名')
  if (form.value.storePass.length < 6) return notify('warning', '密钥库密码至少 6 位')
  if (!form.value.cert.commonName) return notify('warning', '请填写通用名 (CN)')

  storeTypeByExt()
  busy.value = true
  try {
    const res = await api.generateKeystore(form.value)
    detail.value = res.detail || ''
    notify(res.success ? 'success' : 'error', res.message)
    if (res.success) {
      viewPath.value = form.value.outputPath
      viewPass.value = form.value.storePass
      tab.value = 'view'
      await view()
      await loadHome()
    }
  } catch (err) {
    notify('error', `生成失败: ${err}`)
  } finally {
    busy.value = false
  }
}

function storeTypeByExt() {
  const lower = form.value.outputPath.toLowerCase()
  if (lower.endsWith('.p12') || lower.endsWith('.pfx')) form.value.storeType = 'PKCS12'
  else form.value.storeType = 'JKS'
}

// view 读取密钥库详情。silent=true 时（自动触发）不弹成功提示，避免输入过程刷屏。
async function view(silent = false) {
  if (!viewPath.value) return notify('warning', '请选择密钥库文件')
  if (!viewPass.value) return notify('warning', '请输入密钥库密码')
  busy.value = true
  try {
    info.value = await api.listKeystore(viewPath.value, viewPass.value)
    if (!info.value.entries?.length) {
      if (!silent) notify('warning', '未读取到证书条目')
    } else {
      if (!silent) notify('success', `共读取到 ${info.value.entries.length} 个条目`)
      if (!exportTarget.value.alias) exportTarget.value.alias = info.value.entries[0].alias
    }
  } catch (err) {
    notify('error', `读取失败: ${err}`)
    info.value = null
  } finally {
    busy.value = false
  }
}

// autoView 路径与密码均非空时防抖自动校验并加载证书信息。
let viewTimer: ReturnType<typeof setTimeout> | undefined
watch([viewPath, viewPass], () => {
  if (!viewPath.value || !viewPass.value || busy.value) return
  if (viewTimer) clearTimeout(viewTimer)
  viewTimer = setTimeout(() => {
    view(true).catch(() => {})
  }, 600)
})

// deleteAlias 删除密钥库中的指定别名，操作前二次确认。
async function deleteAlias(alias: string) {
  if (!viewPath.value) return notify('warning', '请先选择密钥库')
  const action = await DialogPlugin.confirm({
    header: '删除别名',
    body: `确定要删除别名「${alias}」吗？该操作不可撤销，且会影响依赖此别名的签名。`,
    theme: 'warning',
    confirmBtn: '删除',
    cancelBtn: '取消',
  })
  if (action !== 'confirm') return
  busy.value = true
  try {
    const res = await api.deleteAlias(viewPath.value, viewPass.value, alias)
    notify(res.success ? 'success' : 'error', res.message)
    if (res.success) await view()
  } catch (err) {
    notify('error', `删除失败: ${err}`)
  } finally {
    busy.value = false
  }
}

async function exportCert() {
  if (!viewPath.value || !exportTarget.value.alias) return notify('warning', '请选择密钥库并填写别名')
  busy.value = true
  try {
    const res = await api.exportCertificate(viewPath.value, viewPass.value, exportTarget.value.alias, exportTarget.value.output)
    detail.value = res.detail || ''
    notify(res.success ? 'success' : 'error', res.message)
  } catch (err) {
    notify('error', `导出失败: ${err}`)
  } finally {
    busy.value = false
  }
}

async function convertKeystore() {
  if (!convert.value.src || !convert.value.dst) return notify('warning', '请填写源与目标路径')
  busy.value = true
  try {
    const res = await api.convertKeystore(
      convert.value.src,
      convert.value.srcPass,
      convert.value.dst,
      convert.value.dstPass,
      convert.value.type,
    )
    detail.value = res.detail || ''
    notify(res.success ? 'success' : 'error', res.message)
  } catch (err) {
    notify('error', `转换失败: ${err}`)
  } finally {
    busy.value = false
  }
}

function useEntryAlias(alias: string) {
  exportTarget.value.alias = alias
  tab.value = 'view'
  notify('info', `已选择别名 ${alias}，可点击导出证书`)
}

// 默认生成路径指向 keystores 目录，并加载首页列表。
onMounted(async () => {
  await loadHome()
  try {
    const dir = await api.keystoreDir()
    if (dir) form.value.outputPath = dir + '/release.jks'
  } catch {
    /* 取默认值 */
  }
})

function fmtSize(n: number): string {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1024 / 1024).toFixed(2) + ' MB'
}
function fmtTime(ms: number): string {
  if (!ms) return ''
  return new Date(ms).toLocaleString()
}
</script>

<template>
  <div class="space-y-5 animate-fade-up">
    <div class="flex gap-2 flex-wrap">
      <button
        v-for="item in tabs"
        :key="item.key"
        class="rounded-xl px-4 py-2 text-sm transition"
        :class="tab === item.key ? 'bg-brand/15 text-brand-light ring-1 ring-brand/40' : 'text-slate-300 hover:bg-white/5'"
        @click="tab = item.key"
      >
        {{ item.label }}
      </button>
    </div>

    <!-- 首页：keystores 目录下的已有证书 -->
    <section v-if="tab === 'home'" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        密钥库目录（keystores）
      </div>
      <p class="text-[11px] leading-relaxed text-muted">
        位于可执行文件同级的 <code class="kbd-text">keystores</code> 子目录，仅展示文件名、大小与修改时间。点击任意密钥库输入密码以查看详情。
      </p>
      <div class="mt-3 flex justify-end">
        <button class="btn-ghost" @click="loadHome">刷新</button>
      </div>

      <div v-if="!homeList.length" class="mt-4 rounded-xl border border-dashed border-white/10 p-8 text-center text-sm text-muted">
        暂无密钥库，请前往「导入」或「重新生成」。
      </div>
      <div v-else class="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <button
          v-for="f in homeList"
          :key="f.path"
          class="rounded-xl border border-white/5 bg-ink-900/50 p-4 text-left transition hover:border-brand/40 hover:bg-brand/5"
          @click="openHome(f)"
        >
          <p class="truncate font-mono text-sm text-brand-light" :title="f.path">{{ f.name }}</p>
          <p class="mt-1 text-[11px] text-muted">{{ fmtSize(f.size) }} · {{ fmtTime(f.modTime) }}</p>
        </button>
      </div>
    </section>

    <!-- 导入 -->
    <section v-if="tab === 'import'" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        导入已有密钥库
      </div>
      <p class="text-[11px] leading-relaxed text-muted">
        选择外部密钥库并输入密码，校验通过后复制到 <code class="kbd-text">keystores</code> 目录（保留原文件，重命名为 imported_日期时间）。
      </p>

      <div class="mt-4 flex flex-wrap items-center gap-2">
        <button class="btn-primary" @click="importChoose">选择文件…</button>
        <span class="max-w-xs truncate text-xs text-muted">{{ importForm.src || '未选择' }}</span>
      </div>

      <div class="mt-4 grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">密钥库密码</label>
          <div class="flex gap-2">
            <input v-model="importForm.pass" type="password" class="field-input" autocomplete="new-password" />
            <button class="btn-ghost shrink-0" :disabled="importForm.busy" @click="importLoad">加载信息</button>
          </div>
        </div>
      </div>

      <div v-if="importForm.info?.entries?.length" class="mt-4 space-y-3">
        <p class="text-xs text-muted">读取到 {{ importForm.info.entries.length }} 个条目，确认后导入：</p>
        <div
          v-for="entry in importForm.info.entries"
          :key="entry.alias"
          class="rounded-xl border border-white/5 bg-ink-900/50 p-3 text-xs"
        >
          <p class="font-mono text-brand-light">{{ entry.alias }}</p>
          <p class="mt-1 break-all text-muted">{{ entry.owner }}</p>
        </div>
        <button class="btn-primary" :disabled="importForm.busy" @click="importDo">
          {{ importForm.busy ? '导入中…' : '导入到 keystores 目录' }}
        </button>
      </div>
    </section>

    <!-- 重新生成 -->
    <section v-if="tab === 'generate'" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand"></span>
        证书信息
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">别名 Alias</label>
          <input v-model="form.cert.alias" class="field-input" />
        </div>
        <div>
          <label class="field-label">通用名 CN</label>
          <input v-model="form.cert.commonName" class="field-input" placeholder="My App" />
        </div>
        <div>
          <label class="field-label">组织 O</label>
          <input v-model="form.cert.organization" class="field-input" />
        </div>
        <div>
          <label class="field-label">组织单位 OU</label>
          <input v-model="form.cert.orgUnit" class="field-input" />
        </div>
        <div>
          <label class="field-label">城市 L</label>
          <input v-model="form.cert.locality" class="field-input" />
        </div>
        <div>
          <label class="field-label">省份 ST</label>
          <input v-model="form.cert.state" class="field-input" />
        </div>
        <div>
          <label class="field-label">国家 C</label>
          <input v-model="form.cert.country" class="field-input" maxlength="2" />
        </div>
        <div>
          <label class="field-label">有效期（天）</label>
          <input v-model.number="form.cert.validity" type="number" class="field-input" />
        </div>
        <div>
          <label class="field-label">密钥算法</label>
          <select v-model="form.cert.keyAlgorithm" class="field-input">
            <option value="RSA">RSA</option>
            <option value="EC">EC (secp256r1)</option>
          </select>
        </div>
        <div>
          <label class="field-label">密钥长度</label>
          <select v-model.number="form.cert.keySize" class="field-input" :disabled="form.cert.keyAlgorithm === 'EC'">
            <option :value="2048">2048</option>
            <option :value="4096">4096</option>
          </select>
        </div>
      </div>

      <div class="mt-4 grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">密钥库密码</label>
          <input v-model="form.storePass" type="password" class="field-input" autocomplete="new-password" />
        </div>
        <div>
          <label class="field-label">密钥密码（留空同密钥库密码）</label>
          <input v-model="form.keyPass" type="password" class="field-input" autocomplete="new-password" />
        </div>
      </div>

      <div class="mt-4">
        <label class="field-label">保存路径（默认 keystores 目录，.jks / .p12 决定密钥库类型）</label>
        <div class="flex gap-2">
          <input v-model="form.outputPath" class="field-input" placeholder="新文件或已有密钥库" />
          <button class="btn-ghost shrink-0" @click="chooseSave">新建</button>
          <button class="btn-ghost shrink-0" @click="chooseExistingTarget">选择已有</button>
        </div>
        <p class="mt-2 text-[11px] leading-relaxed text-muted">
          提示：若保存路径指向<strong class="text-slate-300">已存在的密钥库</strong>，将向其<strong class="text-slate-300">追加新别名</strong>（需输入该库密码，且别名不可重复）。
        </p>
      </div>

      <div class="mt-5">
        <button class="btn-primary" :disabled="busy" @click="generate">
          {{ busy ? '处理中…' : '生成密钥库' }}
        </button>
      </div>
    </section>

    <!-- 查看·导出 -->
    <section v-if="tab === 'view'" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-sky-400"></span>
        打开已有密钥库
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <button class="btn-primary" @click="chooseExisting">选择密钥库文件…</button>
        <span class="text-xs text-muted">支持 .jks / .keystore / .p12 / .pfx</span>
      </div>

      <div v-if="recentKeystores.length" class="mt-4">
        <p class="mb-2 text-[11px] uppercase tracking-wider text-muted">最近使用</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="p in recentKeystores"
            :key="p"
            class="max-w-xs truncate rounded-lg border border-white/5 bg-white/[0.03] px-3 py-1.5 text-xs text-slate-300 transition hover:border-brand/40 hover:bg-brand/5 hover:text-brand-light"
            :title="p"
            @click="openRecent(p)"
          >
            {{ basename(p) }}
          </button>
        </div>
      </div>

      <div class="mt-5 grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">密钥库文件</label>
          <div class="flex gap-2">
            <input v-model="viewPath" class="field-input" />
            <button class="btn-ghost shrink-0" @click="chooseExisting">浏览</button>
          </div>
        </div>
        <div>
          <label class="field-label">密钥库密码</label>
          <div class="flex gap-2">
            <input v-model="viewPass" type="password" class="field-input" ref="viewPassInput" />
            <button class="btn-primary shrink-0" :disabled="busy" @click="view">查看</button>
          </div>
        </div>
      </div>

      <div v-if="info?.entries?.length" class="mt-5 space-y-4">
        <div
          v-for="entry in info.entries"
          :key="entry.alias"
          class="rounded-xl border border-white/5 bg-ink-900/50 p-4"
        >
          <div class="mb-3 flex items-center justify-between">
            <div>
              <p class="font-mono text-sm text-brand-light">{{ entry.alias }}</p>
              <p class="text-[11px] text-muted">{{ entry.entryType }} · {{ entry.storeType || info.type }}</p>
            </div>
            <div class="flex gap-2">
              <button class="btn-ghost !py-1 !text-xs" @click="useEntryAlias(entry.alias)">选中为导出别名</button>
              <button
                class="btn-ghost !py-1 !text-xs !text-red-300 hover:!bg-red-500/10"
                :disabled="busy"
                @click="deleteAlias(entry.alias)"
              >删除别名</button>
            </div>
          </div>
          <dl class="grid gap-2 text-xs md:grid-cols-2">
            <div>
              <dt class="text-muted">所有者</dt>
              <dd class="stat-value break-all">{{ entry.owner }}</dd>
            </div>
            <div>
              <dt class="text-muted">签发人</dt>
              <dd class="stat-value break-all">{{ entry.issuer }}</dd>
            </div>
            <div>
              <dt class="text-muted">序列号</dt>
              <dd class="stat-value">{{ entry.serial }}</dd>
            </div>
            <div>
              <dt class="text-muted">有效期</dt>
              <dd class="stat-value">{{ entry.validFrom }} → {{ entry.validTo }}</dd>
            </div>
            <div class="md:col-span-2">
              <dt class="text-muted">MD5</dt>
              <dd class="kbd-text">{{ entry.md5 }}</dd>
            </div>
            <div class="md:col-span-2">
              <dt class="text-muted">SHA1</dt>
              <dd class="kbd-text">{{ entry.sha1 }}</dd>
            </div>
            <div class="md:col-span-2">
              <dt class="text-muted">SHA256</dt>
              <dd class="kbd-text">{{ entry.sha256 }}</dd>
            </div>
          </dl>
        </div>

        <div class="rounded-xl border border-white/5 bg-ink-900/50 p-4">
          <p class="mb-3 text-sm text-white">导出证书</p>
          <div class="grid gap-3 md:grid-cols-3">
            <input v-model="exportTarget.alias" class="field-input" placeholder="别名" />
            <input v-model="exportTarget.output" class="field-input" placeholder="输出文件" />
            <button class="btn-ghost" :disabled="busy" @click="exportCert">导出 PEM 证书</button>
          </div>
        </div>
      </div>
    </section>

    <!-- 格式转换 -->
    <section v-if="tab === 'convert'" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-brand-light"></span>
        JKS / PKCS12 互转
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="field-label">源密钥库</label>
          <input v-model="convert.src" class="field-input" />
        </div>
        <div>
          <label class="field-label">源密码</label>
          <input v-model="convert.srcPass" type="password" class="field-input" />
        </div>
        <div>
          <label class="field-label">目标密钥库</label>
          <input v-model="convert.dst" class="field-input" placeholder="如 release.p12" />
        </div>
        <div>
          <label class="field-label">目标密码</label>
          <input v-model="convert.dstPass" type="password" class="field-input" />
        </div>
        <div>
          <label class="field-label">目标类型</label>
          <select v-model="convert.type" class="field-input">
            <option value="PKCS12">PKCS12</option>
            <option value="JKS">JKS</option>
          </select>
        </div>
      </div>
      <button class="btn-primary mt-4" :disabled="busy" @click="convertKeystore">开始转换</button>
    </section>

    <!-- 首页密码弹窗 -->
    <div
      v-if="homeDialog.open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      @click.self="homeDialog.open = false"
    >
      <div class="w-full max-w-md rounded-2xl border border-white/10 bg-ink-900 p-5 shadow-xl">
        <p class="text-sm font-medium text-white">输入密码：{{ homeDialog.name }}</p>
        <p class="mt-1 break-all text-[11px] text-muted">{{ homeDialog.path }}</p>
        <input
          v-model="homeDialog.pass"
          type="password"
          class="field-input mt-3"
          ref="homePassInput"
          placeholder="密钥库密码"
          @keyup.enter="homeSubmit"
        />
        <p v-if="homeDialog.error" class="mt-2 text-xs text-red-300">{{ homeDialog.error }}</p>
        <label class="mt-3 flex items-center gap-2 text-xs text-slate-300">
          <input type="checkbox" v-model="homeDialog.remember" />
          记住密码（加密保存到本地配置）
        </label>
        <div class="mt-4 flex justify-end gap-2">
          <button class="btn-ghost" @click="homeDialog.open = false">取消</button>
          <button class="btn-primary" :disabled="homeDialog.busy" @click="homeSubmit">
            {{ homeDialog.busy ? '校验中…' : '查看' }}
          </button>
        </div>
      </div>
    </div>

    <section v-if="detail" class="panel">
      <div class="panel-title">
        <span class="h-1.5 w-1.5 rounded-full bg-slate-400"></span>
        keytool 输出
      </div>
      <pre class="max-h-60 overflow-auto whitespace-pre-wrap font-mono text-[11px] text-slate-300">{{ detail }}</pre>
    </section>
  </div>
</template>

// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

import { reactive } from 'vue'
import zhCN from './zh-CN'
import en from './en'

export type Locale = 'zh-CN' | 'en'

// 支持的语言清单（供设置页下拉使用）。
export const locales: { value: Locale; label: string }[] = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en', label: 'English' },
]

const catalogs: Record<Locale, Record<string, string>> = {
  'zh-CN': zhCN,
  en,
}

// 单一事实源：当前界面语言。模板中调用 t() 时会自动追踪该响应式状态。
export const i18nState = reactive<{ locale: Locale }>({ locale: 'zh-CN' })

// initLocale 在应用启动时根据后端配置初始化界面语言（不触发持久化）。
export function initLocale(locale?: string) {
  const l = (locale || 'zh-CN') as Locale
  i18nState.locale = l === 'en' ? 'en' : 'zh-CN'
}

// setLocale 切换界面语言（持久化由调用方负责：写回 settings + 后端 SetLanguage）。
export function setLocale(locale: Locale) {
  i18nState.locale = locale
}

// t 翻译指定 key；支持位置参数 {0} {1} 或命名参数 {name}。
// 兼容直接传字符串/数字的场景（如 t('key', count)），按 [param] 处理。
// 回退链：当前语言 → 中文 → key 原文，保证缺失翻译不崩。
export function t(key: string, params?: Record<string, unknown> | unknown[] | string): string {
  const cat = catalogs[i18nState.locale] ?? catalogs['zh-CN']
  let msg: string = cat[key] ?? catalogs['zh-CN'][key] ?? key
  if (params) {
    const args = typeof params === 'object' && params !== null ? params : [params]
    if (Array.isArray(args)) {
      args.forEach((v, i) => {
        msg = msg.replace(new RegExp(`\\{${i}\\}`, 'g'), String(v))
      })
    } else {
      Object.entries(args).forEach(([k, v]) => {
        msg = msg.replace(new RegExp(`\\{${escapeRegExp(String(k))}\\}`, 'g'), String(v))
      })
    }
  }
  return msg
}

// escapeRegExp 转义正则特殊字符，避免参数名/值中的元字符破坏替换。
function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

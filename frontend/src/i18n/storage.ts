import { defaultLanguage, isLanguage, type Language } from '.'

const storageKey = 'lang'

/** 无头浏览器默认 en-US，要验中文得显式设 localStorage。 */
export function readStoredLanguage(): Language | null {
  const stored = window.localStorage.getItem(storageKey)
  return isLanguage(stored) ? stored : null
}

export function storeLanguage(lang: Language): void {
  window.localStorage.setItem(storageKey, lang)
}

/** 用户没选过语言时按浏览器偏好猜一个，服务端协商结果只作参考。 */
export function detectBrowserLanguage(): Language {
  const candidates = navigator.languages ?? [navigator.language]
  for (const candidate of candidates) {
    if (candidate.toLowerCase().startsWith('zh')) {
      return 'zh-CN'
    }
    if (candidate.toLowerCase().startsWith('en')) {
      return 'en'
    }
  }
  return defaultLanguage
}

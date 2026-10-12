import type { SiteInfo } from './api/site'
import type { Language } from './i18n'

export interface AppDebugState {
  name: string | null
  avatarLoaded: boolean
  site: SiteInfo | null
  lang: Language
  view: number
}

/** 排查时用 CDP 读状态，比翻日志快。 */
export interface AppDebug {
  state(): AppDebugState
  showView(index: number): void
  setLanguage(lang: Language): void
}

declare global {
  interface Window {
    __app?: AppDebug
  }
}

export function exposeDebug(debug: AppDebug): void {
  window.__app = debug
}

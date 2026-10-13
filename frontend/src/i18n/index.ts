export type Language = 'zh-CN' | 'en'

export const languages: readonly Language[] = ['zh-CN', 'en']

export const defaultLanguage: Language = 'zh-CN'

// 中文表是基准：键集合从它派生，英文表少一条或写错一条都过不了 typecheck。
const zhCN = {
  'site.title': '{name} - 个人主页',
  'hero.avatar-alt': '{name} 的头像',
  'hero.scroll-hint': '向下滑动查看更多',
  'record.label': '备案信息',
  'record.copyright': 'Copyright：© {year} {name}',
  // 按钮显示的是「当前语言」，不是「会切到什么语言」
  'language.name': '中文',
  'language.switch-aria': '切换到英文',
  'app.profile-error': '站点信息加载失败，请稍后重试。',
} as const

export type MessageKey = keyof typeof zhCN

const tables: Record<Language, Record<MessageKey, string>> = {
  'zh-CN': zhCN,
  en: {
    'site.title': '{name} - Homepage',
    'hero.avatar-alt': 'Avatar of {name}',
    'hero.scroll-hint': 'Scroll for more',
    'record.label': 'Filing information',
    'record.copyright': 'Copyright © {year} {name}',
    'language.name': 'English',
    'language.switch-aria': 'Switch to Chinese',
    'app.profile-error': 'Failed to load the site profile, please retry later.',
  },
}

export type TextArgs = Record<string, string | number>

export function isLanguage(value: unknown): value is Language {
  return value === 'zh-CN' || value === 'en'
}

export function otherLanguage(lang: Language): Language {
  return lang === 'zh-CN' ? 'en' : 'zh-CN'
}

/** 未配置的键原样返回键名：漏配要能在界面上一眼看见，不做回退。 */
export function t(lang: Language, key: MessageKey, args: TextArgs = {}): string {
  const template = tables[lang][key] ?? key
  return template.replace(/\{(\w+)\}/g, (placeholder, name: string) =>
    name in args ? String(args[name]) : placeholder,
  )
}

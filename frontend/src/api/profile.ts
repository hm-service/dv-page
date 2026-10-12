import { isLanguage, type Language } from '../i18n'

/** 头像是二进制资源，交给浏览器按普通图片加载与缓存（ETag 在服务端）。 */
export const avatarPath = '/api/avatar'

export interface SiteInfo {
  icp: string
  icpUrl: string
  copyrightYear: number
}

export interface Profile {
  name: string
  site: SiteInfo
  /** 服务端按 Accept-Language / ?lang= 协商出来的语言。 */
  lang: Language
}

function parseSiteInfo(payload: unknown): SiteInfo {
  if (typeof payload !== 'object' || payload === null) {
    throw new Error('profile.site is not an object')
  }

  const record = payload as Record<string, unknown>
  const { icp, icpUrl, copyrightYear } = record

  if (typeof icp !== 'string' || icp === '') {
    throw new Error('profile.site has no icp')
  }
  if (typeof icpUrl !== 'string' || icpUrl === '') {
    throw new Error('profile.site has no icpUrl')
  }
  if (typeof copyrightYear !== 'number' || !Number.isInteger(copyrightYear)) {
    throw new Error('profile.site has no copyrightYear')
  }

  return { icp, icpUrl, copyrightYear }
}

function parseProfile(payload: unknown): Profile {
  if (typeof payload !== 'object' || payload === null) {
    throw new Error('profile response is not an object')
  }

  const record = payload as Record<string, unknown>
  const { name, site, lang } = record

  if (typeof name !== 'string' || name === '') {
    throw new Error('profile has no name')
  }
  if (!isLanguage(lang)) {
    throw new Error(`profile has unsupported lang ${String(lang)}`)
  }

  return { name, site: parseSiteInfo(site), lang }
}

export async function fetchProfile(preferred: Language | null): Promise<Profile> {
  const query = preferred ? `?lang=${encodeURIComponent(preferred)}` : ''
  const response = await fetch(`/api/profile${query}`, {
    headers: { accept: 'application/json' },
  })
  if (!response.ok) {
    throw new Error(`GET /api/profile responded ${response.status}`)
  }

  return parseProfile(await response.json())
}

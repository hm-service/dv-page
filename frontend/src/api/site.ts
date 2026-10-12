import { isLanguage, type Language } from '../i18n'

export interface SiteInfo {
  icp: string
  icpUrl: string
  copyrightYear: number
  /** 服务端按 Accept-Language / ?lang= 协商出来的语言。 */
  lang: Language
}

function parseSiteInfo(payload: unknown): SiteInfo {
  if (typeof payload !== 'object' || payload === null) {
    throw new Error('site response is not an object')
  }

  const record = payload as Record<string, unknown>
  const { icp, icpUrl, copyrightYear, lang } = record

  if (typeof icp !== 'string' || icp === '') {
    throw new Error('site has no icp')
  }
  if (typeof icpUrl !== 'string' || icpUrl === '') {
    throw new Error('site has no icpUrl')
  }
  if (typeof copyrightYear !== 'number' || !Number.isInteger(copyrightYear)) {
    throw new Error('site has no copyrightYear')
  }
  if (!isLanguage(lang)) {
    throw new Error(`site has unsupported lang ${String(lang)}`)
  }

  return { icp, icpUrl, copyrightYear, lang }
}

export async function fetchSiteInfo(preferred: Language | null): Promise<SiteInfo> {
  const query = preferred ? `?lang=${encodeURIComponent(preferred)}` : ''
  const response = await fetch(`/api/site${query}`, {
    headers: { accept: 'application/json' },
  })
  if (!response.ok) {
    throw new Error(`GET /api/site responded ${response.status}`)
  }

  return parseSiteInfo(await response.json())
}

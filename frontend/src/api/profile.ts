export interface Profile {
  name: string
  /** 头像的对象 URL：由接口给的 base64 解出来，页面销毁时要 revoke。 */
  avatarUrl: string
}

interface RawProfile {
  name: string
  avatar: string
}

function parseProfile(payload: unknown): RawProfile {
  if (typeof payload !== 'object' || payload === null) {
    throw new Error('profile response is not an object')
  }

  const record = payload as Record<string, unknown>
  const { name, avatar } = record

  if (typeof name !== 'string' || name === '') {
    throw new Error('profile has no name')
  }
  if (typeof avatar !== 'string' || avatar === '') {
    throw new Error('profile has no avatar')
  }

  return { name, avatar }
}

/**
 * base64 → 对象 URL。不解码图片格式：交给浏览器按内容嗅探，换 PNG/WebP 都不用改前端。
 */
function toObjectUrl(base64: string): string {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }

  return URL.createObjectURL(new Blob([bytes]))
}

export async function fetchProfile(): Promise<Profile> {
  const response = await fetch('/api/profile', { headers: { accept: 'application/json' } })
  if (!response.ok) {
    throw new Error(`GET /api/profile responded ${response.status}`)
  }

  const raw = parseProfile(await response.json())

  return { name: raw.name, avatarUrl: toObjectUrl(raw.avatar) }
}

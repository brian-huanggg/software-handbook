// API client for both auth modes.
//
// Session mode: the server sets an HttpOnly cookie; we just send credentials.
// Token mode:   the access JWT lives only in this module's memory (never in
//               localStorage, so an XSS cannot read it from storage), and the
//               refresh token is an HttpOnly cookie that JS cannot see.

export type Mode = 'session' | 'token'

export const API_URL: string =
  import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export interface Me {
  user_id: string
  roles: string[]
}

interface TokenResponse {
  access_token: string
  token_type: string
  expires_in: number
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

let accessToken: string | null = null

export const getAccessToken = () => accessToken

interface Init {
  method?: string
  body?: unknown
  bearer?: string | null
}

function send(path: string, { method = 'GET', body, bearer }: Init = {}) {
  const headers: Record<string, string> = {}
  // CSRF: a custom header on every state-changing request. A cross-site page
  // cannot send it without a CORS preflight, which the API only approves for
  // its own allowed origin.
  if (method !== 'GET') headers['X-Requested-With'] = 'XMLHttpRequest'
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (bearer) headers['Authorization'] = `Bearer ${bearer}`
  return fetch(API_URL + path, {
    method,
    headers,
    credentials: 'include', // send/receive the cookies
    body: body === undefined ? undefined : JSON.stringify(body),
  })
}

async function parse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let message = res.statusText
    try {
      message = ((await res.json()) as { error?: string }).error ?? message
    } catch {
      /* body was not JSON */
    }
    throw new ApiError(res.status, message)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

// Several requests can discover an expired access token at the same moment.
// They must share ONE refresh call: the server rotates the refresh token on
// every use, so a second concurrent call would look like token theft and log
// the user out.
let inflight: Promise<string> | null = null

export function refreshAccessToken(): Promise<string> {
  inflight ??= (async () => {
    try {
      const r = await parse<TokenResponse>(
        await send('/token/refresh', { method: 'POST' }),
      )
      accessToken = r.access_token
      return accessToken
    } catch (e) {
      accessToken = null
      throw e
    } finally {
      inflight = null
    }
  })()
  return inflight
}

export async function register(email: string, password: string) {
  // 202 for both a new and an already-registered email: the API does not
  // reveal which accounts exist.
  return parse<{ status: string }>(
    await send('/register', { method: 'POST', body: { email, password } }),
  )
}

export async function login(mode: Mode, email: string, password: string) {
  const body = { email, password }
  if (mode === 'session') {
    await parse(await send('/session/login', { method: 'POST', body }))
    return
  }
  const r = await parse<TokenResponse>(
    await send('/token/login', { method: 'POST', body }),
  )
  accessToken = r.access_token
}

export async function logout(mode: Mode) {
  accessToken = null
  await parse(
    await send(mode === 'session' ? '/session/logout' : '/token/logout', {
      method: 'POST',
    }),
  )
}

export async function me(mode: Mode): Promise<Me> {
  if (mode === 'session') return parse<Me>(await send('/session/me'))

  let res = await send('/token/me', { bearer: accessToken })
  if (res.status === 401) {
    // Access token missing or expired: refresh once, then retry.
    res = await send('/token/me', { bearer: await refreshAccessToken() })
  }
  return parse<Me>(res)
}

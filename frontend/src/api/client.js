const apiBase = import.meta.env.VITE_API_BASE || '/api'
let authToken = localStorage.getItem('accessToken') || ''

export { apiBase }

export function setAuthToken(token) {
  authToken = token || ''
  if (authToken) localStorage.setItem('accessToken', authToken)
  else localStorage.removeItem('accessToken')
}

export function getAuthToken() {
  return authToken
}

export async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) }
  if (!(options.body instanceof FormData) && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }
  if (authToken) {
    headers.Authorization = `Bearer ${authToken}`
  }

  const response = await fetch(`${apiBase}${path}`, {
    headers,
    ...options,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }))
    throw new Error(error.message || error.error || response.statusText)
  }

  const result = await response.json()
  if (result && typeof result === 'object' && 'code' in result && 'data' in result) {
    if (result.code !== 0) throw new Error(result.message || response.statusText)
    return result.data
  }
  return result
}

export async function requestBlob(path, options = {}) {
  const headers = { ...(options.headers || {}) }
  if (authToken) {
    headers.Authorization = `Bearer ${authToken}`
  }

  const response = await fetch(`${apiBase}${path}`, {
    headers,
    ...options,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }))
    throw new Error(error.message || error.error || response.statusText)
  }

  return response.blob()
}

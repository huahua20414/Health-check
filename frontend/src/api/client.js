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
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) }
  if (authToken) {
    headers.Authorization = `Bearer ${authToken}`
  }

  const response = await fetch(`${apiBase}${path}`, {
    headers,
    ...options,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(error.error || response.statusText)
  }

  return response.json()
}

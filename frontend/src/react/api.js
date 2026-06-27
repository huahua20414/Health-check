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
  const tokenForRequest = authToken
  if (tokenForRequest) headers.Authorization = `Bearer ${tokenForRequest}`

  const response = await fetch(`${apiBase}${path}`, { headers, ...options })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }))
    const message = error.message || error.error || response.statusText
    if (response.status === 401 && path !== '/auth/login') clearStaleSession(tokenForRequest)
    throw new Error(localizeApiError(message))
  }
  const result = await response.json()
  if (result && typeof result === 'object' && 'code' in result && 'data' in result) {
    if (result.code !== 0) throw new Error(localizeApiError(result.message || response.statusText))
    return result.data
  }
  return result
}

export async function requestBlob(path, options = {}) {
  const headers = { ...(options.headers || {}) }
  const tokenForRequest = authToken
  if (tokenForRequest) headers.Authorization = `Bearer ${tokenForRequest}`
  const response = await fetch(`${apiBase}${path}`, { headers, ...options })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }))
    const message = error.message || error.error || response.statusText
    if (response.status === 401) clearStaleSession(tokenForRequest)
    throw new Error(localizeApiError(message))
  }
  return response.blob()
}

function clearStaleSession(tokenForRequest) {
  if (!tokenForRequest || tokenForRequest !== authToken) return
  setAuthToken('')
  localStorage.removeItem('currentUser')
  window.dispatchEvent(new CustomEvent('auth-expired'))
}

function localizeApiError(message) {
  const text = String(message || '').trim()
  if (!text) return '操作失败，请稍后重试'
  if (text.includes('Field validation') || text.includes('ShouldBindJSON')) return '请检查必填项和输入格式'
  const exact = {
    'email code requests are too frequent': '验证码发送太频繁，请稍后再试',
    'generate email code failed': '验证码生成失败，请稍后重试',
    'save email code failed': '验证码保存失败，请稍后重试',
    'send email code failed': '验证码邮件发送失败，请检查邮箱或稍后重试',
    'email code expired': '验证码已过期，请重新获取',
    'invalid email code': '验证码不正确，请重新输入',
    'invalid email or code': '邮箱未注册或验证码不正确',
    'too many requests': '操作太频繁，请稍后再试',
    'email already exists': '该邮箱已注册，请直接登录',
    'account is not active': '账号暂未启用，请联系管理员或等待审核',
    'issue token failed': '登录凭证生成失败，请稍后再试',
    'name is required': '请输入姓名',
    'invalid id card': '身份证号无效',
    'appointment already exists for this time slot': '这个时间段已经有预约了，不能重复预约',
    'schedule slot overlaps with existing slot': '该医生在这个时间段已有号源，请调整时间后再试',
    'capacity cannot be lower than booked count': '容量不能小于已预约人数',
    'missing bearer token': '登录已过期，请重新登录',
    'invalid token': '登录已过期，请重新登录',
    'session expired': '登录状态已失效，请重新登录',
    'invalid user': '登录状态已失效，请重新登录',
    Unauthorized: '登录已过期，请重新登录',
    Forbidden: '没有权限执行该操作',
    'Not Found': '请求的资源不存在',
  }
  return exact[text] || text
}

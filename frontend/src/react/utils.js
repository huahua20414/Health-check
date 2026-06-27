export const appointmentTypes = ['个人体检', '入职体检', '年度体检', '复查体检']
export const doctorDepartments = ['健康管理科', '内科', '影像科', '妇科', '老年医学科', '检验科', '心电科']
export const specialtyOptions = ['入职体检', '慢病筛查', '年度综合', '影像专项', '女性专项', '老年体检']
export const devAuthEnabled = import.meta.env.VITE_DEV_AUTH === 'true'
export const devAuthShortcutEmail = 'huahua20414@foxmail.com'

export function statusText(status) {
  return {
    booked: '已预约',
    checked: '已体检',
    reported: '已出报告',
    canceled: '已取消',
    waiting: '候补中',
    promoted: '已递补',
    active: '启用',
    pending: '待审核',
    disabled: '停用',
    deleted: '已归档',
    available: '可预约',
    full: '已满',
    draft: '草稿',
    published: '已发布',
    hidden: '已隐藏',
    unread: '未读',
    read: '已读',
    open: '待处理',
    replied: '已回复',
    closed: '已关闭',
    none: '未申请',
    requested: '已申请',
    issued: '已开具',
  }[status] || status || '-'
}

export function statusKind(status) {
  return {
    booked: 'warn',
    checked: 'info',
    reported: 'success',
    canceled: 'muted',
    waiting: 'violet',
    promoted: 'success',
    active: 'success',
    pending: 'warn',
    disabled: 'danger',
    deleted: 'muted',
    available: 'success',
    full: 'danger',
    draft: 'muted',
    published: 'success',
    hidden: 'warn',
    unread: 'warn',
    read: 'muted',
    open: 'warn',
    replied: 'success',
    closed: 'muted',
    none: 'muted',
    requested: 'warn',
    issued: 'success',
  }[status] || 'muted'
}

export function paymentStatusText(status) {
  return { paid: '已支付', unpaid: '未支付', refunded: '已退款' }[status] || status || '-'
}

export function announcementAudienceText(audience) {
  return { user: '用户公告', doctor: '医生公告', all: '全部公告', admin: '管理员公告' }[audience] || audience || '-'
}

export function moneyText(value) {
  const amount = Number(value)
  if (!Number.isFinite(amount)) return '-'
  return `￥${amount.toFixed(2)}`
}

export function formatDate(value) {
  if (!value) return '-'
  const text = String(value)
  const datePart = text.match(/^(\d{4}-\d{2}-\d{2})/)?.[1]
  if (datePart) return datePart
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return localDateString(date)
}

export function localDateString(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function normalizeIDCard(value) {
  return String(value || '').trim().toUpperCase()
}

export function isValidIDCard(value) {
  const id = normalizeIDCard(value)
  if (!/^\d{17}[\dX]$/.test(id)) return false
  const weights = [7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2]
  const checks = '10X98765432'
  const sum = weights.reduce((total, weight, index) => total + Number(id[index]) * weight, 0)
  return id[17] === checks[sum % 11]
}

export function calculateAgeFromIDCard(value, now = new Date()) {
  const id = normalizeIDCard(value)
  if (!/^\d{17}[\dXx]$/.test(id)) return null
  if (!isValidIDCard(id)) return null
  const birth = id.slice(6, 14)
  const year = Number(birth.slice(0, 4))
  const month = Number(birth.slice(4, 6))
  const day = Number(birth.slice(6, 8))
  const date = new Date(year, month - 1, day)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) return null
  let age = now.getFullYear() - year
  const currentMonth = now.getMonth() + 1
  const currentDay = now.getDate()
  if (currentMonth < month || (currentMonth === month && currentDay < day)) age -= 1
  return age >= 0 && age < 130 ? age : null
}

export function assertEmail(email) {
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(String(email || '').trim())) throw new Error('请输入有效邮箱')
}

export function assertCode(code) {
  if (!/^\d{6}$/.test(String(code || '').trim())) throw new Error('请输入 6 位邮箱验证码')
}

export function assertRequired(value, message) {
  if (value === undefined || value === null || String(value).trim() === '') throw new Error(message)
}

export function assertIDCard(value, required = false) {
  const id = normalizeIDCard(value)
  if (!id && !required) return
  if (calculateAgeFromIDCard(id) === null) throw new Error('请输入有效的 18 位身份证号，最后一位可为 X')
}

export function toQuery(params = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') query.set(key, value)
  })
  return query.toString()
}

export function nextDateString() {
  const date = new Date()
  date.setDate(date.getDate() + 1)
  return localDateString(date)
}

export function homePath(role) {
  if (role === 'doctor') return '/doctor'
  if (role === 'admin') return '/admin'
  return '/'
}

export function appointmentOriginalAmount(appointment) {
  const amount = Number(appointment?.originalAmount)
  if (Number.isFinite(amount) && amount > 0) return amount
  const packagePrice = Number(appointment?.package?.price)
  return Number.isFinite(packagePrice) ? packagePrice : 0
}

export function appointmentDiscountAmount(appointment) {
  const amount = Number(appointment?.discountAmount)
  return Number.isFinite(amount) && amount > 0 ? amount : 0
}

export function appointmentPayableAmount(appointment) {
  const amount = Number(appointment?.payableAmount)
  if (Number.isFinite(amount) && (amount > 0 || appointmentDiscountAmount(appointment) > 0)) return amount
  return Math.max(0, appointmentOriginalAmount(appointment) - appointmentDiscountAmount(appointment))
}

export function downloadBlob(filename, blob) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

function pngFilename(filename) {
  const base = String(filename || 'checkup-report').replace(/\.[^.]+$/, '')
  return `${base}.png`
}

export function documentHTML(title, rows, footer) {
  const escape = (value) => String(value || '').replace(/[&<>"']/g, (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]))
  const cells = rows.map(([label, value]) => `<div class=\"label\">${escape(label)}</div><div class=\"value\">${escape(value)}</div>`).join('')
  return `<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>${escape(title)}</title><style>body{font-family:Arial,\"Microsoft YaHei\",sans-serif;margin:0;background:#f3f6fa;color:#111827}.doc{max-width:860px;margin:32px auto;background:#fff;border:1px solid #d8e2ec;padding:32px}.head{border-bottom:3px solid #19c2d9;padding-bottom:16px;margin-bottom:24px}.head h1{margin:0;font-size:28px}.grid{display:grid;grid-template-columns:160px 1fr;border-top:1px solid #e3ebf2;border-left:1px solid #e3ebf2}.label,.value{padding:13px 16px;border-right:1px solid #e3ebf2;border-bottom:1px solid #e3ebf2}.label{font-weight:700;background:#f8fafc}.value{white-space:pre-wrap}.footer{margin-top:24px;color:#667085}</style></head><body><main class=\"doc\"><section class=\"head\"><h1>${escape(title)}</h1><p>熙心健康体检管理系统</p></section><section class=\"grid\">${cells}</section><p class=\"footer\">${escape(footer)}</p></main></body></html>`
}

export function downloadHTML(filename, html) {
  downloadBlob(filename, new Blob([html], { type: 'text/html;charset=utf-8' }))
}

export function downloadReportImage(filename, rows, footer) {
  const width = 1200
  const margin = 64
  const labelWidth = 220
  const rowGap = 0
  const contentWidth = width - margin * 2 - labelWidth
  const font = '"Microsoft YaHei", "PingFang SC", Arial, sans-serif'
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  ctx.font = `30px ${font}`

  const normalizedRows = rows.map(([label, value]) => {
    const text = String(value || '-')
    const lines = wrapCanvasText(ctx, text, contentWidth - 56)
    return { label, lines, height: Math.max(74, lines.length * 38 + 34) }
  })
  const gridHeight = normalizedRows.reduce((sum, row) => sum + row.height + rowGap, 0)
  const height = margin + 112 + 40 + gridHeight + 96
  const scale = window.devicePixelRatio || 1
  canvas.width = width * scale
  canvas.height = height * scale
  canvas.style.width = `${width}px`
  canvas.style.height = `${height}px`
  ctx.scale(scale, scale)

  ctx.fillStyle = '#f3f6fa'
  ctx.fillRect(0, 0, width, height)
  ctx.fillStyle = '#ffffff'
  ctx.strokeStyle = '#d8e2ec'
  ctx.lineWidth = 2
  ctx.fillRect(32, 32, width - 64, height - 64)
  ctx.strokeRect(32, 32, width - 64, height - 64)

  ctx.fillStyle = '#111827'
  ctx.font = `700 42px ${font}`
  ctx.fillText('体检报告详情', margin, 98)
  ctx.fillStyle = '#667085'
  ctx.font = `24px ${font}`
  ctx.fillText('熙心健康体检管理系统', margin, 136)
  ctx.strokeStyle = '#19c2d9'
  ctx.lineWidth = 5
  ctx.beginPath()
  ctx.moveTo(margin, 164)
  ctx.lineTo(width - margin, 164)
  ctx.stroke()

  let y = 204
  for (const row of normalizedRows) {
    ctx.fillStyle = '#f8fafc'
    ctx.fillRect(margin, y, labelWidth, row.height)
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(margin + labelWidth, y, contentWidth, row.height)
    ctx.strokeStyle = '#e3ebf2'
    ctx.lineWidth = 1
    ctx.strokeRect(margin, y, labelWidth, row.height)
    ctx.strokeRect(margin + labelWidth, y, contentWidth, row.height)

    ctx.fillStyle = '#111827'
    ctx.font = `700 25px ${font}`
    ctx.fillText(String(row.label), margin + 28, y + 45)
    ctx.font = `25px ${font}`
    row.lines.forEach((line, index) => ctx.fillText(line, margin + labelWidth + 28, y + 45 + index * 38))
    y += row.height + rowGap
  }

  ctx.fillStyle = '#667085'
  ctx.font = `22px ${font}`
  ctx.fillText(footer, margin, y + 52)
  canvas.toBlob((blob) => {
    if (blob) downloadBlob(pngFilename(filename), blob)
  }, 'image/png')
}

function wrapCanvasText(ctx, text, maxWidth) {
  const lines = []
  for (const paragraph of String(text || '-').split('\n')) {
    let line = ''
    for (const char of paragraph) {
      const next = line + char
      if (line && ctx.measureText(next).width > maxWidth) {
        lines.push(line)
        line = char
      } else {
        line = next
      }
    }
    lines.push(line || ' ')
  }
  return lines
}

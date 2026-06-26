import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { request, requestBlob, setAuthToken, getAuthToken } from './api'
import {
  appointmentTypes,
  assertCode,
  assertEmail,
  assertIDCard,
  assertRequired,
  devAuthEnabled,
  devAuthShortcutEmail,
  documentHTML,
  downloadBlob,
  downloadHTML,
  formatDate,
  homePath,
  moneyText,
  nextDateString,
  paymentStatusText,
  statusText,
  toQuery,
} from './utils'

const HealthContext = createContext(null)

const emptyPagination = { page: 1, pageSize: 10, total: 0 }

const defaultForms = {
  login: { email: devAuthShortcutEmail, code: '', role: 'user' },
  userRegister: { name: '', email: '', code: '', gender: '', idCard: '' },
  doctorRegister: { name: '', email: '', code: '', employeeNo: '', department: '', title: '' },
  appointment: { appointmentType: '个人体检', institutionId: '', packageId: '', familyMemberId: '', slotId: '', couponId: '', date: '', period: '', note: '', paymentStatus: 'unpaid', invoiceTitle: '', invoiceTaxNo: '', selectedPackageItemIds: [] },
  waitlist: { appointmentType: '个人体检', institutionId: '', packageId: '', date: '', period: '', note: '' },
  profile: { name: '', gender: '', age: 0, idCard: '', email: '', avatarUrl: '', bio: '', emailNotify: true },
  adminUser: { id: null, name: '', gender: '', idCard: '', email: '', avatarUrl: '', bio: '', emailNotify: true, status: 'active' },
  email: { email: '', code: '' },
  familyMember: { id: null, name: '', relation: '', gender: '', age: null, idCard: '', phone: '' },
  reschedule: { appointmentId: null, institutionId: '', slotId: '', date: '', period: '', note: '' },
  invoice: { appointmentId: null, invoiceTitle: '', invoiceTaxNo: '' },
  package: { id: null, name: '', category: '年度综合', description: '', price: 0, items: '', status: 'active' },
  institution: { id: null, name: '', address: '', phone: '', openHours: '', status: 'active' },
  coupon: { id: null, name: '', code: '', type: 'amount', value: 0, minAmount: 0, packageId: '', status: 'active', startDate: '', endDate: '', description: '' },
  review: { appointmentId: '', rating: 5, content: '' },
  reviewReply: { id: null, reply: '', status: 'published' },
  announcement: { id: null, title: '', content: '', audience: 'all', status: 'draft' },
  notification: { userId: '', role: 'user', channel: 'in_app', type: 'admin_notice', title: '', content: '' },
  supportTicket: { subject: '', content: '' },
  supportTicketReply: { id: null, reply: '', status: 'replied' },
  reminder: { date: nextDateString() },
  systemSetting: { id: null, key: '', label: '', value: '', valueType: 'string', group: '', status: 'active', description: '' },
  checkupItem: { id: null, name: '', category: '', department: '', price: 0, durationMin: 10, description: '', status: 'active' },
  packageItem: { id: null, packageId: '', itemId: '', sortOrder: 0, required: true },
  schedule: { id: null, doctorId: '', institutionId: '', date: '', period: '上午', category: '', startTime: '08:30', endTime: '09:00', capacity: 1, status: 'available' },
  report: { appointmentId: '', summary: '', conclusion: '', recommendation: '' },
}

const dataKeys = [
  'packages', 'appointments', 'reports', 'users', 'institutions', 'institutionRows', 'slots', 'scheduleSlotRows',
  'waitlist', 'mailLogs', 'familyMembers', 'favorites', 'browseHistories', 'popularPackages', 'recommendedPackages',
  'notifications', 'adminNotifications', 'supportTickets', 'adminSupportTickets', 'loginLogs', 'operationLogs',
  'rolePermissions', 'rolePermissionRows', 'permissionCodes', 'systemSettings', 'systemSettingRows', 'coupons',
  'activeCoupons', 'reviews', 'announcements', 'activeAnnouncements', 'checkupItems', 'checkupItemRows', 'packageItems',
  'doctorUsers', 'pendingDoctorUsers', 'activeDoctors',
]

function initialData() {
  return Object.fromEntries(dataKeys.map((key) => [key, []]))
}

function initialPaginations() {
  return Object.fromEntries(['appointments', 'users', 'doctors', 'pendingDoctors', 'activeDoctors', 'reports', 'waitlist', 'mailLogs', 'loginLogs', 'operationLogs', 'rolePermissions', 'systemSettings', 'packages', 'institutions', 'notifications', 'adminNotifications', 'supportTickets', 'adminSupportTickets', 'familyMembers', 'coupons', 'reviews', 'announcements', 'checkupItems', 'packageItems', 'slots'].map((key) => [key, { ...emptyPagination }]))
}

function useObjectState(initial) {
  const [value, setValue] = useState(initial)
  const patch = useCallback((updates) => setValue((current) => ({ ...current, ...updates })), [])
  const reset = useCallback((next = initial) => setValue(next), [initial])
  return [value, patch, reset, setValue]
}

export function HealthProvider({ children }) {
  const [currentUser, setCurrentUser] = useState(() => JSON.parse(localStorage.getItem('currentUser') || 'null'))
  const [data, setData] = useState(initialData)
  const [supportInfo, setSupportInfo] = useState({ customerServiceUrl: '', customerServiceHours: '', faq: [] })
  const [adminDashboard, setAdminDashboard] = useState({ summary: {}, appointmentTrend: [], packageSales: [], paymentStatus: [], userGrowth: [] })
  const [paginations, setPaginations] = useState(initialPaginations)
  const [forms, setForms] = useState(defaultForms)
  const [loading, setLoading] = useState({})
  const [authCodeCooldown, setAuthCodeCooldown] = useState(0)
  const [toast, setToast] = useState(null)
  const bootstrapped = useRef(false)
  const role = currentUser?.role || ''
  const isAuthenticated = Boolean(getAuthToken() && currentUser)
  const isUser = role === 'user'
  const isDoctor = role === 'doctor'
  const isAdmin = role === 'admin'
  const can = useCallback((permission) => data.permissionCodes.includes(permission), [data.permissionCodes])

  const setLoadingKey = useCallback((key, value) => setLoading((current) => ({ ...current, [key]: value })), [])
  const notify = useCallback((type, message) => {
    setToast({ id: Date.now(), type, message })
    window.clearTimeout(notify.timer)
    notify.timer = window.setTimeout(() => setToast(null), 3200)
  }, [])
  const dismissToast = useCallback(() => {
    window.clearTimeout(notify.timer)
    setToast(null)
  }, [notify])
  const saveUser = useCallback((user) => {
    setCurrentUser(user)
    if (user) localStorage.setItem('currentUser', JSON.stringify(user))
    else localStorage.removeItem('currentUser')
    if (user) {
      setForms((current) => ({
        ...current,
        profile: { ...current.profile, name: user.name || '', gender: user.gender || '', age: user.age || 0, idCard: user.idCard || '', email: user.email || '', avatarUrl: user.avatarUrl || '', bio: user.bio || '', emailNotify: user.emailNotify !== false },
        email: { ...current.email, email: user.email || '', code: '' },
      }))
    }
  }, [])
  const updateData = useCallback((patch) => setData((current) => ({ ...current, ...patch })), [])
  const updateForm = useCallback((key, patch) => setForms((current) => ({ ...current, [key]: { ...current[key], ...patch } })), [])
  const resetForm = useCallback((key) => setForms((current) => ({ ...current, [key]: defaultForms[key] })), [])
  const setLoginShortcut = useCallback((role) => {
    if (!devAuthEnabled) return
    setForms((current) => ({ ...current, login: { ...current.login, email: devAuthShortcutEmail, code: '', role } }))
  }, [])

  useEffect(() => {
    if (!authCodeCooldown) return undefined
    const timer = window.setInterval(() => setAuthCodeCooldown((value) => Math.max(0, value - 1)), 1000)
    return () => window.clearInterval(timer)
  }, [authCodeCooldown])

  useEffect(() => {
    const handleAuthExpired = () => {
      saveUser(null)
      setData(initialData())
    }
    window.addEventListener('auth-expired', handleAuthExpired)
    return () => window.removeEventListener('auth-expired', handleAuthExpired)
  }, [saveUser])

  const requestPage = useCallback(async (path, key, params = {}) => {
    const query = toQuery(params)
    const result = await request(`${path}${query ? `?${query}` : ''}`)
    const pagination = result.pagination || (Array.isArray(result.items) ? {
      page: result.page,
      pageSize: result.pageSize,
      total: result.total,
    } : {})
    setPaginations((current) => ({ ...current, [key]: { ...current[key], ...pagination } }))
    return result.items || result
  }, [])

  const loadMyPermissions = useCallback(async () => {
    if (!getAuthToken()) return
    const result = await request('/permissions/me').catch(() => ({ permissions: [] }))
    updateData({ permissionCodes: result.permissions || [] })
  }, [updateData])

  const loadAll = useCallback(async () => {
    if (loading.load) return
    setLoadingKey('load', true)
    try {
      const storedUser = JSON.parse(localStorage.getItem('currentUser') || 'null')
      const userRole = storedUser?.role || currentUser?.role
      const announcementPath = userRole === 'user' || userRole === 'doctor' ? `/announcements/active?audience=${userRole}` : '/announcements/active'
      const [packages, popularPackages, recommendedPackages, activeCoupons, activeAnnouncements, institutions, support] = await Promise.all([
        request('/packages'),
        request('/packages/popular'),
        request('/packages/recommended'),
        request('/coupons/active'),
        request(announcementPath),
        request('/institutions'),
        request('/support'),
      ])
      const base = { packages, popularPackages, recommendedPackages, activeCoupons, activeAnnouncements, institutions }
      setSupportInfo(support)
      if (!getAuthToken()) {
        updateData(base)
        return
      }
      const protectedData = { ...base, users: currentUser ? [currentUser] : [] }
      if (userRole === 'user') {
        const [favorites, browseHistories] = await Promise.all([
          request('/package-favorites'),
          request('/package-browses?page=1&pageSize=5').then((result) => result.items || result),
        ])
        Object.assign(protectedData, { favorites, browseHistories })
      }
      if (userRole === 'admin') {
        const [dashboard, pendingDoctors] = await Promise.all([
          request('/admin/dashboard'),
          request('/users?role=doctor&status=pending&page=1&pageSize=8').then((result) => result.items || result),
        ])
        protectedData.pendingDoctorUsers = pendingDoctors
        setAdminDashboard(dashboard)
      }
      updateData(protectedData)
      setForms((current) => ({
        ...current,
        appointment: {
          ...current.appointment,
          institutionId: current.appointment.institutionId || protectedData.institutions?.[0]?.id || '',
          packageId: current.appointment.packageId || protectedData.packages?.[0]?.id || '',
        },
        report: { ...current.report, appointmentId: current.report.appointmentId || protectedData.appointments?.[0]?.id || '' },
      }))
    } finally {
      setLoadingKey('load', false)
    }
  }, [currentUser, loading.load, setLoadingKey, updateData])

  const ensureBootstrapped = useCallback(async () => {
    if (bootstrapped.current) return
    bootstrapped.current = true
    if (getAuthToken()) {
      const user = await request('/auth/me').catch(() => null)
      if (user) {
        saveUser(user)
        await loadMyPermissions()
      } else {
        setAuthToken('')
        saveUser(null)
      }
    }
    await loadAll()
  }, [loadAll, loadMyPermissions, saveUser])

  const sendAuthEmailCode = useCallback(async (email) => {
    if (loading.authCode) return notify('warn', '验证码正在发送，请稍候')
    if (authCodeCooldown > 0) return notify('warn', `${authCodeCooldown} 秒后可重新发送验证码`)
    assertEmail(email)
    setLoadingKey('authCode', true)
    try {
      await request('/auth/email-code', { method: 'POST', body: JSON.stringify({ email }) })
      setAuthCodeCooldown(60)
      notify('success', '验证码已发送，请查看邮箱')
    } finally {
      setLoadingKey('authCode', false)
    }
  }, [authCodeCooldown, loading.authCode, notify, setLoadingKey])

  const login = useCallback(async () => {
    const form = forms.login
    assertEmail(form.email)
    const shortcutLogin = form.email.trim().toLowerCase() === devAuthShortcutEmail && ['1', '2', '3'].includes(String(form.code).trim())
    if (!shortcutLogin && !devAuthEnabled) assertCode(form.code)
    setLoadingKey('login', true)
    try {
      const result = await request('/auth/login', { method: 'POST', body: JSON.stringify(form) })
      setAuthToken(result.accessToken)
      saveUser(result.user)
      notify('success', '登录成功')
      Promise.allSettled([loadMyPermissions(), loadAll()]).catch(() => null)
      return result.user
    } finally {
      setLoadingKey('login', false)
    }
  }, [forms.login, loadAll, loadMyPermissions, notify, saveUser, setLoadingKey])

  const registerUser = useCallback(async () => {
    const form = forms.userRegister
    assertRequired(form.name, '请输入姓名')
    assertEmail(form.email)
    assertCode(form.code)
    assertIDCard(form.idCard, true)
    setLoadingKey('register', true)
    try {
      const result = await request('/auth/register/user', { method: 'POST', body: JSON.stringify(form) })
      setAuthToken(result.accessToken)
      saveUser(result.user)
      await loadMyPermissions()
      await loadAll()
      notify('success', '注册成功，已自动登录')
      return result.user
    } finally {
      setLoadingKey('register', false)
    }
  }, [forms.userRegister, loadAll, loadMyPermissions, notify, saveUser, setLoadingKey])

  const registerDoctor = useCallback(async () => {
    const form = forms.doctorRegister
    assertRequired(form.name, '请输入姓名')
    assertEmail(form.email)
    assertCode(form.code)
    assertRequired(form.employeeNo, '请输入工号')
    assertRequired(form.department, '请选择科室')
    assertRequired(form.title, '请输入职称')
    setLoadingKey('register', true)
    try {
      await request('/auth/register/doctor', { method: 'POST', body: JSON.stringify(form) })
      notify('success', '医生注册已提交，审核通过后可登录')
    } finally {
      setLoadingKey('register', false)
    }
  }, [forms.doctorRegister, notify, setLoadingKey])

  const logout = useCallback(async () => {
    setLoadingKey('logout', true)
    try {
      if (getAuthToken()) await request('/auth/logout', { method: 'POST' }).catch(() => null)
      setAuthToken('')
      saveUser(null)
      setData(initialData())
    } finally {
      setLoadingKey('logout', false)
    }
  }, [saveUser, setLoadingKey])

  const loadPage = useCallback(async (path, key, stateKey = key, params = {}) => {
    const rows = await requestPage(path, key, params)
    updateData({ [stateKey]: rows })
    return rows
  }, [requestPage, updateData])

  const loaders = useMemo(() => ({
    loadAppointmentsPage: (params = {}) => loadPage('/appointments', 'appointments', 'appointments', params),
    loadReportsPage: (params = {}) => loadPage('/reports', 'reports', 'reports', params),
    loadUsersPage: (params = {}, key = 'users', stateKey = 'users') => loadPage('/users', key, stateKey, params),
    loadWaitlistPage: (params = {}) => loadPage('/waitlist', 'waitlist', 'waitlist', params),
    loadMailLogsPage: (params = {}) => loadPage('/mail-logs', 'mailLogs', 'mailLogs', params),
    loadLoginLogsPage: (params = {}) => loadPage('/login-logs', 'loginLogs', 'loginLogs', params),
    loadOperationLogsPage: (params = {}) => loadPage('/operation-logs', 'operationLogs', 'operationLogs', params),
    loadRolePermissionsPage: (params = {}) => loadPage('/role-permissions', 'rolePermissions', 'rolePermissionRows', params),
    loadSystemSettingsPage: (params = {}) => loadPage('/system-settings', 'systemSettings', 'systemSettingRows', params),
    loadPackagesPage: (params = {}) => loadPage('/packages', 'packages', 'packages', params),
    loadInstitutionsPage: (params = {}) => loadPage('/institutions', 'institutions', 'institutionRows', params),
    loadNotificationsPage: (params = {}) => loadPage('/notifications', 'notifications', 'notifications', params),
    loadSupportTicketsPage: (params = {}) => loadPage('/support-tickets', 'supportTickets', 'supportTickets', params),
    loadFamilyMembersPage: (params = {}) => loadPage('/family-members', 'familyMembers', 'familyMembers', params),
    loadAdminNotificationsPage: (params = {}) => loadPage('/admin/notifications', 'adminNotifications', 'adminNotifications', params),
    loadAdminSupportTicketsPage: (params = {}) => loadPage('/admin/support-tickets', 'adminSupportTickets', 'adminSupportTickets', params),
    loadCouponsPage: (params = {}) => loadPage('/coupons', 'coupons', 'coupons', params),
    loadReviewsPage: (params = {}) => loadPage('/reviews', 'reviews', 'reviews', params),
    loadAnnouncementsPage: (params = {}) => loadPage('/announcements', 'announcements', 'announcements', params),
    loadCheckupItemsPage: (params = {}) => loadPage('/checkup-items', 'checkupItems', 'checkupItemRows', params),
    loadPackageItemsPage: (params = {}) => loadPage('/package-items', 'packageItems', 'packageItems', params),
    loadSlotsPage: (params = {}, key = 'slots', stateKey = 'scheduleSlotRows') => loadPage('/schedule/slots', key, stateKey, params),
  }), [loadPage])

  const paginationActions = useMemo(() => ({
    setPaginationPage: (key, page) => setPaginations((current) => ({ ...current, [key]: { ...current[key], page } })),
    setPaginationPageSize: (key, pageSize) => setPaginations((current) => ({ ...current, [key]: { ...current[key], pageSize, page: 1 } })),
  }), [])

  const action = useCallback(async (key, success, fn) => {
    if (loading[key]) return
    setLoadingKey(key, true)
    try {
      const result = await fn()
      if (success) notify('success', typeof success === 'function' ? success(result) : success)
      return result
    } finally {
      setLoadingKey(key, false)
    }
  }, [loading, notify, setLoadingKey])

  const actions = useMemo(() => ({
    ...loaders,
    loadRolePermissions: async () => updateData({ rolePermissions: await request('/role-permissions') }),
    loadSystemSettings: async () => updateData({ systemSettings: await request('/system-settings') }),
    loadSupportInfo: async () => setSupportInfo(await request('/support')),
    loadInstitutions: async () => updateData({ institutions: await request('/institutions') }),
    loadAdminDashboard: async (params = {}) => setAdminDashboard(await request(`/admin/dashboard${toQuery(params) ? `?${toQuery(params)}` : ''}`)),
    createAppointment: () => action('appointment', (r) => r?.type === 'waitlist' ? '当前号源已满，已自动加入候补' : '预约成功，医生和时间已分配', async () => {
      const result = await request('/appointments', { method: 'POST', body: JSON.stringify(forms.appointment) })
      await loadAll()
      return result
    }),
    joinWaitlist: (slot) => action('appointment', '已提交候补请求', async () => {
      const body = { ...forms.waitlist, appointmentType: forms.appointment.appointmentType, institutionId: forms.appointment.institutionId, packageId: forms.appointment.packageId, date: forms.appointment.date, period: slot?.period || forms.appointment.period, note: forms.appointment.note, slotId: slot?.id || 0 }
      await request('/appointments', { method: 'POST', body: JSON.stringify(body) })
      await loadAll()
    }),
    cancelAppointment: (row) => action('status', '预约已取消', async () => { await request(`/appointments/${row.id}/cancel`, { method: 'PATCH' }); await loaders.loadAppointmentsPage({ page: 1, pageSize: 20 }) }),
    cancelWaitlist: (row) => action('status', '候补已取消', async () => { await request(`/waitlist/${row.id}/cancel`, { method: 'PATCH' }); await loaders.loadWaitlistPage() }),
    updateAppointmentPayment: (appointment, paymentStatus) => action('appointment', paymentStatus === 'paid' ? '支付状态已标记为已支付' : '已撤销支付状态', async () => { await request(`/appointments/${appointment.id}/payment`, { method: 'PATCH', body: JSON.stringify({ paymentStatus }) }); await loaders.loadAppointmentsPage() }),
    saveInvoice: () => action('appointment', '发票信息已保存', async () => { await request(`/appointments/${forms.invoice.appointmentId}/invoice`, { method: 'PATCH', body: JSON.stringify({ invoiceTitle: forms.invoice.invoiceTitle, invoiceTaxNo: forms.invoice.invoiceTaxNo }) }); await loaders.loadAppointmentsPage() }),
    updateAppointmentInvoiceStatus: (appointment, invoiceStatus) => action('appointment', '发票状态已更新', async () => { await request(`/appointments/${appointment.id}/invoice/status`, { method: 'PATCH', body: JSON.stringify({ invoiceStatus }) }); await loaders.loadAppointmentsPage() }),
    createReview: () => action('review', '评价已提交', async () => { await request('/reviews', { method: 'POST', body: JSON.stringify({ appointmentId: forms.review.appointmentId, rating: Number(forms.review.rating || 5), content: forms.review.content }) }); updateForm('review', defaultForms.review); await loaders.loadReviewsPage() }),
    markNotificationRead: (n) => action('notification', '', async () => { await request(`/notifications/${n.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: 'read' }) }); await loaders.loadNotificationsPage() }),
    createSupportTicket: () => action('notification', '咨询已提交', async () => { await request('/support-tickets', { method: 'POST', body: JSON.stringify(forms.supportTicket) }); resetForm('supportTicket'); await loaders.loadSupportTicketsPage() }),
    saveFamilyMember: () => action('familyMember', '家庭成员已保存', async () => {
      const body = JSON.stringify({ name: forms.familyMember.name, relation: forms.familyMember.relation, gender: forms.familyMember.gender, idCard: forms.familyMember.idCard, phone: forms.familyMember.phone })
      assertRequired(forms.familyMember.name, '请输入成员姓名')
      assertRequired(forms.familyMember.relation, '请输入成员关系')
      assertIDCard(forms.familyMember.idCard)
      if (forms.familyMember.id) await request(`/family-members/${forms.familyMember.id}`, { method: 'PATCH', body })
      else await request('/family-members', { method: 'POST', body })
      resetForm('familyMember')
      await loaders.loadFamilyMembersPage({ page: 1, pageSize: 20 })
    }),
    deleteFamilyMember: (member) => action('familyMember', '家庭成员已删除', async () => { await request(`/family-members/${member.id}`, { method: 'DELETE' }); await loaders.loadFamilyMembersPage({ page: 1, pageSize: 20 }) }),
    toggleFavorite: (pkg) => action('favorite', data.favorites.some((i) => i.packageId === pkg.id) ? '已取消收藏' : '已收藏套餐', async () => { const exists = data.favorites.some((i) => i.packageId === pkg.id); await request(`/package-favorites/${pkg.id}`, { method: exists ? 'DELETE' : 'POST' }); await loadAll() }),
    recordPackageBrowse: async (pkg) => { if (getAuthToken() && role === 'user' && pkg?.id) await request(`/packages/${pkg.id}/browse`, { method: 'POST' }).catch(() => null) },
    saveProfile: () => action('profile', '个人资料已保存', async () => {
      assertRequired(forms.profile.name, '请输入姓名')
      assertIDCard(forms.profile.idCard)
      const user = await request('/profile', { method: 'PATCH', body: JSON.stringify(forms.profile) })
      saveUser(user)
      await loadAll()
    }),
    sendEmailCode: () => action('emailCode', '验证码已发送，请查看目标邮箱', async () => request('/profile/email-code', { method: 'POST', body: JSON.stringify({ email: forms.email.email }) })),
    updateEmail: () => action('emailUpdate', '邮箱已验证并更新', async () => { const user = await request('/profile/email', { method: 'PATCH', body: JSON.stringify(forms.email) }); saveUser(user); await loadAll() }),
    markDone: (row) => action('status', '已标记完成体检，可继续生成报告', async () => { await request(`/appointments/${row.id}/status`, { method: 'PATCH', body: JSON.stringify({ status: 'checked' }) }); updateForm('report', { appointmentId: row.id }); await loaders.loadAppointmentsPage({ page: 1, pageSize: 20 }) }),
    createReport: () => action('report', '报告已生成', async () => { await request('/reports', { method: 'POST', body: JSON.stringify(forms.report) }); await loaders.loadReportsPage({ page: 1, pageSize: 20 }); await loaders.loadAppointmentsPage({ page: 1, pageSize: 20 }) }),
    updateUserStatus: (user, status) => action('status', '状态已更新', async () => { await request(`/users/${user.id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }); await loadAll() }),
    saveAdminUser: () => action('adminUser', '用户资料已保存', async () => {
      assertRequired(forms.adminUser.name, '请输入姓名')
      assertEmail(forms.adminUser.email)
      assertIDCard(forms.adminUser.idCard)
      const user = await request(`/users/${forms.adminUser.id}`, { method: 'PATCH', body: JSON.stringify(forms.adminUser) })
      resetForm('adminUser')
      return user
    }),
    updateDoctorProfile: (user, payload) => action('doctorProfile', '医生资料已更新', async () => { await request(`/users/${user.id}/doctor-profile`, { method: 'PATCH', body: JSON.stringify({ ...payload, specialties: Array.isArray(payload.specialties) ? payload.specialties.join(',') : payload.specialties }) }); await loadAll() }),
    savePackage: () => action('package', '套餐已保存', async () => {
      const body = JSON.stringify({ ...forms.package, price: Number(forms.package.price || 0) })
      if (forms.package.id) await request(`/packages/${forms.package.id}`, { method: 'PATCH', body })
      else await request('/packages', { method: 'POST', body })
      resetForm('package')
      await loadAll()
    }),
    archivePackage: (pkg) => action('package', '套餐已归档', async () => { await request(`/packages/${pkg.id}`, { method: 'DELETE' }); await loaders.loadPackagesPage() }),
    saveInstitution: () => action('institution', '体检机构已保存', async () => { const body = JSON.stringify(forms.institution); if (forms.institution.id) await request(`/institutions/${forms.institution.id}`, { method: 'PATCH', body }); else await request('/institutions', { method: 'POST', body }); resetForm('institution'); await loaders.loadInstitutionsPage() }),
    archiveInstitution: (row) => action('institution', '体检机构已归档', async () => { await request(`/institutions/${row.id}`, { method: 'DELETE' }); await loaders.loadInstitutionsPage() }),
    saveCheckupItem: () => action('checkupItem', '体检项目已保存', async () => { const body = JSON.stringify({ ...forms.checkupItem, price: Number(forms.checkupItem.price || 0), durationMin: Number(forms.checkupItem.durationMin || 10) }); if (forms.checkupItem.id) await request(`/checkup-items/${forms.checkupItem.id}`, { method: 'PATCH', body }); else await request('/checkup-items', { method: 'POST', body }); resetForm('checkupItem'); await loaders.loadCheckupItemsPage() }),
    archiveCheckupItem: (row) => action('checkupItem', '体检项目已归档', async () => { await request(`/checkup-items/${row.id}`, { method: 'DELETE' }); await loaders.loadCheckupItemsPage() }),
    savePackageItem: () => action('packageItem', '套餐项目组合已保存', async () => {
      const body = JSON.stringify({
        id: forms.packageItem.id,
        packageId: Number(forms.packageItem.packageId || 0),
        itemId: Number(forms.packageItem.itemId || 0),
        sortOrder: Number(forms.packageItem.sortOrder || 0),
        required: forms.packageItem.required !== false,
      })
      if (forms.packageItem.id) await request(`/package-items/${forms.packageItem.id}`, { method: 'PATCH', body })
      else await request('/package-items', { method: 'POST', body })
      resetForm('packageItem')
      await loaders.loadPackageItemsPage()
    }),
    deletePackageItem: (row) => action('packageItem', '套餐项目已移除', async () => { await request(`/package-items/${row.id}`, { method: 'DELETE' }); await loaders.loadPackageItemsPage() }),
    saveScheduleSlot: () => action('schedule', '排班号源已保存', async () => {
      const body = JSON.stringify({
        ...forms.schedule,
        doctorId: Number(forms.schedule.doctorId || 0),
        institutionId: Number(forms.schedule.institutionId || 0),
        capacity: Number(forms.schedule.capacity || 1),
      })
      if (forms.schedule.id) await request(`/schedule/slots/${forms.schedule.id}`, { method: 'PATCH', body })
      else await request('/schedule/slots', { method: 'POST', body })
      resetForm('schedule')
      await loaders.loadSlotsPage()
    }),
    archiveScheduleSlot: (row) => action('schedule', '排班号源已归档', async () => { await request(`/schedule/slots/${row.id}`, { method: 'DELETE' }); await loaders.loadSlotsPage() }),
    saveCoupon: () => action('coupon', '优惠券已保存', async () => { const body = JSON.stringify({ ...forms.coupon, value: Number(forms.coupon.value || 0), minAmount: Number(forms.coupon.minAmount || 0), packageId: Number(forms.coupon.packageId || 0) }); if (forms.coupon.id) await request(`/coupons/${forms.coupon.id}`, { method: 'PATCH', body }); else await request('/coupons', { method: 'POST', body }); resetForm('coupon'); await loaders.loadCouponsPage() }),
    archiveCoupon: (row) => action('coupon', '优惠券已归档', async () => { await request(`/coupons/${row.id}`, { method: 'DELETE' }); await loaders.loadCouponsPage() }),
    saveReviewReply: () => action('review', '评价处理已保存', async () => { await request(`/reviews/${forms.reviewReply.id}/reply`, { method: 'PATCH', body: JSON.stringify(forms.reviewReply) }); resetForm('reviewReply'); await loaders.loadReviewsPage() }),
    saveAnnouncement: () => action('announcement', '公告已保存', async () => { const body = JSON.stringify(forms.announcement); if (forms.announcement.id) await request(`/announcements/${forms.announcement.id}`, { method: 'PATCH', body }); else await request('/announcements', { method: 'POST', body }); resetForm('announcement'); await loaders.loadAnnouncementsPage() }),
    archiveAnnouncement: (row) => action('announcement', '公告已归档', async () => { await request(`/announcements/${row.id}`, { method: 'DELETE' }); await loaders.loadAnnouncementsPage() }),
    sendAdminNotification: () => action('adminNotification', '通知已发送', async () => { await request('/admin/notifications', { method: 'POST', body: JSON.stringify(forms.notification) }); resetForm('notification'); await loaders.loadAdminNotificationsPage() }),
    sendCheckupReminders: (payload) => action('reminder', '体检前提醒已生成', async () => { await request('/admin/notifications/reminders', { method: 'POST', body: JSON.stringify(payload || forms.reminder) }); await loaders.loadAdminNotificationsPage() }),
    updateAdminNotificationStatus: (row, status) => action('adminNotification', status === 'archived' ? '通知已归档' : '通知状态已更新', async () => { if (status === 'archived') await request(`/admin/notifications/${row.id}`, { method: 'DELETE' }); else await request(`/admin/notifications/${row.id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }); await loaders.loadAdminNotificationsPage() }),
    saveSupportTicketReply: () => action('adminNotification', '客服工单已处理', async () => { await request(`/admin/support-tickets/${forms.supportTicketReply.id}/reply`, { method: 'PATCH', body: JSON.stringify(forms.supportTicketReply) }); resetForm('supportTicketReply'); await loaders.loadAdminSupportTicketsPage() }),
    updateRolePermission: (permission) => action('permission', '权限配置已更新', async () => { await request(`/role-permissions/${permission.id}`, { method: 'PATCH', body: JSON.stringify({ enabled: permission.enabled }) }); await loaders.loadRolePermissionsPage() }),
    updateSystemSetting: (setting) => action('systemSetting', '配置已保存', async () => { await request(`/system-settings/${setting.id}`, { method: 'PATCH', body: JSON.stringify(setting) }); await loaders.loadSystemSettingsPage() }),
    exportBlob: (path, filename, key) => action(key, 'CSV 已导出', async () => downloadBlob(filename, await requestBlob(path))),
    importFile: (path, file, key, after) => action(key, '导入完成', async () => { const formData = new FormData(); formData.append('file', file); await request(path, { method: 'POST', body: formData }); if (after) await after() }),
    downloadAppointment: (appointment) => downloadHTML(`appointment-${appointment.orderNo || appointment.id}.html`, documentHTML('体检预约订单', [['订单号', appointment.orderNo], ['客户', appointment.user?.name], ['机构', appointment.institution?.name], ['套餐', appointment.package?.name], ['日期', appointment.date], ['时段', `${appointment.startTime || ''}-${appointment.endTime || ''}`], ['支付状态', paymentStatusText(appointment.paymentStatus)], ['状态', statusText(appointment.status)]], '请按预约时间携带有效证件到检。')),
    downloadReport: (report) => downloadHTML(`report-${report.reportNo || report.id}.html`, documentHTML('体检报告详情', [['报告编号', report.reportNo], ['客户', report.user?.name], ['套餐', report.appointment?.package?.name], ['医生', report.doctor?.name], ['检查摘要', report.summary], ['体检结论', report.conclusion], ['健康建议', report.recommendation], ['报告时间', formatDate(report.createdAt)]], '本报告仅供健康管理参考。')),
  }), [action, data.favorites, forms, loadAll, loaders, notify, role, saveUser, updateData, updateForm, resetForm])

  const derived = useMemo(() => {
    const peopleMap = new Map()
    if (isAdmin) data.users.forEach((u) => peopleMap.set(`user-${u.id}`, u))
    data.appointments.forEach((item) => { if (item.user?.id) peopleMap.set(`user-${item.user.id}`, { ...item.user, source: '预约客户' }) })
    data.reports.forEach((report) => {
      if (report.user?.id) peopleMap.set(`user-${report.user.id}`, { ...report.user, source: '报告客户' })
    })
    if (currentUser?.id && currentUser.role === 'user') peopleMap.set(`current-${currentUser.role}-${currentUser.id}`, { ...currentUser, source: '当前登录' })
    return {
      role,
      isAuthenticated,
      isUser,
      isDoctor,
      isAdmin,
      myAppointments: data.appointments.filter((item) => item.userId === currentUser?.id || isDoctor || isAdmin),
      bookedCount: data.appointments.filter((item) => item.status === 'booked').length,
      reportedCount: data.appointments.filter((item) => item.status === 'reported').length,
      pendingDoctorCount: data.appointments.filter((item) => item.status !== 'reported').length,
      pendingDoctors: data.pendingDoctorUsers.length ? data.pendingDoctorUsers : data.users.filter((item) => item.role === 'doctor' && item.status === 'pending'),
      peopleRows: Array.from(peopleMap.values()),
    }
  }, [currentUser, data.appointments, data.pendingDoctorUsers, data.reports, data.users, isAdmin, isAuthenticated, isDoctor, isUser, role])

  const value = useMemo(() => ({
    currentUser,
    ...data,
    supportInfo,
    adminDashboard,
    paginations,
    forms,
    loading,
    toast,
    authCodeCooldown,
    appointmentTypes,
    can,
    notify,
    dismissToast,
    updateForm,
    resetForm,
    ensureBootstrapped,
    sendAuthEmailCode,
    setLoginShortcut,
    login,
    registerUser,
    registerDoctor,
    logout,
    loadAll,
    homePath,
    ...derived,
    ...actions,
    ...paginationActions,
  }), [actions, adminDashboard, authCodeCooldown, can, currentUser, data, derived, dismissToast, ensureBootstrapped, forms, loadAll, loading, login, logout, notify, paginationActions, paginations, registerDoctor, registerUser, resetForm, sendAuthEmailCode, setLoginShortcut, supportInfo, toast, updateForm])

  return <HealthContext.Provider value={value}>{children}</HealthContext.Provider>
}

export function useHealth() {
  const context = useContext(HealthContext)
  if (!context) throw new Error('useHealth must be used inside HealthProvider')
  return context
}

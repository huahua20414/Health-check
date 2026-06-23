import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { request, requestBlob, setAuthToken, getAuthToken } from '../api/client'

export const appointmentTypes = ['个人体检', '入职体检', '年度体检', '复查体检']
export const devAuthEnabled = import.meta.env.VITE_DEV_AUTH === 'true'
export const doctorDepartments = ['健康管理科', '内科', '影像科', '妇科', '老年医学科', '检验科', '心电科']
export const specialtyOptions = ['入职体检', '慢病筛查', '年度综合', '影像专项', '女性专项', '老年体检']

const loginForm = reactive({ email: 'huahua20414@foxmail.com', password: '123456', code: '', role: 'user' })
const userRegisterForm = reactive({ name: '', email: '', code: '', password: '', confirmPassword: '', gender: '', age: null, idCard: '' })
const doctorRegisterForm = reactive({
  name: '',
  email: '',
  code: '',
  password: '',
  confirmPassword: '',
  employeeNo: '',
  department: '',
  title: '',
})
const currentUser = ref(JSON.parse(localStorage.getItem('currentUser') || 'null'))
const packages = ref([])
const appointments = ref([])
const reports = ref([])
const users = ref([])
const institutions = ref([])
const institutionRows = ref([])
const slots = ref([])
const scheduleSlotRows = ref([])
const waitlist = ref([])
const mailLogs = ref([])
const familyMembers = ref([])
const favorites = ref([])
const browseHistories = ref([])
const popularPackages = ref([])
const recommendedPackages = ref([])
const notifications = ref([])
const adminNotifications = ref([])
const supportTickets = ref([])
const adminSupportTickets = ref([])
const loginLogs = ref([])
const operationLogs = ref([])
const rolePermissions = ref([])
const rolePermissionRows = ref([])
const permissionCodes = ref([])
const systemSettings = ref([])
const systemSettingRows = ref([])
const coupons = ref([])
const activeCoupons = ref([])
const reviews = ref([])
const announcements = ref([])
const activeAnnouncements = ref([])
const supportInfo = ref({ customerServiceUrl: '', customerServiceHours: '', faq: [] })
const adminDashboard = ref({ summary: {}, appointmentTrend: [], packageSales: [], userGrowth: [] })
const checkupItems = ref([])
const checkupItemRows = ref([])
const packageItems = ref([])
const paginations = reactive({
  appointments: { page: 1, pageSize: 10, total: 0 },
  users: { page: 1, pageSize: 10, total: 0 },
  doctors: { page: 1, pageSize: 10, total: 0 },
  reports: { page: 1, pageSize: 6, total: 0 },
  waitlist: { page: 1, pageSize: 10, total: 0 },
  mailLogs: { page: 1, pageSize: 10, total: 0 },
  loginLogs: { page: 1, pageSize: 10, total: 0 },
  operationLogs: { page: 1, pageSize: 10, total: 0 },
  rolePermissions: { page: 1, pageSize: 10, total: 0 },
  systemSettings: { page: 1, pageSize: 10, total: 0 },
  packages: { page: 1, pageSize: 10, total: 0 },
  institutions: { page: 1, pageSize: 10, total: 0 },
  notifications: { page: 1, pageSize: 10, total: 0 },
  adminNotifications: { page: 1, pageSize: 10, total: 0 },
  supportTickets: { page: 1, pageSize: 10, total: 0 },
  adminSupportTickets: { page: 1, pageSize: 10, total: 0 },
  coupons: { page: 1, pageSize: 10, total: 0 },
  reviews: { page: 1, pageSize: 10, total: 0 },
  announcements: { page: 1, pageSize: 10, total: 0 },
  checkupItems: { page: 1, pageSize: 10, total: 0 },
  packageItems: { page: 1, pageSize: 10, total: 0 },
  slots: { page: 1, pageSize: 10, total: 0 },
})
const appointmentForm = reactive({
  appointmentType: '个人体检',
  institutionId: null,
  packageId: null,
  familyMemberId: null,
  slotId: null,
  couponId: null,
  date: '',
  period: '',
  note: '',
  paymentStatus: 'unpaid',
  invoiceTitle: '',
  invoiceTaxNo: '',
})
const waitlistForm = reactive({ appointmentType: '个人体检', institutionId: null, packageId: null, date: '', period: '', note: '' })
const profileForm = reactive({ name: '', gender: '', age: 0, idCard: '', email: '', avatarUrl: '', bio: '', emailNotify: true })
const emailForm = reactive({ email: '', code: '' })
const familyMemberForm = reactive({ id: null, name: '', relation: '', gender: '', age: null, idCard: '', phone: '' })
const rescheduleForm = reactive({ appointmentId: null, institutionId: null, slotId: null, date: '', period: '', note: '' })
const invoiceForm = reactive({ appointmentId: null, invoiceTitle: '', invoiceTaxNo: '' })
const packageForm = reactive({ id: null, name: '', category: '年度综合', description: '', price: 0, items: '', status: 'active' })
const institutionForm = reactive({ id: null, name: '', address: '', phone: '', openHours: '', status: 'active' })
const couponForm = reactive({ id: null, name: '', code: '', type: 'amount', value: 0, minAmount: 0, packageId: null, status: 'active', startDate: '', endDate: '', description: '' })
const reviewForm = reactive({ appointmentId: null, rating: 5, content: '' })
const reviewReplyForm = reactive({ id: null, reply: '', status: 'published' })
const announcementForm = reactive({ id: null, title: '', content: '', audience: 'all', status: 'draft' })
const notificationForm = reactive({ userId: null, role: 'user', channel: 'in_app', type: 'admin_notice', title: '', content: '' })
const supportTicketForm = reactive({ subject: '', content: '' })
const supportTicketReplyForm = reactive({ id: null, reply: '', status: 'replied' })
const reminderForm = reactive({ date: nextDateString() })
const checkupItemForm = reactive({ id: null, name: '', category: '', department: '', price: 0, durationMin: 10, description: '', status: 'active' })
const packageItemForm = reactive({ packageId: null, itemId: null, sortOrder: 0, required: true })
const scheduleForm = reactive({ id: null, doctorId: null, institutionId: null, date: '', period: '上午', category: '', startTime: '08:30', endTime: '', capacity: 1, status: 'available' })
const reportForm = reactive({
  appointmentId: null,
  summary: '',
  conclusion: '',
  recommendation: '',
})
const loading = reactive({
  login: false,
  register: false,
  logout: false,
  load: false,
  appointment: false,
  report: false,
  status: false,
  package: false,
  institution: false,
  doctorProfile: false,
  profile: false,
  emailCode: false,
  emailUpdate: false,
  authCode: false,
  familyMember: false,
  favorite: false,
  notification: false,
  adminNotification: false,
  reminder: false,
  coupon: false,
  review: false,
  announcement: false,
  checkupItem: false,
  packageItem: false,
  schedule: false,
  importPackages: false,
  exportPackages: false,
  importCheckupItems: false,
  exportCheckupItems: false,
  importPackageItems: false,
  exportPackageItems: false,
  importInstitutions: false,
  exportInstitutions: false,
  importScheduleSlots: false,
  exportScheduleSlots: false,
  importCoupons: false,
  exportCoupons: false,
  exportReviews: false,
  exportAnnouncements: false,
  exportAppointments: false,
  exportSupportTickets: false,
  exportUsers: false,
  exportMailLogs: false,
  exportLoginLogs: false,
  exportOperationLogs: false,
  exportSystemSettings: false,
  permission: false,
  systemSetting: false,
})

let bootstrapped = false

export function statusText(status) {
  return { booked: '已预约', checked: '已体检', reported: '已出报告', canceled: '已取消', waiting: '候补中', promoted: '已递补', active: '启用', pending: '待审核', disabled: '停用', deleted: '已归档', available: '可预约', full: '已满', draft: '草稿', published: '已发布', hidden: '已隐藏', unread: '未读', read: '已读', open: '待处理', replied: '已回复', closed: '已关闭', none: '未申请', requested: '已申请', issued: '已开具' }[status] || status
}

export function statusType(status) {
  return { booked: 'warning', checked: 'primary', reported: 'success', canceled: 'info', waiting: 'warning', promoted: 'success', active: 'success', pending: 'warning', disabled: 'danger', deleted: 'info', available: 'success', full: 'danger', draft: 'info', published: 'success', hidden: 'warning', unread: 'warning', read: 'info', open: 'warning', replied: 'success', closed: 'info', none: 'info', requested: 'warning', issued: 'success' }[status] || 'info'
}

export function paymentStatusText(status) {
  return { paid: '已支付', unpaid: '未支付', refunded: '已退款' }[status] || status || '-'
}

export function moneyText(value) {
  const amount = Number(value)
  if (!Number.isFinite(amount)) return '-'
  return `￥${amount.toFixed(2)}`
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

export function formatDate(value) {
  if (!value) return '-'
  return new Date(value).toLocaleDateString('zh-CN')
}

function escapeHTML(value) {
  return String(value || '').replace(/[&<>"']/g, (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]))
}

function nextDateString() {
  const date = new Date()
  date.setDate(date.getDate() + 1)
  return date.toISOString().slice(0, 10)
}

function documentHTML(title, rows, footer) {
  const cells = rows
    .map(([label, value]) => `<div class="label">${escapeHTML(label)}</div><div class="value">${escapeHTML(value)}</div>`)
    .join('')
  return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>${escapeHTML(title)}</title><style>body{font-family:Arial,"Microsoft YaHei",sans-serif;margin:0;background:#f3f6fa;color:#1f2d3d}.doc{max-width:860px;margin:32px auto;background:#fff;border:1px solid #d8e2ec;padding:32px}.head{border-bottom:3px solid #1f78b4;padding-bottom:16px;margin-bottom:24px}.head h1{margin:0;font-size:28px}.head p{margin:8px 0 0;color:#6b7c8f}.grid{display:grid;grid-template-columns:160px 1fr;border-top:1px solid #e3ebf2;border-left:1px solid #e3ebf2}.label,.value{padding:13px 16px;border-right:1px solid #e3ebf2;border-bottom:1px solid #e3ebf2}.label{font-weight:700;background:#f8fafc}.value{white-space:pre-wrap}.footer{margin-top:24px;color:#6b7c8f}</style></head><body><main class="doc"><section class="head"><h1>${escapeHTML(title)}</h1><p>东软熙心健康体检管理系统</p></section><section class="grid">${cells}</section><p class="footer">${escapeHTML(footer)}</p></main></body></html>`
}

export function appointmentDocumentHTML(appointment) {
  return documentHTML('体检预约订单', [
    ['订单号', appointment.orderNo],
    ['客户', appointment.user?.name],
    ['体检人', appointment.familyMember?.name ? `${appointment.familyMember.name}（${appointment.familyMember.relation || '家庭成员'}）` : '本人'],
    ['预约类型', appointment.appointmentType],
    ['体检分类', appointment.category],
    ['体检机构', appointment.institution?.name],
    ['机构地址', appointment.institution?.address],
    ['套餐', appointment.package?.name],
    ['项目明细', appointment.package?.items],
    ['医生', `${appointment.doctor?.name || ''} ${appointment.doctor?.title || ''}`],
    ['日期', appointment.date],
    ['时间', `${appointment.startTime}-${appointment.endTime}`],
    ['订单原价', moneyText(appointmentOriginalAmount(appointment))],
    ['优惠券', appointment.coupon?.name || '-'],
    ['优惠金额', moneyText(appointmentDiscountAmount(appointment))],
    ['应付金额', moneyText(appointmentPayableAmount(appointment))],
    ['支付状态', paymentStatusText(appointment.paymentStatus)],
    ['发票抬头', appointment.invoiceTitle],
    ['纳税人识别号', appointment.invoiceTaxNo],
    ['发票状态', statusText(appointment.invoiceStatus || 'none')],
    ['备注', appointment.note],
    ['状态', statusText(appointment.status)],
  ], '请按预约时间携带有效证件到检。')
}

export function reportDocumentHTML(report) {
  return documentHTML('体检报告详情', [
    ['报告编号', report.reportNo],
    ['订单号', report.appointment?.orderNo],
    ['客户', report.user?.name],
    ['体检机构', report.appointment?.institution?.name],
    ['体检分类', report.appointment?.category],
    ['套餐', report.appointment?.package?.name],
    ['医生', `${report.doctor?.name || ''} ${report.doctor?.title || ''}`],
    ['检查摘要', report.summary],
    ['体检结论', report.conclusion],
    ['健康建议', report.recommendation],
    ['报告时间', formatDate(report.createdAt)],
  ], '本报告仅供健康管理参考，如有不适请及时就医。')
}

export function downloadHTML(filename, html) {
  const blob = new Blob([html], { type: 'text/html;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

function saveUser(user) {
  currentUser.value = user
  if (user) localStorage.setItem('currentUser', JSON.stringify(user))
  else localStorage.removeItem('currentUser')
  if (user) {
    Object.assign(profileForm, {
      name: user.name || '',
      gender: user.gender || '',
      age: user.age || 0,
      idCard: user.idCard || '',
      email: user.email || '',
      avatarUrl: user.avatarUrl || '',
      bio: user.bio || '',
      emailNotify: user.emailNotify !== false,
    })
    Object.assign(emailForm, { email: user.email || '', code: '' })
  }
}

function assertPasswordsMatch(form) {
  if (form.password !== form.confirmPassword) {
    throw new Error('两次输入的密码不一致')
  }
}

function toQuery(params) {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') query.set(key, value)
  }
  return query.toString()
}

async function requestPage(path, state, params = {}) {
  const query = toQuery({ page: state.page, pageSize: state.pageSize, ...params })
  const result = await request(`${path}?${query}`)
  state.total = Number(result.total || 0)
  state.page = Number(result.page || state.page)
  state.pageSize = Number(result.pageSize || state.pageSize)
  return result.items || []
}

export function useHealthData() {
  const isAuthenticated = computed(() => Boolean(getAuthToken() && currentUser.value))
  const role = computed(() => currentUser.value?.role || '')
  const isUser = computed(() => role.value === 'user')
  const isDoctor = computed(() => role.value === 'doctor')
  const isAdmin = computed(() => role.value === 'admin')
  const can = (permission) => permissionCodes.value.includes(permission)
  const myAppointments = computed(() => appointments.value.filter((item) => item.userId === currentUser.value?.id))
  const bookedCount = computed(() => appointments.value.filter((item) => item.status === 'booked').length)
  const reportedCount = computed(() => appointments.value.filter((item) => item.status === 'reported').length)
  const pendingDoctorCount = computed(() => appointments.value.filter((item) => item.status !== 'reported').length)
  const pendingDoctors = computed(() => users.value.filter((item) => item.role === 'doctor' && item.status === 'pending'))
  const peopleRows = computed(() => {
    if (isAdmin.value) return users.value
    const rows = new Map()
    for (const item of appointments.value) {
      if (item.user?.id) rows.set(`user-${item.user.id}`, { ...item.user, source: '预约客户' })
    }
    for (const report of reports.value) {
      if (report.doctor?.id) rows.set(`doctor-${report.doctor.id}`, { ...report.doctor, source: '报告医生' })
      if (report.user?.id) rows.set(`user-${report.user.id}`, { ...report.user, source: '报告客户' })
    }
    if (currentUser.value?.id) rows.set(`current-${currentUser.value.role}-${currentUser.value.id}`, { ...currentUser.value, source: '当前登录' })
    return Array.from(rows.values())
  })

  async function sendAuthEmailCode(email) {
    if (loading.authCode) return
    loading.authCode = true
    try {
      await request('/auth/email-code', {
        method: 'POST',
        body: JSON.stringify({ email }),
      })
      ElMessage.success('验证码已发送，请查看邮箱')
    } finally {
      loading.authCode = false
    }
  }

  async function login() {
    if (loading.login) return
    loading.login = true
    try {
      const result = await request('/auth/login', {
        method: 'POST',
        body: JSON.stringify(loginForm),
      })
      setAuthToken(result.accessToken)
      saveUser(result.user)
      await loadMyPermissions()
      await loadAll()
      ElMessage.success('登录成功')
      return result.user
    } finally {
      loading.login = false
    }
  }

  async function registerUser() {
    if (loading.register) return
    loading.register = true
    try {
      assertPasswordsMatch(userRegisterForm)
      await request('/auth/register/user', {
        method: 'POST',
        body: JSON.stringify({
          name: userRegisterForm.name,
          email: userRegisterForm.email,
          code: userRegisterForm.code,
          password: userRegisterForm.password,
          gender: userRegisterForm.gender,
          age: Number(userRegisterForm.age || 0),
          idCard: userRegisterForm.idCard,
        }),
      })
      ElMessage.success('用户注册成功，请登录')
    } finally {
      loading.register = false
    }
  }

  async function registerDoctor() {
    if (loading.register) return
    loading.register = true
    try {
      assertPasswordsMatch(doctorRegisterForm)
      await request('/auth/register/doctor', {
        method: 'POST',
        body: JSON.stringify({
          name: doctorRegisterForm.name,
          email: doctorRegisterForm.email,
          code: doctorRegisterForm.code,
          password: doctorRegisterForm.password,
          employeeNo: doctorRegisterForm.employeeNo,
          department: doctorRegisterForm.department,
          title: doctorRegisterForm.title,
        }),
      })
      ElMessage.success('医生注册已提交，审核通过后可登录')
    } finally {
      loading.register = false
    }
  }

  async function logout() {
    if (loading.logout) return
    loading.logout = true
    try {
      if (getAuthToken()) await request('/auth/logout', { method: 'POST' }).catch(() => null)
      setAuthToken('')
      saveUser(null)
      appointments.value = []
      reports.value = []
      users.value = []
      slots.value = []
      waitlist.value = []
      familyMembers.value = []
      favorites.value = []
      browseHistories.value = []
      notifications.value = []
      rolePermissions.value = []
      permissionCodes.value = []
    } finally {
      loading.logout = false
    }
  }

  async function loadAll() {
    if (loading.load) return
    loading.load = true
    try {
      packages.value = await request('/packages')
      popularPackages.value = await request('/packages/popular')
      recommendedPackages.value = await request('/packages/recommended')
      activeCoupons.value = await request('/coupons/active')
      activeAnnouncements.value = await request('/announcements/active')
      supportInfo.value = await request('/support')
      institutions.value = await request('/institutions')
      if (!getAuthToken()) return
      appointments.value = await request('/appointments')
      reports.value = await request('/reports')
      slots.value = await request('/schedule/slots')
      if (isUser.value) {
        waitlist.value = await request('/waitlist')
        familyMembers.value = await request('/family-members')
        favorites.value = await request('/package-favorites')
        browseHistories.value = await request('/package-browses')
        supportTickets.value = await request('/support-tickets')
      }
      notifications.value = await request('/notifications')
      if (isDoctor.value || isAdmin.value) users.value = await request('/users')
      else users.value = currentUser.value ? [currentUser.value] : []
      if (isAdmin.value) mailLogs.value = await request('/mail-logs')
      if (isAdmin.value) adminDashboard.value = await request('/admin/dashboard')
      if (isAdmin.value) {
        checkupItems.value = await request('/checkup-items')
        packageItems.value = await request('/package-items')
      }
      if (!appointmentForm.institutionId && institutions.value[0]) appointmentForm.institutionId = institutions.value[0].id
      if (!appointmentForm.packageId && packages.value[0]) appointmentForm.packageId = packages.value[0].id
      if (!reportForm.appointmentId && appointments.value[0]) reportForm.appointmentId = appointments.value[0].id
    } finally {
      loading.load = false
    }
  }

  async function loadAppointmentsPage(params = {}) {
    appointments.value = await requestPage('/appointments', paginations.appointments, params)
  }

  async function loadReportsPage(params = {}) {
    reports.value = await requestPage('/reports', paginations.reports, params)
  }

  async function loadUsersPage(params = {}, key = 'users') {
    users.value = await requestPage('/users', paginations[key] || paginations.users, params)
  }

  async function loadWaitlistPage(params = {}) {
    waitlist.value = await requestPage('/waitlist', paginations.waitlist, params)
  }

  async function loadMailLogsPage(params = {}) {
    mailLogs.value = await requestPage('/mail-logs', paginations.mailLogs, params)
  }

  async function loadLoginLogsPage(params = {}) {
    loginLogs.value = await requestPage('/login-logs', paginations.loginLogs, params)
  }

  async function loadOperationLogsPage(params = {}) {
    operationLogs.value = await requestPage('/operation-logs', paginations.operationLogs, params)
  }

  async function loadMyPermissions() {
    if (!getAuthToken()) return
    const result = await request('/permissions/me').catch(() => ({ permissions: [] }))
    permissionCodes.value = result.permissions || []
  }

  async function loadRolePermissions() {
    if (!isAdmin.value) return
    rolePermissions.value = await request('/role-permissions')
  }

  async function loadRolePermissionsPage(params = {}) {
    if (!isAdmin.value) return
    rolePermissionRows.value = await requestPage('/role-permissions', paginations.rolePermissions, params)
  }

  async function loadSystemSettings() {
    if (!isAdmin.value) return
    systemSettings.value = await request('/system-settings')
  }

  async function loadSystemSettingsPage(params = {}) {
    if (!isAdmin.value) return
    systemSettingRows.value = await requestPage('/system-settings', paginations.systemSettings, params)
  }

  async function loadSupportInfo() {
    supportInfo.value = await request('/support')
  }

  async function loadPackagesPage(params = {}) {
    packages.value = await requestPage('/packages', paginations.packages, params)
  }

  async function loadInstitutions() {
    institutions.value = await request('/institutions')
  }

  async function loadInstitutionsPage(params = {}) {
    institutionRows.value = await requestPage('/institutions', paginations.institutions, params)
  }

  async function loadNotificationsPage(params = {}) {
    notifications.value = await requestPage('/notifications', paginations.notifications, params)
  }

  async function loadSupportTicketsPage(params = {}) {
    supportTickets.value = await requestPage('/support-tickets', paginations.supportTickets, params)
  }

  async function loadAdminNotificationsPage(params = {}) {
    if (!isAdmin.value) return
    adminNotifications.value = await requestPage('/admin/notifications', paginations.adminNotifications, params)
  }

  async function loadAdminSupportTicketsPage(params = {}) {
    if (!isAdmin.value) return
    adminSupportTickets.value = await requestPage('/admin/support-tickets', paginations.adminSupportTickets, params)
  }

  async function loadCouponsPage(params = {}) {
    coupons.value = await requestPage('/coupons', paginations.coupons, params)
  }

  async function loadReviewsPage(params = {}) {
    reviews.value = await requestPage('/reviews', paginations.reviews, params)
  }

  async function loadAnnouncementsPage(params = {}) {
    announcements.value = await requestPage('/announcements', paginations.announcements, params)
  }

  async function loadAdminDashboard(params = {}) {
    if (!isAdmin.value) return
    const query = toQuery(params)
    adminDashboard.value = await request(`/admin/dashboard${query ? `?${query}` : ''}`)
  }

  async function loadCheckupItemsPage(params = {}) {
    checkupItemRows.value = await requestPage('/checkup-items', paginations.checkupItems, params)
  }

  async function loadPackageItemsPage(params = {}) {
    packageItems.value = await requestPage('/package-items', paginations.packageItems, params)
  }

  async function loadSlotsPage(params = {}) {
    scheduleSlotRows.value = await requestPage('/schedule/slots', paginations.slots, params)
  }

  async function ensureBootstrapped() {
    if (bootstrapped) return
    bootstrapped = true
    if (getAuthToken()) {
      const user = await request('/auth/me').catch(() => null)
      if (user) {
        saveUser(user)
        await loadMyPermissions()
      }
      else {
        setAuthToken('')
        saveUser(null)
      }
    }
    await loadAll()
  }

  async function createAppointment() {
    if (!currentUser.value || loading.appointment) return
    loading.appointment = true
    try {
      const result = await request('/appointments', {
        method: 'POST',
        body: JSON.stringify(appointmentForm),
      })
      if (result.type === 'waitlist') ElMessage.warning('当前号源已满，已自动加入候补')
      else ElMessage.success('预约成功，医生和时间已分配')
      await loadAll()
    } finally {
      loading.appointment = false
    }
  }

  function editFamilyMember(member) {
    Object.assign(familyMemberForm, {
      id: member?.id || null,
      name: member?.name || '',
      relation: member?.relation || '',
      gender: member?.gender || '',
      age: member?.age || null,
      idCard: member?.idCard || '',
      phone: member?.phone || '',
    })
  }

  async function saveFamilyMember() {
    if (loading.familyMember) return
    loading.familyMember = true
    try {
      const body = JSON.stringify({
        name: familyMemberForm.name,
        relation: familyMemberForm.relation,
        gender: familyMemberForm.gender,
        age: Number(familyMemberForm.age || 0),
        idCard: familyMemberForm.idCard,
        phone: familyMemberForm.phone,
      })
      if (familyMemberForm.id) await request(`/family-members/${familyMemberForm.id}`, { method: 'PATCH', body })
      else await request('/family-members', { method: 'POST', body })
      ElMessage.success('家庭成员已保存')
      editFamilyMember(null)
      await loadAll()
    } finally {
      loading.familyMember = false
    }
  }

  async function deleteFamilyMember(member) {
    if (loading.familyMember) return
    loading.familyMember = true
    try {
      await request(`/family-members/${member.id}`, { method: 'DELETE' })
      ElMessage.success('家庭成员已删除')
      await loadAll()
    } finally {
      loading.familyMember = false
    }
  }

  async function toggleFavorite(pkg) {
    if (loading.favorite) return
    loading.favorite = true
    try {
      const exists = favorites.value.some((item) => item.packageId === pkg.id)
      await request(`/package-favorites/${pkg.id}`, { method: exists ? 'DELETE' : 'POST' })
      ElMessage.success(exists ? '已取消收藏' : '已收藏套餐')
      await loadAll()
    } finally {
      loading.favorite = false
    }
  }

  async function recordPackageBrowse(pkg) {
    if (!getAuthToken() || !isUser.value || !pkg?.id) return
    await request(`/packages/${pkg.id}/browse`, { method: 'POST' }).catch(() => null)
    browseHistories.value = await request('/package-browses').catch(() => browseHistories.value)
  }

  function editReschedule(appointment) {
    Object.assign(rescheduleForm, {
      appointmentId: appointment?.id || null,
      institutionId: appointment?.institutionId || null,
      slotId: null,
      date: appointment?.date || '',
      period: appointment?.period || '',
      note: appointment?.note || '',
    })
  }

  async function rescheduleAppointment() {
    if (!rescheduleForm.appointmentId || loading.appointment) return
    loading.appointment = true
    try {
      await request(`/appointments/${rescheduleForm.appointmentId}/reschedule`, {
        method: 'PATCH',
        body: JSON.stringify({
          appointmentType: '个人体检',
          institutionId: rescheduleForm.institutionId,
          slotId: rescheduleForm.slotId,
          date: rescheduleForm.date,
          period: rescheduleForm.period,
          note: rescheduleForm.note,
        }),
      })
      ElMessage.success('预约已改期')
      await loadAll()
    } finally {
      loading.appointment = false
    }
  }

  async function updateAppointmentPayment(appointment, paymentStatus) {
    if (!appointment?.id || loading.appointment) return
    loading.appointment = true
    try {
      await request(`/appointments/${appointment.id}/payment`, {
        method: 'PATCH',
        body: JSON.stringify({ paymentStatus }),
      })
      ElMessage.success(paymentStatus === 'paid' ? '已模拟支付成功' : '已撤销模拟支付')
      await loadAppointmentsPage()
      notifications.value = await request('/notifications').catch(() => notifications.value)
    } finally {
      loading.appointment = false
    }
  }

  function editInvoice(appointment) {
    Object.assign(invoiceForm, {
      appointmentId: appointment?.id || null,
      invoiceTitle: appointment?.invoiceTitle || '',
      invoiceTaxNo: appointment?.invoiceTaxNo || '',
    })
  }

  async function saveInvoice() {
    if (!invoiceForm.appointmentId || loading.appointment) return
    loading.appointment = true
    try {
      await request(`/appointments/${invoiceForm.appointmentId}/invoice`, {
        method: 'PATCH',
        body: JSON.stringify({
          invoiceTitle: invoiceForm.invoiceTitle,
          invoiceTaxNo: invoiceForm.invoiceTaxNo,
        }),
      })
      ElMessage.success('发票信息已保存')
      editInvoice(null)
      await loadAppointmentsPage()
    } finally {
      loading.appointment = false
    }
  }

  async function updateAppointmentInvoiceStatus(appointment, invoiceStatus) {
    if (!appointment?.id || loading.appointment) return
    loading.appointment = true
    try {
      await request(`/appointments/${appointment.id}/invoice/status`, {
        method: 'PATCH',
        body: JSON.stringify({ invoiceStatus }),
      })
      ElMessage.success(invoiceStatus === 'issued' ? '发票已标记开具' : '发票状态已更新')
      await loadAppointmentsPage()
    } finally {
      loading.appointment = false
    }
  }

  async function markNotificationRead(notification) {
    return updateMyNotificationStatus(notification, 'read')
  }

  async function updateMyNotificationStatus(notification, status) {
    if (loading.notification) return
    loading.notification = true
    try {
      await request(`/notifications/${notification.id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      })
      notification.status = status
      notification.readAt = status === 'read' ? new Date().toISOString() : null
    } finally {
      loading.notification = false
    }
  }

  function editCoupon(coupon) {
    Object.assign(couponForm, {
      id: coupon?.id || null,
      name: coupon?.name || '',
      code: coupon?.code || '',
      type: coupon?.type || 'amount',
      value: Number(coupon?.value || 0),
      minAmount: Number(coupon?.minAmount || 0),
      packageId: coupon?.packageId || null,
      status: coupon?.status || 'active',
      startDate: coupon?.startDate || '',
      endDate: coupon?.endDate || '',
      description: coupon?.description || '',
    })
  }

  async function saveCoupon() {
    if (loading.coupon) return
    loading.coupon = true
    try {
      const body = JSON.stringify({
        name: couponForm.name,
        code: couponForm.code,
        type: couponForm.type,
        value: Number(couponForm.value || 0),
        minAmount: Number(couponForm.minAmount || 0),
        packageId: Number(couponForm.packageId || 0),
        status: couponForm.status,
        startDate: couponForm.startDate,
        endDate: couponForm.endDate,
        description: couponForm.description,
      })
      if (couponForm.id) await request(`/coupons/${couponForm.id}`, { method: 'PATCH', body })
      else await request('/coupons', { method: 'POST', body })
      ElMessage.success('优惠券已保存')
      editCoupon(null)
      await loadCouponsPage()
      activeCoupons.value = await request('/coupons/active')
    } finally {
      loading.coupon = false
    }
  }

  async function archiveCoupon(coupon) {
    if (loading.coupon) return
    loading.coupon = true
    try {
      await request(`/coupons/${coupon.id}`, { method: 'DELETE' })
      ElMessage.success('优惠券已归档')
      await loadCouponsPage()
    } finally {
      loading.coupon = false
    }
  }

  async function createReview() {
    if (loading.review) return
    loading.review = true
    try {
      await request('/reviews', {
        method: 'POST',
        body: JSON.stringify({
          appointmentId: reviewForm.appointmentId,
          rating: Number(reviewForm.rating || 5),
          content: reviewForm.content,
        }),
      })
      ElMessage.success('评价已提交')
      Object.assign(reviewForm, { appointmentId: null, rating: 5, content: '' })
      await loadReviewsPage()
    } finally {
      loading.review = false
    }
  }

  function editReviewReply(review) {
    Object.assign(reviewReplyForm, { id: review?.id || null, reply: review?.reply || '', status: review?.status || 'published' })
  }

  async function saveReviewReply(params = {}) {
    if (!reviewReplyForm.id || loading.review) return
    loading.review = true
    try {
      await request(`/reviews/${reviewReplyForm.id}/reply`, {
        method: 'PATCH',
        body: JSON.stringify({ reply: reviewReplyForm.reply, status: reviewReplyForm.status }),
      })
      ElMessage.success('评价处理已保存')
      editReviewReply(null)
      await loadReviewsPage(params)
    } finally {
      loading.review = false
    }
  }

  function editAnnouncement(announcement) {
    Object.assign(announcementForm, {
      id: announcement?.id || null,
      title: announcement?.title || '',
      content: announcement?.content || '',
      audience: announcement?.audience || 'all',
      status: announcement?.status || 'draft',
    })
  }

  async function saveAnnouncement(params = {}) {
    if (loading.announcement) return
    loading.announcement = true
    try {
      const body = JSON.stringify({
        title: announcementForm.title,
        content: announcementForm.content,
        audience: announcementForm.audience,
        status: announcementForm.status,
      })
      if (announcementForm.id) await request(`/announcements/${announcementForm.id}`, { method: 'PATCH', body })
      else await request('/announcements', { method: 'POST', body })
      ElMessage.success('公告已保存')
      editAnnouncement(null)
      await loadAnnouncementsPage(params)
      activeAnnouncements.value = await request('/announcements/active')
    } finally {
      loading.announcement = false
    }
  }

  async function archiveAnnouncement(announcement, params = {}) {
    if (loading.announcement) return
    loading.announcement = true
    try {
      await request(`/announcements/${announcement.id}`, { method: 'DELETE' })
      ElMessage.success('公告已归档')
      await loadAnnouncementsPage(params)
      activeAnnouncements.value = await request('/announcements/active')
    } finally {
      loading.announcement = false
    }
  }

  function resetNotificationForm() {
    Object.assign(notificationForm, { userId: null, role: 'user', channel: 'in_app', type: 'admin_notice', title: '', content: '' })
  }

  async function sendAdminNotification() {
    if (loading.adminNotification) return
    loading.adminNotification = true
    try {
      const body = JSON.stringify({
        userId: Number(notificationForm.userId || 0),
        role: notificationForm.userId ? '' : notificationForm.role,
        channel: notificationForm.channel,
        type: notificationForm.type,
        title: notificationForm.title,
        content: notificationForm.content,
      })
      const result = await request('/admin/notifications', { method: 'POST', body })
      ElMessage.success(`通知已发送给 ${result.created || 0} 人`)
      resetNotificationForm()
      await loadAdminNotificationsPage()
    } finally {
      loading.adminNotification = false
    }
  }

  async function sendCheckupReminders() {
    if (loading.reminder) return
    loading.reminder = true
    try {
      const result = await request('/admin/notifications/reminders', {
        method: 'POST',
        body: JSON.stringify({ date: reminderForm.date }),
      })
      ElMessage.success(`已生成 ${result.created || 0} 条体检前提醒`)
      await loadAdminNotificationsPage()
    } finally {
      loading.reminder = false
    }
  }

  async function archiveAdminNotification(notification) {
    return updateAdminNotificationStatus(notification, 'archived')
  }

  async function updateAdminNotificationStatus(notification, status) {
    if (loading.adminNotification) return
    loading.adminNotification = true
    try {
      if (status === 'archived') await request(`/admin/notifications/${notification.id}`, { method: 'DELETE' })
      else {
        await request(`/admin/notifications/${notification.id}/status`, {
          method: 'PATCH',
          body: JSON.stringify({ status }),
        })
      }
      ElMessage.success(status === 'archived' ? '通知已归档' : '通知状态已更新')
      await loadAdminNotificationsPage()
    } finally {
      loading.adminNotification = false
    }
  }

  async function createSupportTicket() {
    if (loading.notification) return
    loading.notification = true
    try {
      await request('/support-tickets', {
        method: 'POST',
        body: JSON.stringify({
          subject: supportTicketForm.subject,
          content: supportTicketForm.content,
        }),
      })
      supportTicketForm.subject = ''
      supportTicketForm.content = ''
      ElMessage.success('咨询已提交')
      await loadSupportTicketsPage()
    } finally {
      loading.notification = false
    }
  }

  function editSupportTicketReply(ticket) {
    Object.assign(supportTicketReplyForm, {
      id: ticket?.id || null,
      reply: ticket?.reply || '',
      status: ticket?.status === 'closed' ? 'closed' : 'replied',
    })
  }

  async function saveSupportTicketReply() {
    if (!supportTicketReplyForm.id || loading.adminNotification) return
    loading.adminNotification = true
    try {
      await request(`/admin/support-tickets/${supportTicketReplyForm.id}/reply`, {
        method: 'PATCH',
        body: JSON.stringify({
          reply: supportTicketReplyForm.reply,
          status: supportTicketReplyForm.status,
        }),
      })
      ElMessage.success('客服工单已处理')
      editSupportTicketReply(null)
      await loadAdminSupportTicketsPage()
    } finally {
      loading.adminNotification = false
    }
  }

  function editCheckupItem(item) {
    Object.assign(checkupItemForm, {
      id: item?.id || null,
      name: item?.name || '',
      category: item?.category || '',
      department: item?.department || '',
      price: Number(item?.price || 0),
      durationMin: Number(item?.durationMin || 10),
      description: item?.description || '',
      status: item?.status || 'active',
    })
  }

  async function saveCheckupItem() {
    if (loading.checkupItem) return
    loading.checkupItem = true
    try {
      const body = JSON.stringify({
        name: checkupItemForm.name,
        category: checkupItemForm.category,
        department: checkupItemForm.department,
        price: Number(checkupItemForm.price || 0),
        durationMin: Number(checkupItemForm.durationMin || 10),
        description: checkupItemForm.description,
        status: checkupItemForm.status,
      })
      if (checkupItemForm.id) await request(`/checkup-items/${checkupItemForm.id}`, { method: 'PATCH', body })
      else await request('/checkup-items', { method: 'POST', body })
      ElMessage.success('体检项目已保存')
      editCheckupItem(null)
      checkupItems.value = await request('/checkup-items')
      await loadCheckupItemsPage()
    } finally {
      loading.checkupItem = false
    }
  }

  async function archiveCheckupItem(item) {
    if (loading.checkupItem) return
    loading.checkupItem = true
    try {
      await request(`/checkup-items/${item.id}`, { method: 'DELETE' })
      ElMessage.success('体检项目已归档')
      checkupItems.value = await request('/checkup-items')
      await loadCheckupItemsPage()
    } finally {
      loading.checkupItem = false
    }
  }

  async function exportCheckupItems(params = {}) {
    if (loading.exportCheckupItems) return
    loading.exportCheckupItems = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/checkup-items/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'checkup-items.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('体检项目 CSV 已导出')
    } finally {
      loading.exportCheckupItems = false
    }
  }

  async function importCheckupItems(file) {
    if (!file || loading.importCheckupItems) return
    loading.importCheckupItems = true
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await request('/checkup-items/import', { method: 'POST', body: formData })
      ElMessage.success(`导入完成，新增 ${result.created || 0} 条，更新 ${result.updated || 0} 条`)
      checkupItems.value = await request('/checkup-items')
      await loadCheckupItemsPage()
    } finally {
      loading.importCheckupItems = false
    }
  }

  async function savePackageItem() {
    if (loading.packageItem) return
    loading.packageItem = true
    try {
      await request('/package-items', {
        method: 'POST',
        body: JSON.stringify({
          packageId: packageItemForm.packageId,
          itemId: packageItemForm.itemId,
          sortOrder: Number(packageItemForm.sortOrder || 0),
          required: packageItemForm.required,
        }),
      })
      ElMessage.success('套餐项目组合已保存')
      await loadPackageItemsPage({ packageId: packageItemForm.packageId })
    } finally {
      loading.packageItem = false
    }
  }

  async function exportPackageItems(params = {}) {
    if (loading.exportPackageItems) return
    loading.exportPackageItems = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/package-items/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'package-items.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('套餐项目组合 CSV 已导出')
    } finally {
      loading.exportPackageItems = false
    }
  }

  async function importPackageItems(file) {
    if (!file || loading.importPackageItems) return
    loading.importPackageItems = true
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await request('/package-items/import', { method: 'POST', body: formData })
      ElMessage.success(`导入完成，新增 ${result.created || 0} 条，更新 ${result.updated || 0} 条`)
      await loadPackageItemsPage(packageItemForm.packageId ? { packageId: packageItemForm.packageId } : {})
    } finally {
      loading.importPackageItems = false
    }
  }

  async function deletePackageItem(row) {
    if (loading.packageItem) return
    loading.packageItem = true
    try {
      await request(`/package-items/${row.id}`, { method: 'DELETE' })
      ElMessage.success('套餐项目已移除')
      await loadPackageItemsPage(packageItemForm.packageId ? { packageId: packageItemForm.packageId } : {})
    } finally {
      loading.packageItem = false
    }
  }

  function editScheduleSlot(slot) {
    Object.assign(scheduleForm, {
      id: slot?.id || null,
      doctorId: slot?.doctorId || null,
      institutionId: slot?.institutionId || null,
      date: slot?.date || '',
      period: slot?.period || '上午',
      category: slot?.category || '',
      startTime: slot?.startTime || '08:30',
      endTime: slot?.endTime || '',
      capacity: Number(slot?.capacity || 1),
      status: slot?.status || 'available',
    })
  }

  async function saveScheduleSlot() {
    if (loading.schedule) return
    loading.schedule = true
    try {
      const body = JSON.stringify({
        doctorId: scheduleForm.doctorId,
        institutionId: scheduleForm.institutionId,
        date: scheduleForm.date,
        period: scheduleForm.period,
        category: scheduleForm.category,
        startTime: scheduleForm.startTime,
        endTime: scheduleForm.endTime,
        capacity: Number(scheduleForm.capacity || 1),
        status: scheduleForm.status,
      })
      if (scheduleForm.id) await request(`/schedule/slots/${scheduleForm.id}`, { method: 'PATCH', body })
      else await request('/schedule/slots', { method: 'POST', body })
      ElMessage.success('排班号源已保存')
      editScheduleSlot(null)
      slots.value = await request('/schedule/slots')
      await loadSlotsPage()
    } finally {
      loading.schedule = false
    }
  }

  async function archiveScheduleSlot(slot) {
    if (loading.schedule) return
    loading.schedule = true
    try {
      await request(`/schedule/slots/${slot.id}`, { method: 'DELETE' })
      ElMessage.success('排班号源已归档')
      slots.value = await request('/schedule/slots')
      await loadSlotsPage()
    } finally {
      loading.schedule = false
    }
  }

  function editInstitution(institution) {
    Object.assign(institutionForm, {
      id: institution?.id || null,
      name: institution?.name || '',
      address: institution?.address || '',
      phone: institution?.phone || '',
      openHours: institution?.openHours || '',
      status: institution?.status || 'active',
    })
  }

  async function saveInstitution() {
    if (loading.institution) return
    loading.institution = true
    try {
      const body = JSON.stringify({
        name: institutionForm.name,
        address: institutionForm.address,
        phone: institutionForm.phone,
        openHours: institutionForm.openHours,
        status: institutionForm.status,
      })
      if (institutionForm.id) await request(`/institutions/${institutionForm.id}`, { method: 'PATCH', body })
      else await request('/institutions', { method: 'POST', body })
      ElMessage.success('体检机构已保存')
      editInstitution(null)
      await loadInstitutions()
      await loadInstitutionsPage()
    } finally {
      loading.institution = false
    }
  }

  async function archiveInstitution(institution) {
    if (loading.institution) return
    loading.institution = true
    try {
      await request(`/institutions/${institution.id}`, { method: 'DELETE' })
      ElMessage.success('体检机构已归档')
      await loadInstitutions()
      await loadInstitutionsPage()
    } finally {
      loading.institution = false
    }
  }

  async function exportInstitutions(params = {}) {
    if (loading.exportInstitutions) return
    loading.exportInstitutions = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/institutions/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'institutions.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('体检机构 CSV 已导出')
    } finally {
      loading.exportInstitutions = false
    }
  }

  async function importInstitutions(file) {
    if (!file || loading.importInstitutions) return
    loading.importInstitutions = true
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await request('/institutions/import', { method: 'POST', body: formData })
      ElMessage.success(`导入完成，新增 ${result.created || 0} 条，更新 ${result.updated || 0} 条`)
      await loadInstitutions()
      await loadInstitutionsPage()
    } finally {
      loading.importInstitutions = false
    }
  }

  async function exportScheduleSlots(params = {}) {
    if (loading.exportScheduleSlots) return
    loading.exportScheduleSlots = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/schedule/slots/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'schedule-slots.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('号源 CSV 已导出')
    } finally {
      loading.exportScheduleSlots = false
    }
  }

  async function importScheduleSlots(file) {
    if (!file || loading.importScheduleSlots) return
    loading.importScheduleSlots = true
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await request('/schedule/slots/import', { method: 'POST', body: formData })
      ElMessage.success(`导入完成，新增 ${result.created || 0} 条，更新 ${result.updated || 0} 条`)
      slots.value = await request('/schedule/slots')
      await loadSlotsPage()
    } finally {
      loading.importScheduleSlots = false
    }
  }

  async function joinWaitlist(slot) {
    if (!currentUser.value || loading.appointment) return
    loading.appointment = true
    try {
      Object.assign(waitlistForm, {
        appointmentType: appointmentForm.appointmentType,
        institutionId: appointmentForm.institutionId,
        packageId: appointmentForm.packageId,
        date: appointmentForm.date,
        period: slot?.period || appointmentForm.period,
        note: appointmentForm.note,
      })
      const result = await request('/appointments', {
        method: 'POST',
        body: JSON.stringify({ ...waitlistForm, slotId: slot?.id || 0 }),
      })
      if (result.type === 'waitlist') ElMessage.warning('已加入候补，系统会在有号源释放时自动递补')
      else ElMessage.success('预约成功，医生和时间已分配')
      await loadAll()
    } finally {
      loading.appointment = false
    }
  }

  async function saveProfile() {
    if (loading.profile) return
    loading.profile = true
    try {
      const payload = {
        name: profileForm.name,
        gender: profileForm.gender,
        age: Number(profileForm.age || 0),
        idCard: profileForm.idCard,
        avatarUrl: profileForm.avatarUrl,
        bio: profileForm.bio,
        emailNotify: profileForm.emailNotify,
      }
      const user = await request('/profile', {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
      saveUser(user)
      ElMessage.success('个人资料已保存')
      await loadAll()
    } finally {
      loading.profile = false
    }
  }

  async function sendEmailCode() {
    if (loading.emailCode) return
    loading.emailCode = true
    try {
      await request('/profile/email-code', {
        method: 'POST',
        body: JSON.stringify({ email: emailForm.email }),
      })
      emailForm.code = ''
      ElMessage.success('验证码已发送，请查看目标邮箱')
    } finally {
      loading.emailCode = false
    }
  }

  async function updateEmail() {
    if (loading.emailUpdate) return
    loading.emailUpdate = true
    try {
      const user = await request('/profile/email', {
        method: 'PATCH',
        body: JSON.stringify({ email: emailForm.email, code: emailForm.code }),
      })
      saveUser(user)
      ElMessage.success('邮箱已验证并更新')
      await loadAll()
    } finally {
      loading.emailUpdate = false
    }
  }

  async function cancelAppointment(row) {
    if (loading.status) return
    loading.status = true
    try {
      const result = await request(`/appointments/${row.id}/cancel`, { method: 'PATCH' })
      ElMessage.success(result.paymentStatus === 'refunded' ? '预约已取消，支付状态已模拟退款' : '预约已取消')
      await loadAll()
    } finally {
      loading.status = false
    }
  }

  async function cancelWaitlist(row) {
    if (!row?.id || loading.status) return
    loading.status = true
    try {
      await request(`/waitlist/${row.id}/cancel`, { method: 'PATCH' })
      ElMessage.success('候补已取消')
      await loadWaitlistPage()
    } finally {
      loading.status = false
    }
  }

  async function markDone(row) {
    if (loading.status) return
    loading.status = true
    try {
      await request(`/appointments/${row.id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status: 'checked' }),
      })
      reportForm.appointmentId = row.id
      ElMessage.success('已标记完成体检，可继续生成报告')
      await loadAll()
    } finally {
      loading.status = false
    }
  }

  async function createReport() {
    if (!currentUser.value || loading.report) return
    loading.report = true
    try {
      await request('/reports', {
        method: 'POST',
        body: JSON.stringify(reportForm),
      })
      ElMessage.success('报告已生成')
      await loadAll()
    } finally {
      loading.report = false
    }
  }

  function editPackage(pkg) {
    Object.assign(packageForm, {
      id: pkg?.id || null,
      name: pkg?.name || '',
      category: pkg?.category || '年度综合',
      description: pkg?.description || '',
      price: Number(pkg?.price || 0),
      items: pkg?.items || '',
      status: pkg?.status || 'active',
    })
  }

  async function savePackage() {
    if (loading.package) return
    loading.package = true
    try {
      const body = JSON.stringify({
        name: packageForm.name,
        category: packageForm.category,
        description: packageForm.description,
        price: Number(packageForm.price || 0),
        items: packageForm.items,
        status: packageForm.status,
      })
      if (packageForm.id) await request(`/packages/${packageForm.id}`, { method: 'PATCH', body })
      else await request('/packages', { method: 'POST', body })
      ElMessage.success('套餐已保存')
      editPackage(null)
      await loadAll()
    } finally {
      loading.package = false
    }
  }

  async function archivePackage(pkg) {
    if (loading.package) return
    loading.package = true
    try {
      await request(`/packages/${pkg.id}`, { method: 'DELETE' })
      ElMessage.success('套餐已归档')
      await loadPackagesPage()
    } finally {
      loading.package = false
    }
  }

  async function exportPackages() {
    if (loading.exportPackages) return
    loading.exportPackages = true
    try {
      const blob = await requestBlob('/packages/export')
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'packages.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('套餐 CSV 已导出')
    } finally {
      loading.exportPackages = false
    }
  }

  async function exportAppointments(params = {}) {
    if (loading.exportAppointments) return
    loading.exportAppointments = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/appointments/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'appointments.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('预约 CSV 已导出')
    } finally {
      loading.exportAppointments = false
    }
  }

  async function exportCoupons(params = {}) {
    if (loading.exportCoupons) return
    loading.exportCoupons = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/coupons/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'coupons.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('优惠券 CSV 已导出')
    } finally {
      loading.exportCoupons = false
    }
  }

  async function exportReviews(params = {}) {
    if (loading.exportReviews) return
    loading.exportReviews = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/reviews/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'reviews.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('评价 CSV 已导出')
    } finally {
      loading.exportReviews = false
    }
  }

  async function exportAnnouncements(params = {}) {
    if (loading.exportAnnouncements) return
    loading.exportAnnouncements = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/announcements/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'announcements.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('系统公告 CSV 已导出')
    } finally {
      loading.exportAnnouncements = false
    }
  }

  async function exportSupportTickets(params = {}) {
    if (loading.exportSupportTickets) return
    loading.exportSupportTickets = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/admin/support-tickets/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'support-tickets.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('客服工单 CSV 已导出')
    } finally {
      loading.exportSupportTickets = false
    }
  }

  async function exportUsers(params = {}) {
    if (loading.exportUsers) return
    loading.exportUsers = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/users/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'users.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('人员档案 CSV 已导出')
    } finally {
      loading.exportUsers = false
    }
  }

  async function exportMailLogs(params = {}) {
    if (loading.exportMailLogs) return
    loading.exportMailLogs = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/mail-logs/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'mail-logs.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('邮件日志 CSV 已导出')
    } finally {
      loading.exportMailLogs = false
    }
  }

  async function exportLoginLogs(params = {}) {
    if (loading.exportLoginLogs) return
    loading.exportLoginLogs = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/login-logs/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'login-logs.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('登录日志 CSV 已导出')
    } finally {
      loading.exportLoginLogs = false
    }
  }

  async function exportOperationLogs(params = {}) {
    if (loading.exportOperationLogs) return
    loading.exportOperationLogs = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/operation-logs/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'operation-logs.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('操作日志 CSV 已导出')
    } finally {
      loading.exportOperationLogs = false
    }
  }

  async function exportSystemSettings(params = {}) {
    if (loading.exportSystemSettings) return
    loading.exportSystemSettings = true
    try {
      const query = toQuery(params)
      const blob = await requestBlob(`/system-settings/export${query ? `?${query}` : ''}`)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'system-settings.csv'
      link.click()
      URL.revokeObjectURL(url)
      ElMessage.success('系统设置 CSV 已导出')
    } finally {
      loading.exportSystemSettings = false
    }
  }

  async function importPackages(file) {
    if (!file || loading.importPackages) return
    loading.importPackages = true
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await request('/packages/import', { method: 'POST', body: formData })
      ElMessage.success(`导入完成，新增 ${result.created || 0} 条，更新 ${result.updated || 0} 条`)
      await loadPackagesPage()
    } finally {
      loading.importPackages = false
    }
  }

  async function importCoupons(file) {
    if (!file || loading.importCoupons) return
    loading.importCoupons = true
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await request('/coupons/import', { method: 'POST', body: formData })
      ElMessage.success(`导入完成，新增 ${result.created || 0} 条，更新 ${result.updated || 0} 条`)
      await loadCouponsPage()
      activeCoupons.value = await request('/coupons/active')
    } finally {
      loading.importCoupons = false
    }
  }

  async function updateRolePermission(permission) {
    if (loading.permission) return
    loading.permission = true
    try {
      const updated = await request(`/role-permissions/${permission.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ enabled: permission.enabled }),
      })
      const index = rolePermissions.value.findIndex((item) => item.id === updated.id)
      if (index >= 0) rolePermissions.value[index] = updated
      const rowIndex = rolePermissionRows.value.findIndex((item) => item.id === updated.id)
      if (rowIndex >= 0) rolePermissionRows.value[rowIndex] = updated
      if (updated.role === currentUser.value?.role) await loadMyPermissions()
      ElMessage.success('权限配置已更新')
    } finally {
      loading.permission = false
    }
  }

  async function updateSystemSetting(setting) {
    if (loading.systemSetting) return
    loading.systemSetting = true
    try {
      const updated = await request(`/system-settings/${setting.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          value: String(setting.value ?? ''),
          valueType: setting.valueType,
          label: setting.label,
          description: setting.description,
          status: setting.status,
        }),
      })
      const index = systemSettings.value.findIndex((item) => item.id === updated.id)
      if (index >= 0) systemSettings.value[index] = updated
      const rowIndex = systemSettingRows.value.findIndex((item) => item.id === updated.id)
      if (rowIndex >= 0) systemSettingRows.value[rowIndex] = updated
      if (updated.key === 'service.faq' || updated.key === 'service.customer_service_url' || updated.key === 'service.customer_service_hours') {
        await loadSupportInfo()
      }
      ElMessage.success('系统设置已保存')
    } finally {
      loading.systemSetting = false
    }
  }

  async function updateUserStatus(user, status) {
    if (loading.status) return
    loading.status = true
    try {
      await request(`/users/${user.id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      })
      ElMessage.success('状态已更新')
      await loadAll()
    } finally {
      loading.status = false
    }
  }

  async function updateDoctorProfile(user, payload) {
    if (loading.doctorProfile) return
    loading.doctorProfile = true
    try {
      const specialties = Array.isArray(payload.specialties) ? payload.specialties.join(',') : payload.specialties
      const updated = await request(`/users/${user.id}/doctor-profile`, {
        method: 'PATCH',
        body: JSON.stringify({
          department: payload.department,
          title: payload.title || user.title,
          specialties,
        }),
      })
      const index = users.value.findIndex((item) => item.id === updated.id)
      if (index >= 0) users.value[index] = updated
      ElMessage.success('医生资料已更新')
      await loadAll()
    } finally {
      loading.doctorProfile = false
    }
  }

  function selectPackage(pkg) {
    appointmentForm.packageId = pkg.id
  }

  return {
    loginForm,
    userRegisterForm,
    doctorRegisterForm,
    currentUser,
    packages,
    appointments,
    reports,
    users,
    institutions,
    institutionRows,
    slots,
    scheduleSlotRows,
    waitlist,
    mailLogs,
    loginLogs,
    operationLogs,
    rolePermissions,
    rolePermissionRows,
    permissionCodes,
    systemSettings,
    systemSettingRows,
    familyMembers,
    favorites,
    browseHistories,
    popularPackages,
    recommendedPackages,
    notifications,
    adminNotifications,
    supportTickets,
    adminSupportTickets,
    coupons,
    activeCoupons,
    reviews,
    announcements,
    activeAnnouncements,
    supportInfo,
    adminDashboard,
    checkupItems,
    checkupItemRows,
    packageItems,
    paginations,
    appointmentForm,
    waitlistForm,
    profileForm,
    emailForm,
    familyMemberForm,
    rescheduleForm,
    invoiceForm,
    packageForm,
    institutionForm,
    couponForm,
    reviewForm,
    reviewReplyForm,
    announcementForm,
    notificationForm,
    supportTicketForm,
    supportTicketReplyForm,
    reminderForm,
    checkupItemForm,
    packageItemForm,
    scheduleForm,
    reportForm,
    loading,
    role,
    isAuthenticated,
    isUser,
    isDoctor,
    isAdmin,
    can,
    myAppointments,
    bookedCount,
    reportedCount,
    pendingDoctorCount,
    pendingDoctors,
    peopleRows,
    login,
    sendAuthEmailCode,
    registerUser,
    registerDoctor,
    logout,
    loadAll,
    loadAppointmentsPage,
    loadReportsPage,
    loadUsersPage,
    loadWaitlistPage,
    loadMailLogsPage,
    loadLoginLogsPage,
    loadOperationLogsPage,
    loadMyPermissions,
    loadRolePermissions,
    loadRolePermissionsPage,
    loadSystemSettings,
    loadSystemSettingsPage,
    loadSupportInfo,
    loadPackagesPage,
    loadInstitutions,
    loadInstitutionsPage,
    loadNotificationsPage,
    loadSupportTicketsPage,
    loadAdminNotificationsPage,
    loadAdminSupportTicketsPage,
    loadCouponsPage,
    loadReviewsPage,
    loadAnnouncementsPage,
    loadAdminDashboard,
    loadCheckupItemsPage,
    loadPackageItemsPage,
    loadSlotsPage,
    ensureBootstrapped,
    createAppointment,
    joinWaitlist,
    cancelAppointment,
    cancelWaitlist,
    editFamilyMember,
    saveFamilyMember,
    deleteFamilyMember,
    toggleFavorite,
    recordPackageBrowse,
    editReschedule,
    rescheduleAppointment,
    updateAppointmentPayment,
    editInvoice,
    saveInvoice,
    updateAppointmentInvoiceStatus,
    markNotificationRead,
    updateMyNotificationStatus,
    editCoupon,
    saveCoupon,
    archiveCoupon,
    createReview,
    editReviewReply,
    saveReviewReply,
    exportReviews,
    editAnnouncement,
    saveAnnouncement,
    archiveAnnouncement,
    exportAnnouncements,
    resetNotificationForm,
    sendAdminNotification,
    sendCheckupReminders,
    archiveAdminNotification,
    updateAdminNotificationStatus,
    createSupportTicket,
    editSupportTicketReply,
    saveSupportTicketReply,
    editCheckupItem,
    saveCheckupItem,
    archiveCheckupItem,
    exportCheckupItems,
    importCheckupItems,
    savePackageItem,
    exportPackageItems,
    importPackageItems,
    deletePackageItem,
    editScheduleSlot,
    saveScheduleSlot,
    archiveScheduleSlot,
    exportScheduleSlots,
    importScheduleSlots,
    editInstitution,
    saveInstitution,
    archiveInstitution,
    exportInstitutions,
    importInstitutions,
    saveProfile,
    sendEmailCode,
    updateEmail,
    markDone,
    createReport,
    updateUserStatus,
    updateDoctorProfile,
    editPackage,
    savePackage,
    archivePackage,
    exportPackages,
    exportCoupons,
    exportAppointments,
    exportSupportTickets,
    exportUsers,
    exportMailLogs,
    exportLoginLogs,
    exportOperationLogs,
    exportSystemSettings,
    importPackages,
    importCoupons,
    updateRolePermission,
    updateSystemSetting,
    selectPackage,
  }
}

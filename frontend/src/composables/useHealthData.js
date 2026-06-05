import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { request, setAuthToken, getAuthToken } from '../api/client'

const loginForm = reactive({ phone: '13800000001', password: '123456' })
const userRegisterForm = reactive({ name: '', phone: '', password: '', confirmPassword: '', gender: '', age: null, idCard: '' })
const doctorRegisterForm = reactive({
  name: '',
  phone: '',
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
const slots = ref([])
const waitlist = ref([])
const mailLogs = ref([])
const appointmentForm = reactive({ packageId: null, date: '2026-06-05', period: '上午', note: '' })
const profileForm = reactive({ name: '', gender: '', age: 0, idCard: '', email: '', avatarUrl: '', bio: '', emailNotify: true })
const packageForm = reactive({ id: null, name: '', description: '', price: 0, items: '', status: 'active' })
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
  profile: false,
})

let bootstrapped = false

export function statusText(status) {
  return { booked: '已预约', checked: '已体检', reported: '已出报告', canceled: '已取消', waiting: '候补中', promoted: '已递补', active: '启用', pending: '待审核', disabled: '停用', available: '可预约' }[status] || status
}

export function statusType(status) {
  return { booked: 'warning', checked: 'primary', reported: 'success', canceled: 'info', waiting: 'warning', promoted: 'success', active: 'success', pending: 'warning', disabled: 'danger', available: 'success' }[status] || 'info'
}

export function formatDate(value) {
  if (!value) return '-'
  return new Date(value).toLocaleDateString('zh-CN')
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
  }
}

function assertPasswordsMatch(form) {
  if (form.password !== form.confirmPassword) {
    throw new Error('两次输入的密码不一致')
  }
}

export function useHealthData() {
  const isAuthenticated = computed(() => Boolean(getAuthToken() && currentUser.value))
  const role = computed(() => currentUser.value?.role || '')
  const isUser = computed(() => role.value === 'user')
  const isDoctor = computed(() => role.value === 'doctor')
  const isAdmin = computed(() => role.value === 'admin')
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
          phone: userRegisterForm.phone,
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
          phone: doctorRegisterForm.phone,
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
    } finally {
      loading.logout = false
    }
  }

  async function loadAll() {
    if (loading.load) return
    loading.load = true
    try {
      packages.value = await request('/packages')
      if (!getAuthToken()) return
      appointments.value = await request('/appointments')
      reports.value = await request('/reports')
      slots.value = await request(`/schedule/slots?date=${appointmentForm.date}&period=${appointmentForm.period}`)
      if (isUser.value) waitlist.value = await request('/waitlist')
      if (isDoctor.value || isAdmin.value) users.value = await request('/users')
      else users.value = currentUser.value ? [currentUser.value] : []
      if (isAdmin.value) mailLogs.value = await request('/mail-logs')
      if (!appointmentForm.packageId && packages.value[0]) appointmentForm.packageId = packages.value[0].id
      if (!reportForm.appointmentId && appointments.value[0]) reportForm.appointmentId = appointments.value[0].id
    } finally {
      loading.load = false
    }
  }

  async function ensureBootstrapped() {
    if (bootstrapped) return
    bootstrapped = true
    if (getAuthToken()) {
      const user = await request('/auth/me').catch(() => null)
      if (user) saveUser(user)
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
      await request('/appointments', {
        method: 'POST',
        body: JSON.stringify(appointmentForm),
      })
      ElMessage.success('预约请求已提交，请查看预约或候补状态')
      await loadAll()
    } finally {
      loading.appointment = false
    }
  }

  async function saveProfile() {
    if (loading.profile) return
    loading.profile = true
    try {
      const user = await request('/profile', {
        method: 'PATCH',
        body: JSON.stringify(profileForm),
      })
      saveUser(user)
      ElMessage.success('个人资料已保存')
      await loadAll()
    } finally {
      loading.profile = false
    }
  }

  async function cancelAppointment(row) {
    if (loading.status) return
    loading.status = true
    try {
      await request(`/appointments/${row.id}/cancel`, { method: 'PATCH' })
      ElMessage.success('预约已取消')
      await loadAll()
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
    slots,
    waitlist,
    mailLogs,
    appointmentForm,
    profileForm,
    packageForm,
    reportForm,
    loading,
    role,
    isAuthenticated,
    isUser,
    isDoctor,
    isAdmin,
    myAppointments,
    bookedCount,
    reportedCount,
    pendingDoctorCount,
    pendingDoctors,
    peopleRows,
    login,
    registerUser,
    registerDoctor,
    logout,
    loadAll,
    ensureBootstrapped,
    createAppointment,
    cancelAppointment,
    saveProfile,
    markDone,
    createReport,
    updateUserStatus,
    editPackage,
    savePackage,
    selectPackage,
  }
}

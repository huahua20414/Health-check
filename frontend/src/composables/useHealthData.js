import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { request } from '../api/client'

const role = ref('user')
const loginForm = reactive({ phone: '13800000001', name: '张三' })
const currentUser = ref(null)
const packages = ref([])
const appointments = ref([])
const reports = ref([])
const appointmentForm = reactive({ packageId: null, date: '2026-06-05', period: '上午', note: '' })
const reportForm = reactive({
  appointmentId: null,
  summary: '本次体检主要指标未见明显异常。',
  conclusion: '总体健康状况良好。',
  recommendation: '建议保持规律作息，按年度复查。',
})

let bootstrapped = false

export function statusText(status) {
  return { booked: '已预约', checked: '已体检', reported: '已出报告' }[status] || status
}

export function statusType(status) {
  return { booked: 'warning', checked: 'primary', reported: 'success' }[status] || 'info'
}

export function formatDate(value) {
  if (!value) return '-'
  return new Date(value).toLocaleDateString('zh-CN')
}

export function useHealthData() {
  const isUser = computed(() => currentUser.value?.role === 'user')
  const isDoctor = computed(() => currentUser.value?.role === 'doctor')
  const myAppointments = computed(() => appointments.value.filter((item) => item.userId === currentUser.value?.id))
  const bookedCount = computed(() => appointments.value.filter((item) => item.status === 'booked').length)
  const reportedCount = computed(() => appointments.value.filter((item) => item.status === 'reported').length)
  const pendingDoctorCount = computed(() => appointments.value.filter((item) => item.status !== 'reported').length)
  const peopleRows = computed(() => {
    const rows = new Map()
    for (const item of appointments.value) {
      if (item.user?.id) rows.set(`user-${item.user.id}`, { ...item.user, source: '预约客户' })
    }
    for (const report of reports.value) {
      if (report.doctor?.id) rows.set(`doctor-${report.doctor.id}`, { ...report.doctor, source: '报告医生' })
      if (report.user?.id) rows.set(`user-${report.user.id}`, { ...report.user, source: '报告客户' })
    }
    if (currentUser.value?.id) {
      rows.set(`current-${currentUser.value.role}-${currentUser.value.id}`, { ...currentUser.value, source: '当前登录' })
    }
    return Array.from(rows.values())
  })

  async function login() {
    currentUser.value = await request('/login', {
      method: 'POST',
      body: JSON.stringify({ ...loginForm, role: role.value }),
    })
    await loadAll()
    ElMessage.success('登录成功')
  }

  async function quickLogin(nextRole = role.value) {
    role.value = nextRole
    if (role.value === 'doctor') {
      loginForm.phone = '13900000001'
      loginForm.name = '李医生'
    } else {
      loginForm.phone = '13800000001'
      loginForm.name = '张三'
    }
    await login()
  }

  async function loadAll() {
    packages.value = await request('/packages')
    appointments.value = await request('/appointments')
    const userId = isUser.value && currentUser.value ? `?userId=${currentUser.value.id}` : ''
    reports.value = currentUser.value ? await request(`/reports${userId}`) : []
    if (!appointmentForm.packageId && packages.value[0]) appointmentForm.packageId = packages.value[0].id
    if (!reportForm.appointmentId && appointments.value[0]) reportForm.appointmentId = appointments.value[0].id
  }

  async function ensureBootstrapped() {
    if (bootstrapped) return
    bootstrapped = true
    await login()
  }

  async function createAppointment() {
    if (!currentUser.value) return
    await request('/appointments', {
      method: 'POST',
      body: JSON.stringify({ ...appointmentForm, userId: currentUser.value.id }),
    })
    ElMessage.success('预约已提交')
    await loadAll()
  }

  async function markDone(row) {
    await request(`/appointments/${row.id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status: 'checked' }),
    })
    reportForm.appointmentId = row.id
    ElMessage.success('已标记完成体检，可继续生成报告')
    await loadAll()
  }

  async function createReport() {
    if (!currentUser.value) return
    await request('/reports', {
      method: 'POST',
      body: JSON.stringify({ ...reportForm, doctorId: currentUser.value.id }),
    })
    ElMessage.success('报告已生成')
    await loadAll()
  }

  function selectPackage(pkg) {
    appointmentForm.packageId = pkg.id
  }

  return {
    role,
    loginForm,
    currentUser,
    packages,
    appointments,
    reports,
    appointmentForm,
    reportForm,
    isUser,
    isDoctor,
    myAppointments,
    bookedCount,
    reportedCount,
    pendingDoctorCount,
    peopleRows,
    login,
    quickLogin,
    loadAll,
    ensureBootstrapped,
    createAppointment,
    markDone,
    createReport,
    selectPackage,
  }
}

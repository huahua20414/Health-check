<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="topbar-inner">
        <div class="brand">
          <h1>东软熙心健康体检管理系统</h1>
          <p>用户预约、医生报告、用户查看报告完整演示闭环</p>
        </div>
        <el-tag type="success" size="large">Docker + Go/Gin + Vue3 + MySQL</el-tag>
      </div>
    </header>

    <main class="content">
      <section class="toolbar">
        <el-segmented v-model="role" :options="roleOptions" @change="quickLogin" />
        <el-input v-model="loginForm.phone" placeholder="手机号" style="width: 180px" />
        <el-input v-model="loginForm.name" placeholder="姓名" style="width: 160px" />
        <el-button type="primary" @click="login">登录/切换身份</el-button>
        <el-button @click="loadAll">刷新数据</el-button>
        <span v-if="currentUser" class="muted">当前：{{ currentUser.name }}（{{ currentUser.role }}）</span>
      </section>

      <section class="grid">
        <div class="panel">
          <h2>体检套餐</h2>
          <div v-for="pkg in packages" :key="pkg.id" class="package-card">
            <h3>{{ pkg.name }}</h3>
            <p class="muted">{{ pkg.description }}</p>
            <p>{{ pkg.items }}</p>
            <p class="price">￥{{ pkg.price }}</p>
            <el-button type="primary" plain :disabled="!isUser" @click="selectPackage(pkg)">预约</el-button>
          </div>
        </div>

        <div class="panel">
          <h2>用户预约</h2>
          <el-form label-position="top" class="stack">
            <el-form-item label="套餐">
              <el-select v-model="appointmentForm.packageId" placeholder="选择套餐">
                <el-option v-for="pkg in packages" :key="pkg.id" :label="pkg.name" :value="pkg.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="日期">
              <el-date-picker v-model="appointmentForm.date" value-format="YYYY-MM-DD" type="date" placeholder="选择日期" style="width: 100%" />
            </el-form-item>
            <el-form-item label="时段">
              <el-select v-model="appointmentForm.period">
                <el-option label="上午" value="上午" />
                <el-option label="下午" value="下午" />
              </el-select>
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="appointmentForm.note" type="textarea" :rows="3" />
            </el-form-item>
            <el-button type="primary" :disabled="!isUser" @click="createAppointment">提交预约</el-button>
          </el-form>
        </div>

        <div class="panel">
          <h2>医生报告</h2>
          <el-form label-position="top" class="stack">
            <el-form-item label="预约记录">
              <el-select v-model="reportForm.appointmentId" placeholder="选择预约">
                <el-option
                  v-for="item in appointments"
                  :key="item.id"
                  :label="`#${item.id} ${item.user?.name || ''} ${item.package?.name || ''}`"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="检查摘要">
              <el-input v-model="reportForm.summary" type="textarea" :rows="3" />
            </el-form-item>
            <el-form-item label="结论">
              <el-input v-model="reportForm.conclusion" type="textarea" :rows="2" />
            </el-form-item>
            <el-form-item label="建议">
              <el-input v-model="reportForm.recommendation" type="textarea" :rows="2" />
            </el-form-item>
            <el-button type="success" :disabled="!isDoctor" @click="createReport">生成报告</el-button>
          </el-form>
        </div>

        <div class="panel wide">
          <h2>预约列表</h2>
          <el-table :data="appointments" stripe>
            <el-table-column prop="id" label="编号" width="70" />
            <el-table-column label="用户" width="100">
              <template #default="{ row }">{{ row.user?.name }}</template>
            </el-table-column>
            <el-table-column label="套餐">
              <template #default="{ row }">{{ row.package?.name }}</template>
            </el-table-column>
            <el-table-column prop="date" label="日期" width="120" />
            <el-table-column prop="period" label="时段" width="90" />
            <el-table-column prop="status" label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="row.status === 'reported' ? 'success' : 'warning'">{{ statusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="160">
              <template #default="{ row }">
                <div class="actions">
                  <el-button size="small" :disabled="!isDoctor" @click="markDone(row)">完成体检</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="panel">
          <h2>我的报告</h2>
          <div class="stack">
            <el-empty v-if="reports.length === 0" description="暂无报告" />
            <el-card v-for="report in reports" :key="report.id" shadow="never">
              <template #header>
                <strong>{{ report.appointment?.package?.name || '体检报告' }}</strong>
              </template>
              <p><strong>摘要：</strong>{{ report.summary }}</p>
              <p><strong>结论：</strong>{{ report.conclusion }}</p>
              <p><strong>建议：</strong>{{ report.recommendation }}</p>
              <p class="muted">医生：{{ report.doctor?.name }}</p>
            </el-card>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

const apiBase = import.meta.env.VITE_API_BASE || '/api'
const role = ref('user')
const roleOptions = [
  { label: '用户端', value: 'user' },
  { label: '医生端', value: 'doctor' }
]
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
  recommendation: '建议保持规律作息，按年度复查。'
})

const isUser = computed(() => currentUser.value?.role === 'user')
const isDoctor = computed(() => currentUser.value?.role === 'doctor')

async function request(path, options = {}) {
  const response = await fetch(`${apiBase}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options
  })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(error.error || response.statusText)
  }
  return response.json()
}

function quickLogin() {
  if (role.value === 'doctor') {
    loginForm.phone = '13900000001'
    loginForm.name = '李医生'
  } else {
    loginForm.phone = '13800000001'
    loginForm.name = '张三'
  }
  login()
}

async function login() {
  currentUser.value = await request('/login', {
    method: 'POST',
    body: JSON.stringify({ ...loginForm, role: role.value })
  })
  await loadAll()
  ElMessage.success('登录成功')
}

async function loadAll() {
  packages.value = await request('/packages')
  appointments.value = await request('/appointments')
  const userId = isUser.value ? `?userId=${currentUser.value.id}` : ''
  reports.value = currentUser.value ? await request(`/reports${userId}`) : []
  if (!appointmentForm.packageId && packages.value[0]) {
    appointmentForm.packageId = packages.value[0].id
  }
  if (!reportForm.appointmentId && appointments.value[0]) {
    reportForm.appointmentId = appointments.value[0].id
  }
}

function selectPackage(pkg) {
  appointmentForm.packageId = pkg.id
}

async function createAppointment() {
  if (!currentUser.value) return
  await request('/appointments', {
    method: 'POST',
    body: JSON.stringify({ ...appointmentForm, userId: currentUser.value.id })
  })
  ElMessage.success('预约已提交')
  await loadAll()
}

async function markDone(row) {
  await request(`/appointments/${row.id}/status`, {
    method: 'PATCH',
    body: JSON.stringify({ status: 'checked' })
  })
  reportForm.appointmentId = row.id
  ElMessage.success('已标记完成体检')
  await loadAll()
}

async function createReport() {
  if (!currentUser.value) return
  await request('/reports', {
    method: 'POST',
    body: JSON.stringify({ ...reportForm, doctorId: currentUser.value.id })
  })
  ElMessage.success('报告已生成')
  await loadAll()
}

function statusText(status) {
  return { booked: '已预约', checked: '已体检', reported: '已出报告' }[status] || status
}

onMounted(async () => {
  await login()
})
</script>

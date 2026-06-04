<template>
  <header class="topbar">
    <div>
      <p class="eyebrow">东软熙心健康体检管理系统</p>
      <h2>{{ route.meta.title || '工作台' }}</h2>
    </div>
    <div class="top-actions">
      <el-segmented v-model="role" :options="roleOptions" @change="handleRoleChange" />
      <el-input v-model="loginForm.phone" placeholder="手机号" class="login-input" />
      <el-input v-model="loginForm.name" placeholder="姓名" class="login-input compact" />
      <el-button type="primary" @click="login">切换身份</el-button>
      <el-button :icon="Refresh" @click="loadAll">刷新</el-button>
    </div>
  </header>
</template>

<script setup>
import { useRouter, useRoute } from 'vue-router'
import { Refresh } from '@element-plus/icons-vue'
import { useHealthData } from '../composables/useHealthData'

const router = useRouter()
const route = useRoute()
const { role, loginForm, login, quickLogin, loadAll } = useHealthData()
const roleOptions = [
  { label: '用户端', value: 'user' },
  { label: '医生端', value: 'doctor' },
]

async function handleRoleChange(nextRole) {
  await quickLogin(nextRole)
  if (nextRole === 'doctor') router.push('/appointments')
  if (nextRole === 'user' && route.path === '/appointments') router.push('/')
}
</script>

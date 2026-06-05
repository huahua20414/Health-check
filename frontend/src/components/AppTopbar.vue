<template>
  <header class="topbar">
    <div>
      <p class="eyebrow">东软熙心健康体检管理系统</p>
      <h2>{{ route.meta.title || '工作台' }}</h2>
    </div>
    <div class="top-actions">
      <div class="account-chip">
        <strong>{{ currentUser?.name }}</strong>
        <span>{{ roleLabel }}</span>
      </div>
      <el-button :icon="Refresh" :loading="loading.load" @click="loadAll">刷新</el-button>
      <el-button type="danger" plain :loading="loading.logout" @click="handleLogout">退出登录</el-button>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Refresh } from '@element-plus/icons-vue'
import { useHealthData } from '../composables/useHealthData'

const router = useRouter()
const route = useRoute()
const { currentUser, loading, loadAll, logout } = useHealthData()
const roleLabel = computed(() => ({ user: '用户端', doctor: '医生端', admin: '管理端' }[currentUser.value?.role] || '-'))

async function handleLogout() {
  await logout()
  router.push('/login')
}
</script>

<template>
  <div class="admin-shell">
    <AppSidebar />
    <section class="main-shell">
      <AppTopbar />
      <main class="workspace">
        <section class="operator-strip">
          <div>
            <span class="muted">当前登录</span>
            <strong>{{ currentUser?.name || '-' }}</strong>
            <el-tag :type="isDoctor ? 'warning' : isAdmin ? 'danger' : 'success'">{{ roleLabel }}</el-tag>
          </div>
          <div>
            <span class="muted">系统状态</span>
            <strong>服务运行正常</strong>
            <el-tag type="success">Docker 已启动</el-tag>
          </div>
        </section>
        <router-view />
      </main>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppSidebar from '../components/AppSidebar.vue'
import AppTopbar from '../components/AppTopbar.vue'
import { useHealthData } from '../composables/useHealthData'
import { menuItems } from '../router'

const { currentUser, isDoctor, isAdmin, ensureBootstrapped } = useHealthData()
const route = useRoute()
const router = useRouter()

function enforceRouteAccess() {
  const role = currentUser.value?.role
  if (!role) return
  const routeMenu = menuItems.find((item) => item.name === route.name)
  if (routeMenu && !routeMenu.roles.includes(role)) {
    router.replace(role === 'doctor' ? '/doctor' : role === 'admin' ? '/admin' : '/')
  }
}

const roleLabel = computed(() => ({ user: '用户端', doctor: '医生端', admin: '管理端' }[currentUser.value?.role] || '-'))

onMounted(async () => {
  await ensureBootstrapped()
  enforceRouteAccess()
})

watch([currentUser, () => route.name], enforceRouteAccess)
</script>

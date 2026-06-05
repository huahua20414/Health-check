<template>
  <div class="admin-shell">
    <AppSidebar />
    <section class="main-shell">
      <AppTopbar />
      <main class="workspace" :class="{ 'booking-workspace': route.name === 'booking' }">
        <section v-if="route.name !== 'booking'" class="operator-strip">
          <div>
            <span class="muted">当前登录</span>
            <strong>{{ currentUser?.name || '-' }}</strong>
            <el-tag :type="isDoctor ? 'warning' : isAdmin ? 'danger' : 'success'">{{ roleLabel }}</el-tag>
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
import { routeMenuItems } from '../router'

const { currentUser, isDoctor, isAdmin, ensureBootstrapped } = useHealthData()
const route = useRoute()
const router = useRouter()

function enforceRouteAccess() {
  const role = currentUser.value?.role
  if (!role) return
  const routeMenu = routeMenuItems.find((item) => item.name === route.name)
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

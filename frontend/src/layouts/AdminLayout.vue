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
            <el-tag :type="isDoctor ? 'warning' : 'success'">{{ isDoctor ? '医生端' : '用户端' }}</el-tag>
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
import { onMounted } from 'vue'
import AppSidebar from '../components/AppSidebar.vue'
import AppTopbar from '../components/AppTopbar.vue'
import { useHealthData } from '../composables/useHealthData'

const { currentUser, isDoctor, ensureBootstrapped } = useHealthData()

onMounted(ensureBootstrapped)
</script>

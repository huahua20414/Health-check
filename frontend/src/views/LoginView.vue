<template>
  <main class="auth-page">
    <section class="auth-card">
      <div class="auth-hero">
        <div class="brand-mark">熙</div>
        <h1>熙心健康体检管理系统</h1>
        <p>统一认证入口，登录后根据账号角色进入对应工作台。</p>
      </div>
      <el-form class="auth-form" label-position="top" @submit.prevent>
        <el-form-item label="手机号">
          <el-input v-model="loginForm.phone" placeholder="请输入手机号" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="loginForm.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-button type="primary" size="large" :loading="loading.login" @click="handleLogin">登录系统</el-button>
        <div class="auth-links">
          <router-link to="/register/user">用户注册</router-link>
          <router-link to="/register/doctor">医生注册</router-link>
        </div>
        <div class="demo-accounts">
          <span>用户：13800000001 / 123456</span>
          <span>医生：13900000001 / 123456</span>
          <span>管理员：13700000001 / admin123</span>
        </div>
      </el-form>
    </section>
  </main>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { homePath } from '../router'
import { useHealthData } from '../composables/useHealthData'

const router = useRouter()
const { loginForm, loading, login } = useHealthData()

async function handleLogin() {
  try {
    const user = await login()
    if (user) router.push(homePath(user.role))
  } catch (error) {
    ElMessage.error(error.message)
  }
}
</script>

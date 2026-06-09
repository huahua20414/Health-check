<template>
  <main class="auth-page">
    <section class="auth-card">
      <div class="auth-hero">
        <div class="brand-mark">熙</div>
        <h1>熙心健康体检管理系统</h1>
        <p>统一认证入口，登录后根据账号角色进入对应工作台。</p>
      </div>
      <el-form class="auth-form" label-position="top" @submit.prevent>
        <el-form-item v-if="devAuthEnabled" label="登录身份">
          <el-segmented v-model="loginForm.role" :options="roleOptions" class="role-segmented" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="loginForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="loginForm.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item v-if="!devAuthEnabled" label="邮箱验证码">
          <div class="inline-code">
            <el-input v-model="loginForm.code" maxlength="6" placeholder="6 位验证码" />
            <el-button :loading="loading.authCode" :disabled="!loginForm.email" @click="sendCode">发送验证码</el-button>
          </div>
        </el-form-item>
        <div v-if="devAuthEnabled" class="dev-login-panel">
          <el-button @click="quickLogin('user')">用户端</el-button>
          <el-button @click="quickLogin('doctor')">医生端</el-button>
          <el-button @click="quickLogin('admin')">管理端</el-button>
        </div>
        <el-button type="primary" size="large" :loading="loading.login" @click="handleLogin">登录系统</el-button>
        <div class="auth-links">
          <router-link to="/register/user">用户注册</router-link>
          <router-link to="/register/doctor">医生注册</router-link>
        </div>
      </el-form>
    </section>
  </main>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { homePath } from '../router'
import { devAuthEnabled, useHealthData } from '../composables/useHealthData'
import { useDebouncedFn } from '../composables/useDebouncedFn'

const router = useRouter()
const { loginForm, loading, login, sendAuthEmailCode } = useHealthData()
const sendCode = useDebouncedFn(() => sendAuthEmailCode(loginForm.email), 800)
const roleOptions = [
  { label: '用户', value: 'user' },
  { label: '医生', value: 'doctor' },
  { label: '管理员', value: 'admin' },
]

async function handleLogin() {
  try {
    const user = await login()
    if (user) router.push(homePath(user.role))
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function quickLogin(role) {
  loginForm.role = role
  loginForm.email = 'huahua20414@foxmail.com'
  loginForm.password = '123456'
  loginForm.code = ''
  await handleLogin()
}
</script>

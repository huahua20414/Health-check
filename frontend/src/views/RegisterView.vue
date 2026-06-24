<template>
  <main class="auth-page">
    <section class="auth-card">
      <div class="auth-hero">
        <div class="brand-mark">熙</div>
        <h1>{{ isDoctorRegister ? '医生注册' : '用户注册' }}</h1>
        <p>{{ isDoctorRegister ? '医生账号提交后需要管理员审核。' : '注册后可预约体检并查看报告。' }}</p>
      </div>

      <el-form class="auth-form" label-position="top" @submit.prevent>
        <template v-if="!isDoctorRegister">
          <el-form-item label="姓名"><el-input v-model="userRegisterForm.name" /></el-form-item>
          <el-form-item label="邮箱"><el-input v-model="userRegisterForm.email" /></el-form-item>
          <el-form-item label="邮箱验证码">
            <div class="inline-code">
              <el-input v-model="userRegisterForm.code" maxlength="6" placeholder="6 位验证码" />
              <el-button :loading="loading.authCode" :disabled="!userRegisterForm.email || authCodeCooldown > 0" @click="sendUserCode">{{ codeButtonText }}</el-button>
            </div>
          </el-form-item>
          <el-form-item label="性别">
            <el-select v-model="userRegisterForm.gender" placeholder="请选择">
              <el-option label="男" value="男" />
              <el-option label="女" value="女" />
            </el-select>
          </el-form-item>
          <el-form-item label="身份证号"><el-input v-model="userRegisterForm.idCard" maxlength="18" /></el-form-item>
          <el-form-item label="年龄"><el-input :model-value="userAgeText" disabled /></el-form-item>
        </template>

        <template v-else>
          <el-form-item label="姓名"><el-input v-model="doctorRegisterForm.name" /></el-form-item>
          <el-form-item label="邮箱"><el-input v-model="doctorRegisterForm.email" /></el-form-item>
          <el-form-item label="邮箱验证码">
            <div class="inline-code">
              <el-input v-model="doctorRegisterForm.code" maxlength="6" placeholder="6 位验证码" />
              <el-button :loading="loading.authCode" :disabled="!doctorRegisterForm.email || authCodeCooldown > 0" @click="sendDoctorCode">{{ codeButtonText }}</el-button>
            </div>
          </el-form-item>
          <el-form-item label="工号"><el-input v-model="doctorRegisterForm.employeeNo" /></el-form-item>
          <el-form-item label="科室">
            <el-select v-model="doctorRegisterForm.department" placeholder="请选择科室">
              <el-option v-for="department in doctorDepartments" :key="department" :label="department" :value="department" />
            </el-select>
          </el-form-item>
          <el-form-item label="职称"><el-input v-model="doctorRegisterForm.title" /></el-form-item>
        </template>

        <el-button type="primary" size="large" :loading="loading.register" :disabled="!canSubmit" @click="submit">提交注册</el-button>
        <div class="auth-links"><router-link to="/login">返回登录</router-link></div>
      </el-form>
    </section>
  </main>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { calculateAgeFromIDCard, doctorDepartments, useHealthData } from '../composables/useHealthData'

const route = useRoute()
const router = useRouter()
const isDoctorRegister = computed(() => route.params.role === 'doctor')
const { userRegisterForm, doctorRegisterForm, loading, authCodeCooldown, registerUser, registerDoctor, sendAuthEmailCode } = useHealthData()
const userAge = computed(() => calculateAgeFromIDCard(userRegisterForm.idCard))
const userAgeText = computed(() => (userAge.value === null ? '身份证校验通过后自动计算' : String(userAge.value) + ' 岁'))
const codeButtonText = computed(() => {
  if (loading.authCode) return '发送中'
  if (authCodeCooldown.value > 0) return `${authCodeCooldown.value} 秒后重发`
  return '发送验证码'
})
const canSubmit = computed(() => {
  if (isDoctorRegister.value) {
    return Boolean(doctorRegisterForm.name && doctorRegisterForm.email && doctorRegisterForm.code?.length === 6 && doctorRegisterForm.employeeNo && doctorRegisterForm.department && doctorRegisterForm.title)
  }
  return Boolean(userRegisterForm.name && userRegisterForm.email && userRegisterForm.code?.length === 6 && userAge.value !== null)
})

async function submit() {
  try {
    if (isDoctorRegister.value) {
      await registerDoctor()
      router.push('/login')
      return
    }
    const user = await registerUser()
    if (user) router.push(user.role === 'doctor' ? '/doctor' : user.role === 'admin' ? '/admin' : '/')
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function sendUserCode() {
  try {
    await sendAuthEmailCode(userRegisterForm.email)
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function sendDoctorCode() {
  try {
    await sendAuthEmailCode(doctorRegisterForm.email)
  } catch (error) {
    ElMessage.error(error.message)
  }
}
</script>

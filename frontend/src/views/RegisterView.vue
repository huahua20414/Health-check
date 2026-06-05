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
          <el-form-item label="手机号"><el-input v-model="userRegisterForm.phone" /></el-form-item>
          <el-form-item label="性别">
            <el-select v-model="userRegisterForm.gender" placeholder="请选择">
              <el-option label="男" value="男" />
              <el-option label="女" value="女" />
            </el-select>
          </el-form-item>
          <el-form-item label="年龄"><el-input-number v-model="userRegisterForm.age" :min="0" :max="120" /></el-form-item>
          <el-form-item label="身份证号"><el-input v-model="userRegisterForm.idCard" /></el-form-item>
          <el-form-item label="密码"><el-input v-model="userRegisterForm.password" type="password" show-password /></el-form-item>
          <el-form-item label="确认密码"><el-input v-model="userRegisterForm.confirmPassword" type="password" show-password /></el-form-item>
        </template>

        <template v-else>
          <el-form-item label="姓名"><el-input v-model="doctorRegisterForm.name" /></el-form-item>
          <el-form-item label="手机号"><el-input v-model="doctorRegisterForm.phone" /></el-form-item>
          <el-form-item label="工号"><el-input v-model="doctorRegisterForm.employeeNo" /></el-form-item>
          <el-form-item label="科室"><el-input v-model="doctorRegisterForm.department" /></el-form-item>
          <el-form-item label="职称"><el-input v-model="doctorRegisterForm.title" /></el-form-item>
          <el-form-item label="密码"><el-input v-model="doctorRegisterForm.password" type="password" show-password /></el-form-item>
          <el-form-item label="确认密码"><el-input v-model="doctorRegisterForm.confirmPassword" type="password" show-password /></el-form-item>
        </template>

        <el-button type="primary" size="large" :loading="loading.register" @click="submit">提交注册</el-button>
        <div class="auth-links"><router-link to="/login">返回登录</router-link></div>
      </el-form>
    </section>
  </main>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useHealthData } from '../composables/useHealthData'

const route = useRoute()
const router = useRouter()
const isDoctorRegister = computed(() => route.params.role === 'doctor')
const { userRegisterForm, doctorRegisterForm, loading, registerUser, registerDoctor } = useHealthData()

async function submit() {
  try {
    if (isDoctorRegister.value) await registerDoctor()
    else await registerUser()
    router.push('/login')
  } catch (error) {
    ElMessage.error(error.message)
  }
}
</script>

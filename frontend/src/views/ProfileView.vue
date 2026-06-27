<template>
  <section class="view">
    <div class="profile-layout">
      <aside class="profile-card panel">
        <div class="avatar">{{ initials }}</div>
        <h3>{{ currentUser?.name }}</h3>
        <p>{{ currentUser?.email }}</p>
        <el-tag :type="currentUser?.status === 'active' ? 'success' : 'warning'">{{ statusText(currentUser?.status || 'active') }}</el-tag>
        <div class="profile-stats">
          <div><strong>{{ myAppointments.length }}</strong><span>预约</span></div>
          <div><strong>{{ reports.length }}</strong><span>报告</span></div>
          <div><strong>{{ waitlist.length }}</strong><span>候补</span></div>
        </div>
        <div class="profile-meta">
          <span>当前邮箱</span>
          <strong>{{ currentUser?.email || '未绑定' }}</strong>
        </div>
      </aside>

      <div class="panel profile-main">
        <div class="panel-head">
          <div>
            <h3>资料设置</h3>
            <p>邮箱用于接收预约分配、候补递补和报告生成通知。</p>
          </div>
        </div>
        <el-form label-position="top" class="profile-form">
          <el-form-item label="姓名"><el-input v-model="profileForm.name" /></el-form-item>
          <el-form-item label="性别">
            <el-select v-model="profileForm.gender" clearable>
              <el-option label="男" value="男" />
              <el-option label="女" value="女" />
            </el-select>
          </el-form-item>
          <el-form-item label="身份证号"><el-input v-model="profileForm.idCard" maxlength="18" /></el-form-item>
          <el-form-item label="年龄"><el-input :model-value="profileAgeText" disabled /></el-form-item>
          <el-form-item label="个人简介"><el-input v-model="profileForm.bio" type="textarea" :rows="4" /></el-form-item>
          <el-form-item label="消息通知">
            <el-switch v-model="profileForm.emailNotify" active-text="接收邮件" inactive-text="不接收" />
          </el-form-item>
          <el-button type="primary" :loading="loading.profile" @click="submit">保存资料</el-button>
        </el-form>

        <div class="email-verify">
          <div class="section-head">
            <div>
              <h4>邮箱验证</h4>
              <p>修改邮箱前必须向目标邮箱发送验证码，验证通过后才会生效。</p>
            </div>
            <el-tag type="success" v-if="currentUser?.email">已绑定</el-tag>
            <el-tag type="warning" v-else>未绑定</el-tag>
          </div>
          <el-form label-position="top" class="email-form">
            <el-form-item label="目标邮箱">
              <el-input v-model="emailForm.email" placeholder="请输入要绑定的新邮箱" />
            </el-form-item>
            <el-form-item label="验证码">
              <el-input v-model="emailForm.code" maxlength="6" placeholder="6 位验证码" />
            </el-form-item>
            <div class="email-actions">
              <el-button :loading="loading.emailCode" :disabled="!emailForm.email" @click="sendCode">发送验证码</el-button>
              <el-button type="primary" :loading="loading.emailUpdate" :disabled="!canUpdateEmail" @click="confirmEmail">验证并更换邮箱</el-button>
            </div>
          </el-form>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { calculateAgeFromIDCard, statusText, useHealthData } from '../composables/useHealthData'
import { useDebouncedFn } from '../composables/useDebouncedFn'

const { currentUser, profileForm, emailForm, myAppointments, reports, waitlist, loading, saveProfile, sendEmailCode, updateEmail } = useHealthData()
const initials = computed(() => currentUser.value?.name?.slice(0, 1) || '用')
const profileAge = computed(() => calculateAgeFromIDCard(profileForm.idCard))
const profileAgeText = computed(() => {
  if (!profileForm.idCard) return '填写身份证后自动计算'
  return profileAge.value === null ? '身份证号未通过校验' : String(profileAge.value) + ' 岁'
})
const submit = useDebouncedFn(saveProfile, 350)
const sendCode = useDebouncedFn(sendEmailCode, 800)
const confirmEmail = useDebouncedFn(updateEmail, 500)
const canUpdateEmail = computed(() => Boolean(emailForm.email && emailForm.code?.length === 6))
</script>

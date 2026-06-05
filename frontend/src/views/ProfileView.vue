<template>
  <section class="view">
    <div class="profile-layout">
      <aside class="profile-card panel">
        <div class="avatar">{{ initials }}</div>
        <h3>{{ currentUser?.name }}</h3>
        <p>{{ currentUser?.phone }}</p>
        <el-tag :type="currentUser?.status === 'active' ? 'success' : 'warning'">{{ statusText(currentUser?.status || 'active') }}</el-tag>
        <div class="profile-stats">
          <div><strong>{{ myAppointments.length }}</strong><span>预约</span></div>
          <div><strong>{{ reports.length }}</strong><span>报告</span></div>
          <div><strong>{{ waitlist.length }}</strong><span>候补</span></div>
        </div>
      </aside>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>资料设置</h3>
            <p>邮箱用于接收预约分配、候补递补和报告生成通知。</p>
          </div>
        </div>
        <el-form label-position="top" class="profile-form">
          <el-form-item label="姓名"><el-input v-model="profileForm.name" /></el-form-item>
          <el-form-item label="邮箱"><el-input v-model="profileForm.email" placeholder="用于接收通知邮件" /></el-form-item>
          <el-form-item label="性别">
            <el-select v-model="profileForm.gender" clearable>
              <el-option label="男" value="男" />
              <el-option label="女" value="女" />
            </el-select>
          </el-form-item>
          <el-form-item label="年龄"><el-input-number v-model="profileForm.age" :min="0" :max="120" /></el-form-item>
          <el-form-item label="身份证号"><el-input v-model="profileForm.idCard" /></el-form-item>
          <el-form-item label="头像 URL"><el-input v-model="profileForm.avatarUrl" /></el-form-item>
          <el-form-item label="个人简介"><el-input v-model="profileForm.bio" type="textarea" :rows="4" /></el-form-item>
          <el-form-item label="消息通知">
            <el-switch v-model="profileForm.emailNotify" active-text="接收邮件" inactive-text="不接收" />
          </el-form-item>
          <el-button type="primary" :loading="loading.profile" @click="submit">保存资料</el-button>
        </el-form>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { statusText, useHealthData } from '../composables/useHealthData'
import { useDebouncedFn } from '../composables/useDebouncedFn'

const { currentUser, profileForm, myAppointments, reports, waitlist, loading, saveProfile } = useHealthData()
const initials = computed(() => currentUser.value?.name?.slice(0, 1) || '用')
const submit = useDebouncedFn(saveProfile, 350)
</script>

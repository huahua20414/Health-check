<template>
  <aside class="sidebar">
    <div class="system-brand">
      <div class="brand-mark">熙</div>
      <div>
        <h1>熙心体检</h1>
        <p>健康体检管理系统</p>
      </div>
    </div>

    <el-menu
      class="side-menu"
      router
      :default-active="activePath"
      background-color="transparent"
      text-color="#c9d7e5"
      active-text-color="#ffffff"
    >
      <el-menu-item v-for="item in visibleMenuItems" :key="item.name" :index="item.path">
        <el-icon><component :is="icons[item.icon]" /></el-icon>
        <span>{{ item.label }}</span>
      </el-menu-item>
    </el-menu>

    <div class="sidebar-footer">
      <p>演示账号</p>
      <span>用户 13800000001 / 123456</span>
      <span>医生 13900000001 / 123456</span>
      <span>管理员 13700000001 / admin123</span>
    </div>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  Calendar,
  DataAnalysis,
  Document,
  DocumentChecked,
  Files,
  House,
  Setting,
  Tickets,
  User,
} from '@element-plus/icons-vue'
import { menuItems } from '../router'
import { useHealthData } from '../composables/useHealthData'

const route = useRoute()
const { currentUser } = useHealthData()
const icons = { Calendar, DataAnalysis, Document, DocumentChecked, Files, House, Setting, Tickets, User }
const activePath = computed(() => route.path)
const visibleMenuItems = computed(() => {
  const currentRole = currentUser.value?.role
  return menuItems.filter((item) => item.roles.includes(currentRole))
})
</script>

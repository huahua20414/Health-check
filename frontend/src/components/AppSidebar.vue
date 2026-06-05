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
      <template v-for="item in visibleMenuItems" :key="item.name || item.label">
        <el-sub-menu v-if="item.children" :index="item.label">
          <template #title>
            <el-icon><component :is="icons[item.icon]" /></el-icon>
            <span>{{ item.label }}</span>
          </template>
          <el-menu-item v-for="child in item.children" :key="child.name" :index="child.path">
            <el-icon><component :is="icons[child.icon]" /></el-icon>
            <span>{{ child.label }}</span>
          </el-menu-item>
        </el-sub-menu>
        <el-menu-item v-else :index="item.path">
          <el-icon><component :is="icons[item.icon]" /></el-icon>
          <span>{{ item.label }}</span>
        </el-menu-item>
      </template>
    </el-menu>

    <div class="sidebar-footer">
      <p>{{ currentUser?.name || '未登录' }}</p>
      <span>{{ currentUser?.email || '' }}</span>
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

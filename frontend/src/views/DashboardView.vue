<template>
  <section class="view">
    <div class="metric-grid">
      <div class="metric-card">
        <span>套餐数量</span>
        <strong>{{ packages.length }}</strong>
        <p>覆盖基础、白领、深度体检</p>
      </div>
      <div class="metric-card">
        <span>预约总数</span>
        <strong>{{ appointments.length }}</strong>
        <p>{{ bookedCount }} 个待体检，{{ reportedCount }} 个已出报告</p>
      </div>
      <div class="metric-card">
        <span>报告数量</span>
        <strong>{{ reports.length }}</strong>
        <p>支持医生生成、用户查看</p>
      </div>
      <div class="metric-card">
        <span>今日待办</span>
        <strong>{{ pendingDoctorCount }}</strong>
        <p>医生可在预约管理中处理</p>
      </div>
    </div>

    <div class="layout-two">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>待处理事项</h3>
            <p>根据当前角色和实时业务数据生成</p>
          </div>
        </div>
        <div class="todo-list">
          <div v-for="item in todoItems" :key="item.title" class="todo-item">
            <strong>{{ item.count }}</strong>
            <div>
              <h4>{{ item.title }}</h4>
              <p>{{ item.description }}</p>
            </div>
          </div>
          <el-empty v-if="todoItems.length === 0" description="暂无待处理事项" />
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>最近预约</h3>
            <p>用于前台和医生端快速掌握进度</p>
          </div>
        </div>
        <el-table :data="appointments.slice(0, 5)" stripe>
          <el-table-column label="客户" width="90">
            <template #default="{ row }">{{ row.user?.name }}</template>
          </el-table-column>
          <el-table-column label="套餐">
            <template #default="{ row }">{{ row.package?.name }}</template>
          </el-table-column>
          <el-table-column prop="date" label="日期" width="120" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }"><StatusTag :status="row.status" /></template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </section>
</template>

<script setup>
import StatusTag from '../components/StatusTag.vue'
import { useHealthData } from '../composables/useHealthData'

import { computed } from 'vue'

const { packages, appointments, reports, bookedCount, reportedCount, pendingDoctorCount, pendingDoctors, isAdmin, isDoctor, isUser } = useHealthData()

const todoItems = computed(() => {
  if (isAdmin.value) {
    return [
      { title: '待审核医生', count: pendingDoctors.value.length, description: '医生注册后需要管理员审核启用' },
    ].filter((item) => item.count > 0)
  }
  if (isDoctor.value) {
    return [
      { title: '待体检预约', count: appointments.value.filter((item) => item.status === 'booked').length, description: '需要确认客户到检并更新状态' },
      { title: '待生成报告', count: appointments.value.filter((item) => item.status === 'checked').length, description: '已完成体检但尚未归档报告' },
    ].filter((item) => item.count > 0)
  }
  if (isUser.value) {
    return [
      { title: '我的有效预约', count: appointments.value.filter((item) => item.status !== 'reported').length, description: '请按预约日期到检' },
      { title: '可查看报告', count: reports.value.length, description: '体检报告由医生生成后展示' },
    ].filter((item) => item.count > 0)
  }
  return []
})
</script>

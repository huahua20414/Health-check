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
            <h3>业务流程</h3>
            <p>覆盖体检预约、执行、报告生成、报告查看</p>
          </div>
        </div>
        <el-steps :active="3" finish-status="success">
          <el-step title="选择套餐" description="用户根据需求选择体检套餐" />
          <el-step title="提交预约" description="选择日期和时段生成预约" />
          <el-step title="医生处理" description="确认体检完成并录入结论" />
          <el-step title="查看报告" description="用户端查看体检建议" />
        </el-steps>
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

const { packages, appointments, reports, bookedCount, reportedCount, pendingDoctorCount } = useHealthData()
</script>

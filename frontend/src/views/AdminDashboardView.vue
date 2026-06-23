<template>
  <section class="view">
    <div class="metric-grid">
      <div class="metric-card">
        <span>用户数</span>
        <strong>{{ summary.users || 0 }}</strong>
        <p>平台注册用户总量</p>
      </div>
      <div class="metric-card">
        <span>医生数</span>
        <strong>{{ summary.doctors || 0 }}</strong>
        <p>含待审核与启用医生</p>
      </div>
      <div class="metric-card">
        <span>预约数</span>
        <strong>{{ summary.appointments || 0 }}</strong>
        <p>全部预约订单</p>
      </div>
      <div class="metric-card">
        <span>平均评分</span>
        <strong>{{ Number(summary.averageRating || 0).toFixed(1) }}</strong>
        <p>{{ summary.reviews || 0 }} 条服务评价</p>
      </div>
    </div>

    <div class="layout-two">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>预约趋势</h3>
            <p>按体检日期聚合预约量。</p>
          </div>
        </div>
        <div class="bar-list">
          <div v-for="row in adminDashboard.appointmentTrend" :key="row.label" class="bar-row">
            <span>{{ row.label || '未排期' }}</span>
            <div><i :style="{ width: `${barWidth(row.count, maxTrend)}%` }" /></div>
            <strong>{{ row.count }}</strong>
          </div>
          <el-empty v-if="adminDashboard.appointmentTrend.length === 0" description="暂无趋势数据" />
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>套餐销量</h3>
            <p>统计非取消预约的套餐销量和销售额。</p>
          </div>
        </div>
        <el-table :data="adminDashboard.packageSales" stripe>
          <el-table-column prop="label" label="套餐" />
          <el-table-column prop="count" label="销量" width="90" />
          <el-table-column label="销售额" width="120">
            <template #default="{ row }">￥{{ Number(row.total || 0).toFixed(2) }}</template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>用户增长</h3>
          <p>按注册日期统计新增用户。</p>
        </div>
      </div>
      <div class="bar-list compact-bars">
        <div v-for="row in adminDashboard.userGrowth" :key="row.label" class="bar-row">
          <span>{{ row.label }}</span>
          <div><i :style="{ width: `${barWidth(row.count, maxGrowth)}%` }" /></div>
          <strong>{{ row.count }}</strong>
        </div>
        <el-empty v-if="adminDashboard.userGrowth.length === 0" description="暂无增长数据" />
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useHealthData } from '../composables/useHealthData'

const { adminDashboard, loadAdminDashboard } = useHealthData()
const summary = computed(() => adminDashboard.value.summary || {})
const maxTrend = computed(() => Math.max(1, ...adminDashboard.value.appointmentTrend.map((item) => item.count || 0)))
const maxGrowth = computed(() => Math.max(1, ...adminDashboard.value.userGrowth.map((item) => item.count || 0)))

function barWidth(value, max) {
  return Math.max(6, Math.round((Number(value || 0) / Number(max.value || 1)) * 100))
}

onMounted(loadAdminDashboard)
</script>

<template>
  <section class="view">
    <div class="panel compact-toolbar">
      <div class="panel-head">
        <div>
          <h3>数据大屏</h3>
          <p>{{ dashboardRange.appointmentStartDate }} 至 {{ dashboardRange.appointmentEndDate }} 的预约趋势。</p>
        </div>
        <el-segmented v-model="daysFilter" :options="rangeOptions" @change="refreshDashboard" />
      </div>
    </div>

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
      <div class="metric-card">
        <span>候补人数</span>
        <strong>{{ summary.waitlist || 0 }}</strong>
        <p>当前周期等待递补</p>
      </div>
      <div class="metric-card">
        <span>号源利用率</span>
        <strong>{{ percent(summary.capacityUsageRate) }}</strong>
        <p>{{ summary.slotBooked || 0 }} / {{ summary.slotCapacity || 0 }} 个号源</p>
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
          <div v-for="row in appointmentTrendRows" :key="row.label" class="bar-row">
            <span>{{ row.label || '未排期' }}</span>
            <div><i :style="{ width: `${barWidth(row.count, maxTrend)}%` }" /></div>
            <strong>{{ row.count }}</strong>
          </div>
          <el-empty v-if="appointmentTrendRows.length === 0" description="暂无趋势数据" />
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>套餐销量</h3>
            <p>统计非取消预约的套餐销量和销售额。</p>
          </div>
        </div>
        <el-table :data="packageSalesRows" stripe>
          <el-table-column prop="label" label="套餐" />
          <el-table-column prop="count" label="销量" width="90" />
          <el-table-column label="销售额" width="120">
            <template #default="{ row }">￥{{ Number(row.total || 0).toFixed(2) }}</template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <div class="layout-two">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>支付状态</h3>
            <p>按预约支付状态汇总订单数和应收金额。</p>
          </div>
        </div>
        <el-table :data="paymentStatusRows" stripe>
          <el-table-column label="状态" min-width="100">
            <template #default="{ row }">{{ paymentStatusText(row.label) }}</template>
          </el-table-column>
          <el-table-column prop="count" label="订单" width="90" />
          <el-table-column label="金额" width="120">
            <template #default="{ row }">￥{{ Number(row.total || 0).toFixed(2) }}</template>
          </el-table-column>
        </el-table>
        <el-empty v-if="paymentStatusRows.length === 0" description="暂无支付数据" />
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>用户增长</h3>
            <p>按注册日期统计新增用户。</p>
          </div>
        </div>
        <div class="bar-list compact-bars">
          <div v-for="row in userGrowthRows" :key="row.label" class="bar-row">
            <span>{{ row.label }}</span>
            <div><i :style="{ width: `${barWidth(row.count, maxGrowth)}%` }" /></div>
            <strong>{{ row.count }}</strong>
          </div>
          <el-empty v-if="userGrowthRows.length === 0" description="暂无增长数据" />
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useHealthData } from '../composables/useHealthData'

const { adminDashboard, loadAdminDashboard } = useHealthData()
const daysFilter = ref(14)
const rangeOptions = [
  { label: '7 天', value: 7 },
  { label: '14 天', value: 14 },
  { label: '30 天', value: 30 },
  { label: '90 天', value: 90 },
]
const summary = computed(() => adminDashboard.value.summary || {})
const dashboardRange = computed(() => adminDashboard.value.range || { appointmentStartDate: '-', appointmentEndDate: '-' })
const appointmentTrendRows = computed(() => adminDashboard.value.appointmentTrend || [])
const packageSalesRows = computed(() => adminDashboard.value.packageSales || [])
const paymentStatusRows = computed(() => adminDashboard.value.paymentStatus || [])
const userGrowthRows = computed(() => adminDashboard.value.userGrowth || [])
const maxTrend = computed(() => Math.max(1, ...appointmentTrendRows.value.map((item) => item.count || 0)))
const maxGrowth = computed(() => Math.max(1, ...userGrowthRows.value.map((item) => item.count || 0)))

function barWidth(value, max) {
  return Math.max(6, Math.round((Number(value || 0) / Number(max.value || 1)) * 100))
}

function percent(value) {
  return `${Math.round(Number(value || 0) * 100)}%`
}

function paymentStatusText(status) {
  const map = { paid: '已支付', unpaid: '待支付', refunded: '已退款' }
  return map[status] || status || '未知'
}

function refreshDashboard() {
  return loadAdminDashboard({ days: daysFilter.value })
}

onMounted(refreshDashboard)
</script>

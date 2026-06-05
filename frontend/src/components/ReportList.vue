<template>
  <div class="report-list">
    <el-empty v-if="reports.length === 0" description="暂无报告" />
    <article v-for="report in reports" :key="report.id" class="report-card">
      <div class="report-title">
        <strong>{{ report.appointment?.package?.name || '体检报告' }}</strong>
        <el-tag type="success">已归档</el-tag>
      </div>
      <p><b>客户：</b>{{ report.user?.name }}</p>
      <p><b>摘要：</b>{{ report.summary }}</p>
      <p><b>结论：</b>{{ report.conclusion }}</p>
      <p><b>建议：</b>{{ report.recommendation }}</p>
      <span>医生：{{ report.doctor?.name }} · {{ formatDate(report.createdAt) }}</span>
    </article>
  </div>
</template>

<script setup>
import { formatDate } from '../composables/useHealthData'

defineProps({
  reports: {
    type: Array,
    required: true,
  },
  userTitle: {
    type: Boolean,
    default: false,
  },
})
</script>

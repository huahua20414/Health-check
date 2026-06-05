<template>
  <div class="report-list">
    <el-empty v-if="reports.length === 0" description="暂无报告" />
    <article v-for="report in reports" :key="report.id" class="report-card">
      <div class="report-title">
        <div>
          <strong>{{ report.appointment?.package?.name || '体检报告' }}</strong>
          <span>{{ report.reportNo }}</span>
        </div>
        <div class="report-actions">
          <el-button size="small" @click="$emit('view-report', report)">查看报告</el-button>
          <el-tag type="success">已归档</el-tag>
        </div>
      </div>
      <p><b>客户：</b>{{ report.user?.name }}</p>
      <p><b>机构：</b>{{ report.appointment?.institution?.name }}</p>
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

defineEmits(['view-report'])
</script>

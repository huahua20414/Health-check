<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的报告</h3>
          <p>报告由医生生成后自动展示。</p>
        </div>
      </div>
      <ReportList :reports="reports" user-title @view-report="openReport" />
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.reports.total"
        v-model:current-page="paginations.reports.page"
        v-model:page-size="paginations.reports.pageSize"
        :page-sizes="[6, 12, 24]"
      />
    </div>

    <el-dialog v-model="reportVisible" title="体检报告详情" width="920px" class="document-dialog">
      <div class="dialog-actions">
        <el-button type="primary" @click="downloadReport">下载图片</el-button>
      </div>
      <div class="document-preview" v-html="reportHTML" />
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import ReportList from '../components/ReportList.vue'
import { downloadReportImage, reportDocumentHTML, useHealthData } from '../composables/useHealthData'

const { reports, paginations, loadReportsPage } = useHealthData()
const selectedReport = ref(null)
const reportVisible = ref(false)
const reportHTML = computed(() => (selectedReport.value ? reportDocumentHTML(selectedReport.value) : ''))

function openReport(report) {
  selectedReport.value = report
  reportVisible.value = true
}

function downloadReport() {
  if (!selectedReport.value) return
  downloadReportImage(selectedReport.value.reportNo || 'checkup-report', selectedReport.value)
}

watch(() => [paginations.reports.page, paginations.reports.pageSize], () => loadReportsPage())
onMounted(() => loadReportsPage())
</script>

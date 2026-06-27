<template>
  <section class="view management-stack">
    <div class="panel form-panel">
      <div class="panel-head">
        <div>
          <h3>报告录入</h3>
          <p>只能为已体检预约生成或更新报告。</p>
        </div>
      </div>
      <el-form label-position="top" class="form-grid spacious-form">
        <el-form-item label="关联预约">
          <el-select v-model="reportForm.appointmentId" placeholder="选择已体检预约">
            <el-option
              v-for="item in reportableAppointments"
              :key="item.id"
              :label="`#${item.id} ${item.user?.name || ''} ${item.package?.name || ''}`"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="检查摘要"><el-input v-model="reportForm.summary" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="体检结论"><el-input v-model="reportForm.conclusion" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="健康建议"><el-input v-model="reportForm.recommendation" type="textarea" :rows="3" /></el-form-item>
        <el-button type="success" :disabled="!reportForm.appointmentId || !can('report:create')" :loading="loading.report" @click="submit">生成/更新报告</el-button>
      </el-form>
    </div>

    <div class="panel wide-table-panel">
      <div class="panel-head">
        <div>
          <h3>报告列表</h3>
          <p>沉淀客户健康档案和医生建议。</p>
        </div>
      </div>
      <ReportList :reports="reports" @view-report="openReport" />
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
import { useDebouncedFn } from '../composables/useDebouncedFn'
import { downloadReportImage, reportDocumentHTML, useHealthData } from '../composables/useHealthData'

const { appointments, reports, reportForm, loading, can, createReport, paginations, loadReportsPage } = useHealthData()
const reportableAppointments = computed(() => appointments.value.filter((item) => item.status === 'checked' || item.status === 'reported'))
const submit = useDebouncedFn(createReport, 400)
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

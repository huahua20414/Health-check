<template>
  <section class="view">
    <div class="layout-two">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>报告录入</h3>
            <p>只能为已体检预约生成或更新报告。</p>
          </div>
        </div>
        <el-form label-position="top" class="form-grid">
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
          <el-button type="success" :disabled="!reportForm.appointmentId" :loading="loading.report" @click="submit">生成/更新报告</el-button>
        </el-form>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>报告列表</h3>
            <p>沉淀客户健康档案和医生建议。</p>
          </div>
        </div>
        <ReportList :reports="reports" />
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import ReportList from '../components/ReportList.vue'
import { useDebouncedFn } from '../composables/useDebouncedFn'
import { useHealthData } from '../composables/useHealthData'

const { appointments, reports, reportForm, loading, createReport } = useHealthData()
const reportableAppointments = computed(() => appointments.value.filter((item) => item.status === 'checked' || item.status === 'reported'))
const submit = useDebouncedFn(createReport, 400)
</script>

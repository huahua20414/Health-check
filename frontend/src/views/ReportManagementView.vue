<template>
  <section class="view">
    <div :class="isDoctor ? 'layout-two' : 'single-layout'">
      <div v-if="isDoctor" class="panel">
        <div class="panel-head">
          <div>
            <h3>报告录入</h3>
            <p>医生根据体检数据生成用户报告</p>
          </div>
        </div>
        <el-form label-position="top" class="form-grid">
          <el-form-item label="关联预约">
            <el-select v-model="reportForm.appointmentId" placeholder="选择预约">
              <el-option
                v-for="item in appointments"
                :key="item.id"
                :label="`#${item.id} ${item.user?.name || ''} ${item.package?.name || ''}`"
                :value="item.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="检查摘要">
            <el-input v-model="reportForm.summary" type="textarea" :rows="4" />
          </el-form-item>
          <el-form-item label="体检结论">
            <el-input v-model="reportForm.conclusion" type="textarea" :rows="3" />
          </el-form-item>
          <el-form-item label="健康建议">
            <el-input v-model="reportForm.recommendation" type="textarea" :rows="3" />
          </el-form-item>
          <el-button type="success" :disabled="!isDoctor" :loading="loading.report" @click="createReport">生成/更新报告</el-button>
        </el-form>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>{{ isUser ? '我的报告' : '报告列表' }}</h3>
            <p>沉淀客户健康档案和医生建议</p>
          </div>
        </div>
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
      </div>
    </div>
  </section>
</template>

<script setup>
import { formatDate, useHealthData } from '../composables/useHealthData'

const { appointments, reports, reportForm, isDoctor, isUser, loading, createReport } = useHealthData()
</script>

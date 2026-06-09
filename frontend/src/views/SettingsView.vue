<template>
  <section class="view">
    <div class="settings-grid">
      <div class="panel setting-card">
        <el-icon><Connection /></el-icon>
        <h3>邮件通知</h3>
        <p>预约订单、候补递补、体检报告会按用户通知设置发送邮件。</p>
      </div>
      <div class="panel setting-card">
        <el-icon><DataAnalysis /></el-icon>
        <h3>数据初始化</h3>
        <p>管理员可通过初始化命令重建机构、医生、套餐、排班、预约和报告数据。</p>
      </div>
      <div class="panel setting-card">
        <el-icon><Lock /></el-icon>
        <h3>权限说明</h3>
        <p>系统已接入 JWT、Redis Session、角色菜单和后端权限校验。</p>
      </div>
    </div>
    <div class="panel" v-if="isAdmin">
      <div class="panel-head">
        <div>
          <h3>邮件发送记录</h3>
          <p>预约成功、候补递补、报告生成都会记录邮件发送结果。</p>
        </div>
      </div>
      <el-table :data="mailLogs" stripe>
        <el-table-column prop="to" label="收件人" width="190" />
        <el-table-column prop="subject" label="主题" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'sent' ? 'success' : 'danger'">{{ row.status === 'sent' ? '已发送' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error" label="错误信息" />
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.mailLogs.total"
        v-model:current-page="paginations.mailLogs.page"
        v-model:page-size="paginations.mailLogs.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>
  </section>
</template>

<script setup>
import { onMounted, watch } from 'vue'
import { Connection, DataAnalysis, Lock } from '@element-plus/icons-vue'
import { useHealthData } from '../composables/useHealthData'

const { isAdmin, mailLogs, paginations, loadMailLogsPage } = useHealthData()

watch(() => [paginations.mailLogs.page, paginations.mailLogs.pageSize], () => {
  if (isAdmin.value) loadMailLogsPage()
})

onMounted(() => {
  if (isAdmin.value) loadMailLogsPage()
})
</script>

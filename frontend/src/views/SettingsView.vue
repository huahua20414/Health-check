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
          <h3>数据导入导出</h3>
          <p>支持套餐 CSV 导出和按套餐名称幂等导入，便于运营人员批量维护基础服务数据。</p>
        </div>
      </div>
      <div class="data-exchange-actions">
        <el-button type="primary" :loading="loading.exportPackages" @click="exportPackages">导出套餐 CSV</el-button>
        <el-upload accept=".csv" :auto-upload="false" :show-file-list="false" :on-change="handlePackageImport">
          <el-button :loading="loading.importPackages">导入套餐 CSV</el-button>
        </el-upload>
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
    <div class="panel" v-if="isAdmin">
      <div class="panel-head">
        <div>
          <h3>登录日志</h3>
          <p>记录成功、失败和被拦截的登录行为，用于排查账号和安全问题。</p>
        </div>
      </div>
      <el-table :data="loginLogs" stripe>
        <el-table-column prop="email" label="邮箱" min-width="190" />
        <el-table-column prop="role" label="角色" width="100" />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : row.status === 'blocked' ? 'warning' : 'danger'">{{ loginStatusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" min-width="150" />
        <el-table-column prop="createdAt" label="时间" width="180" />
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.loginLogs.total"
        v-model:current-page="paginations.loginLogs.page"
        v-model:page-size="paginations.loginLogs.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>
    <div class="panel" v-if="isAdmin">
      <div class="panel-head">
        <div>
          <h3>操作日志</h3>
          <p>记录管理员对套餐、排班、公告、人员等核心资源的变更。</p>
        </div>
      </div>
      <el-table :data="operationLogs" stripe>
        <el-table-column prop="userName" label="操作人" width="120" />
        <el-table-column prop="action" label="动作" width="120" />
        <el-table-column prop="resource" label="资源" width="130" />
        <el-table-column prop="resourceId" label="资源ID" width="90" />
        <el-table-column prop="detail" label="详情" min-width="180" />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'">{{ row.status === 'success' ? '成功' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="时间" width="180" />
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.operationLogs.total"
        v-model:current-page="paginations.operationLogs.page"
        v-model:page-size="paginations.operationLogs.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>
  </section>
</template>

<script setup>
import { onMounted, watch } from 'vue'
import { Connection, DataAnalysis, Lock } from '@element-plus/icons-vue'
import { useHealthData } from '../composables/useHealthData'

const {
  isAdmin,
  mailLogs,
  loginLogs,
  operationLogs,
  paginations,
  loading,
  loadMailLogsPage,
  loadLoginLogsPage,
  loadOperationLogsPage,
  exportPackages,
  importPackages,
} = useHealthData()

watch(() => [paginations.mailLogs.page, paginations.mailLogs.pageSize], () => {
  if (isAdmin.value) loadMailLogsPage()
})
watch(() => [paginations.loginLogs.page, paginations.loginLogs.pageSize], () => {
  if (isAdmin.value) loadLoginLogsPage()
})
watch(() => [paginations.operationLogs.page, paginations.operationLogs.pageSize], () => {
  if (isAdmin.value) loadOperationLogsPage()
})

function loginStatusText(status) {
  return { success: '成功', failed: '失败', blocked: '拦截' }[status] || status
}

async function handlePackageImport(file) {
  await importPackages(file.raw)
}

onMounted(() => {
  if (!isAdmin.value) return
  loadMailLogsPage()
  loadLoginLogsPage()
  loadOperationLogsPage()
})
</script>

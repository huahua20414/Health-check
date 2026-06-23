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
    <div class="panel" v-if="can('admin:system:manage')">
      <div class="panel-head">
        <div>
          <h3>业务参数配置</h3>
          <p>配置预约提醒、改期限制、通知开关和客服入口等运行参数。</p>
        </div>
      </div>
      <el-table :data="systemSettings" stripe>
        <el-table-column label="分组" width="120">
          <template #default="{ row }">{{ settingGroupText(row.group) }}</template>
        </el-table-column>
        <el-table-column prop="label" label="配置项" min-width="170" />
        <el-table-column prop="description" label="说明" min-width="220" />
        <el-table-column label="配置值" min-width="220">
          <template #default="{ row }">
            <el-switch v-if="row.valueType === 'boolean'" v-model="row.value" active-value="true" inactive-value="false" />
            <el-input-number v-else-if="row.valueType === 'number'" v-model="row.value" :min="0" />
            <el-input v-else v-model="row.value" />
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-select v-model="row.status" size="small">
              <el-option label="启用" value="active" />
              <el-option label="停用" value="disabled" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button size="small" type="primary" :loading="loading.systemSetting" @click="updateSystemSetting(row)">保存</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <div class="panel" v-if="can('admin:data:exchange')">
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
    <div class="panel" v-if="can('admin:system:manage')">
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
    <div class="panel" v-if="can('admin:system:manage')">
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
    <div class="panel" v-if="can('admin:system:manage')">
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
    <div class="panel" v-if="can('admin:permission:manage')">
      <div class="panel-head">
        <div>
          <h3>角色权限管理</h3>
          <p>维护角色可见菜单和关键按钮能力，后端仍保留角色级接口保护。</p>
        </div>
      </div>
      <el-table :data="rolePermissions" stripe>
        <el-table-column label="角色" width="100">
          <template #default="{ row }">{{ roleText(row.role) }}</template>
        </el-table-column>
        <el-table-column prop="permission" label="权限点" min-width="190" />
        <el-table-column prop="description" label="说明" min-width="220" />
        <el-table-column label="启用" width="120">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" :loading="loading.permission" @change="updateRolePermission(row)" />
          </template>
        </el-table-column>
      </el-table>
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
  rolePermissions,
  systemSettings,
  paginations,
  loading,
  can,
  loadMailLogsPage,
  loadLoginLogsPage,
  loadOperationLogsPage,
  loadRolePermissions,
  loadSystemSettings,
  exportPackages,
  importPackages,
  updateRolePermission,
  updateSystemSetting,
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

function roleText(role) {
  return { user: '用户', doctor: '医生', admin: '管理员' }[role] || role
}

function settingGroupText(group) {
  return { appointment: '预约', notification: '通知', security: '安全', service: '服务', system: '系统' }[group] || group
}

async function handlePackageImport(file) {
  await importPackages(file.raw)
}

onMounted(() => {
  if (!isAdmin.value) return
  loadMailLogsPage()
  loadLoginLogsPage()
  loadOperationLogsPage()
  loadRolePermissions()
  loadSystemSettings()
})
</script>

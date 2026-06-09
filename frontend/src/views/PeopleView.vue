<template>
  <section class="view">
    <div class="panel wide-table-panel">
      <div class="panel-head">
        <div>
          <h3>人员档案</h3>
          <p>展示客户与医生基础档案，便于后续扩展权限管理</p>
        </div>
      </div>
      <el-table :data="tableRows" stripe>
        <el-table-column prop="id" label="编号" width="80" />
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="email" label="邮箱" min-width="190" />
        <el-table-column label="科室" width="170">
          <template #default="{ row }">
            <span>{{ row.department || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="负责体检" min-width="260">
          <template #default="{ row }">
            <span>{{ row.specialties || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="roleTagType(row.role)">{{ roleLabel(row.role) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><StatusTag :status="row.status || 'active'" /></template>
        </el-table-column>
        <el-table-column label="来源">
          <template #default="{ row }">{{ row.source }}</template>
        </el-table-column>
        <el-table-column v-if="isAdmin" label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button v-if="row.status !== 'disabled'" size="small" type="danger" plain :loading="loading.status" @click="changeStatus(row, 'disabled')">
                停用
              </el-button>
              <el-tag v-else type="info" effect="plain">已停用</el-tag>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-if="isAdmin"
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.users.total"
        v-model:current-page="paginations.users.page"
        v-model:page-size="paginations.users.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>
  </section>
</template>

<script setup>
import { onMounted, watch } from 'vue'
import { useHealthData } from '../composables/useHealthData'
import StatusTag from '../components/StatusTag.vue'

const { peopleRows: tableRows, isAdmin, loading, updateUserStatus, paginations, loadUsersPage } = useHealthData()

function loadPage() {
  if (isAdmin.value) return loadUsersPage()
  return Promise.resolve()
}

function roleLabel(role) {
  return { admin: '管理员', doctor: '医生', user: '用户' }[role] || role || '-'
}

function roleTagType(role) {
  return { admin: 'danger', doctor: 'warning', user: 'success' }[role] || 'info'
}

async function changeStatus(row, status) {
  await updateUserStatus(row, status)
  await loadPage()
}

watch(() => [paginations.users.page, paginations.users.pageSize], () => loadPage())
onMounted(() => loadPage())
</script>

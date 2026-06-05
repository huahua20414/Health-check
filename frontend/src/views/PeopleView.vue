<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>人员档案</h3>
          <p>展示客户与医生基础档案，便于后续扩展权限管理</p>
        </div>
      </div>
      <el-table :data="peopleRows" stripe>
        <el-table-column prop="id" label="编号" width="80" />
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="email" label="邮箱" min-width="190" />
        <el-table-column label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="row.role === 'doctor' ? 'warning' : 'success'">{{ row.role === 'doctor' ? '医生' : '用户' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><StatusTag :status="row.status || 'active'" /></template>
        </el-table-column>
        <el-table-column label="来源">
          <template #default="{ row }">{{ row.source }}</template>
        </el-table-column>
        <el-table-column v-if="isAdmin" label="操作" width="190">
          <template #default="{ row }">
            <el-button v-if="row.status === 'pending'" size="small" type="success" :loading="loading.status" @click="updateUserStatus(row, 'active')">
              通过审核
            </el-button>
            <el-button v-if="row.status !== 'disabled'" size="small" type="danger" plain :loading="loading.status" @click="updateUserStatus(row, 'disabled')">
              停用
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup>
import { useHealthData } from '../composables/useHealthData'
import StatusTag from '../components/StatusTag.vue'

const { peopleRows, isAdmin, loading, updateUserStatus } = useHealthData()
</script>

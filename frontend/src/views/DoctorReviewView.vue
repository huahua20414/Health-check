<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>医生审核</h3>
          <p>医生注册后默认待审核，管理员审核通过后才可登录业务端。</p>
        </div>
      </div>
      <el-table :data="doctorRows" stripe>
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="phone" label="手机号" width="150" />
        <el-table-column prop="employeeNo" label="工号" width="120" />
        <el-table-column prop="department" label="科室" />
        <el-table-column prop="title" label="职称" width="120" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="操作" width="190">
          <template #default="{ row }">
            <el-button v-if="row.status === 'pending'" size="small" type="success" :loading="loading.status" @click="updateUserStatus(row, 'active')">通过</el-button>
            <el-button v-if="row.status !== 'disabled'" size="small" type="danger" plain :loading="loading.status" @click="updateUserStatus(row, 'disabled')">停用</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import StatusTag from '../components/StatusTag.vue'
import { useHealthData } from '../composables/useHealthData'

const { users, loading, updateUserStatus } = useHealthData()
const doctorRows = computed(() => users.value.filter((item) => item.role === 'doctor'))
</script>

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
        <el-table-column prop="email" label="邮箱" min-width="190" />
        <el-table-column prop="employeeNo" label="工号" width="120" />
        <el-table-column label="科室" width="160">
          <template #default="{ row }">
            <el-select v-model="row.department" size="small" placeholder="选择科室">
              <el-option v-for="department in doctorDepartments" :key="department" :label="department" :value="department" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="职称" width="120" />
        <el-table-column label="负责体检" min-width="260">
          <template #default="{ row }">
            <el-select v-model="row.specialtyValues" multiple collapse-tags collapse-tags-tooltip size="small" placeholder="选择负责分类">
              <el-option v-for="specialty in specialtyOptions" :key="specialty" :label="specialty" :value="specialty" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="操作" width="270">
          <template #default="{ row }">
            <el-button size="small" :loading="loading.doctorProfile" @click="saveDoctorProfile(row)">保存资料</el-button>
            <el-button v-if="row.status === 'pending'" size="small" type="success" :loading="loading.status" @click="changeStatus(row, 'active')">通过</el-button>
            <el-button v-if="row.status !== 'disabled'" size="small" type="danger" plain :loading="loading.status" @click="changeStatus(row, 'disabled')">停用</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
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
import { computed, onMounted, watch } from 'vue'
import StatusTag from '../components/StatusTag.vue'
import { doctorDepartments, specialtyOptions, useHealthData } from '../composables/useHealthData'

const { users, loading, updateUserStatus, updateDoctorProfile, paginations, loadUsersPage } = useHealthData()
const doctorRows = computed(() => users.value.map((item) => ({
  ...item,
  specialtyValues: splitSpecialties(item.specialties),
})))

function loadPage() {
  return loadUsersPage({ role: 'doctor' })
}

function splitSpecialties(value) {
  return String(value || '').split(',').map((item) => item.trim()).filter(Boolean)
}

async function saveDoctorProfile(row) {
  await updateDoctorProfile(row, {
    department: row.department,
    title: row.title,
    specialties: row.specialtyValues,
  })
  await loadPage()
}

async function changeStatus(row, status) {
  await updateUserStatus(row, status)
  await loadPage()
}

watch(() => [paginations.users.page, paginations.users.pageSize], () => loadPage())
onMounted(() => loadPage())
</script>

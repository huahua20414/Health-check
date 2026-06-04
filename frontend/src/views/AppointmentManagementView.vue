<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>预约管理</h3>
          <p>医生和运营人员处理客户体检预约</p>
        </div>
        <div class="filter-bar">
          <el-select v-model="statusFilter" placeholder="状态筛选" clearable>
            <el-option label="已预约" value="booked" />
            <el-option label="已体检" value="checked" />
            <el-option label="已出报告" value="reported" />
          </el-select>
          <el-input v-model="keyword" placeholder="搜索客户/套餐" />
        </div>
      </div>
      <AppointmentTable :rows="filteredAppointments" :is-doctor="isDoctor" @mark-done="handleMarkDone" />
    </div>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppointmentTable from '../components/AppointmentTable.vue'
import { useHealthData } from '../composables/useHealthData'

const router = useRouter()
const statusFilter = ref('')
const keyword = ref('')
const { appointments, isDoctor, markDone } = useHealthData()

const filteredAppointments = computed(() => {
  return appointments.value.filter((item) => {
    const matchesStatus = !statusFilter.value || item.status === statusFilter.value
    const text = `${item.user?.name || ''}${item.package?.name || ''}${item.date || ''}`
    const matchesKeyword = !keyword.value || text.includes(keyword.value)
    return matchesStatus && matchesKeyword
  })
})

async function handleMarkDone(row) {
  await markDone(row)
  router.push('/reports')
}
</script>

<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>预约处理</h3>
          <p>医生只处理客户预约状态，不在此页面录入报告。</p>
        </div>
        <div class="filter-bar">
          <el-select v-model="statusFilter" placeholder="状态筛选" clearable>
            <el-option label="已预约" value="booked" />
            <el-option label="已体检" value="checked" />
            <el-option label="已出报告" value="reported" />
            <el-option label="已取消" value="canceled" />
          </el-select>
          <el-input v-model="keyword" placeholder="搜索客户/套餐" />
        </div>
      </div>
      <AppointmentTable :rows="filteredAppointments" :is-doctor="isDoctor" :loading="loading.status" @mark-done="handleMarkDone" />
    </div>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppointmentTable from '../components/AppointmentTable.vue'
import { useDebouncedRef } from '../composables/useDebouncedRef'
import { useHealthData } from '../composables/useHealthData'

const router = useRouter()
const statusFilter = ref('')
const keyword = ref('')
const debouncedKeyword = useDebouncedRef(keyword, 350)
const { appointments, isDoctor, loading, markDone } = useHealthData()

const filteredAppointments = computed(() => {
  return appointments.value.filter((item) => {
    const matchesStatus = !statusFilter.value || item.status === statusFilter.value
    const text = `${item.user?.name || ''}${item.package?.name || ''}${item.date || ''}`
    return matchesStatus && (!debouncedKeyword.value || text.includes(debouncedKeyword.value))
  })
})

async function handleMarkDone(row) {
  await markDone(row)
  router.push('/reports')
}
</script>

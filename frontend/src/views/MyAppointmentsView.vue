<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的预约</h3>
          <p>只能查看和取消自己的未体检预约。</p>
        </div>
      </div>
      <AppointmentTable :rows="myAppointments" :is-doctor="false" :can-cancel="true" :loading="loading.status" @cancel="cancelAppointment" />
    </div>
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的候补</h3>
          <p>当同日期同时间段有人取消时，系统会按候补提交时间自动递补。</p>
        </div>
      </div>
      <el-table :data="waitlist" stripe>
        <el-table-column label="套餐">
          <template #default="{ row }">{{ row.package?.name }}</template>
        </el-table-column>
        <el-table-column prop="date" label="日期" width="120" />
        <el-table-column prop="period" label="时段" width="90" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup>
import AppointmentTable from '../components/AppointmentTable.vue'
import StatusTag from '../components/StatusTag.vue'
import { useHealthData } from '../composables/useHealthData'

const { myAppointments, waitlist, loading, cancelAppointment } = useHealthData()
</script>

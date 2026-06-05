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
  </section>
</template>

<script setup>
import AppointmentTable from '../components/AppointmentTable.vue'
import { useHealthData } from '../composables/useHealthData'

const { myAppointments, loading, cancelAppointment } = useHealthData()
</script>

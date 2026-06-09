<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的预约</h3>
          <p>只能查看和取消自己的未体检预约。</p>
        </div>
      </div>
      <AppointmentTable :rows="myAppointments" :is-doctor="false" :can-cancel="true" :loading="loading.status" @cancel="cancelAppointment" @view-order="openOrder" />
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.appointments.total"
        v-model:current-page="paginations.appointments.page"
        v-model:page-size="paginations.appointments.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的候补</h3>
          <p>当同日期同时间段有人取消时，系统会按候补提交时间自动递补。</p>
        </div>
      </div>
      <el-table :data="waitlistRows" stripe>
        <el-table-column prop="position" label="序号" width="80" />
        <el-table-column label="套餐">
          <template #default="{ row }">{{ row.package?.name }}</template>
        </el-table-column>
        <el-table-column prop="date" label="日期" width="120" />
        <el-table-column label="机构">
          <template #default="{ row }">{{ row.institution?.name }}</template>
        </el-table-column>
        <el-table-column prop="category" label="分类" width="110" />
        <el-table-column prop="appointmentType" label="类型" width="100" />
        <el-table-column label="时段" width="150">
          <template #default="{ row }">
            {{ row.startTime ? `${row.startTime}-${row.endTime}` : row.period }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.waitlist.total"
        v-model:current-page="paginations.waitlist.page"
        v-model:page-size="paginations.waitlist.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>

    <el-dialog v-model="orderVisible" title="体检预约订单" width="920px" class="document-dialog">
      <div class="dialog-actions">
        <el-button type="primary" @click="downloadOrder">下载 HTML</el-button>
      </div>
      <div class="document-preview" v-html="orderHTML" />
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import AppointmentTable from '../components/AppointmentTable.vue'
import StatusTag from '../components/StatusTag.vue'
import { appointmentDocumentHTML, downloadHTML, useHealthData } from '../composables/useHealthData'

const { myAppointments, waitlist, loading, cancelAppointment, paginations, loadAppointmentsPage, loadWaitlistPage } = useHealthData()
const selectedOrder = ref(null)
const orderVisible = ref(false)
const orderHTML = computed(() => (selectedOrder.value ? appointmentDocumentHTML(selectedOrder.value) : ''))
const waitlistRows = computed(() => waitlist.value.map((item, index) => ({
  ...item,
  position: (paginations.waitlist.page - 1) * paginations.waitlist.pageSize + index + 1,
})))

function openOrder(row) {
  selectedOrder.value = row
  orderVisible.value = true
}

function downloadOrder() {
  if (!selectedOrder.value) return
  downloadHTML(`${selectedOrder.value.orderNo || 'appointment-order'}.html`, orderHTML.value)
}

watch(() => [paginations.appointments.page, paginations.appointments.pageSize], () => loadAppointmentsPage())
watch(() => [paginations.waitlist.page, paginations.waitlist.pageSize], () => loadWaitlistPage())
onMounted(() => {
  loadAppointmentsPage()
  loadWaitlistPage()
})
</script>

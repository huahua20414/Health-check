<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的预约</h3>
          <p>只能查看和取消自己的未体检预约。</p>
        </div>
      </div>
      <AppointmentTable
        :rows="myAppointments"
        :is-doctor="false"
        :can-cancel="can('appointment:cancel')"
        :can-reschedule="can('appointment:reschedule')"
        :can-review="can('review:create')"
        :can-pay="can('appointment:pay')"
        :can-invoice="can('appointment:invoice')"
        :loading="loading.status || loading.appointment"
        @cancel="cancelAppointment"
        @reschedule="openReschedule"
        @review="openReview"
        @view-order="openOrder"
        @pay="(row) => updateAppointmentPayment(row, 'paid')"
        @unpay="(row) => updateAppointmentPayment(row, 'unpaid')"
        @invoice="openInvoice"
      />
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

    <el-dialog v-model="rescheduleVisible" title="预约改期" width="860px" class="choice-dialog">
      <el-form label-position="top">
        <el-form-item label="新日期">
          <el-select v-model="rescheduleForm.date" filterable placeholder="选择有库存日期" @change="rescheduleForm.slotId = null">
            <el-option v-for="date in availableDates" :key="date" :label="date" :value="date" />
          </el-select>
        </el-form-item>
        <el-form-item label="新号源">
          <el-select v-model="rescheduleForm.slotId" filterable placeholder="选择新号源" @change="syncSelectedSlot">
            <el-option
              v-for="slot in availableSlots"
              :key="slot.id"
              :label="`${slot.date} ${slot.startTime}-${slot.endTime} ${slot.doctor?.name || ''}`"
              :value="slot.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="改期备注">
          <el-input v-model="rescheduleForm.note" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rescheduleVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading.appointment" :disabled="!rescheduleForm.slotId || !can('appointment:reschedule')" @click="submitReschedule">确认改期</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="invoiceVisible" title="发票信息" width="640px">
      <div v-if="activeInvoiceAppointment" class="invoice-summary">
        <div>
          <span>订单号</span>
          <strong>{{ activeInvoiceAppointment.orderNo }}</strong>
        </div>
        <div>
          <span>开票金额</span>
          <strong>{{ moneyText(appointmentPayableAmount(activeInvoiceAppointment)) }}</strong>
        </div>
        <div>
          <span>支付状态</span>
          <strong>{{ paymentStatusText(activeInvoiceAppointment.paymentStatus) }}</strong>
        </div>
      </div>
      <el-form label-position="top">
        <el-form-item label="发票抬头">
          <el-input v-model="invoiceForm.invoiceTitle" maxlength="128" show-word-limit placeholder="个人或企业名称" />
        </el-form-item>
        <el-form-item label="纳税人识别号">
          <el-input v-model="invoiceForm.invoiceTaxNo" maxlength="64" show-word-limit placeholder="企业发票可填写" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="invoiceVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading.appointment" :disabled="!invoiceForm.appointmentId || !can('appointment:invoice')" @click="submitInvoice">保存发票</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="reviewVisible" title="评价体检服务" width="640px">
      <el-form label-position="top">
        <el-form-item label="评分">
          <el-rate v-model="reviewForm.rating" />
        </el-form-item>
        <el-form-item label="评价内容">
          <el-input v-model="reviewForm.content" type="textarea" :rows="4" placeholder="请描述服务体验、环境或报告质量" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reviewVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading.review" :disabled="!reviewForm.appointmentId || !can('review:create')" @click="submitReview">提交评价</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import AppointmentTable from '../components/AppointmentTable.vue'
import StatusTag from '../components/StatusTag.vue'
import { appointmentDocumentHTML, appointmentPayableAmount, downloadHTML, moneyText, paymentStatusText, useHealthData } from '../composables/useHealthData'

const { myAppointments, waitlist, slots, rescheduleForm, invoiceForm, reviewForm, loading, can, cancelAppointment, editReschedule, rescheduleAppointment, updateAppointmentPayment, editInvoice, saveInvoice, createReview, paginations, loadAppointmentsPage, loadWaitlistPage } = useHealthData()
const selectedOrder = ref(null)
const orderVisible = ref(false)
const rescheduleVisible = ref(false)
const invoiceVisible = ref(false)
const reviewVisible = ref(false)
const orderHTML = computed(() => (selectedOrder.value ? appointmentDocumentHTML(selectedOrder.value) : ''))
const activeRescheduleAppointment = computed(() => myAppointments.value.find((item) => item.id === rescheduleForm.appointmentId))
const activeInvoiceAppointment = computed(() => myAppointments.value.find((item) => item.id === invoiceForm.appointmentId))
const compatibleSlots = computed(() => slots.value.filter((slot) => (
  slot.institutionId === rescheduleForm.institutionId &&
  slot.category === activeRescheduleAppointment.value?.category &&
  slot.status === 'available' &&
  slot.capacity > slot.bookedCount
)))
const availableDates = computed(() => Array.from(new Set(compatibleSlots.value.map((slot) => slot.date))).sort())
const availableSlots = computed(() => compatibleSlots.value.filter((slot) => !rescheduleForm.date || slot.date === rescheduleForm.date))
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

function openReschedule(row) {
  editReschedule(row)
  rescheduleVisible.value = true
}

function syncSelectedSlot(slotId) {
  const slot = slots.value.find((item) => item.id === slotId)
  if (!slot) return
  rescheduleForm.date = slot.date
  rescheduleForm.period = slot.period
}

async function submitReschedule() {
  await rescheduleAppointment()
  rescheduleVisible.value = false
}

function openInvoice(row) {
  editInvoice(row)
  invoiceVisible.value = true
}

async function submitInvoice() {
  await saveInvoice()
  invoiceVisible.value = false
}

function openReview(row) {
  reviewForm.appointmentId = row.id
  reviewForm.rating = 5
  reviewForm.content = ''
  reviewVisible.value = true
}

async function submitReview() {
  await createReview()
  reviewVisible.value = false
}

watch(() => [paginations.appointments.page, paginations.appointments.pageSize], () => loadAppointmentsPage())
watch(() => [paginations.waitlist.page, paginations.waitlist.pageSize], () => loadWaitlistPage())
onMounted(() => {
  loadAppointmentsPage()
  loadWaitlistPage()
})
</script>

<style scoped>
.invoice-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 18px;
}

.invoice-summary div {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px 12px;
}

.invoice-summary span {
  display: block;
  color: #6b7280;
  font-size: 12px;
  margin-bottom: 4px;
}

.invoice-summary strong {
  display: block;
  color: #111827;
  font-size: 14px;
  overflow-wrap: anywhere;
}

@media (max-width: 720px) {
  .invoice-summary {
    grid-template-columns: 1fr;
  }
}
</style>

<template>
  <section class="view booking-screen">
    <div class="booking-console">
      <div class="panel booking-picker">
        <div class="panel-head">
          <div>
            <h3>预约体检</h3>
            <p>按流程选择预约类型、机构、套餐、日期和具体医生号源。</p>
          </div>
          <el-button :loading="loading.load" @click="loadAll">刷新库存</el-button>
        </div>

        <div class="selection-board">
          <button type="button" class="selection-tile" @click="dialogs.type = true">
            <span>预约类型</span>
            <strong>{{ appointmentForm.appointmentType || '请选择' }}</strong>
          </button>
          <button type="button" class="selection-tile" @click="dialogs.institution = true">
            <span>体检机构</span>
            <strong>{{ selectedInstitution?.name || '请选择' }}</strong>
          </button>
          <button type="button" class="selection-tile" @click="dialogs.package = true">
            <span>体检套餐</span>
            <strong>{{ selectedPackage?.name || '请选择' }}</strong>
          </button>
          <button type="button" class="selection-tile" :disabled="!appointmentForm.institutionId" @click="dialogs.date = true">
            <span>预约日期</span>
            <strong>{{ appointmentForm.date || '请选择' }}</strong>
          </button>
          <button type="button" class="selection-tile wide" :disabled="!appointmentForm.date" @click="dialogs.slot = true">
            <span>医生号源</span>
            <strong>{{ selectedSlotText || '请选择具体半小时号源' }}</strong>
          </button>
        </div>
      </div>

      <aside class="panel booking-summary">
        <div class="panel-head">
          <div>
            <h3>预约确认</h3>
            <p>确认后生成正式预约订单。</p>
          </div>
        </div>
        <dl class="summary-list compact">
          <div><dt>类型</dt><dd>{{ appointmentForm.appointmentType || '-' }}</dd></div>
          <div><dt>机构</dt><dd>{{ selectedInstitution?.name || '-' }}</dd></div>
          <div><dt>套餐</dt><dd>{{ selectedPackage?.name || '-' }}</dd></div>
          <div><dt>日期</dt><dd>{{ appointmentForm.date || '-' }}</dd></div>
          <div><dt>时间</dt><dd>{{ selectedSlot ? `${selectedSlot.startTime}-${selectedSlot.endTime}` : '-' }}</dd></div>
          <div><dt>医生</dt><dd>{{ selectedSlot?.doctor?.name || '-' }}</dd></div>
        </dl>
        <el-form label-position="top">
          <el-form-item label="备注说明">
            <el-input v-model="appointmentForm.note" type="textarea" :rows="3" placeholder="例如既往病史、特殊检查需求" />
          </el-form-item>
          <el-button type="primary" class="summary-submit" :disabled="!canSubmit" :loading="loading.appointment" @click="submit">
            提交预约
          </el-button>
        </el-form>
      </aside>
    </div>

    <el-dialog v-model="dialogs.type" title="选择预约类型" width="720px" class="choice-dialog">
      <div class="segmented-grid">
        <button
          v-for="type in appointmentTypes"
          :key="type"
          type="button"
          class="segment-item"
          :class="{ selected: appointmentForm.appointmentType === type }"
          @click="selectType(type)"
        >
          {{ type }}
        </button>
      </div>
    </el-dialog>

    <el-dialog v-model="dialogs.institution" title="选择体检机构" width="860px" class="choice-dialog">
      <div class="institution-grid">
        <button
          v-for="institution in activeInstitutions"
          :key="institution.id"
          type="button"
          class="institution-card"
          :class="{ selected: appointmentForm.institutionId === institution.id }"
          @click="selectInstitution(institution)"
        >
          <strong>{{ institution.name }}</strong>
          <span>{{ institution.address }}</span>
          <small>{{ institution.openHours }}</small>
        </button>
      </div>
    </el-dialog>

    <el-dialog v-model="dialogs.package" title="选择体检套餐" width="880px" class="choice-dialog">
      <div class="package-list dialog-list">
        <button
          v-for="pkg in activePackages"
          :key="pkg.id"
          class="package-row"
          :class="{ selected: appointmentForm.packageId === pkg.id }"
          type="button"
          @click="selectPackageAndClose(pkg)"
        >
          <div>
            <h4>{{ pkg.name }}</h4>
            <p>{{ pkg.description }}</p>
            <span>{{ pkg.items }}</span>
          </div>
          <strong>￥{{ pkg.price }}</strong>
        </button>
      </div>
    </el-dialog>

    <el-dialog v-model="dialogs.date" title="选择预约日期" width="760px" class="choice-dialog">
      <div class="date-stock-grid">
        <button
          v-for="item in dateStocks"
          :key="item.date"
          type="button"
          class="date-stock"
          :class="{ selected: appointmentForm.date === item.date, disabled: item.available === 0 }"
          :disabled="item.available === 0"
          @click="selectDate(item.date)"
        >
          <strong>{{ item.date }}</strong>
          <span>{{ item.available > 0 ? `剩余 ${item.available}` : '无库存' }}</span>
        </button>
      </div>
    </el-dialog>

    <el-dialog v-model="dialogs.slot" title="选择医生排班" width="980px" class="choice-dialog schedule-dialog">
      <div v-if="appointmentForm.date" class="schedule-board dialog-schedule">
        <div v-for="group in groupedSlots" :key="group.time" class="schedule-row" :class="{ full: group.available === 0 }">
          <div class="time-cell">
            <strong>{{ group.time }}</strong>
            <span>{{ group.available > 0 ? `可约 ${group.available}` : '已满' }}</span>
          </div>
          <div class="doctor-slots">
            <button
              v-for="slot in group.slots"
              :key="slot.id"
              type="button"
              class="doctor-slot"
              :class="{ selected: appointmentForm.slotId === slot.id, disabled: !hasStock(slot) }"
              :disabled="!hasStock(slot)"
              @click="selectSlot(slot)"
            >
              <strong>{{ slot.doctor?.name }}</strong>
              <span>{{ slot.doctor?.department }} · {{ slot.doctor?.title }}</span>
              <small>{{ hasStock(slot) ? `剩余 ${slot.capacity - slot.bookedCount}` : '满员' }}</small>
            </button>
          </div>
        </div>
      </div>
      <el-empty v-else description="请先选择预约日期" />
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, reactive } from 'vue'
import { useDebouncedFn } from '../composables/useDebouncedFn'
import { appointmentTypes, useHealthData } from '../composables/useHealthData'

const { packages, institutions, slots, appointmentForm, loading, loadAll, selectPackage, createAppointment } = useHealthData()
const dialogs = reactive({ type: false, institution: false, package: false, date: false, slot: false })
const activePackages = computed(() => packages.value.filter((item) => item.status !== 'disabled'))
const activeInstitutions = computed(() => institutions.value.filter((item) => item.status !== 'disabled'))
const selectedInstitution = computed(() => activeInstitutions.value.find((item) => item.id === appointmentForm.institutionId))
const selectedPackage = computed(() => activePackages.value.find((item) => item.id === appointmentForm.packageId))
const institutionSlots = computed(() => slots.value.filter((slot) => slot.institutionId === appointmentForm.institutionId))
const dateStocks = computed(() => {
  const map = new Map()
  for (const slot of institutionSlots.value) {
    const current = map.get(slot.date) || { date: slot.date, available: 0, total: 0 }
    current.total += slot.capacity
    current.available += Math.max(0, slot.capacity - slot.bookedCount)
    map.set(slot.date, current)
  }
  return Array.from(map.values()).sort((a, b) => a.date.localeCompare(b.date))
})
const selectedDateSlots = computed(() => institutionSlots.value.filter((slot) => slot.date === appointmentForm.date).sort((a, b) => `${a.startTime}-${a.doctorId}`.localeCompare(`${b.startTime}-${b.doctorId}`)))
const groupedSlots = computed(() => {
  const map = new Map()
  for (const slot of selectedDateSlots.value) {
    const time = `${slot.startTime}-${slot.endTime}`
    const group = map.get(time) || { time, slots: [], available: 0 }
    group.slots.push(slot)
    if (hasStock(slot)) group.available += slot.capacity - slot.bookedCount
    map.set(time, group)
  }
  return Array.from(map.values())
})
const selectedSlot = computed(() => slots.value.find((slot) => slot.id === appointmentForm.slotId))
const selectedSlotText = computed(() => {
  if (!selectedSlot.value) return ''
  return `${selectedSlot.value.date} ${selectedSlot.value.startTime}-${selectedSlot.value.endTime} ${selectedSlot.value.doctor?.name || ''}`
})
const canSubmit = computed(() => Boolean(appointmentForm.appointmentType && appointmentForm.institutionId && appointmentForm.packageId && appointmentForm.date && appointmentForm.period && appointmentForm.slotId))
const submit = useDebouncedFn(createAppointment, 500)

function hasStock(slot) {
  return slot.status === 'available' && slot.capacity > slot.bookedCount
}

function selectType(type) {
  appointmentForm.appointmentType = type
  dialogs.type = false
}

function selectInstitution(institution) {
  appointmentForm.institutionId = institution.id
  appointmentForm.date = ''
  appointmentForm.period = ''
  appointmentForm.slotId = null
  dialogs.institution = false
}

function selectPackageAndClose(pkg) {
  selectPackage(pkg)
  dialogs.package = false
}

function selectDate(date) {
  appointmentForm.date = date
  appointmentForm.slotId = null
  appointmentForm.period = ''
  dialogs.date = false
  dialogs.slot = true
}

function selectSlot(slot) {
  appointmentForm.slotId = slot.id
  appointmentForm.date = slot.date
  appointmentForm.period = slot.period
  dialogs.slot = false
}
</script>

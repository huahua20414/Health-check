<template>
  <section class="view">
    <div class="layout-two booking-layout">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>选择套餐</h3>
            <p>先选体检套餐，再选择有库存的日期和具体半小时号源。</p>
          </div>
        </div>
        <div class="package-list">
          <button
            v-for="pkg in activePackages"
            :key="pkg.id"
            class="package-row"
            :class="{ selected: appointmentForm.packageId === pkg.id }"
            type="button"
            @click="selectPackage(pkg)"
          >
            <div>
              <h4>{{ pkg.name }}</h4>
              <p>{{ pkg.description }}</p>
              <span>{{ pkg.items }}</span>
            </div>
            <strong>￥{{ pkg.price }}</strong>
          </button>
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>日期库存</h3>
            <p>无库存日期不可选择；选择日期后再选具体号源。</p>
          </div>
          <el-button :loading="loading.load" @click="loadAll">刷新库存</el-button>
        </div>
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
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>选择具体时间段</h3>
          <p>一个医生一个半小时号源只能预约一人。</p>
        </div>
      </div>
      <div v-if="appointmentForm.date" class="slot-grid">
        <button
          v-for="slot in selectedDateSlots"
          :key="slot.id"
          type="button"
          class="slot-card"
          :class="{ selected: appointmentForm.slotId === slot.id, disabled: !hasStock(slot) }"
          :disabled="!hasStock(slot)"
          @click="selectSlot(slot)"
        >
          <div>
            <strong>{{ slot.startTime }}-{{ slot.endTime }}</strong>
            <span>{{ slot.doctor?.name }} · {{ slot.period }}</span>
          </div>
          <el-tag :type="hasStock(slot) ? 'success' : 'warning'">
            {{ hasStock(slot) ? `剩余 ${slot.capacity - slot.bookedCount}` : '满员' }}
          </el-tag>
        </button>
      </div>
      <el-empty v-else description="请先选择有库存的日期" />
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>确认预约</h3>
          <p>提交后系统会发送邮件，并在“我的预约”中展示具体医生和时间。</p>
        </div>
      </div>
      <el-form label-position="top" class="form-grid">
        <el-form-item label="已选套餐">
          <el-input :model-value="selectedPackage?.name || ''" disabled />
        </el-form-item>
        <el-form-item label="已选号源">
          <el-input :model-value="selectedSlotText" disabled />
        </el-form-item>
        <el-form-item label="备注说明">
          <el-input v-model="appointmentForm.note" type="textarea" :rows="4" placeholder="例如既往病史、特殊检查需求" />
        </el-form-item>
        <el-button type="primary" :disabled="!canSubmit" :loading="loading.appointment" @click="submit">
          提交预约
        </el-button>
      </el-form>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { useDebouncedFn } from '../composables/useDebouncedFn'
import { useHealthData } from '../composables/useHealthData'

const { packages, slots, appointmentForm, loading, loadAll, selectPackage, createAppointment } = useHealthData()
const activePackages = computed(() => packages.value.filter((item) => item.status !== 'disabled'))
const selectedPackage = computed(() => activePackages.value.find((item) => item.id === appointmentForm.packageId))
const dateStocks = computed(() => {
  const map = new Map()
  for (const slot of slots.value) {
    const current = map.get(slot.date) || { date: slot.date, available: 0, total: 0 }
    current.total += slot.capacity
    current.available += Math.max(0, slot.capacity - slot.bookedCount)
    map.set(slot.date, current)
  }
  return Array.from(map.values()).sort((a, b) => a.date.localeCompare(b.date))
})
const selectedDateSlots = computed(() => slots.value.filter((slot) => slot.date === appointmentForm.date).sort((a, b) => a.startTime.localeCompare(b.startTime)))
const selectedSlot = computed(() => slots.value.find((slot) => slot.id === appointmentForm.slotId))
const selectedSlotText = computed(() => {
  if (!selectedSlot.value) return ''
  return `${selectedSlot.value.date} ${selectedSlot.value.startTime}-${selectedSlot.value.endTime} ${selectedSlot.value.doctor?.name || ''}`
})
const canSubmit = computed(() => Boolean(appointmentForm.packageId && appointmentForm.date && appointmentForm.period && appointmentForm.slotId))
const submit = useDebouncedFn(createAppointment, 400)

function hasStock(slot) {
  return slot.status === 'available' && slot.capacity > slot.bookedCount
}

function selectDate(date) {
  appointmentForm.date = date
  appointmentForm.slotId = null
  appointmentForm.period = ''
}

function selectSlot(slot) {
  appointmentForm.slotId = slot.id
  appointmentForm.date = slot.date
  appointmentForm.period = slot.period
}
</script>

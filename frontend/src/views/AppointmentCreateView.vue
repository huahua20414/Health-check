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
            <strong>{{ selectedPackageText || '请选择' }}</strong>
          </button>
          <button type="button" class="selection-tile" :disabled="!appointmentForm.institutionId" @click="dialogs.date = true">
            <span>预约日期</span>
            <strong>{{ appointmentForm.date || '请选择' }}</strong>
          </button>
          <button type="button" class="selection-tile wide" :disabled="!appointmentForm.date" @click="dialogs.slot = true">
            <span>医生号源</span>
            <strong>{{ selectedSlotText || '请选择具体半小时号源' }}</strong>
          </button>
          <button type="button" class="selection-tile" @click="dialogs.member = true">
            <span>体检人</span>
            <strong>{{ selectedMember?.name || '本人' }}</strong>
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
          <div><dt>分类</dt><dd>{{ selectedCategory || '-' }}</dd></div>
          <div><dt>日期</dt><dd>{{ appointmentForm.date || '-' }}</dd></div>
          <div><dt>时间</dt><dd>{{ selectedSlot ? `${selectedSlot.startTime}-${selectedSlot.endTime}` : '-' }}</dd></div>
          <div><dt>医生</dt><dd>{{ selectedSlot?.doctor?.name || '-' }}</dd></div>
          <div><dt>体检人</dt><dd>{{ selectedMember?.name || '本人' }}</dd></div>
          <div><dt>原价</dt><dd>￥{{ Number(selectedPackage?.price || 0).toFixed(2) }}</dd></div>
          <div><dt>优惠</dt><dd>-￥{{ discountAmount.toFixed(2) }}</dd></div>
          <div><dt>应付</dt><dd>￥{{ payableAmount.toFixed(2) }}</dd></div>
          <div><dt>支付</dt><dd>{{ appointmentForm.paymentStatus === 'paid' ? '已支付' : '未支付' }}</dd></div>
        </dl>
        <el-form label-position="top">
          <el-form-item label="可用优惠券">
            <el-select v-model="appointmentForm.couponId" clearable filterable placeholder="不使用优惠券">
              <el-option
                v-for="coupon in usableCoupons"
                :key="coupon.id"
                :label="couponLabel(coupon)"
                :value="coupon.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="支付状态模拟">
            <el-segmented v-model="appointmentForm.paymentStatus" :options="paymentOptions" />
          </el-form-item>
          <el-form-item label="发票抬头">
            <el-input v-model="appointmentForm.invoiceTitle" placeholder="个人或企业名称" />
          </el-form-item>
          <el-form-item label="纳税人识别号">
            <el-input v-model="appointmentForm.invoiceTaxNo" placeholder="企业发票可填写" />
          </el-form-item>
          <el-form-item label="备注说明">
            <el-input v-model="appointmentForm.note" type="textarea" :rows="3" placeholder="例如既往病史、特殊检查需求" />
          </el-form-item>
          <el-button type="primary" class="summary-submit" :disabled="!canSubmit || !can('appointment:create')" :loading="loading.appointment" @click="submit">
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
            <em>{{ pkg.category }}</em>
            <p>{{ pkg.description }}</p>
            <span>{{ pkg.items }}</span>
          </div>
          <strong>￥{{ pkg.price }}</strong>
        </button>
      </div>
    </el-dialog>

    <el-dialog v-model="dialogs.member" title="选择体检人" width="760px" class="choice-dialog">
      <div class="segmented-grid">
        <button type="button" class="segment-item" :class="{ selected: !appointmentForm.familyMemberId }" @click="selectMember(null)">本人</button>
        <button
          v-for="member in familyMembers"
          :key="member.id"
          type="button"
          class="segment-item"
          :class="{ selected: appointmentForm.familyMemberId === member.id }"
          @click="selectMember(member)"
        >
          {{ member.name }} · {{ member.relation }}
        </button>
      </div>
      <el-empty v-if="familyMembers.length === 0" description="暂无家庭成员，可在个人中心维护" />
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
            <el-button v-if="group.available === 0 && can('appointment:create')" size="small" type="warning" plain :loading="loading.appointment" @click="joinFullGroup(group)">
              加入候补
            </el-button>
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
              <span>{{ slot.category }}</span>
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
import { computed, onMounted, onUnmounted, reactive, watch } from 'vue'
import { useDebouncedFn } from '../composables/useDebouncedFn'
import { appointmentTypes, useHealthData } from '../composables/useHealthData'

const { packages, institutions, slots, familyMembers, activeCoupons, appointmentForm, loading, can, loadAll, selectPackage, createAppointment, joinWaitlist } = useHealthData()
const dialogs = reactive({ type: false, institution: false, package: false, date: false, slot: false, member: false })
let refreshTimer = 0
const paymentOptions = [
  { label: '未支付', value: 'unpaid' },
  { label: '已支付', value: 'paid' },
]
const activePackages = computed(() => packages.value.filter((item) => item.status !== 'disabled'))
const activeInstitutions = computed(() => institutions.value.filter((item) => item.status !== 'disabled'))
const selectedInstitution = computed(() => activeInstitutions.value.find((item) => item.id === appointmentForm.institutionId))
const selectedPackage = computed(() => activePackages.value.find((item) => item.id === appointmentForm.packageId))
const selectedMember = computed(() => familyMembers.value.find((item) => item.id === appointmentForm.familyMemberId))
const usableCoupons = computed(() => activeCoupons.value.filter((coupon) => couponApplies(coupon)))
const selectedCoupon = computed(() => usableCoupons.value.find((coupon) => coupon.id === appointmentForm.couponId))
const discountAmount = computed(() => discountValue(selectedPackage.value, selectedCoupon.value))
const payableAmount = computed(() => Math.max(0, Number(selectedPackage.value?.price || 0) - discountAmount.value))
const selectedCategory = computed(() => selectedPackage.value?.category || '')
const selectedPackageText = computed(() => (selectedPackage.value ? `${selectedPackage.value.name} · ${selectedPackage.value.category}` : ''))
const institutionSlots = computed(() => slots.value.filter((slot) => slot.institutionId === appointmentForm.institutionId && (!selectedCategory.value || slot.category === selectedCategory.value)))
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
  appointmentForm.couponId = null
  appointmentForm.date = ''
  appointmentForm.period = ''
  appointmentForm.slotId = null
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

function couponApplies(coupon) {
  const price = Number(selectedPackage.value?.price || 0)
  if (!selectedPackage.value || coupon.status !== 'active') return false
  if (coupon.packageId && coupon.packageId !== selectedPackage.value.id) return false
  if (Number(coupon.minAmount || 0) > price) return false
  if (appointmentForm.date) {
    if (coupon.startDate && appointmentForm.date < coupon.startDate) return false
    if (coupon.endDate && appointmentForm.date > coupon.endDate) return false
  }
  return true
}

function discountValue(pkg, coupon) {
  if (!pkg || !coupon) return 0
  const price = Number(pkg.price || 0)
  const value = coupon.type === 'percent' ? price * Number(coupon.value || 0) / 100 : Number(coupon.value || 0)
  return Math.min(price, Math.max(0, value))
}

function couponLabel(coupon) {
  const discount = discountValue(selectedPackage.value, coupon).toFixed(2)
  const threshold = Number(coupon.minAmount || 0) > 0 ? `满￥${Number(coupon.minAmount).toFixed(2)} ` : ''
  return `${coupon.name} · ${threshold}优惠￥${discount}`
}

function selectMember(member) {
  appointmentForm.familyMemberId = member?.id || null
  dialogs.member = false
}

async function joinFullGroup(group) {
  const first = group.slots[0]
  if (!first) return
  await joinWaitlist(first)
  dialogs.slot = false
}

watch(() => dialogs.date, (open) => {
  if (open) loadAll()
})

watch(() => dialogs.slot, (open) => {
  if (open) loadAll()
})

onMounted(() => {
  refreshTimer = window.setInterval(loadAll, 10000)
})

onUnmounted(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
})
</script>

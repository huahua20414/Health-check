<template>
  <section class="view">
    <div class="layout-two booking-layout">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>选择套餐</h3>
            <p>选择套餐、日期和时段后，系统自动分配医生和半小时号源。</p>
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
            <h3>创建预约</h3>
            <p>提交后可在“我的预约”中跟踪状态。</p>
          </div>
        </div>
        <el-form label-position="top" class="form-grid">
          <el-form-item label="预约套餐">
            <el-select v-model="appointmentForm.packageId" placeholder="选择套餐">
              <el-option v-for="pkg in activePackages" :key="pkg.id" :label="pkg.name" :value="pkg.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="预约日期">
            <el-date-picker v-model="appointmentForm.date" value-format="YYYY-MM-DD" type="date" placeholder="选择日期" />
          </el-form-item>
          <el-form-item label="预约时段">
            <el-select v-model="appointmentForm.period">
              <el-option label="上午" value="上午" />
              <el-option label="下午" value="下午" />
            </el-select>
          </el-form-item>
          <el-form-item label="备注说明">
            <el-input v-model="appointmentForm.note" type="textarea" :rows="4" placeholder="例如既往病史、特殊检查需求" />
          </el-form-item>
          <el-button type="primary" :disabled="!appointmentForm.packageId" :loading="loading.appointment" @click="submit">
            {{ availableSlotCount > 0 ? '提交预约' : '加入候补' }}
          </el-button>
        </el-form>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>可用号源</h3>
          <p>一个医生每个半小时号源只能预约一人，满员时自动进入候补。</p>
        </div>
        <el-button :loading="loading.load" @click="loadAll">刷新号源</el-button>
      </div>
      <el-table :data="slots" stripe>
        <el-table-column label="医生" width="120">
          <template #default="{ row }">{{ row.doctor?.name }}</template>
        </el-table-column>
        <el-table-column prop="date" label="日期" width="120" />
        <el-table-column prop="period" label="时段" width="90" />
        <el-table-column label="时间" width="130">
          <template #default="{ row }">{{ row.startTime }}-{{ row.endTime }}</template>
        </el-table-column>
        <el-table-column label="库存">
          <template #default="{ row }">{{ row.capacity - row.bookedCount }} / {{ row.capacity }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.capacity > row.bookedCount ? 'success' : 'warning'">{{ row.capacity > row.bookedCount ? '可预约' : '满员' }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useDebouncedFn } from '../composables/useDebouncedFn'
import { useHealthData } from '../composables/useHealthData'

const { packages, slots, appointmentForm, loading, loadAll, selectPackage, createAppointment } = useHealthData()
const activePackages = computed(() => packages.value.filter((item) => item.status !== 'disabled'))
const availableSlotCount = computed(() => slots.value.filter((item) => item.capacity > item.bookedCount).length)
const submit = useDebouncedFn(createAppointment, 400)
const refreshSlots = useDebouncedFn(loadAll, 350)

watch(() => [appointmentForm.date, appointmentForm.period], refreshSlots)
</script>

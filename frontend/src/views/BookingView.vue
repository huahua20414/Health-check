<template>
  <section class="view">
    <div v-if="showBookingForm" class="layout-two booking-layout">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>套餐目录</h3>
            <p>面向用户展示可预约体检服务</p>
          </div>
        </div>
        <div class="package-list">
          <button
            v-for="pkg in packages"
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
            <p>真实系统中可扩展排班、名额和支付</p>
          </div>
        </div>
        <el-form label-position="top" class="form-grid">
          <el-form-item label="预约套餐">
            <el-select v-model="appointmentForm.packageId" placeholder="选择套餐">
              <el-option v-for="pkg in packages" :key="pkg.id" :label="pkg.name" :value="pkg.id" />
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
          <el-button type="primary" :disabled="!isUser" :loading="loading.appointment" @click="createAppointment">提交预约</el-button>
        </el-form>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的预约</h3>
          <p>用户端可跟踪预约状态和报告状态</p>
        </div>
      </div>
      <AppointmentTable :rows="myAppointments" :is-doctor="false" />
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppointmentTable from '../components/AppointmentTable.vue'
import { useHealthData } from '../composables/useHealthData'

const route = useRoute()
const showBookingForm = computed(() => route.name !== 'myAppointments')
const {
  packages,
  appointmentForm,
  myAppointments,
  isUser,
  loading,
  selectPackage,
  createAppointment,
} = useHealthData()
</script>

<template>
  <el-table :data="rows" stripe>
    <el-table-column prop="orderNo" label="订单号" width="170" />
    <el-table-column label="客户" width="100">
      <template #default="{ row }">{{ row.user?.name || '-' }}</template>
    </el-table-column>
    <el-table-column label="体检人" width="100">
      <template #default="{ row }">{{ row.familyMember?.name || '本人' }}</template>
    </el-table-column>
    <el-table-column prop="appointmentType" label="类型" width="100" />
    <el-table-column prop="category" label="分类" width="110" />
    <el-table-column label="机构" min-width="170">
      <template #default="{ row }">{{ row.institution?.name || '-' }}</template>
    </el-table-column>
    <el-table-column label="套餐">
      <template #default="{ row }">{{ row.package?.name || '-' }}</template>
    </el-table-column>
    <el-table-column prop="date" label="日期" width="120" />
    <el-table-column label="时间" width="130">
      <template #default="{ row }">{{ row.startTime }}-{{ row.endTime }}</template>
    </el-table-column>
    <el-table-column label="状态" width="110">
      <template #default="{ row }"><StatusTag :status="row.status" /></template>
    </el-table-column>
    <el-table-column label="支付" width="90">
      <template #default="{ row }">{{ row.paymentStatus === 'paid' ? '已支付' : '未支付' }}</template>
    </el-table-column>
    <el-table-column label="操作" width="250">
      <template #default="{ row }">
        <el-button size="small" @click="$emit('view-order', row)">查看订单</el-button>
        <el-button v-if="isDoctor && canMarkDone" size="small" :loading="loading" :disabled="row.status === 'reported' || row.status === 'canceled'" @click="$emit('mark-done', row)">
          完成体检
        </el-button>
        <template v-else>
          <el-button v-if="canPay" size="small" type="primary" plain :loading="loading" :disabled="row.status !== 'booked' || row.paymentStatus === 'paid'" @click="$emit('pay', row)">
            模拟支付
          </el-button>
          <el-button v-if="canPay && row.paymentStatus === 'paid'" size="small" plain :loading="loading" :disabled="row.status !== 'booked'" @click="$emit('unpay', row)">
            撤销支付
          </el-button>
          <el-button v-if="canInvoice" size="small" plain :disabled="row.status === 'canceled'" @click="$emit('invoice', row)">
            发票
          </el-button>
          <el-button v-if="canReschedule" size="small" :loading="loading" :disabled="row.status !== 'booked'" @click="$emit('reschedule', row)">
            改期
          </el-button>
          <el-button v-if="canReview" size="small" type="success" plain :disabled="row.status !== 'reported' && row.status !== 'checked'" @click="$emit('review', row)">
            评价
          </el-button>
          <el-button v-if="canCancel" size="small" type="danger" plain :loading="loading" :disabled="row.status !== 'booked'" @click="$emit('cancel', row)">
            取消预约
          </el-button>
        </template>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup>
import StatusTag from './StatusTag.vue'

defineProps({
  rows: {
    type: Array,
    required: true,
  },
  isDoctor: {
    type: Boolean,
    default: false,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  canCancel: {
    type: Boolean,
    default: false,
  },
  canReschedule: {
    type: Boolean,
    default: false,
  },
  canReview: {
    type: Boolean,
    default: false,
  },
  canPay: {
    type: Boolean,
    default: false,
  },
  canInvoice: {
    type: Boolean,
    default: false,
  },
  canMarkDone: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['mark-done', 'cancel', 'reschedule', 'review', 'view-order', 'pay', 'unpay', 'invoice'])
</script>

<template>
  <el-table :data="rows" stripe>
    <el-table-column prop="orderNo" label="订单号" width="170" />
    <el-table-column label="客户" width="100">
      <template #default="{ row }">{{ row.user?.name || '-' }}</template>
    </el-table-column>
    <el-table-column prop="appointmentType" label="类型" width="100" />
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
    <el-table-column label="操作" width="250">
      <template #default="{ row }">
        <el-button size="small" @click="$emit('view-order', row)">查看订单</el-button>
        <el-button v-if="isDoctor" size="small" :loading="loading" :disabled="row.status === 'reported' || row.status === 'canceled'" @click="$emit('mark-done', row)">
          完成体检
        </el-button>
        <el-button v-else-if="canCancel" size="small" type="danger" plain :loading="loading" :disabled="row.status !== 'booked'" @click="$emit('cancel', row)">
          取消预约
        </el-button>
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
})

defineEmits(['mark-done', 'cancel', 'view-order'])
</script>

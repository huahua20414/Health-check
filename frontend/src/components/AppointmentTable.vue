<template>
  <el-table :data="rows" stripe>
    <el-table-column prop="id" label="编号" width="70" />
    <el-table-column label="客户" width="100">
      <template #default="{ row }">{{ row.user?.name || '-' }}</template>
    </el-table-column>
    <el-table-column label="套餐">
      <template #default="{ row }">{{ row.package?.name || '-' }}</template>
    </el-table-column>
    <el-table-column prop="date" label="日期" width="120" />
    <el-table-column prop="period" label="时段" width="90" />
    <el-table-column label="状态" width="110">
      <template #default="{ row }"><StatusTag :status="row.status" /></template>
    </el-table-column>
    <el-table-column label="操作" width="170">
      <template #default="{ row }">
        <el-button size="small" :disabled="!isDoctor || row.status === 'reported'" @click="$emit('mark-done', row)">
          完成体检
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
})

defineEmits(['mark-done'])
</script>

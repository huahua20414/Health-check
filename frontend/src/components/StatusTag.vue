<template>
  <el-tag :type="statusType(status)">{{ statusText(status) }}</el-tag>
</template>

<script setup>
import { statusText, statusType } from '../composables/useHealthData'

defineProps({
  status: {
    type: String,
    required: true,
  },
})
</script>

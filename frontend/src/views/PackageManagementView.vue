<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>体检套餐管理</h3>
          <p>维护套餐价格、项目和适用人群</p>
        </div>
        <el-button type="primary" :icon="Plus" disabled>新增套餐</el-button>
      </div>
      <el-table :data="packages" stripe>
        <el-table-column prop="id" label="编号" width="80" />
        <el-table-column prop="name" label="套餐名称" width="160" />
        <el-table-column prop="description" label="说明" />
        <el-table-column prop="items" label="检查项目" />
        <el-table-column label="价格" width="120">
          <template #default="{ row }">￥{{ row.price }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default><el-tag type="success">启用</el-tag></template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup>
import { Plus } from '@element-plus/icons-vue'
import { useHealthData } from '../composables/useHealthData'

const { packages } = useHealthData()
</script>

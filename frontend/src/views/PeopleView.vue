<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>人员档案</h3>
          <p>展示客户与医生基础档案，便于后续扩展权限管理</p>
        </div>
      </div>
      <el-table :data="peopleRows" stripe>
        <el-table-column prop="id" label="编号" width="80" />
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="phone" label="手机号" width="160" />
        <el-table-column label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="row.role === 'doctor' ? 'warning' : 'success'">{{ row.role === 'doctor' ? '医生' : '用户' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="来源">
          <template #default="{ row }">{{ row.source }}</template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup>
import { useHealthData } from '../composables/useHealthData'

const { peopleRows } = useHealthData()
</script>

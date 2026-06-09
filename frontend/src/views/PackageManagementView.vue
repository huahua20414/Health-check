<template>
  <section class="view management-stack">
    <div class="panel form-panel">
      <div class="panel-head">
        <div>
          <h3>套餐表单</h3>
          <p>管理员维护体检服务，用户端只展示启用套餐。</p>
        </div>
      </div>
      <el-form label-position="top" class="form-grid spacious-form">
        <el-form-item label="套餐名称"><el-input v-model="packageForm.name" /></el-form-item>
        <el-form-item label="体检分类">
          <el-select v-model="packageForm.category" filterable allow-create default-first-option>
            <el-option v-for="category in packageCategories" :key="category" :label="category" :value="category" />
          </el-select>
        </el-form-item>
        <el-form-item label="套餐说明"><el-input v-model="packageForm.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="价格"><el-input-number v-model="packageForm.price" :min="0" :precision="2" /></el-form-item>
        <el-form-item label="检查项目"><el-input v-model="packageForm.items" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="packageForm.status">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <div class="actions">
          <el-button type="primary" :loading="loading.package" @click="submit">保存套餐</el-button>
          <el-button @click="editPackage(null)">清空</el-button>
        </div>
      </el-form>
    </div>

    <div class="panel wide-table-panel">
      <div class="panel-head">
        <div>
          <h3>套餐管理</h3>
          <p>套餐变更会影响用户端可预约项目。</p>
        </div>
      </div>
      <el-table :data="packages" stripe>
        <el-table-column prop="name" label="套餐名称" width="150" />
        <el-table-column prop="category" label="分类" width="120" />
        <el-table-column prop="description" label="说明" />
        <el-table-column label="价格" width="100">
          <template #default="{ row }">￥{{ row.price }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><StatusTag :status="row.status || 'active'" /></template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button size="small" @click="editPackage(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.packages.total"
        v-model:current-page="paginations.packages.page"
        v-model:page-size="paginations.packages.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>
  </section>
</template>

<script setup>
import { onMounted, watch } from 'vue'
import StatusTag from '../components/StatusTag.vue'
import { useDebouncedFn } from '../composables/useDebouncedFn'
import { useHealthData } from '../composables/useHealthData'

const { packages, packageForm, loading, editPackage, savePackage, paginations, loadPackagesPage } = useHealthData()
const packageCategories = ['入职体检', '慢病筛查', '年度综合', '影像专项', '女性专项', '老年体检']
const submit = useDebouncedFn(async () => {
  await savePackage()
  await loadPackagesPage()
}, 350)

watch(() => [paginations.packages.page, paginations.packages.pageSize], () => loadPackagesPage())
onMounted(() => loadPackagesPage())
</script>

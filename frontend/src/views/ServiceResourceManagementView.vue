<template>
  <section class="view management-stack">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>体检项目管理</h3>
          <p>维护检查项目、科室、价格和预计时长，供套餐组合复用。</p>
        </div>
        <div class="head-actions">
          <el-button :loading="loading.exportCheckupItems" :disabled="!can('admin:data:exchange')" @click="exportCheckupItems">
            导出项目 CSV
          </el-button>
          <el-upload accept=".csv" :auto-upload="false" :show-file-list="false" :on-change="handleCheckupItemImport">
            <el-button :loading="loading.importCheckupItems" :disabled="!can('admin:data:exchange')">导入项目 CSV</el-button>
          </el-upload>
        </div>
      </div>
      <el-form label-position="top" class="form-grid spacious-form">
        <el-form-item label="项目名称"><el-input v-model="checkupItemForm.name" /></el-form-item>
        <el-form-item label="项目分类"><el-input v-model="checkupItemForm.category" placeholder="如 检验 / 影像 / 基础检查" /></el-form-item>
        <el-form-item label="执行科室"><el-input v-model="checkupItemForm.department" /></el-form-item>
        <el-form-item label="价格"><el-input-number v-model="checkupItemForm.price" :min="0" :precision="2" /></el-form-item>
        <el-form-item label="预计分钟"><el-input-number v-model="checkupItemForm.durationMin" :min="1" :step="5" /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="checkupItemForm.status">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="项目说明"><el-input v-model="checkupItemForm.description" type="textarea" :rows="3" /></el-form-item>
        <div class="actions">
          <el-button type="primary" :loading="loading.checkupItem" :disabled="!checkupItemForm.name || !can('admin:resource:manage')" @click="submitCheckupItem">保存项目</el-button>
          <el-button @click="editCheckupItem(null)">清空</el-button>
        </div>
      </el-form>
      <el-table :data="checkupItems" stripe>
        <el-table-column prop="name" label="项目" min-width="150" />
        <el-table-column prop="category" label="分类" width="120" />
        <el-table-column prop="department" label="科室" width="120" />
        <el-table-column label="价格" width="100"><template #default="{ row }">￥{{ row.price }}</template></el-table-column>
        <el-table-column prop="durationMin" label="分钟" width="80" />
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button v-if="can('admin:resource:manage')" size="small" @click="editCheckupItem(row)">编辑</el-button>
              <el-button v-if="can('admin:resource:manage')" size="small" type="danger" plain :loading="loading.checkupItem" @click="archiveCheckupItem(row)">归档</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.checkupItems.total"
        v-model:current-page="paginations.checkupItems.page"
        v-model:page-size="paginations.checkupItems.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>套餐项目组合</h3>
          <p>把体检项目挂到套餐中，控制是否必检和展示顺序。</p>
        </div>
        <div class="head-actions">
          <el-button :loading="loading.exportPackageItems" :disabled="!can('admin:data:exchange')" @click="handlePackageItemExport">
            导出组合 CSV
          </el-button>
          <el-upload accept=".csv" :auto-upload="false" :show-file-list="false" :on-change="handlePackageItemImport">
            <el-button :loading="loading.importPackageItems" :disabled="!can('admin:data:exchange')">导入组合 CSV</el-button>
          </el-upload>
        </div>
      </div>
      <el-form label-position="top" class="form-grid compact-resource-form">
        <el-form-item label="套餐">
          <el-select v-model="packageItemForm.packageId" filterable @change="reloadPackageItems">
            <el-option v-for="pkg in packages" :key="pkg.id" :label="pkg.name" :value="pkg.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="体检项目">
          <el-select v-model="packageItemForm.itemId" filterable>
            <el-option v-for="item in checkupItems" :key="item.id" :label="`${item.name} · ${item.department || item.category || '未分组'}`" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="排序"><el-input-number v-model="packageItemForm.sortOrder" :min="0" /></el-form-item>
        <el-form-item label="必检"><el-switch v-model="packageItemForm.required" active-text="必检" inactive-text="可选" /></el-form-item>
        <div class="actions">
          <el-button type="primary" :loading="loading.packageItem" :disabled="!packageItemForm.packageId || !packageItemForm.itemId || !can('admin:resource:manage')" @click="submitPackageItem">保存组合</el-button>
        </div>
      </el-form>
      <el-table :data="packageItems" stripe>
        <el-table-column label="套餐" min-width="150"><template #default="{ row }">{{ row.package?.name || '-' }}</template></el-table-column>
        <el-table-column label="项目" min-width="150"><template #default="{ row }">{{ row.item?.name || '-' }}</template></el-table-column>
        <el-table-column label="科室" width="120"><template #default="{ row }">{{ row.item?.department || '-' }}</template></el-table-column>
        <el-table-column prop="sortOrder" label="排序" width="80" />
        <el-table-column label="属性" width="90"><template #default="{ row }"><el-tag :type="row.required ? 'success' : 'info'">{{ row.required ? '必检' : '可选' }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="90"><template #default="{ row }"><el-button v-if="can('admin:resource:manage')" size="small" type="danger" plain @click="removePackageItem(row)">移除</el-button></template></el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.packageItems.total"
        v-model:current-page="paginations.packageItems.page"
        v-model:page-size="paginations.packageItems.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>

    <div class="panel wide-table-panel">
      <div class="panel-head">
        <div>
          <h3>医生号源与容量</h3>
          <p>按医生、机构、日期和时段维护库存，已预约数量不能被压低。</p>
        </div>
      </div>
      <el-form label-position="top" class="form-grid spacious-form">
        <el-form-item label="医生">
          <el-select v-model="scheduleForm.doctorId" filterable>
            <el-option v-for="doctor in activeDoctors" :key="doctor.id" :label="doctor.name" :value="doctor.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="机构">
          <el-select v-model="scheduleForm.institutionId" filterable>
            <el-option v-for="institution in institutions" :key="institution.id" :label="institution.name" :value="institution.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="日期"><el-date-picker v-model="scheduleForm.date" value-format="YYYY-MM-DD" type="date" /></el-form-item>
        <el-form-item label="时段">
          <el-select v-model="scheduleForm.period">
            <el-option label="上午" value="上午" />
            <el-option label="下午" value="下午" />
          </el-select>
        </el-form-item>
        <el-form-item label="分类"><el-input v-model="scheduleForm.category" placeholder="如 年度综合" /></el-form-item>
        <el-form-item label="开始时间"><el-time-picker v-model="scheduleForm.startTime" value-format="HH:mm" format="HH:mm" /></el-form-item>
        <el-form-item label="结束时间"><el-time-picker v-model="scheduleForm.endTime" value-format="HH:mm" format="HH:mm" placeholder="默认 30 分钟后" /></el-form-item>
        <el-form-item label="容量"><el-input-number v-model="scheduleForm.capacity" :min="1" /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="scheduleForm.status">
            <el-option label="可预约" value="available" />
            <el-option label="已满" value="full" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <div class="actions">
          <el-button type="primary" :loading="loading.schedule" :disabled="!canSaveSchedule || !can('admin:resource:manage')" @click="submitSchedule">保存号源</el-button>
          <el-button @click="editScheduleSlot(null)">清空</el-button>
        </div>
      </el-form>
      <el-table :data="slots" stripe>
        <el-table-column prop="date" label="日期" width="110" />
        <el-table-column prop="period" label="时段" width="80" />
        <el-table-column label="时间" width="120"><template #default="{ row }">{{ row.startTime }}-{{ row.endTime }}</template></el-table-column>
        <el-table-column label="医生" width="120"><template #default="{ row }">{{ row.doctor?.name || '-' }}</template></el-table-column>
        <el-table-column label="机构" min-width="150"><template #default="{ row }">{{ row.institution?.name || '-' }}</template></el-table-column>
        <el-table-column prop="category" label="分类" width="110" />
        <el-table-column label="库存" width="100"><template #default="{ row }">{{ row.bookedCount }}/{{ row.capacity }}</template></el-table-column>
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button v-if="can('admin:resource:manage')" size="small" @click="editScheduleSlot(row)">编辑</el-button>
              <el-button
                v-if="can('admin:resource:manage')"
                size="small"
                type="danger"
                plain
                :loading="loading.schedule"
                :disabled="row.bookedCount > 0"
                @click="archiveScheduleSlot(row)"
              >
                归档
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.slots.total"
        v-model:current-page="paginations.slots.page"
        v-model:page-size="paginations.slots.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import StatusTag from '../components/StatusTag.vue'
import { useDebouncedFn } from '../composables/useDebouncedFn'
import { useHealthData } from '../composables/useHealthData'

const {
  packages,
  users,
  institutions,
  slots,
  checkupItems,
  packageItems,
  checkupItemForm,
  packageItemForm,
  scheduleForm,
  paginations,
  loading,
  can,
  loadPackagesPage,
  loadUsersPage,
  loadCheckupItemsPage,
  loadPackageItemsPage,
  loadSlotsPage,
  editCheckupItem,
  saveCheckupItem,
  archiveCheckupItem,
  exportCheckupItems,
  importCheckupItems,
  savePackageItem,
  exportPackageItems,
  importPackageItems,
  deletePackageItem,
  editScheduleSlot,
  saveScheduleSlot,
  archiveScheduleSlot,
} = useHealthData()

const activeDoctors = computed(() => users.value.filter((user) => user.role === 'doctor' && user.status === 'active'))
const canSaveSchedule = computed(() => scheduleForm.doctorId && scheduleForm.institutionId && scheduleForm.date && scheduleForm.period && scheduleForm.startTime)

const submitCheckupItem = useDebouncedFn(saveCheckupItem, 350)
const submitPackageItem = useDebouncedFn(savePackageItem, 350)
const submitSchedule = useDebouncedFn(saveScheduleSlot, 350)

function reloadPackageItems() {
  paginations.packageItems.page = 1
  loadPackageItemsPage(packageItemForm.packageId ? { packageId: packageItemForm.packageId } : {})
}

async function handleCheckupItemImport(file) {
  await importCheckupItems(file.raw)
}

function handlePackageItemExport() {
  return exportPackageItems(packageItemForm.packageId ? { packageId: packageItemForm.packageId } : {})
}

async function handlePackageItemImport(file) {
  await importPackageItems(file.raw)
}

async function removePackageItem(row) {
  await deletePackageItem(row)
}

watch(() => [paginations.checkupItems.page, paginations.checkupItems.pageSize], () => loadCheckupItemsPage())
watch(() => [paginations.packageItems.page, paginations.packageItems.pageSize], () => loadPackageItemsPage(packageItemForm.packageId ? { packageId: packageItemForm.packageId } : {}))
watch(() => [paginations.slots.page, paginations.slots.pageSize], () => loadSlotsPage())

onMounted(() => {
  loadPackagesPage()
  loadUsersPage({ role: 'doctor', status: 'active' })
  loadCheckupItemsPage()
  loadPackageItemsPage()
  loadSlotsPage()
})
</script>

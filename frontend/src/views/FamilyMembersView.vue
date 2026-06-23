<template>
  <section class="view">
    <div class="layout-two">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>家庭成员</h3>
            <p>为家人预约体检时可直接选择成员档案。</p>
          </div>
        </div>
        <el-form label-position="top" class="form-grid">
          <el-form-item label="姓名"><el-input v-model="familyMemberForm.name" /></el-form-item>
          <el-form-item label="关系"><el-input v-model="familyMemberForm.relation" placeholder="父母、配偶、子女" /></el-form-item>
          <el-form-item label="性别">
            <el-select v-model="familyMemberForm.gender" clearable>
              <el-option label="男" value="男" />
              <el-option label="女" value="女" />
            </el-select>
          </el-form-item>
          <el-form-item label="年龄"><el-input-number v-model="familyMemberForm.age" :min="0" :max="120" /></el-form-item>
          <el-form-item label="证件号"><el-input v-model="familyMemberForm.idCard" /></el-form-item>
          <el-form-item label="联系电话"><el-input v-model="familyMemberForm.phone" /></el-form-item>
        </el-form>
        <div class="dialog-actions">
          <el-button @click="editFamilyMember(null)">清空</el-button>
          <el-button type="primary" :loading="loading.familyMember" :disabled="!familyMemberForm.name || !familyMemberForm.relation" @click="saveFamilyMember">
            保存成员
          </el-button>
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>成员列表</h3>
            <p>所有成员只归属当前登录用户。</p>
          </div>
        </div>
        <el-table :data="familyMembers" stripe>
          <el-table-column prop="name" label="姓名" />
          <el-table-column prop="relation" label="关系" width="100" />
          <el-table-column prop="gender" label="性别" width="80" />
          <el-table-column prop="age" label="年龄" width="80" />
          <el-table-column prop="phone" label="电话" />
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <el-button size="small" @click="editFamilyMember(row)">编辑</el-button>
              <el-button size="small" type="danger" plain :loading="loading.familyMember" @click="deleteFamilyMember(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="familyMembers.length === 0" description="暂无家庭成员" />
      </div>
    </div>
  </section>
</template>

<script setup>
import { onMounted } from 'vue'
import { useHealthData } from '../composables/useHealthData'

const { familyMembers, familyMemberForm, loading, loadAll, editFamilyMember, saveFamilyMember, deleteFamilyMember } = useHealthData()

onMounted(loadAll)
</script>

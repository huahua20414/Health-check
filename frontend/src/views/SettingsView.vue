<template>
  <section class="view">
    <div class="settings-grid">
      <div class="panel setting-card">
        <el-icon><Connection /></el-icon>
        <h3>接口配置</h3>
        <p>当前前端通过 <code>{{ apiBase }}</code> 访问后端服务，Docker 环境已配置代理。</p>
      </div>
      <div class="panel setting-card">
        <el-icon><DataAnalysis /></el-icon>
        <h3>数据初始化</h3>
        <p>使用 <code>make seed</code> 插入用户、医生、套餐、预约和报告模拟数据。</p>
      </div>
      <div class="panel setting-card">
        <el-icon><Lock /></el-icon>
        <h3>权限说明</h3>
        <p>系统已接入 JWT、Redis Session、角色菜单和后端权限校验。</p>
      </div>
    </div>
    <div class="panel" v-if="isAdmin">
      <div class="panel-head">
        <div>
          <h3>邮件发送记录</h3>
          <p>预约成功、候补递补、报告生成都会记录邮件发送结果。</p>
        </div>
      </div>
      <el-table :data="mailLogs" stripe>
        <el-table-column prop="to" label="收件人" width="190" />
        <el-table-column prop="subject" label="主题" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'sent' ? 'success' : 'danger'">{{ row.status === 'sent' ? '已发送' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error" label="错误信息" />
      </el-table>
    </div>
  </section>
</template>

<script setup>
import { Connection, DataAnalysis, Lock } from '@element-plus/icons-vue'
import { apiBase } from '../api/client'
import { useHealthData } from '../composables/useHealthData'

const { isAdmin, mailLogs } = useHealthData()
</script>

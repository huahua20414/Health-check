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
        <p>当前为作业演示版，手机号模拟登录；生产环境可接入 JWT、RBAC 和审计日志。</p>
      </div>
    </div>
  </section>
</template>

<script setup>
import { Connection, DataAnalysis, Lock } from '@element-plus/icons-vue'
import { apiBase } from '../api/client'
</script>

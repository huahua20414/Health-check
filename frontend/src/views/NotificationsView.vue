<template>
  <section class="view">
    <div class="layout-two">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>消息通知</h3>
            <p>预约成功、短信模拟、体检前提醒会在这里留痕。</p>
          </div>
          <div class="filter-bar">
            <el-select v-model="notificationStatusFilter" placeholder="状态" clearable>
              <el-option label="未读" value="unread" />
              <el-option label="已读" value="read" />
            </el-select>
          </div>
        </div>
        <div class="notice-list">
          <article v-for="item in notifications" :key="item.id" class="notice-item" :class="{ unread: item.status === 'unread' }">
            <div>
              <strong>{{ item.title }}</strong>
              <p>{{ item.content }}</p>
              <span>{{ item.channel }} · {{ formatDate(item.createdAt) }}</span>
            </div>
            <div class="table-actions">
              <el-button v-if="item.status === 'unread'" size="small" :loading="loading.notification" @click="updateMyNotificationStatus(item, 'read')">标记已读</el-button>
              <el-button v-else size="small" plain :loading="loading.notification" @click="updateMyNotificationStatus(item, 'unread')">标为未读</el-button>
              <StatusTag :status="item.status" />
            </div>
          </article>
          <el-empty v-if="notifications.length === 0" description="暂无消息" />
        </div>
        <el-pagination
          class="table-pagination"
          background
          layout="total, sizes, prev, pager, next"
          :total="paginations.notifications.total"
          v-model:current-page="paginations.notifications.page"
          v-model:page-size="paginations.notifications.pageSize"
          :page-sizes="[10, 20, 50]"
        />
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>FAQ 与客服</h3>
            <p>常见问题和在线客服入口。</p>
          </div>
        </div>
        <el-collapse>
          <el-collapse-item v-for="(item, index) in supportInfo.faq" :key="item.question" :title="item.question" :name="String(index + 1)">
            {{ item.answer }}
          </el-collapse-item>
        </el-collapse>
        <div class="support-box">
          <strong>在线客服</strong>
          <p>服务时间 {{ supportInfo.customerServiceHours || '工作日 08:30-18:00' }}</p>
          <el-button type="primary" :disabled="!supportInfo.customerServiceUrl" @click="openSupport">进入客服</el-button>
        </div>
        <el-form label-position="top" class="support-form">
          <el-form-item label="咨询主题">
            <el-input v-model="supportTicketForm.subject" maxlength="128" show-word-limit placeholder="例如：发票、改期、报告问题" />
          </el-form-item>
          <el-form-item label="问题描述">
            <el-input v-model="supportTicketForm.content" type="textarea" :rows="4" maxlength="2000" show-word-limit placeholder="请描述需要客服协助的事项" />
          </el-form-item>
          <el-button type="primary" plain :loading="loading.notification" :disabled="!supportTicketForm.subject || !supportTicketForm.content" @click="createSupportTicket">提交咨询</el-button>
        </el-form>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的咨询</h3>
          <p>查看客服处理进度和回复。</p>
        </div>
      </div>
      <el-table :data="supportTickets" stripe>
        <el-table-column prop="subject" label="主题" min-width="140" />
        <el-table-column prop="content" label="描述" min-width="180" />
        <el-table-column prop="reply" label="客服回复" min-width="180" />
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="提交时间" width="120"><template #default="{ row }">{{ formatDate(row.createdAt) }}</template></el-table-column>
      </el-table>
      <el-empty v-if="supportTickets.length === 0" description="暂无咨询记录" />
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.supportTickets.total"
        v-model:current-page="paginations.supportTickets.page"
        v-model:page-size="paginations.supportTickets.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>系统公告</h3>
          <p>由管理端发布的服务通知。</p>
        </div>
      </div>
      <div class="notice-list">
        <article v-for="item in activeAnnouncements" :key="item.id" class="notice-item">
          <div>
            <strong>{{ item.title }}</strong>
            <p>{{ item.content }}</p>
            <span>{{ item.audience }} · {{ formatDate(item.publishedAt || item.createdAt) }}</span>
          </div>
        </article>
        <el-empty v-if="activeAnnouncements.length === 0" description="暂无公告" />
      </div>
    </div>
  </section>
</template>

<script setup>
import { ElMessage } from 'element-plus'
import { onMounted, ref, watch } from 'vue'
import StatusTag from '../components/StatusTag.vue'
import { formatDate, useHealthData } from '../composables/useHealthData'

const { notifications, activeAnnouncements, supportInfo, supportTickets, supportTicketForm, loading, paginations, loadAll, loadNotificationsPage, loadSupportTicketsPage, updateMyNotificationStatus, createSupportTicket } = useHealthData()
const notificationStatusFilter = ref('')

function openSupport() {
  if (!supportInfo.value.customerServiceUrl) {
    ElMessage.warning('客服入口暂未配置')
    return
  }
  window.open(supportInfo.value.customerServiceUrl, '_blank', 'noopener,noreferrer')
}

function loadNotificationPage(reset = false) {
  if (reset) paginations.notifications.page = 1
  return loadNotificationsPage({ status: notificationStatusFilter.value })
}

onMounted(() => {
  loadAll()
  loadNotificationPage()
  loadSupportTicketsPage()
})

watch(notificationStatusFilter, () => loadNotificationPage(true))
watch(() => [paginations.notifications.page, paginations.notifications.pageSize], () => loadNotificationPage())
watch(() => [paginations.supportTickets.page, paginations.supportTickets.pageSize], () => loadSupportTicketsPage())
</script>

<template>
  <section class="view">
    <div class="layout-two">
      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>消息通知</h3>
            <p>预约成功、短信模拟、体检前提醒会在这里留痕。</p>
          </div>
        </div>
        <div class="notice-list">
          <article v-for="item in notifications" :key="item.id" class="notice-item" :class="{ unread: item.status === 'unread' }">
            <div>
              <strong>{{ item.title }}</strong>
              <p>{{ item.content }}</p>
              <span>{{ item.channel }} · {{ formatDate(item.createdAt) }}</span>
            </div>
            <el-button v-if="item.status === 'unread'" size="small" :loading="loading.notification" @click="markNotificationRead(item)">标记已读</el-button>
            <StatusTag v-else status="read" />
          </article>
          <el-empty v-if="notifications.length === 0" description="暂无消息" />
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <div>
            <h3>FAQ 与客服</h3>
            <p>常见问题和在线客服入口。</p>
          </div>
        </div>
        <el-collapse>
          <el-collapse-item title="体检前需要注意什么？" name="1">
            前一天清淡饮食，部分抽血项目建议空腹；请携带有效证件并提前 15 分钟到达。
          </el-collapse-item>
          <el-collapse-item title="可以为家人预约吗？" name="2">
            可以。先在家庭成员中维护家人档案，提交预约时选择对应成员。
          </el-collapse-item>
          <el-collapse-item title="预约成功后会有什么提醒？" name="3">
            系统会生成站内信，并模拟短信通知；邮件通知按 SMTP 配置实际发送。
          </el-collapse-item>
        </el-collapse>
        <div class="support-box">
          <strong>在线客服</strong>
          <p>服务时间 08:30-18:00，当前为模拟入口。</p>
          <el-button type="primary" @click="openSupport">进入客服</el-button>
        </div>
      </div>
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
import { onMounted } from 'vue'
import StatusTag from '../components/StatusTag.vue'
import { formatDate, useHealthData } from '../composables/useHealthData'

const { notifications, activeAnnouncements, loading, loadAll, markNotificationRead } = useHealthData()

function openSupport() {
  ElMessage.info('客服入口已记录，真实环境可接入企业 IM 或工单系统')
}

onMounted(loadAll)
</script>

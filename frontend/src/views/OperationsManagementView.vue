<template>
  <section class="view management-stack">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>预约数据导出</h3>
          <p>按状态、客户、套餐或订单号导出预约 CSV，用于运营对账和线下交接。</p>
        </div>
      </div>
      <div class="filter-bar export-filter-bar">
        <el-select v-model="appointmentExportStatus" placeholder="预约状态" clearable>
          <el-option label="已预约" value="booked" />
          <el-option label="已体检" value="checked" />
          <el-option label="已出报告" value="reported" />
          <el-option label="已取消" value="cancelled" />
          <el-option label="候补" value="waitlisted" />
        </el-select>
        <el-input v-model="appointmentExportKeyword" placeholder="搜索客户/套餐/订单号/日期" clearable />
        <el-button
          type="primary"
          :loading="loading.exportAppointments"
          :disabled="!can('admin:data:exchange')"
          @click="handleAppointmentExport"
        >
          导出预约 CSV
        </el-button>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>优惠券管理</h3>
          <p>用于活动价、满减和特定套餐促销。</p>
        </div>
        <div class="head-actions">
          <el-button :loading="loading.exportCoupons" :disabled="!can('admin:data:exchange')" @click="handleCouponExport">
            导出优惠券 CSV
          </el-button>
          <el-upload accept=".csv" :auto-upload="false" :show-file-list="false" :on-change="handleCouponImport">
            <el-button :loading="loading.importCoupons" :disabled="!can('admin:data:exchange')">导入优惠券 CSV</el-button>
          </el-upload>
        </div>
      </div>
      <div class="filter-bar operation-filter-bar">
        <el-select v-model="couponStatusFilter" placeholder="优惠券状态" clearable>
          <el-option label="启用" value="active" />
          <el-option label="停用" value="disabled" />
          <el-option label="已归档" value="deleted" />
        </el-select>
        <el-input v-model="couponKeyword" placeholder="搜索券名/券码/说明" clearable />
      </div>
      <el-form label-position="top" class="form-grid spacious-form">
        <el-form-item label="名称"><el-input v-model="couponForm.name" /></el-form-item>
        <el-form-item label="券码"><el-input v-model="couponForm.code" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="couponForm.type">
            <el-option label="立减金额" value="amount" />
            <el-option label="折扣比例" value="percent" />
          </el-select>
        </el-form-item>
        <el-form-item label="优惠值"><el-input-number v-model="couponForm.value" :min="0" :precision="2" /></el-form-item>
        <el-form-item label="最低消费"><el-input-number v-model="couponForm.minAmount" :min="0" :precision="2" /></el-form-item>
        <el-form-item label="限定套餐">
          <el-select v-model="couponForm.packageId" clearable filterable>
            <el-option v-for="pkg in packages" :key="pkg.id" :label="pkg.name" :value="pkg.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="开始日期"><el-input v-model="couponForm.startDate" placeholder="YYYY-MM-DD" /></el-form-item>
        <el-form-item label="结束日期"><el-input v-model="couponForm.endDate" placeholder="YYYY-MM-DD" /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="couponForm.status">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="说明"><el-input v-model="couponForm.description" type="textarea" :rows="3" /></el-form-item>
        <div class="actions">
          <el-button type="primary" :loading="loading.coupon" :disabled="!couponForm.name || !couponForm.code || !can('admin:operation:manage')" @click="handleSaveCoupon">保存优惠券</el-button>
          <el-button @click="editCoupon(null)">清空</el-button>
        </div>
      </el-form>
      <el-table :data="coupons" stripe>
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="code" label="券码" width="120" />
        <el-table-column label="优惠" width="120">
          <template #default="{ row }">{{ row.type === 'percent' ? `${row.value}%` : `￥${row.value}` }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button v-if="can('admin:operation:manage')" size="small" @click="editCoupon(row)">编辑</el-button>
              <el-button v-if="can('admin:operation:manage')" size="small" type="danger" plain :loading="loading.coupon" @click="handleArchiveCoupon(row)">归档</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.coupons.total"
        v-model:current-page="paginations.coupons.page"
        v-model:page-size="paginations.coupons.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>评价管理</h3>
          <p>查看用户体检评价并维护回复或隐藏状态。</p>
        </div>
      </div>
      <div class="filter-bar operation-filter-bar">
        <el-select v-model="reviewStatusFilter" placeholder="评价状态" clearable>
          <el-option label="展示" value="published" />
          <el-option label="隐藏" value="hidden" />
        </el-select>
        <el-select v-model="reviewRatingFilter" placeholder="评分" clearable>
          <el-option v-for="score in [5, 4, 3, 2, 1]" :key="score" :label="`${score} 分`" :value="score" />
        </el-select>
        <el-input v-model="reviewKeyword" placeholder="搜索用户/医生/套餐/机构/评价" clearable />
        <el-button :loading="loading.exportReviews" :disabled="!can('admin:data:exchange')" @click="handleReviewExport">导出评价</el-button>
      </div>
      <el-table :data="reviews" stripe>
        <el-table-column label="用户" width="120"><template #default="{ row }">{{ row.user?.name || '-' }}</template></el-table-column>
        <el-table-column label="套餐"><template #default="{ row }">{{ row.package?.name || '-' }}</template></el-table-column>
        <el-table-column prop="rating" label="评分" width="80" />
        <el-table-column prop="content" label="评价内容" />
        <el-table-column prop="reply" label="回复" />
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="操作" width="90"><template #default="{ row }"><el-button v-if="can('admin:operation:manage')" size="small" @click="editReviewReply(row)">处理</el-button></template></el-table-column>
      </el-table>
      <div class="review-reply-box">
        <el-input v-model="reviewReplyForm.reply" type="textarea" :rows="3" placeholder="选择评价后填写回复" />
        <el-select v-model="reviewReplyForm.status">
          <el-option label="展示" value="published" />
          <el-option label="隐藏" value="hidden" />
        </el-select>
        <el-button type="primary" :disabled="!reviewReplyForm.id || !can('admin:operation:manage')" :loading="loading.review" @click="handleSaveReviewReply">保存处理</el-button>
      </div>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.reviews.total"
        v-model:current-page="paginations.reviews.page"
        v-model:page-size="paginations.reviews.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>系统公告管理</h3>
          <p>发布给用户、医生或全员的系统公告。</p>
        </div>
      </div>
      <div class="filter-bar operation-filter-bar">
        <el-select v-model="announcementStatusFilter" placeholder="公告状态" clearable>
          <el-option label="草稿" value="draft" />
          <el-option label="发布" value="published" />
          <el-option label="停用" value="disabled" />
          <el-option label="已归档" value="deleted" />
        </el-select>
        <el-input v-model="announcementKeyword" placeholder="搜索标题/内容/受众" clearable />
        <el-button :loading="loading.exportAnnouncements" :disabled="!can('admin:data:exchange')" @click="handleAnnouncementExport">导出公告</el-button>
      </div>
      <el-form label-position="top" class="form-grid spacious-form">
        <el-form-item label="标题"><el-input v-model="announcementForm.title" /></el-form-item>
        <el-form-item label="受众">
          <el-select v-model="announcementForm.audience">
            <el-option label="全员" value="all" />
            <el-option label="用户" value="user" />
            <el-option label="医生" value="doctor" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="announcementForm.status">
            <el-option label="草稿" value="draft" />
            <el-option label="发布" value="published" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容"><el-input v-model="announcementForm.content" type="textarea" :rows="4" /></el-form-item>
        <div class="actions">
          <el-button type="primary" :loading="loading.announcement" :disabled="!announcementForm.title || !announcementForm.content || !can('admin:operation:manage')" @click="handleSaveAnnouncement">保存公告</el-button>
          <el-button @click="editAnnouncement(null)">清空</el-button>
        </div>
      </el-form>
      <el-table :data="announcements" stripe>
        <el-table-column prop="title" label="标题" />
        <el-table-column prop="audience" label="受众" width="100" />
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column prop="content" label="内容" />
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button v-if="can('admin:operation:manage')" size="small" @click="editAnnouncement(row)">编辑</el-button>
              <el-button v-if="can('admin:operation:manage')" size="small" type="danger" plain :loading="loading.announcement" @click="handleArchiveAnnouncement(row)">归档</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.announcements.total"
        v-model:current-page="paginations.announcements.page"
        v-model:page-size="paginations.announcements.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>消息通知管理</h3>
          <p>向指定用户或角色发送站内信，也可维护历史通知状态。</p>
        </div>
      </div>
      <div class="notification-toolbar">
        <div class="reminder-actions">
          <el-date-picker v-model="reminderForm.date" value-format="YYYY-MM-DD" type="date" />
          <el-button
            type="primary"
            plain
            :loading="loading.reminder"
            :disabled="!reminderForm.date || !can('admin:notification:manage')"
            @click="sendCheckupReminders"
          >
            发送体检前提醒
          </el-button>
        </div>
        <div class="filter-bar">
          <el-select v-model="notificationStatusFilter" placeholder="状态" clearable>
            <el-option label="未读" value="unread" />
            <el-option label="已读" value="read" />
            <el-option label="已归档" value="archived" />
          </el-select>
          <el-select v-model="notificationChannelFilter" placeholder="渠道" clearable>
            <el-option label="站内信" value="in_app" />
            <el-option label="短信模拟" value="sms_mock" />
          </el-select>
          <el-input v-model="notificationKeyword" placeholder="搜索标题/用户" clearable />
        </div>
      </div>
      <el-form label-position="top" class="form-grid spacious-form">
        <el-form-item label="指定用户">
          <el-select v-model="notificationForm.userId" clearable filterable placeholder="不选则按角色群发">
            <el-option v-for="user in users" :key="user.id" :label="`${user.name}（${user.email || user.role}）`" :value="user.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="群发角色">
          <el-select v-model="notificationForm.role" :disabled="Boolean(notificationForm.userId)">
            <el-option label="用户" value="user" />
            <el-option label="医生" value="doctor" />
            <el-option label="管理员" value="admin" />
            <el-option label="全员" value="all" />
          </el-select>
        </el-form-item>
        <el-form-item label="渠道">
          <el-select v-model="notificationForm.channel">
            <el-option label="站内信" value="in_app" />
            <el-option label="短信模拟" value="sms_mock" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型"><el-input v-model="notificationForm.type" /></el-form-item>
        <el-form-item label="标题"><el-input v-model="notificationForm.title" /></el-form-item>
        <el-form-item label="内容"><el-input v-model="notificationForm.content" type="textarea" :rows="3" /></el-form-item>
        <div class="actions">
          <el-button type="primary" :loading="loading.adminNotification" :disabled="!notificationForm.title || !notificationForm.content || !can('admin:notification:manage')" @click="sendAdminNotification">发送通知</el-button>
          <el-button @click="resetNotificationForm">清空</el-button>
        </div>
      </el-form>
      <el-table :data="adminNotifications" stripe>
        <el-table-column label="用户" width="140"><template #default="{ row }">{{ row.user?.name || row.userId }}</template></el-table-column>
        <el-table-column prop="channel" label="渠道" width="100" />
        <el-table-column prop="type" label="类型" width="130" />
        <el-table-column prop="title" label="标题" />
        <el-table-column prop="content" label="内容" />
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="操作" width="190">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button
                v-if="can('admin:notification:manage') && row.status === 'unread'"
                size="small"
                :loading="loading.adminNotification"
                @click="updateAdminNotificationStatus(row, 'read')"
              >
                标已读
              </el-button>
              <el-button
                v-if="can('admin:notification:manage') && row.status === 'read'"
                size="small"
                plain
                :loading="loading.adminNotification"
                @click="updateAdminNotificationStatus(row, 'unread')"
              >
                标未读
              </el-button>
              <el-button v-if="can('admin:notification:manage')" size="small" type="danger" plain :loading="loading.adminNotification" @click="archiveAdminNotification(row)">归档</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.adminNotifications.total"
        v-model:current-page="paginations.adminNotifications.page"
        v-model:page-size="paginations.adminNotifications.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>客服工单</h3>
          <p>处理用户在线咨询并将回复同步到用户消息。</p>
        </div>
        <div class="filter-bar">
          <el-select v-model="supportTicketStatusFilter" placeholder="状态" clearable>
            <el-option label="待处理" value="open" />
            <el-option label="已回复" value="replied" />
            <el-option label="已关闭" value="closed" />
          </el-select>
          <el-input v-model="supportTicketKeyword" placeholder="搜索主题/内容/用户" clearable />
          <el-button
            plain
            :loading="loading.exportSupportTickets"
            :disabled="!can('admin:data:exchange')"
            @click="handleSupportTicketExport"
          >
            导出工单 CSV
          </el-button>
        </div>
      </div>
      <el-table :data="adminSupportTickets" stripe>
        <el-table-column label="用户" width="140"><template #default="{ row }">{{ row.user?.name || row.userId }}</template></el-table-column>
        <el-table-column prop="subject" label="主题" min-width="140" />
        <el-table-column prop="content" label="内容" min-width="180" />
        <el-table-column prop="reply" label="回复" min-width="180" />
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column label="操作" width="90"><template #default="{ row }"><el-button v-if="can('admin:operation:manage')" size="small" @click="editSupportTicketReply(row)">处理</el-button></template></el-table-column>
      </el-table>
      <div class="review-reply-box">
        <el-input v-model="supportTicketReplyForm.reply" type="textarea" :rows="3" placeholder="选择工单后填写客服回复" />
        <el-select v-model="supportTicketReplyForm.status">
          <el-option label="已回复" value="replied" />
          <el-option label="已关闭" value="closed" />
        </el-select>
        <el-button type="primary" :disabled="!supportTicketReplyForm.id || !supportTicketReplyForm.reply || !can('admin:operation:manage')" :loading="loading.adminNotification" @click="saveSupportTicketReply">保存回复</el-button>
      </div>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.adminSupportTickets.total"
        v-model:current-page="paginations.adminSupportTickets.page"
        v-model:page-size="paginations.adminSupportTickets.pageSize"
        :page-sizes="[10, 20, 50]"
      />
    </div>
  </section>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import StatusTag from '../components/StatusTag.vue'
import { useDebouncedRef } from '../composables/useDebouncedRef'
import { useHealthData } from '../composables/useHealthData'

const {
  packages,
  users,
  coupons,
  reviews,
  announcements,
  adminNotifications,
  adminSupportTickets,
  couponForm,
  reviewReplyForm,
  announcementForm,
  notificationForm,
  supportTicketReplyForm,
  reminderForm,
  loading,
  can,
  paginations,
  loadPackagesPage,
  loadUsersPage,
  loadCouponsPage,
  loadReviewsPage,
  loadAnnouncementsPage,
  loadAdminNotificationsPage,
  loadAdminSupportTicketsPage,
  exportAppointments,
  exportCoupons,
  exportReviews,
  importCoupons,
  exportSupportTickets,
  editCoupon,
  saveCoupon,
  archiveCoupon,
  editReviewReply,
  saveReviewReply,
  editAnnouncement,
  saveAnnouncement,
  archiveAnnouncement,
  exportAnnouncements,
  resetNotificationForm,
  sendAdminNotification,
  sendCheckupReminders,
  archiveAdminNotification,
  updateAdminNotificationStatus,
  editSupportTicketReply,
  saveSupportTicketReply,
} = useHealthData()

const notificationStatusFilter = ref('')
const notificationChannelFilter = ref('')
const notificationKeyword = ref('')
const appointmentExportStatus = ref('')
const appointmentExportKeyword = ref('')
const supportTicketStatusFilter = ref('')
const supportTicketKeyword = ref('')
const couponStatusFilter = ref('')
const couponKeyword = ref('')
const reviewStatusFilter = ref('')
const reviewRatingFilter = ref(null)
const reviewKeyword = ref('')
const announcementStatusFilter = ref('')
const announcementKeyword = ref('')
const debouncedNotificationKeyword = useDebouncedRef(notificationKeyword, 350)
const debouncedAppointmentExportKeyword = useDebouncedRef(appointmentExportKeyword, 350)
const debouncedSupportTicketKeyword = useDebouncedRef(supportTicketKeyword, 350)
const debouncedCouponKeyword = useDebouncedRef(couponKeyword, 350)
const debouncedReviewKeyword = useDebouncedRef(reviewKeyword, 350)
const debouncedAnnouncementKeyword = useDebouncedRef(announcementKeyword, 350)

function loadNotificationPage(reset = false) {
  if (reset) paginations.adminNotifications.page = 1
  return loadAdminNotificationsPage({
    status: notificationStatusFilter.value,
    channel: notificationChannelFilter.value,
    keyword: debouncedNotificationKeyword.value,
  })
}

function handleAppointmentExport() {
  return exportAppointments({
    status: appointmentExportStatus.value,
    keyword: debouncedAppointmentExportKeyword.value,
  })
}

function couponFilters() {
  return {
    status: couponStatusFilter.value,
    keyword: debouncedCouponKeyword.value,
  }
}

function loadCouponPage(reset = false) {
  if (reset) paginations.coupons.page = 1
  return loadCouponsPage(couponFilters())
}

function handleCouponExport() {
  return exportCoupons(couponFilters())
}

async function handleSaveCoupon() {
  await saveCoupon()
  await loadCouponPage()
}

async function handleArchiveCoupon(row) {
  await archiveCoupon(row)
  await loadCouponPage()
}

async function handleCouponImport(file) {
  await importCoupons(file.raw)
  await loadCouponPage()
}

function reviewFilters() {
  return {
    status: reviewStatusFilter.value,
    rating: reviewRatingFilter.value,
    keyword: debouncedReviewKeyword.value,
  }
}

function loadReviewPage(reset = false) {
  if (reset) paginations.reviews.page = 1
  return loadReviewsPage(reviewFilters())
}

async function handleSaveReviewReply() {
  await saveReviewReply(reviewFilters())
}

function handleReviewExport() {
  return exportReviews(reviewFilters())
}

function announcementFilters() {
  return {
    status: announcementStatusFilter.value,
    keyword: debouncedAnnouncementKeyword.value,
  }
}

function loadAnnouncementPage(reset = false) {
  if (reset) paginations.announcements.page = 1
  return loadAnnouncementsPage(announcementFilters())
}

async function handleSaveAnnouncement() {
  await saveAnnouncement(announcementFilters())
}

async function handleArchiveAnnouncement(row) {
  await archiveAnnouncement(row, announcementFilters())
}

function handleAnnouncementExport() {
  return exportAnnouncements(announcementFilters())
}

function handleSupportTicketExport() {
  return exportSupportTickets({
    status: supportTicketStatusFilter.value,
    keyword: debouncedSupportTicketKeyword.value,
  })
}

function loadSupportTicketPage(reset = false) {
  if (reset) paginations.adminSupportTickets.page = 1
  return loadAdminSupportTicketsPage({
    status: supportTicketStatusFilter.value,
    keyword: debouncedSupportTicketKeyword.value,
  })
}

watch([notificationStatusFilter, notificationChannelFilter, debouncedNotificationKeyword], () => loadNotificationPage(true))
watch(() => [paginations.adminNotifications.page, paginations.adminNotifications.pageSize], () => loadNotificationPage())
watch([supportTicketStatusFilter, debouncedSupportTicketKeyword], () => loadSupportTicketPage(true))
watch(() => [paginations.adminSupportTickets.page, paginations.adminSupportTickets.pageSize], () => loadSupportTicketPage())
watch([couponStatusFilter, debouncedCouponKeyword], () => loadCouponPage(true))
watch(() => [paginations.coupons.page, paginations.coupons.pageSize], () => loadCouponPage())
watch([reviewStatusFilter, reviewRatingFilter, debouncedReviewKeyword], () => loadReviewPage(true))
watch(() => [paginations.reviews.page, paginations.reviews.pageSize], () => loadReviewPage())
watch([announcementStatusFilter, debouncedAnnouncementKeyword], () => loadAnnouncementPage(true))
watch(() => [paginations.announcements.page, paginations.announcements.pageSize], () => loadAnnouncementPage())

onMounted(() => {
  loadPackagesPage()
  loadUsersPage({ role: 'user', status: 'active' })
  loadCouponPage()
  loadReviewPage()
  loadAnnouncementPage()
  loadNotificationPage()
  loadSupportTicketPage()
})
</script>

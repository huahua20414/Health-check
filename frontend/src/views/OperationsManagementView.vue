<template>
  <section class="view management-stack">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>优惠券管理</h3>
          <p>用于活动价、满减和特定套餐促销。</p>
        </div>
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
          <el-button type="primary" :loading="loading.coupon" :disabled="!couponForm.name || !couponForm.code || !can('admin:operation:manage')" @click="saveCoupon">保存优惠券</el-button>
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
        <el-table-column label="操作" width="90"><template #default="{ row }"><el-button v-if="can('admin:operation:manage')" size="small" @click="editCoupon(row)">编辑</el-button></template></el-table-column>
      </el-table>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>评价管理</h3>
          <p>查看用户体检评价并维护回复或隐藏状态。</p>
        </div>
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
        <el-button type="primary" :disabled="!reviewReplyForm.id || !can('admin:operation:manage')" :loading="loading.review" @click="saveReviewReply">保存处理</el-button>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>系统公告管理</h3>
          <p>发布给用户、医生或全员的系统公告。</p>
        </div>
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
          <el-button type="primary" :loading="loading.announcement" :disabled="!announcementForm.title || !announcementForm.content || !can('admin:operation:manage')" @click="saveAnnouncement">保存公告</el-button>
          <el-button @click="editAnnouncement(null)">清空</el-button>
        </div>
      </el-form>
      <el-table :data="announcements" stripe>
        <el-table-column prop="title" label="标题" />
        <el-table-column prop="audience" label="受众" width="100" />
        <el-table-column label="状态" width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
        <el-table-column prop="content" label="内容" />
        <el-table-column label="操作" width="90"><template #default="{ row }"><el-button v-if="can('admin:operation:manage')" size="small" @click="editAnnouncement(row)">编辑</el-button></template></el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup>
import { onMounted } from 'vue'
import StatusTag from '../components/StatusTag.vue'
import { useHealthData } from '../composables/useHealthData'

const {
  packages,
  coupons,
  reviews,
  announcements,
  couponForm,
  reviewReplyForm,
  announcementForm,
  loading,
  can,
  loadPackagesPage,
  loadCouponsPage,
  loadReviewsPage,
  loadAnnouncementsPage,
  editCoupon,
  saveCoupon,
  editReviewReply,
  saveReviewReply,
  editAnnouncement,
  saveAnnouncement,
} = useHealthData()

onMounted(() => {
  loadPackagesPage()
  loadCouponsPage()
  loadReviewsPage()
  loadAnnouncementsPage()
})
</script>

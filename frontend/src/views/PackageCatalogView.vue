<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>体检套餐</h3>
          <p>支持热门、推荐、收藏和最近浏览。</p>
        </div>
      </div>
      <div class="insight-strip">
        <div>
          <span>热门套餐</span>
          <strong>{{ popularPackages.map((item) => item.name).slice(0, 2).join('、') || '暂无' }}</strong>
        </div>
        <div>
          <span>推荐套餐</span>
          <strong>{{ recommendedPackages.map((item) => item.name).slice(0, 2).join('、') || '暂无' }}</strong>
        </div>
        <div>
          <span>最近浏览</span>
          <strong>{{ browseHistories[0]?.package?.name || '暂无' }}</strong>
        </div>
      </div>
      <div class="package-list">
        <div
          v-for="pkg in activePackages"
          :key="pkg.id"
          class="package-row"
        >
          <button type="button" class="package-main" @click="goBooking(pkg)">
            <h4>{{ pkg.name }}</h4>
            <em>{{ pkg.category }}</em>
            <p>{{ pkg.description }}</p>
            <span>{{ pkg.items }}</span>
          </button>
          <div class="package-actions">
            <strong>￥{{ pkg.price }}</strong>
            <small v-if="bestCoupon(pkg)">活动价约 ￥{{ campaignPrice(pkg).toFixed(2) }}</small>
            <el-button v-if="can('favorite:manage')" size="small" :type="isFavorite(pkg) ? 'warning' : 'primary'" plain :loading="loading.favorite" @click="toggleFavorite(pkg)">
              {{ isFavorite(pkg) ? '已收藏' : '收藏' }}
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useHealthData } from '../composables/useHealthData'

const router = useRouter()
const { packages, favorites, browseHistories, popularPackages, recommendedPackages, activeCoupons, loading, can, selectPackage, toggleFavorite, recordPackageBrowse } = useHealthData()
const activePackages = computed(() => packages.value.filter((item) => item.status !== 'disabled'))

function goBooking(pkg) {
  selectPackage(pkg)
  recordPackageBrowse(pkg)
  router.push('/booking')
}

function isFavorite(pkg) {
  return favorites.value.some((item) => item.packageId === pkg.id)
}

function bestCoupon(pkg) {
  return activeCoupons.value
    .filter((item) => (!item.packageId || item.packageId === pkg.id) && Number(pkg.price) >= Number(item.minAmount || 0))
    .sort((a, b) => discountValue(pkg, b) - discountValue(pkg, a))[0]
}

function discountValue(pkg, coupon) {
  if (!coupon) return 0
  if (coupon.type === 'percent') return Number(pkg.price || 0) * Number(coupon.value || 0) / 100
  return Number(coupon.value || 0)
}

function campaignPrice(pkg) {
  return Math.max(0, Number(pkg.price || 0) - discountValue(pkg, bestCoupon(pkg)))
}
</script>

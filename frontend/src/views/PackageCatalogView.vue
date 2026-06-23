<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>体检套餐</h3>
          <p>支持热门、推荐、收藏和最近浏览。</p>
        </div>
      </div>
      <div class="filter-bar package-filter-bar">
        <el-input v-model="keyword" placeholder="搜索套餐、项目或说明" clearable />
        <el-select v-model="categoryFilter" placeholder="全部分类" clearable>
          <el-option v-for="category in packageCategories" :key="category" :label="category" :value="category" />
        </el-select>
        <el-select v-model="sortBy" placeholder="排序">
          <el-option label="价格从低到高" value="price_asc" />
          <el-option label="价格从高到低" value="price_desc" />
          <el-option label="最新上架" value="created_desc" />
        </el-select>
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
        <el-empty v-if="!loading.load && activePackages.length === 0" description="暂无匹配套餐" />
      </div>
      <el-pagination
        class="table-pagination"
        background
        layout="total, sizes, prev, pager, next"
        :total="paginations.packages.total"
        v-model:current-page="paginations.packages.page"
        v-model:page-size="paginations.packages.pageSize"
        :page-sizes="[6, 10, 20]"
      />
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useDebouncedRef } from '../composables/useDebouncedRef'
import { useHealthData } from '../composables/useHealthData'

const router = useRouter()
const keyword = ref('')
const categoryFilter = ref('')
const sortBy = ref('price_asc')
const debouncedKeyword = useDebouncedRef(keyword, 350)
const { packages, favorites, browseHistories, popularPackages, recommendedPackages, activeCoupons, loading, can, selectPackage, toggleFavorite, recordPackageBrowse, paginations, loadPackagesPage } = useHealthData()
const activePackages = computed(() => packages.value.filter((item) => item.status !== 'disabled'))
const basePackageCategories = ['入职体检', '慢病筛查', '年度综合', '影像专项', '女性专项', '老年体检']
const packageCategories = computed(() => {
  const categories = new Set([
    ...basePackageCategories,
    ...packages.value.map((item) => item.category).filter(Boolean),
    ...popularPackages.value.map((item) => item.category).filter(Boolean),
    ...recommendedPackages.value.map((item) => item.category).filter(Boolean),
  ])
  return [...categories]
})

function loadCatalog(reset = false) {
  if (reset) paginations.packages.page = 1
  return loadPackagesPage({
    keyword: debouncedKeyword.value,
    category: categoryFilter.value,
    sort: sortBy.value,
  })
}

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

watch([debouncedKeyword, categoryFilter, sortBy], () => loadCatalog(true))
watch(() => [paginations.packages.page, paginations.packages.pageSize], () => loadCatalog())
onMounted(() => loadCatalog())
</script>

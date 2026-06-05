<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>体检套餐</h3>
          <p>套餐数据来自后端，停用套餐不会展示给用户预约。</p>
        </div>
      </div>
      <div class="package-list">
        <button
          v-for="pkg in activePackages"
          :key="pkg.id"
          class="package-row"
          type="button"
          @click="goBooking(pkg)"
        >
          <div>
            <h4>{{ pkg.name }}</h4>
            <p>{{ pkg.description }}</p>
            <span>{{ pkg.items }}</span>
          </div>
          <strong>￥{{ pkg.price }}</strong>
        </button>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useHealthData } from '../composables/useHealthData'

const router = useRouter()
const { packages, selectPackage } = useHealthData()
const activePackages = computed(() => packages.value.filter((item) => item.status !== 'disabled'))

function goBooking(pkg) {
  selectPackage(pkg)
  router.push('/booking')
}
</script>

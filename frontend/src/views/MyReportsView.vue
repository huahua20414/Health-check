<template>
  <section class="view">
    <div class="panel">
      <div class="panel-head">
        <div>
          <h3>我的报告</h3>
          <p>报告由医生生成后自动展示。</p>
        </div>
      </div>
      <ReportList :reports="reports" user-title />
    </div>
  </section>
</template>

<script setup>
import ReportList from '../components/ReportList.vue'
import { useHealthData } from '../composables/useHealthData'

const { reports } = useHealthData()
</script>

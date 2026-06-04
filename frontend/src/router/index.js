import { createRouter, createWebHashHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import DashboardView from '../views/DashboardView.vue'
import BookingView from '../views/BookingView.vue'
import AppointmentManagementView from '../views/AppointmentManagementView.vue'
import ReportManagementView from '../views/ReportManagementView.vue'
import PackageManagementView from '../views/PackageManagementView.vue'
import PeopleView from '../views/PeopleView.vue'
import SettingsView from '../views/SettingsView.vue'

export const menuItems = [
  { path: '/', name: 'dashboard', label: '工作台', icon: 'House' },
  { path: '/booking', name: 'booking', label: '用户预约', icon: 'Calendar' },
  { path: '/appointments', name: 'appointments', label: '预约管理', icon: 'Files' },
  { path: '/reports', name: 'reports', label: '报告管理', icon: 'Document' },
  { path: '/packages', name: 'packages', label: '体检套餐', icon: 'DataAnalysis' },
  { path: '/people', name: 'people', label: '人员档案', icon: 'User' },
  { path: '/settings', name: 'settings', label: '系统设置', icon: 'Setting' },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      component: AdminLayout,
      children: [
        { path: '', name: 'dashboard', component: DashboardView, meta: { title: '工作台' } },
        { path: 'booking', name: 'booking', component: BookingView, meta: { title: '用户预约' } },
        { path: 'appointments', name: 'appointments', component: AppointmentManagementView, meta: { title: '预约管理' } },
        { path: 'reports', name: 'reports', component: ReportManagementView, meta: { title: '报告管理' } },
        { path: 'packages', name: 'packages', component: PackageManagementView, meta: { title: '体检套餐' } },
        { path: 'people', name: 'people', component: PeopleView, meta: { title: '人员档案' } },
        { path: 'settings', name: 'settings', component: SettingsView, meta: { title: '系统设置' } },
      ],
    },
  ],
})

export default router

import { createRouter, createWebHashHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import DashboardView from '../views/DashboardView.vue'
import BookingView from '../views/BookingView.vue'
import AppointmentManagementView from '../views/AppointmentManagementView.vue'
import ReportManagementView from '../views/ReportManagementView.vue'
import PackageManagementView from '../views/PackageManagementView.vue'
import PeopleView from '../views/PeopleView.vue'
import SettingsView from '../views/SettingsView.vue'
import { useHealthData } from '../composables/useHealthData'

export const menuItems = [
  { path: '/', name: 'dashboard', label: '用户工作台', icon: 'House', roles: ['user'] },
  { path: '/booking', name: 'booking', label: '体检预约', icon: 'Calendar', roles: ['user'] },
  { path: '/my-appointments', name: 'myAppointments', label: '我的预约', icon: 'Tickets', roles: ['user'] },
  { path: '/my-reports', name: 'myReports', label: '我的报告', icon: 'DocumentChecked', roles: ['user'] },
  { path: '/profile', name: 'profile', label: '个人信息', icon: 'User', roles: ['user'] },
  { path: '/doctor', name: 'doctorDashboard', label: '医生工作台', icon: 'House', roles: ['doctor'] },
  { path: '/appointments', name: 'appointments', label: '预约管理', icon: 'Files', roles: ['doctor'] },
  { path: '/reports', name: 'reports', label: '报告管理', icon: 'Document', roles: ['doctor'] },
  { path: '/packages', name: 'packages', label: '体检套餐', icon: 'DataAnalysis', roles: ['doctor'] },
  { path: '/people', name: 'people', label: '客户档案', icon: 'User', roles: ['doctor'] },
  { path: '/settings', name: 'settings', label: '系统设置', icon: 'Setting', roles: ['doctor'] },
  { path: '/admin', name: 'adminDashboard', label: '管理工作台', icon: 'House', roles: ['admin'] },
  { path: '/admin/users', name: 'adminUsers', label: '用户管理', icon: 'User', roles: ['admin'] },
  { path: '/admin/packages', name: 'adminPackages', label: '套餐管理', icon: 'DataAnalysis', roles: ['admin'] },
  { path: '/admin/settings', name: 'adminSettings', label: '系统设置', icon: 'Setting', roles: ['admin'] },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { public: true, title: '登录' } },
    { path: '/register/:role', name: 'register', component: RegisterView, meta: { public: true, title: '注册' } },
    {
      path: '/',
      component: AdminLayout,
      children: [
        { path: '', name: 'dashboard', component: DashboardView, meta: { title: '用户工作台', roles: ['user'] } },
        { path: 'booking', name: 'booking', component: BookingView, meta: { title: '体检预约', roles: ['user'] } },
        { path: 'my-appointments', name: 'myAppointments', component: BookingView, meta: { title: '我的预约', roles: ['user'] } },
        { path: 'my-reports', name: 'myReports', component: ReportManagementView, meta: { title: '我的报告', roles: ['user'] } },
        { path: 'profile', name: 'profile', component: PeopleView, meta: { title: '个人信息', roles: ['user'] } },
        { path: 'doctor', name: 'doctorDashboard', component: DashboardView, meta: { title: '医生工作台', roles: ['doctor'] } },
        { path: 'appointments', name: 'appointments', component: AppointmentManagementView, meta: { title: '预约管理', roles: ['doctor'] } },
        { path: 'reports', name: 'reports', component: ReportManagementView, meta: { title: '报告管理', roles: ['doctor'] } },
        { path: 'packages', name: 'packages', component: PackageManagementView, meta: { title: '体检套餐', roles: ['doctor'] } },
        { path: 'people', name: 'people', component: PeopleView, meta: { title: '客户档案', roles: ['doctor'] } },
        { path: 'settings', name: 'settings', component: SettingsView, meta: { title: '系统设置', roles: ['doctor'] } },
        { path: 'admin', name: 'adminDashboard', component: DashboardView, meta: { title: '管理工作台', roles: ['admin'] } },
        { path: 'admin/users', name: 'adminUsers', component: PeopleView, meta: { title: '用户管理', roles: ['admin'] } },
        { path: 'admin/packages', name: 'adminPackages', component: PackageManagementView, meta: { title: '套餐管理', roles: ['admin'] } },
        { path: 'admin/settings', name: 'adminSettings', component: SettingsView, meta: { title: '系统设置', roles: ['admin'] } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const { ensureBootstrapped, currentUser, isAuthenticated } = useHealthData()
  await ensureBootstrapped()
  if (to.meta.public) {
    if (isAuthenticated.value && to.name === 'login') return homePath(currentUser.value.role)
    return true
  }
  if (!isAuthenticated.value) return '/login'
  const roles = to.meta.roles || []
  if (roles.length && !roles.includes(currentUser.value.role)) return homePath(currentUser.value.role)
  return true
})

export function homePath(role) {
  if (role === 'doctor') return '/doctor'
  if (role === 'admin') return '/admin'
  return '/'
}

export default router

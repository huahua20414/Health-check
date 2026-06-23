import { createRouter, createWebHashHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import DashboardView from '../views/DashboardView.vue'
import AppointmentCreateView from '../views/AppointmentCreateView.vue'
import MyAppointmentsView from '../views/MyAppointmentsView.vue'
import MyReportsView from '../views/MyReportsView.vue'
import DoctorAppointmentsView from '../views/DoctorAppointmentsView.vue'
import DoctorReportsView from '../views/DoctorReportsView.vue'
import PackageCatalogView from '../views/PackageCatalogView.vue'
import PackageManagementView from '../views/PackageManagementView.vue'
import PeopleView from '../views/PeopleView.vue'
import ProfileView from '../views/ProfileView.vue'
import FamilyMembersView from '../views/FamilyMembersView.vue'
import NotificationsView from '../views/NotificationsView.vue'
import DoctorReviewView from '../views/DoctorReviewView.vue'
import SettingsView from '../views/SettingsView.vue'
import AdminDashboardView from '../views/AdminDashboardView.vue'
import OperationsManagementView from '../views/OperationsManagementView.vue'
import ServiceResourceManagementView from '../views/ServiceResourceManagementView.vue'
import { useHealthData } from '../composables/useHealthData'

export const menuItems = [
  { path: '/', name: 'dashboard', label: '用户工作台', icon: 'House', roles: ['user'] },
  {
    label: '健康服务',
    icon: 'DataAnalysis',
    roles: ['user'],
    children: [
      { path: '/packages/catalog', name: 'packageCatalog', label: '体检套餐', icon: 'DataAnalysis', roles: ['user'] },
      { path: '/booking', name: 'booking', label: '预约体检', icon: 'Calendar', roles: ['user'] },
    ],
  },
  {
    label: '我的体检',
    icon: 'Tickets',
    roles: ['user'],
    children: [
      { path: '/my-appointments', name: 'myAppointments', label: '我的预约', icon: 'Tickets', roles: ['user'] },
      { path: '/my-reports', name: 'myReports', label: '我的报告', icon: 'DocumentChecked', roles: ['user'] },
    ],
  },
  {
    label: '个人中心',
    icon: 'User',
    roles: ['user'],
    children: [
      { path: '/profile', name: 'profile', label: '个人资料', icon: 'User', roles: ['user'] },
      { path: '/family-members', name: 'familyMembers', label: '家庭成员', icon: 'UserFilled', roles: ['user'] },
      { path: '/notifications', name: 'notifications', label: '消息与客服', icon: 'Bell', roles: ['user'] },
    ],
  },
  { path: '/doctor', name: 'doctorDashboard', label: '医生工作台', icon: 'House', roles: ['doctor'] },
  {
    label: '体检业务',
    icon: 'Files',
    roles: ['doctor'],
    children: [
      { path: '/appointments', name: 'appointments', label: '预约处理', icon: 'Files', roles: ['doctor'] },
      { path: '/reports', name: 'reports', label: '报告录入', icon: 'Document', roles: ['doctor'] },
    ],
  },
  {
    label: '档案查询',
    icon: 'User',
    roles: ['doctor'],
    children: [{ path: '/people', name: 'people', label: '客户档案', icon: 'User', roles: ['doctor'] }],
  },
  { path: '/admin', name: 'adminDashboard', label: '管理工作台', icon: 'House', roles: ['admin'] },
  {
    label: '用户与权限',
    icon: 'User',
    roles: ['admin'],
    children: [
      { path: '/admin/users', name: 'adminUsers', label: '用户管理', icon: 'User', roles: ['admin'] },
      { path: '/admin/doctors', name: 'doctorReview', label: '医生审核', icon: 'DocumentChecked', roles: ['admin'] },
    ],
  },
  {
    label: '体检服务管理',
    icon: 'DataAnalysis',
    roles: ['admin'],
    children: [
      { path: '/admin/packages', name: 'adminPackages', label: '套餐管理', icon: 'DataAnalysis', roles: ['admin'] },
      { path: '/admin/service-resources', name: 'serviceResources', label: '项目与排班', icon: 'Calendar', roles: ['admin'] },
    ],
  },
  {
    label: '系统管理',
    icon: 'Setting',
    roles: ['admin'],
    children: [
      { path: '/admin/operations', name: 'adminOperations', label: '运营管理', icon: 'Operation', roles: ['admin'] },
      { path: '/admin/settings', name: 'adminSettings', label: '系统设置', icon: 'Setting', roles: ['admin'] },
    ],
  },
]

export const routeMenuItems = menuItems.flatMap((item) => item.children || [item])

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
        { path: 'packages/catalog', name: 'packageCatalog', component: PackageCatalogView, meta: { title: '体检套餐', roles: ['user'] } },
        { path: 'booking', name: 'booking', component: AppointmentCreateView, meta: { title: '预约体检', roles: ['user'] } },
        { path: 'my-appointments', name: 'myAppointments', component: MyAppointmentsView, meta: { title: '我的预约', roles: ['user'] } },
        { path: 'my-reports', name: 'myReports', component: MyReportsView, meta: { title: '我的报告', roles: ['user'] } },
        { path: 'profile', name: 'profile', component: ProfileView, meta: { title: '个人资料', roles: ['user'] } },
        { path: 'family-members', name: 'familyMembers', component: FamilyMembersView, meta: { title: '家庭成员', roles: ['user'] } },
        { path: 'notifications', name: 'notifications', component: NotificationsView, meta: { title: '消息与客服', roles: ['user'] } },
        { path: 'doctor', name: 'doctorDashboard', component: DashboardView, meta: { title: '医生工作台', roles: ['doctor'] } },
        { path: 'appointments', name: 'appointments', component: DoctorAppointmentsView, meta: { title: '预约处理', roles: ['doctor'] } },
        { path: 'reports', name: 'reports', component: DoctorReportsView, meta: { title: '报告录入', roles: ['doctor'] } },
        { path: 'people', name: 'people', component: PeopleView, meta: { title: '客户档案', roles: ['doctor'] } },
        { path: 'admin', name: 'adminDashboard', component: AdminDashboardView, meta: { title: '管理工作台', roles: ['admin'] } },
        { path: 'admin/users', name: 'adminUsers', component: PeopleView, meta: { title: '用户管理', roles: ['admin'] } },
        { path: 'admin/doctors', name: 'doctorReview', component: DoctorReviewView, meta: { title: '医生审核', roles: ['admin'] } },
        { path: 'admin/packages', name: 'adminPackages', component: PackageManagementView, meta: { title: '套餐管理', roles: ['admin'] } },
        { path: 'admin/service-resources', name: 'serviceResources', component: ServiceResourceManagementView, meta: { title: '项目与排班', roles: ['admin'] } },
        { path: 'admin/operations', name: 'adminOperations', component: OperationsManagementView, meta: { title: '运营管理', roles: ['admin'] } },
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

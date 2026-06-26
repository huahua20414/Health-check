import React, { useEffect } from 'react'
import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { HealthProvider, useHealth } from './HealthContext.jsx'
import { AppShell } from './components/Shell.jsx'
import { AuthView, RegisterView } from './views/AuthViews.jsx'
import { DashboardView } from './views/DashboardView.jsx'
import { PackagesView } from './views/PackagesView.jsx'
import { BookingView } from './views/BookingView.jsx'
import { AppointmentsView } from './views/AppointmentsView.jsx'
import { ReportsView } from './views/ReportsView.jsx'
import { ProfileView } from './views/ProfileView.jsx'
import { FamilyView } from './views/FamilyView.jsx'
import { NotificationsView } from './views/NotificationsView.jsx'
import { DoctorAppointmentsView, DoctorReportsView, PeopleView } from './views/DoctorViews.jsx'
import { AdminDashboardView, AdminUsersView, DoctorReviewView, PackageManageView, ResourceManageView } from './views/AdminViews.jsx'
import { AdminCommunicationView, AdminEngagementView, AdminSystemView } from './views/AdminExtraViews.jsx'
import { homePath } from './utils.js'

function Guarded({ roles, children }) {
  const health = useHealth()
  if (!health.isAuthenticated) return <Navigate to="/login" replace />
  if (roles?.length && !roles.includes(health.role)) return <Navigate to={homePath(health.role)} replace />
  return children
}

function Bootstrap({ children }) {
  const health = useHealth()
  const location = useLocation()
  useEffect(() => { health.ensureBootstrapped().catch((error) => health.notify('error', error.message)) }, [])
  if (health.toast) {
    // rendering below; this branch intentionally keeps effect dependency lean
  }
  return (
    <>
      {children}
      {health.toast && <div className={`toast toast-${health.toast.type}`}>{health.toast.message}</div>}
      {location.pathname !== '/login' && location.pathname !== '/register/user' && location.pathname !== '/register/doctor' && health.loading.load && <div className="page-loader">数据加载中</div>}
    </>
  )
}

function AppRoutes() {
  return (
    <Bootstrap>
      <Routes>
        <Route path="/login" element={<AuthView />} />
        <Route path="/register/:role" element={<RegisterView />} />
        <Route element={<Guarded><AppShell /></Guarded>}>
          <Route index element={<Guarded roles={['user']}><DashboardView /></Guarded>} />
          <Route path="packages/catalog" element={<Guarded roles={['user']}><PackagesView /></Guarded>} />
          <Route path="booking" element={<Guarded roles={['user']}><BookingView /></Guarded>} />
          <Route path="my-appointments" element={<Guarded roles={['user']}><AppointmentsView /></Guarded>} />
          <Route path="my-reports" element={<Guarded roles={['user']}><ReportsView /></Guarded>} />
          <Route path="profile" element={<Guarded roles={['user']}><ProfileView /></Guarded>} />
          <Route path="family-members" element={<Guarded roles={['user']}><FamilyView /></Guarded>} />
          <Route path="notifications" element={<Guarded roles={['user']}><NotificationsView /></Guarded>} />
          <Route path="doctor" element={<Guarded roles={['doctor']}><DashboardView /></Guarded>} />
          <Route path="appointments" element={<Guarded roles={['doctor']}><DoctorAppointmentsView /></Guarded>} />
          <Route path="reports" element={<Guarded roles={['doctor']}><DoctorReportsView /></Guarded>} />
          <Route path="people" element={<Guarded roles={['doctor']}><PeopleView /></Guarded>} />
          <Route path="admin" element={<Guarded roles={['admin']}><AdminDashboardView /></Guarded>} />
          <Route path="admin/users" element={<Guarded roles={['admin']}><AdminUsersView /></Guarded>} />
          <Route path="admin/doctors" element={<Guarded roles={['admin']}><DoctorReviewView /></Guarded>} />
          <Route path="admin/packages" element={<Guarded roles={['admin']}><PackageManageView /></Guarded>} />
          <Route path="admin/service-resources" element={<Guarded roles={['admin']}><ResourceManageView /></Guarded>} />
          <Route path="admin/engagement" element={<Guarded roles={['admin']}><AdminEngagementView /></Guarded>} />
          <Route path="admin/communication" element={<Guarded roles={['admin']}><AdminCommunicationView /></Guarded>} />
          <Route path="admin/system" element={<Guarded roles={['admin']}><AdminSystemView /></Guarded>} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Bootstrap>
  )
}

export default function App() {
  return (
    <HealthProvider>
      <AppRoutes />
    </HealthProvider>
  )
}

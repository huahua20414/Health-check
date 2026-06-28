import React, { useEffect, useState } from 'react'
import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { HealthProvider, useHealth } from './HealthContext.jsx'
import { AppShell } from './components/Shell.jsx'
import { Button, Modal } from './components/UI.jsx'
import { AuthView, RegisterView } from './views/AuthViews.jsx'
import { DashboardView } from './views/DashboardView.jsx'
import { PackagesView } from './views/PackagesView.jsx'
import { BookingView } from './views/BookingView.jsx'
import { AppointmentsView } from './views/AppointmentsView.jsx'
import { ReportsView } from './views/ReportsView.jsx'
import { ProfileView } from './views/ProfileView.jsx'
import { FamilyView } from './views/FamilyView.jsx'
import { NotificationsView } from './views/NotificationsView.jsx'
import { DoctorAppointmentsView, DoctorReportsView, DoctorScheduleView, PeopleView } from './views/DoctorViews.jsx'
import { AdminDashboardView, AdminUsersView, DoctorReviewView, PackageManageView, ResourceManageView } from './views/AdminViews.jsx'
import { AdminCommunicationView, AdminEngagementView, AdminSystemView } from './views/AdminExtraViews.jsx'
import { formatDate, homePath } from './utils.js'

function Guarded({ roles, children }) {
  const health = useHealth()
  if (!health.isAuthenticated) return <Navigate to="/login" replace />
  if (roles?.length && !roles.includes(health.role)) return <Navigate to={homePath(health.role)} replace />
  return children
}

function Bootstrap({ children }) {
  const health = useHealth()
  const location = useLocation()
  const [announcement, setAnnouncement] = useState(null)
  useEffect(() => { health.ensureBootstrapped().catch((error) => health.notify('error', error.message)) }, [])
  useEffect(() => {
    if (!['user', 'doctor'].includes(health.role)) {
      setAnnouncement(null)
      return
    }
    const dismissed = JSON.parse(localStorage.getItem('dismissedAnnouncementIds') || '[]').map(String)
    const next = (health.activeAnnouncements || []).find((item) => !dismissed.includes(String(item.id)))
    setAnnouncement(next || null)
  }, [health.role, health.activeAnnouncements])
  const closeAnnouncement = () => {
    if (announcement?.id) {
      const dismissed = new Set(JSON.parse(localStorage.getItem('dismissedAnnouncementIds') || '[]').map(String))
      dismissed.add(String(announcement.id))
      localStorage.setItem('dismissedAnnouncementIds', JSON.stringify(Array.from(dismissed)))
    }
    setAnnouncement(null)
  }
  if (health.toast) {
    // rendering below; this branch intentionally keeps effect dependency lean
  }
  return (
    <>
      {children}
      {health.toast && <div className={`toast toast-${health.toast.type}`}><span>{health.toast.message}</span><button type="button" onClick={health.dismissToast} aria-label="关闭通知">×</button></div>}
      <AnnouncementPopup announcement={announcement} onClose={closeAnnouncement} />
      {location.pathname !== '/login' && location.pathname !== '/register/user' && location.pathname !== '/register/doctor' && health.loading.load && <div className="page-loader">数据加载中</div>}
    </>
  )
}

function AnnouncementPopup({ announcement, onClose }) {
  if (!announcement) return null
  const publishedAt = formatDate(announcement.publishedAt || announcement.createdAt)
  return (
    <Modal open title="系统公告" onClose={onClose} className="announcement-modal" backdropClassName="announcement-backdrop" actions={<Button size="lg" onClick={onClose}>知道了</Button>}>
      <div className="announcement-popup">
        <div className="announcement-kicker">
          <span>Health Checkup Notice</span>
          <time>{publishedAt}</time>
        </div>
        <h2>{announcement.title}</h2>
        <div className="announcement-content">{announcement.content}</div>
      </div>
    </Modal>
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
          <Route path="doctor/schedule" element={<Guarded roles={['doctor']}><DoctorScheduleView /></Guarded>} />
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

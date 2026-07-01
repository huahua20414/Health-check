import React, { useEffect, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { Bell, Bot, Calendar, ChevronDown, ChevronRight, ClipboardList, Database, FileText, Home, LogOut, Megaphone, Moon, RefreshCcw, Settings, ShieldCheck, Sun, User, Users } from 'lucide-react'
import { Button } from './UI.jsx'
import { useHealth } from '../HealthContext.jsx'

const iconMap = { Home, Database, Calendar, ClipboardList, FileText, User, Users, Bell, ShieldCheck, Megaphone, Settings, Bot }

export const menuItems = [
  { path: '/', label: '工作台', icon: 'Home', roles: ['user'] },
  { label: '健康服务', icon: 'Database', roles: ['user'], children: [
    { path: '/packages/catalog', label: '体检套餐', icon: 'Database', roles: ['user'] },
    { path: '/booking', label: '预约体检', icon: 'Calendar', roles: ['user'] },
  ] },
  { label: '我的体检', icon: 'ClipboardList', roles: ['user'], children: [
    { path: '/my-appointments', label: '我的预约', icon: 'ClipboardList', roles: ['user'] },
    { path: '/my-reports', label: '我的报告', icon: 'FileText', roles: ['user'] },
  ] },
  { label: '个人中心', icon: 'User', roles: ['user'], children: [
    { path: '/profile', label: '个人资料', icon: 'User', roles: ['user'] },
    { path: '/family-members', label: '家庭成员', icon: 'Users', roles: ['user'] },
    { path: '/notifications', label: '消息与客服', icon: 'Bell', roles: ['user'] },
  ] },
  { path: '/ai-assistant', label: 'AI 助手', icon: 'Bot', roles: ['user', 'doctor', 'admin'] },
  { path: '/doctor', label: '医生工作台', icon: 'Home', roles: ['doctor'] },
  { label: '体检业务', icon: 'ClipboardList', roles: ['doctor'], children: [
    { path: '/doctor/schedule', label: '我的排班', icon: 'Calendar', roles: ['doctor'] },
    { path: '/appointments', label: '预约处理', icon: 'ClipboardList', roles: ['doctor'] },
    { path: '/reports', label: '报告录入', icon: 'FileText', roles: ['doctor'] },
  ] },
  { label: '档案查询', icon: 'User', roles: ['doctor'], children: [{ path: '/people', label: '客户档案', icon: 'User', roles: ['doctor'] }] },
  { path: '/admin', label: '管理工作台', icon: 'Home', roles: ['admin'] },
  { label: '用户与权限', icon: 'Users', roles: ['admin'], children: [
    { path: '/admin/users', label: '用户管理', icon: 'Users', roles: ['admin'] },
    { path: '/admin/doctors', label: '医生审核', icon: 'ShieldCheck', roles: ['admin'] },
  ] },
  { label: '体检服务管理', icon: 'Database', roles: ['admin'], children: [
    { path: '/admin/packages', label: '套餐管理', icon: 'Database', roles: ['admin'] },
    { path: '/admin/service-resources', label: '项目与排班', icon: 'Calendar', roles: ['admin'] },
  ] },
  { label: '运营支持', icon: 'Megaphone', roles: ['admin'], children: [
    { path: '/admin/engagement', label: '营销与公告', icon: 'Megaphone', roles: ['admin'] },
    { path: '/admin/communication', label: '公告与客服', icon: 'Bell', roles: ['admin'] },
  ] },
  { label: '系统治理', icon: 'Settings', roles: ['admin'], children: [
    { path: '/admin/system', label: '日志与设置', icon: 'Settings', roles: ['admin'] },
  ] },
]

function Icon({ name }) {
  const Component = iconMap[name] || Home
  return <Component size={16} strokeWidth={2.2} />
}

function MenuLink({ item }) {
  return <NavLink to={item.path} end className={({ isActive }) => `side-link ${isActive ? 'is-active' : ''}`}><Icon name={item.icon} /><span>{item.label}</span></NavLink>
}

export function AppShell() {
  const health = useHealth()
  const navigate = useNavigate()
  const [closedGroups, setClosedGroups] = useState({})
  const [theme, setTheme] = useState(() => localStorage.getItem('theme') || 'dark')
  const visible = menuItems.filter((item) => item.roles.includes(health.role)).map((item) => item.children ? { ...item, children: item.children.filter((child) => child.roles.includes(health.role)) } : item)
  useEffect(() => {
    document.body.dataset.theme = theme
    localStorage.setItem('theme', theme)
  }, [theme])
  function toggleGroup(label) {
    setClosedGroups((current) => ({ ...current, [label]: !current[label] }))
  }
  function toggleTheme() {
    setTheme((current) => current === 'dark' ? 'light' : 'dark')
  }
  async function handleLogout() {
    await health.logout()
    navigate('/login')
  }
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand"><div className="brand-code">XIXIN</div><strong>健康体检管理</strong><span>{roleLabel(health.role)}</span></div>
        <nav className="side-nav">
          {visible.map((item) => item.children ? (
            <div className="side-group" key={item.label}>
              <button className="side-group-title" type="button" onClick={() => toggleGroup(item.label)}>
                <span><Icon name={item.icon} />{item.label}</span>
                {closedGroups[item.label] ? <ChevronRight size={14} /> : <ChevronDown size={14} />}
              </button>
              {!closedGroups[item.label] && <div className="side-children">{item.children.map((child) => <MenuLink key={child.path} item={child} />)}</div>}
            </div>
          ) : <MenuLink key={item.path} item={item} />)}
        </nav>
        <div className="side-foot"><span>{health.currentUser?.name || '-'}</span><small>{health.currentUser?.email || ''}</small></div>
      </aside>
      <section className="main-panel">
        <header className="topbar">
          <div><span className="eyebrow">熙心健康体检管理系统</span><h2>{roleLabel(health.role)}</h2></div>
          <div className="top-actions"><span className="role-pill">{roleLabel(health.role)}</span><Button className="theme-toggle" variant="ghost" title={theme === 'dark' ? '切换到亮色' : '切换到暗色'} aria-label={theme === 'dark' ? '切换到亮色' : '切换到暗色'} onClick={toggleTheme}>{theme === 'dark' ? <Sun size={15} /> : <Moon size={15} />}</Button><Button variant="ghost" loading={health.loading.load} onClick={health.loadAll}><RefreshCcw size={15} />刷新</Button><Button variant="danger" loading={health.loading.logout} onClick={handleLogout}><LogOut size={15} />退出</Button></div>
        </header>
        <main className="workspace"><Outlet /></main>
      </section>
    </div>
  )
}

function roleLabel(role) {
  return { user: '用户端', doctor: '医生端', admin: '管理员端' }[role] || '工作台'
}

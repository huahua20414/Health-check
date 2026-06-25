import React, { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { Bell, Calendar, ChevronDown, ChevronRight, ClipboardList, Database, FileText, Home, LogOut, RefreshCcw, ShieldCheck, User, Users } from 'lucide-react'
import { Button } from './UI.jsx'
import { useHealth } from '../HealthContext.jsx'

const iconMap = { Home, Database, Calendar, ClipboardList, FileText, User, Users, Bell, ShieldCheck }

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
  { path: '/doctor', label: '医生工作台', icon: 'Home', roles: ['doctor'] },
  { label: '体检业务', icon: 'ClipboardList', roles: ['doctor'], children: [
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
]

function Icon({ name }) {
  const Component = iconMap[name] || Home
  return <Component size={16} strokeWidth={2.2} />
}

function MenuLink({ item }) {
  return <NavLink to={item.path} end={item.path === '/'} className={({ isActive }) => `side-link ${isActive ? 'is-active' : ''}`}><Icon name={item.icon} /><span>{item.label}</span></NavLink>
}

export function AppShell() {
  const health = useHealth()
  const navigate = useNavigate()
  const [closedGroups, setClosedGroups] = useState({})
  const visible = menuItems.filter((item) => item.roles.includes(health.role)).map((item) => item.children ? { ...item, children: item.children.filter((child) => child.roles.includes(health.role)) } : item)
  function toggleGroup(label) {
    setClosedGroups((current) => ({ ...current, [label]: !current[label] }))
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
          <div className="top-actions"><span className="role-pill">{roleLabel(health.role)}</span><Button variant="ghost" loading={health.loading.load} onClick={health.loadAll}><RefreshCcw size={15} />刷新</Button><Button variant="danger" loading={health.loading.logout} onClick={handleLogout}><LogOut size={15} />退出</Button></div>
        </header>
        <main className="workspace"><Outlet /></main>
      </section>
    </div>
  )
}

function roleLabel(role) {
  return { user: '用户端', doctor: '医生端', admin: '管理员端' }[role] || '工作台'
}

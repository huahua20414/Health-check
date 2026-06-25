import React from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, Card, DataTable, Metric, PageHeader, StatusTag } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { moneyText, statusText } from '../utils'

export function DashboardView() {
  const h = useHealth()
  const navigate = useNavigate()
  if (h.role === 'doctor') return <DoctorDashboard h={h} />
  if (h.role === 'admin') return <AdminLiteDashboard h={h} />
  return (
    <>
      <PageHeader title="用户工作台" subtitle="套餐、预约、候补、报告、通知集中入口。" />
      <div className="metrics-grid">
        <Metric label="最近预约" value={h.myAppointments.length} />
        <Metric label="候补中" value={h.waitlist.length} tone="violet" />
        <Metric label="待支付" value={h.appointments.filter((a) => a.paymentStatus === 'unpaid').length} tone="amber" />
        <Metric label="新报告" value={h.reports.length} tone="green" />
      </div>
      <div className="two-col">
        <Card title="推荐套餐">
          <div className="stack-list">{(h.recommendedPackages.length ? h.recommendedPackages : h.packages).slice(0, 4).map((pkg) => <div className="accent-row" key={pkg.id}><div><strong>{pkg.name}</strong><span>{pkg.description || pkg.items}</span></div><b>{moneyText(pkg.price)}</b></div>)}</div>
        </Card>
        <Card title="我的预约与报告">
          <DataTable columns={[
            { title: '项目', render: (r) => r.package?.name || r.appointmentType },
            { title: '机构', render: (r) => r.institution?.name || '-' },
            { title: '状态', render: (r) => <StatusTag status={r.status} /> },
            { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => navigate(r.status === 'reported' ? '/my-reports' : '/my-appointments')}>{r.status === 'reported' ? '查看报告' : '处理预约'}</Button> },
          ]} rows={h.myAppointments.slice(0, 5)} />
        </Card>
      </div>
    </>
  )
}

function DoctorDashboard({ h }) {
  return (
    <>
      <PageHeader title="医生工作台" subtitle="预约处理、状态确认、报告录入、客户档案查询。" />
      <div className="metrics-grid"><Metric label="今日待处理" value={h.pendingDoctorCount} /><Metric label="待生成报告" value={h.appointments.filter((a) => a.status === 'checked').length} tone="amber" /><Metric label="已完成" value={h.reportedCount} tone="green" /></div>
      <Card title="预约处理队列"><DataTable columns={[{ title: '客户', render: (r) => r.user?.name || '-' }, { title: '套餐', render: (r) => r.package?.name || r.appointmentType }, { title: '时间', render: (r) => `${r.date || ''} ${r.startTime || ''}` }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }]} rows={h.appointments.slice(0, 8)} /></Card>
    </>
  )
}

function AdminLiteDashboard({ h }) {
  const summary = h.adminDashboard?.summary || {}
  return (
    <>
      <PageHeader title="管理员工作台" subtitle="审核、用户、机构、套餐、号源和运营数据。" />
      <div className="metrics-grid"><Metric label="待审核医生" value={h.pendingDoctors.length || summary.pendingDoctors || 0} tone="amber" /><Metric label="机构数量" value={h.institutions.length} /><Metric label="邮件失败" value={h.mailLogs.filter((m) => m.status === 'failed').length} tone="red" /><Metric label="操作日志" value={summary.operationLogs || h.operationLogs.length} tone="violet" /></div>
      <Card title="医生审核"><DataTable columns={[{ title: '医生', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }]} rows={h.pendingDoctors.slice(0, 8)} /></Card>
    </>
  )
}

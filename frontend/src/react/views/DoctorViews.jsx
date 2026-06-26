import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, Card, Field, Modal, PageHeader, PaginatedTable, Select, StatusTag, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'

export function DoctorAppointmentsView() {
  const h = useHealth()
  const navigate = useNavigate()
  useEffect(() => {
    h.loadAppointmentsPage({ page: 1, pageSize: 20 }).catch((e) => h.notify('error', e.message))
  }, [])
  const markDone = (row) => h.markDone(row).then(() => navigate('/reports?draft=1')).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="预约处理" subtitle="医生查看预约并更新体检状态。" />
      <Card title="预约列表"><PaginatedTable columns={[
        { title: '客户', render: (r) => r.user?.name || '-' },
        { title: '套餐', render: (r) => r.package?.name || r.appointmentType },
        { title: '时间', render: (r) => `${r.date || ''} ${r.startTime || ''}` },
        { title: '状态', render: (r) => <StatusTag status={r.status} /> },
        { title: '操作', render: (r) => <DoctorAppointmentActions row={r} h={h} navigate={navigate} markDone={markDone} /> },
      ]} rows={h.appointments} /></Card>
    </>
  )
}

function DoctorAppointmentActions({ row, h, navigate, markDone }) {
  if (row.status === 'booked') {
    return <Button size="sm" loading={h.loading.status} onClick={() => markDone(row)}>标记完成</Button>
  }
  if (row.status === 'checked') {
    return <Button size="sm" variant="ghost" onClick={() => { h.updateForm('report', { appointmentId: row.id }); navigate('/reports?draft=1') }}>录入报告</Button>
  }
  return <span className="muted-text">无可用操作</span>
}

export function DoctorReportsView() {
  const h = useHealth()
  const [open, setOpen] = useState(false)
  const draftRequested = new URLSearchParams(window.location.search).get('draft') === '1'
  const reportableAppointments = h.appointments.filter((appointment) => appointment.status === 'checked' || appointment.status === 'reported')
  useEffect(() => {
    h.loadReportsPage({ page: 1, pageSize: 20 }).catch((e) => h.notify('error', e.message))
    h.loadAppointmentsPage({ status: 'checked', page: 1, pageSize: 50 }).catch((e) => h.notify('error', e.message))
  }, [])
  useEffect(() => {
    if (draftRequested && h.forms.report.appointmentId) setOpen(true)
  }, [draftRequested, h.forms.report.appointmentId])
  const openCreate = () => { h.resetForm('report'); setOpen(true) }
  const save = () => h.createReport().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="报告录入" subtitle="医生生成或更新体检报告。" actions={<Button onClick={openCreate}>录入报告</Button>} />
      <Card title="已生成报告"><PaginatedTable columns={[{ title: '编号', render: (r) => r.reportNo || r.id }, { title: '客户', render: (r) => r.user?.name || '-' }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => h.downloadReport(r)}>下载</Button> }]} rows={h.reports} /></Card>
      <Modal open={open} title="报告录入" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.report} onClick={save}>保存报告</Button></>}>
        <Field label="预约"><Select value={h.forms.report.appointmentId} onChange={(e) => h.updateForm('report', { appointmentId: e.target.value })}><option value="">请选择已体检预约</option>{reportableAppointments.map((a) => <option key={a.id} value={a.id}>{a.user?.name || '客户'} · {a.package?.name || a.appointmentType}</option>)}</Select></Field>
        <Field label="检查摘要"><Textarea value={h.forms.report.summary} onChange={(e) => h.updateForm('report', { summary: e.target.value })} /></Field>
        <Field label="体检结论"><Textarea value={h.forms.report.conclusion} onChange={(e) => h.updateForm('report', { conclusion: e.target.value })} /></Field>
        <Field label="健康建议"><Textarea value={h.forms.report.recommendation} onChange={(e) => h.updateForm('report', { recommendation: e.target.value })} /></Field>
      </Modal>
    </>
  )
}

export function PeopleView({ admin = false }) {
  const h = useHealth()
  const rows = admin ? h.users : h.peopleRows
  useEffect(() => {
    if (!admin) {
      h.loadAppointmentsPage({ page: 1, pageSize: 50 }).catch((e) => h.notify('error', e.message))
      h.loadReportsPage({ page: 1, pageSize: 50 }).catch((e) => h.notify('error', e.message))
    }
  }, [admin])
  return (
    <>
      <PageHeader title={admin ? '用户管理' : '客户档案'} subtitle={admin ? '管理员管理用户与账号状态。' : '医生查看预约和报告相关客户档案。'} />
      <Card title="人员列表"><PaginatedTable columns={[{ title: '姓名', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '角色', render: (r) => r.role || r.source }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => admin ? <UserStatusActions row={r} h={h} /> : '-' }]} rows={rows} /></Card>
    </>
  )
}

function UserStatusActions({ row, h }) {
  return (
    <div className="row-actions">
      {row.status !== 'active' && <Button size="sm" variant="ghost" loading={h.loading.status} onClick={() => h.updateUserStatus(row, 'active')}>启用</Button>}
      {row.status !== 'disabled' && <Button size="sm" variant="danger" loading={h.loading.status} onClick={() => h.updateUserStatus(row, 'disabled')}>停用</Button>}
      {row.status === 'active' && <span className="muted-text">已启用</span>}
    </div>
  )
}

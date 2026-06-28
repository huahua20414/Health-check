import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CalendarDays, Clock3, MapPin, Stethoscope } from 'lucide-react'
import { Button, Card, Empty, Field, Metric, Modal, PageHeader, PaginatedTable, Select, StatusTag, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { formatDate } from '../utils'

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
        { title: '时间', render: (r) => `${formatDate(r.date)} ${r.startTime || ''}` },
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

export function DoctorScheduleView() {
  const h = useHealth()
  const doctorId = h.currentUser?.id
  useEffect(() => {
    if (!doctorId) return
    h.loadSlotsPage({ doctorId, fromDate: formatDate(new Date()), page: 1, pageSize: 100 }, 'slots', 'scheduleSlotRows').catch((e) => h.notify('error', e.message))
  }, [doctorId])

  const rows = (h.scheduleSlotRows || [])
    .filter((slot) => Number(slot.doctorId) === Number(doctorId) && slot.status !== 'deleted')
    .sort((a, b) => `${a.date} ${a.startTime}`.localeCompare(`${b.date} ${b.startTime}`))

  const groups = groupDoctorSchedule(rows)
  const today = formatDate(new Date())
  const thisWeekEnd = addDays(today, 6)
  const todayCount = rows.filter((slot) => slot.date === today).length
  const thisWeekCount = rows.filter((slot) => slot.date >= today && slot.date <= thisWeekEnd).length
  const bookedCount = rows.reduce((sum, slot) => sum + Number(slot.bookedCount || 0), 0)
  const remainingCount = rows.reduce((sum, slot) => sum + Math.max(0, Number(slot.capacity || 0) - Number(slot.bookedCount || 0)), 0)

  return (
    <>
      <PageHeader title="我的排班" subtitle="查看自己接下来在哪家机构上班、什么时间出诊，仅展示不编辑。" />
      <div className="metrics-grid doctor-schedule-metrics">
        <Metric label="未来班次数" value={rows.length} tone="cyan" />
        <Metric label="今日班次" value={todayCount} tone="green" />
        <Metric label="本周班次" value={thisWeekCount} tone="violet" />
        <Metric label="剩余可约/已约" value={`${remainingCount}/${bookedCount}`} tone="amber" />
      </div>
      <Card title="近期出诊安排" subtitle="按日期查看机构、时间段、分类和当前预约情况。">
        {!rows.length && <Empty text={h.loading.slots ? '正在加载排班...' : '最近暂无排班'} />}
        {!!rows.length && <div className="doctor-schedule-list">
          {groups.map((group) => <section className="doctor-schedule-day" key={group.date}>
            <header className="doctor-schedule-day-head">
              <div>
                <strong>{group.date}</strong>
                <span>{weekdayText(group.date)} · {group.items.length} 个班次</span>
              </div>
              <span className="doctor-schedule-day-total">总容量 {group.items.reduce((sum, item) => sum + Number(item.capacity || 0), 0)}</span>
            </header>
            <div className="doctor-schedule-grid">
              {group.items.map((slot) => {
                const remaining = Math.max(0, Number(slot.capacity || 0) - Number(slot.bookedCount || 0))
                return (
                  <article className="doctor-schedule-slot" key={slot.id}>
                    <div className="doctor-schedule-slot-top">
                      <div className="doctor-schedule-slot-time"><Clock3 size={16} />{slot.startTime}-{slot.endTime}</div>
                      <StatusTag status={slot.status}>{slot.status === 'available' ? '可出诊' : undefined}</StatusTag>
                    </div>
                    <div className="doctor-schedule-slot-main">
                      <p><MapPin size={15} />{slot.institution?.name || '未设置机构'}</p>
                      <p><Stethoscope size={15} />{slot.category || '未设置分类'}</p>
                      <p><CalendarDays size={15} />{slot.period || timePeriod(slot.startTime)}</p>
                    </div>
                    <div className="doctor-schedule-slot-foot">
                      <span>已约 {slot.bookedCount || 0} / 容量 {slot.capacity || 0}</span>
                      <strong>{remaining > 0 ? `剩余 ${remaining}` : '已满'}</strong>
                    </div>
                  </article>
                )
              })}
            </div>
          </section>)}
        </div>}
      </Card>
    </>
  )
}

function groupDoctorSchedule(rows) {
  const grouped = new Map()
  rows.forEach((row) => {
    if (!grouped.has(row.date)) grouped.set(row.date, [])
    grouped.get(row.date).push(row)
  })
  return Array.from(grouped.entries()).map(([date, items]) => ({
    date,
    items: items.sort((a, b) => `${a.startTime}-${a.id}`.localeCompare(`${b.startTime}-${b.id}`)),
  }))
}

function weekdayText(value) {
  const date = new Date(`${value}T00:00:00`)
  if (Number.isNaN(date.getTime())) return '-'
  return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()]
}

function addDays(value, days) {
  const date = new Date(`${value}T00:00:00`)
  if (Number.isNaN(date.getTime())) return value
  date.setDate(date.getDate() + days)
  return formatDate(date)
}

function timePeriod(startTime) {
  const hour = Number(String(startTime || '').split(':')[0] || 0)
  return hour < 12 ? '上午' : '下午'
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

import React, { useEffect } from 'react'
import { Button, Card, Field, Metric, PageHeader, PaginatedTable, Select, StatusTag, TextInput, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { doctorDepartments, moneyText } from '../utils'

export function AdminDashboardView() {
  const h = useHealth()
  const summary = h.adminDashboard?.summary || {}
  return (
    <>
      <PageHeader title="管理员工作台" subtitle="审核、用户、机构、套餐、号源和运营数据。" />
      <div className="metrics-grid"><Metric label="待审核医生" value={h.pendingDoctors.length || summary.pendingDoctors || 0} tone="amber" /><Metric label="机构数量" value={h.institutions.length} /><Metric label="套餐数量" value={h.packages.length} tone="green" /><Metric label="邮件日志" value={h.mailLogs.length} tone="violet" /></div>
      <Card title="待审核医生"><PaginatedTable columns={[{ title: '医生', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" onClick={() => h.updateUserStatus(r, 'active')}>通过</Button><Button size="sm" variant="danger" onClick={() => h.updateUserStatus(r, 'disabled')}>拒绝</Button></div> }]} rows={h.pendingDoctors} /></Card>
    </>
  )
}

export function DoctorReviewView() {
  const h = useHealth()
  return (
    <>
      <PageHeader title="医生审核" subtitle="审核医生账号，并维护科室、职称与专长。" />
      <Card title="医生列表"><PaginatedTable columns={[{ title: '姓名', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '科室', render: (r) => r.doctorProfile?.department || r.department || '-' }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" onClick={() => h.updateUserStatus(r, 'active')}>通过</Button><Button size="sm" variant="danger" onClick={() => h.updateUserStatus(r, 'disabled')}>停用</Button></div> }]} rows={h.users.filter((u) => u.role === 'doctor')} /></Card>
    </>
  )
}

export function PackageManageView() {
  const h = useHealth()
  const f = h.forms.package
  return (
    <>
      <PageHeader title="套餐管理" subtitle="套餐 CRUD、导入导出和状态管理。" actions={<Button variant="ghost" onClick={() => h.exportBlob('/packages/export', 'packages.csv', 'exportPackages')}>导出</Button>} />
      <div className="two-col">
        <Card title="套餐列表"><PaginatedTable columns={[{ title: '名称', render: (r) => r.name }, { title: '分类', render: (r) => r.category }, { title: '价格', render: (r) => moneyText(r.price) }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => h.updateForm('package', r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archivePackage(r)}>归档</Button></div> }]} rows={h.packages} /></Card>
        <Card title={f.id ? '编辑套餐' : '新增套餐'}>
          <Field label="名称"><TextInput value={f.name} onChange={(e) => h.updateForm('package', { name: e.target.value })} /></Field>
          <Field label="分类"><TextInput value={f.category} onChange={(e) => h.updateForm('package', { category: e.target.value })} /></Field>
          <Field label="价格"><TextInput type="number" value={f.price} onChange={(e) => h.updateForm('package', { price: e.target.value })} /></Field>
          <Field label="项目明细"><Textarea value={f.items} onChange={(e) => h.updateForm('package', { items: e.target.value })} /></Field>
          <Button loading={h.loading.package} onClick={() => h.savePackage().catch((e) => h.notify('error', e.message))}>保存套餐</Button>
        </Card>
      </div>
    </>
  )
}

export function ResourceManageView() {
  const h = useHealth()
  useEffect(() => { h.loadInstitutionsPage(); h.loadCheckupItemsPage(); h.loadPackageItemsPage(); h.loadSlotsPage() }, [])
  return (
    <>
      <PageHeader title="项目与排班" subtitle="机构、体检项目、套餐项目组合和医生号源。" />
      <div className="management-grid">
        <InstitutionPanel h={h} />
        <CheckupItemPanel h={h} />
        <PackageItemPanel h={h} />
        <SchedulePanel h={h} />
      </div>
    </>
  )
}

function InstitutionPanel({ h }) {
  const f = h.forms.institution
  return <Card title="机构管理" actions={<Button size="sm" variant="ghost" onClick={() => h.exportBlob('/institutions/export', 'institutions.csv', 'exportInstitutions')}>导出</Button>}><PaginatedTable columns={[{ title: '名称', render: (r) => r.name }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => h.updateForm('institution', r)}>编辑</Button> }]} rows={h.institutionRows.length ? h.institutionRows : h.institutions} /><div className="mini-form"><TextInput placeholder="机构名称" value={f.name} onChange={(e) => h.updateForm('institution', { name: e.target.value })} /><TextInput placeholder="地址" value={f.address} onChange={(e) => h.updateForm('institution', { address: e.target.value })} /><Button onClick={() => h.saveInstitution()}>保存</Button></div></Card>
}

function CheckupItemPanel({ h }) {
  const f = h.forms.checkupItem
  return <Card title="体检项目"><PaginatedTable columns={[{ title: '名称', render: (r) => r.name }, { title: '科室', render: (r) => r.department }, { title: '价格', render: (r) => moneyText(r.price) }]} rows={h.checkupItemRows.length ? h.checkupItemRows : h.checkupItems} /><div className="mini-form"><TextInput placeholder="项目名称" value={f.name} onChange={(e) => h.updateForm('checkupItem', { name: e.target.value })} /><TextInput placeholder="科室" value={f.department} onChange={(e) => h.updateForm('checkupItem', { department: e.target.value })} /><Button onClick={() => h.saveCheckupItem()}>保存</Button></div></Card>
}

function PackageItemPanel({ h }) {
  const f = h.forms.packageItem
  return <Card title="套餐项目组合"><PaginatedTable columns={[{ title: '套餐', render: (r) => r.package?.name || r.packageId }, { title: '项目', render: (r) => r.item?.name || r.itemId }, { title: '排序', render: (r) => r.sortOrder }]} rows={h.packageItems} /><div className="mini-form"><Select value={f.packageId} onChange={(e) => h.updateForm('packageItem', { packageId: e.target.value })}><option value="">套餐</option>{h.packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</Select><Select value={f.itemId} onChange={(e) => h.updateForm('packageItem', { itemId: e.target.value })}><option value="">项目</option>{h.checkupItems.map((i) => <option key={i.id} value={i.id}>{i.name}</option>)}</Select><Button onClick={() => h.savePackageItem()}>添加</Button></div></Card>
}

function SchedulePanel({ h }) {
  const f = h.forms.schedule
  const rows = h.scheduleSlotRows.length ? h.scheduleSlotRows : h.slots
  const doctors = h.users.filter((u) => u.role === 'doctor' && u.status === 'active')
  const categories = [...new Set(h.packages.map((p) => p.category).filter(Boolean))]
  const editSlot = (slot) => h.updateForm('schedule', {
    id: slot.id,
    doctorId: slot.doctorId,
    institutionId: slot.institutionId,
    date: slot.date,
    period: slot.period || '上午',
    category: slot.category || '',
    startTime: slot.startTime || '08:30',
    endTime: slot.endTime || '',
    capacity: slot.capacity || 1,
    status: slot.status || 'available',
  })
  return <Card title="医生号源"><PaginatedTable columns={[
    { title: '医生', render: (r) => r.doctor?.name || r.doctorId },
    { title: '机构', render: (r) => r.institution?.name || r.institutionId },
    { title: '日期', render: (r) => r.date },
    { title: '时段', render: (r) => `${r.startTime || ''}-${r.endTime || ''}` },
    { title: '分类', render: (r) => r.category || '-' },
    { title: '余号', render: (r) => `${Math.max(0, Number(r.capacity || 0) - Number(r.bookedCount || 0))}/${r.capacity || 0}` },
    { title: '状态', render: (r) => <StatusTag status={r.status} /> },
    { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => editSlot(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveScheduleSlot(r)}>归档</Button></div> },
  ]} rows={rows} />
    <div className="mini-form schedule-form">
      <Select value={f.doctorId} onChange={(e) => h.updateForm('schedule', { doctorId: e.target.value })}><option value="">医生</option>{doctors.map((u) => <option key={u.id} value={u.id}>{u.name}</option>)}</Select>
      <Select value={f.institutionId} onChange={(e) => h.updateForm('schedule', { institutionId: e.target.value })}><option value="">机构</option>{h.institutions.map((i) => <option key={i.id} value={i.id}>{i.name}</option>)}</Select>
      <Select value={f.category} onChange={(e) => h.updateForm('schedule', { category: e.target.value })}><option value="">分类</option>{categories.map((category) => <option key={category} value={category}>{category}</option>)}</Select>
      <TextInput type="date" value={f.date} onChange={(e) => h.updateForm('schedule', { date: e.target.value })} />
      <TextInput placeholder="开始 08:30" value={f.startTime} onChange={(e) => h.updateForm('schedule', { startTime: e.target.value })} />
      <TextInput placeholder="结束 09:00" value={f.endTime} onChange={(e) => h.updateForm('schedule', { endTime: e.target.value })} />
      <Select value={f.period} onChange={(e) => h.updateForm('schedule', { period: e.target.value })}><option value="上午">上午</option><option value="下午">下午</option></Select>
      <TextInput type="number" min="1" value={f.capacity} onChange={(e) => h.updateForm('schedule', { capacity: e.target.value })} />
      <Select value={f.status} onChange={(e) => h.updateForm('schedule', { status: e.target.value })}><option value="available">可预约</option><option value="disabled">停用</option></Select>
      <div className="row-actions"><Button onClick={() => h.saveScheduleSlot()}>{f.id ? '保存编辑' : '新增号源'}</Button>{f.id && <Button variant="ghost" onClick={() => h.resetForm('schedule')}>取消编辑</Button>}</div>
    </div>
  </Card>
}

export function OperationsView() {
  const h = useHealth()
  useEffect(() => { h.loadCouponsPage(); h.loadReviewsPage(); h.loadAnnouncementsPage(); h.loadAdminNotificationsPage(); h.loadAdminSupportTicketsPage() }, [])
  return (
    <>
      <PageHeader title="运营管理" subtitle="优惠券、评价、公告、通知、客服工单。" />
      <div className="management-grid">
        <Card title="优惠券"><PaginatedTable columns={[{ title: '名称', render: (r) => r.name }, { title: '编码', render: (r) => r.code }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }]} rows={h.coupons} /><div className="mini-form"><TextInput placeholder="名称" value={h.forms.coupon.name} onChange={(e) => h.updateForm('coupon', { name: e.target.value })} /><TextInput placeholder="编码" value={h.forms.coupon.code} onChange={(e) => h.updateForm('coupon', { code: e.target.value })} /><Button onClick={() => h.saveCoupon()}>保存</Button></div></Card>
        <Card title="评价回复"><PaginatedTable columns={[{ title: '用户', render: (r) => r.user?.name || '-' }, { title: '评分', render: (r) => r.rating }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }]} rows={h.reviews} /></Card>
        <Card title="公告管理"><PaginatedTable columns={[{ title: '标题', render: (r) => r.title }, { title: '受众', render: (r) => r.audience }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }]} rows={h.announcements} /><div className="mini-form"><TextInput placeholder="标题" value={h.forms.announcement.title} onChange={(e) => h.updateForm('announcement', { title: e.target.value })} /><Button onClick={() => h.saveAnnouncement()}>保存</Button></div></Card>
        <Card title="客服工单"><PaginatedTable columns={[{ title: '主题', render: (r) => r.subject }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '回复', render: (r) => r.reply || '-' }]} rows={h.adminSupportTickets} /></Card>
        <Card title="通知管理"><div className="mini-form"><TextInput placeholder="标题" value={h.forms.notification.title} onChange={(e) => h.updateForm('notification', { title: e.target.value })} /><TextInput placeholder="内容" value={h.forms.notification.content} onChange={(e) => h.updateForm('notification', { content: e.target.value })} /><Button onClick={() => h.sendAdminNotification()}>发送</Button></div><PaginatedTable columns={[{ title: '标题', render: (r) => r.title }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }]} rows={h.adminNotifications} /></Card>
      </div>
    </>
  )
}

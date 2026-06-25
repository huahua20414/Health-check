import React, { useEffect, useState } from 'react'
import { Button, Card, Field, Metric, Modal, PageHeader, PaginatedTable, Select, StatusTag, TextInput, Textarea } from '../components/UI.jsx'
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
  const [open, setOpen] = useState(false)
  const openCreate = () => { h.resetForm('package'); setOpen(true) }
  const openEdit = (pkg) => { h.updateForm('package', pkg); setOpen(true) }
  const save = () => h.savePackage().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="套餐管理" subtitle="套餐新增、编辑、归档和导出。" actions={<><Button onClick={openCreate}>新增套餐</Button><Button variant="ghost" onClick={() => h.exportBlob('/packages/export', 'packages.csv', 'exportPackages')}>导出</Button></>} />
      <Card title="套餐列表"><PaginatedTable columns={[{ title: '名称', render: (r) => r.name }, { title: '分类', render: (r) => r.category }, { title: '价格', render: (r) => moneyText(r.price) }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archivePackage(r)}>归档</Button></div> }]} rows={h.packages} /></Card>
      <Modal open={open} title={f.id ? '编辑套餐' : '新增套餐'} onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.package} onClick={save}>保存</Button></>}>
        <div className="form-grid">
          <Field label="名称"><TextInput value={f.name} onChange={(e) => h.updateForm('package', { name: e.target.value })} /></Field>
          <Field label="分类"><TextInput value={f.category} onChange={(e) => h.updateForm('package', { category: e.target.value })} /></Field>
          <Field label="价格"><TextInput type="number" value={f.price} onChange={(e) => h.updateForm('package', { price: e.target.value })} /></Field>
          <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('package', { status: e.target.value })}><option value="active">启用</option><option value="disabled">停用</option></Select></Field>
        </div>
        <Field label="项目明细"><Textarea value={f.items} onChange={(e) => h.updateForm('package', { items: e.target.value })} /></Field>
      </Modal>
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
  const [open, setOpen] = useState(false)
  const openCreate = () => { h.resetForm('institution'); setOpen(true) }
  const openEdit = (row) => { h.updateForm('institution', row); setOpen(true) }
  const save = () => h.saveInstitution().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return <Card title="机构管理" actions={<><Button size="sm" onClick={openCreate}>新增</Button><Button size="sm" variant="ghost" onClick={() => h.exportBlob('/institutions/export', 'institutions.csv', 'exportInstitutions')}>导出</Button></>}>
    <PaginatedTable columns={[{ title: '名称', render: (r) => r.name }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button> }]} rows={h.institutionRows.length ? h.institutionRows : h.institutions} />
    <Modal open={open} title={f.id ? '编辑机构' : '新增机构'} onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.institution} onClick={save}>保存</Button></>}>
      <Field label="机构名称"><TextInput value={f.name} onChange={(e) => h.updateForm('institution', { name: e.target.value })} /></Field>
      <Field label="地址"><TextInput value={f.address} onChange={(e) => h.updateForm('institution', { address: e.target.value })} /></Field>
      <Field label="电话"><TextInput value={f.phone} onChange={(e) => h.updateForm('institution', { phone: e.target.value })} /></Field>
      <Field label="营业时间"><TextInput value={f.openHours} onChange={(e) => h.updateForm('institution', { openHours: e.target.value })} /></Field>
      <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('institution', { status: e.target.value })}><option value="active">启用</option><option value="disabled">停用</option></Select></Field>
    </Modal>
  </Card>
}

function CheckupItemPanel({ h }) {
  const f = h.forms.checkupItem
  const [open, setOpen] = useState(false)
  const openCreate = () => { h.resetForm('checkupItem'); setOpen(true) }
  const openEdit = (row) => { h.updateForm('checkupItem', row); setOpen(true) }
  const save = () => h.saveCheckupItem().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return <Card title="体检项目" actions={<Button size="sm" onClick={openCreate}>新增</Button>}>
    <PaginatedTable columns={[{ title: '名称', render: (r) => r.name }, { title: '科室', render: (r) => r.department }, { title: '价格', render: (r) => moneyText(r.price) }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveCheckupItem(r)}>归档</Button></div> }]} rows={h.checkupItemRows.length ? h.checkupItemRows : h.checkupItems} />
    <Modal open={open} title={f.id ? '编辑体检项目' : '新增体检项目'} onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.checkupItem} onClick={save}>保存</Button></>}>
      <div className="form-grid">
        <Field label="项目名称"><TextInput value={f.name} onChange={(e) => h.updateForm('checkupItem', { name: e.target.value })} /></Field>
        <Field label="分类"><TextInput value={f.category} onChange={(e) => h.updateForm('checkupItem', { category: e.target.value })} /></Field>
        <Field label="科室"><TextInput value={f.department} onChange={(e) => h.updateForm('checkupItem', { department: e.target.value })} /></Field>
        <Field label="价格"><TextInput type="number" value={f.price} onChange={(e) => h.updateForm('checkupItem', { price: e.target.value })} /></Field>
        <Field label="时长分钟"><TextInput type="number" value={f.durationMin} onChange={(e) => h.updateForm('checkupItem', { durationMin: e.target.value })} /></Field>
        <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('checkupItem', { status: e.target.value })}><option value="active">启用</option><option value="disabled">停用</option></Select></Field>
      </div>
      <Field label="说明"><Textarea value={f.description} onChange={(e) => h.updateForm('checkupItem', { description: e.target.value })} /></Field>
    </Modal>
  </Card>
}

function PackageItemPanel({ h }) {
  const f = h.forms.packageItem
  const [open, setOpen] = useState(false)
  const openCreate = () => { h.resetForm('packageItem'); setOpen(true) }
  const save = () => h.savePackageItem().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return <Card title="套餐项目组合" actions={<Button size="sm" onClick={openCreate}>新增</Button>}>
    <PaginatedTable columns={[{ title: '套餐', render: (r) => r.package?.name || r.packageId }, { title: '项目', render: (r) => r.item?.name || r.itemId }, { title: '排序', render: (r) => r.sortOrder }, { title: '操作', render: (r) => <Button size="sm" variant="danger" onClick={() => h.deletePackageItem(r)}>移除</Button> }]} rows={h.packageItems} />
    <Modal open={open} title="新增套餐项目组合" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.packageItem} onClick={save}>保存</Button></>}>
      <Field label="套餐"><Select value={f.packageId} onChange={(e) => h.updateForm('packageItem', { packageId: e.target.value })}><option value="">请选择套餐</option>{h.packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</Select></Field>
      <Field label="项目"><Select value={f.itemId} onChange={(e) => h.updateForm('packageItem', { itemId: e.target.value })}><option value="">请选择项目</option>{h.checkupItems.map((i) => <option key={i.id} value={i.id}>{i.name}</option>)}</Select></Field>
      <Field label="排序"><TextInput type="number" value={f.sortOrder} onChange={(e) => h.updateForm('packageItem', { sortOrder: e.target.value })} /></Field>
      <Field label="是否必选"><Select value={String(f.required)} onChange={(e) => h.updateForm('packageItem', { required: e.target.value === 'true' })}><option value="true">必选</option><option value="false">可选</option></Select></Field>
    </Modal>
  </Card>
}

function SchedulePanel({ h }) {
  const f = h.forms.schedule
  const [open, setOpen] = useState(false)
  const rows = h.scheduleSlotRows.length ? h.scheduleSlotRows : h.slots
  const doctors = h.users.filter((u) => u.role === 'doctor' && u.status === 'active')
  const categories = [...new Set(h.packages.map((p) => p.category).filter(Boolean))]
  const openCreate = () => { h.resetForm('schedule'); setOpen(true) }
  const openEdit = (slot) => {
    h.updateForm('schedule', {
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
    setOpen(true)
  }
  const save = () => h.saveScheduleSlot().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return <Card title="医生号源" actions={<Button size="sm" onClick={openCreate}>新增</Button>}><PaginatedTable columns={[
    { title: '医生', render: (r) => r.doctor?.name || r.doctorId },
    { title: '机构', render: (r) => r.institution?.name || r.institutionId },
    { title: '日期', render: (r) => r.date },
    { title: '时段', render: (r) => `${r.startTime || ''}-${r.endTime || ''}` },
    { title: '分类', render: (r) => r.category || '-' },
    { title: '余号', render: (r) => `${Math.max(0, Number(r.capacity || 0) - Number(r.bookedCount || 0))}/${r.capacity || 0}` },
    { title: '状态', render: (r) => <StatusTag status={r.status} /> },
    { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveScheduleSlot(r)}>归档</Button></div> },
  ]} rows={rows} />
    <Modal open={open} title={f.id ? '编辑医生号源' : '新增医生号源'} onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.schedule} onClick={save}>{f.id ? '保存编辑' : '新增号源'}</Button></>}>
      <div className="form-grid">
        <Field label="医生"><Select value={f.doctorId} onChange={(e) => h.updateForm('schedule', { doctorId: e.target.value })}><option value="">请选择医生</option>{doctors.map((u) => <option key={u.id} value={u.id}>{u.name}</option>)}</Select></Field>
        <Field label="机构"><Select value={f.institutionId} onChange={(e) => h.updateForm('schedule', { institutionId: e.target.value })}><option value="">请选择机构</option>{h.institutions.map((i) => <option key={i.id} value={i.id}>{i.name}</option>)}</Select></Field>
        <Field label="分类"><Select value={f.category} onChange={(e) => h.updateForm('schedule', { category: e.target.value })}><option value="">请选择分类</option>{categories.map((category) => <option key={category} value={category}>{category}</option>)}</Select></Field>
        <Field label="日期"><TextInput type="date" value={f.date} onChange={(e) => h.updateForm('schedule', { date: e.target.value })} /></Field>
        <Field label="开始时间"><TextInput placeholder="08:30" value={f.startTime} onChange={(e) => h.updateForm('schedule', { startTime: e.target.value })} /></Field>
        <Field label="结束时间"><TextInput placeholder="09:00" value={f.endTime} onChange={(e) => h.updateForm('schedule', { endTime: e.target.value })} /></Field>
        <Field label="上午/下午"><Select value={f.period} onChange={(e) => h.updateForm('schedule', { period: e.target.value })}><option value="上午">上午</option><option value="下午">下午</option></Select></Field>
        <Field label="容量"><TextInput type="number" min="1" value={f.capacity} onChange={(e) => h.updateForm('schedule', { capacity: e.target.value })} /></Field>
        <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('schedule', { status: e.target.value })}><option value="available">可预约</option><option value="disabled">停用</option></Select></Field>
      </div>
    </Modal>
  </Card>
}

export function OperationsView() {
  const h = useHealth()
  const [modal, setModal] = useState('')
  useEffect(() => { h.loadCouponsPage(); h.loadReviewsPage(); h.loadAnnouncementsPage(); h.loadAdminNotificationsPage(); h.loadAdminSupportTicketsPage() }, [])
  const openCouponCreate = () => { h.resetForm('coupon'); setModal('coupon') }
  const openCouponEdit = (row) => { h.updateForm('coupon', row); setModal('coupon') }
  const openAnnouncementCreate = () => { h.resetForm('announcement'); setModal('announcement') }
  const openAnnouncementEdit = (row) => { h.updateForm('announcement', row); setModal('announcement') }
  const openReviewReply = (row) => { h.updateForm('reviewReply', { id: row.id, reply: row.reply || '', status: row.status || 'published' }); setModal('reviewReply') }
  const openTicketReply = (row) => { h.updateForm('supportTicketReply', { id: row.id, reply: row.reply || '', status: row.status === 'closed' ? 'closed' : 'replied' }); setModal('supportTicketReply') }
  const saveCoupon = () => h.saveCoupon().then(() => setModal('')).catch((e) => h.notify('error', e.message))
  const saveAnnouncement = () => h.saveAnnouncement().then(() => setModal('')).catch((e) => h.notify('error', e.message))
  const saveReviewReply = () => h.saveReviewReply().then(() => setModal('')).catch((e) => h.notify('error', e.message))
  const saveTicketReply = () => h.saveSupportTicketReply().then(() => setModal('')).catch((e) => h.notify('error', e.message))
  const sendNotice = () => h.sendAdminNotification().then(() => setModal('')).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="运营管理" subtitle="优惠券、评价、公告、通知、客服工单。" />
      <div className="management-grid">
        <Card title="优惠券" actions={<Button size="sm" onClick={openCouponCreate}>新增</Button>}><PaginatedTable columns={[{ title: '名称', render: (r) => r.name }, { title: '编码', render: (r) => r.code }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openCouponEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveCoupon(r)}>归档</Button></div> }]} rows={h.coupons} /></Card>
        <Card title="评价回复"><PaginatedTable columns={[{ title: '用户', render: (r) => r.user?.name || '-' }, { title: '评分', render: (r) => r.rating }, { title: '回复', render: (r) => r.reply || '-' }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => openReviewReply(r)}>处理</Button> }]} rows={h.reviews} /></Card>
        <Card title="公告管理" actions={<Button size="sm" onClick={openAnnouncementCreate}>新增</Button>}><PaginatedTable columns={[{ title: '标题', render: (r) => r.title }, { title: '受众', render: (r) => r.audience }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openAnnouncementEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveAnnouncement(r)}>归档</Button></div> }]} rows={h.announcements} /></Card>
        <Card title="客服工单"><PaginatedTable columns={[{ title: '主题', render: (r) => r.subject }, { title: '用户', render: (r) => r.user?.name || '-' }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '回复', render: (r) => r.reply || '-' }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => openTicketReply(r)}>回复</Button> }]} rows={h.adminSupportTickets} /></Card>
        <Card title="通知管理" actions={<Button size="sm" onClick={() => { h.resetForm('notification'); setModal('notification') }}>发送通知</Button>}><PaginatedTable columns={[{ title: '标题', render: (r) => r.title }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }]} rows={h.adminNotifications} /></Card>
      </div>
      <Modal open={modal === 'coupon'} title={h.forms.coupon.id ? '编辑优惠券' : '新增优惠券'} onClose={() => setModal('')} actions={<><Button variant="ghost" onClick={() => setModal('')}>取消</Button><Button loading={h.loading.coupon} onClick={saveCoupon}>保存</Button></>}>
        <div className="form-grid">
          <Field label="名称"><TextInput value={h.forms.coupon.name} onChange={(e) => h.updateForm('coupon', { name: e.target.value })} /></Field>
          <Field label="编码"><TextInput value={h.forms.coupon.code} onChange={(e) => h.updateForm('coupon', { code: e.target.value })} /></Field>
          <Field label="类型"><Select value={h.forms.coupon.type} onChange={(e) => h.updateForm('coupon', { type: e.target.value })}><option value="amount">固定金额</option><option value="percent">折扣比例</option></Select></Field>
          <Field label="面值"><TextInput type="number" value={h.forms.coupon.value} onChange={(e) => h.updateForm('coupon', { value: e.target.value })} /></Field>
          <Field label="最低金额"><TextInput type="number" value={h.forms.coupon.minAmount} onChange={(e) => h.updateForm('coupon', { minAmount: e.target.value })} /></Field>
          <Field label="适用套餐"><Select value={h.forms.coupon.packageId} onChange={(e) => h.updateForm('coupon', { packageId: e.target.value })}><option value="">全部套餐</option>{h.packages.map((pkg) => <option key={pkg.id} value={pkg.id}>{pkg.name}</option>)}</Select></Field>
          <Field label="开始日期"><TextInput type="date" value={h.forms.coupon.startDate} onChange={(e) => h.updateForm('coupon', { startDate: e.target.value })} /></Field>
          <Field label="结束日期"><TextInput type="date" value={h.forms.coupon.endDate} onChange={(e) => h.updateForm('coupon', { endDate: e.target.value })} /></Field>
          <Field label="状态"><Select value={h.forms.coupon.status} onChange={(e) => h.updateForm('coupon', { status: e.target.value })}><option value="active">启用</option><option value="disabled">停用</option></Select></Field>
        </div>
        <Field label="说明"><Textarea value={h.forms.coupon.description} onChange={(e) => h.updateForm('coupon', { description: e.target.value })} /></Field>
      </Modal>
      <Modal open={modal === 'announcement'} title={h.forms.announcement.id ? '编辑公告' : '新增公告'} onClose={() => setModal('')} actions={<><Button variant="ghost" onClick={() => setModal('')}>取消</Button><Button loading={h.loading.announcement} onClick={saveAnnouncement}>保存</Button></>}>
        <Field label="标题"><TextInput value={h.forms.announcement.title} onChange={(e) => h.updateForm('announcement', { title: e.target.value })} /></Field>
        <Field label="内容"><Textarea value={h.forms.announcement.content} onChange={(e) => h.updateForm('announcement', { content: e.target.value })} /></Field>
        <div className="form-grid">
          <Field label="受众"><Select value={h.forms.announcement.audience} onChange={(e) => h.updateForm('announcement', { audience: e.target.value })}><option value="all">全部</option><option value="user">用户</option><option value="doctor">医生</option><option value="admin">管理员</option></Select></Field>
          <Field label="状态"><Select value={h.forms.announcement.status} onChange={(e) => h.updateForm('announcement', { status: e.target.value })}><option value="draft">草稿</option><option value="published">发布</option></Select></Field>
        </div>
      </Modal>
      <Modal open={modal === 'reviewReply'} title="处理评价" onClose={() => setModal('')} actions={<><Button variant="ghost" onClick={() => setModal('')}>取消</Button><Button loading={h.loading.review} onClick={saveReviewReply}>保存处理</Button></>}>
        <Field label="回复内容"><Textarea value={h.forms.reviewReply.reply} onChange={(e) => h.updateForm('reviewReply', { reply: e.target.value })} /></Field>
        <Field label="状态"><Select value={h.forms.reviewReply.status} onChange={(e) => h.updateForm('reviewReply', { status: e.target.value })}><option value="published">展示</option><option value="hidden">隐藏</option></Select></Field>
      </Modal>
      <Modal open={modal === 'supportTicketReply'} title="回复客服工单" onClose={() => setModal('')} actions={<><Button variant="ghost" onClick={() => setModal('')}>取消</Button><Button loading={h.loading.adminNotification} onClick={saveTicketReply}>保存回复</Button></>}>
        <Field label="回复内容"><Textarea value={h.forms.supportTicketReply.reply} onChange={(e) => h.updateForm('supportTicketReply', { reply: e.target.value })} /></Field>
        <Field label="状态"><Select value={h.forms.supportTicketReply.status} onChange={(e) => h.updateForm('supportTicketReply', { status: e.target.value })}><option value="replied">已回复</option><option value="closed">已关闭</option><option value="open">重新打开</option></Select></Field>
      </Modal>
      <Modal open={modal === 'notification'} title="发送通知" onClose={() => setModal('')} actions={<><Button variant="ghost" onClick={() => setModal('')}>取消</Button><Button loading={h.loading.adminNotification} onClick={sendNotice}>发送</Button></>}>
        <Field label="标题"><TextInput value={h.forms.notification.title} onChange={(e) => h.updateForm('notification', { title: e.target.value })} /></Field>
        <Field label="内容"><Textarea value={h.forms.notification.content} onChange={(e) => h.updateForm('notification', { content: e.target.value })} /></Field>
      </Modal>
    </>
  )
}

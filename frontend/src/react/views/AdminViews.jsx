import React, { useEffect, useState } from 'react'
import { Button, Card, Field, Metric, Modal, PageHeader, PaginatedTable, RemoteTable, Select, StatusTag, TextInput, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { doctorDepartments, formatDate, moneyText, normalizeIDCard } from '../utils'

export function AdminDashboardView() {
  const h = useHealth()
  const summary = h.adminDashboard?.summary || {}
  return (
    <>
      <PageHeader title="管理员工作台" subtitle="审核、用户、机构、套餐和号源管理。" />
      <div className="metrics-grid"><Metric label="待审核医生" value={h.pendingDoctors.length || summary.pendingDoctors || 0} tone="amber" /><Metric label="机构数量" value={h.institutions.length} /><Metric label="套餐数量" value={h.packages.length} tone="green" /><Metric label="邮件日志" value={h.mailLogs.length} tone="violet" /></div>
      <Card title="待审核医生"><PaginatedTable columns={[{ title: '医生', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <DoctorReviewActions row={r} h={h} compact /> }]} rows={h.pendingDoctors} /></Card>
    </>
  )
}

export function DoctorReviewView() {
  const h = useHealth()
  const [filters, setFilters] = useState({ page: 1, pageSize: 10, role: 'doctor' })
  const [doctorOpen, setDoctorOpen] = useState(false)
  const [editingDoctor, setEditingDoctor] = useState(null)
  useEffect(() => { h.loadUsersPage(filters, 'doctors', 'doctorUsers').catch((e) => h.notify('error', e.message)) }, [filters.page, filters.pageSize])
  const refresh = () => h.loadUsersPage(filters, 'doctors', 'doctorUsers').catch((e) => h.notify('error', e.message))
  const updateStatus = (row, status) => h.updateUserStatus(row, status).then(refresh).catch((e) => h.notify('error', e.message))
  const openDoctorEdit = (row) => {
    setEditingDoctor(row)
    h.updateForm('doctorRegister', { department: row.department || row.doctorProfile?.department || '', title: row.title || row.doctorProfile?.title || '', employeeNo: row.employeeNo || '', name: row.name || '', email: row.email || '', code: '' })
    setDoctorOpen(true)
  }
  const saveDoctor = () => h.updateDoctorProfile(editingDoctor, { department: h.forms.doctorRegister.department, title: h.forms.doctorRegister.title, specialties: editingDoctor?.specialties || h.forms.doctorRegister.department }).then(() => { setDoctorOpen(false); refresh() }).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="医生审核" subtitle="审核医生账号，并维护科室、职称与专长。" />
      <Card title="医生列表">
        <RemoteTable
          columns={[{ title: '姓名', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '科室', render: (r) => r.doctorProfile?.department || r.department || '-' }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><DoctorReviewActions row={r} h={h} onStatus={updateStatus} /><Button size="sm" variant="ghost" onClick={() => openDoctorEdit(r)}>编辑资料</Button></div> }]}
          rows={h.doctorUsers}
          pagination={h.paginations.doctors}
          onPageChange={(page) => setFilters((current) => ({ ...current, page }))}
          onPageSizeChange={(pageSize) => setFilters((current) => ({ ...current, page: 1, pageSize }))}
        />
      </Card>
      <Modal open={doctorOpen} title="编辑医生资料" onClose={() => setDoctorOpen(false)} actions={<><Button variant="ghost" onClick={() => setDoctorOpen(false)}>取消</Button><Button loading={h.loading.doctorProfile} onClick={saveDoctor}>保存</Button></>}>
        <div className="form-grid">
          <Field label="科室"><Select value={h.forms.doctorRegister.department} onChange={(e) => h.updateForm('doctorRegister', { department: e.target.value })}><option value="">请选择科室</option>{doctorDepartments.map((item) => <option key={item} value={item}>{item}</option>)}</Select></Field>
          <Field label="职称"><TextInput value={h.forms.doctorRegister.title} onChange={(e) => h.updateForm('doctorRegister', { title: e.target.value })} /></Field>
          <Field label="专长"><TextInput value={editingDoctor?.specialties || h.forms.doctorRegister.department} onChange={(e) => setEditingDoctor((current) => ({ ...current, specialties: e.target.value }))} /></Field>
        </div>
      </Modal>
    </>
  )
}

export function AdminUsersView() {
  const h = useHealth()
  const [filters, setFilters] = useState({ page: 1, pageSize: 10, keyword: '', role: '', status: '' })
  const [draft, setDraft] = useState({ keyword: '', role: '', status: '' })
  const [open, setOpen] = useState(false)
  useEffect(() => { h.loadUsersPage(filters).catch((e) => h.notify('error', e.message)) }, [filters.page, filters.pageSize, filters.keyword, filters.role, filters.status])
  const apply = () => setFilters((current) => ({ ...current, page: 1, ...draft }))
  const reset = () => {
    setDraft({ keyword: '', role: '', status: '' })
    setFilters((current) => ({ ...current, page: 1, keyword: '', role: '', status: '' }))
  }
  const refresh = () => h.loadUsersPage(filters).catch((e) => h.notify('error', e.message))
  const updateStatus = (row, status) => h.updateUserStatus(row, status).then(refresh).catch((e) => h.notify('error', e.message))
  const openEdit = (row) => {
    h.updateForm('adminUser', { id: row.id, name: row.name || '', gender: row.gender || '', idCard: row.idCard || '', email: row.email || '', avatarUrl: row.avatarUrl || '', bio: row.bio || '', emailNotify: row.emailNotify !== false, status: row.status || 'active' })
    setOpen(true)
  }
  const save = () => h.saveAdminUser().then(() => { setOpen(false); refresh() }).catch((e) => h.notify('error', e.message))
  const f = h.forms.adminUser
  return (
    <>
      <PageHeader title="用户管理" subtitle="管理员查看账号、按角色和状态筛选，并维护启停状态。" />
      <Card title="账号列表">
        <div className="filter-bar">
          <TextInput placeholder="姓名、邮箱、工号" value={draft.keyword} onChange={(e) => setDraft((current) => ({ ...current, keyword: e.target.value }))} />
          <Select value={draft.role} onChange={(e) => setDraft((current) => ({ ...current, role: e.target.value }))}><option value="">全部角色</option><option value="user">用户</option><option value="doctor">医生</option><option value="admin">管理员</option></Select>
          <Select value={draft.status} onChange={(e) => setDraft((current) => ({ ...current, status: e.target.value }))}><option value="">全部状态</option><option value="active">启用</option><option value="pending">待审核</option><option value="disabled">停用</option></Select>
          <div className="row-actions"><Button onClick={apply}>查询</Button><Button variant="ghost" onClick={reset}>重置</Button><Button variant="ghost" onClick={() => h.exportBlob('/users/export', 'users.csv', 'exportUsers')}>导出</Button></div>
        </div>
        <RemoteTable
          columns={[{ title: '姓名', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '角色', render: (r) => r.role }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><DoctorReviewActions row={r} h={h} onStatus={updateStatus} /></div> }]}
          rows={h.users}
          pagination={h.paginations.users}
          onPageChange={(page) => setFilters((current) => ({ ...current, page }))}
          onPageSizeChange={(pageSize) => setFilters((current) => ({ ...current, page: 1, pageSize }))}
        />
      </Card>
      <Modal open={open} title="编辑用户资料" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.adminUser} onClick={save}>保存</Button></>}>
        <div className="form-grid">
          <Field label="姓名"><TextInput value={f.name} onChange={(e) => h.updateForm('adminUser', { name: e.target.value })} /></Field>
          <Field label="邮箱"><TextInput value={f.email} onChange={(e) => h.updateForm('adminUser', { email: e.target.value })} /></Field>
          <Field label="性别"><Select value={f.gender} onChange={(e) => h.updateForm('adminUser', { gender: e.target.value })}><option value="">未填写</option><option value="男">男</option><option value="女">女</option></Select></Field>
          <Field label="身份证"><TextInput value={f.idCard} maxLength={18} placeholder="例如 11010519491231002X" onChange={(e) => h.updateForm('adminUser', { idCard: normalizeIDCard(e.target.value) })} /></Field>
          <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('adminUser', { status: e.target.value })}><option value="active">启用</option><option value="pending">待审核</option><option value="disabled">停用</option></Select></Field>
          <Field label="邮箱通知"><Select value={String(f.emailNotify)} onChange={(e) => h.updateForm('adminUser', { emailNotify: e.target.value === 'true' })}><option value="true">开启</option><option value="false">关闭</option></Select></Field>
        </div>
        <Field label="头像地址"><TextInput value={f.avatarUrl} onChange={(e) => h.updateForm('adminUser', { avatarUrl: e.target.value })} /></Field>
        <Field label="简介"><Textarea value={f.bio} onChange={(e) => h.updateForm('adminUser', { bio: e.target.value })} /></Field>
      </Modal>
    </>
  )
}

function DoctorReviewActions({ row, h, compact = false, onStatus }) {
  const update = onStatus || ((user, status) => h.updateUserStatus(user, status))
  if (row.status === 'pending') {
    return (
      <div className="row-actions">
        <Button size="sm" loading={h.loading.status} onClick={() => update(row, 'active')}>通过</Button>
        <Button size="sm" variant="danger" loading={h.loading.status} onClick={() => update(row, 'disabled')}>{compact ? '拒绝' : '停用'}</Button>
      </div>
    )
  }
  if (row.status === 'active') {
    return (
      <div className="row-actions">
        <Button size="sm" variant="danger" loading={h.loading.status} onClick={() => update(row, 'disabled')}>停用</Button>
        <span className="muted-text">已启用</span>
      </div>
    )
  }
  if (row.status === 'disabled') {
    return <Button size="sm" variant="ghost" loading={h.loading.status} onClick={() => update(row, 'active')}>重新启用</Button>
  }
  return <span className="muted-text">无可用操作</span>
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
  useEffect(() => { h.loadInstitutionsPage(); h.loadCheckupItemsPage(); h.loadUsersPage({ role: 'doctor', status: 'active', pageSize: 100 }, 'activeDoctors', 'activeDoctors') }, [])
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
  const [params, setParams] = useState({ page: 1, pageSize: 10 })
  useEffect(() => { h.loadPackageItemsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize])
  const openCreate = () => { h.resetForm('packageItem'); setOpen(true) }
  const openEdit = (row) => {
    h.updateForm('packageItem', { id: row.id, packageId: row.packageId, itemId: row.itemId, sortOrder: row.sortOrder, required: row.required })
    setOpen(true)
  }
  const itemOptions = h.checkupItemRows.length ? h.checkupItemRows : h.checkupItems
  const reload = () => h.loadPackageItemsPage(params).catch((e) => h.notify('error', e.message))
  const save = () => h.savePackageItem().then(() => { setOpen(false); reload() }).catch((e) => h.notify('error', e.message))
  return <Card title="套餐项目组合" actions={<Button size="sm" onClick={openCreate}>新增</Button>}>
    <RemoteTable
      columns={[{ title: '套餐', render: (r) => r.package?.name || r.packageId }, { title: '项目', render: (r) => r.item?.name || r.itemId }, { title: '排序', render: (r) => r.sortOrder }, { title: '类型', render: (r) => r.required ? '必选' : '可选' }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.deletePackageItem(r).then(reload).catch((e) => h.notify('error', e.message))}>移除</Button></div> }]}
      rows={h.packageItems}
      pagination={h.paginations.packageItems}
      onPageChange={(page) => setParams((current) => ({ ...current, page }))}
      onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
    />
    <Modal open={open} title="套餐项目组合" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.packageItem} onClick={save}>保存</Button></>}>
      <Field label="套餐"><Select value={f.packageId} onChange={(e) => h.updateForm('packageItem', { packageId: e.target.value })}><option value="">请选择套餐</option>{h.packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</Select></Field>
      <Field label="项目"><Select value={f.itemId} onChange={(e) => h.updateForm('packageItem', { itemId: e.target.value })}><option value="">请选择项目</option>{itemOptions.map((i) => <option key={i.id} value={i.id}>{i.name}</option>)}</Select></Field>
      <Field label="排序"><TextInput type="number" value={f.sortOrder} onChange={(e) => h.updateForm('packageItem', { sortOrder: e.target.value })} /></Field>
      <Field label="是否必选"><Select value={String(f.required)} onChange={(e) => h.updateForm('packageItem', { required: e.target.value === 'true' })}><option value="true">必选</option><option value="false">可选</option></Select></Field>
    </Modal>
  </Card>
}

function SchedulePanel({ h }) {
  const f = h.forms.schedule
  const [open, setOpen] = useState(false)
  const [params, setParams] = useState({ page: 1, pageSize: 10 })
  useEffect(() => { h.loadSlotsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize])
  const rows = h.scheduleSlotRows.length ? h.scheduleSlotRows : h.slots
  const doctors = h.activeDoctors.length ? h.activeDoctors : h.users.filter((u) => u.role === 'doctor' && u.status === 'active')
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
  const reload = () => h.loadSlotsPage(params).catch((e) => h.notify('error', e.message))
  const save = () => h.saveScheduleSlot().then(() => { setOpen(false); reload() }).catch((e) => h.notify('error', e.message))
  return <Card title="医生号源" actions={<Button size="sm" onClick={openCreate}>新增</Button>}><RemoteTable columns={[
    { title: '医生', render: (r) => r.doctor?.name || r.doctorId },
    { title: '机构', render: (r) => r.institution?.name || r.institutionId },
    { title: '日期', render: (r) => formatDate(r.date) },
    { title: '时段', render: (r) => `${r.startTime || ''}-${r.endTime || ''}` },
    { title: '分类', render: (r) => r.category || '-' },
    { title: '余号', render: (r) => `${Math.max(0, Number(r.capacity || 0) - Number(r.bookedCount || 0))}/${r.capacity || 0}` },
    { title: '状态', render: (r) => <StatusTag status={r.status} /> },
    { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveScheduleSlot(r).then(reload).catch((e) => h.notify('error', e.message))}>归档</Button></div> },
  ]} rows={rows} pagination={h.paginations.slots} onPageChange={(page) => setParams((current) => ({ ...current, page }))} onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))} />
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

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
  const [doctorForm, setDoctorForm] = useState({ department: '', title: '', specialties: '' })
  useEffect(() => { h.loadUsersPage(filters, 'doctors', 'doctorUsers').catch((e) => h.notify('error', e.message)) }, [filters.page, filters.pageSize])
  const refresh = () => h.loadUsersPage(filters, 'doctors', 'doctorUsers').catch((e) => h.notify('error', e.message))
  const updateStatus = (row, status) => h.updateUserStatus(row, status).then(refresh).catch((e) => h.notify('error', e.message))
  const openDoctorEdit = (row) => {
    setEditingDoctor(row)
    setDoctorForm({
      department: row.department || row.doctorProfile?.department || '',
      title: row.title || row.doctorProfile?.title || '',
      specialties: row.specialties || row.doctorProfile?.specialties || row.department || row.doctorProfile?.department || '',
    })
    setDoctorOpen(true)
  }
  const saveDoctor = () => h.updateDoctorProfile(editingDoctor, doctorForm).then(() => { setDoctorOpen(false); refresh() }).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="医生审核与资料" subtitle="审核医生账号，并维护科室、职称与专长；姓名、邮箱和工号保持注册记录。" />
      <Card title="医生列表">
        <RemoteTable
          columns={[{ title: '姓名', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '工号', render: (r) => r.employeeNo || '-' }, { title: '科室', render: (r) => r.doctorProfile?.department || r.department || '-' }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><DoctorReviewActions row={r} h={h} onStatus={updateStatus} /><Button size="sm" variant="ghost" onClick={() => openDoctorEdit(r)}>编辑资料</Button></div> }]}
          rows={h.doctorUsers}
          pagination={h.paginations.doctors}
          onPageChange={(page) => setFilters((current) => ({ ...current, page }))}
          onPageSizeChange={(pageSize) => setFilters((current) => ({ ...current, page: 1, pageSize }))}
        />
      </Card>
      <Modal open={doctorOpen} title="编辑医生资料" onClose={() => setDoctorOpen(false)} actions={<><Button variant="ghost" onClick={() => setDoctorOpen(false)}>取消</Button><Button loading={h.loading.doctorProfile} onClick={saveDoctor}>保存</Button></>}>
        <div className="readonly-summary">
          <div><span>姓名</span><strong>{editingDoctor?.name || '-'}</strong></div>
          <div><span>邮箱</span><strong>{editingDoctor?.email || '-'}</strong></div>
          <div><span>工号</span><strong>{editingDoctor?.employeeNo || '-'}</strong></div>
          <div><span>状态</span><strong><StatusTag status={editingDoctor?.status} /></strong></div>
        </div>
        <div className="form-grid">
          <Field label="科室"><Select value={doctorForm.department} onChange={(e) => setDoctorForm((current) => ({ ...current, department: e.target.value }))}><option value="">请选择科室</option>{doctorDepartments.map((item) => <option key={item} value={item}>{item}</option>)}</Select></Field>
          <Field label="职称"><TextInput value={doctorForm.title} onChange={(e) => setDoctorForm((current) => ({ ...current, title: e.target.value }))} /></Field>
          <Field label="专长"><TextInput value={doctorForm.specialties} onChange={(e) => setDoctorForm((current) => ({ ...current, specialties: e.target.value }))} /></Field>
        </div>
      </Modal>
    </>
  )
}

export function AdminUsersView() {
  const h = useHealth()
  const [filters, setFilters] = useState({ page: 1, pageSize: 10, keyword: '', role: 'user', status: '' })
  const [draft, setDraft] = useState({ keyword: '', status: '' })
  const [open, setOpen] = useState(false)
  useEffect(() => { h.loadUsersPage(filters).catch((e) => h.notify('error', e.message)) }, [filters.page, filters.pageSize, filters.keyword, filters.role, filters.status])
  const apply = () => setFilters((current) => ({ ...current, page: 1, role: 'user', ...draft }))
  const reset = () => {
    setDraft({ keyword: '', status: '' })
    setFilters((current) => ({ ...current, page: 1, keyword: '', role: 'user', status: '' }))
  }
  const refresh = () => h.loadUsersPage(filters).catch((e) => h.notify('error', e.message))
  const updateStatus = (row, status) => h.updateUserStatus(row, status).then(refresh).catch((e) => h.notify('error', e.message))
  const openEdit = (row) => {
    h.updateForm('adminUser', { id: row.id, name: row.name || '', gender: row.gender || '', idCard: row.idCard || '', email: row.email || '', bio: row.bio || '', emailNotify: row.emailNotify !== false, status: row.status || 'active' })
    setOpen(true)
  }
  const save = () => h.saveAdminUser().then(() => { setOpen(false); refresh() }).catch((e) => h.notify('error', e.message))
  const f = h.forms.adminUser
  return (
    <>
      <PageHeader title="普通用户管理" subtitle="仅管理用户端账号资料与启停状态；医生账号请到医生审核与资料中处理。" />
      <Card title="用户列表">
        <div className="filter-bar">
          <TextInput placeholder="姓名、邮箱、工号" value={draft.keyword} onChange={(e) => setDraft((current) => ({ ...current, keyword: e.target.value }))} />
          <Select value={draft.status} onChange={(e) => setDraft((current) => ({ ...current, status: e.target.value }))}><option value="">全部状态</option><option value="active">启用</option><option value="pending">待审核</option><option value="disabled">停用</option></Select>
          <div className="row-actions"><Button onClick={apply}>查询</Button><Button variant="ghost" onClick={reset}>重置</Button><Button variant="ghost" onClick={() => h.exportBlob('/users/export?role=user', 'users.csv', 'exportUsers')}>导出</Button></div>
        </div>
        <RemoteTable
          columns={[{ title: '姓名', render: (r) => r.name }, { title: '邮箱', render: (r) => r.email }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><UserAccountActions row={r} h={h} onStatus={updateStatus} /></div> }]}
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
        <Field label="简介"><Textarea value={f.bio} onChange={(e) => h.updateForm('adminUser', { bio: e.target.value })} /></Field>
      </Modal>
    </>
  )
}

function UserAccountActions({ row, h, onStatus }) {
  if (row.status === 'disabled') {
    return <Button size="sm" variant="ghost" loading={h.loading.status} onClick={() => onStatus(row, 'active')}>重新启用</Button>
  }
  return <Button size="sm" variant="danger" loading={h.loading.status} onClick={() => onStatus(row, 'disabled')}>停用</Button>
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
  const openEdit = (row) => { h.updateForm('institution', { ...row, packageIds: institutionPackageIds(row) }); setOpen(true) }
  const save = () => h.saveInstitution().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  const selectedPackageIds = new Set((f.packageIds || []).map(Number))
  const togglePackage = (packageId) => {
    const next = new Set((f.packageIds || []).map(Number))
    if (next.has(packageId)) next.delete(packageId)
    else next.add(packageId)
    h.updateForm('institution', { packageIds: Array.from(next) })
  }
  return <Card title="机构管理" actions={<><Button size="sm" onClick={openCreate}>新增</Button><Button size="sm" variant="ghost" onClick={() => h.exportBlob('/institutions/export', 'institutions.csv', 'exportInstitutions')}>导出</Button></>}>
    <PaginatedTable loading={h.loading.institutions} columns={[{ title: '名称', render: (r) => r.name }, { title: '可服务套餐', render: (r) => institutionPackageNames(r, h.packages) }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" loading={h.loading.institution} onClick={() => h.archiveInstitution(r).catch((e) => h.notify('error', e.message))}>归档</Button></div> }]} rows={h.institutionRows.length ? h.institutionRows : h.institutions} />
    <Modal open={open} title={f.id ? '编辑机构' : '新增机构'} onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.institution} onClick={save}>保存</Button></>}>
      <Field label="机构名称"><TextInput value={f.name} onChange={(e) => h.updateForm('institution', { name: e.target.value })} /></Field>
      <Field label="地址"><TextInput value={f.address} onChange={(e) => h.updateForm('institution', { address: e.target.value })} /></Field>
      <Field label="电话"><TextInput value={f.phone} onChange={(e) => h.updateForm('institution', { phone: e.target.value })} /></Field>
      <Field label="营业时间"><TextInput value={f.openHours} onChange={(e) => h.updateForm('institution', { openHours: e.target.value })} /></Field>
      <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('institution', { status: e.target.value })}><option value="active">启用</option><option value="disabled">停用</option></Select></Field>
      <Field label="可服务套餐">
        <div className="package-item-picker institution-package-picker">
          {h.packages.map((pkg) => (
            <label key={pkg.id} className={`package-item-option ${selectedPackageIds.has(pkg.id) ? 'is-checked' : ''}`}>
              <input type="checkbox" checked={selectedPackageIds.has(pkg.id)} onChange={() => togglePackage(pkg.id)} />
              <span>{pkg.category || '套餐'}</span>
              <strong>{pkg.name}</strong>
              <small>{moneyText(pkg.price)}</small>
            </label>
          ))}
          {!h.packages.length && <p className="muted">暂无可绑定套餐。</p>}
        </div>
      </Field>
    </Modal>
  </Card>
}

function institutionPackageIds(institution) {
  const ids = new Set((institution.packageIds || []).map(Number).filter(Boolean))
  for (const pkg of institution.packages || []) ids.add(Number(pkg.id))
  return Array.from(ids)
}

function institutionPackageNames(institution, packages) {
  const ids = institutionPackageIds(institution)
  if (!ids.length) return <span className="muted">未绑定</span>
  const names = ids.map((id) => packages.find((pkg) => Number(pkg.id) === id)?.name || `套餐 #${id}`)
  return names.join('、')
}

function CheckupItemPanel({ h }) {
  const f = h.forms.checkupItem
  const [open, setOpen] = useState(false)
  const openCreate = () => { h.resetForm('checkupItem'); setOpen(true) }
  const openEdit = (row) => { h.updateForm('checkupItem', row); setOpen(true) }
  const save = () => h.saveCheckupItem().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return <Card title="体检项目" actions={<Button size="sm" onClick={openCreate}>新增</Button>}>
    <PaginatedTable loading={h.loading.checkupItems} columns={[{ title: '名称', render: (r) => r.name }, { title: '科室', render: (r) => r.department }, { title: '价格', render: (r) => moneyText(r.price) }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveCheckupItem(r)}>归档</Button></div> }]} rows={h.checkupItemRows.length ? h.checkupItemRows : h.checkupItems} />
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
  const [selectedPackageId, setSelectedPackageId] = useState('')
  const [draggingId, setDraggingId] = useState(null)
  const [dragOverId, setDragOverId] = useState(null)
  const [localOrder, setLocalOrder] = useState([])
  useEffect(() => {
    if (!selectedPackageId && h.packages[0]?.id) setSelectedPackageId(String(h.packages[0].id))
  }, [h.packages, selectedPackageId])
  useEffect(() => {
    if (selectedPackageId) h.loadPackageItemsCollection({ packageId: selectedPackageId }).catch((e) => h.notify('error', e.message))
  }, [selectedPackageId])
  const openCreate = () => {
    h.resetForm('packageItem')
    h.updateForm('packageItem', { packageId: selectedPackageId, sortOrder: selectedItems.length + 1 })
    setOpen(true)
  }
  const openEdit = (row) => {
    h.updateForm('packageItem', { id: row.id, packageId: row.packageId, itemId: row.itemId, sortOrder: row.sortOrder, required: row.required })
    setOpen(true)
  }
  const itemOptions = h.checkupItemRows.length ? h.checkupItemRows : h.checkupItems
  const selectedItems = h.packageItems
    .filter((item) => String(item.packageId) === String(selectedPackageId))
    .sort((a, b) => Number(a.sortOrder || 0) - Number(b.sortOrder || 0) || Number(a.id || 0) - Number(b.id || 0))
  const visibleItems = localOrder.length ? localOrder : selectedItems
  const reload = () => h.loadPackageItemsCollection({ packageId: f.packageId || selectedPackageId }).catch((e) => h.notify('error', e.message))
  const save = () => h.savePackageItem().then(() => { setOpen(false); reload() }).catch((e) => h.notify('error', e.message))
  const moveItem = (sourceId, targetId) => {
    if (h.loading.packageItem) return
    if (!sourceId || sourceId === targetId) return
    const currentItems = localOrder.length ? localOrder : selectedItems
    const from = currentItems.findIndex((item) => item.id === sourceId)
    const to = currentItems.findIndex((item) => item.id === targetId)
    if (from < 0 || to < 0) return
    const next = [...currentItems]
    const [moved] = next.splice(from, 1)
    next.splice(to, 0, moved)
    setLocalOrder(next)
    h.reorderPackageItems(next).then(() => setLocalOrder([])).catch((e) => { setLocalOrder([]); h.notify('error', e.message) })
  }
  return <Card title="套餐项目组合" actions={<Button size="sm" onClick={openCreate}>新增</Button>}>
    <div className="package-composer">
      <div className="package-composer-toolbar">
        <Field label="选择套餐"><Select value={selectedPackageId} onChange={(e) => setSelectedPackageId(e.target.value)}><option value="">请选择套餐</option>{h.packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</Select></Field>
        <span>{selectedItems.length ? `${selectedItems.length} 个项目，拖动左侧手柄调整顺序` : '选择套餐后管理项目顺序'}</span>
      </div>
      <div className="sortable-package-list">
        {visibleItems.map((row, index) => (
          <div
            key={row.id}
            className={`sortable-package-row ${draggingId === row.id ? 'is-dragging' : ''} ${dragOverId === row.id && draggingId !== row.id ? 'is-drop-target' : ''}`}
            draggable={!h.loading.packageItem}
            onDragStart={(e) => { setDraggingId(row.id); e.dataTransfer.effectAllowed = 'move' }}
            onDragOver={(e) => { e.preventDefault(); setDragOverId(row.id); e.dataTransfer.dropEffect = 'move' }}
            onDragLeave={() => setDragOverId((current) => current === row.id ? null : current)}
            onDrop={(e) => { e.preventDefault(); moveItem(draggingId, row.id); setDraggingId(null); setDragOverId(null) }}
            onDragEnd={() => { setDraggingId(null); setDragOverId(null) }}
          >
            <button type="button" className="drag-handle" aria-label="拖动排序">⋮⋮</button>
            <span className="sort-index">{index + 1}</span>
            <div className="sortable-package-main">
              <strong>{row.item?.name || row.itemId}</strong>
              <small>{row.item?.category || '体检项目'} · {row.item?.department || '未分科室'}</small>
            </div>
            <span className={`package-required-pill ${row.required ? 'is-required' : ''}`}>{row.required ? '必选' : '可选'}</span>
            <div className="row-actions">
              <Button size="sm" variant="ghost" onClick={() => openEdit(row)}>编辑</Button>
              <Button size="sm" variant="danger" onClick={() => h.deletePackageItem(row).then(reload).catch((e) => h.notify('error', e.message))}>移除</Button>
            </div>
          </div>
        ))}
        {!visibleItems.length && <p className="muted">{h.loading.packageItems ? '套餐项目加载中...' : '暂无组合项目，点击右上角新增。'}</p>}
      </div>
    </div>
    <Modal open={open} title="套餐项目组合" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.packageItem} onClick={save}>保存</Button></>}>
      <Field label="套餐"><Select value={f.packageId} onChange={(e) => h.updateForm('packageItem', { packageId: e.target.value })}><option value="">请选择套餐</option>{h.packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</Select></Field>
      <Field label="项目"><Select value={f.itemId} onChange={(e) => h.updateForm('packageItem', { itemId: e.target.value })}><option value="">请选择项目</option>{itemOptions.map((i) => <option key={i.id} value={i.id}>{i.name}</option>)}</Select></Field>
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
      dates: slot.date,
      period: slot.period || '上午',
      category: slot.category || '',
      startTime: slot.startTime || '08:30',
      startTimes: slot.startTime || '08:30',
      endTime: slot.endTime || '',
      capacity: slot.capacity || 1,
      status: slot.status || 'available',
    })
    setOpen(true)
  }
  const reload = () => h.loadSlotsPage(params).catch((e) => h.notify('error', e.message))
  const save = () => h.saveScheduleSlot().then(() => { setOpen(false); reload() }).catch((e) => h.notify('error', e.message))
  return <Card title="医生号源" actions={<Button size="sm" onClick={openCreate}>新增</Button>}><RemoteTable loading={h.loading.slots} columns={[
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
        {f.id ? <Field label="日期"><TextInput type="date" value={f.date} onChange={(e) => h.updateForm('schedule', { date: e.target.value, dates: e.target.value })} /></Field> : <Field label="日期"><Textarea placeholder="可输入多个日期，用空格、逗号或换行分隔" value={f.dates} onChange={(e) => h.updateForm('schedule', { dates: e.target.value, date: e.target.value.split(/[\n,，\s]+/)[0] || '' })} /></Field>}
        {f.id ? <Field label="开始时间"><TextInput placeholder="08:30" value={f.startTime} onChange={(e) => h.updateForm('schedule', { startTime: e.target.value, startTimes: e.target.value })} /></Field> : <Field label="开始时间"><Textarea placeholder="如 08:30 09:00 09:30，可多选多个班" value={f.startTimes} onChange={(e) => h.updateForm('schedule', { startTimes: e.target.value, startTime: e.target.value.split(/[\n,，\s]+/)[0] || '' })} /></Field>}
        {f.id && <Field label="结束时间"><TextInput placeholder="09:00" value={f.endTime} onChange={(e) => h.updateForm('schedule', { endTime: e.target.value })} /></Field>}
        <Field label="上午/下午"><Select value={f.period} onChange={(e) => h.updateForm('schedule', { period: e.target.value })}><option value="上午">上午</option><option value="下午">下午</option></Select></Field>
        <Field label="容量"><TextInput type="number" min="1" value={f.capacity} onChange={(e) => h.updateForm('schedule', { capacity: e.target.value })} /></Field>
        <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('schedule', { status: e.target.value })}><option value="available">可预约</option><option value="disabled">停用</option></Select></Field>
      </div>
    </Modal>
  </Card>
}

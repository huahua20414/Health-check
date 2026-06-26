import React, { useEffect, useState } from 'react'
import { Button, Card, Field, Modal, PageHeader, RemoteTable, Select, StatusTag, TextInput, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { formatDate, moneyText, nextDateString } from '../utils'

function usePagedParams(initial = {}) {
  return useState({ page: 1, pageSize: 10, keyword: '', status: '', ...initial })
}

function FilterBar({ draft, setDraft, onApply, onReset, children }) {
  return (
    <div className="filter-bar">
      <TextInput placeholder="关键字" value={draft.keyword || ''} onChange={(e) => setDraft((current) => ({ ...current, keyword: e.target.value }))} />
      {children}
      <div className="row-actions"><Button onClick={onApply}>查询</Button><Button variant="ghost" onClick={onReset}>重置</Button></div>
    </div>
  )
}

export function AdminEngagementView() {
  return (
    <>
      <PageHeader title="营销与公告" subtitle="优惠券、系统公告和用户可见运营内容。" />
      <div className="management-grid">
        <CouponPanel />
        <AnnouncementPanel />
      </div>
    </>
  )
}

function CouponPanel() {
  const h = useHealth()
  const [params, setParams] = usePagedParams()
  const [draft, setDraft] = useState({ keyword: '', status: '' })
  const [open, setOpen] = useState(false)
  const f = h.forms.coupon
  useEffect(() => { h.loadCouponsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize, params.keyword, params.status])
  const apply = () => setParams((current) => ({ ...current, page: 1, ...draft }))
  const reset = () => { setDraft({ keyword: '', status: '' }); setParams((current) => ({ ...current, page: 1, keyword: '', status: '' })) }
  const openCreate = () => { h.resetForm('coupon'); setOpen(true) }
  const openEdit = (row) => { h.updateForm('coupon', row); setOpen(true) }
  const save = () => h.saveCoupon().then(() => { setOpen(false); h.loadCouponsPage(params) }).catch((e) => h.notify('error', e.message))
  return (
    <Card title="优惠券" actions={<><Button size="sm" onClick={openCreate}>新增</Button><Button size="sm" variant="ghost" onClick={() => h.exportBlob('/coupons/export', 'coupons.csv', 'exportCoupons')}>导出</Button></>}>
      <FilterBar draft={draft} setDraft={setDraft} onApply={apply} onReset={reset}>
        <Select value={draft.status} onChange={(e) => setDraft((current) => ({ ...current, status: e.target.value }))}><option value="">全部状态</option><option value="active">启用</option><option value="disabled">停用</option></Select>
      </FilterBar>
      <RemoteTable
        columns={[{ title: '券码', render: (r) => r.code }, { title: '名称', render: (r) => r.name }, { title: '优惠', render: (r) => r.type === 'percent' ? `${r.value}%` : moneyText(r.value) }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveCoupon(r).then(() => h.loadCouponsPage(params)).catch((e) => h.notify('error', e.message))}>归档</Button></div> }]}
        rows={h.coupons}
        pagination={h.paginations.coupons}
        onPageChange={(page) => setParams((current) => ({ ...current, page }))}
        onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
      />
      <Modal open={open} title={f.id ? '编辑优惠券' : '新增优惠券'} onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.coupon} onClick={save}>保存</Button></>}>
        <div className="form-grid">
          <Field label="名称"><TextInput value={f.name} onChange={(e) => h.updateForm('coupon', { name: e.target.value })} /></Field>
          <Field label="券码"><TextInput value={f.code} onChange={(e) => h.updateForm('coupon', { code: e.target.value })} /></Field>
          <Field label="类型"><Select value={f.type} onChange={(e) => h.updateForm('coupon', { type: e.target.value })}><option value="amount">金额</option><option value="percent">折扣百分比</option></Select></Field>
          <Field label="优惠值"><TextInput type="number" value={f.value} onChange={(e) => h.updateForm('coupon', { value: e.target.value })} /></Field>
          <Field label="最低金额"><TextInput type="number" value={f.minAmount} onChange={(e) => h.updateForm('coupon', { minAmount: e.target.value })} /></Field>
          <Field label="限定套餐"><Select value={f.packageId} onChange={(e) => h.updateForm('coupon', { packageId: e.target.value })}><option value="">不限套餐</option>{h.packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</Select></Field>
          <Field label="开始日期"><TextInput type="date" value={f.startDate} onChange={(e) => h.updateForm('coupon', { startDate: e.target.value })} /></Field>
          <Field label="结束日期"><TextInput type="date" value={f.endDate} onChange={(e) => h.updateForm('coupon', { endDate: e.target.value })} /></Field>
          <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('coupon', { status: e.target.value })}><option value="active">启用</option><option value="disabled">停用</option></Select></Field>
        </div>
        <Field label="说明"><Textarea value={f.description} onChange={(e) => h.updateForm('coupon', { description: e.target.value })} /></Field>
      </Modal>
    </Card>
  )
}

function AnnouncementPanel() {
  const h = useHealth()
  const [params, setParams] = usePagedParams()
  const [draft, setDraft] = useState({ keyword: '', status: '' })
  const [open, setOpen] = useState(false)
  const f = h.forms.announcement
  useEffect(() => { h.loadAnnouncementsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize, params.keyword, params.status])
  const apply = () => setParams((current) => ({ ...current, page: 1, ...draft }))
  const reset = () => { setDraft({ keyword: '', status: '' }); setParams((current) => ({ ...current, page: 1, keyword: '', status: '' })) }
  const openCreate = () => { h.resetForm('announcement'); setOpen(true) }
  const openEdit = (row) => { h.updateForm('announcement', row); setOpen(true) }
  const save = () => h.saveAnnouncement().then(() => { setOpen(false); h.loadAnnouncementsPage(params) }).catch((e) => h.notify('error', e.message))
  return (
    <Card title="公告" actions={<><Button size="sm" onClick={openCreate}>新增</Button><Button size="sm" variant="ghost" onClick={() => h.exportBlob('/announcements/export', 'announcements.csv', 'exportAnnouncements')}>导出</Button></>}>
      <FilterBar draft={draft} setDraft={setDraft} onApply={apply} onReset={reset}>
        <Select value={draft.status} onChange={(e) => setDraft((current) => ({ ...current, status: e.target.value }))}><option value="">全部状态</option><option value="draft">草稿</option><option value="published">已发布</option><option value="hidden">已隐藏</option></Select>
      </FilterBar>
      <RemoteTable
        columns={[{ title: '标题', render: (r) => r.title }, { title: '受众', render: (r) => r.audience }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '时间', render: (r) => formatDate(r.createdAt) }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.archiveAnnouncement(r).then(() => h.loadAnnouncementsPage(params)).catch((e) => h.notify('error', e.message))}>归档</Button></div> }]}
        rows={h.announcements}
        pagination={h.paginations.announcements}
        onPageChange={(page) => setParams((current) => ({ ...current, page }))}
        onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
      />
      <Modal open={open} title={f.id ? '编辑公告' : '新增公告'} onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.announcement} onClick={save}>保存</Button></>}>
        <Field label="标题"><TextInput value={f.title} onChange={(e) => h.updateForm('announcement', { title: e.target.value })} /></Field>
        <div className="form-grid">
          <Field label="受众"><Select value={f.audience} onChange={(e) => h.updateForm('announcement', { audience: e.target.value })}><option value="all">全部</option><option value="user">用户</option><option value="doctor">医生</option><option value="admin">管理员</option></Select></Field>
          <Field label="状态"><Select value={f.status} onChange={(e) => h.updateForm('announcement', { status: e.target.value })}><option value="draft">草稿</option><option value="published">发布</option><option value="hidden">隐藏</option></Select></Field>
        </div>
        <Field label="内容"><Textarea value={f.content} onChange={(e) => h.updateForm('announcement', { content: e.target.value })} /></Field>
      </Modal>
    </Card>
  )
}

export function AdminCommunicationView() {
  return (
    <>
      <PageHeader title="通知与客服" subtitle="管理员通知、体检提醒、客服工单和服务评价。" />
      <div className="management-grid">
        <NotificationPanel />
        <SupportPanel />
        <ReviewPanel />
      </div>
    </>
  )
}

function NotificationPanel() {
  const h = useHealth()
  const [params, setParams] = usePagedParams()
  const [draft, setDraft] = useState({ keyword: '', status: '' })
  const [open, setOpen] = useState(false)
  useEffect(() => { h.loadAdminNotificationsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize, params.keyword, params.status])
  const apply = () => setParams((current) => ({ ...current, page: 1, ...draft }))
  const reset = () => { setDraft({ keyword: '', status: '' }); setParams((current) => ({ ...current, page: 1, keyword: '', status: '' })) }
  const send = () => h.sendAdminNotification().then(() => { setOpen(false); h.loadAdminNotificationsPage(params) }).catch((e) => h.notify('error', e.message))
  const sendTomorrowReminders = () => {
    const date = nextDateString()
    h.updateForm('reminder', { date })
    h.sendCheckupReminders({ date }).then(() => h.loadAdminNotificationsPage(params)).catch((e) => h.notify('error', e.message))
  }
  return (
    <Card title="通知中心" actions={<><Button size="sm" onClick={() => { h.resetForm('notification'); setOpen(true) }}>发送通知</Button><Button size="sm" variant="ghost" loading={h.loading.reminder} onClick={sendTomorrowReminders}>生成明日提醒</Button></>}>
      <FilterBar draft={draft} setDraft={setDraft} onApply={apply} onReset={reset}>
        <Select value={draft.status} onChange={(e) => setDraft((current) => ({ ...current, status: e.target.value }))}><option value="">全部状态</option><option value="unread">未读</option><option value="read">已读</option></Select>
      </FilterBar>
      <RemoteTable
        columns={[{ title: '标题', render: (r) => r.title }, { title: '用户', render: (r) => r.user?.name || r.userId }, { title: '渠道', render: (r) => r.channel }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <Button size="sm" variant="danger" onClick={() => h.updateAdminNotificationStatus(r, 'archived').then(() => h.loadAdminNotificationsPage(params)).catch((e) => h.notify('error', e.message))}>归档</Button> }]}
        rows={h.adminNotifications}
        pagination={h.paginations.adminNotifications}
        onPageChange={(page) => setParams((current) => ({ ...current, page }))}
        onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
      />
      <Modal open={open} title="发送通知" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.adminNotification} onClick={send}>发送</Button></>}>
        <div className="form-grid">
          <Field label="目标角色"><Select value={h.forms.notification.role} onChange={(e) => h.updateForm('notification', { role: e.target.value, userId: '' })}><option value="user">用户</option><option value="doctor">医生</option><option value="admin">管理员</option><option value="all">全部</option></Select></Field>
          <Field label="渠道"><Select value={h.forms.notification.channel} onChange={(e) => h.updateForm('notification', { channel: e.target.value })}><option value="in_app">站内信</option><option value="sms_mock">短信模拟</option></Select></Field>
        </div>
        <Field label="标题"><TextInput value={h.forms.notification.title} onChange={(e) => h.updateForm('notification', { title: e.target.value })} /></Field>
        <Field label="内容"><Textarea value={h.forms.notification.content} onChange={(e) => h.updateForm('notification', { content: e.target.value })} /></Field>
      </Modal>
    </Card>
  )
}

function SupportPanel() {
  const h = useHealth()
  const [params, setParams] = usePagedParams()
  const [open, setOpen] = useState(false)
  useEffect(() => { h.loadAdminSupportTicketsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize, params.status])
  const openReply = (row) => { h.updateForm('supportTicketReply', { id: row.id, reply: row.reply || '', status: row.status === 'open' ? 'replied' : row.status }); setOpen(true) }
  const save = () => h.saveSupportTicketReply().then(() => { setOpen(false); h.loadAdminSupportTicketsPage(params) }).catch((e) => h.notify('error', e.message))
  return (
    <Card title="客服工单" actions={<Button size="sm" variant="ghost" onClick={() => h.exportBlob('/admin/support-tickets/export', 'support-tickets.csv', 'exportSupportTickets')}>导出</Button>}>
      <div className="filter-bar">
        <Select value={params.status} onChange={(e) => setParams((current) => ({ ...current, page: 1, status: e.target.value }))}><option value="">全部状态</option><option value="open">待处理</option><option value="replied">已回复</option><option value="closed">已关闭</option></Select>
      </div>
      <RemoteTable
        columns={[{ title: '主题', render: (r) => r.subject }, { title: '用户', render: (r) => r.user?.name || r.userId }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => openReply(r)}>处理</Button> }]}
        rows={h.adminSupportTickets}
        pagination={h.paginations.adminSupportTickets}
        onPageChange={(page) => setParams((current) => ({ ...current, page }))}
        onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
      />
      <Modal open={open} title="处理工单" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.adminNotification} onClick={save}>保存处理</Button></>}>
        <Field label="状态"><Select value={h.forms.supportTicketReply.status} onChange={(e) => h.updateForm('supportTicketReply', { status: e.target.value })}><option value="replied">已回复</option><option value="closed">已关闭</option><option value="open">待处理</option></Select></Field>
        <Field label="回复"><Textarea value={h.forms.supportTicketReply.reply} onChange={(e) => h.updateForm('supportTicketReply', { reply: e.target.value })} /></Field>
      </Modal>
    </Card>
  )
}

function ReviewPanel() {
  const h = useHealth()
  const [params, setParams] = usePagedParams()
  const [open, setOpen] = useState(false)
  useEffect(() => { h.loadReviewsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize, params.status, params.keyword])
  const openReply = (row) => { h.updateForm('reviewReply', { id: row.id, reply: row.reply || '', status: row.status || 'published' }); setOpen(true) }
  const save = () => h.saveReviewReply().then(() => { setOpen(false); h.loadReviewsPage(params) }).catch((e) => h.notify('error', e.message))
  return (
    <Card title="服务评价" actions={<Button size="sm" variant="ghost" onClick={() => h.exportBlob('/reviews/export', 'reviews.csv', 'exportReviews')}>导出</Button>}>
      <RemoteTable
        columns={[{ title: '用户', render: (r) => r.user?.name || r.userId }, { title: '套餐', render: (r) => r.package?.name || '-' }, { title: '评分', render: (r) => `${r.rating} 星` }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => openReply(r)}>回复</Button> }]}
        rows={h.reviews}
        pagination={h.paginations.reviews}
        onPageChange={(page) => setParams((current) => ({ ...current, page }))}
        onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
      />
      <Modal open={open} title="评价处理" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.review} onClick={save}>保存</Button></>}>
        <Field label="状态"><Select value={h.forms.reviewReply.status} onChange={(e) => h.updateForm('reviewReply', { status: e.target.value })}><option value="published">展示</option><option value="hidden">隐藏</option></Select></Field>
        <Field label="回复"><Textarea value={h.forms.reviewReply.reply} onChange={(e) => h.updateForm('reviewReply', { reply: e.target.value })} /></Field>
      </Modal>
    </Card>
  )
}

export function AdminSystemView() {
  return (
    <>
      <PageHeader title="系统治理" subtitle="邮件、登录、操作日志，角色权限与系统设置。" />
      <div className="management-grid">
        <LogPanel kind="mail" title="邮件日志" rowsKey="mailLogs" pageKey="mailLogs" loader="loadMailLogsPage" exportPath="/mail-logs/export" filename="mail-logs.csv" columns={[{ title: '收件人', render: (r) => r.to }, { title: '主题', render: (r) => r.subject }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '时间', render: (r) => formatDate(r.createdAt) }]} />
        <LogPanel kind="login" title="登录日志" rowsKey="loginLogs" pageKey="loginLogs" loader="loadLoginLogsPage" exportPath="/login-logs/export" filename="login-logs.csv" columns={[{ title: '邮箱', render: (r) => r.email }, { title: '角色', render: (r) => r.role }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '时间', render: (r) => formatDate(r.createdAt) }]} />
        <LogPanel kind="operation" title="操作日志" rowsKey="operationLogs" pageKey="operationLogs" loader="loadOperationLogsPage" exportPath="/operation-logs/export" filename="operation-logs.csv" columns={[{ title: '用户', render: (r) => r.userName || r.userId }, { title: '动作', render: (r) => r.action }, { title: '资源', render: (r) => r.resource }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }]} />
        <PermissionPanel />
        <SettingPanel />
      </div>
    </>
  )
}

function LogPanel({ title, rowsKey, pageKey, loader, exportPath, filename, columns }) {
  const h = useHealth()
  const [params, setParams] = usePagedParams()
  const [draft, setDraft] = useState({ keyword: '', status: '' })
  useEffect(() => { h[loader](params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize, params.keyword, params.status])
  const apply = () => setParams((current) => ({ ...current, page: 1, ...draft }))
  const reset = () => { setDraft({ keyword: '', status: '' }); setParams((current) => ({ ...current, page: 1, keyword: '', status: '' })) }
  return (
    <Card title={title} actions={<Button size="sm" variant="ghost" onClick={() => h.exportBlob(exportPath, filename, `export-${pageKey}`)}>导出</Button>}>
      <FilterBar draft={draft} setDraft={setDraft} onApply={apply} onReset={reset} />
      <RemoteTable
        columns={columns}
        rows={h[rowsKey]}
        pagination={h.paginations[pageKey]}
        onPageChange={(page) => setParams((current) => ({ ...current, page }))}
        onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
      />
    </Card>
  )
}

function PermissionPanel() {
  const h = useHealth()
  const [params, setParams] = usePagedParams()
  useEffect(() => { h.loadRolePermissionsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize])
  return (
    <Card title="角色权限" actions={<Button size="sm" variant="ghost" onClick={() => h.exportBlob('/role-permissions/export', 'role-permissions.csv', 'exportRolePermissions')}>导出</Button>}>
      <RemoteTable
        columns={[{ title: '角色', render: (r) => r.role }, { title: '权限', render: (r) => r.permission }, { title: '状态', render: (r) => <StatusTag status={r.enabled ? 'active' : 'disabled'} /> }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => h.updateRolePermission({ ...r, enabled: !r.enabled }).then(() => h.loadRolePermissionsPage(params)).catch((e) => h.notify('error', e.message))}>{r.enabled ? '停用' : '启用'}</Button> }]}
        rows={h.rolePermissionRows}
        pagination={h.paginations.rolePermissions}
        onPageChange={(page) => setParams((current) => ({ ...current, page }))}
        onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
      />
    </Card>
  )
}

function SettingPanel() {
  const h = useHealth()
  const [params, setParams] = usePagedParams()
  const [open, setOpen] = useState(false)
  const settingForm = h.forms.systemSetting || { id: null, key: '', label: '', value: '', valueType: 'string', group: '', status: 'active', description: '' }
  useEffect(() => { h.loadSystemSettingsPage(params).catch((e) => h.notify('error', e.message)) }, [params.page, params.pageSize])
  const openEdit = (row) => { h.updateForm('systemSetting', row); setOpen(true) }
  const save = () => h.updateSystemSetting(settingForm).then(() => { setOpen(false); h.loadSystemSettingsPage(params) }).catch((e) => h.notify('error', e.message))
  return (
    <Card title="系统设置" actions={<Button size="sm" variant="ghost" onClick={() => h.exportBlob('/system-settings/export', 'system-settings.csv', 'exportSystemSettings')}>导出</Button>}>
      <RemoteTable
        columns={[{ title: '配置', render: (r) => r.label || r.key }, { title: '分组', render: (r) => r.group }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button> }]}
        rows={h.systemSettingRows}
        pagination={h.paginations.systemSettings}
        onPageChange={(page) => setParams((current) => ({ ...current, page }))}
        onPageSizeChange={(pageSize) => setParams((current) => ({ ...current, page: 1, pageSize }))}
      />
      <Modal open={open} title="编辑系统设置" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.systemSetting} onClick={save}>保存</Button></>}>
        <Field label="名称"><TextInput value={settingForm.label || ''} onChange={(e) => h.updateForm('systemSetting', { label: e.target.value })} /></Field>
        <Field label="值"><Textarea value={settingForm.value || ''} onChange={(e) => h.updateForm('systemSetting', { value: e.target.value })} /></Field>
        <div className="form-grid">
          <Field label="类型"><Select value={settingForm.valueType || 'string'} onChange={(e) => h.updateForm('systemSetting', { valueType: e.target.value })}><option value="string">字符串</option><option value="number">数字</option><option value="bool">布尔</option><option value="json">JSON</option></Select></Field>
          <Field label="状态"><Select value={settingForm.status || 'active'} onChange={(e) => h.updateForm('systemSetting', { status: e.target.value })}><option value="active">启用</option><option value="disabled">停用</option></Select></Field>
        </div>
        <Field label="说明"><Textarea value={settingForm.description || ''} onChange={(e) => h.updateForm('systemSetting', { description: e.target.value })} /></Field>
      </Modal>
    </Card>
  )
}

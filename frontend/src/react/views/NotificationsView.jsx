import React, { useState } from 'react'
import { Button, Card, Field, Modal, PageHeader, PaginatedTable, StatusTag, TextInput, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { formatDate } from '../utils'

export function NotificationsView() {
  const h = useHealth()
  const [open, setOpen] = useState(false)
  const openCreate = () => { h.resetForm('supportTicket'); setOpen(true) }
  const save = () => h.createSupportTicket().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="消息与客服" subtitle="通知、客服工单和 FAQ。" actions={<Button onClick={openCreate}>提交咨询</Button>} />
      <Card title="通知中心"><PaginatedTable columns={[{ title: '标题', render: (r) => r.title }, { title: '类型', render: (r) => r.type }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '时间', render: (r) => formatDate(r.createdAt) }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => h.markNotificationRead(r)}>标记已读</Button> }]} rows={h.notifications} /></Card>
      <Card title="我的工单"><PaginatedTable columns={[{ title: '主题', render: (r) => r.subject }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '回复', render: (r) => r.reply || '-' }]} rows={h.supportTickets} /></Card>
      <Card title="FAQ"><div className="faq-grid">{(h.supportInfo.faq || []).map((item, i) => <div className="faq-item" key={i}><strong>{item.question || item.q}</strong><p>{item.answer || item.a}</p></div>)}</div></Card>
      <Modal open={open} title="提交客服工单" onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.notification} onClick={save}>提交咨询</Button></>}>
        <Field label="主题"><TextInput value={h.forms.supportTicket.subject} onChange={(e) => h.updateForm('supportTicket', { subject: e.target.value })} /></Field>
        <Field label="内容"><Textarea value={h.forms.supportTicket.content} onChange={(e) => h.updateForm('supportTicket', { content: e.target.value })} /></Field>
      </Modal>
    </>
  )
}

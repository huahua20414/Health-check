import React from 'react'
import { Button, Card, Field, PageHeader, PaginatedTable, StatusTag, TextInput, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { formatDate } from '../utils'

export function NotificationsView() {
  const h = useHealth()
  return (
    <>
      <PageHeader title="消息与客服" subtitle="通知、客服工单和 FAQ。" />
      <div className="two-col">
        <Card title="通知中心"><PaginatedTable columns={[{ title: '标题', render: (r) => r.title }, { title: '类型', render: (r) => r.type }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '时间', render: (r) => formatDate(r.createdAt) }, { title: '操作', render: (r) => <Button size="sm" variant="ghost" onClick={() => h.markNotificationRead(r)}>标记已读</Button> }]} rows={h.notifications} /></Card>
        <Card title="提交客服工单">
          <Field label="主题"><TextInput value={h.forms.supportTicket.subject} onChange={(e) => h.updateForm('supportTicket', { subject: e.target.value })} /></Field>
          <Field label="内容"><Textarea value={h.forms.supportTicket.content} onChange={(e) => h.updateForm('supportTicket', { content: e.target.value })} /></Field>
          <Button loading={h.loading.notification} onClick={() => h.createSupportTicket().catch((e) => h.notify('error', e.message))}>提交咨询</Button>
        </Card>
      </div>
      <Card title="我的工单"><PaginatedTable columns={[{ title: '主题', render: (r) => r.subject }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '回复', render: (r) => r.reply || '-' }]} rows={h.supportTickets} /></Card>
      <Card title="FAQ"><div className="faq-grid">{(h.supportInfo.faq || []).map((item, i) => <div className="faq-item" key={i}><strong>{item.question || item.q}</strong><p>{item.answer || item.a}</p></div>)}</div></Card>
    </>
  )
}

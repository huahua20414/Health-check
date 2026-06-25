import React, { useState } from 'react'
import { Button, Card, Field, PageHeader, PaginatedTable, Select, StatusTag, TextInput, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { moneyText, paymentStatusText } from '../utils'

export function AppointmentsView() {
  const h = useHealth()
  const [selected, setSelected] = useState(null)
  return (
    <>
      <PageHeader title="我的预约" subtitle="取消、改期、支付状态、发票、评价与候补状态。" />
      <Card title="预约记录">
        <PaginatedTable columns={[
          { title: '订单', render: (r) => r.orderNo || r.id },
          { title: '套餐', render: (r) => r.package?.name || r.appointmentType },
          { title: '日期', render: (r) => `${r.date || '-'} ${r.startTime || ''}` },
          { title: '支付', render: (r) => paymentStatusText(r.paymentStatus) },
          { title: '状态', render: (r) => <StatusTag status={r.status} /> },
          { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => h.downloadAppointment(r)}>下载</Button><Button size="sm" variant="secondary" onClick={() => setSelected(r)}>处理</Button></div> },
        ]} rows={h.myAppointments} />
      </Card>
      {selected && <Card title={`处理预约：${selected.orderNo || selected.id}`} actions={<Button variant="ghost" onClick={() => setSelected(null)}>关闭</Button>}>
        <div className="action-grid">
          <Button variant="secondary" onClick={() => h.updateAppointmentPayment(selected, selected.paymentStatus === 'paid' ? 'unpaid' : 'paid').catch((e) => h.notify('error', e.message))}>{selected.paymentStatus === 'paid' ? '撤销支付' : '模拟支付'}</Button>
          <Button variant="danger" onClick={() => h.cancelAppointment(selected).catch((e) => h.notify('error', e.message))}>取消预约</Button>
          <Button variant="ghost" onClick={() => h.updateForm('invoice', { appointmentId: selected.id, invoiceTitle: selected.invoiceTitle || '', invoiceTaxNo: selected.invoiceTaxNo || '' })}>填写发票</Button>
          <Button variant="ghost" onClick={() => h.updateForm('review', { appointmentId: selected.id })}>评价</Button>
        </div>
        <div className="form-grid compact">
          <Field label="发票抬头"><TextInput value={h.forms.invoice.invoiceTitle} onChange={(e) => h.updateForm('invoice', { invoiceTitle: e.target.value })} /></Field>
          <Field label="纳税人识别号"><TextInput value={h.forms.invoice.invoiceTaxNo} onChange={(e) => h.updateForm('invoice', { invoiceTaxNo: e.target.value })} /></Field>
          <Button onClick={() => h.saveInvoice().catch((e) => h.notify('error', e.message))}>保存发票</Button>
        </div>
        <div className="form-grid compact">
          <Field label="评分"><Select value={h.forms.review.rating} onChange={(e) => h.updateForm('review', { rating: e.target.value })}>{[5, 4, 3, 2, 1].map((n) => <option key={n} value={n}>{n} 星</option>)}</Select></Field>
          <Field label="评价内容"><Textarea value={h.forms.review.content} onChange={(e) => h.updateForm('review', { content: e.target.value })} /></Field>
          <Button onClick={() => h.createReview().catch((e) => h.notify('error', e.message))}>提交评价</Button>
        </div>
      </Card>}
      <Card title="候补记录"><PaginatedTable columns={[{ title: '套餐', render: (r) => r.package?.name || r.appointmentType }, { title: '日期', render: (r) => r.date }, { title: '状态', render: (r) => <StatusTag status={r.status} /> }, { title: '操作', render: (r) => <Button size="sm" variant="danger" onClick={() => h.cancelWaitlist(r).catch((e) => h.notify('error', e.message))}>取消候补</Button> }]} rows={h.waitlist} /></Card>
    </>
  )
}

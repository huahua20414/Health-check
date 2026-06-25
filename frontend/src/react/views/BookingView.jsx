import React from 'react'
import { Button, Card, Field, PageHeader, Select, TextInput, Textarea, StatusTag } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { appointmentTypes } from '../utils'

export function BookingView() {
  const h = useHealth()
  const form = h.forms.appointment
  const availableSlots = h.slots.filter((slot) => (!form.institutionId || slot.institutionId === Number(form.institutionId)) && (!form.date || slot.date === form.date))
  return (
    <>
      <PageHeader title="预约体检" subtitle="用户只选择机构、套餐、日期和期望时段；医生由后端自动分配。" />
      <div className="steps"><span>1 选机构</span><span>2 选套餐</span><span className="active">3 选日期时段</span><span>4 支付/提交</span></div>
      <div className="two-col">
        <Card title="预约信息">
          <div className="form-grid">
            <Field label="预约类型"><Select value={form.appointmentType} onChange={(e) => h.updateForm('appointment', { appointmentType: e.target.value })}>{appointmentTypes.map((t) => <option key={t}>{t}</option>)}</Select></Field>
            <Field label="机构"><Select value={form.institutionId} onChange={(e) => h.updateForm('appointment', { institutionId: e.target.value })}><option value="">请选择机构</option>{h.institutions.map((i) => <option key={i.id} value={i.id}>{i.name}</option>)}</Select></Field>
            <Field label="套餐"><Select value={form.packageId} onChange={(e) => h.updateForm('appointment', { packageId: e.target.value })}><option value="">请选择套餐</option>{h.packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</Select></Field>
            <Field label="家庭成员"><Select value={form.familyMemberId} onChange={(e) => h.updateForm('appointment', { familyMemberId: e.target.value })}><option value="">本人</option>{h.familyMembers.map((m) => <option key={m.id} value={m.id}>{m.name} · {m.relation}</option>)}</Select></Field>
            <Field label="日期"><TextInput type="date" value={form.date} onChange={(e) => h.updateForm('appointment', { date: e.target.value })} /></Field>
            <Field label="时段"><TextInput value={form.period} onChange={(e) => h.updateForm('appointment', { period: e.target.value })} placeholder="上午 / 09:00" /></Field>
            <Field label="优惠券"><Select value={form.couponId} onChange={(e) => h.updateForm('appointment', { couponId: e.target.value })}><option value="">不使用</option>{h.activeCoupons.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</Select></Field>
            <Field label="发票抬头"><TextInput value={form.invoiceTitle} onChange={(e) => h.updateForm('appointment', { invoiceTitle: e.target.value })} /></Field>
          </div>
          <Field label="备注"><Textarea value={form.note} onChange={(e) => h.updateForm('appointment', { note: e.target.value })} /></Field>
          <Button loading={h.loading.appointment} onClick={() => h.createAppointment().catch((e) => h.notify('error', e.message))}>提交预约</Button>
        </Card>
        <Card title="号源状态">
          <div className="slot-grid">{availableSlots.slice(0, 12).map((slot) => <button key={slot.id} className={`slot-card ${slot.status === 'full' ? 'is-full' : ''}`} onClick={() => h.updateForm('appointment', { slotId: slot.id, date: slot.date, period: slot.period || slot.startTime })}><strong>{slot.startTime || slot.period}</strong><span>{slot.doctor?.name || slot.doctorName || '系统分配医生'}</span><StatusTag status={slot.status || 'available'} /></button>)}</div>
          {!availableSlots.length && <p className="muted">当前筛选下暂无号源，请切换日期或直接提交进入候补逻辑。</p>}
        </Card>
      </div>
    </>
  )
}

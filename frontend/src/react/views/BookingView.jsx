import React, { useMemo } from 'react'
import { Button, Card, Field, PageHeader, Select, TextInput, Textarea, StatusTag } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { appointmentTypes } from '../utils'

export function BookingView() {
  const h = useHealth()
  const form = h.forms.appointment
  const selectedPackage = h.packages.find((pkg) => pkg.id === Number(form.packageId))
  const days = useMemo(() => nextDays(14), [])
  const visibleDate = form.date || days[0]?.value || ''
  const filteredSlots = h.slots.filter((slot) => {
    if (form.institutionId && slot.institutionId !== Number(form.institutionId)) return false
    if (selectedPackage?.category && slot.category !== selectedPackage.category) return false
    return days.some((day) => day.value === slot.date)
  })
  const slotsByTime = groupSlotsByDateTime(filteredSlots)
  const selectedDaySlots = slotsByTime[visibleDate] || []
  return (
    <>
      <PageHeader title="预约体检" subtitle="选择机构、套餐和未来两周号源，医生由后端按可用号自动分配。" />
      <div className="steps"><span>1 选机构</span><span>2 选套餐</span><span className="active">3 选日期时段</span><span>4 支付/提交</span></div>
      <div className="two-col">
        <Card title="预约信息">
          <div className="form-grid">
            <Field label="预约类型"><Select value={form.appointmentType} onChange={(e) => h.updateForm('appointment', { appointmentType: e.target.value })}>{appointmentTypes.map((t) => <option key={t}>{t}</option>)}</Select></Field>
            <Field label="机构"><Select value={form.institutionId} onChange={(e) => h.updateForm('appointment', { institutionId: e.target.value })}><option value="">请选择机构</option>{h.institutions.map((i) => <option key={i.id} value={i.id}>{i.name}</option>)}</Select></Field>
            <Field label="套餐"><Select value={form.packageId} onChange={(e) => h.updateForm('appointment', { packageId: e.target.value })}><option value="">请选择套餐</option>{h.packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</Select></Field>
            <Field label="家庭成员"><Select value={form.familyMemberId} onChange={(e) => h.updateForm('appointment', { familyMemberId: e.target.value })}><option value="">本人</option>{h.familyMembers.map((m) => <option key={m.id} value={m.id}>{m.name} · {m.relation}</option>)}</Select></Field>
            <Field label="日期"><TextInput type="date" value={form.date} onChange={(e) => h.updateForm('appointment', { date: e.target.value })} /></Field>
            <Field label="时段"><TextInput value={form.period} onChange={(e) => h.updateForm('appointment', { period: e.target.value, slotId: '' })} placeholder="请选择右侧号源" /></Field>
            <Field label="优惠券"><Select value={form.couponId} onChange={(e) => h.updateForm('appointment', { couponId: e.target.value })}><option value="">不使用</option>{h.activeCoupons.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</Select></Field>
            <Field label="发票抬头"><TextInput value={form.invoiceTitle} onChange={(e) => h.updateForm('appointment', { invoiceTitle: e.target.value })} /></Field>
          </div>
          <Field label="备注"><Textarea value={form.note} onChange={(e) => h.updateForm('appointment', { note: e.target.value })} /></Field>
          <Button loading={h.loading.appointment} onClick={() => h.createAppointment().catch((e) => h.notify('error', e.message))}>提交预约</Button>
        </Card>
        <Card title="未来两周号源">
          <div className="date-strip">
            {days.map((day) => (
              <button key={day.value} className={`date-chip ${visibleDate === day.value ? 'is-active' : ''}`} onClick={() => h.updateForm('appointment', { date: day.value, slotId: '', period: '' })}>
                <strong>{day.day}</strong><span>{day.week}</span>
              </button>
            ))}
          </div>
          <div className="slot-grid">
            {selectedDaySlots.map((group) => {
              const chosen = group.availableSlot || group.slots[0]
              const full = group.remaining <= 0
              return (
                <button key={`${group.date}-${group.startTime}`} className={`slot-card ${full ? 'is-full' : ''} ${Number(form.slotId) === chosen.id ? 'is-selected' : ''}`} onClick={() => h.updateForm('appointment', { slotId: chosen.id, date: group.date, period: group.period || group.startTime })}>
                  <strong>{group.startTime}-{group.endTime}</strong>
                  <span>{group.doctorCount} 位医生 · 余 {group.remaining}</span>
                  <StatusTag status={full ? 'full' : 'available'} />
                </button>
              )
            })}
          </div>
          {!selectedDaySlots.length && <p className="muted">当前筛选下暂无号源，请切换机构或套餐。</p>}
        </Card>
      </div>
    </>
  )
}

function nextDays(count) {
  const formatter = new Intl.DateTimeFormat('zh-CN', { weekday: 'short' })
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Array.from({ length: count }, (_, index) => {
    const date = new Date(today)
    date.setDate(today.getDate() + index)
    return {
      value: localDateValue(date),
      day: `${date.getMonth() + 1}/${date.getDate()}`,
      week: index === 0 ? '今天' : formatter.format(date),
    }
  })
}

function localDateValue(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function groupSlotsByDateTime(slots) {
  const grouped = {}
  for (const slot of slots) {
    const key = `${slot.date}|${slot.startTime}`
    const remaining = Math.max(0, Number(slot.capacity || 0) - Number(slot.bookedCount || 0))
    if (!grouped[key]) {
      grouped[key] = {
        date: slot.date,
        period: slot.period,
        startTime: slot.startTime,
        endTime: slot.endTime,
        slots: [],
        doctorIds: new Set(),
        remaining: 0,
        availableSlot: null,
      }
    }
    grouped[key].slots.push(slot)
    grouped[key].doctorIds.add(slot.doctorId)
    grouped[key].remaining += remaining
    if (!grouped[key].availableSlot && remaining > 0) grouped[key].availableSlot = slot
  }
  return Object.values(grouped)
    .sort((a, b) => a.date.localeCompare(b.date) || a.startTime.localeCompare(b.startTime))
    .reduce((acc, group) => {
      const item = { ...group, doctorCount: group.doctorIds.size }
      delete item.doctorIds
      acc[item.date] = [...(acc[item.date] || []), item]
      return acc
    }, {})
}

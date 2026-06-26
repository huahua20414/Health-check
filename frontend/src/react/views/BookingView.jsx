import React, { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, Card, Field, PageHeader, Select, TextInput, Textarea, StatusTag } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { appointmentTypes, moneyText } from '../utils'

const bookingSteps = ['选择套餐', '选择机构', '选择号源', '确认提交']

export function BookingView() {
  const h = useHealth()
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  useEffect(() => {
    h.loadSlotsPage({ page: 1, pageSize: 2000, status: 'available', availableOnly: 'true', fromDate: localDateValue(new Date()) }, 'bookingSlots', 'slots').catch((e) => h.notify('error', e.message))
    h.loadFamilyMembersPage({ page: 1, pageSize: 50 }).catch((e) => h.notify('error', e.message))
  }, [])
  const form = h.forms.appointment
  const selectedPackage = h.packages.find((pkg) => pkg.id === Number(form.packageId))
  const selectedPackageItems = [...(selectedPackage?.packageItems || [])].sort((a, b) => Number(a.sortOrder || 0) - Number(b.sortOrder || 0) || Number(a.id || 0) - Number(b.id || 0))
  const selectedPackageItemIds = new Set((form.selectedPackageItemIds || []).map(Number))
  const optionalAmount = selectedPackageItems.filter((item) => !item.required && selectedPackageItemIds.has(item.id)).reduce((sum, item) => sum + Number(item.item?.price || 0), 0)
  const selectedInstitution = h.institutions.find((item) => item.id === Number(form.institutionId))
  const selectedSlot = h.slots.find((slot) => slot.id === Number(form.slotId))
  const selectedMember = h.familyMembers.find((member) => member.id === Number(form.familyMemberId))
  const selectedCoupon = h.activeCoupons.find((coupon) => coupon.id === Number(form.couponId))
  const days = useMemo(() => nextDays(14), [])
  const filteredSlots = h.slots.filter((slot) => {
    if (form.institutionId && slot.institutionId !== Number(form.institutionId)) return false
    if (selectedPackage?.category && slot.category !== selectedPackage.category) return false
    return days.some((day) => day.value === slot.date)
  })
  const firstAvailableDate = filteredSlots[0]?.date || ''
  const visibleDate = form.date || firstAvailableDate || days[0]?.value || ''
  const slotsByTime = groupSlotsByDateTime(filteredSlots)
  const selectedDaySlots = slotsByTime[visibleDate] || []
  const canNext = [Boolean(form.packageId), Boolean(form.institutionId), Boolean(form.slotId || form.period), true][step]
  const next = () => {
    if (!canNext) {
      h.notify('warn', ['请选择套餐', '请选择体检机构', '请选择日期和号源', ''][step])
      return
    }
    setStep((current) => Math.min(current + 1, bookingSteps.length - 1))
  }
  const selectPackage = (pkg) => {
    const requiredIds = (pkg.packageItems || []).filter((item) => item.required).map((item) => item.id)
    h.updateForm('appointment', { packageId: pkg.id, selectedPackageItemIds: requiredIds, slotId: '', date: '', period: '' })
  }
  const toggleOptionalItem = (item) => {
    if (item.required) return
    const current = new Set((form.selectedPackageItemIds || []).map(Number))
    if (current.has(item.id)) current.delete(item.id)
    else current.add(item.id)
    for (const required of selectedPackageItems.filter((link) => link.required)) current.add(required.id)
    h.updateForm('appointment', { selectedPackageItemIds: Array.from(current) })
  }
  const submit = async () => {
    if (!form.packageId || !form.institutionId || (!form.slotId && !form.period)) {
      h.notify('warn', '请先完成套餐、机构和号源选择')
      return
    }
    await h.createAppointment()
    navigate('/my-appointments')
  }
  return (
    <>
      <PageHeader title="预约体检" subtitle="选择机构、套餐和未来两周号源，医生由后端按可用号自动分配。" />
      <div className="steps">{bookingSteps.map((label, index) => <span key={label} className={index === step ? 'active' : index < step ? 'done' : ''}>{index + 1} {label}</span>)}</div>
      <Card title={bookingSteps[step]} className="booking-step-card">
        {step === 0 && (
          <>
            <div className="package-grid booking-package-grid">
              {h.packages.map((pkg) => (
                <button key={pkg.id} className={`choice-card ${Number(form.packageId) === pkg.id ? 'is-selected' : ''}`} onClick={() => selectPackage(pkg)}>
                  <span>{pkg.category || '综合体检'}</span>
                  <strong>{pkg.name}</strong>
                  <small>{pkg.description || pkg.items}</small>
                  {!!pkg.packageItems?.length && <small>{pkg.packageItems.filter((item) => item.required).length} 项必选 · {pkg.packageItems.filter((item) => !item.required).length} 项可选</small>}
                  <b>{moneyText(pkg.price)}</b>
                </button>
              ))}
            </div>
            {!h.packages.length && <p className="muted">暂无可预约套餐。</p>}
          </>
        )}
        {step === 1 && (
          <div className="choice-grid">
            {h.institutions.map((institution) => (
              <button key={institution.id} className={`choice-card ${Number(form.institutionId) === institution.id ? 'is-selected' : ''}`} onClick={() => h.updateForm('appointment', { institutionId: institution.id, slotId: '', date: '', period: '' })}>
                <span>{institution.openHours || '营业中'}</span>
                <strong>{institution.name}</strong>
                <small>{institution.address}</small>
                <small>{institution.phone}</small>
              </button>
            ))}
          </div>
        )}
        {step === 2 && (
          <>
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
              if (full) {
                return (
                  <div key={`${group.date}-${group.startTime}`} className="slot-card is-full">
                    <strong>{group.startTime}-{group.endTime}</strong>
                    <span>{group.doctorCount} 位医生 · 已满</span>
                    <StatusTag status="full" />
                    <Button size="sm" variant="secondary" loading={h.loading.appointment} onClick={() => h.joinWaitlist(chosen).then(() => navigate('/my-appointments')).catch((e) => h.notify('error', e.message))}>加入候补</Button>
                  </div>
                )
              }
              return (
                <button key={`${group.date}-${group.startTime}`} className={`slot-card ${Number(form.slotId) === chosen.id ? 'is-selected' : ''}`} onClick={() => h.updateForm('appointment', { slotId: chosen.id, date: group.date, period: group.period || group.startTime })}>
                  <strong>{group.startTime}-{group.endTime}</strong>
                  <span>{group.doctorCount} 位医生 · 余 {group.remaining}</span>
                  <StatusTag status="available" />
                </button>
              )
            })}
          </div>
          {!selectedDaySlots.length && <p className="muted">当前筛选下暂无号源，请切换机构或套餐。</p>}
          </>
        )}
        {step === 3 && (
          <div className="booking-confirm">
            <ConfirmRow label="预约类型" value={form.appointmentType} />
            <ConfirmRow label="套餐" value={selectedPackage ? `${selectedPackage.name} · ${moneyText(selectedPackage.price)}` : '未选择'} />
            {optionalAmount > 0 && <ConfirmRow label="可选项目" value={`加选 ${moneyText(optionalAmount)}`} />}
            <ConfirmRow label="机构" value={selectedInstitution?.name || '未选择'} />
            <ConfirmRow label="日期" value={form.date || '未选择'} />
            <ConfirmRow label="时间" value={selectedSlot ? `${selectedSlot.startTime}-${selectedSlot.endTime}` : form.period || '未选择'} />
            <ConfirmRow label="体检人" value={selectedMember ? `${selectedMember.name} · ${selectedMember.relation}` : '本人'} />
            {!!selectedPackageItems.length && (
              <div className="package-item-picker">
                <div className="package-item-picker-head"><span>套餐项目</span><strong>必选固定，可选项目按需勾选</strong></div>
                {selectedPackageItems.map((link) => {
                  const checked = link.required || selectedPackageItemIds.has(link.id)
                  return (
                    <label key={link.id} className={`package-item-option ${checked ? 'is-checked' : ''} ${link.required ? 'is-required' : ''}`}>
                      <input type="checkbox" checked={checked} disabled={link.required} onChange={() => toggleOptionalItem(link)} />
                      <span>{link.sortOrder || '-'}</span>
                      <strong>{link.item?.name || link.itemId}</strong>
                      <small>{link.required ? '必选' : `可选 · ${moneyText(link.item?.price || 0)}`}</small>
                    </label>
                  )
                })}
              </div>
            )}
            <div className="form-grid compact booking-extra-form">
              <Field label="预约类型"><Select value={form.appointmentType} onChange={(e) => h.updateForm('appointment', { appointmentType: e.target.value })}>{appointmentTypes.map((t) => <option key={t}>{t}</option>)}</Select></Field>
              <Field label="家庭成员"><Select value={form.familyMemberId} onChange={(e) => h.updateForm('appointment', { familyMemberId: e.target.value })}><option value="">本人</option>{h.familyMembers.map((m) => <option key={m.id} value={m.id}>{m.name} · {m.relation}</option>)}</Select></Field>
              <Field label="优惠券"><Select value={form.couponId} onChange={(e) => h.updateForm('appointment', { couponId: e.target.value })}><option value="">不使用</option>{h.activeCoupons.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</Select></Field>
              <Field label="发票抬头"><TextInput value={form.invoiceTitle} onChange={(e) => h.updateForm('appointment', { invoiceTitle: e.target.value })} /></Field>
              <Field label="发票税号"><TextInput value={form.invoiceTaxNo} onChange={(e) => h.updateForm('appointment', { invoiceTaxNo: e.target.value })} /></Field>
              <Field label="备注"><Textarea value={form.note} onChange={(e) => h.updateForm('appointment', { note: e.target.value })} /></Field>
            </div>
            {selectedCoupon && <p className="muted">已选择优惠券：{selectedCoupon.name}</p>}
          </div>
        )}
        <div className="booking-actions">
          <Button variant="ghost" disabled={step === 0} onClick={() => setStep((current) => Math.max(0, current - 1))}>上一步</Button>
          {step < bookingSteps.length - 1 ? (
            <Button disabled={!canNext} onClick={next}>下一步</Button>
          ) : (
            <Button loading={h.loading.appointment} onClick={() => submit().catch((e) => h.notify('error', e.message))}>提交预约</Button>
          )}
        </div>
      </Card>
    </>
  )
}

function ConfirmRow({ label, value }) {
  return <div className="confirm-row"><span>{label}</span><strong>{value}</strong></div>
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

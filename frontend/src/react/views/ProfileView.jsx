import React, { useState } from 'react'
import { Button, Card, Field, Modal, PageHeader, Select, TextInput, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { calculateAgeFromIDCard, normalizeIDCard } from '../utils'

export function ProfileView() {
  const h = useHealth()
  const f = h.forms.profile
  const age = calculateAgeFromIDCard(f.idCard)
  const [modal, setModal] = useState('')
  const saveProfile = () => h.saveProfile().then(() => setModal('')).catch((e) => h.notify('error', e.message))
  const updateEmail = () => h.updateEmail().then(() => setModal('')).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="个人资料" subtitle="邮箱验证码变更，年龄由身份证自动计算。" />
      <div className="two-col">
        <Card title="基础资料" actions={<Button size="sm" onClick={() => setModal('profile')}>编辑资料</Button>}>
          <div className="detail-list">
            <p><span>姓名</span><strong>{f.name || '-'}</strong></p>
            <p><span>性别</span><strong>{f.gender || '-'}</strong></p>
            <p><span>年龄</span><strong>{age === null ? '-' : `${age} 岁`}</strong></p>
            <p><span>身份证号</span><strong>{f.idCard || '-'}</strong></p>
            <p><span>个人说明</span><strong>{f.bio || '-'}</strong></p>
          </div>
        </Card>
        <Card title="邮箱变更" actions={<Button size="sm" onClick={() => setModal('email')}>修改邮箱</Button>}>
          <div className="detail-list">
            <p><span>当前邮箱</span><strong>{h.currentUser?.email || f.email || '-'}</strong></p>
          </div>
        </Card>
      </div>
      <Modal open={modal === 'profile'} title="编辑资料" onClose={() => setModal('')} actions={<><Button variant="ghost" onClick={() => setModal('')}>取消</Button><Button loading={h.loading.profile} onClick={saveProfile}>保存资料</Button></>}>
        <div className="form-grid">
          <Field label="姓名"><TextInput value={f.name} onChange={(e) => h.updateForm('profile', { name: e.target.value })} /></Field>
          <Field label="性别"><Select value={f.gender} onChange={(e) => h.updateForm('profile', { gender: e.target.value })}><option value="">请选择</option><option value="男">男</option><option value="女">女</option></Select></Field>
          <Field label="身份证号"><TextInput value={f.idCard} maxLength={18} placeholder="例如 11010519491231002X" onChange={(e) => h.updateForm('profile', { idCard: normalizeIDCard(e.target.value) })} /></Field>
          <Field label="年龄"><TextInput value={age === null ? '身份证校验通过后自动计算' : `${age} 岁`} readOnly /></Field>
        </div>
        <Field label="个人说明"><Textarea value={f.bio} onChange={(e) => h.updateForm('profile', { bio: e.target.value })} /></Field>
      </Modal>
      <Modal open={modal === 'email'} title="修改邮箱" onClose={() => setModal('')} actions={<><Button variant="ghost" onClick={() => setModal('')}>取消</Button><Button loading={h.loading.emailUpdate} onClick={updateEmail}>验证并更新</Button></>}>
        <Field label="新邮箱"><TextInput value={h.forms.email.email} onChange={(e) => h.updateForm('email', { email: e.target.value })} /></Field>
        <Field label="验证码"><div className="inline-control"><TextInput value={h.forms.email.code} onChange={(e) => h.updateForm('email', { code: e.target.value })} /><Button variant="secondary" loading={h.loading.emailCode} onClick={() => h.sendEmailCode().catch((e) => h.notify('error', e.message))}>发送验证码</Button></div></Field>
      </Modal>
    </>
  )
}

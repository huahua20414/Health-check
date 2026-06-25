import React from 'react'
import { Button, Card, Field, PageHeader, Select, TextInput, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { calculateAgeFromIDCard } from '../utils'

export function ProfileView() {
  const h = useHealth()
  const f = h.forms.profile
  const age = calculateAgeFromIDCard(f.idCard)
  return (
    <>
      <PageHeader title="个人资料" subtitle="邮箱验证码变更，年龄由身份证自动计算。" />
      <div className="two-col">
        <Card title="基础资料">
          <div className="form-grid">
            <Field label="姓名"><TextInput value={f.name} onChange={(e) => h.updateForm('profile', { name: e.target.value })} /></Field>
            <Field label="性别"><Select value={f.gender} onChange={(e) => h.updateForm('profile', { gender: e.target.value })}><option value="">请选择</option><option value="男">男</option><option value="女">女</option></Select></Field>
            <Field label="身份证号"><TextInput value={f.idCard} onChange={(e) => h.updateForm('profile', { idCard: e.target.value })} /></Field>
            <Field label="年龄"><TextInput value={age === null ? '身份证校验通过后自动计算' : `${age} 岁`} readOnly /></Field>
          </div>
          <Field label="个人说明"><Textarea value={f.bio} onChange={(e) => h.updateForm('profile', { bio: e.target.value })} /></Field>
          <Button loading={h.loading.profile} onClick={() => h.saveProfile().catch((e) => h.notify('error', e.message))}>保存资料</Button>
        </Card>
        <Card title="邮箱变更">
          <Field label="新邮箱"><TextInput value={h.forms.email.email} onChange={(e) => h.updateForm('email', { email: e.target.value })} /></Field>
          <Field label="验证码"><div className="inline-control"><TextInput value={h.forms.email.code} onChange={(e) => h.updateForm('email', { code: e.target.value })} /><Button variant="secondary" loading={h.loading.emailCode} onClick={() => h.sendEmailCode().catch((e) => h.notify('error', e.message))}>发送验证码</Button></div></Field>
          <Button loading={h.loading.emailUpdate} onClick={() => h.updateEmail().catch((e) => h.notify('error', e.message))}>验证并更新</Button>
        </Card>
      </div>
    </>
  )
}

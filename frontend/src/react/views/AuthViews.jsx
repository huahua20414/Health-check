import React, { useMemo } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Button, Field, Select, TextInput } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { calculateAgeFromIDCard, devAuthEnabled, devAuthShortcutEmail, doctorDepartments, homePath } from '../utils.js'

export function AuthView() {
  const h = useHealth()
  const navigate = useNavigate()
  const shortcutMode = h.forms.login.email.trim().toLowerCase() === devAuthShortcutEmail
  const canSubmit = Boolean(h.forms.login.email && (shortcutMode ? ['1', '2', '3'].includes(h.forms.login.code) : devAuthEnabled || h.forms.login.code.length === 6))
  const codeText = h.loading.authCode ? '发送中' : h.authCodeCooldown > 0 ? `${h.authCodeCooldown} 秒后重发` : '发送验证码'
  function updateCode(value) {
    const role = { 1: 'user', 2: 'doctor', 3: 'admin' }[value]
    h.updateForm('login', { code: value, ...(role ? { role } : {}) })
  }
  async function submit() {
    try {
      const user = await h.login()
      if (user) navigate(homePath(user.role))
    } catch (error) {
      h.notify('error', error.message)
    }
  }
  return (
    <main className="auth-page">
      <section className="auth-panel">
        <div className="auth-copy">
          <span className="brand-code">XIXIN</span>
          <h1>健康体检管理系统</h1>
          <p>邮箱验证码登录，注册后自动进入对应角色页面。</p>
        </div>
        <div className="auth-form">
          <h2>邮箱验证码登录</h2>
          {devAuthEnabled && <Field label="开发登录身份"><Select value={h.forms.login.role} onChange={(e) => h.updateForm('login', { role: e.target.value })}><option value="user">用户</option><option value="doctor">医生</option><option value="admin">管理员</option></Select></Field>}
          <Field label="邮箱"><TextInput value={h.forms.login.email} onChange={(e) => h.updateForm('login', { email: e.target.value })} placeholder="请输入邮箱" /></Field>
          <Field label={shortcutMode ? '入口码' : '邮箱验证码'}>
            <div className="inline-control">
              <TextInput value={h.forms.login.code} onChange={(e) => updateCode(e.target.value)} maxLength={shortcutMode ? 1 : 6} placeholder={shortcutMode ? '1 用户 / 2 医生 / 3 管理员' : '6 位验证码'} />
              {!shortcutMode && <Button variant="secondary" disabled={!h.forms.login.email || h.authCodeCooldown > 0} loading={h.loading.authCode} onClick={() => h.sendAuthEmailCode(h.forms.login.email).catch((e) => h.notify('error', e.message))}>{codeText}</Button>}
            </div>
          </Field>
          <Button className="full" loading={h.loading.login} disabled={!canSubmit} onClick={submit}>登录</Button>
          <div className="auth-links"><Link to="/register/user">注册用户</Link><Link to="/register/doctor">医生入驻</Link></div>
        </div>
      </section>
    </main>
  )
}

export function RegisterView() {
  const { role } = useParams()
  const isDoctor = role === 'doctor'
  const h = useHealth()
  const navigate = useNavigate()
  const formKey = isDoctor ? 'doctorRegister' : 'userRegister'
  const form = h.forms[formKey]
  const age = useMemo(() => calculateAgeFromIDCard(form.idCard), [form.idCard])
  const codeText = h.loading.authCode ? '发送中' : h.authCodeCooldown > 0 ? `${h.authCodeCooldown} 秒后重发` : '发送验证码'
  async function submit() {
    try {
      if (isDoctor) {
        await h.registerDoctor()
        navigate('/login')
      } else {
        const user = await h.registerUser()
        if (user) navigate(homePath(user.role))
      }
    } catch (error) {
      h.notify('error', error.message)
    }
  }
  return (
    <main className="auth-page">
      <section className="auth-panel">
        <div className="auth-copy">
          <span className="brand-code">XIXIN</span>
          <h1>{isDoctor ? '医生入驻申请' : '用户注册'}</h1>
          <p>{isDoctor ? '医生账号提交后由管理员审核。' : '注册成功后自动登录。'}</p>
        </div>
        <div className="auth-form">
          <h2>{isDoctor ? '医生注册' : '用户注册'}</h2>
          <Field label="姓名"><TextInput value={form.name} onChange={(e) => h.updateForm(formKey, { name: e.target.value })} /></Field>
          <Field label="邮箱"><TextInput value={form.email} onChange={(e) => h.updateForm(formKey, { email: e.target.value })} /></Field>
          <Field label="邮箱验证码"><div className="inline-control"><TextInput value={form.code} onChange={(e) => h.updateForm(formKey, { code: e.target.value })} maxLength={6} /><Button variant="secondary" disabled={!form.email || h.authCodeCooldown > 0} loading={h.loading.authCode} onClick={() => h.sendAuthEmailCode(form.email).catch((e) => h.notify('error', e.message))}>{codeText}</Button></div></Field>
          {!isDoctor ? (
            <>
              <Field label="性别"><Select value={form.gender} onChange={(e) => h.updateForm(formKey, { gender: e.target.value })}><option value="">请选择</option><option value="男">男</option><option value="女">女</option></Select></Field>
              <Field label="身份证号"><TextInput value={form.idCard} maxLength={18} onChange={(e) => h.updateForm(formKey, { idCard: e.target.value })} /></Field>
              <Field label="年龄"><TextInput value={age === null ? '身份证校验通过后自动计算' : `${age} 岁`} readOnly /></Field>
            </>
          ) : (
            <>
              <Field label="工号"><TextInput value={form.employeeNo} onChange={(e) => h.updateForm(formKey, { employeeNo: e.target.value })} /></Field>
              <Field label="科室"><Select value={form.department} onChange={(e) => h.updateForm(formKey, { department: e.target.value })}><option value="">请选择科室</option>{doctorDepartments.map((d) => <option key={d} value={d}>{d}</option>)}</Select></Field>
              <Field label="职称"><TextInput value={form.title} onChange={(e) => h.updateForm(formKey, { title: e.target.value })} /></Field>
            </>
          )}
          <Button className="full" loading={h.loading.register} onClick={submit}>提交注册</Button>
          <div className="auth-links"><Link to="/login">返回登录</Link></div>
        </div>
      </section>
    </main>
  )
}

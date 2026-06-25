import React from 'react'
import { Button, Card, Field, PageHeader, PaginatedTable, Select, TextInput } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { calculateAgeFromIDCard } from '../utils'

export function FamilyView() {
  const h = useHealth()
  const f = h.forms.familyMember
  return (
    <>
      <PageHeader title="家庭成员" subtitle="用于代预约和成员报告管理。" />
      <div className="two-col">
        <Card title="成员列表"><PaginatedTable columns={[{ title: '姓名', render: (r) => r.name }, { title: '关系', render: (r) => r.relation }, { title: '年龄', render: (r) => r.age || calculateAgeFromIDCard(r.idCard) || '-' }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => h.updateForm('familyMember', r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.deleteFamilyMember(r).catch((e) => h.notify('error', e.message))}>删除</Button></div> }]} rows={h.familyMembers} /></Card>
        <Card title={f.id ? '编辑成员' : '新增成员'}>
          <Field label="姓名"><TextInput value={f.name} onChange={(e) => h.updateForm('familyMember', { name: e.target.value })} /></Field>
          <Field label="关系"><TextInput value={f.relation} onChange={(e) => h.updateForm('familyMember', { relation: e.target.value })} /></Field>
          <Field label="性别"><Select value={f.gender} onChange={(e) => h.updateForm('familyMember', { gender: e.target.value })}><option value="">请选择</option><option value="男">男</option><option value="女">女</option></Select></Field>
          <Field label="身份证号"><TextInput value={f.idCard} onChange={(e) => h.updateForm('familyMember', { idCard: e.target.value })} /></Field>
          <Field label="电话"><TextInput value={f.phone} onChange={(e) => h.updateForm('familyMember', { phone: e.target.value })} /></Field>
          <Button loading={h.loading.familyMember} onClick={() => h.saveFamilyMember().catch((e) => h.notify('error', e.message))}>保存成员</Button>
        </Card>
      </div>
    </>
  )
}

import React, { useEffect, useState } from 'react'
import { Button, Card, Field, Modal, PageHeader, PaginatedTable, Select, TextInput } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { calculateAgeFromIDCard } from '../utils'

export function FamilyView() {
  const h = useHealth()
  const f = h.forms.familyMember
  const [open, setOpen] = useState(false)
  useEffect(() => {
    h.loadFamilyMembersPage({ page: 1, pageSize: 20 }).catch((e) => h.notify('error', e.message))
  }, [])
  const openCreate = () => { h.resetForm('familyMember'); setOpen(true) }
  const openEdit = (member) => { h.updateForm('familyMember', member); setOpen(true) }
  const save = () => h.saveFamilyMember().then(() => setOpen(false)).catch((e) => h.notify('error', e.message))
  return (
    <>
      <PageHeader title="家庭成员" subtitle="用于代预约和成员报告管理。" actions={<Button onClick={openCreate}>新增成员</Button>} />
      <Card title="成员列表"><PaginatedTable columns={[{ title: '姓名', render: (r) => r.name }, { title: '关系', render: (r) => r.relation }, { title: '年龄', render: (r) => r.age || calculateAgeFromIDCard(r.idCard) || '-' }, { title: '操作', render: (r) => <div className="row-actions"><Button size="sm" variant="ghost" onClick={() => openEdit(r)}>编辑</Button><Button size="sm" variant="danger" onClick={() => h.deleteFamilyMember(r).catch((e) => h.notify('error', e.message))}>删除</Button></div> }]} rows={h.familyMembers} /></Card>
      <Modal open={open} title={f.id ? '编辑成员' : '新增成员'} onClose={() => setOpen(false)} actions={<><Button variant="ghost" onClick={() => setOpen(false)}>取消</Button><Button loading={h.loading.familyMember} onClick={save}>保存成员</Button></>}>
        <div className="form-grid">
          <Field label="姓名"><TextInput value={f.name} onChange={(e) => h.updateForm('familyMember', { name: e.target.value })} /></Field>
          <Field label="关系"><TextInput value={f.relation} onChange={(e) => h.updateForm('familyMember', { relation: e.target.value })} /></Field>
          <Field label="性别"><Select value={f.gender} onChange={(e) => h.updateForm('familyMember', { gender: e.target.value })}><option value="">请选择</option><option value="男">男</option><option value="女">女</option></Select></Field>
          <Field label="身份证号"><TextInput value={f.idCard} onChange={(e) => h.updateForm('familyMember', { idCard: e.target.value })} /></Field>
          <Field label="电话"><TextInput value={f.phone} onChange={(e) => h.updateForm('familyMember', { phone: e.target.value })} /></Field>
        </div>
      </Modal>
    </>
  )
}

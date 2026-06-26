import React from 'react'
import { useNavigate } from 'react-router-dom'
import { Button, Card, DataTable, PageHeader, StatusTag } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { formatDate, moneyText } from '../utils'

export function PackagesView() {
  const h = useHealth()
  const navigate = useNavigate()
  const selectPackage = (pkg) => {
    h.recordPackageBrowse(pkg)
    h.updateForm('appointment', { packageId: pkg.id, slotId: '', date: '', period: '' })
    navigate('/booking')
  }
  return (
    <>
      <PageHeader title="体检套餐" subtitle="选择套餐后进入预约流程。" />
      <div className="package-grid">
        {h.packages.map((pkg) => {
          return <Card key={pkg.id} className="package-card"><div className="package-head"><span>{pkg.category || '综合体检'}</span><StatusTag status={pkg.status || 'active'} /></div><h3>{pkg.name}</h3><p>{pkg.description || pkg.items}</p><div className="package-foot"><strong>{moneyText(pkg.price)}</strong><Button onClick={() => selectPackage(pkg)}>选择</Button></div></Card>
        })}
      </div>
      <div className="two-col">
        <Card title="热门套餐"><DataTable columns={[{ title: '套餐', render: (r) => r.name }, { title: '价格', render: (r) => moneyText(r.price) }]} rows={h.popularPackages.slice(0, 5)} /></Card>
        <Card title="浏览记录"><DataTable columns={[{ title: '套餐', render: (r) => r.package?.name || r.packageName || '-' }, { title: '时间', render: (r) => formatDate(r.createdAt) }]} rows={h.browseHistories.slice(0, 5)} /></Card>
      </div>
    </>
  )
}

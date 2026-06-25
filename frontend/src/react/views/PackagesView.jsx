import React from 'react'
import { Button, Card, DataTable, PageHeader, StatusTag } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { moneyText } from '../utils'

export function PackagesView() {
  const h = useHealth()
  return (
    <>
      <PageHeader title="体检套餐" subtitle="套餐、推荐、收藏和浏览记录来自后端接口。" />
      <div className="package-grid">
        {h.packages.map((pkg) => {
          const fav = h.favorites.some((item) => item.packageId === pkg.id)
          return <Card key={pkg.id} className="package-card"><div className="package-head"><span>{pkg.category || '综合体检'}</span><StatusTag status={pkg.status || 'active'} /></div><h3>{pkg.name}</h3><p>{pkg.description || pkg.items}</p><div className="package-foot"><strong>{moneyText(pkg.price)}</strong><Button variant={fav ? 'secondary' : 'ghost'} onClick={() => h.toggleFavorite(pkg)}>{fav ? '已收藏' : '收藏'}</Button><Button onClick={() => { h.recordPackageBrowse(pkg); h.updateForm('appointment', { packageId: pkg.id }) }}>选择</Button></div></Card>
        })}
      </div>
      <div className="two-col">
        <Card title="热门套餐"><DataTable columns={[{ title: '套餐', render: (r) => r.name }, { title: '价格', render: (r) => moneyText(r.price) }]} rows={h.popularPackages.slice(0, 5)} /></Card>
        <Card title="浏览记录"><DataTable columns={[{ title: '套餐', render: (r) => r.package?.name || r.packageName || '-' }, { title: '时间', render: (r) => r.createdAt ? new Date(r.createdAt).toLocaleString('zh-CN') : '-' }]} rows={h.browseHistories.slice(0, 5)} /></Card>
      </div>
    </>
  )
}

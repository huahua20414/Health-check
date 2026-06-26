import React, { useEffect } from 'react'
import { Button, Card, PageHeader, PaginatedTable } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'
import { formatDate } from '../utils'

export function ReportsView() {
  const h = useHealth()
  useEffect(() => {
    h.loadReportsPage({ page: 1, pageSize: 20 }).catch((e) => h.notify('error', e.message))
  }, [])
  return (
    <>
      <PageHeader title="我的报告" subtitle="查看和下载体检报告。" />
      <Card title="报告列表">
        <PaginatedTable columns={[{ title: '报告编号', render: (r) => r.reportNo || r.id }, { title: '套餐', render: (r) => r.appointment?.package?.name || '-' }, { title: '医生', render: (r) => r.doctor?.name || '-' }, { title: '时间', render: (r) => formatDate(r.createdAt) }, { title: '操作', render: (r) => <Button size="sm" onClick={() => h.downloadReport(r)}>下载报告</Button> }]} rows={h.reports} />
      </Card>
    </>
  )
}

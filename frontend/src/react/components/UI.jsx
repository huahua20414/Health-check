import React, { useEffect, useState } from 'react'
import { statusKind, statusText } from '../utils'

export function Button({ children, variant = 'primary', size = 'md', loading, className = '', ...props }) {
  return <button className={`btn btn-${variant} btn-${size} ${className}`} disabled={loading || props.disabled} {...props}>{loading ? '处理中...' : children}</button>
}

export function Card({ title, subtitle, actions, children, className = '' }) {
  return (
    <section className={`card ${className}`}>
      {(title || actions) && <div className="card-head"><div>{title && <h3>{title}</h3>}{subtitle && <p>{subtitle}</p>}</div><div className="card-actions">{actions}</div></div>}
      {children}
    </section>
  )
}

export function Modal({ open, title, actions, children, onClose, className = '', backdropClassName = '' }) {
  if (!open) return null
  return (
    <div className={`modal-backdrop ${backdropClassName}`} role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose?.() }}>
      <section className={`modal-panel ${className}`} role="dialog" aria-modal="true" aria-label={title}>
        <div className="modal-head">
          <h3>{title}</h3>
          <button className="modal-close" type="button" onClick={onClose} aria-label="关闭">×</button>
        </div>
        <div className="modal-body">{children}</div>
        {actions && <div className="modal-actions">{actions}</div>}
      </section>
    </div>
  )
}

export function Metric({ label, value, tone = 'cyan' }) {
  return <div className={`metric metric-${tone}`}><strong>{value}</strong><span>{label}</span></div>
}

export function StatusTag({ status, children }) {
  return <span className={`tag tag-${statusKind(status)}`}>{children || statusText(status)}</span>
}

export function Empty({ text = '暂无数据' }) {
  return <div className="empty-state">{text}</div>
}

export function Field({ label, children }) {
  return <label className="field"><span>{label}</span>{children}</label>
}

export function TextInput(props) {
  return <input className="input" {...props} />
}

export function Select({ children, ...props }) {
  return <select className="input" {...props}>{children}</select>
}

export function Textarea(props) {
  return <textarea className="input textarea" {...props} />
}

export function DataTable({ columns, rows, empty = '暂无数据', loading = false, loadingText = '数据加载中...' }) {
  return (
    <div className="table-wrap">
      <table className="data-table">
        <thead><tr>{columns.map((col) => <th key={col.key || col.title}>{col.title}</th>)}</tr></thead>
        <tbody>
          {loading && !rows?.length && <tr><td colSpan={columns.length}><Empty text={loadingText} /></td></tr>}
          {!loading && !rows?.length && <tr><td colSpan={columns.length}><Empty text={empty} /></td></tr>}
          {rows?.map((row, index) => <tr key={row.__rowKey || `${row.id || 'row'}-${index}`}>{columns.map((col) => <td key={col.key || col.title}>{col.render ? col.render(row, index) : row[col.key]}</td>)}</tr>)}
        </tbody>
      </table>
    </div>
  )
}

export function Pagination({ page, pageSize, total, onPageChange, onPageSizeChange }) {
  const pageCount = Math.max(1, Math.ceil((total || 0) / (pageSize || 10)))
  const safePage = Math.min(Math.max(page || 1, 1), pageCount)
  const start = total ? (safePage - 1) * pageSize + 1 : 0
  const end = Math.min(total || 0, safePage * pageSize)
  return (
    <div className="pagination-bar">
      <span>{total ? `${start}-${end} / ${total}` : '0 / 0'}</span>
      <div className="pagination-actions">
        <button className="btn btn-ghost btn-sm" onClick={() => onPageChange(Math.max(1, safePage - 1))} disabled={safePage <= 1}>上一页</button>
        <button className="btn btn-ghost btn-sm" onClick={() => onPageChange(Math.min(pageCount, safePage + 1))} disabled={safePage >= pageCount}>下一页</button>
        <select className="input pagination-size" value={pageSize} onChange={(e) => onPageSizeChange(Number(e.target.value))}>
          {[10, 20, 50].map((size) => <option key={size} value={size}>{size} 条/页</option>)}
        </select>
      </div>
    </div>
  )
}

export function PaginatedTable({ columns, rows, empty = '暂无数据', initialPageSize = 10, loading = false }) {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(initialPageSize)
  const total = rows?.length || 0
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  useEffect(() => { setPage((current) => Math.min(current, pageCount)) }, [pageCount])
  const safePage = Math.min(Math.max(page, 1), pageCount)
  const pageRows = (rows || []).slice((safePage - 1) * pageSize, safePage * pageSize)
  return (
    <>
      <DataTable columns={columns} rows={pageRows} empty={empty} loading={loading} />
      <Pagination page={safePage} pageSize={pageSize} total={total} onPageChange={setPage} onPageSizeChange={setPageSize} />
    </>
  )
}

export function RemoteTable({ columns, rows, pagination, empty = '暂无数据', loading = false, onPageChange, onPageSizeChange }) {
  const page = pagination?.page || 1
  const pageSize = pagination?.pageSize || 10
  const total = pagination?.total ?? rows?.length ?? 0
  const isLoading = loading || Boolean(pagination?.loading)
  return (
    <>
      <DataTable columns={columns} rows={rows || []} empty={empty} loading={isLoading} />
      <Pagination page={page} pageSize={pageSize} total={total} onPageChange={onPageChange} onPageSizeChange={onPageSizeChange} />
    </>
  )
}

export function PageHeader({ title, subtitle, actions }) {
  return <div className="page-title"><div><h1>{title}</h1>{subtitle && <p>{subtitle}</p>}</div><div className="page-actions">{actions}</div></div>
}

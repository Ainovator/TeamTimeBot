import { Icon } from './Icon'
import { Select } from './Select'

export function Pagination({ total, page, pageSize, onPageChange, onPageSizeChange, navigationLabel = 'Страницы событий' }: {
  total: number
  page: number
  pageSize: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
  navigationLabel?: string
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  const visible = Array.from(new Set([1, page - 1, page, page + 1, pages]))
    .filter(value => value >= 1 && value <= pages).sort((a, b) => a - b)
  const controls: (number | string)[] = []
  visible.forEach((value, index) => {
    const previous = visible[index - 1]
    if (previous && value - previous === 2) controls.push(previous + 1)
    else if (previous && value - previous > 2) controls.push(`gap-${value}`)
    controls.push(value)
  })

  return <div className="studio-pagination">
    <p className="studio-pagination-summary" role="status">{total ? `${(page - 1) * pageSize + 1}–${Math.min(page * pageSize, total)} из ${total}` : '0 из 0'}</p>
    <label className="studio-pagination-size"><span>На странице</span><Select compact value={pageSize} onChange={event => onPageSizeChange(Number(event.target.value))}>
      <option value={12}>12</option><option value={24}>24</option><option value={28}>28</option>
    </Select></label>
    <nav aria-label={navigationLabel} className="studio-pagination-pages">
      <button type="button" className="btn-secondary" aria-label="Предыдущая страница" disabled={page <= 1} onClick={() => onPageChange(page - 1)}><Icon name="back" size={16}/></button>
      {controls.map(value => typeof value === 'number'
        ? <button type="button" key={value} className="btn-secondary" aria-label={`Страница ${value}`} aria-current={page === value ? 'page' : undefined} onClick={() => onPageChange(value)}>{value}</button>
        : <span key={value} aria-hidden="true">…</span>)}
      <button type="button" className="btn-secondary" aria-label="Следующая страница" disabled={page >= pages} onClick={() => onPageChange(page + 1)}><Icon name="arrow" size={16}/></button>
    </nav>
  </div>
}

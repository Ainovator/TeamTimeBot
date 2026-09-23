import type { ReactNode } from 'react'
import { Icon } from './components'

export function PageHeading({ eyebrow, title, description, action }: { eyebrow: string; title: string; description: string; action?: ReactNode }) {
  return <div className="dl-page-heading"><div><p className="dl-eyebrow">{eyebrow}</p><h1>{title}<span>.</span></h1><p>{description}</p></div>{action}</div>
}

export function MiniStats({ items }: { items: { label: string; value: string; note: string; icon: string }[] }) {
  return <div className="dl-product-stats">{items.map(item => <div className="dl-panel" key={item.label}><span>{item.label}<Icon name={item.icon} size={18}/></span><strong>{item.value}</strong><small>{item.note}</small></div>)}</div>
}

export function BackButton({ onClick, children }: { onClick: () => void; children: ReactNode }) {
  return <button className="dl-text-button dl-back-button" onClick={onClick}><Icon name="back" size={17}/>{children}</button>
}

export function Notice({ children }: { children: ReactNode }) {
  return children ? <p className="dl-inline-notice" role="status"><Icon name="check" size={17}/>{children}</p> : null
}

export function Empty({ children }: { children: ReactNode }) { return <div className="dl-empty">{children}</div> }

export function Search({ value, onChange, placeholder }: { value: string; onChange: (value: string) => void; placeholder: string }) {
  return <label className="dl-search"><Icon name="search"/><input aria-label={placeholder} placeholder={placeholder} value={value} onChange={e => onChange(e.target.value)}/></label>
}

export const money = (value: number) => `${value.toLocaleString('ru-RU')} ₽`

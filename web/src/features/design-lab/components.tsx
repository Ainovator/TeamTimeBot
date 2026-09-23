import { useEffect, useRef, type ReactNode } from 'react'
import { players } from './demoData'

export { Icon } from '../../components/Icon'
import { Icon } from '../../components/Icon'

export function Avatar({ index = 0, large = false }: { index?: number; large?: boolean }) {
  const p = players[index % players.length]
  return <span className={`dl-avatar dl-avatar-${p.color}${large ? ' dl-avatar-large' : ''}`} title={p.name}>{p.initials}</span>
}

export function AvatarStack({ count = 16 }: { count?: number }) {
  if (!count) return <span className="dl-demo-note">Будьте первым участником</span>
  return <div className="dl-avatar-stack" aria-label={`${count} участников`}>{Array.from({length: Math.min(4, count)}, (_,i) => <Avatar key={i} index={i}/>)}{count>4 && <span className="dl-avatar dl-avatar-more">+{count-4}</span>}</div>
}

export function Badge({ children, muted = false }: { children: ReactNode; muted?: boolean }) { return <span className={`dl-badge${muted ? ' dl-badge-muted' : ''}`}><span/>{children}</span> }

export function SectionTitle({ title, action, onClick }: { title: string; action?: string; onClick?: () => void }) {
  return <div className="dl-section-title"><h2>{title}</h2>{action && <button className="dl-text-button" onClick={onClick}>{action}<Icon name="arrow" size={16}/></button>}</div>
}

export function DemoDialog({ title, close, children }: { title: string; close: () => void; children: ReactNode }) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => { const dialog = ref.current; dialog?.showModal(); return () => dialog?.close() }, [])
  return <dialog ref={ref} className="dl-dialog" aria-label={title} onCancel={close} onClick={e => {if(e.target === e.currentTarget) close()}}><div className="dl-dialog-heading"><h2>{title}</h2><button className="dl-icon-button" aria-label="Закрыть окно" onClick={close}><Icon name="close"/></button></div>{children}</dialog>
}

import { useEffect, useRef, type ReactNode } from 'react'
import { Icon } from './Icon'

export function Modal({ title, close, children, busy = false }: { title: string; close: () => void; children: ReactNode; busy?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => { const dialog = ref.current; dialog?.showModal(); return () => dialog?.close() }, [])
  return <dialog ref={ref} className="studio-modal" aria-label={title} onCancel={event => { event.preventDefault(); if (!busy) close() }} onClick={event => { if (!busy && event.target === event.currentTarget) close() }}><div className="studio-modal-content"><header><h2>{title}</h2><button type="button" className="btn-icon btn-secondary" aria-label="Закрыть окно" disabled={busy} onClick={close}><Icon name="close" size={18}/></button></header>{children}</div></dialog>
}

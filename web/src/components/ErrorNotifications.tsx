import { useCallback, useEffect, useState, useSyncExternalStore } from 'react'
import { createPortal } from 'react-dom'
import { readableError } from '../app/errors'
import { Icon } from './Icon'
import './error-notifications.css'

type Notice = { id: number; message: string; revision: number }
let notices: Notice[] = []
let nextID = 0
const listeners = new Set<() => void>()
const subscribe = (listener: () => void) => { listeners.add(listener); return () => { listeners.delete(listener) } }
const snapshot = () => notices
const emit = () => listeners.forEach(listener => listener())
export function notifyError(error: unknown) {
  const message = readableError(error)
  const existing = notices.find(notice => notice.message === message)
  notices = existing
    ? notices.map(notice => notice.id === existing.id ? { ...notice, revision: notice.revision + 1 } : notice)
    : [...notices.slice(-2), { id: ++nextID, message, revision: 0 }]
  emit()
}
function dismiss(id: number) { notices = notices.filter(notice => notice.id !== id); emit() }

/** Preserve error state for loading/empty-state guards, but display each failure centrally. */
export function useErrorState() {
  const [error, setError] = useState('')
  const update = useCallback((message: string) => {
    const translated = message ? readableError(message) : ''
    setError(translated)
    if (translated) notifyError(translated)
  }, [])
  return [error, update] as const
}

function ErrorNotice({ notice }: { notice: Notice }) {
  const [hovered, setHovered] = useState(false)
  const [focused, setFocused] = useState(false)
  const paused = hovered || focused
  useEffect(() => {
    if (paused) return
    const timeout = window.setTimeout(() => dismiss(notice.id), 12000)
    return () => window.clearTimeout(timeout)
  }, [notice.id, notice.revision, paused])
  return <div className="tt-error-notice" onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)} onFocus={() => setFocused(true)} onBlur={event => { if (!event.currentTarget.contains(event.relatedTarget)) setFocused(false) }}>
    <span className="tt-error-symbol"><Icon name="error" size={20}/></span>
    <div role="alert" aria-atomic="true"><strong>Не удалось выполнить действие</strong><p>{notice.message}</p></div>
    <button type="button" aria-label="Закрыть уведомление об ошибке" onClick={() => dismiss(notice.id)}><Icon name="close" size={16}/></button>
  </div>
}

export function ErrorNotifications() {
  const current = useSyncExternalStore(subscribe, snapshot)
  const [target, setTarget] = useState<Element>(document.body)
  useEffect(() => {
    // Native modal dialogs occupy the browser's top layer. Put notices inside the active
    // dialog so they remain visible and their close buttons remain keyboard-accessible.
    const updateTarget = () => {
      const dialogs = document.querySelectorAll('dialog[open]')
      setTarget(dialogs[dialogs.length - 1] ?? document.body)
    }
    updateTarget()
    const observer = new MutationObserver(updateTarget)
    observer.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['open'] })
    return () => observer.disconnect()
  }, [])
  return createPortal(<div className="tt-error-notifications" aria-label="Уведомления об ошибках">{current.map(notice => <ErrorNotice key={notice.id} notice={notice}/>)}</div>, target)
}

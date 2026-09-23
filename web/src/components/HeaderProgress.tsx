import { useEffect, useRef, useState, type CSSProperties } from 'react'
import './headerProgress.css'

const MIN_VISIBLE_MS = 1200
const FADE_OUT_MS = 320

export function HeaderProgress({ active }: { active: boolean }) {
  const [visible, setVisible] = useState(active)
  const [fading, setFading] = useState(false)
  const startedAt = useRef<number | null>(null)

  useEffect(() => {
    if (active) {
      startedAt.current = performance.now()
      setVisible(true)
      setFading(false)
      return
    }
    if (startedAt.current === null) return

    // Finish a visible pass even when the request resolves almost immediately.
    // Only the indicator waits; fresh data and the refresh button are ready already.
    const remaining = Math.max(0, MIN_VISIBLE_MS - (performance.now() - startedAt.current))
    let hideTimer: number | undefined
    const timer = window.setTimeout(() => {
      setFading(true)
      hideTimer = window.setTimeout(() => {
        setVisible(false)
        setFading(false)
        startedAt.current = null
      }, FADE_OUT_MS)
    }, remaining)
    return () => {
      window.clearTimeout(timer)
      window.clearTimeout(hideTimer)
    }
  }, [active])

  if (!active && !visible) return null

  return (
    <div
      className={`studio-header-progress${fading && !active ? ' is-fading' : ''}`}
      style={{ '--header-progress-fade-duration': `${FADE_OUT_MS}ms` } as CSSProperties}
      role={active ? 'progressbar' : undefined}
      aria-label={active ? 'Обновляем данные команды' : undefined}
      aria-hidden={!active || undefined}
    >
      <span className="studio-header-progress-runner" />
    </div>
  )
}

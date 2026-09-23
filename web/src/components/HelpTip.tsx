import { useEffect, useId, useRef, useState } from 'react'

export function HelpTip({ label, children }: { label: string; children: string }) {
  const [open, setOpen] = useState(false)
  const id = useId()
  const root = useRef<HTMLSpanElement>(null)
  const trigger = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) return
    const dismiss = (event: PointerEvent) => {
      if (!root.current?.contains(event.target as Node)) setOpen(false)
    }
    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
        trigger.current?.focus()
      }
    }
    document.addEventListener('pointerdown', dismiss)
    document.addEventListener('keydown', escape)
    return () => {
      document.removeEventListener('pointerdown', dismiss)
      document.removeEventListener('keydown', escape)
    }
  }, [open])

  return (
    <span className="field-help" ref={root} onBlur={(event) => {
      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setOpen(false)
    }}>
      <button ref={trigger} type="button" className="field-help-trigger" aria-label={label}
        aria-expanded={open} aria-controls={id} aria-describedby={open ? id : undefined}
        onClick={() => setOpen((value) => !value)}>?</button>
      <span id={id} className="field-help-popup" hidden={!open}>{children}</span>
    </span>
  )
}

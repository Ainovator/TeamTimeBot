import { notifyError } from './ErrorNotifications'
import { useEffect, useId, useLayoutEffect, useRef, useState, type CSSProperties, type KeyboardEvent, type SelectHTMLAttributes } from 'react'
import { createPortal } from 'react-dom'
import './select.css'

type Option = { value: string; label: string; disabled: boolean; group: string }

/** Keeps the native form control and change events; presents an accessible, themed menu. */
export function Select({ children, className = '', style, compact = false, ...props }: SelectHTMLAttributes<HTMLSelectElement> & { compact?: boolean }) {
  const native = useRef<HTMLSelectElement>(null)
  const trigger = useRef<HTMLButtonElement>(null)
  const popup = useRef<HTMLDivElement>(null)
  const [options, setOptions] = useState<Option[]>([])
  const [selected, setSelected] = useState('')
  const [label, setLabel] = useState('Выберите вариант')
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(-1)
  const [invalid, setInvalid] = useState('')
  const [position, setPosition] = useState<CSSProperties>({})
  const [portal, setPortal] = useState<HTMLElement | null>(null)
  const typed = useRef({ text: '', at: 0 })
  const id = useId()

  useLayoutEffect(() => {
    const el = native.current
    if (!el) return
    const next = Array.from(el.options).map(option => ({ value: option.value, label: option.label, disabled: option.disabled || Boolean(option.closest('optgroup')?.disabled), group: option.closest('optgroup')?.label ?? '' }))
    setOptions(prev => JSON.stringify(prev) === JSON.stringify(next) ? prev : next)
    setSelected(el.value)
    const fieldLabel = el.labels?.[0]?.cloneNode(true) as HTMLElement | undefined
    fieldLabel?.querySelectorAll('.tt-select, select').forEach(node => node.remove())
    setLabel(props['aria-label'] || fieldLabel?.textContent?.trim() || props.title || 'Выберите вариант')
  }, [children, props.value, props.defaultValue, props['aria-label'], props.title])

  useEffect(() => {
    const form = native.current?.form
    const reset = () => { setOpen(false); requestAnimationFrame(() => setSelected(native.current?.value ?? '')) }
    form?.addEventListener('reset', reset)
    return () => form?.removeEventListener('reset', reset)
  }, [])

  useEffect(() => {
    if (!open) return
    const outside = (event: PointerEvent) => { if (!popup.current?.contains(event.target as Node) && !trigger.current?.contains(event.target as Node)) setOpen(false) }
    const scroll = (event: Event) => { if (!popup.current?.contains(event.target as Node)) setOpen(false) }
    const close = () => setOpen(false)
    document.addEventListener('pointerdown', outside)
    document.addEventListener('scroll', scroll, true)
    window.addEventListener('resize', close)
    return () => { document.removeEventListener('pointerdown', outside); document.removeEventListener('scroll', scroll, true); window.removeEventListener('resize', close) }
  }, [open])

  useEffect(() => {
    if (open) popup.current?.querySelector<HTMLElement>(`[data-index="${active}"]`)?.scrollIntoView({ block: 'nearest' })
  }, [open, active])

  function show(index?: number) {
    if (props.disabled || !trigger.current) return
    const rect = trigger.current.getBoundingClientRect()
    const theme = getComputedStyle(trigger.current)
    const below = window.innerHeight - rect.bottom - 12
    const above = rect.top - 12
    const upwards = below < 210 && above > below
    const height = Math.max(80, Math.min(320, upwards ? above : below))
    const width = Math.min(compact ? rect.width : Math.max(rect.width, 220), window.innerWidth - 24)
    setPosition({
      position: 'fixed', width, maxHeight: height, left: Math.max(12, Math.min(rect.left, window.innerWidth - width - 12)),
      ...(upwards ? { bottom: window.innerHeight - rect.top + 6 } : { top: rect.bottom + 6 }),
      '--tt-select-bg': theme.getPropertyValue('--tt-select-bg'), '--tt-select-text': theme.getPropertyValue('--tt-select-text'),
      '--tt-select-muted': theme.getPropertyValue('--tt-select-muted'), '--tt-select-border': theme.getPropertyValue('--tt-select-border'),
      '--tt-select-accent': theme.getPropertyValue('--tt-select-accent'), '--tt-select-soft': theme.getPropertyValue('--tt-select-soft'),
      fontFamily: theme.fontFamily,
    } as CSSProperties & { [key: `--${string}`]: string })
    setPortal(trigger.current.closest('dialog') || document.body)
    const current = options.findIndex(option => option.value === native.current?.value && !option.disabled)
    setActive(index ?? (current >= 0 ? current : options.findIndex(option => !option.disabled)))
    setOpen(true)
  }

  function choose(index: number) {
    const option = options[index]
    if (!option || option.disabled || !native.current) return
    native.current.value = option.value
    native.current.dispatchEvent(new Event('change', { bubbles: true }))
    setSelected(native.current.value)
    setInvalid('')
    setOpen(false)
    trigger.current?.focus()
  }

  function keyDown(event: KeyboardEvent<HTMLButtonElement>) {
    if (event.key === 'Escape' && open) { event.preventDefault(); event.stopPropagation(); setOpen(false); return }
    if (event.key === 'Tab') { setOpen(false); return }
    if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); open ? choose(active) : show(); return }
    if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
      event.preventDefault()
      const enabled = options.map((option, i) => option.disabled ? -1 : i).filter(i => i >= 0)
      if (!enabled.length) return
      let next = active
      if (event.key === 'Home') next = enabled[0]
      else if (event.key === 'End') next = enabled[enabled.length - 1]
      else if (open) next = enabled[Math.max(0, Math.min(enabled.length - 1, enabled.indexOf(active) + (event.key === 'ArrowDown' ? 1 : -1)))]
      else { show(); return }
      open ? setActive(next) : show(next)
      return
    }
    if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
      event.preventDefault()
      const now = Date.now()
      typed.current = { text: now - typed.current.at < 700 ? typed.current.text + event.key : event.key, at: now }
      const text = typed.current.text.toLocaleLowerCase()
      const next = options.findIndex(option => !option.disabled && option.label.toLocaleLowerCase().startsWith(text))
      if (next >= 0) open ? setActive(next) : show(next)
    }
  }

  // Multi-selects retain their native semantics, should a future form introduce one.
  if (props.multiple || (props.size && props.size > 1)) return <select {...props} className={className} style={style}>{children}</select>

  return <span className={`tt-select ${compact ? 'tt-select-compact' : ''} ${className}`} style={style} data-open={open || undefined}>
    <select {...props} ref={native} className="tt-select-native" tabIndex={-1} aria-hidden="true" onFocus={() => trigger.current?.focus()} onChange={event => { setSelected(event.currentTarget.value); setInvalid(''); props.onChange?.(event) }} onInvalid={event => { event.preventDefault(); const message = `Выберите значение в поле «${label}».`; setInvalid(message); notifyError(message); trigger.current?.focus(); props.onInvalid?.(event) }}>{children}</select>
    <button ref={trigger} type="button" className="tt-select-trigger" role="combobox" aria-label={label} aria-labelledby={props['aria-labelledby']} aria-haspopup="listbox" aria-expanded={open} aria-controls={open ? `${id}-menu` : undefined} aria-activedescendant={open && active >= 0 ? `${id}-${active}` : undefined} aria-required={props.required} aria-invalid={Boolean(invalid) || undefined} aria-describedby={invalid ? `${id}-error` : props['aria-describedby']} disabled={props.disabled} onClick={() => open ? setOpen(false) : show()} onKeyDown={keyDown}>
      <span>{options.find(option => option.value === selected)?.label || 'Выберите вариант'}</span><svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 9 6 6 6-6" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"/></svg>
    </button>
    {invalid && <span id={`${id}-error`} className="studio-visually-hidden">{invalid}</span>}
    {open && portal && createPortal(<div ref={popup} className={`tt-select-popup${compact ? ' tt-select-popup-compact' : ''}`} style={position} id={`${id}-menu`} role="listbox" aria-label={label} onMouseDown={event => event.preventDefault()}>
      {options.map((option, index) => <div key={`${option.value}-${index}`}>
        {option.group && option.group !== options[index - 1]?.group && <div className="tt-select-group">{option.group}</div>}
        <div id={`${id}-${index}`} role="option" aria-selected={option.value === selected} aria-disabled={option.disabled || undefined} data-index={index} className={`tt-select-option${active === index ? ' is-active' : ''}`} onPointerMove={() => !option.disabled && setActive(index)} onClick={() => choose(index)}><span>{option.label}</span>{!compact && option.value === selected && <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m5 12 4 4L19 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/></svg>}</div>
      </div>)}
      {!options.length && <div className="tt-select-group">Нет доступных вариантов</div>}
    </div>, portal)}
  </span>
}

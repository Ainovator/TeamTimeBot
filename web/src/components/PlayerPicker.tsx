import { useEffect, useId, useRef, useState, type CSSProperties } from 'react'
import { createPortal } from 'react-dom'
import { Icon } from './Icon'
import './playerPicker.css'

type PlayerOption = { id: number; label: string; username: string }

export function PlayerPicker({ options, value, onChange, label = 'Игрок для связи' }: {
  options: PlayerOption[]; value: string; onChange: (id: number) => void; label?: string
}) {
  const id = useId()
  const trigger = useRef<HTMLButtonElement>(null)
  const popup = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)
  const [position, setPosition] = useState<CSSProperties>({})
  const [portal, setPortal] = useState<HTMLElement | null>(null)
  const selected = options.find(option => String(option.id) === value)
  const search = query.trim().toLocaleLowerCase('ru')
  const filtered = options.filter(option => `${option.label} ${option.username} ${option.id}`.toLocaleLowerCase('ru').includes(search))
  const activeIndex = Math.min(active, filtered.length - 1)

  useEffect(() => {
    if (!open) return
    const outside = (event: PointerEvent) => {
      if (!popup.current?.contains(event.target as Node) && !trigger.current?.contains(event.target as Node)) setOpen(false)
    }
    const scroll = (event: Event) => { if (!popup.current?.contains(event.target as Node)) setOpen(false) }
    const resize = () => setOpen(false)
    document.addEventListener('pointerdown', outside)
    document.addEventListener('scroll', scroll, true)
    window.addEventListener('resize', resize)
    return () => {
      document.removeEventListener('pointerdown', outside)
      document.removeEventListener('scroll', scroll, true)
      window.removeEventListener('resize', resize)
    }
  }, [open])

  useEffect(() => {
    if (open) popup.current?.querySelector(`[data-index="${activeIndex}"]`)?.scrollIntoView({ block: 'nearest' })
  }, [open, activeIndex, query])

  function show() {
    if (!trigger.current) return
    const rect = trigger.current.getBoundingClientRect()
    const theme = getComputedStyle(trigger.current)
    const below = window.innerHeight - rect.bottom - 16
    const above = rect.top - 16
    const upwards = below < 300 && above > below
    const width = Math.min(480, Math.max(300, rect.width), window.innerWidth - 24)
    const variables = Object.fromEntries(['bg', 'text', 'muted', 'border', 'accent', 'soft'].map(key =>
      [`--tt-select-${key}`, theme.getPropertyValue(`--tt-select-${key}`)]))
    setPosition({ ...variables, position: 'fixed', width, maxHeight: Math.min(360, Math.max(120, upwards ? above : below)),
      left: Math.max(12, Math.min(rect.left, window.innerWidth - width - 12)),
      ...(upwards ? { bottom: window.innerHeight - rect.top + 8 } : { top: rect.bottom + 8 }),
      fontFamily: theme.fontFamily,
    })
    setPortal(trigger.current.closest('dialog') || document.body)
    setQuery('')
    setActive(Math.max(0, options.findIndex(option => String(option.id) === value)))
    setOpen(true)
  }

  function choose(option: PlayerOption) {
    onChange(option.id)
    setOpen(false)
    trigger.current?.focus()
  }

  return <div className="tt-select player-picker" data-open={open || undefined}>
    <button ref={trigger} type="button" className="tt-select-trigger" aria-label={label}
      aria-haspopup="dialog" aria-expanded={open} aria-controls={open ? `${id}-popup` : undefined}
      onClick={() => open ? setOpen(false) : show()}
      onKeyDown={event => { if (event.key === 'ArrowDown') { event.preventDefault(); show() } }}>
      <span>{selected?.label || 'Выберите игрока'}</span><Icon name="down" size={16}/>
    </button>
    {open && portal && createPortal(<div ref={popup} id={`${id}-popup`} role="dialog" aria-label="Выбор игрока"
      className="player-picker-popup" style={position}
      onBlur={event => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setOpen(false) }}>
      <div className="player-picker-search">
        <Icon name="search" size={18}/>
        <input autoFocus role="combobox" aria-label="Поиск игрока" aria-autocomplete="list" aria-expanded="true"
          aria-controls={`${id}-list`} aria-activedescendant={activeIndex >= 0 ? `${id}-option-${filtered[activeIndex].id}` : undefined}
          placeholder="Имя или ник игрока" value={query}
          onChange={event => { setQuery(event.target.value); setActive(0) }}
          onKeyDown={event => {
            if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); setOpen(false); trigger.current?.focus() }
            if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
              event.preventDefault()
              setActive(Math.max(0, Math.min(filtered.length - 1, activeIndex + (event.key === 'ArrowDown' ? 1 : -1))))
            }
            if (event.key === 'Enter') { event.preventDefault(); if (filtered[activeIndex]) choose(filtered[activeIndex]) }
          }}/>
      </div>
      <div className="player-picker-options" role="listbox" id={`${id}-list`} aria-label="Игроки">
        {filtered.map((option, index) => <div key={option.id} id={`${id}-option-${option.id}`} role="option"
          aria-selected={String(option.id) === value} data-index={index}
          className={`player-picker-option${activeIndex === index ? ' is-active' : ''}`}
          onMouseDown={event => event.preventDefault()} onClick={() => choose(option)}>
          <span className="player-picker-avatar" aria-hidden="true">{option.label.split(/\s+/).slice(0, 2).map(part => part[0]).join('')}</span>
          <span className="player-picker-person"><strong>{option.label}</strong>{option.username && <small>@{option.username}</small>}</span>
          {String(option.id) === value && <Icon name="check" size={18}/>}
        </div>)}
        {!filtered.length && <p className="player-picker-empty" role="status">Никого не нашли. Попробуйте другое имя.</p>}
      </div>
    </div>, portal)}
  </div>
}

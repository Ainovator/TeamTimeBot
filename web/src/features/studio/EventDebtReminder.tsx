import { useEffect, useRef, useState } from 'react'
import { fetchEventBillingForInstance, publishEventBillingDebtorsForInstance } from '../../api'
import { formatMoney } from '../../app/constants'
import { Checkbox } from '../../components/Checkbox'
import { useErrorState } from '../../components/ErrorNotifications'
import { Icon } from '../../components/Icon'
import { Modal } from '../../components/Modal'
import type { EventBilling, EventBillingPlayer } from '../../types'
import './eventDebtReminder.css'

const playerName = (player: EventBillingPlayer) => player.realName?.trim() || `${player.firstName ?? ''} ${player.lastName ?? ''}`.trim() || (player.username ? `@${player.username}` : `Игрок ${player.userID}`)

export function EventDebtReminder({ chatID, instanceID, eventName, groupTitle, billing, pendingChanges, unavailable, onPublished }: {
  chatID: number; instanceID: number; eventName: string; groupTitle: string; billing: EventBilling | null
  pendingChanges: boolean; unavailable: boolean; onPublished: () => void
}) {
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [preview, setPreview] = useState<EventBilling | null>(null)
  const [selected, setSelected] = useState<number[]>([])
  const [, setError] = useErrorState()
  const lock = useRef(false)
  const alive = useRef(true)
  useEffect(() => { alive.current = true; return () => { alive.current = false } }, [])
  const debtors = (preview?.players ?? []).filter(player => !player.isPaid && player.amountDue > 0)
  const included = debtors.filter(player => selected.includes(player.userID))
  const total = included.reduce((sum, player) => sum + player.amountDue, 0)
  const hasDebt = billing?.players.some(player => !player.isPaid && player.amountDue > 0) ?? false
  const localDate = preview?.localDate ?? billing?.localDate
  const date = localDate ? new Date(localDate).toLocaleDateString('ru-RU', { timeZone: 'UTC' }) : ''

  async function showPreview() {
    if (lock.current || pendingChanges || unavailable || !hasDebt) return
    lock.current = true
    setBusy(true)
    setOpen(true)
    setPreview(null)
    try {
      const current = await fetchEventBillingForInstance(chatID, instanceID)
      if (!alive.current) return
      setPreview(current)
      setSelected((current?.players ?? []).filter(player => !player.isPaid && player.amountDue > 0).map(player => player.userID))
    } catch (error) {
      if (alive.current) { setOpen(false); setError((error as Error).message) }
    } finally {
      lock.current = false
      if (alive.current) setBusy(false)
    }
  }

  async function publish() {
    if (lock.current || !included.length || pendingChanges) return
    lock.current = true
    setBusy(true)
    try {
      await publishEventBillingDebtorsForInstance(chatID, instanceID, included.map(player => player.userID))
      if (alive.current) { setOpen(false); onPublished() }
    } catch (error) {
      if (alive.current) setError((error as Error).message)
    } finally {
      lock.current = false
      if (alive.current) setBusy(false)
    }
  }

  return <>
    <div className="studio-event-debt-action">
      <button type="button" aria-haspopup="dialog" onClick={() => void showPreview()}
        disabled={busy || pendingChanges || unavailable || !hasDebt}>
        <Icon name="arrow" size={16}/>Опубликовать должников
      </button>
      <small>{unavailable ? 'Ожидаем актуальные данные об оплатах.' : pendingChanges ? 'Сначала сохраните изменения оплаты.' : !billing ? 'Сначала сформируйте расчёт события.' : !hasDebt ? 'Должников по этому событию нет.' : 'В Telegram · только по этому событию'}</small>
    </div>
    {open && <Modal title="Задолженность за тренировку" busy={busy} close={() => setOpen(false)}>
      <p className="muted">В группу «{groupTitle}» будет опубликован долг за «{eventName}» ({date}). Имена выбранных игроков будут упоминаниями Telegram.</p>
      {busy && !preview ? <p role="status">Проверяем оплаты…</p> : debtors.length ? <>
        <div className="studio-event-debt-heading"><h3>Кому напомнить об оплате</h3><button type="button" className="btn-secondary" disabled={busy} onClick={() => setSelected(included.length === debtors.length ? [] : debtors.map(player => player.userID))}>{included.length === debtors.length ? 'Снять выбор' : 'Выбрать всех'}</button></div>
        <div className="studio-event-debt-recipients">{debtors.map(player => <label key={player.userID}>
          <Checkbox aria-label={`Напомнить: ${playerName(player)}`} checked={selected.includes(player.userID)} disabled={busy} onChange={event => setSelected(prev => event.target.checked ? [...prev, player.userID] : prev.filter(id => id !== player.userID))}/>
          <span>{playerName(player)}</span><strong>{formatMoney(player.amountDue)}</strong>
        </label>)}</div>
        <div className="studio-event-debt-preview" aria-label="Предпросмотр задолженности">
          <strong>Задолженности за «{eventName}»</strong><span>Дата: {date}</span>
          {included.map(player => <div key={player.userID}><span className="studio-event-debt-mention">{playerName(player)}</span><strong>{formatMoney(player.amountDue)}</strong></div>)}
          <div><strong>Итого</strong><strong>{formatMoney(total)}</strong></div>
        </div>
      </> : <p role="status">Неоплаченных взносов за эту тренировку нет.</p>}
      {!busy && debtors.length > 0 && !included.length && <p className="muted" role="status">Выберите хотя бы одного игрока.</p>}
      <button type="button" className="studio-wide" disabled={busy || !included.length || pendingChanges} onClick={() => void publish()}>{busy ? 'Подождите…' : 'Опубликовать в Telegram'}</button>
    </Modal>}
  </>
}

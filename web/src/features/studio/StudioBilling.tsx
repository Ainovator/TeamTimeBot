import { Checkbox } from '../../components/Checkbox'
import { useErrorState } from '../../components/ErrorNotifications'
import { useEffect, useRef, useState } from 'react'
import { fetchGroupDebtors, fetchGroupDebtSummary, publishGroupDebtors, saveEventBillingForInstance } from '../../api'
import { Icon } from '../../components/Icon'
import { Modal } from '../../components/Modal'
import { formatMoney } from '../../app/constants'
import { memberInitials } from './StudioMembers'
import type { DebtorTrainingDebt, GroupDebtor, GroupDebtSummary } from '../../types'
import './billingReminder.css'
import './billingList.css'

type Payment = { userID: number; instanceID: number }
const countLabel = (count: number, one: string, few: string, many: string) => count % 100 >= 11 && count % 100 <= 14 ? many : count % 10 === 1 ? one : count % 10 >= 2 && count % 10 <= 4 ? few : many
const nameOf = (debtor: GroupDebtor) => debtor.realName?.trim() || `${debtor.firstName || ''} ${debtor.lastName || ''}`.trim() || (debtor.username ? `@${debtor.username}` : `Игрок ${debtor.userID}`)

function meetingDate(training: DebtorTrainingDebt) {
  const date = new Date(training.startAt)
  return Number.isNaN(date.getTime()) ? 'Дата не указана' : date.toLocaleDateString('ru-RU', { day:'numeric', month:'long', year:'numeric' })
}

function FeeMeeting({ training, openEvent }: { training: DebtorTrainingDebt; openEvent: (id: number) => void }) {
  return <button type="button" className="studio-fee-meeting" onClick={() => openEvent(training.instanceID)}><strong>{training.eventName || 'Встреча команды'}</strong><span>{meetingDate(training)}</span></button>
}

function FeePayment({ debtor, training, busy, mark }: { debtor: GroupDebtor; training: DebtorTrainingDebt; busy: boolean; mark: (payments: Payment[]) => Promise<void> }) {
  return <div className="studio-fee-payment"><strong>{formatMoney(training.amountDue)}</strong><button type="button" className="btn-secondary studio-fee-pay" disabled={busy} aria-label={`Отметить оплату: ${nameOf(debtor)}, ${training.eventName || 'Встреча команды'}, ${meetingDate(training)}`} onClick={() => void mark([{userID:debtor.userID,instanceID:training.instanceID}])}><Icon name="check" size={16}/>Отметить оплату</button></div>
}

export function StudioBilling({ chatID, groupTitle, debtors, loading, error, summaryChanged, openEvent }: { chatID: number; groupTitle: string; debtors: GroupDebtor[]; loading: boolean; error: string; summaryChanged: (summary: GroupDebtSummary) => void; openEvent: (id: number) => void }) {
  const [rows, setRows] = useState(debtors)
  const [excluded, setExcluded] = useState<number[]>([])
  const [expanded, setExpanded] = useState<Record<number, boolean>>({})
  const [query, setQuery] = useState('')
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  const [, setFailure] = useErrorState()
  const [undo, setUndo] = useState<Payment[]>([])
  const [preview, setPreview] = useState(false)
  const alive = useRef(true)
  const lock = useRef(false)
  useEffect(() => { alive.current = true; return () => { alive.current = false } }, [])
  useEffect(() => { setRows(debtors) }, [debtors])
  useEffect(() => { setExpanded({}); setExcluded([]) }, [chatID])
  const selected = rows.filter(row => !excluded.includes(row.userID))
  const total = rows.reduce((sum, row) => sum + row.totalDebt, 0)
  const unpaidCount = rows.reduce((sum, row) => sum + (row.trainings?.length ?? 0), 0)
  const selectedTotal = selected.reduce((sum, row) => sum + row.totalDebt, 0)
  const filtered = rows.filter(row => `${nameOf(row)} ${row.username}`.toLowerCase().includes(query.trim().toLowerCase())).sort((a,b) => b.totalDebt-a.totalDebt)

  function includeInReminder(userID: number, included: boolean) {
    setExcluded(prev => included ? prev.filter(id => id !== userID) : [...new Set([...prev, userID])])
  }
  function toggleAllRecipients() {
    setExcluded(selected.length === rows.length ? rows.map(row => row.userID) : [])
  }

  async function refresh() {
    const [next, summary] = await Promise.all([fetchGroupDebtors(chatID), fetchGroupDebtSummary(chatID)])
    if (alive.current) { setRows(next ?? []); summaryChanged(summary) }
  }
  async function mark(payments: Payment[], paid = true) {
    if (lock.current || !payments.length) return
    lock.current = true; setBusy(true); setFailure(''); setNotice('')
    const saved: Payment[] = []
    try {
      for (const payment of payments) {
        await saveEventBillingForInstance(chatID, payment.instanceID, [{ userID: payment.userID, paid }])
        saved.push(payment)
      }
      if (alive.current) setNotice(paid ? 'Оплата сохранена' : 'Отметка оплаты отменена')
    } catch (err) {
      if (alive.current) setFailure(`${saved.length ? `Сохранено взносов: ${saved.length}. ` : ''}${(err as Error).message}`)
    } finally {
      if (alive.current) setUndo(paid ? saved : payments.filter(payment => !saved.includes(payment)))
      try { await refresh() } catch (err) { if (alive.current) setFailure(`Не удалось обновить расчёт: ${(err as Error).message}`) }
      lock.current = false
      if (alive.current) setBusy(false)
    }
  }
  async function publish() {
    if (lock.current || !selected.length) return
    lock.current = true; setBusy(true); setFailure('')
    try { await publishGroupDebtors(chatID, selected.map(row=>row.userID)); if (alive.current) { setPreview(false); setNotice('Напоминание опубликовано в Telegram') } }
    catch (err) { if (alive.current) setFailure((err as Error).message) }
    finally { lock.current=false; if (alive.current) setBusy(false) }
  }
  return <div className="studio-billing">
    <section className="studio-reminder-hero" aria-labelledby="billing-reminder-title">
      <div className="studio-reminder-copy">
        <p className="studio-reminder-eyebrow"><Icon name="wallet" size={16}/>ВЗНОСЫ КОМАНДЫ</p>
        <h2 id="billing-reminder-title">Напоминание команде</h2>
        <p className="studio-reminder-description">Отметьте игроков в списке и отправьте напоминание о взносах.</p>
      </div>
      <div className="studio-reminder-actions">
        <button type="button" className="studio-reminder-cta" aria-haspopup="dialog" disabled={busy || loading || Boolean(error) || !rows.length} onClick={()=>setPreview(true)}>Напомнить<Icon name="arrow" size={18}/></button>
      </div>
      <dl className="studio-reminder-stats" aria-label="Сводка взносов">
        <div className="studio-reminder-total"><dt>Осталось собрать</dt><dd>{loading || error ? '—' : formatMoney(total)}</dd></div>
        <div><dt>{countLabel(rows.length, 'игрок', 'игрока', 'игроков')} с неоплаченными взносами</dt><dd>{loading || error ? '—' : rows.length}</dd></div>
        <div><dt>{countLabel(unpaidCount, 'взнос', 'взноса', 'взносов')} к оплате</dt><dd>{loading || error ? '—' : unpaidCount}</dd></div>
      </dl>
    </section>
    {(notice || undo.length > 0) && <div className="studio-feedback" role="status"><Icon name="check" size={18}/><span>{notice || 'Изменения оплаты сохранены'}</span>{undo.length>0&&<button className="btn-secondary" disabled={busy} onClick={()=>void mark(undo,false)}>Отменить оплату</button>}</div>}
    <div className="studio-list-tools"><label className="studio-search"><Icon name="search" size={19}/><input aria-label="Поиск по игрокам" placeholder="Найти игрока" value={query} onChange={event=>setQuery(event.target.value)}/></label></div>
    <section className="studio-debt-list" aria-label="Неоплаченные взносы игроков">
      <div className="studio-debt-list-heading"><h2>Игроки</h2><span className="studio-debt-count" role="status">{loading || error ? '—' : query.trim() ? `${filtered.length} из ${rows.length}` : filtered.length}</span></div>
      <div className="studio-fee-columns" aria-hidden="true"><span>Игрок</span><span>Встреча</span><span className="studio-fee-reminder-heading">Напомнить</span><span>К оплате</span></div>
      {loading ? <p className="studio-empty-inline" role="status">Загружаем взносы…</p> : error ? <p className="studio-empty-inline">Данные о взносах недоступны.</p> : !filtered.length ? <div className="studio-empty-inline"><Icon name={rows.length?'search':'check'} size={28}/><h3>{rows.length ? 'Игроки не найдены' : 'Все взносы закрыты'}</h3><p>{rows.length ? 'Попробуйте другое имя.' : 'Неоплаченных взносов нет.'}</p></div> : filtered.map(debtor => {
        const trainings = debtor.trainings ?? []
        const single = trainings.length === 1 ? trainings[0] : null
        const isExpanded = expanded[debtor.userID] ?? false
        return <article className={`studio-fee-player${trainings.length > 1 ? ' is-grouped' : single ? ' is-single' : ''}`} key={debtor.userID}>
          <div className="studio-fee-row">
            <div className="studio-fee-person"><span className="studio-person-avatar" aria-hidden="true">{memberInitials(nameOf(debtor))}</span><div><strong>{nameOf(debtor)}</strong>{debtor.username && <span>@{debtor.username}</span>}</div></div>
            {single ? <FeeMeeting training={single} openEvent={openEvent}/> : trainings.length > 1 ? <button type="button" className="studio-fee-group-toggle" aria-label={`Встречи: ${nameOf(debtor)}`} aria-expanded={isExpanded} aria-controls={`fee-meetings-${debtor.userID}`} onClick={() => setExpanded(prev => ({...prev,[debtor.userID]:!prev[debtor.userID]}))}><span className="studio-fee-group-info"><strong>{trainings.length} {countLabel(trainings.length,'встреча','встречи','встреч')}</strong><span>{isExpanded ? 'Скрыть встречи' : 'Показать встречи'}</span></span><span className={isExpanded ? 'is-expanded' : ''}><Icon name="down" size={16}/></span></button> : <div className="studio-fee-group-info"><strong>Нет данных о встречах</strong><span>Обновите список взносов</span></div>}
            <label className="studio-fee-reminder"><Checkbox aria-label={`В напоминание: ${nameOf(debtor)}`} checked={!excluded.includes(debtor.userID)} disabled={busy} onChange={event => includeInReminder(debtor.userID, event.target.checked)}/><span>Напомнить</span></label>
            {single ? <FeePayment debtor={debtor} training={single} busy={busy} mark={mark}/> : <div className="studio-fee-payment"><strong>{formatMoney(debtor.totalDebt)}</strong>{trainings.length > 1 && <button type="button" className="btn-secondary studio-fee-pay" disabled={busy} aria-label={`Отметить все взносы: ${nameOf(debtor)}`} onClick={() => void mark(trainings.map(training => ({userID:debtor.userID,instanceID:training.instanceID})))}><Icon name="check" size={16}/>Отметить всё</button>}</div>}
          </div>
          {trainings.length > 1 && <div className="studio-fee-details" id={`fee-meetings-${debtor.userID}`} role="region" aria-label={`Встречи: ${nameOf(debtor)}`} hidden={!isExpanded}>{trainings.map(training => <div className="studio-fee-detail" key={training.instanceID}><FeeMeeting training={training} openEvent={openEvent}/><FeePayment debtor={debtor} training={training} busy={busy} mark={mark}/></div>)}</div>}
        </article>
      })}
    </section>
    {preview && <Modal title="Напоминание команде" busy={busy} close={()=>setPreview(false)}>
      <div className="studio-fee-recipients-head"><h3 id="fee-recipients-title">Получатели</h3><button type="button" className="studio-text-button" disabled={busy || !rows.length} onClick={toggleAllRecipients}>{selected.length===rows.length ? 'Снять выбор' : 'Выбрать всех'}</button></div>
      <div className="studio-fee-recipients" role="group" aria-labelledby="fee-recipients-title">{rows.map(debtor => <label className="studio-fee-recipient" key={debtor.userID}><Checkbox aria-label={`Напомнить: ${nameOf(debtor)}`} checked={!excluded.includes(debtor.userID)} disabled={busy} onChange={event => includeInReminder(debtor.userID, event.target.checked)}/><span>{nameOf(debtor)}</span><strong>{formatMoney(debtor.totalDebt)}</strong></label>)}</div>
      <h3 className="studio-fee-preview-title">Сообщение</h3>
      {selected.length ? <div className="studio-message"><h3>{groupTitle}</h3><p>Взносы за встречи команды</p>{selected.map(debtor=><div key={debtor.userID}><span>{nameOf(debtor)}</span><strong>{formatMoney(debtor.totalDebt)}</strong></div>)}<div><strong>Всего</strong><strong>{formatMoney(selectedTotal)}</strong></div></div> : <p className="studio-fee-preview-empty" role="status">Выберите хотя бы одного игрока.</p>}
      <p className="muted">Будет опубликован список неоплаченных взносов выбранных игроков в Telegram-группе. Имена игроков будут упоминаниями Telegram.</p>
      <button type="button" className="studio-wide" disabled={busy || !selected.length} onClick={()=>void publish()}>{busy?'Публикуем…':'Опубликовать в Telegram'}<Icon name="arrow" size={16}/></button>
    </Modal>}
  </div>
}

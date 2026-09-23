import { Checkbox } from '../../components/Checkbox'
import { Select } from '../../components/Select'
import { useId, useState } from 'react'
import { DemoDialog, Icon } from './components'
import { PlayerPortrait } from './PlayerPortrait'
import { roster, type RosterPlayer } from './rosterData'
import { initialDebts } from './productDemoData'
import { money, Search } from './screenComponents'
import './debts.css'

type Debt = typeof initialDebts[number]
type Account = { player: RosterPlayer; entries: Debt[]; balance: number; unpaid: number }
type Filter = 'debt' | 'all' | 'clear'

function plural(value: number, forms: [string, string, string]) {
  const tens = value % 100
  return forms[tens >= 11 && tens <= 14 ? 2 : value % 10 === 1 ? 0 : value % 10 >= 2 && value % 10 <= 4 ? 1 : 2]
}

function DebtRow({ account, included, toggleIncluded, mark }: { account: Account; included: boolean; toggleIncluded: () => void; mark: (entries: Debt[], paid: boolean) => void }) {
  const { player, entries, balance, unpaid } = account
  const [expanded, setExpanded] = useState(player.id === 2)
  const detailsId = useId()
  return <article className={`dl-ledger-row${!balance ? ' dl-ledger-row-clear' : ''}`} aria-label={`Расчёт: ${player.name}`}>
    <div className="dl-ledger-row-main">
      <button className="dl-ledger-person" aria-expanded={expanded} aria-controls={detailsId} onClick={() => setExpanded(value => !value)} aria-label={`Взносы: ${player.name}`}>
        <span className="dl-ledger-expand"><Icon name="chevron" size={15}/></span><PlayerPortrait player={player}/><span className="dl-ledger-person-info"><strong>{player.name}</strong><small>{unpaid ? `${unpaid} ${plural(unpaid, ['неоплаченный взнос', 'неоплаченных взноса', 'неоплаченных взносов'])}` : 'Все взносы оплачены'}</small></span>
      </button>
      <strong className="dl-ledger-row-amount">{money(balance)}</strong>
      <label className="dl-ledger-row-select"><Checkbox checked={Boolean(balance) && included} disabled={!balance} onChange={toggleIncluded} aria-label={`Включить в напоминание: ${player.name}`}/><span>{balance ? included ? 'Включён' : 'Добавить' : 'Не требуется'}</span></label>
      {balance ? <button className="dl-ledger-settle" onClick={() => mark(entries.filter(e => !e.paid), true)} aria-label={`Отметить все оплаты: ${player.name}`}><Icon name="check" size={15}/>Отметить оплату</button> : <span className="dl-ledger-row-paid"><Icon name="check" size={15}/>Оплачено</span>}
    </div>
    <div className="dl-ledger-row-details" id={detailsId} hidden={!expanded} role="region" aria-label={`Взносы по встречам: ${player.name}`}>
      {entries.length > 0 && <div className="dl-ledger-detail-heading" aria-hidden="true"><span>Встреча</span><span>Взнос</span><span>Оплата</span></div>}
      {entries.map(entry => <div className={`dl-ledger-entry${entry.paid ? ' dl-ledger-entry-paid' : ''}`} key={entry.id}>
        <div className="dl-ledger-entry-meeting"><span className="dl-ledger-entry-icon"><Icon name="calendar" size={20}/></span><span className="dl-ledger-entry-info"><strong className="dl-ledger-entry-name">{entry.event}</strong><span className="dl-ledger-entry-date">{entry.date}</span></span></div>
        <div className="dl-ledger-entry-amount"><strong>{money(entry.amount)}</strong><span>{entry.paid ? 'Оплачено' : 'Не оплачено'}</span></div>
        <button onClick={() => mark([entry], !entry.paid)} aria-label={`${entry.paid ? 'Отменить оплату' : 'Отметить оплату'}: ${player.name}, ${entry.date}`}><Icon name={entry.paid ? 'back' : 'check'} size={16}/>{entry.paid ? 'Отменить' : 'Отметить оплату'}</button>
      </div>)}
      {!entries.length && <p className="dl-ledger-row-empty"><Icon name="check" size={16}/>Неоплаченных встреч нет.</p>}
    </div>
  </article>
}

export function DebtsScreen() {
  const [debts, setDebts] = useState(initialDebts)
  const [included, setIncluded] = useState([2, 4])
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState<Filter>('debt')
  const [sort, setSort] = useState('По сумме долга')
  const [preview, setPreview] = useState(false)
  const [undo, setUndo] = useState<{ before: Debt[]; message: string } | null>(null)
  const accounts: Account[] = roster.map(player => {
    const entries = debts.filter(d => d.player === player.id)
    const unpaid = entries.filter(d => !d.paid)
    return { player, entries, balance: unpaid.reduce((sum, d) => sum + d.amount, 0), unpaid: unpaid.length }
  })
  const owing = accounts.filter(a => a.balance > 0)
  const total = owing.reduce((sum, a) => sum + a.balance, 0)
  const unpaidCount = debts.filter(d => !d.paid).length
  const recipients = owing.filter(a => included.includes(a.player.id))
  const reminderTotal = recipients.reduce((sum, a) => sum + a.balance, 0)
  const filtered = accounts.filter(a => (filter === 'all' || (filter === 'debt' ? a.balance > 0 : a.balance === 0)) && a.player.name.toLowerCase().includes(query.trim().toLowerCase())).sort((a, b) => sort === 'По фамилии' ? a.player.name.split(' ')[1].localeCompare(b.player.name.split(' ')[1], 'ru') : b.balance - a.balance || a.player.id - b.player.id)
  const tabs: { id: Filter; label: string; count: number }[] = [{ id: 'debt', label: 'С задолженностью', count: owing.length }, { id: 'all', label: 'Все игроки', count: roster.length }, { id: 'clear', label: 'Без долга', count: roster.length - owing.length }]

  function mark(entries: Debt[], paid: boolean) {
    if (!entries.length) return
    const ids = entries.map(entry => entry.id)
    const amount = entries.reduce((sum, entry) => sum + entry.amount, 0)
    const player = roster[entries[0].player]
    setUndo({ before: debts, message: paid ? `${player.name} · отмечена оплата ${money(amount)}` : `${player.name} · отметка ${money(amount)} отменена` })
    setDebts(values => values.map(d => ids.includes(d.id) ? { ...d, paid } : d))
  }

  return <section className="dl-ledger-screen">
    <div className="dl-ledger-heading"><div><p className="dl-eyebrow">ОРБИТА / ОБЩИЕ РАСХОДЫ</p><h1>Задолженности<span>.</span></h1><p>Кто, сколько и за какую встречу. Всё на виду.</p></div><span className="dl-ledger-period"><Icon name="calendar" size={16}/>Сентябрь 2026</span></div>
    <div className={`dl-ledger-summary${!total ? ' dl-ledger-settled' : ''}`}><div className="dl-ledger-total"><span className="dl-ledger-wallet"><Icon name={total ? 'wallet' : 'check'} size={25}/></span><div><span>{total ? 'Осталось собрать' : 'Все взносы оплачены'}</span><strong>{money(total)}</strong></div></div><div className="dl-ledger-summary-detail"><strong>{owing.length}</strong><span>{plural(owing.length, ['игрок', 'игрока', 'игроков'])}<small>с задолженностью</small></span></div><div className="dl-ledger-summary-detail"><strong>{unpaidCount}</strong><span>{plural(unpaidCount, ['взнос', 'взноса', 'взносов'])}<small>за прошедшие встречи</small></span></div><div className="dl-ledger-settled-note"><Icon name="check" size={16}/><span><strong>{roster.length - owing.length} из {roster.length}</strong> игроков без долга</span></div></div>
    {undo && <div className="dl-ledger-notice" role="status"><Icon name="check" size={17}/><span>{undo.message}</span><button onClick={() => { setDebts(undo.before); setUndo(null) }}>Отменить</button></div>}
    <div className="dl-ledger-toolbar"><Search placeholder="Найти игрока в расчётах" value={query} onChange={setQuery}/><Select aria-label="Сортировка задолженностей" value={sort} onChange={e => setSort(e.target.value)}><option>По сумме долга</option><option>По фамилии</option></Select></div>
    <div className="dl-ledger-tabs" role="group" aria-label="Фильтр задолженностей">{tabs.map(tab => <button key={tab.id} aria-pressed={filter === tab.id} onClick={() => setFilter(tab.id)}>{tab.label}<span>{tab.count}</span></button>)}<span className="dl-ledger-found" aria-live="polite">Найдено: {filtered.length}</span></div>
    <div className="dl-ledger-list" aria-label="Задолженности игроков">
      <div className="dl-ledger-columns" aria-hidden="true"><span>Игрок / встречи</span><span>К оплате</span><span>Напоминание</span><span/></div>
      {filtered.map(account => <DebtRow key={account.player.id} account={account} included={included.includes(account.player.id)} toggleIncluded={() => setIncluded(values => values.includes(account.player.id) ? values.filter(id => id !== account.player.id) : [...values, account.player.id])} mark={mark}/>)}
      {!filtered.length && <div className="dl-ledger-empty"><span><Icon name={!total && !query ? 'check' : 'search'} size={30}/></span><h2>{!total && !query && filter === 'debt' ? 'Команда закрыла все взносы' : 'Никого не нашли'}</h2><p>{!total && !query && filter === 'debt' ? 'Спасибо! Можно спокойно готовиться к следующей игре.' : 'Попробуйте другое имя или измените фильтр.'}</p><button className="dl-button dl-secondary" onClick={() => { setQuery(''); setFilter('all') }}>Показать всех игроков<Icon name="arrow" size={16}/></button></div>}
    </div>
    <div className="dl-ledger-reminder" aria-label="Игроки для напоминания"><span className="dl-ledger-reminder-icon"><Icon name="poll" size={20}/></span><div className="dl-ledger-reminder-info"><strong>Напоминание команде</strong><span aria-live="polite">{recipients.length ? `${recipients.length} ${plural(recipients.length, ['игрок', 'игрока', 'игроков'])} · ${money(reminderTotal)}` : 'Выберите игроков в списке'}</span></div><button className="dl-ledger-reminder-select" disabled={!owing.length} onClick={() => setIncluded(recipients.length === owing.length ? [] : owing.map(a => a.player.id))}>{recipients.length === owing.length && owing.length ? 'Снять выбор' : 'Выбрать всех'}</button><button className="dl-button dl-primary" disabled={!recipients.length} onClick={() => setPreview(true)}>Посмотреть сообщение<Icon name="arrow" size={17}/></button></div>
    <p className="dl-ledger-demo"><Icon name="wallet" size={15}/>Демонстрационный расчёт. Отметки оплаты сохраняются только в примерочной.</p>
    {preview && <DemoDialog title="Напоминание команде" close={() => setPreview(false)}><div className="dl-ledger-message"><div className="dl-ledger-message-sender"><span><Icon name="ball" size={23}/></span><div><strong>Орбита · TeamTime</strong><small>Взносы за тренировки</small></div></div><p>Команда, закроем расходы за прошлые встречи. Спасибо!</p>{recipients.map(account => <div className="dl-ledger-message-row" key={account.player.id}><span>{account.player.name}<small>{account.entries.filter(e => !e.paid).map(e => e.date).join(' · ')}</small></span><strong>{money(account.balance)}</strong></div>)}<div className="dl-ledger-message-total"><span>Всего</span><strong>{money(reminderTotal)}</strong></div></div><p className="dl-demo-note">Это предпросмотр. Сообщение не отправлено в Telegram.</p><button className="dl-button dl-primary dl-wide" onClick={() => setPreview(false)}>Готово<Icon name="check" size={17}/></button></DemoDialog>}
  </section>
}

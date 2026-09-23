import { useState } from 'react'
import { Icon } from '../../components/Icon'
import { formatMoney } from '../../app/constants'
import { historyStatusLabel } from '../history/historyUtils'
import type { EventHistoryItem, GroupDebtSummary, GroupDetails, GroupMember } from '../../types'

export function StudioOverview({ details, members, debt, history, loading, error, openEvent, openMembers, openEvents, openBilling, createEvent }: { details: GroupDetails; members: GroupMember[]; debt: GroupDebtSummary | null; history: EventHistoryItem[]; loading: boolean; error: string; openEvent: (id: number) => void; openMembers: () => void; openEvents: () => void; openBilling: () => void; createEvent: () => void }) {
  const [monthOffset, setMonthOffset] = useState(0)
  const today = new Date()
  const month = new Date(today.getFullYear(), today.getMonth() + monthOffset, 1)
  const next = [...history].filter(event => new Date(event.nextStartAt).getTime() >= today.getTime() && event.status !== 'not_held').sort((a, b) => a.nextStartAt.localeCompare(b.nextStartAt))
  const recent = [...history].sort((a, b) => b.nextStartAt.localeCompare(a.nextStartAt)).slice(0, 4)
  const nearest = next[0]
  const days = new Date(month.getFullYear(), month.getMonth() + 1, 0).getDate()
  const leading = (month.getDay() + 6) % 7
  const sameDay = (date: Date, day: number) => date.getFullYear() === month.getFullYear() && date.getMonth() === month.getMonth() && date.getDate() === day
  const dateLabel = (value: string) => new Date(value).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', weekday: 'short' })
  return <div className="studio-overview">
    <div className="metrics-grid">
      {[{label:'Игроков в команде',value:members.length,icon:'users',action:openMembers}, {label:'Ближайших встреч',value:loading || error ? '—' : next.length,icon:'calendar',action:openEvents}, {label:'Шаблонов событий',value:details.events.length,icon:'repeat',action:createEvent}, {label:'Осталось собрать',value:debt ? formatMoney(debt.totalDebt) : '—',icon:'wallet',action:openBilling}].map(item => <button className="metric-card" key={item.label} onClick={item.action}><span className="metric-label">{item.label}<Icon name={item.icon} size={19}/></span><strong className="metric-value">{item.value}</strong><span className="studio-metric-link">Подробнее<Icon name="arrow" size={15}/></span></button>)}
    </div>
    <div className="studio-overview-grid">
      <section className="studio-next-session"><div className="studio-next-copy"><p className="studio-eyebrow">БЛИЖАЙШАЯ ВСТРЕЧА</p><h2>{loading ? 'Загружаем расписание…' : nearest?.name || 'Время собраться'}</h2><p>{nearest ? `${dateLabel(nearest.nextStartAt)} · ${new Date(nearest.nextStartAt).toLocaleTimeString('ru-RU',{hour:'2-digit',minute:'2-digit'})}` : 'Назначьте следующую тренировку — и команда увидит её в расписании.'}</p><button onClick={() => nearest ? openEvent(nearest.instanceID) : createEvent()}>{nearest ? 'Открыть встречу' : 'К шаблонам событий'}<Icon name="arrow" size={17}/></button></div><div className="studio-court" aria-hidden="true"><div className="studio-court-lines"><i/><span/><b/></div><span className="studio-court-ball"><Icon name="ball" size={50}/></span></div></section>
      <section className="content-card studio-calendar"><div className="studio-section-head"><h3>{month.toLocaleDateString('ru-RU',{month:'long',year:'numeric'})}</h3><div><button className="btn-icon btn-secondary" aria-label="Предыдущий месяц" onClick={() => setMonthOffset(value=>value-1)}><Icon name="back" size={15}/></button><button className="btn-icon btn-secondary" aria-label="Следующий месяц" onClick={() => setMonthOffset(value=>value+1)}><Icon name="arrow" size={15}/></button></div></div><div className="studio-calendar-days">{['Пн','Вт','Ср','Чт','Пт','Сб','Вс'].map(day=><small key={day}>{day}</small>)}{Array.from({length:leading},(_,i)=><span key={`blank-${i}`}/>)}{Array.from({length:days},(_,i)=>{const day=i+1;const events=history.filter(event=>sameDay(new Date(event.nextStartAt),day));return <button key={day} className={`${sameDay(today,day)?'today ':''}${events.length?'has-event':''}`} disabled={!events.length} title={events.map(event=>event.name).join(', ')} aria-label={`${day} ${month.toLocaleDateString('ru-RU',{month:'long'})}${events.length?`: ${events.map(e=>e.name).join(', ')}`:''}`} onClick={()=>events[0]&&openEvent(events[0].instanceID)}>{day}{events.length>0&&<i/>}</button>})}</div><p className="muted">Точки отмечают встречи команды</p></section>
    </div>
    <section className="content-card"><div className="studio-section-head"><h3>Встречи команды</h3><span className="muted">{history.length} в истории</span></div>{recent.length ? recent.map(event=><button className="studio-activity-row" key={event.instanceID} onClick={()=>openEvent(event.instanceID)}><span className="studio-date-tile"><strong>{new Date(event.nextStartAt).getDate()}</strong><small>{new Date(event.nextStartAt).toLocaleDateString('ru-RU',{month:'short'})}</small></span><span><strong>{event.name}</strong><small>{dateLabel(event.nextStartAt)}</small></span><span className="badge">{historyStatusLabel(event.status)}</span><Icon name="arrow" size={17}/></button>) : <p className="studio-empty-inline">{loading ? 'Загружаем встречи…' : 'История появится после первой встречи.'}</p>}</section>
  </div>
}

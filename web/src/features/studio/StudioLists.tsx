import { Icon } from '../../components/Icon'
import { formatMoney, toHourMinute, weekdayLabel } from '../../app/constants'
import { historyStatusLabel } from '../history/historyUtils'
import type { EventHistoryItem, EventView, GroupPollItem } from '../../types'

export function StudioEvents({ items, view, open }: { items: EventHistoryItem[]; view: 'cards' | 'list'; open: (id: number) => void }) {
  return <div className={view === 'list' ? 'studio-events-list' : 'studio-events-grid'}>{items.map(event => {
    const date = new Date(event.nextStartAt)
    if (view === 'list') return <button key={event.instanceID} className="studio-event-item studio-event-row" onClick={() => open(event.instanceID)}>
      <span className="studio-event-date"><strong>{date.getDate()}</strong><small>{date.toLocaleDateString('ru-RU', {month:'short'})}</small></span>
      <span className="studio-event-row-copy"><strong>{event.name}</strong><small>{date.toLocaleDateString('ru-RU', {weekday:'long'})} · {date.toLocaleTimeString('ru-RU', {hour:'2-digit',minute:'2-digit'})}{event.endAt ? `–${new Date(event.endAt).toLocaleTimeString('ru-RU', {hour:'2-digit',minute:'2-digit'})}` : ''} · {event.eventType === 'training' ? 'Тренировка' : 'Мероприятие'}</small></span>
      <span className={`studio-status status-${event.status}`}>{historyStatusLabel(event.status)}</span>
      <span className="studio-event-row-payment">{event.debtAmount > 0 ? <><small>К оплате</small><strong>{formatMoney(event.debtAmount)}</strong></> : event.status === 'completed' ? 'Взносы закрыты' : 'Встреча команды'}</span>
      <Icon name="arrow" size={18}/>
    </button>
    return <button key={event.instanceID} className="studio-event-item" onClick={() => open(event.instanceID)}><header><span className="studio-event-date"><strong>{date.getDate()}</strong><small>{date.toLocaleDateString('ru-RU',{month:'long'})}</small></span><span className={`studio-status status-${event.status}`}>{historyStatusLabel(event.status)}</span></header><h3>{event.name}</h3><p><Icon name="clock" size={16}/>{date.toLocaleDateString('ru-RU',{weekday:'long'})} · {date.toLocaleTimeString('ru-RU',{hour:'2-digit',minute:'2-digit'})}{event.endAt?`–${new Date(event.endAt).toLocaleTimeString('ru-RU',{hour:'2-digit',minute:'2-digit'})}`:''}</p><p><Icon name="ball" size={16}/>{event.eventType==='training'?'Тренировка':'Мероприятие'}</p><footer><span>{event.debtAmount>0 ? `К оплате ${formatMoney(event.debtAmount)}` : event.status==='completed' ? 'Взносы закрыты' : 'Встреча команды'}</span><span>Открыть<Icon name="arrow" size={16}/></span></footer></button>
  })}</div>
}

export function StudioPolls({ items, open }: { items: GroupPollItem[]; open: (id: number) => void }) {
  return <div className="studio-polls-list">{items.map(poll => <button key={poll.postID} className="studio-poll-item" onClick={()=>open(poll.postID)}><span className="studio-poll-symbol"><Icon name="poll" size={23}/></span><span className="studio-poll-copy"><small>{poll.eventName || 'Голосование команды'}</small><strong>{poll.question || poll.templateName || 'Голосование'}</strong><span className="studio-poll-progress"><i><b style={{width:`${poll.totalVotes?Math.min(100,100*poll.countedVotes/poll.totalVotes):0}%`}}/></i><span>В учёте {poll.countedVotes} из {poll.totalVotes} голосов</span></span></span><span className="studio-poll-meta"><span className="badge">{['closed','stopped','completed'].includes(poll.status)?'Завершено':['open','active','published'].includes(poll.status)?'Открыто':poll.status}</span><small>{new Date(poll.publishedAt).toLocaleDateString('ru-RU',{day:'numeric',month:'long'})}</small></span><Icon name="arrow" size={18}/></button>)}</div>
}

export function StudioEventTemplates({ items, archived, open, create, archive, restore }: { items: EventView[]; archived: boolean; open: (id:number)=>void; create:(id:number)=>void; archive:(id:number)=>void; restore:(id:number)=>void }) {
  return <div className="studio-event-template-list">{items.map(event=><article key={event.id} className="studio-event-template"><div className="studio-template-symbol"><Icon name="repeat" size={25}/></div><div className="studio-event-template-copy"><button className="studio-template-title" disabled={archived} onClick={()=>open(event.id)}>{event.name}<Icon name="arrow" size={16}/></button><p>{weekdayLabel(event.startWeekday)} · {toHourMinute(event.startTime)}–{toHourMinute(event.endTime)}</p><div><span className="badge">{event.eventType==='training'?'Тренировка':'Мероприятие'}</span><span>{event.pollTemplate || 'Без шаблона голосования'}</span><span>{event.publishEnabled?'Публикации включены':'Публикации выключены'}</span></div></div><div className="studio-template-actions"><strong>{formatMoney(event.costAmount)}</strong>{archived?<button className="btn-secondary" onClick={()=>restore(event.id)}>Восстановить</button>:<><button onClick={()=>create(event.id)}><Icon name="plus" size={16}/>Создать встречу</button><button className="btn-icon btn-secondary" aria-label={`В архив: ${event.name}`} title="В архив" onClick={()=>archive(event.id)}><Icon name="template" size={17}/></button></>}</div></article>)}</div>
}

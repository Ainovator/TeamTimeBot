import { Icon } from '../../components/Icon'
import { Select } from '../../components/Select'
import { historyStatusLabel } from '../history/historyUtils'
import type { EventHistoryItem, EventPollHistoryItem } from '../../types'
import './eventDetail.css'

function validDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function shortDateTime(value: string) {
  return validDate(value)?.toLocaleString('ru-RU', {
    day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit',
  }) ?? 'Дата не указана'
}

export function EventDetailHeader({ event }: {
  event: EventHistoryItem
}) {
  const start = validDate(event.nextStartAt)
  const end = validDate(event.endAt)
  const time = (date: Date) => date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  const sameDay = start && end && start.toDateString() === end.toDateString()
  const timeRange = start
    ? `${time(start)}${end ? ` — ${sameDay ? time(end) : shortDateTime(event.endAt)}` : ''}`
    : end ? `Окончание: ${shortDateTime(event.endAt)}` : 'Время не указано'

  return <header className="studio-event-summary">
    <div className="studio-event-summary-copy">
      <div className="studio-event-summary-meta">
        <span className="studio-event-kicker">{event.eventType === 'training' ? 'Тренировка' : 'Мероприятие'} · Событие #{event.instanceID}</span>
        <span className={`studio-status status-${event.status}`}>{historyStatusLabel(event.status)}</span>
      </div>
      <h2>{event.name}</h2>
      <div className="studio-event-schedule">
        <span><Icon name="calendar" size={18}/>{start?.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric', weekday: 'long' }) ?? 'Дата не указана'}</span>
        <span><Icon name="clock" size={18}/>{timeRange}</span>
      </div>
    </div>
  </header>
}

export function EventPollContext({ polls, selectedPostID, loading, onSelect, mode }: {
  polls: EventPollHistoryItem[]
  selectedPostID: number | null
  loading: boolean
  onSelect: (postID: number) => void
  mode: 'roster' | 'votes'
}) {
  if (loading) return <p className="studio-event-poll-empty" role="status">Загружаем голосование события…</p>
  const poll = polls.find(item => item.postID === selectedPostID)
  if (!polls.length) return <p className="studio-event-poll-empty">Для этого события ещё нет опубликованных голосований.</p>

  const rosterMode = mode === 'roster'
  if (rosterMode && polls.length === 1) return null

  return <section className={rosterMode ? 'studio-event-roster-context' : 'studio-event-poll-context'} aria-label={rosterMode ? 'Источник состава' : 'Голосование события'}>
    {polls.length > 1 && <label className="studio-event-poll-picker">
      <span>{rosterMode ? 'Состав по голосованию' : 'Голосование'}</span>
      <Select value={selectedPostID ?? ''} onChange={event => onSelect(Number(event.target.value))}>
        {selectedPostID === null && <option value="" disabled>Выберите голосование</option>}
        {polls.map(item => <option key={item.postID} value={item.postID}>#{item.postID} · {shortDateTime(item.publishedAt)}{rosterMode ? '' : ` · ${item.question}`}</option>)}
      </Select>
    </label>}
    {!rosterMode && poll && <div className="studio-event-poll-overview">
      <div className="studio-event-poll-copy">
        <span className="studio-event-kicker">Голосование #{poll.postID}</span>
        <h3>{poll.question || 'Запись на событие'}</h3>
        <p>Опубликовано {shortDateTime(poll.publishedAt)}</p>
      </div>
      <dl className="studio-event-poll-counts">
        <div><dt>Ответов в учёте</dt><dd>{poll.countedVotes}</dd></div>
        <div><dt>Всего ответов</dt><dd>{poll.totalVotes}</dd></div>
      </dl>
    </div>}
  </section>
}

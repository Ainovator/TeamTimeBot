import { useEffect, useState } from 'react'
import { fetchEventMentionMembers } from '../../api'
import { useErrorState } from '../../components/ErrorNotifications'
import { Checkbox } from '../../components/Checkbox'
import { Icon } from '../../components/Icon'
import type { EventMentionSettings, GroupMember } from '../../types'
import './eventMentions.css'

export const emptyEventMentions = (): EventMentionSettings => ({ userIDs: [], onPoll: true, onAnnouncement: true })
export const equalEventMentions = (left: EventMentionSettings, right = emptyEventMentions()) =>
  left.onPoll === right.onPoll && left.onAnnouncement === right.onAnnouncement &&
  [...left.userIDs].sort((a, b) => a - b).join(',') === [...(right.userIDs ?? [])].sort((a, b) => a - b).join(',')

export function EventMentionsEditor({ chatID, value, onChange }: {
  value: EventMentionSettings; onChange: (value: EventMentionSettings) => void
  chatID: number
}) {
  const [members, setMembers] = useState<GroupMember[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useErrorState()
  const [query, setQuery] = useState('')
  useEffect(() => {
    let alive = true
    setLoading(true)
    setError('')
    void fetchEventMentionMembers(chatID).then(rows => { if (alive) setMembers(rows) })
      .catch(error => { if (alive) setError((error as Error).message) })
      .finally(() => { if (alive) setLoading(false) })
    return () => { alive = false }
  }, [chatID, setError])
  const options = members.filter(member => member.status === 'active').map(member => ({
    id: member.userTelegramID,
    name: member.realName?.trim() || `${member.firstName ?? ''} ${member.lastName ?? ''}`.trim() || member.username || `Игрок ${member.userTelegramID}`,
    username: member.username,
  }))
  if (!loading && !error) {
    for (const id of value.userIDs) {
      if (!options.some(option => option.id === id)) options.push({ id, name: `Игрок ${id} · больше не в списке`, username: '' })
    }
  }
  const search = query.trim().toLocaleLowerCase('ru')
  const filtered = options.filter(option => `${option.name} ${option.username || ''}`.toLocaleLowerCase('ru').includes(search))
  const allSelected = filtered.length > 0 && filtered.every(option => value.userIDs.includes(option.id))
  function toggle(id: number, checked: boolean) {
    onChange({ ...value, userIDs: checked ? [...new Set([...value.userIDs, id])] : value.userIDs.filter(item => item !== id) })
  }
  return <section className="studio-event-mentions" aria-label="Упоминания игроков">
    <div className="studio-event-mentions-heading"><div><h4>Кого упоминать</h4><p>Бот упомянет выбранных игроков в Telegram при публикации по этому шаблону.</p></div><span>Выбрано: {value.userIDs.length}</span></div>
    <div className="studio-event-mention-modes">
      <label><Checkbox checked={value.onPoll} onChange={event => onChange({ ...value, onPoll: event.target.checked })}/><span>При публикации опроса</span></label>
      <label><Checkbox checked={value.onAnnouncement} onChange={event => onChange({ ...value, onAnnouncement: event.target.checked })}/><span>В напоминании о тренировке</span></label>
    </div>
    <div className="studio-event-mention-tools">
      <label className="studio-search"><Icon name="search" size={18}/><input aria-label="Поиск игроков для упоминания" placeholder="Имя или ник игрока" value={query} onChange={event => setQuery(event.target.value)}/></label>
      <button type="button" className="btn-secondary" disabled={loading || Boolean(error) || !filtered.length} onClick={() => {
        const ids = new Set(value.userIDs)
        filtered.forEach(option => allSelected ? ids.delete(option.id) : ids.add(option.id))
        onChange({ ...value, userIDs: [...ids] })
      }}>{allSelected ? 'Снять выбор' : query.trim() ? 'Выбрать найденных' : 'Выбрать всех'}</button>
    </div>
    {loading ? <p className="muted" role="status">Загружаем игроков…</p> : error ? <p className="muted">Не удалось загрузить игроков. Обновите страницу, чтобы изменить список.</p> :
      <div className="studio-event-mention-list">{filtered.map(option => <label key={option.id}>
        <Checkbox aria-label={`Упоминать: ${option.name}`} checked={value.userIDs.includes(option.id)} onChange={event => toggle(option.id, event.target.checked)}/>
        <span><strong>{option.name}</strong>{option.username && <small>@{option.username}</small>}</span>
      </label>)}{!filtered.length && <p className="muted">{query.trim() ? 'Игроки не найдены.' : 'В организации пока нет игроков.'}</p>}</div>}
    {value.userIDs.length > 0 && !value.onPoll && !value.onAnnouncement && <p className="muted">Упоминания выключены. Выберите, в каких сообщениях использовать этот список.</p>}
  </section>
}

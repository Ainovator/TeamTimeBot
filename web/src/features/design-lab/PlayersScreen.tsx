import { Checkbox } from '../../components/Checkbox'
import { Select } from '../../components/Select'
import { useState } from 'react'
import { Badge, DemoDialog, Icon } from './components'
import { roster, rosterRoles, skillLabels, type RosterPlayer } from './rosterData'
import './players.css'
import { PlayerPortrait as Portrait } from './PlayerPortrait'

function Skills({ player, compact = false }: { player: RosterPlayer; compact?: boolean }) {
  return <div className={compact ? 'dl-roster-skills-compact' : 'dl-roster-skills'}>{player.skills.slice(0, compact ? 3 : 5).map((value, index) => <div key={skillLabels[index]}><span>{skillLabels[index]}<strong>{value.toFixed(1)}</strong></span><i><b style={{ width: `${value * 10}%` }}/></i></div>)}</div>
}

function PlayerCard({ player, open }: { player: RosterPlayer; open: () => void }) {
  const [first, last] = player.name.split(' ')
  return <article className={`dl-roster-card dl-roster-tone-${player.color}`}>
    <button className="dl-roster-card-open" onClick={open} aria-label={`Открыть профиль: ${player.name}`}>
      <div className="dl-roster-photo"><Portrait player={player}/><span className="dl-roster-shirt" aria-label={`Номер ${player.number}`}>{player.number}</span><span className="dl-roster-position">{player.role}</span>{player.clubRole !== 'Игрок' && <span className="dl-roster-club-role">{player.clubRole}</span>}</div>
      <div className="dl-roster-card-body"><div className="dl-roster-name-row"><h2><span>{first}</span>{last}</h2><span className="dl-roster-level"><strong>{player.skill.toFixed(1)}</strong><small>уровень</small></span></div><p className="dl-roster-strength"><Icon name="sparkle" size={13}/>{player.strength}</p><Skills player={player} compact/><div className="dl-roster-card-foot"><span className={player.ready ? 'dl-roster-ready' : 'dl-roster-away'}><i/>{player.ready ? 'На ближайшей игре' : 'Пропускает игру'}</span><span className="dl-roster-open-arrow"><Icon name="diagonal" size={17}/></span></div></div>
    </button>
  </article>
}

function CourtLineup({ open }: { open: (player: RosterPlayer) => void }) {
  // A six-position preview, not the product's automatic team balancing algorithm.
  const positions = [2, 0, 1, 3, 5, 4]
  return <><p className="dl-roster-court-intro">Шесть игроков, у каждого своя сильная сторона. Нажмите на игрока, чтобы познакомиться ближе.</p><div className="dl-roster-court"><span className="dl-roster-net">СЕТКА</span><div className="dl-roster-court-players">{positions.map((id, index) => <button key={id} onClick={() => open(roster[id])}><span className="dl-roster-court-zone">{[4, 3, 2, 5, 6, 1][index]}</span><Portrait player={roster[id]}/><strong>{roster[id].name.split(' ')[0]}</strong><small>{roster[id].role}</small></button>)}</div></div><p className="dl-demo-note">Иллюстрация расстановки из шести демопрофилей. Состав на реальную встречу формируется в событии.</p></>
}

function PlayerDetails({ player }: { player: RosterPlayer }) {
  return <div className={`dl-roster-profile dl-roster-tone-${player.color}`}><div className="dl-roster-profile-cover"><Portrait player={player}/><span className="dl-roster-shirt">{player.number}</span><span className="dl-roster-position">{player.role}</span></div><div className="dl-roster-profile-copy"><p className="dl-eyebrow">ОРБИТА / {player.clubRole.toUpperCase()}</p><h3>{player.title}</h3><p className="dl-roster-profile-note">С командой с {player.joined.toLowerCase()} года</p><div className="dl-roster-profile-stats"><div><strong>{player.skill.toFixed(1)}</strong><span>уровень игры</span></div><div><strong>{player.games}</strong><span>тренировок</span></div><div><strong>{player.attendance}%</strong><span>посещаемость</span></div></div><h4>Сильные стороны на площадке</h4><Skills player={player}/><div className="dl-roster-profile-status"><Icon name="calendar" size={18}/><span><strong>{player.ready ? 'Вечерний волейбол · 14 сентября' : 'Следующая встреча · 16 сентября'}</strong><small>{player.ready ? 'В составе · 19:00–21:00' : 'Планирует присоединиться'}</small></span></div><p className="dl-demo-note">Демонстрационные портрет, навыки и история игрока.</p></div></div>
}

export function PlayersScreen() {
  const [query, setQuery] = useState('')
  const [role, setRole] = useState('Все игроки')
  const [sort, setSort] = useState('По составу')
  const [view, setView] = useState<'cards' | 'list'>('cards')
  const [onlyReady, setOnlyReady] = useState(false)
  const [selected, setSelected] = useState<RosterPlayer | null>(null)
  const [court, setCourt] = useState(false)
  const filtered = roster.filter(p => p.name.toLowerCase().includes(query.trim().toLowerCase()) && (role === 'Все игроки' || p.role === role) && (!onlyReady || p.ready)).sort((a, b) => sort === 'По уровню' ? b.skill - a.skill : sort === 'По фамилии' ? a.name.split(' ')[1].localeCompare(b.name.split(' ')[1], 'ru') : a.id - b.id)
  const average = (roster.reduce((sum, p) => sum + p.skill, 0) / roster.length).toFixed(1)
  function reset() { setQuery(''); setRole('Все игроки'); setOnlyReady(false) }
  return <section className="dl-roster-screen">
    <div className="dl-roster-heading"><div><p className="dl-eyebrow">ОРБИТА / ЛЮДИ КЛУБА</p><h1>Наша команда<span>.</span></h1><p>Разные характеры. Общий ритм игры.</p></div><button className="dl-button dl-secondary" onClick={() => setCourt(true)}><Icon name="grid" size={17}/>На площадке<Icon name="diagonal" size={16}/></button></div>
    <div className="dl-roster-summary"><div className="dl-roster-summary-people"><span className="dl-roster-small-stack">{[0, 1, 3].map(id => <Portrait key={id} player={roster[id]}/>)}</span><span><strong>{roster.length} игроков</strong><small>в демонстрационном составе</small></span></div><div className="dl-roster-summary-stat"><strong>{average}</strong><span>средний<br/>уровень игры</span></div><div className="dl-roster-summary-stat"><strong>{roster.filter(p => p.ready).length}<i> / {roster.length}</i></strong><span>встречаемся<br/>в понедельник</span></div><span className="dl-roster-season">СЕЗОН<br/><strong>20—26</strong></span></div>
    <div className="dl-roster-controls"><label className="dl-search"><Icon name="search" size={19}/><input aria-label="Найти игрока" placeholder="Имя или фамилия игрока" value={query} onChange={e => setQuery(e.target.value)}/></label><Select aria-label="Сортировка игроков" value={sort} onChange={e => setSort(e.target.value)}>{['По составу', 'По фамилии', 'По уровню'].map(s => <option key={s}>{s}</option>)}</Select><div className="dl-roster-view" role="group" aria-label="Вид списка игроков"><button aria-label="Карточки игроков" aria-pressed={view === 'cards'} onClick={() => setView('cards')}><Icon name="grid" size={18}/></button><button aria-label="Список игроков" aria-pressed={view === 'list'} onClick={() => setView('list')}><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true"><path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/></svg></button></div></div>
    <div className="dl-roster-filter-row"><div className="dl-roster-role-tabs" role="group" aria-label="Позиции игроков"><button aria-pressed={role === 'Все игроки'} onClick={() => setRole('Все игроки')}>Все игроки<span>{roster.length}</span></button>{rosterRoles.map(r => <button key={r.name} className={`dl-roster-tone-${r.color}`} aria-pressed={role === r.name} onClick={() => setRole(r.name)}><i/>{r.plural}<span>{roster.filter(p => p.role === r.name).length}</span></button>)}</div><label className="dl-roster-ready-filter"><Checkbox checked={onlyReady} onChange={e => setOnlyReady(e.target.checked)}/>На ближайшей игре</label></div>
    <div className="dl-roster-results-caption"><span>{role === 'Все игроки' ? 'СОСТАВ КЛУБА' : role.toUpperCase()}</span><span aria-live="polite">Показано {filtered.length} из {roster.length}</span></div>
    {view === 'cards' ? <div className="dl-roster-grid">{filtered.map(p => <PlayerCard key={p.id} player={p} open={() => setSelected(p)}/>)}</div> : <div className="dl-roster-list">{filtered.map(p => <button className={`dl-roster-list-row dl-roster-tone-${p.color}`} key={p.id} onClick={() => setSelected(p)} aria-label={`Открыть профиль: ${p.name}`}><span className="dl-roster-list-number">{p.number}</span><Portrait player={p}/><span className="dl-roster-list-name"><strong>{p.name}</strong><small>{p.strength}</small></span><span className="dl-roster-list-role">{p.role}</span><span className="dl-roster-list-level"><strong>{p.skill.toFixed(1)}</strong><small>уровень</small></span><span className={p.ready ? 'dl-roster-ready' : 'dl-roster-away'}><i/>{p.ready ? 'В составе' : 'Пропускает'}</span><Icon name="diagonal" size={18}/></button>)}</div>}
    {!filtered.length && <div className="dl-roster-empty"><Icon name="users" size={35}/><h2>Кажется, здесь пока никого</h2><p>Попробуйте другое имя или выберите все позиции.</p><button className="dl-button dl-secondary" onClick={reset}>Сбросить фильтры<Icon name="back" size={16}/></button></div>}
    <p className="dl-roster-footnote"><Icon name="ball" size={16}/><span>Хорошая команда складывается из разных сильных сторон.</span><small>Портреты и данные вымышлены</small></p>
    {selected && <DemoDialog title={selected.name} close={() => setSelected(null)}><PlayerDetails player={selected}/></DemoDialog>}
    {court && <DemoDialog title="Шестёрка на площадке" close={() => setCourt(false)}><Badge>Пример расстановки</Badge><CourtLineup open={player => { setCourt(false); setSelected(player) }}/></DemoDialog>}
  </section>
}

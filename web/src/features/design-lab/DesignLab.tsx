import { useEffect, useRef, useState, type FormEvent } from 'react'
import { concepts, initialEvents, navigation, players, type Concept, type DemoEvent, type Screen } from './demoData'
import '@fontsource-variable/golos-text'
import './design-lab.css'
import './studio.css'
import './product-screens.css'
import { Icon, Avatar, AvatarStack, Badge, SectionTitle, DemoDialog } from './components'
import { ProductScreens } from './ProductScreens'
import { PlayersScreen } from './PlayersScreen'

function Court({ club = false }: { club?: boolean }) {
  return <svg className={`dl-court${club ? ' dl-court-club' : ''}`} viewBox="0 0 480 330" fill="none" aria-hidden="true">
    <ellipse cx="257" cy="286" rx="193" ry="22" fill="currentColor" opacity=".08"/>
    <g transform="translate(18 38) rotate(-12 220 130)">
      <path d="M41 100 291 34 439 164 184 242Z" fill="currentColor" opacity=".13"/>
      <path d="M41 100 291 34 439 164 184 242 41 100Zm72-19 146 139m-4-176 147 139M41 100l143 142" stroke="currentColor" strokeWidth="2"/>
      <path d="m171 66 143 139v-71L171-2v68Z" fill="currentColor" opacity=".08"/>
      <path d="m171 82 0-88m143 221v-91M171-2l143 136" stroke="currentColor" strokeWidth="3"/>
      {[0,1,2,3,4,5,6,7].map(i => <path key={i} d={`m${179+i*17} ${6+i*16}v62`} stroke="currentColor" opacity=".3"/>)}
      {[0,1,2,3].map(i => <path key={i} d={`m171 ${10+i*16} 143 136`} stroke="currentColor" opacity=".3"/>)}
      <circle cx="307" cy="25" r="30" fill="var(--dl-ball, #e4efca)" stroke="currentColor" strokeWidth="2"/>
      <path d="M285 5c16 2 29 13 36 36m-39-6c8-9 20-14 37-12m-17-27c-6 13-6 24 0 36" stroke="currentColor" strokeWidth="2"/>
      <circle cx="123" cy="140" r="8" fill="currentColor" opacity=".65"/><circle cx="220" cy="180" r="8" fill="currentColor" opacity=".65"/>
      <circle cx="300" cy="91" r="8" fill="currentColor" opacity=".4"/><circle cx="350" cy="156" r="8" fill="currentColor" opacity=".4"/>
    </g>
  </svg>
}

function Brand() { return <div className="dl-brand"><span className="dl-brand-symbol"><Icon name="ball" size={26}/></span><span>teamtime<span className="dl-brand-dot">.</span></span></div> }
function EventCard({ event, open, featured = false }: { event: DemoEvent; open: (event: DemoEvent) => void; featured?: boolean }) {
  return <article className={`dl-event-card${featured ? ' dl-event-featured' : ''}`}>
    <div className="dl-event-card-top"><div className="dl-date-block"><strong>{event.day}</strong><span>{event.date.split(' ').slice(1).join(' ').toUpperCase()} · {event.weekday}</span></div><Badge muted={event.count >= event.limit}>{event.status}</Badge></div>
    <h3>{event.title}</h3>
    <p className="dl-meta"><Icon name="clock" size={16}/>{event.time}<span>·</span>{event.price} ₽ / игрок</p>
    <p className="dl-meta"><Icon name="pin" size={16}/>{event.place}</p>
    <div className="dl-event-capacity"><span>{event.count} <small>из {event.limit} игроков</small></span><small>{event.limit-event.count > 0 ? `Свободно: ${event.limit-event.count}` : 'Полный состав'}</small></div>
    <div className="dl-progress"><span style={{ width: `${event.count / event.limit * 100}%` }}/></div>
    <div className="dl-event-card-bottom"><AvatarStack count={event.count}/><button className="dl-icon-button" onClick={() => open(event)} aria-label={`Открыть событие «${event.title}»`}><Icon name="arrow"/></button></div>
  </article>
}

function Stats({ arena = false }: { arena?: boolean }) {
  return <div className="dl-stats">{[
    ['48', 'игроков в клубе', '+4 за месяц', 'users'],
    ['12', 'игр в сентябре', '3 на этой неделе', 'calendar'],
    ['89%', 'посещаемость', '↑ 7% к августу', 'ball'],
    ['9 600 ₽', 'собрано за игру', '16 из 18 оплатили', 'wallet'],
  ].map(([value, label, note, icon], i) => <div className="dl-stat" key={label}><div className="dl-stat-label">{label}{!arena && <Icon name={icon} size={18}/>}</div><strong>{value}<span>{arena ? ` / 0${i+1}` : ''}</span></strong><small>{note}</small></div>)}</div>
}

function Teams({ onOpen }: { onOpen: () => void }) {
  return <section className="dl-panel dl-teams"><SectionTitle title="Хорошая игра — равные команды"/><div className="dl-teams-versus"><div><span className="dl-team-circle">A</span><strong>Орбита</strong><span>8.2 <small>средний уровень</small></span></div><span className="dl-vs">vs</span><div><span className="dl-team-circle dl-team-b">B</span><strong>Импульс</strong><span>8.1 <small>средний уровень</small></span></div></div><div className="dl-balance"><span>Баланс составов</span><strong>98%</strong></div><div className="dl-progress"><span style={{width:'98%'}}/></div><button className="dl-text-button" onClick={onOpen}>Посмотреть составы<Icon name="arrow" size={16}/></button></section>
}

function Activity() {
  return <section className="dl-panel dl-activity"><SectionTitle title="Жизнь команды"/>{[
    ['Мария записалась на игру', 'Вечерний волейбол · 10 минут назад', 1],
    ['Алексей оплатил тренировку', '600 ₽ · 25 минут назад', 0],
    ['Анна присоединилась к клубу', 'Добро пожаловать! · час назад', 3],
  ].map(([title, detail, index]) => <div className="dl-activity-row" key={String(title)}><Avatar index={Number(index)}/><div><strong>{title}</strong><small>{detail}</small></div></div>)}</section>
}

type OverviewProps = { events: DemoEvent[]; open: (event: DemoEvent) => void; navigate: (screen: Screen) => void; create: () => void; teams: () => void }

function StudioOverview({ events, open, navigate, create, teams }: OverviewProps) {
  return <>
    <div className="dl-page-heading"><div><p className="dl-eyebrow">ПОНЕДЕЛЬНИК, 14 СЕНТЯБРЯ</p><h1>Команда в ритме<span>.</span></h1><p>Меньше организации. Больше хорошей игры.</p></div><button className="dl-button dl-primary" onClick={create}><Icon name="plus" size={18}/>Создать событие</button></div>
    <Stats/>
    <div className="dl-studio-feature"><div className="dl-feature-copy"><Badge>Ближайшая игра · сегодня</Badge><h2>Вечер понедельника.<br/>Встречаемся на площадке.</h2><p>19:00–21:00 <span>·</span> Спортзал «Динамо»</p><div className="dl-feature-bottom"><button className="dl-button dl-primary" onClick={() => open(events[0])}>К событию<Icon name="arrow" size={17}/></button><AvatarStack/></div></div><Court/></div>
    <SectionTitle title="На этой неделе" action="Все события" onClick={() => navigate('events')}/>
    <div className="dl-week-strip">{['Пн','Вт','Ср','Чт','Пт','Сб','Вс'].map((day,i) => <button key={day} className={`${i===0 ? 'dl-day-selected' : ''} ${[0,2,5].includes(i) ? 'dl-day-event' : ''}`} onClick={() => [0,2,5].includes(i) ? open(events[[0,2,5].indexOf(i)]) : navigate('events')}><span>{day}</span><strong>{14+i}</strong><small>{[0,2,5].includes(i) ? '●' : '—'}</small></button>)}</div>
    <div className="dl-two-columns"><Teams onOpen={teams}/><Activity/></div>
  </>
}

function ArenaOverview({ events, open, navigate, create, teams }: OverviewProps) {
  return <><div className="dl-arena-hero"><div className="dl-arena-copy"><p className="dl-eyebrow"><span className="dl-live-dot"/>СЕЗОН 2026 / КЛУБ «ОРБИТА»</p><h1>ТВОЯ КОМАНДА.<br/><em>ТВОЯ ИГРА.</em></h1><p>Собирай состав. Держи темп. Выходи побеждать.</p><button className="dl-button dl-primary" onClick={() => open(events[0])}>НА ПЛОЩАДКУ<Icon name="diagonal"/></button></div><div className="dl-arena-art"><span className="dl-art-number">18</span><Court/><span className="dl-art-caption">ONE COURT. ONE TEAM.</span></div></div>
    <Stats arena/>
    <div className="dl-arena-content"><div><SectionTitle title="СЛЕДУЮЩИЙ ВЫХОД" action="Расписание" onClick={() => navigate('events')}/><div className="dl-arena-events">{events.slice(0,2).map(e => <EventCard event={e} open={open} key={e.id}/>)}</div></div><section className="dl-panel dl-score-panel"><p className="dl-eyebrow">ПОСЛЕДНЯЯ ИГРА · 12 СЕН</p><div className="dl-score-names"><span>ОРБИТА</span><span>ИМПУЛЬС</span></div><div className="dl-big-score">3 <span>:</span> 2</div><div className="dl-set-scores"><span>25:21</span><span>22:25</span><span>25:18</span><span>23:25</span><span>15:12</span></div><button className="dl-button dl-secondary" onClick={teams}>СОСТАВЫ КОМАНД<Icon name="arrow" size={16}/></button></section></div>
    <div className="dl-arena-banner"><Icon name="sparkle" size={28}/><div><strong>СИЛЬНАЯ ИГРА НАЧИНАЕТСЯ С БАЛАНСА.</strong><p>Навыки, позиции и сыгранность — учтём всё при подборе команд.</p></div><button className="dl-button dl-primary" onClick={create}>НОВАЯ ИГРА<Icon name="plus" size={18}/></button></div>
  </>
}

function ClubOverview({ events, open, navigate, create }: OverviewProps) {
  return <><div className="dl-club-hero"><div><p className="dl-eyebrow">ЛЮБИМОЕ ДЕЛО. СВОИ ЛЮДИ.</p><h1>Хорошая игра<br/>начинается <em>с людей.</em></h1><p>Место, где из «кто сегодня играет?»<br/>получается настоящая команда.</p><button className="dl-button dl-primary" onClick={() => open(events[0])}>Увидимся на игре<Icon name="arrow" size={18}/></button><div className="dl-club-people"><AvatarStack/><span><strong>48 своих людей</strong><small>И всегда рады новым</small></span></div></div><div className="dl-club-art"><span className="dl-club-orbit"/><Court club/><div className="dl-club-seal"><Icon name="ball" size={32}/><span>ИГРАЕМ<br/>ВМЕСТЕ</span></div><span className="dl-club-note">У каждой команды<br/>есть своё время.</span></div></div>
    <div className="dl-club-facts"><span><strong>48</strong> участников</span><span><strong>12</strong> встреч в месяц</span><span><strong>1</strong> общая любовь к игре</span><Icon name="ball" size={24}/></div>
    <SectionTitle title="Поводы встретиться" action="Всё расписание" onClick={() => navigate('events')}/>
    <div className="dl-events-grid">{events.slice(0,3).map((e,i) => <EventCard key={e.id} event={e} open={open} featured={i===0}/>)}</div>
    <div className="dl-club-bottom"><section><p className="dl-eyebrow">У КАЖДОГО ЕСТЬ СВОЯ РОЛЬ</p><h2>Разные люди.<br/><em>Одна команда.</em></h2><button className="dl-text-button" onClick={() => navigate('members')}>Познакомиться с игроками<Icon name="arrow" size={18}/></button></section><div className="dl-club-player"><Avatar index={3} large/><strong>Анна</strong><span>Либеро · душа защиты</span></div><div className="dl-club-player"><Avatar index={0} large/><strong>Алексей</strong><span>Связующий · задаёт ритм</span></div><button className="dl-club-invite" onClick={create}><Icon name="plus" size={30}/><strong>Соберёмся?</strong><span>Создать свою встречу</span></button></div>
  </>
}

function readScreen(): Screen {
  const value = new URLSearchParams(window.location.search).get('screen')
  return navigation.some(item => item.id === value) ? value as Screen : 'overview'
}

function Experience({ concept }: { concept: Concept }) {
  const [screen, setScreen] = useState<Screen>(readScreen)
  const [visited, setVisited] = useState<Screen[]>([readScreen()])
  useEffect(() => { const onPop = () => { const value = readScreen(); setScreen(value); setVisited(previous => previous.includes(value) ? previous : [...previous, value]) }; window.addEventListener('popstate', onPop); return () => window.removeEventListener('popstate', onPop) }, [])
  const [events, setEvents] = useState(initialEvents)
  const [selected, setSelected] = useState<DemoEvent | null>(null)
  const [modal, setModal] = useState<'create' | 'teams' | null>(null)
  const [joined, setJoined] = useState<number[]>([])
  const [filter, setFilter] = useState('Все события')
  const [reversed, setReversed] = useState(false)
  const contentRef = useRef<HTMLDivElement>(null)
  function navigate(value: Screen) { setScreen(value); setVisited(previous => previous.includes(value) ? previous : [...previous, value]); const url = new URL(window.location.href); url.searchParams.set('screen', value); window.history.pushState({}, '', url); contentRef.current?.scrollIntoView({ block: 'start', behavior: 'smooth' }) }
  function createEvent(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const data = new FormData(e.currentTarget)
    const date = new Date(`${data.get('date')}T12:00:00`)
    const event: DemoEvent = { id: Date.now(), title: String(data.get('title')).trim(), day: String(date.getDate()).padStart(2,'0'), weekday: date.toLocaleDateString('ru-RU',{weekday:'short'}).toUpperCase(), date: date.toLocaleDateString('ru-RU',{day:'numeric',month:'long'}), time: `${data.get('time')}–${data.get('end')}`, place: String(data.get('place')).trim(), count:0, limit:18, price:Number(data.get('price')), status:'Открыта запись' }
    if (!event.title || !event.place) return
    setEvents(value => [...value,event]); setModal(null); navigate('events'); setFilter('Все события')
  }
  const nav = <nav className="dl-navigation dl-product-nav" aria-label="Разделы демонстрации">{navigation.map(item => <button className={screen === item.id ? 'dl-nav-active' : ''} aria-current={screen===item.id ? 'page' : undefined} onClick={() => navigate(item.id)} key={item.id}><Icon name={item.icon} size={19}/><span>{item.label}</span>{item.id==='events' && <small>{events.length}</small>}</button>)}</nav>
  const overviewProps: OverviewProps = {events, open:setSelected, navigate, create:() => setModal('create'), teams:() => setModal('teams')}
  return <div className={`dl-concept dl-${concept}`}>
    {concept === 'studio' && (
      <aside className="dl-studio-rail" aria-label="Навигация клуба">
        <button className="dl-rail-logo" onClick={() => navigate('overview')} aria-label="На главную клуба">
          <Icon name="ball" size={28}/>
        </button>
        {nav}
        <button className="dl-rail-create" onClick={() => setModal('create')} aria-label="Создать встречу">
          <Icon name="plus" size={22}/>
          <span>Создать</span>
        </button>
        <span className="dl-rail-season">20 / 26</span>
      </aside>
    )}
    <div className="dl-workspace">
      <header className="dl-app-header">
        {concept === 'studio' ? (
          <>
            <div className="dl-studio-identity">
              <Brand/>
              <span className="dl-identity-divider"/>
              <span className="dl-studio-club"><strong>Орбита</strong><small>Волейбольный клуб</small></span>
            </div>
            <button className="dl-studio-account" aria-label="Открыть мой профиль" onClick={() => navigate('profile')}><span className="dl-studio-account-name">Алексей<small>Организатор</small></span><Avatar/></button>
          </>
        ) : <><Brand/>{nav}<span className="dl-header-club">Клуб «Орбита»<Avatar/></span></>}
      </header>
      <div className="dl-mobile-nav">{concept==='studio' && nav}</div>
      <main className="dl-content" ref={contentRef}>
        {screen==='overview' && (concept==='studio' ? <StudioOverview {...overviewProps}/> : concept==='arena' ? <ArenaOverview {...overviewProps}/> : <ClubOverview {...overviewProps}/>)}
        {screen==='events' && <><div className="dl-page-heading"><div><p className="dl-eyebrow">ВСТРЕЧАЕМСЯ НА ПЛОЩАДКЕ</p><h1>События клуба<span>.</span></h1><p>Запись, участники и команды — в одном месте.</p></div><button className="dl-button dl-primary" onClick={() => setModal('create')}><Icon name="plus" size={18}/>Создать событие</button></div><div className="dl-filter-tabs">{['Все события','Есть места'].map(f => <button key={f} className={filter===f ? 'dl-filter-active' : ''} onClick={() => setFilter(f)}>{f}</button>)}</div><div className="dl-events-grid">{events.filter(e => filter==='Все события' || e.count<e.limit).map(e => <EventCard key={e.id} event={e} open={setSelected}/>)}</div></>}
        {screen==='members' && <PlayersScreen/>}
        <ProductScreens screen={screen} visited={visited} navigate={navigate}/>
        <footer className="dl-app-footer"><span>teamtime. Время быть командой.</span><span>Демонстрационные данные · сентябрь 2026</span></footer>
      </main>
    </div>
    {selected && <DemoDialog title={selected.title} close={() => setSelected(null)}><Badge>{selected.status}</Badge><p className="dl-dialog-event-date">{selected.date} · {selected.time}</p><p className="dl-meta"><Icon name="pin" size={18}/>{selected.place}</p><div className="dl-dialog-summary"><div><small>Участники</small><strong>{selected.count} / {selected.limit}</strong></div><div><small>Взнос за игру</small><strong>{selected.price} ₽</strong></div></div><h3>Встречаемся на площадке</h3><p className="dl-dialog-description">Разминаемся, делимся на равные команды и играем. Возьмите сменную обувь, воду и хорошее настроение.</p><AvatarStack count={selected.count}/><button className="dl-button dl-primary dl-wide" disabled={selected.count>=selected.limit && !joined.includes(selected.id)} onClick={() => { const wasJoined=joined.includes(selected.id); const next={...selected,count:selected.count+(wasJoined ? -1 : 1)}; next.status=next.count>=next.limit ? 'Мест нет' : 'Открыта запись'; setEvents(value=>value.map(e=>e.id===next.id ? next:e)); setSelected(next); setJoined(value=>wasJoined ? value.filter(id=>id!==next.id):[...value,next.id]) }}>{joined.includes(selected.id) ? 'Вы записаны · отменить запись' : selected.count>=selected.limit ? 'Все места заняты' : 'Записаться на игру'}<Icon name={joined.includes(selected.id) ? 'check':'arrow'} size={18}/></button><p className="dl-demo-note">Деморежим: запись не отправляется в Telegram.</p></DemoDialog>}
    {modal==='create' && <DemoDialog title="Новая встреча команды" close={() => setModal(null)}><form className="dl-form" onSubmit={createEvent}><label>Название<input name="title" placeholder="Например, вечерний волейбол" required maxLength={70}/></label><label>Площадка<input name="place" defaultValue="Спортзал «Динамо»" required maxLength={80}/></label><div className="dl-form-row"><label>Дата<input name="date" type="date" defaultValue="2026-09-21" required/></label><label>Взнос, ₽<input name="price" type="number" min="0" max="50000" defaultValue="600" required/></label></div><div className="dl-form-row"><label>Начало<input name="time" type="time" defaultValue="19:00" required/></label><label>Окончание<input name="end" type="time" defaultValue="21:00" required/></label></div><p className="dl-demo-note">Событие появится только в примерочной. Лимит — 18 игроков.</p><button className="dl-button dl-primary dl-wide" type="submit">Создать демособытие<Icon name="plus" size={18}/></button></form></DemoDialog>}
    {modal==='teams' && <DemoDialog title="Равные команды — интересная игра" close={() => setModal(null)}><p className="dl-dialog-description">Пример распределения по уровню игры и позициям. В продукте подбор учитывает также предпочтения игроков.</p><div className="dl-demo-teams">{['Орбита','Импульс'].map((name,team) => <div key={name}><h3>{name} <small>{team===0 ? '8.2':'8.1'}</small></h3>{players.filter((_,i)=>(i+(reversed?1:0))%2===team).map(p=><div className="dl-team-person" key={p.name}><Avatar index={players.indexOf(p)}/><span><strong>{p.name}</strong><small>{p.role}</small></span></div>)}</div>)}</div><button className="dl-button dl-primary dl-wide" onClick={() => setReversed(value => !value)}><Icon name="sparkle" size={18}/>Другой вариант составов</button><p className="dl-demo-note">Показаны 6 демопрофилей. Это иллюстрация интерфейса подбора.</p></DemoDialog>}
  </div>
}

function MiniPreview({ concept }: { concept: Concept }) {
  return <div className={`dl-mini dl-mini-${concept}`} aria-hidden="true"><div className="dl-mini-nav"><span>◉ teamtime.</span><i/><i/><i/></div><div className="dl-mini-body">{concept==='studio' && <div className="dl-mini-sidebar"><i/><i/><i/><i/></div>}<div className="dl-mini-main"><strong>{concept==='studio' ? <>Команда в ритме.</> : concept==='arena' ? <>ТВОЯ КОМАНДА.<br/><em>ТВОЯ ИГРА.</em></> : <>Хорошая игра<br/>начинается <em>с людей.</em></>}</strong><div className="dl-mini-hero"><Court club={concept==='club'}/><i/></div><div className="dl-mini-cards"><i/><i/><i/></div></div></div></div>
}

function initialConcept(): Concept {
  const value = new URLSearchParams(window.location.search).get('concept')
  return concepts.some(c => c.id===value) ? value as Concept : 'studio'
}

export default function DesignLab() {
  const [concept, setConcept] = useState<Concept>(initialConcept)
  const [compare, setCompare] = useState(new URLSearchParams(window.location.search).get('view')==='compare')
  const [mobile, setMobile] = useState(false)
  const [favorite, setFavorite] = useState<string | null>(() => { try { return localStorage.getItem('teamtime-design-favorite') } catch { return null } })
  const [notice, setNotice] = useState('')
  const current = concepts.find(c => c.id===concept)!
  useEffect(() => { document.title='TeamTime — примерочная дизайна'; const listener=() => {setConcept(initialConcept());setCompare(new URLSearchParams(window.location.search).get('view')==='compare')}; window.addEventListener('popstate',listener);return()=>window.removeEventListener('popstate',listener) },[])
  useEffect(() => {if(!notice) return; const timer=window.setTimeout(()=>setNotice(''),3500);return()=>window.clearTimeout(timer)},[notice])
  function choose(value: Concept) { const screen=readScreen();setConcept(value);setCompare(false);window.history.pushState({},'',`/design-lab?concept=${value}${screen!=='overview' ? `&screen=${screen}`:''}`) }
  function toggleCompare() {const next=!compare;const url=new URL(window.location.href);url.searchParams.set('concept',concept);if(next) url.searchParams.set('view','compare');else url.searchParams.delete('view');setCompare(next);window.history.pushState({},'',url)}
  function saveFavorite() {const next=favorite===concept ? null:concept;setFavorite(next);try {if(next) localStorage.setItem('teamtime-design-favorite',next);else localStorage.removeItem('teamtime-design-favorite');setNotice(next ? `${current.name} сохранён как фаворит`:'Фаворит сброшен')} catch {setNotice('Выбор сохранён на время просмотра')}}
  return <div className="dl-lab"><header className="dl-lab-toolbar"><a className="dl-lab-title" href="/design-lab"><span className="dl-lab-mark"><Icon name="sparkle" size={18}/></span><span>Примерочная<small>TEAMTIME / DESIGN EXPLORATIONS</small></span></a><div className="dl-concept-tabs" role="group" aria-label="Варианты дизайна">{concepts.map((c,i)=><button key={c.id} onClick={()=>choose(c.id)} aria-pressed={concept===c.id && !compare} className={concept===c.id && !compare ? 'dl-tab-active':''}><span>0{i+1}</span>{c.name}{favorite===c.id && <span aria-label="Фаворит">♥</span>}</button>)}</div><div className="dl-lab-actions"><button onClick={toggleCompare} className={compare ? 'dl-action-active':''} aria-pressed={compare}><Icon name="grid" size={16}/><span>Сравнить</span></button><a href="/" title="Открыть основной интерфейс Studio">В продукт<Icon name="diagonal" size={16}/></a></div></header>
    {compare ? <main className="dl-comparison"><p className="dl-eyebrow">ОДИН ПРОДУКТ. ТРИ ХАРАКТЕРА.</p><h1>Найдём свой TeamTime.</h1><p className="dl-comparison-intro">Сравните настроение, структуру и акценты. В каждом варианте доступны все девять разделов: от личного профиля и шаблонов до результатов игр и задолженностей.</p><div className="dl-comparison-grid">{concepts.map((c,i)=><article key={c.id} className="dl-comparison-card"><button className="dl-mini-button" onClick={()=>choose(c.id)} aria-label={`Примерить ${c.name}`}><MiniPreview concept={c.id}/></button><div className="dl-comparison-copy"><span className="dl-concept-number">КОНЦЕПЦИЯ 0{i+1}</span><h2>{c.name}{favorite===c.id && <Icon name="heart"/>}</h2><h3>{c.tag}</h3><p>{c.description}</p><div className="dl-swatches">{c.colors.map(color=><i key={color} style={{background:color}}/>)}</div><button className="dl-compare-open" onClick={()=>choose(c.id)}>Примерить {c.name}<Icon name="arrow" size={18}/></button></div></article>)}</div><p className="dl-comparison-footnote">Основано на сценариях TeamTime: события → запись → команды → игра → оплата.</p></main> : <><section className="dl-concept-info"><div className="dl-concept-description"><span className={`dl-concept-indicator dl-indicator-${concept}`}/><p><strong>{current.tag}</strong><span>{current.description}</span></p></div><div className="dl-preview-controls"><div role="group" aria-label="Размер предпросмотра" className="dl-device-switch"><button aria-label="Компьютер" aria-pressed={!mobile} onClick={()=>setMobile(false)}><Icon name="desktop" size={17}/></button><button aria-label="Телефон" aria-pressed={mobile} onClick={()=>setMobile(true)}><Icon name="mobile" size={17}/></button></div><button className={`dl-favorite${favorite===concept ? ' dl-favorited':''}`} onClick={saveFavorite} aria-label={favorite===concept ? 'Убрать из избранного' : 'Добавить в избранное'} aria-pressed={favorite===concept}><Icon name="heart" size={17}/><span>{favorite===concept ? 'В избранном':'В избранное'}</span></button></div></section><div className={`dl-preview-stage${mobile ? ' dl-preview-mobile':''}`}><div className="dl-preview"><Experience key={concept} concept={concept}/></div></div></>}
    {notice && <div className="dl-notice" role="status"><Icon name="check" size={18}/>{notice}</div>}
  </div>
}

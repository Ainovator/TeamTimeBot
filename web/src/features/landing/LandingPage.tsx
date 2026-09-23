import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { Icon } from '../../components/Icon'
import '@fontsource-variable/golos-text'
import './landing.css'

const features = [
  { id: 'events', icon: 'calendar', title: 'События и расписание', benefit: 'Следующая игра уже в планах.', description: 'Игры и тренировки в общем расписании. Дата, время и детали встречи доступны всей команде.', image: 'events', caption: 'Расписание и история встреч команды', tags: ['Игры и тренировки', 'Шаблоны событий'] },
  { id: 'votes', icon: 'poll', title: 'Голосования и запись', benefit: 'Понятно, кто выйдет на площадку.', description: 'Участники отмечают, придут ли на встречу. Организатор видит ответы и состав в одном месте.', image: 'votes', caption: 'Ответы участников внутри события', tags: ['Ответы участников', 'Состав встречи'] },
  { id: 'billing', icon: 'wallet', title: 'Взносы и задолженности', benefit: 'Все оплаты перед глазами.', description: 'Сколько собрали, кто ещё не оплатил и за какую встречу. Отмечайте взносы и напоминайте о задолженности.', image: 'billing', caption: 'Взносы команды и неоплаченные встречи', tags: ['Учёт оплат', 'Напоминания'] },
  { id: 'passes', icon: 'template', title: 'Абонементы и посещаемость', benefit: 'Каждое занятие на счету.', description: 'Выдавайте абонементы, отмечайте посещения и следите за остатком занятий. Сроки, заморозка и история списаний рядом.', image: 'passes', caption: 'Абонементы: срок действия, оплата и остаток занятий', tags: ['Пакеты и безлимит', 'Журнал посещений'] },
  { id: 'telegram', icon: 'telegram', title: 'Интеграция с Telegram', benefit: 'Команда остаётся в своём чате.', description: 'Публикуйте анонсы и голосования через бота. Собирайте ответы там, где игроки уже общаются.', image: 'votes', caption: 'Ответы из голосования доступны организатору в TeamTime', tags: ['Анонсы встреч', 'Голосования в чате'] },
  { id: 'games', icon: 'score', title: 'История игр и результаты', benefit: 'Игра закончилась. История осталась.', description: 'Сохраняйте результаты партий и возвращайтесь к прошедшим встречам. У команды появляется общая история игр.', image: 'games', caption: 'Сыгранные партии и их результаты', tags: ['Результаты партий', 'История встреч'] },
  { id: 'players', icon: 'users', title: 'Профили игроков', benefit: 'Знайте свою команду лучше.', description: 'Амплуа, оценки навыков и предпочтения игроков помогают организатору подготовить состав к следующей игре.', image: 'players', caption: 'Игроки команды, их амплуа и рейтинги', tags: ['Амплуа и навыки', 'Предпочтения игроков'] },
] as const

function TelegramIcon({ size = 22 }: { size?: number }) {
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m21 4-3.4 16-5.4-4-2.7 2.6.5-4.5L18 7.2 7.7 13.1 3 11.5 21 4Z" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round"/></svg>
}

function Brand() {
  return <a href="/" className="lp-brand" aria-label="TeamTime — на главную"><img src="/icon.png" width="40" height="40" alt=""/><span>teamtime<span>.</span></span></a>
}

function Court({ className = '' }: { className?: string }) {
  return <svg className={`lp-court ${className}`} viewBox="0 0 600 700" fill="none" aria-hidden="true"><g stroke="currentColor" strokeWidth="1.5"><rect x="50" y="40" width="500" height="620" rx="2"/><path d="M50 350h500M50 240h500M50 460h500M300 40v620"/><circle cx="300" cy="350" r="92"/><path d="M0 350h600" strokeDasharray="6 8"/></g></svg>
}

function MatchPreview() {
  return <div className="lp-hero-visual">
    <Court/>
    <span className="lp-visual-label">ВАША КОМАНДА. ВАШ РИТМ.</span>
    <div className="lp-match-card">
      <div className="lp-match-top"><span><i/> Ближайшая встреча</span><Icon name="diagonal" size={18}/></div>
      <div className="lp-match-date"><strong>24</strong><div>сентября<span>четверг · 19:00</span></div><span className="lp-ball"><Icon name="ball" size={49}/></span></div>
      <h2>Вечерний волейбол</h2>
      <p>Тренировка · 19:00–21:00</p>
      <div className="lp-match-divider"/>
      <div className="lp-attending"><div className="lp-avatars" aria-hidden="true"><span>АМ</span><span>ДС</span><span>АВ</span><span>+9</span></div><div><strong>12 игроков</strong><span>готовы к игре</span></div><span className="lp-check"><Icon name="check"/></span></div>
      <div className="lp-match-progress"><i/></div>
      <div className="lp-match-foot"><span>Команда собрана</span><span>12 / 12</span></div>
    </div>
    <div className="lp-floating-message"><span className="lp-message-icon"><TelegramIcon/></span><div><strong>Встречаемся на площадке!</strong><span>Анонс и запись — в Telegram</span></div><Icon name="check" size={16}/></div>
    <div className="lp-visual-bottom"><span>МЕНЬШЕ РУТИНЫ</span><Icon name="arrow"/><span>БОЛЬШЕ КОМАНДЫ</span></div>
    <span className="lp-preview-note">Пример встречи в TeamTime</span>
  </div>
}

export default function LandingPage() {
  const [active, setActive] = useState(0)
  const [menuOpen, setMenuOpen] = useState(false)
  const screenDialog = useRef<HTMLDialogElement>(null)
  const feature = features[active]

  useEffect(() => {
    const title = document.title
    document.title = 'TeamTime — больше игры, меньше организации'
    document.documentElement.classList.add('lp-document')
    return () => { document.title = title; document.documentElement.classList.remove('lp-document') }
  }, [])

  function changeFeature(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    let next = index
    if (event.key === 'ArrowDown' || event.key === 'ArrowRight') next = (index + 1) % features.length
    else if (event.key === 'ArrowUp' || event.key === 'ArrowLeft') next = (index + features.length - 1) % features.length
    else if (event.key === 'Home') next = 0
    else if (event.key === 'End') next = features.length - 1
    else return
    event.preventDefault()
    setActive(next)
    document.getElementById(`lp-tab-${features[next].id}`)?.focus()
  }

  return <div className="lp">
    <a className="lp-skip" href="#main">К содержимому</a>
    <header className="lp-header"><div className="lp-container lp-header-inner">
      <Brand/>
      <button className="lp-menu-toggle" type="button" aria-label={menuOpen ? 'Закрыть меню' : 'Открыть меню'} aria-expanded={menuOpen} aria-controls="lp-nav" onClick={() => setMenuOpen(!menuOpen)}><Icon name={menuOpen ? 'close' : 'list'}/></button>
      <nav id="lp-nav" className={menuOpen ? 'lp-nav is-open' : 'lp-nav'} aria-label="Навигация по странице" onClick={() => setMenuOpen(false)}>
        <a href="#features">Возможности</a><a href="#how-it-works">Как это работает</a><a href="#questions">Вопросы</a>
      </nav>
      <a className="lp-header-login" href="/app">Войти <Icon name="arrow" size={18}/></a>
    </div></header>

    <main id="main">
      <section className="lp-container lp-hero" aria-labelledby="lp-title">
        <div className="lp-hero-copy"><p className="lp-eyebrow"><span/> ДЛЯ ТЕХ, КТО СОБИРАЕТ КОМАНДУ</p>
          <h1 id="lp-title">Больше игры.<br/><span>Меньше{' '}<br/>организации.</span></h1>
          <p className="lp-lead">Расписание, запись игроков и оплаты — в одном месте. А у вас остаётся время на то, ради чего все собрались.</p>
          <div className="lp-hero-actions"><a href="/app" className="lp-button lp-primary">Открыть TeamTime <Icon name="arrow" size={19}/></a><a className="lp-text-link" href="#features">Посмотреть возможности <Icon name="down" size={17}/></a></div>
          <p className="lp-hero-note"><TelegramIcon size={17}/> С интеграцией с Telegram</p>
        </div>
        <MatchPreview/>
      </section>

      <section className="lp-container lp-proof" aria-label="TeamTime уже используют 10 спортивных команд"><div className="lp-proof-number">10<span>спортивных<br/>команд</span></div><p>Уже используют<br/><strong>TeamTime</strong></p><div className="lp-proof-line"/><span>Любительские команды</span><i/><span>Тренеры и группы</span><i/><span>Регулярные игры</span></section>

      <section className="lp-features" id="features" aria-labelledby="lp-features-title"><div className="lp-container">
        <div className="lp-section-heading"><div><p className="lp-eyebrow">ОДНА КОМАНДА. ОДНО ПРОСТРАНСТВО.</p><h2 id="lp-features-title">Всё вокруг игры.<br/><span>Всё на своих местах.</span></h2></div><p>От первого «кто сегодня играет?»{' '}<br/>до последнего оплаченного взноса.{' '}<br/>Семь возможностей для вашей команды.</p></div>
        <div className="lp-product-tour">
          <div className="lp-feature-tabs" role="tablist" aria-label="Возможности TeamTime" aria-orientation="vertical">{features.map((item, index) => <button key={item.id} type="button" role="tab" id={`lp-tab-${item.id}`} aria-controls="lp-feature-panel" aria-selected={active === index} tabIndex={active === index ? 0 : -1} onClick={() => setActive(index)} onKeyDown={event => changeFeature(event,index)}><span className="lp-feature-number">0{index+1}</span><span>{item.title}</span><Icon name="arrow" size={17}/></button>)}</div>
          <div className="lp-feature-panel" id="lp-feature-panel" role="tabpanel" aria-labelledby={`lp-tab-${feature.id}`} tabIndex={0}>
            <div className="lp-feature-copy" key={feature.id}><div className="lp-feature-icon">{feature.icon === 'telegram' ? <TelegramIcon/> : <Icon name={feature.icon} size={23}/>}</div><h3>{feature.benefit}</h3><p>{feature.description}</p><div className="lp-tags">{feature.tags.map(tag=><span key={tag}><Icon name="check" size={14}/>{tag}</span>)}</div></div>
            <button className="lp-screen" type="button" onClick={() => screenDialog.current?.showModal()} aria-label={`Увеличить экран: ${feature.title}`}>
              <span className="lp-screen-bar"><span><i/><i/><i/></span><span>teamtime · {feature.title.toLowerCase()}</span><Icon name="diagonal" size={16}/></span>
              <img key={feature.image} src={`/landing/${feature.image}.png`} width="1360" height="920" loading="lazy" alt={feature.caption}/>
              <span className="lp-screen-hint"><Icon name="diagonal" size={14}/> Рассмотреть экран</span>
            </button>
            <p className="lp-screen-caption">Реальный интерфейс · данные тестовой команды</p>
          </div>
        </div>
      </div></section>

      <section className="lp-container lp-audience" aria-labelledby="lp-audience-title"><div><p className="lp-eyebrow">ВАША РОЛЬ — БЫТЬ С КОМАНДОЙ</p><h2 id="lp-audience-title">Вы организуете.<br/><span>TeamTime помогает.</span></h2></div><div className="lp-audience-item"><Icon name="ball" size={30}/><h3>Собираете любительскую команду?</h3><p>Планируйте игры, собирайте ответы и учитывайте взносы на аренду площадки.</p></div><div className="lp-audience-item"><Icon name="users" size={30}/><h3>Ведёте тренировочную группу?</h3><p>Соберите расписание, абонементы и посещаемость в одном рабочем пространстве.</p></div></section>

      <section className="lp-how" id="how-it-works" aria-labelledby="lp-how-title"><div className="lp-container"><div className="lp-section-heading"><div><p className="lp-eyebrow">ОТ ПЛАНА ДО ПЛОЩАДКИ</p><h2 id="lp-how-title">Привычный ритм.<br/><span>Понятный порядок.</span></h2></div><a href="/app" className="lp-text-link">Перейти в TeamTime <Icon name="arrow" size={18}/></a></div><ol className="lp-steps"><li><span>01</span><Icon name="calendar" size={26}/><h3>Запланируйте встречу</h3><p>Укажите время и детали игры или тренировки. Используйте шаблон для регулярных встреч.</p></li><li><span>02</span><TelegramIcon size={26}/><h3>Соберите участников</h3><p>Опубликуйте голосование в Telegram. Ответы игроков будут доступны в событии.</p></li><li><span>03</span><Icon name="check" size={26}/><h3>Играйте. Остальное учтено.</h3><p>Отметьте посещения, взносы и результаты. К следующей встрече всё останется под рукой.</p></li></ol></div></section>

      <section className="lp-container lp-faq" id="questions" aria-labelledby="lp-faq-title"><div><p className="lp-eyebrow">ПЕРЕД ПЕРВОЙ ИГРОЙ</p><h2 id="lp-faq-title">Есть вопросы?</h2><p>Вот что стоит знать о TeamTime.</p></div><div className="lp-faq-list">{[
        ['Кому подойдёт TeamTime?', 'Организаторам любительских спортивных команд и тренерам регулярных групп. Особенно если вы уже собираете участников в Telegram и отдельно ведёте расписание, взносы или абонементы.'],
        ['Нужно ли игрокам устанавливать новое приложение?', 'TeamTime открывается в браузере на телефоне и компьютере. Голосования и анонсы можно публиковать в привычном Telegram-чате через подключённого бота.'],
        ['Можно ли вести и разовые оплаты, и абонементы?', 'Да. В TeamTime есть учёт взносов за отдельные встречи и абонементы на занятия. Оплату отмечает организатор, а занятие списывается при отметке присутствия.'],
        ['TeamTime сам принимает платежи?', 'Пока нет: организатор отмечает полученную оплату вручную. Сервис помогает учитывать взносы, видеть задолженности и остаток занятий по абонементу.'],
        ['Можно посмотреть продукт до входа?', 'Да — выше доступны экраны тестовой команды. Выберите возможность и нажмите на изображение, чтобы рассмотреть интерфейс крупнее.'],
      ].map(([question, answer])=><details key={question}><summary>{question}<span><Icon name="plus" size={19}/></span></summary><p>{answer}</p></details>)}</div></section>

      <section className="lp-container lp-cta" aria-labelledby="lp-cta-title"><Court/><div><p className="lp-eyebrow">TEAMTIME · ВРЕМЯ БЫТЬ КОМАНДОЙ</p><h2 id="lp-cta-title">Следующая игра<br/>начинается здесь.</h2><p>Соберите всё, чем живёт ваша команда, в одном месте.</p><a href="/app" className="lp-button lp-white">Открыть TeamTime <Icon name="arrow" size={20}/></a></div><span className="lp-cta-ball"><Icon name="ball" size={140}/></span></section>
    </main>
    <footer className="lp-container lp-footer"><Brand/><span>Время быть командой.</span><a href="/app">Войти в приложение <Icon name="arrow" size={16}/></a><small>© {new Date().getFullYear()} TeamTime</small></footer>
    <dialog ref={screenDialog} className="lp-screen-dialog" aria-labelledby="lp-dialog-title" onClick={event => { if(event.target === event.currentTarget) screenDialog.current?.close() }}><div><header><div><strong id="lp-dialog-title">{feature.title}</strong><p>Данные тестовой команды</p></div><button type="button" autoFocus onClick={()=>screenDialog.current?.close()} aria-label="Закрыть экран"><Icon name="close" size={24}/></button></header><div className="lp-dialog-image"><img src={`/landing/${feature.image}.png`} alt={feature.caption} width="1360" height="920"/></div></div></dialog>
  </div>
}

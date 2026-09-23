import type { ReactNode } from 'react'
import { Icon } from '../../components/Icon'
import './login.css'

export function LoginPage({ checking, configured, widgetError, onRetry, children }: {
  checking: boolean; configured: boolean; widgetError: boolean; onRetry: () => void; children: ReactNode
}) {
  return <div className="login-page">
    <header className="login-header">
      <a className="login-brand" href="/" aria-label="TeamTime — на главную"><img src="/icon.png" width="46" height="46" alt=""/><span>teamtime<span>.</span></span></a>
      <span className="login-tagline">Время быть командой</span>
      <a className="login-header-link" href="#teamtime-login">Войти в команду<Icon name="diagonal" size={17}/></a>
    </header>

    <main className="login-main">
      <section className="login-hero" aria-labelledby="login-title">
        <div className="login-hero-copy">
          <p className="login-eyebrow"><span/>Для тех, кто собирает команду</p>
          <h1 id="login-title">Меньше переписок.<br/><span>Больше игры.</span></h1>
          <p className="login-lead">Собирайте игроков, распределяйте команды и следите за оплатами. TeamTime берёт организацию на себя — вы выходите на площадку.</p>
          <a className="login-hero-link" href="#teamtime-login">Ваша следующая тренировка начинается здесь<Icon name="arrow" size={19}/></a>
        </div>
        <div className="login-match" aria-label="Пример тренировки в TeamTime">
          <div className="login-match-heading"><span>Пример тренировки</span><span><Icon name="ball" size={15}/>Волейбол</span></div>
          <div className="login-match-body">
            <div className="login-match-copy"><span className="login-match-day">СРЕДА · 19:00</span><h2>Вечерний волейбол</h2><p>Команда собрана.<br/>Осталось сыграть.</p><span className="login-match-status"><Icon name="check" size={15}/>12 из 12 игроков</span></div>
            <div className="login-court" aria-hidden="true"><div className="login-court-lines"><i/><i/><i/><i/><i/><i/><b/><b/><b/><b/><b/><b/></div><span className="login-court-ball"><Icon name="ball" size={29}/></span></div>
          </div>
          <div className="login-match-footer"><span><Icon name="check" size={14}/>Опрос опубликован</span><span><Icon name="check" size={14}/>Составы готовы</span></div>
        </div>
      </section>

      <section className="login-access" id="teamtime-login" aria-labelledby="login-access-title" tabIndex={-1}>
        <div className="login-access-icon"><Icon name="users" size={26}/></div>
        <p className="login-eyebrow">Ваша команда уже ближе</p>
        <h2 id="login-access-title">Всё начинается<br/>с команды.</h2>
        <p className="login-access-description">Войдите через Telegram, чтобы открыть свои команды, события и оплаты.</p>
        <div className="login-auth" aria-busy={checking}>
          {checking ? <p className="login-auth-status" role="status">Проверяем вход…</p> : configured ? <>
            <div className="login-widget">{children}</div>
            {widgetError ? <div className="login-auth-error" role="status"><p>Не удалось загрузить вход через Telegram. Проверьте подключение и попробуйте ещё раз.</p><button type="button" onClick={onRetry}><Icon name="refresh" size={16}/>Повторить загрузку</button></div> : <p className="login-auth-note">Один аккаунт Telegram — все ваши команды.</p>}
          </> : <p className="login-auth-error" role="status">Вход через Telegram пока не настроен. Обратитесь к администратору сервиса.</p>}
        </div>
        <div className="login-access-divider"/>
        <ul className="login-access-benefits"><li><Icon name="check" size={17}/>Без отдельного логина и пароля</li><li><Icon name="check" size={17}/>Только команды, к которым у вас есть доступ</li></ul>
        <p className="login-access-hint">Используйте тот же Telegram, с которым участвуете в группе команды.</p>
      </section>

      <section className="login-features" aria-label="Возможности TeamTime">
        <article><span className="login-feature-icon"><Icon name="poll" size={22}/></span><div><h2>Соберите игроков</h2><p>Опросы и напоминания в Telegram. Сразу понятно, кто будет на тренировке.</p></div></article>
        <article><span className="login-feature-icon"><Icon name="users" size={22}/></span><div><h2>Подготовьте составы</h2><p>Распределение по командам с учётом уровня игроков — без споров у сетки.</p></div></article>
        <article><span className="login-feature-icon"><Icon name="wallet" size={22}/></span><div><h2>Держите оплаты в порядке</h2><p>Взносы, абонементы и напоминания должникам по каждой тренировке.</p></div></article>
      </section>
    </main>
    <footer className="login-footer"><span>TeamTime · Время быть командой.</span><span>От первого «кто идёт?» до последней оплаты.</span></footer>
  </div>
}

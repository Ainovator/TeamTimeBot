import { useEffect, useState, type ReactNode } from 'react'
import { Icon } from '../../components/Icon'
import './login.css'

export function LoginPage({ checking, configured, startupError = false, widgetError, onRetry, children }: {
  checking: boolean; configured: boolean; startupError?: boolean; widgetError: boolean; onRetry: () => void; children: ReactNode
}) {
  const [slow, setSlow] = useState(false)
  useEffect(() => {
    setSlow(false)
    if (!checking) return
    const timer = window.setTimeout(() => setSlow(true), 15000)
    return () => window.clearTimeout(timer)
  }, [checking])

  return <main className="login-page">
    <section className="login-card" aria-labelledby="login-title">
      <a className="login-brand" href="/" aria-label="TeamTime — на главную"><img src="/icon.png" width="48" height="48" alt=""/><span>teamtime<span>.</span></span></a>
      <h1 id="login-title">Время быть командой.</h1>
      <p className="login-description">Ваши тренировки, составы и оплаты — в одном месте.</p>
      <div className="login-auth" aria-busy={checking}>
        {startupError || (checking && slow) ? <div className="login-auth-error" role="status"><p>{startupError ? 'Не удалось связаться с сервером.' : 'Проверка входа занимает больше времени, чем обычно.'}</p><button type="button" onClick={() => window.location.reload()}>Обновить страницу</button></div> : checking ? <p className="login-auth-status" role="status">Проверяем вход…</p> : configured ? <>
          <div className="login-widget">{children}</div>
          {widgetError && <div className="login-auth-error" role="status"><p>Не удалось загрузить вход через Telegram.</p><button type="button" onClick={onRetry}><Icon name="refresh" size={16}/>Попробовать ещё раз</button></div>}
        </> : <p className="login-auth-error" role="status">Вход пока не настроен. Обратитесь к администратору.</p>}
      </div>
      <p className="login-hint">Войдите с Telegram-аккаунтом вашей команды.</p>
    </section>
  </main>
}

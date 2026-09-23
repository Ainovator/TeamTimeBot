// Isolated visual fixture: no external Telegram widget, authentication or API calls.
import React, { useState } from 'react'
import { createRoot } from 'react-dom/client'
import '../src/styles.css'
import '../src/studio-theme.css'
import '@fontsource-variable/golos-text'
import { LoginPage } from '../src/features/auth/LoginPage'
import { useProductTheme } from '../src/features/studio/ThemeSwitcher'

function Preview() {
  useProductTheme()
  const [state, setState] = useState('ready')
  const [signedIn, setSignedIn] = useState(false)
  return <>
    <LoginPage checking={state === 'loading'} configured={state !== 'unconfigured'} startupError={state === 'offline'} widgetError={state === 'error'} onRetry={() => setState('ready')}>
      {state !== 'error' && <button type="button" style={{ background:'#54a9eb', border:0, borderRadius:20, padding:'10px 18px', fontFamily:'Arial', fontSize:15, fontWeight:600 }} onClick={() => setSignedIn(true)}>Войти через Telegram</button>}
      {signedIn && <p role="status">Демо: кнопка входа работает. Запросы не отправлялись.</p>}
    </LoginPage>
    <details style={{position:'fixed',bottom:8,right:8,zIndex:6000,fontSize:11,background:'#fff',color:'#142b49',border:'1px solid #dce4ee',borderRadius:7,padding:'7px 10px'}}>
      <summary>Тестовый экран · без Telegram</summary>
      <label>Состояние входа <select value={state} onChange={event => setState(event.target.value)}><option value="ready">Готов к входу</option><option value="loading">Проверка авторизации</option><option value="error">Ошибка виджета</option><option value="offline">Сервер недоступен</option><option value="unconfigured">Вход не настроен</option></select></label>
    </details>
  </>
}
createRoot(document.getElementById('root')!).render(<React.StrictMode><Preview/></React.StrictMode>)

// Dev-only control gallery. No API requests or saved theme preferences.
import { useLayoutEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import '../src/styles.css'
import '../src/studio-theme.css'
import '../src/product-themes.css'
import '@fontsource-variable/golos-text'
import { Checkbox } from '../src/components/Checkbox'

function CheckboxGallery() {
  const [theme, setTheme] = useState('club')
  const [checked, setChecked] = useState(false)
  useLayoutEffect(() => { document.documentElement.dataset.theme = theme }, [theme])
  return <main style={{padding:32,height:'100%',overflow:'auto',maxWidth:620,margin:'auto'}}>
    <h1>Чекбоксы TeamTime</h1>
    <label style={{display:'grid',gap:8,marginBottom:24}}>Тема<select value={theme} onChange={event => setTheme(event.target.value)}><option value="club">Спортивный клуб</option><option value="arena">Ночная арена</option><option value="cherry">Вишня</option><option value="light">Светлая</option></select></label>
    <form style={{display:'grid',gap:16}}>
      <label className="toggle-field"><Checkbox checked={checked} onChange={event => setChecked(event.target.checked)}/><span>Проверка с клавиатуры</span></label>
      <label className="option-accounting"><Checkbox defaultChecked/><span>Выбран по умолчанию</span></label>
      <label className="toggle-field"><Checkbox ref={input => { if (input) input.indeterminate = true }}/><span>Частичный выбор</span></label>
      <label className="toggle-field"><Checkbox disabled/><span>Недоступен</span></label>
      <label className="toggle-field"><Checkbox checked disabled/><span>Выбран и недоступен</span></label>
      <label className="toggle-field"><Checkbox aria-invalid="true"/><span>Ошибка поля</span></label>
      <button type="reset">Сбросить форму</button>
    </form>
    <p role="status">{checked ? 'Выбор включён' : 'Выбор выключен'}</p>
  </main>
}

createRoot(document.getElementById('root')!).render(<CheckboxGallery/> )

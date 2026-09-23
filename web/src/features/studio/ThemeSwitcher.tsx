import { useLayoutEffect, useState } from 'react'
import { Icon } from '../../components/Icon'
import '../../product-themes.css'

type ProductTheme = 'arena' | 'club' | 'cherry' | 'light'
const storageKey = 'teamtime-theme'

function savedTheme(): ProductTheme {
  try {
    const stored = localStorage.getItem(storageKey)
    return stored === 'light' || stored === 'cherry' || stored === 'club' ? stored : 'arena'
  } catch { return 'arena' }
}

export function useProductTheme() {
  const [theme, setTheme] = useState<ProductTheme>(savedTheme)
  useLayoutEffect(() => { document.documentElement.dataset.theme = theme }, [theme])
  useLayoutEffect(() => {
    const sync = (event: StorageEvent) => {
      if (event.key === storageKey || event.key === null) setTheme(savedTheme())
    }
    window.addEventListener('storage', sync)
    return () => window.removeEventListener('storage', sync)
  }, [])

  function choose(next: ProductTheme) {
    setTheme(next)
    try { localStorage.setItem(storageKey, next) } catch { /* Switching still works without browser storage. */ }
  }

  return [theme, choose] as const
}

export function ThemeSwitcher({ theme, choose }: { theme: ProductTheme; choose: (theme: ProductTheme) => void }) {
  const options = [
    { id:'arena', label:'Ночная арена', description:'Глубокий синий и ледяной голубой' },
    { id:'club', label:'Спортивный клуб', description:'Белые карточки и тёмно-синее меню' },
    { id:'cherry', label:'Вишня', description:'Тёплый графит и вишнёвые акценты' },
    { id:'light', label:'Светлая', description:'Светлые поверхности и мягкий зелёный' },
  ] as const
  return <section className="content-card studio-appearance" aria-labelledby="appearance-title">
    <div className="studio-appearance-heading"><div><h3 id="appearance-title">Оформление</h3><p>Выберите тему. Изменения применяются сразу и сохраняются в этом браузере.</p></div><Icon name="sparkle" size={23}/></div>
    <div className="studio-theme-options" role="group" aria-label="Тема оформления">
      {options.map(option => <button className="studio-theme-option" data-theme-option={option.id} key={option.id} type="button"
        aria-label={`Тема «${option.label}»`} aria-pressed={theme === option.id} onClick={() => choose(option.id)}>
        <span className="studio-theme-preview" aria-hidden="true"><span className="studio-preview-sidebar"><i/><i/><i/></span><span className="studio-preview-content"><i/><span><b/><b/><b/></span><strong/></span></span>
        <span className="studio-theme-option-title"><strong>{option.label}</strong><span className="studio-theme-selected" aria-hidden="true">{theme === option.id ? <Icon name="check" size={16}/> : null}</span></span>
        <span className="studio-theme-description">{option.description}</span>
      </button>)}
    </div>
  </section>
}

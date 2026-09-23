import React, { lazy, Suspense } from 'react'
import { createRoot } from 'react-dom/client'

const isDesignLab = /^\/design-lab\/?$/.test(window.location.pathname)
const Page = lazy(() => isDesignLab
  ? import('./features/design-lab/DesignLab')
  : import('./styles.css').then(() => import('./studio-theme.css')).then(() => import('@fontsource-variable/golos-text')).then(() => import('./App')))

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <Suspense fallback={<div role="status" style={{ padding: 32, fontFamily: 'sans-serif' }}>Загружаем TeamTime…</div>}>
      <Page />
    </Suspense>
  </React.StrictMode>,
)

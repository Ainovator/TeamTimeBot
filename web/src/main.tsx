import React, { lazy, Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import { AppBoundary, StartupScreen } from './components/AppBoundary'

const isDesignLab = /^\/design-lab\/?$/.test(window.location.pathname)
const isLanding = /^\/(?:landing\/?)?$/.test(window.location.pathname)
const Page = lazy(() => isDesignLab
  ? import('./features/design-lab/DesignLab')
  : isLanding ? import('./features/landing/LandingPage') : import('./ProductApp'))

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AppBoundary><Suspense fallback={<StartupScreen/>}>
      <Page />
    </Suspense></AppBoundary>
  </React.StrictMode>,
)

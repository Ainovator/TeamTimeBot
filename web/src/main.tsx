import React, { lazy, Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import { AppBoundary, StartupScreen } from './components/AppBoundary'

const isDesignLab = /^\/design-lab\/?$/.test(window.location.pathname)
const isLanding = /^\/(?:landing\/?)?$/.test(window.location.pathname)
// Separate factories let Vite preload the CSS belonging to each page.
const DesignLab = lazy(() => import('./features/design-lab/DesignLab'))
const LandingPage = lazy(() => import('./features/landing/LandingPage'))
const ProductApp = lazy(() => import('./ProductApp'))
const Page = isDesignLab ? DesignLab : isLanding ? LandingPage : ProductApp

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AppBoundary><Suspense fallback={<StartupScreen/>}>
      <Page />
    </Suspense></AppBoundary>
  </React.StrictMode>,
)

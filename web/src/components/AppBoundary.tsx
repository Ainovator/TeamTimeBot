import { Component, useEffect, useState, type ReactNode } from 'react'

export function StartupScreen({ failed = false }: { failed?: boolean }) {
  const [slow, setSlow] = useState(false)
  useEffect(() => {
    const timer = window.setTimeout(() => setSlow(true), 15000)
    return () => window.clearTimeout(timer)
  }, [])
  return <main className="tt-startup"><section>
    <h1>TeamTime</h1>
    <p role="status">{failed ? 'Не удалось открыть приложение.' : slow ? 'Загрузка занимает больше времени, чем обычно.' : 'Загружаем TeamTime…'}</p>
    {(failed || slow) && <><p>Проверьте подключение и обновите страницу.</p><button type="button" onClick={() => window.location.reload()}>Обновить страницу</button></>}
  </section></main>
}

export class AppBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() { return { failed: true } }

  componentDidCatch(error: Error) { console.error('TeamTime failed to load', error) }

  render() { return this.state.failed ? <StartupScreen failed/> : this.props.children }
}

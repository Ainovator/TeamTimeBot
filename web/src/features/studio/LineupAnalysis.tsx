import { useId, useState, type ReactNode } from 'react'
import { Icon } from '../../components/Icon'

export function LineupAnalysis({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(true)
  const contentID = useId()

  return <section className="studio-lineup-analysis" data-open={open}>
    <h5 className="team-analysis-title">
      <button type="button" className="studio-lineup-analysis-trigger"
        aria-expanded={open} aria-controls={contentID}
        onClick={() => setOpen(value => !value)}>
        <span>Аналитика расстановки</span>
        <span className="studio-lineup-analysis-toggle" aria-hidden="true">
          <span>{open ? 'Скрыть' : 'Показать'}</span>
          <Icon name="down" size={18}/>
        </span>
      </button>
    </h5>
    <div id={contentID} className="studio-lineup-analysis-body" aria-hidden={!open}>
      <div className="studio-lineup-analysis-content">{children}</div>
    </div>
  </section>
}

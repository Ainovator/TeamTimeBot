import { Icon } from '../../components/Icon'
import type { Section } from '../../app/router'

export function isTemplateSection(section: Section) {
  return section === 'event_templates' || section === 'templates'
}

export function OrganizationTemplates({ openEvents, openPolls }: { openEvents?: () => void; openPolls?: () => void }) {
  if (!openEvents && !openPolls) return null
  return <section className="studio-org-section" aria-labelledby="organization-templates-title">
    <header className="studio-org-section-heading">
      <Icon name="template" size={20}/><h2 id="organization-templates-title">Шаблоны</h2>
      <p>Заготовки встреч и опросов для вашей команды.</p>
    </header>
    <div className="studio-org-template-links">
      {openEvents && <button type="button" onClick={openEvents}><Icon name="calendar" size={22}/><span><strong>Шаблоны событий</strong><small>Расписание, стоимость и правила публикации</small></span><Icon name="arrow" size={18}/></button>}
      {openPolls && <button type="button" onClick={openPolls}><Icon name="poll" size={22}/><span><strong>Шаблоны голосований</strong><small>Вопросы, варианты ответов и учёт участников</small></span><Icon name="arrow" size={18}/></button>}
    </div>
  </section>
}

export function OrganizationTemplateNavigation({ active, available, open }: { active: Section; available: readonly Section[]; open: (section: Section) => void }) {
  return <div className="studio-template-navigation">
    <div className="studio-template-back"><button type="button" className="studio-text-button" onClick={() => open('settings')}><Icon name="back" size={18}/>Организация</button></div>
    <nav className="studio-event-tabs" aria-label="Шаблоны организации">
      {(['event_templates', 'templates'] as const).filter(section => available.includes(section)).map(section =>
        <button key={section} type="button" aria-current={active === section ? 'page' : undefined} className={active === section ? 'active' : ''} onClick={() => open(section)}>{section === 'event_templates' ? 'События' : 'Голосования'}</button>)}
    </nav>
  </div>
}

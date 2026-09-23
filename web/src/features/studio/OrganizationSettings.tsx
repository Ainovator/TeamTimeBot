import { useRef, useState } from 'react'
import { Icon } from '../../components/Icon'
import type { GroupDetails, GroupMember } from '../../types'
import './organizationSettings.css'
import { OrganizationTemplates } from './OrganizationTemplates'

export function OrganizationSettings({ group, members, roleTitle, canPublish, publishRegistration, openMembers, openDocumentation, openEventTemplates, openPollTemplates }: {
  group: GroupDetails['group']
  members: GroupMember[] | null
  roleTitle: string
  canPublish: boolean
  publishRegistration: () => Promise<void>
  openMembers?: () => void
  openDocumentation: () => void
  openEventTemplates?: () => void
  openPollTemplates?: () => void
}) {
  const [publishing, setPublishing] = useState(false)
  const pending = useRef(false)

  async function publish() {
    if (pending.current) return
    pending.current = true
    setPublishing(true)
    try {
      await publishRegistration()
    } finally {
      pending.current = false
      setPublishing(false)
    }
  }

  return <div className="studio-org-settings">
    <section className="studio-org-section" aria-labelledby="organization-details-title">
      <header className="studio-org-section-heading">
        <Icon name="organization" size={20}/>
        <h2 id="organization-details-title">Основные сведения</h2>
        <p>Общие данные команды и время, по которому она работает.</p>
      </header>
      <div className="studio-org-section-body">
        <dl className="studio-org-details">
          <div>
            <dt>Название</dt>
            <dd><strong>{group.title}</strong><span>Название вашей группы в Telegram.</span></dd>
          </div>
          <div>
            <dt>Часовой пояс</dt>
            <dd><strong>{group.timezone || 'Не указан'}</strong><span>Определяет время встреч, голосований и напоминаний.</span></dd>
          </div>
        </dl>
        <p className="studio-org-note">Эти данные задаются при подключении Telegram-группы.</p>
      </div>
    </section>
    <section className="studio-org-section" aria-labelledby="organization-access-title">
      <header className="studio-org-section-heading">
        <Icon name="users" size={20}/>
        <h2 id="organization-access-title">Участники и доступ</h2>
        <p>Состав команды и ваша роль в организации.</p>
      </header>
      <div className="studio-org-section-body">
        <dl className="studio-org-access">
          <div><dt>Участников</dt><dd>{members === null ? '—' : members.length}</dd></div>
          <div><dt>Администраторов</dt><dd>{members === null ? '—' : members.filter(member => member.role === 'admin').length}</dd></div>
          <div><dt>Ваша роль</dt><dd className="studio-org-role">{roleTitle}</dd></div>
        </dl>
        {members === null && <p className="studio-org-note">Сведения об участниках пока недоступны.</p>}
        {openMembers && <button type="button" className="btn-secondary studio-org-members-link" onClick={openMembers}>Открыть список игроков<Icon name="arrow" size={16}/></button>}
      </div>
    </section>
    <section className="studio-org-section" aria-labelledby="organization-registration-title">
      <header className="studio-org-section-heading">
        <Icon name="poll" size={20}/>
        <h2 id="organization-registration-title">Регистрация игроков</h2>
        <p>Добавление участников через опрос в Telegram.</p>
      </header>
      <div className="studio-org-section-body">
        <div className="studio-org-destination"><Icon name="users" size={18}/><div><span>Группа для публикации</span><strong>{group.title}</strong></div></div>
        <p className="studio-org-registration-copy">Опубликуйте регистрационный опрос, чтобы игроки могли присоединиться к команде.</p>
        <div className="studio-org-actions">
          <button type="button" disabled={publishing || !canPublish} aria-busy={publishing} onClick={() => void publish()}>
            <Icon name="plus" size={17}/>{publishing ? 'Публикуем…' : 'Опубликовать регистрацию'}
          </button>
          <button type="button" className="studio-text-button" onClick={openDocumentation}>Как работает регистрация<Icon name="arrow" size={16}/></button>
        </div>
        {!canPublish && <p className="studio-org-note">Для публикации требуется право управления шаблонами.</p>}
      </div>
    </section>
    <OrganizationTemplates openEvents={openEventTemplates} openPolls={openPollTemplates}/>
  </div>
}

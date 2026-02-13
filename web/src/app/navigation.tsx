import type { ReactNode } from 'react'
import type { Section } from './router'

export type SectionItem = {
  id: Section
  title: string
  subtitle: string
  icon: JSX.Element
}

function Icon({ children }: { children: ReactNode }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
      {children}
    </svg>
  )
}

export const sections: SectionItem[] = [
  {
    id: 'overview',
    title: 'Обзор',
    subtitle: 'Ключевые метрики',
    icon: (
      <Icon>
        <path d="M4 13h7V4H4v9Zm9 7h7V4h-7v16Zm-9 0h7v-5H4v5Z" fill="currentColor" />
      </Icon>
    ),
  },
  {
    id: 'members',
    title: 'Игроки',
    subtitle: 'Участники группы',
    icon: (
      <Icon>
        <path
          d="M16 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-8 2a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm0 2c-3.3 0-6 1.7-6 4v2h12v-2c0-2.3-2.7-4-6-4Zm8-2c-.7 0-1.3.1-1.9.3 1.2.9 1.9 2.1 1.9 3.7v2h8v-2c0-2.3-2.7-4-6-4Z"
          fill="currentColor"
        />
      </Icon>
    ),
  },
  {
    id: 'templates',
    title: 'Шаблоны опросов',
    subtitle: 'Справочник шаблонов',
    icon: (
      <Icon>
        <path d="M5 3h11l5 5v13a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Zm10 1v5h5" stroke="currentColor" strokeWidth="1.8" />
      </Icon>
    ),
  },
  {
    id: 'polls',
    title: 'Голосования',
    subtitle: 'Все опубликованные',
    icon: (
      <Icon>
        <path d="M4 4h16a2 2 0 0 1 2 2v4H2V6a2 2 0 0 1 2-2Zm-2 8h9v8H4a2 2 0 0 1-2-2v-6Zm11 0h9v6a2 2 0 0 1-2 2h-7v-8Z" fill="currentColor" />
      </Icon>
    ),
  },
  {
    id: 'event_templates',
    title: 'Шаблоны событий',
    subtitle: 'Настройки и расписание',
    icon: (
      <Icon>
        <path d="M12 2 3 7v10l9 5 9-5V7l-9-5Zm0 2.2L18.8 8 12 11.8 5.2 8 12 4.2Zm-7 5.5 6 3.3v6.6l-6-3.3V9.7Zm8 9.9V13l6-3.3v6.6l-6 3.3Z" fill="currentColor" />
      </Icon>
    ),
  },
  {
    id: 'events',
    title: 'События',
    subtitle: 'Конкретные занятия',
    icon: (
      <Icon>
        <path d="M6 2v2H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2h-2V2h-2v2H8V2H6Zm14 8H4v10h16V10Z" fill="currentColor" />
      </Icon>
    ),
  },
]

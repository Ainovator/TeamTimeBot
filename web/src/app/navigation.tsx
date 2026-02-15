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
    id: 'profile',
    title: 'Мой профиль',
    subtitle: 'Мои данные',
    icon: (
      <Icon>
        <path
          d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm0 2c-4.4 0-8 2.2-8 5v3h16v-3c0-2.8-3.6-5-8-5Z"
          fill="currentColor"
        />
      </Icon>
    ),
  },
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
    title: 'Шаблоны голосований',
    subtitle: 'Справочник шаблонов',
    icon: (
      <Icon>
        <path d="M5 3h11l5 5v13a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Zm10 1v5h5" stroke="currentColor" strokeWidth="1.8" />
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
  {
    id: 'games',
    title: 'Игры',
    subtitle: 'Счёт и составы',
    icon: (
      <Icon>
        <path
          d="M4 7a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7Zm2 0v10h12V7H6Zm1.8 2.2h2.6v2.6H7.8V9.2Zm5.8 0h2.6v2.6h-2.6V9.2Zm-6 5h8v1.8h-8v-1.8Z"
          fill="currentColor"
        />
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
    id: 'billing',
    title: 'Задолженности',
    subtitle: 'Кто должен деньги',
    icon: (
      <Icon>
        <path
          d="M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Zm1 5v1.2c1.4.2 2.4 1 2.8 2.2l-1.7.6c-.2-.7-.8-1.2-1.6-1.3v3c2 .6 3.1 1.5 3.1 3.1 0 1.6-1.1 2.7-2.6 3V19h-2v-1.2c-1.7-.2-2.9-1.2-3.3-2.8l1.8-.5c.2.9.8 1.5 1.5 1.7v-3.2c-2-.6-3-1.4-3-3 0-1.6 1.1-2.7 3-3V7h2Zm-2 2.9c-.7.1-1.1.6-1.1 1.2 0 .6.3.9 1.1 1.2V9.9Zm2 6.3c.7-.1 1.1-.6 1.1-1.3 0-.7-.3-1-1.1-1.3v2.6Z"
          fill="currentColor"
        />
      </Icon>
    ),
  },
  {
    id: 'docs',
    title: 'Документация',
    subtitle: 'Полный workflow',
    icon: (
      <Icon>
        <path
          d="M7 3h10a2 2 0 0 1 2 2v16a1 1 0 0 1-1 1H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Zm0 2v15h10V5H7Zm2 3h6v2H9V8Zm0 4h6v2H9v-2Zm0 4h6v2H9v-2Z"
          fill="currentColor"
        />
      </Icon>
    ),
  },
]

export type Concept = 'studio' | 'arena' | 'club'
import type { Section } from '../../app/router'
export type Screen = Exclude<Section, 'docs'>

export const concepts: { id: Concept; name: string; tag: string; description: string; colors: string[] }[] = [
  { id: 'studio', name: 'Studio', tag: 'Studio / новая итерация', description: 'Больше воздуха, новый шрифт и компактная навигация. Спокойный характер первого варианта.', colors: ['#f5f6f4', '#23634f', '#dcebe2', '#283631'] },
  { id: 'arena', name: 'Arena', tag: 'Энергия каждой игры', description: 'Спортивный характер. Крупная типографика, графит и яркий лайм. Игра в центре внимания.', colors: ['#191d1b', '#d5fa47', '#707970', '#f0f1e9'] },
  { id: 'club', name: 'Club', tag: 'Люди, которых объединяет игра', description: 'Тёплое клубное пространство. Молочные оттенки, терракота и внимание к сообществу.', colors: ['#faf6ed', '#ac4e36', '#ebceba', '#424b36'] },
]

export const navigation: { id: Screen; label: string; icon: string }[] = [
  { id: 'overview', label: 'Обзор', icon: 'grid' },
  { id: 'events', label: 'События', icon: 'calendar' },
  { id: 'members', label: 'Игроки', icon: 'users' },
  { id: 'games', label: 'Игры', icon: 'score' },
  { id: 'polls', label: 'Голосования', icon: 'poll' },
  { id: 'billing', label: 'Задолженности', icon: 'wallet' },
  { id: 'templates', label: 'Шаблоны голосований', icon: 'template' },
  { id: 'event_templates', label: 'Шаблоны событий', icon: 'repeat' },
  { id: 'profile', label: 'Мой профиль', icon: 'profile' },
]

export const players = [
  { name: 'Алексей Морозов', initials: 'АМ', role: 'Связующий', skill: 8.4, color: 'sage', paid: true },
  { name: 'Мария Волкова', initials: 'МВ', role: 'Доигровщик', skill: 8.1, color: 'rose', paid: true },
  { name: 'Дмитрий Соколов', initials: 'ДС', role: 'Центральный', skill: 7.8, color: 'sand', paid: false },
  { name: 'Анна Лебедева', initials: 'АЛ', role: 'Либеро', skill: 8.6, color: 'lilac', paid: true },
  { name: 'Иван Петров', initials: 'ИП', role: 'Доигровщик', skill: 7.5, color: 'blue', paid: false },
  { name: 'Елена Кузнецова', initials: 'ЕК', role: 'Связующий', skill: 8.2, color: 'rose', paid: true },
]

export const initialEvents = [
  { id: 1, title: 'Вечерний волейбол', day: '14', weekday: 'ПН', date: '14 сентября', time: '19:00–21:00', place: 'Спортзал «Динамо»', count: 16, limit: 18, price: 600, status: 'Открыта запись' },
  { id: 2, title: 'Играем в среду', day: '16', weekday: 'СР', date: '16 сентября', time: '20:00–22:00', place: 'Спортзал «Динамо»', count: 12, limit: 18, price: 600, status: 'Открыта запись' },
  { id: 3, title: 'Субботняя тренировка', day: '19', weekday: 'СБ', date: '19 сентября', time: '11:00–13:00', place: 'Спортцентр «Старт»', count: 18, limit: 18, price: 750, status: 'Мест нет' },
]
export type DemoEvent = typeof initialEvents[number]

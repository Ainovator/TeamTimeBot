import { players } from './demoData'

const details = [
  { number: '07', games: 36, attendance: 92, ready: true, title: 'Задаёт ритм игры', strength: 'Точный пас', clubRole: 'Организатор', skills: [7.8, 8.2, 7.5, 7.4, 9.3], joined: 'Марта 2025' },
  { number: '12', games: 28, attendance: 88, ready: true, title: 'Доводит атаку до точки', strength: 'Сильная атака', clubRole: 'Игрок', skills: [8.3, 8.1, 9.0, 7.5, 6.8], joined: 'Мая 2025' },
  { number: '18', games: 24, attendance: 83, ready: true, title: 'Держит высоту у сетки', strength: 'Уверенный блок', clubRole: 'Игрок', skills: [7.5, 6.9, 8.4, 9.1, 6.5], joined: 'Июня 2025' },
  { number: '03', games: 41, attendance: 96, ready: true, title: 'Спасает сложные мячи', strength: 'Надёжный приём', clubRole: 'Капитан', skills: [7.2, 9.5, 7.0, 6.3, 8.8], joined: 'Февраля 2025' },
  { number: '09', games: 19, attendance: 79, ready: false, title: 'Добавляет игре энергии', strength: 'Сильная подача', clubRole: 'Игрок', skills: [8.8, 7.3, 8.0, 7.1, 6.6], joined: 'Января 2026' },
  { number: '06', games: 32, attendance: 90, ready: true, title: 'Видит площадку целиком', strength: 'Читает игру', clubRole: 'Тренер', skills: [7.6, 8.6, 7.2, 7.0, 9.0], joined: 'Апреля 2025' },
]

export const roster = players.map((player, index) => ({ ...player, ...details[index], id: index }))
export type RosterPlayer = typeof roster[number]
export const skillLabels = ['Подача', 'Приём', 'Атака', 'Блок', 'Пас']
export const rosterRoles = [
  { name: 'Связующий', plural: 'Связующие', color: 'sage' },
  { name: 'Доигровщик', plural: 'Доигровщики', color: 'rose' },
  { name: 'Центральный', plural: 'Центральные', color: 'sand' },
  { name: 'Либеро', plural: 'Либеро', color: 'lilac' },
]

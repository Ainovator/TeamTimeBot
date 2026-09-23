export type PollTemplate = { id: number; name: string; question: string; options: { text: string; counted: boolean; weight: number }[] }
export const initialPollTemplates: PollTemplate[] = [
  { id: 1, name: 'Запись на волейбол', question: 'Кто играет в понедельник?', options: [{ text: 'Играю', counted: true, weight: 1 }, { text: 'Буду с другом', counted: true, weight: 2 }, { text: 'В этот раз пропущу', counted: false, weight: 0 }] },
  { id: 2, name: 'Субботняя тренировка', question: 'Встречаемся на тренировке?', options: [{ text: 'Буду', counted: true, weight: 1 }, { text: 'Пропущу', counted: false, weight: 0 }] },
  { id: 3, name: 'Клубная встреча', question: 'Соберёмся после игры?', options: [{ text: 'Да, присоединюсь', counted: true, weight: 1 }, { text: 'Пока не знаю', counted: false, weight: 0 }, { text: 'Не смогу', counted: false, weight: 0 }] },
]
export type EventTemplate = {
  id: number; name: string; type: string; weekday: string; start: string; end: string; pollDay: string; pollTime: string;
  templateID: number; price: number; limit: number; teamSize: number; minVotes: number; cancelMinutes: number;
  active: boolean; publish: boolean; autoTeams: boolean; publishTeams: boolean; settlement: boolean;
  before: boolean; after: boolean; cancelNotify: boolean; announcement: boolean; leadMinutes: number; text: string;
}
export const initialEventTemplates: EventTemplate[] = [
  { id: 1, name: 'Вечерний волейбол', type: 'Тренировка', weekday: 'Понедельник', start: '19:00', end: '21:00', pollDay: 'Воскресенье', pollTime: '12:00', templateID: 1, price: 10800, limit: 18, teamSize: 6, minVotes: 12, cancelMinutes: 120, active: true, publish: true, autoTeams: true, publishTeams: true, settlement: true, before: false, after: true, cancelNotify: true, announcement: true, leadMinutes: 60, text: 'Встречаемся в «Динамо»! Возьмите сменную обувь и воду.' },
  { id: 2, name: 'Играем в среду', type: 'Тренировка', weekday: 'Среда', start: '20:00', end: '22:00', pollDay: 'Вторник', pollTime: '12:00', templateID: 1, price: 10800, limit: 18, teamSize: 6, minVotes: 12, cancelMinutes: 120, active: true, publish: true, autoTeams: true, publishTeams: false, settlement: true, before: false, after: true, cancelNotify: true, announcement: false, leadMinutes: 60, text: '' },
  { id: 3, name: 'Субботняя тренировка', type: 'Тренировка', weekday: 'Суббота', start: '11:00', end: '13:00', pollDay: 'Пятница', pollTime: '10:00', templateID: 2, price: 13500, limit: 18, teamSize: 6, minVotes: 6, cancelMinutes: 180, active: false, publish: false, autoTeams: false, publishTeams: false, settlement: true, before: false, after: false, cancelNotify: true, announcement: false, leadMinutes: 60, text: '' },
]
export const demoPolls = [
  { id: 101, name: 'Вечерний волейбол', date: '14 сентября', published: '13 сентября · 12:00', question: 'Кто играет в понедельник?', template: 'Запись на волейбол', active: true, counts: [14, 1, 3], options: ['Играю', 'Буду с другом', 'В этот раз пропущу'] },
  { id: 102, name: 'Играем в среду', date: '16 сентября', published: '14 сентября · 10:00', question: 'Кто играет в среду?', template: 'Запись на волейбол', active: true, counts: [10, 1, 2], options: ['Играю', 'Буду с другом', 'В этот раз пропущу'] },
  { id: 103, name: 'Субботняя тренировка', date: '12 сентября', published: '11 сентября · 10:00', question: 'Встречаемся на тренировке?', template: 'Субботняя тренировка', active: false, counts: [18, 0, 4], options: ['Буду', 'Буду с другом', 'Пропущу'] },
]
export const demoGames = [
  { id: 1, event: 'Субботняя тренировка', date: '12 сентября', ordinal: 1, a: 25, b: 21, teamA: 'Орбита', teamB: 'Импульс' },
  { id: 2, event: 'Субботняя тренировка', date: '12 сентября', ordinal: 2, a: 22, b: 25, teamA: 'Орбита', teamB: 'Импульс' },
  { id: 3, event: 'Субботняя тренировка', date: '12 сентября', ordinal: 3, a: 25, b: 18, teamA: 'Орбита', teamB: 'Импульс' },
  { id: 4, event: 'Играем в среду', date: '9 сентября', ordinal: 1, a: 25, b: 23, teamA: 'Импульс', teamB: 'Орбита' },
  { id: 5, event: 'Играем в среду', date: '9 сентября', ordinal: 2, a: 19, b: 25, teamA: 'Импульс', teamB: 'Орбита' },
]
export const initialDebts = [
  { id: 1, player: 2, event: 'Вечерний волейбол', date: '7 сентября', amount: 600, paid: false },
  { id: 2, player: 2, event: 'Играем в среду', date: '9 сентября', amount: 600, paid: false },
  { id: 3, player: 4, event: 'Субботняя тренировка', date: '12 сентября', amount: 750, paid: false },
]

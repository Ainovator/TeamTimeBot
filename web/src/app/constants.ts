export const weekdayOptions = [
  { value: 1, short: 'Пн', full: 'Понедельник' },
  { value: 2, short: 'Вт', full: 'Вторник' },
  { value: 3, short: 'Ср', full: 'Среда' },
  { value: 4, short: 'Чт', full: 'Четверг' },
  { value: 5, short: 'Пт', full: 'Пятница' },
  { value: 6, short: 'Сб', full: 'Суббота' },
  { value: 7, short: 'Вс', full: 'Воскресенье' },
]

export const announcementLeadOptions = [
  { value: 60, label: 'За 1 час' },
  { value: 120, label: 'За 2 часа' },
  { value: 1440, label: 'За 1 день' },
]

export function weekdayLabel(weekday: number): string {
  return weekdayOptions.find((day) => day.value === weekday)?.full ?? `День ${weekday}`
}

export function announcementLeadLabel(value: number): string {
  return announcementLeadOptions.find((item) => item.value === value)?.label ?? `${value} мин`
}

export function toHourMinute(value: string): string {
  return value.slice(0, 5)
}

export function formatMoney(value?: number | null): string {
  if (value === undefined || value === null) {
    return 'не указана'
  }
  return `${value.toFixed(2)} ₽`
}

export function ensureList<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

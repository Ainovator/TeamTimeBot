import type { GroupMember, MemberSkillProfile, SkillCatalogItem } from '../../types'

export function fullName(member: GroupMember): string {
  const value = `${member.firstName ?? ''} ${member.lastName ?? ''}`.trim()
  if (value.length > 0) {
    return value
  }
  if (member.username) {
    return `@${member.username}`
  }
  return `ID ${member.userTelegramID}`
}

export function fullNamePartsFromProfile(profile: MemberSkillProfile): { lastName: string; firstName: string; patronymic: string } {
  return {
    lastName: profile.lastName?.trim() || 'Не указано',
    firstName: profile.firstName?.trim() || 'Не указано',
    patronymic: 'Не указано',
  }
}

export function initialsFromProfile(profile: MemberSkillProfile): string {
  const last = profile.lastName?.trim()
  const first = profile.firstName?.trim()
  if (last && first) {
    return `${last[0]}${first[0]}`.toUpperCase()
  }
  if (first) {
    return first.slice(0, 2).toUpperCase()
  }
  if (profile.username) {
    return profile.username.slice(0, 2).toUpperCase()
  }
  return 'TT'
}

export function normalizedSkillScore(score?: number): number {
  if (typeof score !== 'number' || Number.isNaN(score)) {
    return 5
  }
  return Math.max(1, Math.min(10, score))
}

export function profileSkills(profile: MemberSkillProfile | null, catalog: SkillCatalogItem[]): Array<{ skillCode: string; skillName: string; score?: number }> {
  if (profile && profile.skills.length > 0) {
    return profile.skills
  }
  return catalog.map((skill) => ({
    skillCode: skill.code,
    skillName: skill.name,
  }))
}

export function averageScore(profile: MemberSkillProfile | null, catalog: SkillCatalogItem[]): number | null {
  const skills = profileSkills(profile, catalog)
  if (skills.length === 0) {
    return null
  }
  const values = skills.map((skill) => normalizedSkillScore(skill.score))
  return values.reduce((acc, item) => acc + item, 0) / values.length
}

export function scoreColor(score: number | null): string {
  if (score === null) {
    return '#7c8fb1'
  }
  const bounded = Math.max(1, Math.min(10, score))
  const hue = ((bounded - 1) / 9) * 120
  return `hsl(${hue} 70% 52%)`
}

export function playerTypeLabel(value: '' | 'attacker' | 'setter' | 'libero' | 'central'): string {
  if (value === 'attacker') {
    return 'Атакующий'
  }
  if (value === 'setter') {
    return 'Пасующий'
  }
  if (value === 'libero') {
    return 'Либеро'
  }
  if (value === 'central') {
    return 'Центральный'
  }
  return 'Не выбран'
}

export const playerTypeCards: Array<{
  value: 'attacker' | 'setter' | 'libero' | 'central'
  title: string
  subtitle: string
  image: string
}> = [
  { value: 'attacker', title: 'Атакующий', subtitle: 'Сильная атака и завершение', image: '🏐' },
  { value: 'setter', title: 'Пасующий', subtitle: 'Розыгрыш и точный пас', image: '🎯' },
  { value: 'libero', title: 'Либеро', subtitle: 'Прием и защита', image: '🛡️' },
  { value: 'central', title: 'Центральный', subtitle: 'Блок и игра у сетки', image: '🧱' },
]

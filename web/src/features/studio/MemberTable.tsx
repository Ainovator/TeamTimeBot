import { fullName, memberRating, normalizedSkillScore, playerTypeLabel, profileSkills } from '../members/memberUtils'
import type { GroupMember, MemberSkillProfile, SkillCatalogItem } from '../../types'
import { MemberRating } from './MemberRating'
import './memberTable.css'

export const MEMBERS_SORT_AVERAGE_CODE = '__avg__'
export type MembersSortCriterion = { code: string; direction: 'asc' | 'desc' }

export function MemberTable({ members, profiles, catalog, sortCriteria, onSort, open }: {
  members: GroupMember[]
  profiles: Record<number, MemberSkillProfile>
  catalog: SkillCatalogItem[]
  sortCriteria: MembersSortCriterion[]
  onSort: (code: string, direction: 'asc' | 'desc') => void
  open: (member: GroupMember) => void
}) {
  function sortHeader(code: string, label: string) {
    const priority = sortCriteria.findIndex(criterion => criterion.code === code)
    const direction = sortCriteria[priority]?.direction
    return <th key={code} scope="col" className="studio-member-number" aria-sort={priority === 0 ? direction === 'desc' ? 'descending' : 'ascending' : undefined}>
      <div className="studio-member-sort">
        <span className="studio-member-sort-label" title={label}>{label}</span>
        <div className="studio-member-sort-arrows">
          {(['asc', 'desc'] as const).map(order => <button key={order} type="button" className="studio-member-sort-arrow"
            data-active={direction === order || undefined} aria-pressed={direction === order}
            aria-label={`${label}: по ${order === 'asc' ? 'возрастанию' : 'убыванию'}`}
            title={direction === order ? 'Отменить сортировку' : `По ${order === 'asc' ? 'возрастанию' : 'убыванию'}`}
            onClick={() => onSort(code, order)}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d={order === 'asc' ? 'M12 20V4m-5 5 5-5 5 5' : 'M12 4v16m-5-5 5 5 5-5'}/>
            </svg>
          </button>)}
        </div>
      </div>
    </th>
  }

  return <div className="table-wrap studio-members-table-wrap" tabIndex={0} role="region" aria-label="Список игроков и оценки">
    <table className="table studio-members-table" style={{ minWidth: 590 + catalog.length * 78 }}>
      <caption className="studio-visually-hidden">Игроки и оценки навыков. Стрелка вверх сортирует по возрастанию, вниз — по убыванию. Повторное нажатие на активную стрелку отменяет сортировку по этому показателю.</caption>
      <colgroup>
        <col style={{ width: 190 }}/><col style={{ width: 190 }}/><col style={{ width: 120 }}/>
        {catalog.map(skill => <col key={skill.code} style={{ width: 78 }}/>) }
        <col style={{ width: 90 }}/>
      </colgroup>
      <thead><tr>
        <th scope="col">Игрок</th><th scope="col">Реальное имя</th><th scope="col">Амплуа</th>
        {catalog.map(skill => sortHeader(skill.code, skill.name))}
        {sortHeader(MEMBERS_SORT_AVERAGE_CODE, 'Рейтинг')}
      </tr></thead>
      <tbody>{members.map(member => {
        const profile = profiles[member.userTelegramID] ?? null
        const name = fullName(member)
        const realName = (profile?.realName || member.realName || '').trim()
        const role = playerTypeLabel(profile?.playerType || member.playerType || '')
        const skills = new Map(profileSkills(profile, catalog).map(skill => [skill.skillCode, skill.score]))
        return <tr key={member.userTelegramID} className="member-row" onClick={() => open(member)}>
          <td><button type="button" className="studio-member-name" title={name} aria-label={`Открыть профиль: ${realName || name}`} onClick={event => { event.stopPropagation(); open(member) }}>{name}</button></td>
          <td title={realName || undefined}>{realName || <span className="muted">—</span>}</td>
          <td className="studio-member-position" title={role}>{role}</td>
          {catalog.map(skill => {
            const score = skills.get(skill.code)
            return <td className="studio-member-number studio-member-skill" key={skill.code}>{typeof score === 'number' && Number.isFinite(score) ? normalizedSkillScore(score) : <span className="muted" aria-label="Нет оценки">—</span>}</td>
          })}
          <td className="studio-member-number"><MemberRating value={memberRating(profile, catalog)}/></td>
        </tr>
      })}</tbody>
    </table>
  </div>
}

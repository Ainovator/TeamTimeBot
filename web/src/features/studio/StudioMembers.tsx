import { Icon } from '../../components/Icon'
import { MemberRating } from './MemberRating'
import { formatMemberRating, fullName, memberRating, playerTypeLabel, profileSkills } from '../members/memberUtils'
import type { GroupMember, MemberSkillProfile, SkillCatalogItem } from '../../types'
import './memberCards.css'

export function memberInitials(name: string) { return name.trim().split(/\s+/).slice(0, 2).map(part => part[0]).join('').toUpperCase() || 'TT' }

export function StudioMembers({ members, profiles, catalog, open }: { members: GroupMember[]; profiles: Record<number, MemberSkillProfile>; catalog: SkillCatalogItem[]; open: (member: GroupMember) => void }) {
  return <div className="studio-member-grid">{members.map(member => {
    const profile = profiles[member.userTelegramID]
    const name = profile?.realName?.trim() || member.realName?.trim() || fullName(member)
    const role = profile?.playerType || member.playerType
    const skills = profileSkills(profile ?? null, catalog)
    const average = memberRating(profile ?? null, catalog)
    const elite = average !== null && average >= 9
    const rating = formatMemberRating(average)
    return <button
      type="button"
      key={member.userTelegramID}
      className="studio-member-card"
      data-role={role || 'member'}
      data-tier={elite ? 'elite' : 'standard'}
      aria-label={`Открыть профиль: ${name}. Рейтинг: ${rating}${elite ? '. Элита' : ''}`}
      onClick={() => open(member)}
    >
      <span className="studio-player-frame">
        <span className="studio-player-surface">
          <span className="studio-player-hero">
            <span className="studio-player-avatar" aria-hidden="true">{memberInitials(name)}{elite && <span className="studio-player-edition"><Icon name="sparkle" size={11}/>9+</span>}</span>
            <span className="studio-player-rating"><MemberRating value={average} size="large"/><small>Рейтинг</small></span>
          </span>
          <span className="studio-player-identity"><strong>{name}</strong><span>{role ? playerTypeLabel(role) : 'Амплуа не указано'}</span></span>
          <span className="studio-player-stats">{skills.map(skill => <span className="studio-player-stat" key={skill.skillCode}><strong>{typeof skill.score === 'number' && Number.isFinite(skill.score) ? skill.score : '—'}</strong><span>{skill.skillName}</span></span>)}</span>
          <span className="studio-player-link">Профиль и навыки<Icon name="diagonal" size={15}/></span>
        </span>
      </span>
    </button>
  })}</div>
}

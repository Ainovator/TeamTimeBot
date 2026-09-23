import { formatMemberRating } from '../members/memberUtils'
import './memberRating.css'

export function MemberRating({ value, size = 'compact' }: { value: number | null; size?: 'compact' | 'large' }) {
  return <span className={`studio-rating-value studio-rating-value-${size}`} data-unrated={value === null || undefined} aria-label={value === null ? 'Нет оценки' : `${formatMemberRating(value)} из 10`}>
    <strong>{formatMemberRating(value)}</strong>
    {value !== null && <small aria-hidden="true">/10</small>}
  </span>
}

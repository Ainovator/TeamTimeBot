import type { CSSProperties } from 'react'
import type { RosterPlayer } from './rosterData'
import './portraits.css'

export function PlayerPortrait({ player, className = '' }: { player: RosterPlayer; className?: string }) {
  const style = { '--portrait-x': `${(player.id % 3) * 50}%`, '--portrait-y': player.id < 3 ? '0%' : '100%' } as CSSProperties
  return <span className={`dl-roster-portrait dl-roster-tone-${player.color} ${className}`} style={style} role="img" aria-label={`Демопортрет: ${player.name}`}/>
}

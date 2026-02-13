import type { EventHistoryItem, TeamSplitPlayer } from '../../types'

export function historyStatusLabel(status: EventHistoryItem['status']): string {
  if (status === 'in_voting') {
    return 'В голосовании'
  }
  if (status === 'on_distribution') {
    return 'На распределении'
  }
  return 'Завершено'
}

export function playerDisplayName(player: TeamSplitPlayer): string {
  const full = `${player.firstName ?? ''} ${player.lastName ?? ''}`.trim()
  if (full) {
    return full
  }
  if (player.username) {
    return `@${player.username}`
  }
  return `ID ${player.userID}`
}

export type ActiveTeamCode = 'A' | 'B' | 'C'

export function activeTeamCodes(players: TeamSplitPlayer[], includeC = false): ActiveTeamCode[] {
  if (includeC || players.some((player) => player.team === 'C')) {
    return ['A', 'B', 'C']
  }
  return ['A', 'B']
}

export function teamPairProbabilities(players: TeamSplitPlayer[], includeC = false) {
  const teams = activeTeamCodes(players, includeC)
  const scoreByTeam: Record<ActiveTeamCode, number> = { A: 0, B: 0, C: 0 }
  for (const player of players) {
    if (player.team === 'A' || player.team === 'B' || player.team === 'C') {
      scoreByTeam[player.team] += player.rating
    }
  }
  const pairs: Array<{ left: ActiveTeamCode; right: ActiveTeamCode; leftProb: number; rightProb: number }> = []
  for (let i = 0; i < teams.length; i += 1) {
    for (let j = i + 1; j < teams.length; j += 1) {
      const left = teams[i]
      const right = teams[j]
      const leftScore = scoreByTeam[left]
      const rightScore = scoreByTeam[right]
      const total = leftScore + rightScore
      if (total <= 0) {
        pairs.push({ left, right, leftProb: 0.5, rightProb: 0.5 })
      } else {
        pairs.push({ left, right, leftProb: leftScore / total, rightProb: rightScore / total })
      }
    }
  }
  return pairs
}

export function teamColor(team: ActiveTeamCode): string {
  if (team === 'A') {
    return 'var(--team-a)'
  }
  if (team === 'B') {
    return 'var(--team-b)'
  }
  return 'var(--team-c)'
}

export function formatDateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '-'
  }
  return date.toLocaleString('ru-RU')
}

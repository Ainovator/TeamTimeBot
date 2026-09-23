import { activeTeamCodes, teamPairProbabilities, type ActiveTeamCode } from '../history/historyUtils'
import type { TeamSplitPlayer } from '../../types'

export function playerCountLabel(count: number): string {
  const suffix = count % 100 >= 11 && count % 100 <= 14 ? 'игроков'
    : count % 10 === 1 ? 'игрок' : count % 10 >= 2 && count % 10 <= 4 ? 'игрока' : 'игроков'
  return `${count} ${suffix}`
}

export function TeamForecast({ players, includeC }: { players: TeamSplitPlayer[]; includeC: boolean }) {
  const teams = activeTeamCodes(players, includeC)
  const roster = (code: ActiveTeamCode) => players.filter(player => player.team === code)
  const unassigned = players.filter(player => player.team === 'unassigned').length
  const counts = teams.map(code => roster(code).length)
  const hasPlayers = counts.some(count => count > 0)
  const incomplete = unassigned > 0 || counts.some(count => count === 0) || new Set(counts).size > 1
  const average = (code: ActiveTeamCode) => {
    const members = roster(code)
    return members.length ? (members.reduce((sum, player) => sum + player.rating, 0) / members.length)
      .toLocaleString('ru-RU', { minimumFractionDigits: 1, maximumFractionDigits: 1 }) : '—'
  }

  return <section className="studio-forecast" aria-label="Прогноз встречи">
    <div className="studio-forecast-heading">
      <div><h3>Прогноз встречи</h3><p>Оценка баланса по суммарному рейтингу текущих составов</p></div>
      <span className={`studio-forecast-status ${incomplete ? 'is-draft' : ''}`}>
        {incomplete ? 'Составы не завершены' : 'Составы равны по числу игроков'}
      </span>
    </div>
    {!hasPlayers ? <p className="studio-forecast-empty">Распределите игроков хотя бы в две команды — здесь появится сравнение составов.</p>
      : <div className={`studio-forecast-pairs ${teams.length === 3 ? 'has-three' : ''}`}>
        {teamPairProbabilities(players, includeC).map(pair => {
          const leftCount = roster(pair.left).length
          const rightCount = roster(pair.right).length
          const ready = leftCount > 0 && rightCount > 0
          const leftPercent = Math.round(pair.leftProb * 100)
          const rightPercent = 100 - leftPercent
          const balanced = Math.abs(leftPercent - rightPercent) <= 10
          return <article className="studio-forecast-pair" key={`${pair.left}-${pair.right}`} aria-label={`Команда ${pair.left} и команда ${pair.right}`}>
            <div className="studio-forecast-numbers">
              {[pair.left, pair.right].map((code, index) => <div className="studio-forecast-team" data-team={code} key={code}>
                <span className="studio-forecast-name"><i aria-hidden="true"/>Команда {code}</span>
                <strong>{ready ? <>{index === 0 ? leftPercent : rightPercent}<small>%</small></> : '—'}</strong>
                <span className="studio-forecast-meta">{playerCountLabel(roster(code).length)} · ср. рейтинг {average(code)}</span>
              </div>)}
            </div>
            <div className={`studio-forecast-bar ${ready ? '' : 'is-empty'}`} aria-hidden="true">
              {ready && <><span data-team={pair.left} style={{ width: `${leftPercent}%` }}/><span data-team={pair.right} style={{ width: `${rightPercent}%` }}/></>}
            </div>
            <p className="studio-forecast-verdict">{!ready ? 'Добавьте игроков в обе команды'
              : leftCount !== rightCount ? `Разное число игроков: ${leftCount} и ${rightCount}`
              : balanced ? 'Близкий баланс рейтингов'
              : `Перевес по рейтингу у команды ${leftPercent > rightPercent ? pair.left : pair.right}`}</p>
          </article>
        })}
      </div>}
    {hasPlayers && incomplete && <p className="studio-forecast-note">
      {unassigned > 0 ? `Без команды: ${playerCountLabel(unassigned)}. ` : ''}
      Завершите распределение, чтобы сравнить итоговые составы.
    </p>}
  </section>
}

import type { CSSProperties, Dispatch, SetStateAction } from 'react'
import { Icon } from '../../components/Icon'
import { Select } from '../../components/Select'
import { teamColor, type ActiveTeamCode } from '../history/historyUtils'
import './eventSets.css'

export type SetRowDraft = {
  left: ActiveTeamCode
  right: ActiveTeamCode
  leftScore: number
  rightScore: number
}

export function EventSetsEditor({ rows, setRows, teams, loading, available, onSave, onPublish }: {
  rows: SetRowDraft[]
  setRows: Dispatch<SetStateAction<SetRowDraft[]>>
  teams: ActiveTeamCode[]
  loading: boolean
  available: boolean
  onSave: () => void
  onPublish: () => void
}) {
  const availableTeams: ActiveTeamCode[] = teams.length >= 2 ? teams : ['A', 'B']
  const emptyRow: SetRowDraft = { left: availableTeams[0], right: availableTeams[1], leftScore: 0, rightScore: 0 }
  const displayedRows = rows.length ? rows : [emptyRow]
  const ensureOtherTeam = (team: ActiveTeamCode) => availableTeams.find(candidate => candidate !== team) ?? availableTeams[0]
  const updateRow = (index: number, update: (row: SetRowDraft) => SetRowDraft) => {
    setRows(previous => (previous.length ? previous : [emptyRow]).map((row, i) => i === index ? update(row) : row))
  }

  return <section className="content-card studio-sets-editor" aria-label="Результаты партий">
    <div className="studio-sets-heading">
      <h3>Партии</h3>
      <p>Введите счёт каждой партии.</p>
    </div>
    {!available ? <p className="muted">Выберите голосование для этого события.</p> : loading ? <p className="muted" role="status">Загрузка партий…</p> : <>
      <div className="studio-sets-list">
        {displayedRows.map((row, index) => <div className="studio-set-row" key={index}>
          <div className="studio-set-number"><span>Партия</span><strong>{String(index + 1).padStart(2, '0')}</strong></div>
          <div className="studio-set-match">
            {(['left', 'right'] as const).map((side, sideIndex) => <div className={`studio-set-side is-${side}`} key={side}
              style={{ '--set-team-color': teamColor(row[side]) } as CSSProperties}>
              <div className="studio-set-team">
                {availableTeams.length > 2 ? <><span className="studio-set-team-caption">Команда</span><Select
                  aria-label={`Партия ${index + 1}, ${sideIndex === 0 ? 'первая' : 'вторая'} команда`}
                  value={row[side]}
                  onChange={event => {
                    const team = event.target.value as ActiveTeamCode
                    const opposite = side === 'left' ? 'right' : 'left'
                    updateRow(index, current => ({ ...current, [side]: team, [opposite]: team === current[opposite] ? ensureOtherTeam(team) : current[opposite] }))
                  }}
                >{availableTeams.map(team => <option key={team} value={team}>{team}</option>)}</Select></>
                  : <span className="studio-set-team-label"><i aria-hidden="true"/>{`Команда ${row[side]}`}</span>}
              </div>
              <input type="number" min="0" max="99" inputMode="numeric"
                aria-label={`Партия ${index + 1}, очки команды ${row[side]}`}
                value={row[`${side}Score`]}
                onChange={event => {
                  const value = Number(event.target.value)
                  updateRow(index, current => ({ ...current, [`${side}Score`]: value }))
                }}/>
            </div>)}
            <span className="studio-set-divider" aria-hidden="true">:</span>
          </div>
          <button type="button" className="studio-set-remove"
            title="Удалить партию" aria-label={`Удалить партию ${index + 1}`}
            disabled={displayedRows.length <= 1}
            onClick={() => setRows(previous => previous.filter((_, i) => i !== index))}>
            <Icon name="close" size={18}/>
          </button>
        </div>)}
      </div>
      <button type="button" className="studio-set-add" onClick={() => setRows(previous => {
        const current = previous.length ? previous : [emptyRow]
        const last = current[current.length - 1]
        return [...current, { left: last.left, right: last.right, leftScore: 0, rightScore: 0 }]
      })}><Icon name="plus" size={18}/>Добавить партию</button>
      <div className="studio-sets-actions">
        <button type="button" onClick={onSave}>Сохранить результаты</button>
        <button type="button" className="btn-secondary" onClick={onPublish}>Опубликовать</button>
        {availableTeams.length > 2 && <button type="button" className="studio-sets-reset" onClick={() => setRows([])}>Сбросить</button>}
      </div>
    </>}
  </section>
}

import { useEffect, useRef, useState } from 'react'
import { fetchGroupMembers } from '../../api'
import { notifyError } from '../../components/ErrorNotifications'
import { getAttendance, getPasses, markAttendance, type Attendance, type TrainingPass } from './passesApi'
import type { GroupMember } from '../../types'
import './trainingPasses.css'

export function TrainingAttendance({chatID,instanceID,cancelled,startAt,onChanged}:{chatID:number;instanceID:number;cancelled:boolean;startAt:string;onChanged:()=>void}) {
 const [members,setMembers]=useState<GroupMember[]>([]),[rows,setRows]=useState<Attendance[]>([]),[passes,setPasses]=useState<TrainingPass[]>([])
 const [loading,setLoading]=useState(true),[failed,setFailed]=useState(false),[revision,setRevision]=useState(0),[busy,setBusy]=useState<number|null>(null),[query,setQuery]=useState(''),[page,setPage]=useState(0)
 const lock=useRef(false),alive=useRef(true)
 useEffect(()=>{alive.current=true;return()=>{alive.current=false}},[])
 useEffect(()=>{let dead=false;setLoading(true);setFailed(false);Promise.all([fetchGroupMembers(chatID),getAttendance(chatID,instanceID),getPasses(chatID)]).then(([m,a,p])=>{if(!dead){setMembers(m??[]);setRows(a);setPasses(p)}}).catch(e=>{if(!dead){notifyError(e);setFailed(true)}}).finally(()=>{if(!dead)setLoading(false)});return()=>{dead=true}},[chatID,instanceID,revision])
 async function mark(user:number,status:string){if(lock.current)return;lock.current=true;setBusy(user);try{await markAttendance(chatID,instanceID,user,status);if(alive.current){setRevision(x=>x+1);onChanged()}}catch(e){notifyError(e)}finally{lock.current=false;if(alive.current)setBusy(null)}}
 const name=(m:GroupMember)=>m.realName?.trim()||`${m.firstName||''} ${m.lastName||''}`.trim()||m.username||'Игрок'
 const filtered=members.filter(m=>name(m).toLocaleLowerCase('ru').includes(query.toLocaleLowerCase('ru'))),last=Math.max(0,Math.ceil(filtered.length/12)-1),safePage=Math.min(page,last)
 const future=new Date(startAt).getTime()>Date.now()
 return <section className="training-attendance"><header className="passes-heading"><div><h3>Посещаемость</h3><p>Присутствовали: {rows.filter(a=>a.status==='attended').length} · Отсутствовали: {rows.filter(a=>a.status==='absent').length}</p></div></header>
  <p className="muted">При присутствии списывается занятие с оплаченного абонемента с ближайшим окончанием срока. Если подходящего абонемента нет — разовое посещение. Отменить списание можно кнопкой «Сбросить».</p>
  {cancelled?<p className="passes-notice">Тренировка отменена. Занятия возвращены в абонементы.</p>:future?<p className="passes-notice">Отметки доступны после начала тренировки.</p>:null}
  <input aria-label="Поиск в журнале" placeholder="Найти игрока" value={query} onChange={e=>{setQuery(e.target.value);setPage(0)}}/>
  {loading?<p role="status">Загружаем журнал…</p>:failed?<button className="btn-secondary" onClick={()=>setRevision(x=>x+1)}>Повторить загрузку</button>:<div className="attendance-list">{filtered.slice(safePage*12,safePage*12+12).map(m=>{const a=rows.find(x=>x.userTelegramID===m.userTelegramID),p=passes.find(x=>x.id===a?.passID),status=a?.status;return <article key={m.userTelegramID} className="attendance-row"><div><strong>{name(m)}</strong><span>{status==='attended'?(p?`По абонементу «${p.name}»`:'Разовое посещение'):status==='absent'?'Отсутствовал · без списания':status==='cancelled'?'Событие отменено':'Ещё не отмечен'}</span></div><div className="attendance-actions"><button type="button" className="btn-secondary" aria-pressed={status==='attended'} disabled={loading||busy!==null||cancelled||future} onClick={()=>void mark(m.userTelegramID,'attended')}>Присутствовал</button><button type="button" className="btn-secondary" aria-pressed={status==='absent'} disabled={loading||busy!==null||cancelled||future} onClick={()=>void mark(m.userTelegramID,'absent')}>Отсутствовал</button>{status&&status!=='unmarked'&&status!=='cancelled'&&<button type="button" className="btn-secondary" disabled={loading||busy!==null||cancelled||future} onClick={()=>void mark(m.userTelegramID,'unmarked')}>Сбросить</button>}</div></article>})}{!filtered.length&&<p className="muted">Игроки не найдены</p>}</div>}
  {filtered.length>12&&<nav className="passes-pagination" aria-label="Страницы журнала"><button className="btn-secondary" disabled={!safePage} onClick={()=>setPage(safePage-1)}>Назад</button><span>{safePage+1} / {last+1}</span><button className="btn-secondary" disabled={safePage===last} onClick={()=>setPage(safePage+1)}>Далее</button></nav>}
 </section>
}

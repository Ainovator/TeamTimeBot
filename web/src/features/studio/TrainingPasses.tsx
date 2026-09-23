import { Checkbox } from '../../components/Checkbox'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { fetchGroupMembers } from '../../api'
import { notifyError } from '../../components/ErrorNotifications'
import { Modal } from '../../components/Modal'
import { Select } from '../../components/Select'
import { PlayerPicker } from '../../components/PlayerPicker'
import { Icon } from '../../components/Icon'
import { changePass, getPasses, issuePass, type TrainingPass } from './passesApi'
import type { GroupMember } from '../../types'
import './trainingPasses.css'

const labels:Record<TrainingPass['state'],string>={active:'Действует',unpaid:'Не оплачен',scheduled:'Начнётся позже',frozen:'Заморожен',expired:'Срок истёк',exhausted:'Занятия закончились',closed:'Закрыт'}
export const passDate=(value:string)=>new Date(`${value.slice(0,10)}T12:00:00Z`).toLocaleDateString('ru-RU',{day:'numeric',month:'short',year:'numeric',timeZone:'UTC'})
const localDate=(date:Date)=>`${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,'0')}-${String(date.getDate()).padStart(2,'0')}`
const memberName=(m:GroupMember)=>m.realName?.trim()||`${m.firstName||''} ${m.lastName||''}`.trim()||m.username||'Игрок'

export function BillingWorkspace({chatID,children}:{chatID:number;children:ReactNode}) {
 const [tab,setTab]=useState<'dues'|'passes'>('dues')
 return <div className="passes-workspace"><div className="passes-tabs" aria-label="Взносы и абонементы"><button type="button" aria-pressed={tab==='dues'} onClick={()=>setTab('dues')}>Взносы за события</button><button type="button" aria-pressed={tab==='passes'} onClick={()=>setTab('passes')}>Абонементы</button></div>{tab==='dues'?children:<TrainingPasses key={chatID} chatID={chatID} canManage/>}</div>
}

export function TrainingPasses({chatID,userID,mine=false,canManage=false}:{chatID:number;userID?:number;mine?:boolean;canManage?:boolean}) {
 const [passes,setPasses]=useState<TrainingPass[]>([])
 const [members,setMembers]=useState<GroupMember[]>([])
 const [loading,setLoading]=useState(true),[failed,setFailed]=useState(false),[revision,setRevision]=useState(0)
 const [query,setQuery]=useState(''),[filter,setFilter]=useState('current'),[page,setPage]=useState(0)
 const [issuing,setIssuing]=useState(false),[expanded,setExpanded]=useState<number|null>(null)
 const [action,setAction]=useState<{pass:TrainingPass;code:string}|null>(null),[days,setDays]=useState('30')
 const [busy,setBusy]=useState(false),[notice,setNotice]=useState('')
 const lock=useRef(false),alive=useRef(true)
 useEffect(()=>{alive.current=true;return()=>{alive.current=false}},[])
 useEffect(()=>{let cancelled=false;setLoading(true);setFailed(false)
  Promise.all([getPasses(chatID,userID,mine),canManage?fetchGroupMembers(chatID):Promise.resolve([])]).then(([p,m])=>{if(!cancelled){setPasses(p??[]);setMembers(m??[])}}).catch(e=>{if(!cancelled){setFailed(true);notifyError(e)}}).finally(()=>{if(!cancelled)setLoading(false)})
  return()=>{cancelled=true}
 },[chatID,userID,mine,canManage,revision])
 const current=passes.filter(p=>!['closed','expired','exhausted'].includes(p.state))
 const rows=passes.filter(p=>(filter==='all'||(filter==='current'?current.includes(p):p.state==='unpaid'))&&`${p.memberName} ${p.name}`.toLocaleLowerCase('ru').includes(query.toLocaleLowerCase('ru')))
 const maxPage=Math.max(0,Math.ceil(rows.length/12)-1),safePage=Math.min(page,maxPage)
 const expiring=current.filter(p=>p.state==='active'&&((p.totalVisits!==null&&p.totalVisits-p.usedVisits<=1)||new Date(p.endsOn.slice(0,10)).getTime()-Date.now()<7*86400000))
 const refresh=()=>{setRevision(x=>x+1)}
 async function applyAction(){if(!action||lock.current)return;lock.current=true;setBusy(true)
  try{await changePass(chatID,action.pass,action.code,action.code==='extend'?Number(days):undefined);if(alive.current){setAction(null);setNotice('Изменение сохранено');refresh()}}
  catch(e){notifyError(e);if(alive.current)refresh()}finally{lock.current=false;if(alive.current)setBusy(false)}
 }
 return <section className="content-card training-passes" aria-label="Абонементы">
  <header className="passes-heading"><div><h3>Абонементы</h3><p>Тренировки организации · списание по фактическому посещению</p></div>{canManage&&<button type="button" onClick={()=>setIssuing(true)} disabled={loading||failed}><Icon name="plus" size={17}/>Выдать абонемент</button>}</header>
  {notice&&<p className="passes-notice" role="status">{notice}</p>}
  {loading?<p role="status">Загружаем абонементы…</p>:failed?<button type="button" className="btn-secondary" onClick={refresh}>Повторить загрузку</button>:<>
   {!userID&&!mine&&<div className="passes-metrics"><div><span>Действующие и ожидающие</span><strong>{current.length}</strong></div><div><span>Скоро заканчиваются</span><strong>{expiring.length}</strong></div><div><span>К оплате за абонементы</span><strong>{(passes.filter(p=>!p.isPaid&&!p.isClosed).reduce((s,p)=>s+p.priceKopecks,0)/100).toLocaleString('ru-RU')} ₽</strong></div></div>}
   <div className="passes-toolbar"><input aria-label="Поиск абонемента" placeholder="Игрок или название абонемента" value={query} onChange={e=>{setQuery(e.target.value);setPage(0)}}/><Select aria-label="Фильтр абонементов" value={filter} onChange={e=>{setFilter(e.target.value);setPage(0)}}><option value="current">Текущие</option><option value="unpaid">Не оплачены</option><option value="all">Все, включая архив</option></Select></div>
   {!rows.length?<div className="passes-empty"><Icon name="wallet" size={30}/><h4>{passes.length?'Нет абонементов по выбранному фильтру':'Абонементов пока нет'}</h4><p>{canManage?'Выдайте игроку пакет занятий или безлимит на выбранный срок.':'Здесь появятся выданные вам абонементы и история посещений.'}</p></div>:<div className="passes-list">{rows.slice(safePage*12,safePage*12+12).map(p=><article className="pass-row" key={p.id}>
    <div className="pass-row-main"><div className="pass-identity"><span className="pass-symbol"><Icon name="calendar" size={21}/></span><div>{!userID&&!mine&&<strong>{p.memberName}</strong>}<h4>{p.name}</h4><span>{passDate(p.startsOn)} — {passDate(p.endsOn)}</span></div></div>
     <div className="pass-balance"><strong>{p.totalVisits===null?'Безлимит':`${Math.max(0,p.totalVisits-p.usedVisits)} из ${p.totalVisits}`}</strong><span>{p.totalVisits===null?`Посещено: ${p.usedVisits}`:'занятий осталось'}</span>{p.totalVisits!==null&&<progress max={p.totalVisits} value={Math.max(0,p.totalVisits-p.usedVisits)} aria-label="Остаток занятий"/>}</div>
     <div className="pass-status"><span className={`pass-badge ${p.state}`}>{labels[p.state]}</span><span>{(p.priceKopecks/100).toLocaleString('ru-RU')} ₽ · {p.isPaid?'оплачено':'к оплате'}</span>{expiring.includes(p)&&<small>Пора продлить</small>}</div>
     <button type="button" className="btn-secondary" aria-expanded={expanded===p.id} onClick={()=>setExpanded(expanded===p.id?null:p.id)}>Подробнее<Icon name={expanded===p.id?'up':'down'} size={15}/></button>
    </div>
    {expanded===p.id&&<div className="pass-details">{canManage&&!p.isClosed&&<div className="pass-actions">{!p.isPaid?<button className="btn-secondary" onClick={()=>setAction({pass:p,code:'paid'})}>Отметить оплату</button>:p.usedVisits===0&&<button className="btn-secondary" onClick={()=>setAction({pass:p,code:'unpaid'})}>Снять оплату</button>}{p.frozenAt?<button className="btn-secondary" onClick={()=>setAction({pass:p,code:'resume'})}>Снять заморозку</button>:!['expired','exhausted'].includes(p.state)&&<button className="btn-secondary" onClick={()=>setAction({pass:p,code:'freeze'})}>Заморозить</button>}<button className="btn-secondary" onClick={()=>{setDays('30');setAction({pass:p,code:'extend'})}}>Продлить срок</button><button className="btn-secondary" onClick={()=>setAction({pass:p,code:'close'})}>Закрыть абонемент</button></div>}
     <h4>История операций</h4><ol className="pass-history">{p.history.map(h=><li key={h.id}><time>{new Date(h.createdAt).toLocaleString('ru-RU',{dateStyle:'short',timeStyle:'short'})}</time><span>{h.note}{h.instanceID?` · событие №${h.instanceID}`:''}</span></li>)}</ol>
    </div>}
   </article>)}</div>}
   {rows.length>12&&<nav className="passes-pagination" aria-label="Страницы абонементов"><button className="btn-secondary" disabled={safePage===0} onClick={()=>setPage(safePage-1)}>Назад</button><span>{safePage+1} / {maxPage+1}</span><button className="btn-secondary" disabled={safePage>=maxPage} onClick={()=>setPage(safePage+1)}>Далее</button></nav>}
  </>}
  {issuing&&<IssuePassModal chatID={chatID} members={members} userID={userID} close={()=>setIssuing(false)} saved={()=>{setIssuing(false);setFilter('current');setNotice('Абонемент выдан');refresh()}}/>}
  {action&&<Modal title={{paid:'Отметить оплату',unpaid:'Отменить отметку оплаты',freeze:'Заморозить абонемент',resume:'Снять заморозку',extend:'Продлить срок',close:'Закрыть абонемент'}[action.code]||'Абонемент'} busy={busy} close={()=>setAction(null)}><form onSubmit={e=>{e.preventDefault();void applyAction()}}><p><strong>{action.pass.memberName}</strong> · {action.pass.name}</p><p className="muted">{action.code==='freeze'?'Во время заморозки списания недоступны. После снятия срок увеличится на число календарных дней заморозки.':action.code==='close'?'Новые посещения списываться не будут. История сохранится. Возврат денежных средств здесь не выполняется.':action.code==='extend'?'Продлевается только срок. Чтобы добавить пакет занятий, выдайте новый абонемент.':action.code==='paid'?'Подтвердите, что вы получили оплату за абонемент.':'Изменение сохранится в истории абонемента.'}</p>{action.code==='extend'&&<label className="field"><span>На сколько дней</span><input type="number" min="1" max="365" required value={days} onChange={e=>setDays(e.target.value)}/></label>}<div className="pass-actions"><button type="button" className="btn-secondary" disabled={busy} onClick={()=>setAction(null)}>Отмена</button><button disabled={busy}>{busy?'Сохраняем…':'Подтвердить'}</button></div></form></Modal>}
 </section>
}

function IssuePassModal({chatID,members,userID,close,saved}:{chatID:number;members:GroupMember[];userID?:number;close:()=>void;saved:()=>void}){
 const [user,setUser]=useState(userID?String(userID):''),[name,setName]=useState('8 тренировок'),[count,setCount]=useState('8'),[kind,setKind]=useState('visits')
 const [start,setStart]=useState(localDate(new Date())),[end,setEnd]=useState(localDate(new Date(Date.now()+30*86400000))),[price,setPrice]=useState(''),[paid,setPaid]=useState(false),[busy,setBusy]=useState(false)
 const [requestID]=useState(()=>crypto.randomUUID()),lock=useRef(false)
 async function submit(){if(lock.current)return;if(!user){notifyError('Выберите игрока');return}lock.current=true;setBusy(true)
  try{await issuePass(chatID,{userTelegramID:Number(user),name,totalVisits:kind==='unlimited'?null:Number(count),startsOn:start,endsOn:end,priceKopecks:Math.round(Number(price)*100),isPaid:paid,requestID});saved()}catch(e){notifyError(e)}finally{lock.current=false;setBusy(false)}
 }
 return <Modal title="Выдать абонемент" close={close} busy={busy}><form className="pass-issue-form" onSubmit={e=>{e.preventDefault();void submit()}}>
  <div className="field"><span>Игрок</span><PlayerPicker label="Получатель абонемента" options={members.map(m=>({id:m.userTelegramID,label:memberName(m),username:m.username||''}))} value={user} onChange={id=>setUser(String(id))}/></div>
  <div className="pass-presets" aria-label="Готовые пакеты">{[4,8,12].map(n=><button type="button" className="btn-secondary" aria-pressed={kind==='visits'&&Number(count)===n} key={n} onClick={()=>{setKind('visits');setCount(String(n));setName(`${n} ${n===4?'тренировки':'тренировок'}`)}}>{n} {n===4?'занятия':'занятий'}</button>)}<button type="button" className="btn-secondary" aria-pressed={kind==='unlimited'} onClick={()=>{setKind('unlimited');setName('Безлимит')}}>Безлимит</button></div>
  <label className="field"><span>Название</span><input required maxLength={100} value={name} onChange={e=>setName(e.target.value)}/></label>
  <div className="pass-form-grid">{kind==='visits'&&<label className="field"><span>Количество занятий</span><input type="number" min="1" max="1000" required value={count} onChange={e=>setCount(e.target.value)}/></label>}<label className="field"><span>Стоимость, ₽</span><input type="number" min="0" max="1000000" step="0.01" required placeholder="0" value={price} onChange={e=>setPrice(e.target.value)}/></label><label className="field"><span>Начало действия</span><input type="date" required value={start} onChange={e=>setStart(e.target.value)}/></label><label className="field"><span>Действует включительно до</span><input type="date" required min={start} value={end} onChange={e=>setEnd(e.target.value)}/></label></div>
  <label className="pass-paid-toggle"><Checkbox checked={paid} onChange={e=>setPaid(e.target.checked)}/>Оплата уже получена</label><p className="muted">Одно присутствие на тренировке — одно занятие. Пропуски не списываются. Абонемент покрывает самого игрока, без гостей.</p>
  <div className="pass-actions"><button type="button" className="btn-secondary" disabled={busy} onClick={close}>Отмена</button><button disabled={busy}>{busy?'Выдаём…':'Выдать абонемент'}</button></div>
 </form></Modal>
}

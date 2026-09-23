// Dev-only fixture entry. Not imported by src/main.tsx or included in the production build.
// Every /api request is handled here; no test mutation reaches the backend or Telegram.
import React from 'react'
import { createRoot } from 'react-dom/client'
import '../src/styles.css'
import '../src/studio-theme.css'
import '@fontsource-variable/golos-text'

const params = new URLSearchParams(location.search)
const templateOnly = params.get('role') === 'templates'
const memberMode = params.get('role') === 'member' || templateOnly
const allowedPermissions = templateOnly ? ['templates_manage'] : ['events_read','polls_read','profile_read']
const group = { chatID:-900001, title:'Орбита · тест Studio', timezone:'Europe/Moscow', role:memberMode?'member':'admin' }
const groups = [group,{...group,chatID:-900002,title:'Вторая команда · тест'}]
const now = new Date()
const at = (days:number) => new Date(now.getFullYear(),now.getMonth(),now.getDate()+days,19,0).toISOString()
const names = [['Алексей','Морозов','setter'],['Мария','Волкова','attacker'],['Дмитрий','Соколов','central'],['Иван','Петров','attacker'],['Анна','Лебедева','libero'],['Елена','Кузнецова','setter']]
const members = names.map(([firstName,lastName,playerType],i)=>({userTelegramID:101+i,firstName,lastName,realName:`${firstName} ${lastName}`,username:`player_${i+1}`,playerType,role:i?'member':'admin',status:'active',lastSeenAt:at(-1)}))
const catalog = [{code:'serve',name:'Подача'},{code:'receive',name:'Приём'},{code:'attack',name:'Атака'},{code:'block',name:'Блок'},{code:'setting',name:'Пас'}]
const profiles = Object.fromEntries(members.map((m,i)=>[m.userTelegramID,{...m,skills:catalog.map((skill,j)=>({skillCode:skill.code,skillName:skill.name,score:5+(i+j)%5}))}]))
// Card appearance cases: ordinary, elite threshold, above threshold, no ratings,
// just below the threshold, and an unusually long name. Fixture data only.
if (params.get('cards') === 'ratings') {
  const scores = [7.4, 9, 9.6, undefined, 8.96, 10]
  members.forEach((member, i) => {
    profiles[member.userTelegramID].skills = catalog.map(skill => ({skillCode:skill.code,skillName:skill.name,score:scores[i]}))
  })
  members[5].realName = 'Елена Александровна Константинопольская'
  profiles[members[5].userTelegramID].realName = members[5].realName
}
const templates:Record<string,any> = {'Вечерняя тренировка':{name:'Вечерняя тренировка',question:'Кто играет в волейбол?',options:['Играю','Не смогу','Я + 1'],countedOptions:[0,2],optionWeights:[1,1,2]},'Выходные':{name:'Выходные',question:'Собираемся в субботу?',options:['Буду','Пропускаю'],countedOptions:[0],optionWeights:[1,1]}}
const events = ['Вечерний волейбол','Играем в среду','Субботняя тренировка'].map((name,i)=>({id:201+i,name,eventType:'training',startWeekday:i?6:1,pollPublishWeekday:1,pollPublishTime:'10:00',startTime:'19:00',endTime:'21:00',announcementText:'До встречи на площадке!',announcementEnabled:true,announcementLeadMinutes:60,publishEnabled:true,teamsAutoSplit:true,teamsPublishList:true,teamSize:3,maxPlaces:12,minVotesToHold:4,cancelLeadMinutes:180,cancelNotifyEnabled:true,settlementEnabled:true,settlementPublishBefore:false,settlementPublishAfter:true,costAmount:3600,isActive:true,pollTemplate:'Вечерняя тренировка'}))
const history = events.map((event,i)=>({instanceID:301+i,eventID:event.id,name:event.name,eventType:'training',localDate:at(i-3).slice(0,10),startWeekday:event.startWeekday,startTime:'19:00',pollTemplate:event.pollTemplate,latestPostID:401+i,latestPollAt:at(i-4),nextStartAt:at(i-3),endAt:at(i-3).replace('16:00','18:00'),status:'on_review',canDistribute:true,publishEnabled:true,debtAmount:i===2?750:600}))
history.push({...history[0],instanceID:304,name:'Вечерняя тренировка',nextStartAt:at(2),endAt:at(2),localDate:at(2).slice(0,10),status:'in_voting',debtAmount:0})
// Larger event lists for pagination QA; only the isolated browser fixture changes.
const eventCount = Math.min(200, Math.max(4, Number(params.get('eventCount')) || 4))
for (let i = history.length; i < eventCount; i++) {
  history.push({...history[0], instanceID:1000+i, name:`Тестовая встреча ${i+1}`, nextStartAt:at(-i), endAt:at(-i), localDate:at(-i).slice(0,10), status:i%2 ? 'completed' : 'on_review', debtAmount:i%2 ? 0 : 600})
}
const bills:Record<number,any> = Object.fromEntries(history.slice(0,3).map((event,i)=>[event.instanceID,{eventID:event.eventID,settlementID:501+i,localDate:event.localDate,totalAmount:i===2?4500:3600,amountPerPerson:i===2?750:600,participantsCount:6,players:members.map(member=>({userID:member.userTelegramID,...member,amountDue:i===2?750:600,isPaid:member.userTelegramID!==(i===2?104:103)}))}]))
const debtors = () => members.map(member=>{const trainings=history.filter(event=>bills[event.instanceID]?.players.some((p:any)=>p.userID===member.userTelegramID&&!p.isPaid)).map(event=>({instanceID:event.instanceID,eventName:event.name,startAt:event.nextStartAt,amountDue:bills[event.instanceID].amountPerPerson}));return {...member,userID:member.userTelegramID,totalDebt:trainings.reduce((sum,t)=>sum+t.amountDue,0),trainings}}).filter(row=>row.totalDebt>0)
const summary = () => ({totalDebt:debtors().reduce((sum,row)=>sum+row.totalDebt,0),unpaidRows:debtors().reduce((sum,row)=>sum+row.trainings.length,0)})
const billing = (id:number) => {const bill=bills[id];return bill?{...bill,paidCount:bill.players.filter((p:any)=>p.isPaid).length,unpaidCount:bill.players.filter((p:any)=>!p.isPaid).length,debtAmount:bill.players.filter((p:any)=>!p.isPaid).reduce((sum:number,p:any)=>sum+p.amountDue,0),allPaymentsChecked:bill.players.every((p:any)=>p.isPaid)}:null}
const polls = history.map((event,i)=>({postID:401+i,instanceID:event.instanceID,eventID:event.eventID,eventName:event.name,eventLocalDate:event.localDate,templateName:'Вечерняя тренировка',question:'Кто играет в волейбол?',telegramMessageID:601+i,telegramPollID:`qa-${i}`,status:i===3?'open':'closed',publishedAt:at(i-4),countedVotes:5,totalVotes:6,teamsConfigured:true}))
// Event header scenarios stay inside the in-memory fixture.
if (params.get('eventPolls') === 'multiple') polls.push({...polls[0], postID:499, question:'Дополнительная запись на вечернюю тренировку', publishedAt:at(-2), countedVotes:2, totalVotes:4, teamsConfigured:false})
if (params.get('eventPolls') === 'none') polls.splice(0, polls.length)
if (params.get('eventHeader') === 'long') {
  history[0].name = 'Открытая вечерняя тренировка по волейболу для начинающих и постоянных участников команды'
  history[0].endAt = at(-2)
  history[0].debtAmount = 18000
}
const votes = members.map((m,i)=>({userID:m.userTelegramID,...m,choice:i===5?'1':'0',choiceIndex:i===5?1:0,choiceLabel:i===5?'Не смогу':'Играю',counted:i!==5,source:'telegram',votedAt:at(-4)}))
const teams = members.map((m,i)=>({userID:m.userTelegramID,...m,choice:'0',choiceIndex:0,choiceLabel:'Играю',rating:7,team:i<3?'A':'B',position:i%3}))
// Manual layout scenarios, isolated from the local PostgreSQL demo organization.
if (params.get('roster') === 'uneven') teams.forEach((player, i) => { player.team = i < 2 ? 'A' : i < 5 ? 'B' : 'unassigned' })
if (params.get('roster') === 'empty') teams.forEach(player => { player.team = 'unassigned' })
if (params.get('roster') === 'three') teams.forEach((player, i) => { player.team = i < 2 ? 'A' : i < 5 ? 'B' : 'C' })
const gameRows = [{instanceID:301,ordinal:1,eventName:events[0].name,startAt:at(-3),team1:'A',team2:'B',score1:25,score2:21},{instanceID:301,ordinal:2,eventName:events[0].name,startAt:at(-3),team1:'A',team2:'B',score1:23,score2:25}]
const audit:string[] = []
let refreshRequested = false
document.addEventListener('click', event => {
  if (event.target instanceof Element && event.target.closest('.studio-header-refresh, .mobile-refresh-btn')) {
    refreshRequested = true
  }
}, true)
const originalFetch = window.fetch.bind(window)
window.fetch = async (input, init) => {
  const url=new URL(typeof input==='string'?input:input instanceof URL?input.href:input.url,location.origin)
  if (!url.pathname.startsWith('/api/')) return originalFetch(input,init)
  const path=decodeURIComponent(url.pathname), method=init?.method||'GET', data=init?.body?JSON.parse(String(init.body)):{}
  const json=(value:any,status=200)=>Promise.resolve(new Response(JSON.stringify(value),{status,headers:{'Content-Type':'application/json'}}))
  // Keep refresh pending (or fail it) to inspect the header indicator and stable content.
  if (method==='GET' && /\/groups\/-?\d+$/.test(path) && refreshRequested) {
    refreshRequested = false
    const delay = Math.min(10000, Math.max(0, Number(params.get('refreshDelay')) || 0))
    if (delay) await new Promise(resolve => setTimeout(resolve, delay))
    if (params.has('refreshError')) return json({error:'Тест: не удалось обновить данные команды'},503)
  }
  if (method!=='GET') { audit.push(`${method} ${path} ${JSON.stringify(data)}`); const log=document.querySelector('#qa-audit');if(log)log.textContent=audit.join('\n') }
  if(params.has('errors') && method!=='GET')return json({error:params.get('errors')||'bot is not configured'},400)
  if(params.has('networkError') && /\/events\/history$/.test(path))throw new TypeError('Failed to fetch')
  if(params.has('historyError') && /\/events\/history$/.test(path))return json({error:'internal server error'},500)
  if(path==='/api/auth/config')return json({enabled:true})
  if(path==='/api/auth/me')return json({enabled:true,user:{id:101,firstName:'Алексей',lastName:'Морозов',username:'qa_alex',authDate:Math.floor(Date.now()/1000)}})
  if(path==='/api/groups')return json(groups)
  if(/\/permissions$/.test(path))return json({roleCode:memberMode?'member':'admin',roleTitle:memberMode?'Участник':'Организатор',permissions:Object.fromEntries(['events_read','events_manage','polls_read','members_read','members_manage','billing_manage','templates_manage','event_templates_manage','roles_manage','profile_read'].map(key=>[key,!memberMode||allowedPermissions.includes(key)]))})
  if(/\/groups\/-?\d+$/.test(path))return json({group:groups.find(g=>path.endsWith(String(g.chatID)))||group,templateNames:Object.keys(templates),templates:Object.values(templates).map(t=>({name:t.name,question:t.question,countedOptionsCount:t.countedOptions.length})),events,schedules:[]})
  if(/\/billing\/summary$/.test(path))return json(summary())
  if(/\/billing\/debtors$/.test(path))return params.has('billingError') ? json({error:'Тест: сервер временно недоступен'},503) : json(debtors())
  if(/\/billing\/publish$/.test(path))return json({status:'ok'})
  const billMatch=path.match(/\/history\/(\d+)\/billing$/)
  if(billMatch){const id=Number(billMatch[1]);if(params.has('failPayment')&&method==='PUT'&&id===302&&data.statuses?.some((s:any)=>s.paid))return json({error:'Тест: второй взнос не сохранён'},500);if(method==='PUT')for(const item of data.statuses||[]){const player=bills[id]?.players.find((p:any)=>p.userID===item.userID);if(player)player.isPaid=item.paid}return json(method==='PUT'?{status:'ok'}:billing(id))}
  if(/\/members$/.test(path))return json(members)
  if(/\/events\/mention-recipients$/.test(path))return json(members)
  if(/\/members\/skills$/.test(path))return json(catalog)
  const profileMatch=path.match(/\/members\/(\d+)\/(skills|profile)$/)
  if(profileMatch){const profile=profiles[Number(profileMatch[1])];if(method==='PUT'){if(data.scores)profile.skills=profile.skills.map((skill:any)=>({...skill,score:data.scores[skill.skillCode]}));else Object.assign(profile,data);return json({status:'ok'})}return json(profile)}
  if(/\/relations$/.test(path))return json(method==='GET'?[]:{status:'ok'})
  if(/\/events\/archived$/.test(path))return json([])
  if(/\/events\/history$/.test(path))return json(history)
  if(/\/events\/history\/\d+\/polls$/.test(path))return json(polls.filter(p=>path.includes(String(p.instanceID))))
  if(/\/events\/\d+\/activity$/.test(path))return json({eventID:201,pollsTotal:4,votesTotal:24,settlementsTotal:3,announcementsTotal:4})
  if(/\/events$/.test(path)){if(method==='POST'){events.push({...events[0],...data,id:210+events.length});return json({status:'ok'})}return json(events)}
  const eventUpdate=path.match(/\/events\/(\d+)\/(details|cost|bind)$/)
  if(eventUpdate){Object.assign(events.find(e=>e.id===Number(eventUpdate[1]))!,data);return json({status:'ok'})}
  if(/\/polls\/\d+\/teams/.test(path))return json({eventID:201,postID:401,players:teams,chance:{teamAScore:21,teamBScore:21,teamAProb:.5,teamBProb:.5},formations:{}})
  if(/\/polls\/\d+\/votes$/.test(path))return json(method==='GET'?votes:{status:'ok'})
  if(/\/polls\/\d+\/options$/.test(path))return json(templates['Вечерняя тренировка'].options.map((choiceLabel:string,i:number)=>({choice:String(i),choiceIndex:i,choiceLabel,choiceWeight:1,counted:i!==1})))
  if(/\/polls$/.test(path))return json(polls)
  if(/\/sets$/.test(path))return json(method==='GET'?gameRows.map(row=>({ordinal:row.ordinal,team1:row.team1,team2:row.team2,score1:row.score1,score2:row.score2})):{status:'ok'})
  if(/\/games$/.test(path))return json(gameRows)
  if(/\/roster$/.test(path))return json({team1:'A',team2:'B',team1Players:teams.slice(0,3),team2Players:teams.slice(3)})
  if(/\/me$/.test(path))return json({roleCode:memberMode?'member':'admin',roleTitle:memberMode?'Участник':'Организатор',debtAmount:0,trainings:history.map(event=>({instanceID:event.instanceID,name:event.name,startAt:event.nextStartAt,endAt:event.endAt,status:event.status,amountDue:600,isPaid:true}))})
  if(/\/roles$/.test(path))return json([{code:'captain',title:'Капитан',permissions:{events_read:true}},{code:'trainer',title:'Тренер',permissions:{members_read:true}}])
  const templateMatch=path.match(/\/templates(?:\/(.+))?$/)
  if(templateMatch){const name=templateMatch[1]||data.name;if(method==='DELETE'){delete templates[name];return json({status:'ok'})}if(method==='PUT'||method==='POST'){templates[name]={...templates[name],...data,name};return json({status:'ok'})}return templates[name]?json(templates[name]):json({error:'Шаблон не найден'},404)}
  if(method!=='GET')return json({status:'ok'})
  return json({error:`QA fixture missing: ${path}`},501)
}
const screen=params.get('screen')||'overview'
historyState()
function historyState(){window.history.replaceState({},'',`/org/qa-studio--900001/${screen}`)}
const {default:App}=await import('../src/App')
createRoot(document.getElementById('root')!).render(<React.StrictMode><App/><details style={{position:'fixed',bottom:8,right:8,zIndex:6000,fontSize:10,background:'#fff',border:'1px solid #dce4d8',borderRadius:7,padding:'7px 10px',maxWidth:'90vw'}}><summary>Тестовые данные · API изолирован</summary><pre id="qa-audit" style={{maxHeight:180,overflow:'auto',maxWidth:650,whiteSpace:'pre-wrap'}}/><a href="/qa/studio.html">Сбросить тесты</a> · <a href="/">Открыть продукт</a></details></React.StrictMode>)

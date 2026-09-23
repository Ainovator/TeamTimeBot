import { apiRequest } from '../../api'

export type TrainingPass = {
 id:number; userTelegramID:number; memberName:string; name:string; totalVisits:number|null; usedVisits:number;
 startsOn:string; endsOn:string; priceKopecks:number; isPaid:boolean; frozenAt:string|null; isClosed:boolean;
 version:number; state:'active'|'unpaid'|'scheduled'|'frozen'|'expired'|'exhausted'|'closed';
 history:{id:number;instanceID:number|null;actorID:number;note:string;createdAt:string}[];
}
export type Attendance = {userTelegramID:number;passID:number|null;status:'attended'|'absent'|'unmarked'|'cancelled'}
export type IssuePass = {userTelegramID:number;name:string;totalVisits:number|null;startsOn:string;endsOn:string;priceKopecks:number;isPaid:boolean;requestID:string}
export const getPasses=(chat:number,user?:number,mine=false)=>apiRequest<TrainingPass[]>(`/api/groups/${chat}/passes${mine?'/mine':user?`?userID=${user}`:''}`)
export const issuePass=(chat:number,payload:IssuePass)=>apiRequest<{id:number}>(`/api/groups/${chat}/passes`,'POST',payload)
export const changePass=(chat:number,pass:TrainingPass,action:string,days?:number)=>apiRequest(`/api/groups/${chat}/passes/${pass.id}`,'PATCH',{action,version:pass.version,days})
export const getAttendance=(chat:number,event:number)=>apiRequest<Attendance[]>(`/api/groups/${chat}/attendance/${event}`)
export const markAttendance=(chat:number,event:number,user:number,status:string)=>apiRequest(`/api/groups/${chat}/attendance/${event}`,'PUT',{userTelegramID:user,status})

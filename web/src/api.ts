import type {
  ApiError,
  EventHistoryItem,
  EventPollHistoryItem,
  EventTeamSplitState,
  EventActivitySummary,
  EventView,
  Group,
  GroupDetails,
  GroupMember,
  MemberSkillProfile,
  PlayerRelation,
  SkillCatalogItem,
  TemplateDetails,
} from './types'

async function parseResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let message = `HTTP ${res.status}`
    try {
      const body = (await res.json()) as ApiError
      if (body?.error) {
        message = body.error
      }
    } catch {
      // keep default message
    }
    throw new Error(message)
  }
  return (await res.json()) as T
}

export async function fetchGroups(): Promise<Group[]> {
  const res = await fetch('/api/groups')
  return parseResponse<Group[]>(res)
}

export async function fetchGroupDetails(chatID: number): Promise<GroupDetails> {
  const res = await fetch(`/api/groups/${chatID}`)
  return parseResponse<GroupDetails>(res)
}

export async function fetchGroupMembers(chatID: number): Promise<GroupMember[]> {
  const res = await fetch(`/api/groups/${chatID}/members`)
  return parseResponse<GroupMember[]>(res)
}

export async function fetchSkillsCatalog(chatID: number): Promise<SkillCatalogItem[]> {
  const res = await fetch(`/api/groups/${chatID}/members/skills`)
  return parseResponse<SkillCatalogItem[]>(res)
}

export async function fetchMemberSkills(chatID: number, userTelegramID: number): Promise<MemberSkillProfile> {
  const res = await fetch(`/api/groups/${chatID}/members/${userTelegramID}/skills`)
  return parseResponse<MemberSkillProfile>(res)
}

export async function updateMemberSkills(chatID: number, userTelegramID: number, scores: Record<string, number>) {
  const res = await fetch(`/api/groups/${chatID}/members/${userTelegramID}/skills`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scores }),
  })
  await parseResponse<{ status: string }>(res)
}

export async function updateMemberProfile(
  chatID: number,
  userTelegramID: number,
  payload: { playerType: '' | 'attacker' | 'setter' | 'libero' },
) {
  const res = await fetch(`/api/groups/${chatID}/members/${userTelegramID}/profile`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  await parseResponse<{ status: string }>(res)
}

export async function fetchMemberRelations(chatID: number, userTelegramID: number): Promise<PlayerRelation[]> {
  const res = await fetch(`/api/groups/${chatID}/members/${userTelegramID}/relations`)
  return parseResponse<PlayerRelation[]>(res)
}

export async function upsertMemberRelation(
  chatID: number,
  userTelegramID: number,
  payload: { otherUserID: number; relationType: 'prefer_together' | 'avoid_together'; weight: number },
) {
  const res = await fetch(`/api/groups/${chatID}/members/${userTelegramID}/relations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  await parseResponse<{ status: string }>(res)
}

export async function deleteMemberRelation(
  chatID: number,
  userTelegramID: number,
  otherUserID: number,
  relationType: 'prefer_together' | 'avoid_together',
) {
  const res = await fetch(
    `/api/groups/${chatID}/members/${userTelegramID}/relations/${otherUserID}/${encodeURIComponent(relationType)}`,
    { method: 'DELETE' },
  )
  await parseResponse<{ status: string }>(res)
}

export async function createTemplate(
  chatID: number,
  payload: { name: string; question: string; options: string[]; countedOptions: number[] },
) {
  const res = await fetch(`/api/groups/${chatID}/templates`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  await parseResponse<{ status: string }>(res)
}

export async function deleteTemplate(chatID: number, templateName: string) {
  const res = await fetch(`/api/groups/${chatID}/templates/${encodeURIComponent(templateName)}`, {
    method: 'DELETE',
  })
  await parseResponse<{ status: string }>(res)
}

export async function fetchTemplate(chatID: number, templateName: string): Promise<TemplateDetails> {
  const res = await fetch(`/api/groups/${chatID}/templates/${encodeURIComponent(templateName)}`)
  return parseResponse<TemplateDetails>(res)
}

export async function updateTemplate(
  chatID: number,
  templateName: string,
  payload: { name: string; question: string; options: string[]; countedOptions: number[] },
) {
  const res = await fetch(`/api/groups/${chatID}/templates/${encodeURIComponent(templateName)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  return parseResponse<TemplateDetails>(res)
}

export async function createEvent(
  chatID: number,
  payload: {
    name: string
    eventType: 'training' | 'activity'
    weekday: number
    publishWeekday: number
    publishAt: string
    startAt: string
    endAt: string
    announcementText: string
    announcementEnabled: boolean
    announcementLeadMinutes: number
    teamsAutoSplit: boolean
    teamsPublishList: boolean
    teamSize: number
    minVotesToHold: number
    cancelLeadMinutes: number
    cancelNotifyEnabled: boolean
    settlementEnabled: boolean
    settlementPublishBefore: boolean
    settlementPublishAfter: boolean
    costAmount?: number
  },
) {
  const res = await fetch(`/api/groups/${chatID}/events`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  return parseResponse<EventView>(res)
}

export async function updateEventDetails(
  chatID: number,
  eventID: number,
  payload: {
    name: string
    eventType: 'training' | 'activity'
    weekday: number
    publishWeekday: number
    publishAt: string
    startAt: string
    endAt: string
    announcementText: string
    announcementEnabled: boolean
    announcementLeadMinutes: number
    teamsAutoSplit: boolean
    teamsPublishList: boolean
    teamSize: number
    minVotesToHold: number
    cancelLeadMinutes: number
    cancelNotifyEnabled: boolean
    settlementEnabled: boolean
    settlementPublishBefore: boolean
    settlementPublishAfter: boolean
  },
) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/details`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  await parseResponse<EventView>(res)
}

export async function bindEvent(chatID: number, eventID: number, templateName: string) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/bind`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ templateName }),
  })
  await parseResponse<{ status: string }>(res)
}

export async function updateEventCost(chatID: number, eventID: number, costAmount?: number) {
  const payload = costAmount === undefined ? { costAmount: null } : { costAmount }
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/cost`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  await parseResponse<{ status: string }>(res)
}

export async function fetchEvents(chatID: number): Promise<EventView[]> {
  const res = await fetch(`/api/groups/${chatID}/events`)
  return parseResponse<EventView[]>(res)
}

export async function fetchArchivedEvents(chatID: number): Promise<EventView[]> {
  const res = await fetch(`/api/groups/${chatID}/events/archived`)
  return parseResponse<EventView[]>(res)
}

export async function archiveEvent(chatID: number, eventID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/archive`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function unarchiveEvent(chatID: number, eventID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/unarchive`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function fetchEventActivity(chatID: number, eventID: number): Promise<EventActivitySummary> {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/activity`)
  return parseResponse<EventActivitySummary>(res)
}

export async function activateEventPublications(chatID: number, eventID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/activate`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function deactivateEventPublications(chatID: number, eventID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/deactivate`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function publishEventAnnouncement(chatID: number, eventID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/manual/announcement`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function publishEventPoll(chatID: number, eventID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/manual/poll`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function publishEventSettlement(chatID: number, eventID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/manual/settlement`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function fetchEventPollHistory(chatID: number, eventID: number): Promise<EventPollHistoryItem[]> {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/polls`)
  return parseResponse<EventPollHistoryItem[]>(res)
}

export async function fetchEventHistory(chatID: number): Promise<EventHistoryItem[]> {
  const res = await fetch(`/api/groups/${chatID}/events/history`)
  return parseResponse<EventHistoryItem[]>(res)
}

export async function fetchEventTeamSplit(chatID: number, eventID: number, postID: number): Promise<EventTeamSplitState> {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/polls/${postID}/teams`)
  return parseResponse<EventTeamSplitState>(res)
}

export async function saveEventTeamSplit(
  chatID: number,
  eventID: number,
  postID: number,
  assignments: Array<{ userID: number; team: 'unassigned' | 'A' | 'B' | 'C'; position: number }>,
): Promise<EventTeamSplitState> {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/polls/${postID}/teams`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ assignments }),
  })
  return parseResponse<EventTeamSplitState>(res)
}

export async function publishEventTeamSplit(chatID: number, eventID: number, postID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/polls/${postID}/teams/publish`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function autoSplitEventTeams(chatID: number, eventID: number, postID: number): Promise<EventTeamSplitState> {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/polls/${postID}/teams/autosplit`, {
    method: 'POST',
  })
  return parseResponse<EventTeamSplitState>(res)
}

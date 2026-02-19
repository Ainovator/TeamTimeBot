import type {
  ApiError,
  EventHistoryItem,
  EventPollHistoryItem,
  GroupPollItem,
  GroupPollVoteItem,
  EventTeamSplitState,
  EventActivitySummary,
  EventView,
  EventBilling,
  EventSetRow,
  GroupGameRow,
  GameRosterResponse,
  GroupDebtor,
  Group,
  GroupDebtSummary,
  GroupDetails,
  GroupMember,
  MemberSkillProfile,
  PlayerRelation,
  SkillCatalogItem,
  TemplateDetails,
  AuthConfig,
  AuthUser,
  UserGroupProfile,
  GroupPermissionsView,
  GroupRoleView,
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

export async function fetchMyGroupProfile(chatID: number): Promise<UserGroupProfile> {
  const res = await fetch(`/api/groups/${chatID}/me`)
  return parseResponse<UserGroupProfile>(res)
}

export async function fetchGroupPermissions(chatID: number): Promise<GroupPermissionsView> {
  const res = await fetch(`/api/groups/${chatID}/permissions`)
  return parseResponse<GroupPermissionsView>(res)
}

export async function fetchGroupRoles(chatID: number): Promise<GroupRoleView[]> {
  const res = await fetch(`/api/groups/${chatID}/roles`)
  return parseResponse<GroupRoleView[]>(res)
}

export async function assignGroupRole(chatID: number, payload: { userTelegramID: number; roleCode: string }) {
  const res = await fetch(`/api/groups/${chatID}/roles/assign`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  await parseResponse<{ status: string }>(res)
}

export async function fetchAuthConfig(): Promise<AuthConfig> {
  const res = await fetch('/api/auth/config')
  return parseResponse<AuthConfig>(res)
}

export async function fetchAuthMe(): Promise<{ enabled: boolean; user: AuthUser | null }> {
  const res = await fetch('/api/auth/me')
  return parseResponse<{ enabled: boolean; user: AuthUser | null }>(res)
}

export async function telegramAuthLogin(payload: {
  id: number
  first_name: string
  last_name?: string
  username?: string
  photo_url?: string
  auth_date: number
  hash: string
}): Promise<{ status: string; user: AuthUser }> {
  const res = await fetch('/api/auth/telegram', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  return parseResponse<{ status: string; user: AuthUser }>(res)
}

export async function logoutAuth(): Promise<void> {
  const res = await fetch('/api/auth/logout', { method: 'POST' })
  await parseResponse<{ status: string }>(res)
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
  payload: { playerType: '' | 'attacker' | 'setter' | 'libero' | 'central'; realName?: string },
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
  payload: { name: string; question: string; options: string[]; countedOptions: number[]; optionWeights: number[] },
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
  payload: { name: string; question: string; options: string[]; countedOptions: number[]; optionWeights: number[] },
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
    templateName: string
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
    maxPlaces: number
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

export async function createEventInstance(chatID: number, eventID: number, payload?: { localDate?: string }) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/instances`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload ?? {}),
  })
  return parseResponse<{ instanceID: number }>(res)
}

export async function deleteEventInstance(chatID: number, instanceID: number): Promise<void> {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}`, {
    method: 'DELETE',
  })
  await parseResponse<{ status: string }>(res)
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
    maxPlaces: number
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

export async function fetchEventPollHistoryForInstance(chatID: number, instanceID: number): Promise<EventPollHistoryItem[]> {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}/polls`)
  return parseResponse<EventPollHistoryItem[]>(res)
}

export async function fetchGroupPolls(chatID: number): Promise<GroupPollItem[]> {
  const res = await fetch(`/api/groups/${chatID}/polls`)
  return parseResponse<GroupPollItem[]>(res)
}

export async function fetchGroupPollVotes(chatID: number, postID: number): Promise<GroupPollVoteItem[]> {
  const res = await fetch(`/api/groups/${chatID}/polls/${postID}/votes`)
  return parseResponse<GroupPollVoteItem[]>(res)
}

export async function deleteGroupPollVotesForUser(chatID: number, postID: number, userID: number, choice?: string) {
  const qs = choice ? `?choice=${encodeURIComponent(choice)}` : ''
  const res = await fetch(`/api/groups/${chatID}/polls/${postID}/votes/${userID}${qs}`, { method: 'DELETE' })
  await parseResponse<{ status: string }>(res)
}

export async function publishRegistration(chatID: number) {
  const res = await fetch(`/api/groups/${chatID}/registration/publish`, { method: 'POST' })
  await parseResponse<{ status: string }>(res)
}

export async function fetchEventHistory(chatID: number): Promise<EventHistoryItem[]> {
  const res = await fetch(`/api/groups/${chatID}/events/history`)
  return parseResponse<EventHistoryItem[]>(res)
}

export async function fetchGroupDebtSummary(chatID: number): Promise<GroupDebtSummary> {
  const res = await fetch(`/api/groups/${chatID}/billing/summary`)
  return parseResponse<GroupDebtSummary>(res)
}

export async function fetchGroupDebtors(chatID: number): Promise<GroupDebtor[]> {
  const res = await fetch(`/api/groups/${chatID}/billing/debtors`)
  return parseResponse<GroupDebtor[]>(res)
}

export async function publishGroupDebtors(chatID: number, userIDs: number[]) {
  const res = await fetch(`/api/groups/${chatID}/billing/publish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userIDs }),
  })
  await parseResponse<{ status: string }>(res)
}

export async function fetchEventBilling(chatID: number, eventID: number): Promise<EventBilling | null> {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/billing`)
  return parseResponse<EventBilling | null>(res)
}

export async function fetchEventBillingForInstance(chatID: number, instanceID: number): Promise<EventBilling | null> {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}/billing`)
  return parseResponse<EventBilling | null>(res)
}

export async function fetchEventSetRowsForInstance(chatID: number, instanceID: number): Promise<EventSetRow[]> {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}/sets`)
  return parseResponse<EventSetRow[]>(res)
}

export async function saveEventSetRowsForInstance(chatID: number, instanceID: number, rows: EventSetRow[]) {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}/sets`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ rows }),
  })
  await parseResponse<{ status: string }>(res)
}

export async function publishEventSetRowsForInstance(chatID: number, instanceID: number) {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}/sets/publish`, {
    method: 'POST',
  })
  await parseResponse<{ status: string }>(res)
}

export async function fetchGroupGames(chatID: number): Promise<GroupGameRow[]> {
  const res = await fetch(`/api/groups/${chatID}/games`)
  return parseResponse<GroupGameRow[]>(res)
}

export async function fetchGameRoster(
  chatID: number,
  instanceID: number,
  team1: 'A' | 'B' | 'C',
  team2: 'A' | 'B' | 'C',
): Promise<GameRosterResponse> {
  const qs = new URLSearchParams({ team1, team2 })
  const res = await fetch(`/api/groups/${chatID}/games/${instanceID}/roster?${qs.toString()}`)
  return parseResponse<GameRosterResponse>(res)
}

export async function publishEventBillingDebtorsForInstance(chatID: number, instanceID: number, userIDs: number[]) {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}/billing/publish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userIDs }),
  })
  await parseResponse<{ status: string }>(res)
}

export async function generateEventBillingForInstance(chatID: number, instanceID: number): Promise<EventBilling | null> {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}/billing`, {
    method: 'POST',
  })
  return parseResponse<EventBilling | null>(res)
}

export async function saveEventBilling(
  chatID: number,
  eventID: number,
  statuses: Array<{ userID: number; paid: boolean }>,
) {
  const res = await fetch(`/api/groups/${chatID}/events/${eventID}/billing`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ statuses }),
  })
  await parseResponse<{ status: string }>(res)
}

export async function saveEventBillingForInstance(
  chatID: number,
  instanceID: number,
  statuses: Array<{ userID: number; paid: boolean }>,
) {
  const res = await fetch(`/api/groups/${chatID}/events/history/${instanceID}/billing`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ statuses }),
  })
  await parseResponse<{ status: string }>(res)
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

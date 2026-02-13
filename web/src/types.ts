export type Group = {
  chatID: number
  title: string
  timezone: string
}

export type TemplateView = {
  name: string
  question: string
  countedOptionsCount: number
}

export type TemplateDetails = {
  name: string
  question: string
  options: string[]
  countedOptions: number[]
}

export type ScheduleView = {
  id: number
  templateName: string
  sendAt: string
  isActive: boolean
}

export type EventView = {
  id: number
  name: string
  eventType: 'training' | 'activity'
  startWeekday: number
  pollPublishWeekday: number
  pollPublishTime: string
  startTime: string
  endTime: string
  announcementText: string
  announcementEnabled: boolean
  announcementLeadMinutes: number
  publishEnabled: boolean
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
  isActive: boolean
  pollTemplate: string
}

export type EventActivitySummary = {
  eventID: number
  pollsTotal: number
  votesTotal: number
  settlementsTotal: number
  announcementsTotal: number
}

export type EventPollHistoryItem = {
  postID: number
  telegramMessageID: number
  telegramPollID: string
  status: string
  publishedAt: string
  question: string
  countedVotes: number
  totalVotes: number
  teamsConfigured: boolean
}

export type TeamSplitPlayer = {
  userID: number
  username: string
  firstName: string
  lastName: string
  choice: string
  choiceIndex: number
  choiceLabel: string
  rating: number
  team: 'unassigned' | 'A' | 'B' | 'C'
  position: number
}

export type TeamWinChance = {
  teamAScore: number
  teamBScore: number
  teamAProb: number
  teamBProb: number
}

export type EventTeamSplitState = {
  eventID: number
  postID: number
  players: TeamSplitPlayer[]
  chance: TeamWinChance
}

export type EventHistoryItem = {
  eventID: number
  name: string
  eventType: 'training' | 'activity'
  startWeekday: number
  startTime: string
  pollTemplate: string
  latestPostID?: number
  latestPollAt?: string
  nextStartAt: string
  status: 'completed' | 'in_voting' | 'on_distribution' | 'not_held'
  canDistribute: boolean
  publishEnabled: boolean
}

export type GroupDetails = {
  group: Group
  templateNames: string[]
  templates: TemplateView[]
  schedules: ScheduleView[]
  events: EventView[]
}

export type GroupMember = {
  userTelegramID: number
  username: string
  firstName: string
  lastName: string
  playerType: '' | 'attacker' | 'setter' | 'libero'
  role: string
  status: string
  lastSeenAt: string
}

export type SkillCatalogItem = {
  code: string
  name: string
}

export type MemberSkillValue = {
  skillCode: string
  skillName: string
  score?: number
}

export type MemberSkillProfile = {
  userTelegramID: number
  username: string
  firstName: string
  lastName: string
  playerType: '' | 'attacker' | 'setter' | 'libero'
  skills: MemberSkillValue[]
}

export type ApiError = {
  error: string
}

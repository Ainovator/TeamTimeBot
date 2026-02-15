export type Group = {
  chatID: number
  title: string
  timezone: string
  role?: 'admin' | 'member'
}

export type UserGroupTrainingItem = {
  instanceID: number
  name: string
  startAt: string
  endAt: string
  status: string
  amountDue: number
  isPaid: boolean
}

export type UserGroupProfile = {
  roleCode: string
  roleTitle: string
  debtAmount: number
  trainings: UserGroupTrainingItem[]
}

export type GroupPermissionsView = {
  roleCode: string
  roleTitle: string
  permissions: Record<string, boolean>
}

export type GroupRoleView = {
  code: string
  title: string
  permissions: Record<string, boolean>
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
  optionWeights: number[]
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

export type GroupPollItem = {
  postID: number
  instanceID?: number
  eventID?: number
  eventName: string
  eventLocalDate?: string
  templateName: string
  question: string
  telegramMessageID: number
  telegramPollID: string
  status: string
  publishedAt: string
  countedVotes: number
  totalVotes: number
}

export type GroupPollVoteItem = {
  userID: number
  username: string
  firstName: string
  lastName: string
  choice: string
  choiceIndex?: number
  choiceLabel: string
  counted: boolean
  source: string
  votedAt: string
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
  instanceID: number
  eventID: number
  name: string
  eventType: 'training' | 'activity'
  localDate: string
  startWeekday: number
  startTime: string
  pollTemplate: string
  latestPostID?: number
  latestPollAt?: string
  nextStartAt: string
  endAt: string
  status: 'completed' | 'in_voting' | 'on_distribution' | 'on_review' | 'not_held'
  canDistribute: boolean
  publishEnabled: boolean
  debtAmount: number
}

export type EventBillingPlayer = {
  userID: number
  username: string
  firstName: string
  lastName: string
  amountDue: number
  isPaid: boolean
  paidAt?: string
}

export type EventBilling = {
  eventID: number
  settlementID: number
  localDate: string
  totalAmount: number
  amountPerPerson: number
  participantsCount: number
  paidCount: number
  unpaidCount: number
  debtAmount: number
  players: EventBillingPlayer[]
  allPaymentsChecked: boolean
}

export type EventSetRow = {
  ordinal: number
  team1: 'A' | 'B' | 'C'
  score1: number
  team2: 'A' | 'B' | 'C'
  score2: number
}

export type GroupDebtSummary = {
  totalDebt: number
  unpaidRows: number
}

export type DebtorTrainingDebt = {
  instanceID: number
  eventName: string
  startAt: string
  amountDue: number
}

export type GroupDebtor = {
  userID: number
  username: string
  firstName: string
  lastName: string
  realName: string
  totalDebt: number
  trainings: DebtorTrainingDebt[]
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
  realName: string
  playerType: '' | 'attacker' | 'setter' | 'libero' | 'central'
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
  realName: string
  playerType: '' | 'attacker' | 'setter' | 'libero' | 'central'
  skills: MemberSkillValue[]
}

export type PlayerRelation = {
  userAID: number
  userBID: number
  relationType: 'prefer_together' | 'avoid_together'
  weight: number
  relatedUserID: number
  relatedUsername: string
  relatedFirstName: string
  relatedLastName: string
}

export type ApiError = {
  error: string
}

export type AuthConfig = {
  enabled: boolean
  telegramLoginBot?: string
  sessionCookieName?: string
}

export type AuthUser = {
  id: number
  username: string
  firstName: string
  lastName: string
  authDate: number
}

import { BillingWorkspace, TrainingPasses } from './features/studio/TrainingPasses'
import { TrainingAttendance } from './features/studio/TrainingAttendance'
import { Checkbox } from './components/Checkbox'
import './features/studio/playerProfile.css'
import { Documentation } from './features/docs/Documentation'
import { ErrorNotifications, notifyError, useErrorState } from './components/ErrorNotifications'
import { OrganizationTemplates, OrganizationTemplateNavigation, isTemplateSection } from './features/studio/OrganizationTemplates'
import { StudioEvents, StudioPolls, StudioEventTemplates } from './features/studio/StudioLists'
import { Icon } from './components/Icon'
import { HeaderProgress } from './components/HeaderProgress'
import { OrganizationSettings } from './features/studio/OrganizationSettings'
import { Pagination } from './components/Pagination'
import { HelpTip } from './components/HelpTip'
import { PlayerPicker } from './components/PlayerPicker'
import { StudioOverview } from './features/studio/StudioOverview'
import { StudioMembers, memberInitials } from './features/studio/StudioMembers'
import { MemberTable, MEMBERS_SORT_AVERAGE_CODE, type MembersSortCriterion } from './features/studio/MemberTable'
import { StudioBilling } from './features/studio/StudioBilling'
import { TeamForecast, playerCountLabel } from './features/studio/TeamForecast'
import { EventDetailHeader, EventPollContext } from './features/studio/EventDetailHeader'
import { EventMentionsEditor, emptyEventMentions, equalEventMentions } from './features/studio/EventMentionsEditor'
import { EventDebtReminder } from './features/studio/EventDebtReminder'
import { EventSetsEditor, type SetRowDraft } from './features/studio/EventSetsEditor'
import { LineupAnalysis } from './features/studio/LineupAnalysis'
import { ThemeSwitcher, useProductTheme } from './features/studio/ThemeSwitcher'
import { LoginPage } from './features/auth/LoginPage'
import { Select } from './components/Select'
import { FormEvent, Fragment, useEffect, useMemo, useRef, useState } from 'react'
import {
  addGroupPollVoteForUser,
  activateEventPublications,
  archiveEvent,
  assignGroupRole,
  autoSplitEventTeams,
  bindEvent,
  createEvent,
  createEventInstance,
  createTemplate,
  deactivateEventPublications,
  deleteEventInstance,
  deleteGroupPollVotesForUser,
  deleteMemberRelation,
  deleteTemplate,
  fetchArchivedEvents,
  fetchAuthConfig,
  fetchAuthMe,
  fetchEventActivity,
  fetchEventBillingForInstance,
  fetchEventHistory,
  fetchEventPollHistoryForInstance,
  fetchEventSetRowsForInstance,
  fetchEventTeamSplit,
  fetchGameRoster,
  fetchGroupDebtSummary,
  fetchGroupDebtors,
  fetchGroupDetails,
  fetchGroupGames,
  fetchGroupMembers,
  fetchGroupPermissions,
  fetchGroupPollOptions,
  fetchGroupPolls,
  fetchGroupPollVotes,
  fetchGroupRoles,
  fetchGroups,
  fetchMemberRelations,
  fetchMemberSkills,
  fetchMyGroupProfile,
  fetchSkillsCatalog,
  fetchTemplate,
  generateEventBillingForInstance,
  logoutAuth,
  publishEventSetRowsForInstance,
  publishEventTeamSplit,
  publishRegistration,
  saveEventBillingForInstance,
  saveEventSetRowsForInstance,
  saveEventTeamSplit,
  telegramAuthLogin,
  unarchiveEvent,
  updateEventCost,
  updateEventDetails,
  updateMemberProfile,
  updateMemberSkills,
  updateTemplate,
  upsertMemberRelation,
} from './api'
import { announcementLeadLabel, announcementLeadOptions, clockMinutes, ensureList, formatMoney, toHourMinute, weekdayLabel, weekdayOptions } from './app/constants'
import { sections } from './app/navigation'
import { buildRoutePath, makeOrgKey, parseRoute, type RouteState, type Section } from './app/router'
import {
  activeTeamCodes,
  formatDateTime,
  historyStatusLabel,
  playerDisplayName,
  teamColor,
  type ActiveTeamCode,
} from './features/history/historyUtils'
import {
  averageScore,
  fullName,
  initialsFromProfile,
  memberRating,
  normalizedSkillScore,
  playerTypeCards,
  playerTypeLabel,
  profileSkills,
} from './features/members/memberUtils'
import { buildTemplatePayload, ensureAtLeastTwoOptions, ensureAtLeastTwoTemplateState, type TemplateFormState } from './features/templates/templateUtils'
import type {
  EventActivitySummary,
  EventHistoryItem,
  EventBilling,
  EventPollHistoryItem,
  EventTeamSplitState,
  EventView,
  GameRosterResponse,
  Group,
  GroupDebtSummary,
  GroupDetails,
  GroupGameRow,
  GroupPollItem,
  GroupPollOptionItem,
  GroupPollVoteItem,
  GroupMember,
  GroupDebtor,
  GroupPermissionsView,
  GroupRoleView,
  MemberSkillProfile,
  PlayerRelation,
  SkillCatalogItem,
  AuthConfig,
  AuthUser,
  UserGroupProfile,
} from './types'

const initialRoute = parseRoute(window.location.pathname)

declare global {
  interface Window {
    onTelegramAuth?: (payload: {
      id: number
      first_name: string
      last_name?: string
      username?: string
      photo_url?: string
      auth_date: number
      hash: string
    }) => void
  }
}

function clampInt(value: unknown, min: number, max: number): number {
  const n = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(n)) return min
  return Math.max(min, Math.min(max, Math.trunc(n)))
}

export default function App() {
  const [productTheme, chooseProductTheme] = useProductTheme()
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null)
  const [authUser, setAuthUser] = useState<AuthUser | null>(null)
  const [authLoading, setAuthLoading] = useState(true)
  const [loginWidgetError, setLoginWidgetError] = useState(false)
  const [loginWidgetAttempt, setLoginWidgetAttempt] = useState(0)
  const [groups, setGroups] = useState<Group[]>([])
  const [groupsLoaded, setGroupsLoaded] = useState(false)
  const [activeChatID, setActiveChatID] = useState<number | null>(initialRoute.chatID)
  const [activeOrgKey, setActiveOrgKey] = useState<string | null>(initialRoute.orgKey ?? null)
  const [activeSection, setActiveSection] = useState<Section>(initialRoute.section)
  const [activeMemberID, setActiveMemberID] = useState<number | null>(initialRoute.memberID ?? null)
  const [activeHistoryEventID, setActiveHistoryEventID] = useState<number | null>(initialRoute.historyEventID ?? null)
  const [activePollPostID, setActivePollPostID] = useState<number | null>(initialRoute.pollPostID ?? null)
  const [activeTemplateView, setActiveTemplateView] = useState<'list' | 'create' | 'edit'>(initialRoute.templateView)
  const [activeTemplateName, setActiveTemplateName] = useState<string | null>(initialRoute.templateName)
  const [activeEventView, setActiveEventView] = useState<'list' | 'create' | 'edit'>(initialRoute.eventView)
  const [activeEventID, setActiveEventID] = useState<number | null>(initialRoute.eventID)
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    try { return localStorage.getItem('teamtime-sidebar-collapsed') === 'true' } catch { return false }
  })

  const [details, setDetails] = useState<GroupDetails | null>(null)
  const [members, setMembers] = useState<GroupMember[]>([])
  const [membersError, setMembersError] = useErrorState()
  const [skillsCatalog, setSkillsCatalog] = useState<SkillCatalogItem[]>([])
  const [memberSkillProfiles, setMemberSkillProfiles] = useState<Record<number, MemberSkillProfile>>({})
  const [selectedMemberSkills, setSelectedMemberSkills] = useState<MemberSkillProfile | null>(null)
  const [memberSkillDraft, setMemberSkillDraft] = useState<Record<string, string>>({})
  const [memberPlayerTypeDraft, setMemberPlayerTypeDraft] = useState<'' | 'attacker' | 'setter' | 'libero' | 'central'>('')
  const [memberRealNameDraft, setMemberRealNameDraft] = useState('')
  const [memberRelations, setMemberRelations] = useState<PlayerRelation[]>([])
  const [memberRelationDraft, setMemberRelationDraft] = useState({
    otherUserID: '',
    relationType: 'prefer_together' as 'prefer_together' | 'avoid_together',
    weight: '5',
  })
  const [membersSearch, setMembersSearch] = useState('')
  const [membersView, setMembersView] = useState<'cards' | 'list'>('cards')
  const [membersPage, setMembersPage] = useState(1)
  const [membersPageSize, setMembersPageSize] = useState(12)
  const membersHeadingRef = useRef<HTMLDivElement>(null)
  const [eventsView, setEventsView] = useState<'cards' | 'list'>(() => {
    try { return localStorage.getItem('teamtime-events-view') === 'list' ? 'list' : 'cards' }
    catch { return 'cards' }
  })
  useEffect(() => {
    try { localStorage.setItem('teamtime-events-view', eventsView) } catch { /* View switching also works without storage. */ }
  }, [eventsView])
  const [membersTypeFilter, setMembersTypeFilter] = useState<'' | 'attacker' | 'setter' | 'libero' | 'central'>('')
  const [membersRatingMin, setMembersRatingMin] = useState('')
  const [membersRatingMax, setMembersRatingMax] = useState('')
  const [membersSortCriteria, setMembersSortCriteria] = useState<MembersSortCriterion[]>([])
  const [memberSkillLoading, setMemberSkillLoading] = useState(false)
  const [memberSkillError, setMemberSkillError] = useErrorState()
  const [loading, setLoading] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useErrorState()
  const [success, setSuccess] = useState('')
  const [successVisible, setSuccessVisible] = useState(false)

  const [templateForm, setTemplateForm] = useState<TemplateFormState>({
    name: '',
    question: '',
    options: ['', ''],
    counted: [false, false],
    weights: [1, 1],
  })
  const [eventForm, setEventForm] = useState({
    name: '',
    eventType: 'training' as 'training' | 'activity',
    templateName: '',
    weekday: '5',
    publishWeekday: '5',
    publishAt: '10:00',
    startAt: '19:00',
    endAt: '21:00',
    announcementText: '',
    mentions: emptyEventMentions(),
    announcementEnabled: false,
    announcementLeadMinutes: '60',
    teamsAutoSplit: false,
    teamsPublishList: false,
    teamSize: '6',
    maxPlaces: '18',
    minVotesToHold: '0',
    cancelLeadMinutes: '180',
    cancelNotifyEnabled: false,
    settlementEnabled: false,
    settlementPublishBefore: false,
    settlementPublishAfter: true,
    costAmount: '',
  })

  const [templateEditor, setTemplateEditor] = useState<TemplateFormState>({
    name: '',
    question: '',
    options: ['', ''],
    counted: [false, false],
    weights: [1, 1],
  })
  const [templateEditorLoading, setTemplateEditorLoading] = useState(false)
  const [templateEditorError, setTemplateEditorError] = useErrorState()
  const [eventEditor, setEventEditor] = useState({
    name: '',
    eventType: 'training' as 'training' | 'activity',
    weekday: '5',
    publishWeekday: '5',
    publishAt: '10:00',
    startAt: '19:00',
    endAt: '21:00',
    announcementText: '',
    mentions: emptyEventMentions(),
    announcementEnabled: false,
    announcementLeadMinutes: '60',
    teamsAutoSplit: false,
    teamsPublishList: false,
    teamSize: '6',
    maxPlaces: '18',
    minVotesToHold: '0',
    cancelLeadMinutes: '180',
    cancelNotifyEnabled: false,
    settlementEnabled: false,
    settlementPublishBefore: false,
    settlementPublishAfter: true,
    costAmount: '',
    templateName: '',
  })
  const [eventsMode, setEventsMode] = useState<'active' | 'archived'>('active')
  const [eventsTypeFilter, setEventsTypeFilter] = useState<'' | 'training' | 'activity'>('')
  const [archivedEvents, setArchivedEvents] = useState<EventView[]>([])
  const [eventActivity, setEventActivity] = useState<EventActivitySummary | null>(null)
  const [eventActivityLoading, setEventActivityLoading] = useState(false)
  const [eventActivityError, setEventActivityError] = useErrorState()
  const [showEventActivity, setShowEventActivity] = useState(false)
  const [eventHistory, setEventHistory] = useState<EventHistoryItem[]>([])
  const [historyPage, setHistoryPage] = useState(1)
  const [historyPageSize, setHistoryPageSize] = useState(12)
  const historyHeadingRef = useRef<HTMLDivElement>(null)
  const [historyStatusFilter, setHistoryStatusFilter] = useState<'' | 'in_voting' | 'on_distribution' | 'on_review' | 'completed' | 'not_held'>('')
  const [attendanceRevision, setAttendanceRevision] = useState(0)
  const [historyDetailTab, setHistoryDetailTab] = useState<'distribution' | 'votes' | 'billing' | 'sets' | 'attendance'>('distribution')
  const [eventHistoryLoading, setEventHistoryLoading] = useState(false)
  const [eventHistoryError, setEventHistoryError] = useErrorState()
  const [eventPollHistory, setEventPollHistory] = useState<EventPollHistoryItem[]>([])
  const [groupPolls, setGroupPolls] = useState<GroupPollItem[]>([])
  const [groupPollsLoading, setGroupPollsLoading] = useState(false)
  const [groupPollsError, setGroupPollsError] = useErrorState()
  const [groupPollVotes, setGroupPollVotes] = useState<GroupPollVoteItem[]>([])
  const [groupPollVotesLoading, setGroupPollVotesLoading] = useState(false)
  const [groupPollVotesError, setGroupPollVotesError] = useErrorState()
  const [pollOptions, setPollOptions] = useState<GroupPollOptionItem[]>([])
  const [pollOptionsLoading, setPollOptionsLoading] = useState(false)
  const [pollOptionsError, setPollOptionsError] = useErrorState()
  const [manualVoteDraft, setManualVoteDraft] = useState({
    userID: '',
    choice: '',
  })
  const [eventPollHistoryLoading, setEventPollHistoryLoading] = useState(false)
  const [eventPollHistoryError, setEventPollHistoryError] = useErrorState()
  const [selectedHistoryPostID, setSelectedHistoryPostID] = useState<number | null>(null)
  const [historyPollVotes, setHistoryPollVotes] = useState<GroupPollVoteItem[]>([])
  const [historyPollVotesLoading, setHistoryPollVotesLoading] = useState(false)
  const [historyPollVotesError, setHistoryPollVotesError] = useErrorState()
  const [teamSplit, setTeamSplit] = useState<EventTeamSplitState | null>(null)
  const [teamSplitLoading, setTeamSplitLoading] = useState(false)
  const [, setTeamSplitError] = useErrorState()
  const [draggedPlayerID, setDraggedPlayerID] = useState<number | null>(null)
  const [touchDragMode, setTouchDragMode] = useState(false)
  const [teamCEnabled, setTeamCEnabled] = useState(false)
  const [eventBilling, setEventBilling] = useState<EventBilling | null>(null)
  const [eventBillingLoading, setEventBillingLoading] = useState(false)
  const [eventBillingError, setEventBillingError] = useErrorState()
  const [eventBillingDraft, setEventBillingDraft] = useState<Record<number, boolean>>({})
  const [eventSetRowsDraft, setEventSetRowsDraft] = useState<SetRowDraft[]>([])
  const [eventSetsLoading, setEventSetsLoading] = useState(false)
  const [, setEventSetsError] = useErrorState()
  const [groupDebtSummary, setGroupDebtSummary] = useState<GroupDebtSummary | null>(null)
  const [billingDebtors, setBillingDebtors] = useState<GroupDebtor[]>([])
  const [billingLoading, setBillingLoading] = useState(false)
  const [billingError, setBillingError] = useErrorState()
  const [groupGames, setGroupGames] = useState<GroupGameRow[]>([])
  const [groupGamesLoading, setGroupGamesLoading] = useState(false)
  const [groupGamesError, setGroupGamesError] = useErrorState()
  const [gamesExpandedRows, setGamesExpandedRows] = useState<Record<string, boolean>>({})
  const [gamesRosterCache, setGamesRosterCache] = useState<Record<string, GameRosterResponse>>({})
  const [myProfile, setMyProfile] = useState<UserGroupProfile | null>(null)
  const [myProfileLoading, setMyProfileLoading] = useState(false)
  const [myProfileError, setMyProfileError] = useErrorState()
  const [confirmModal, setConfirmModal] = useState<{
    open: boolean
    title: string
    message: string
    confirmLabel: string
    cancelLabel: string
    danger: boolean
    onConfirm: null | (() => Promise<void>)
  }>({
    open: false,
    title: '',
    message: '',
    confirmLabel: 'Подтвердить',
    cancelLabel: 'Отмена',
    danger: false,
    onConfirm: null,
  })
  const [confirmBusy, setConfirmBusy] = useState(false)
  const [activePerms, setActivePerms] = useState<GroupPermissionsView | null>(null)
  const [groupRoles, setGroupRoles] = useState<GroupRoleView[]>([])
  const [groupRolesLoading, setGroupRolesLoading] = useState(false)
  const [, setGroupRolesError] = useErrorState()
  const [roleAssignUserID, setRoleAssignUserID] = useState('')
  const [roleAssignCode, setRoleAssignCode] = useState('member')

  const templateNames = useMemo(() => ensureList(details?.templateNames), [details])
  const templates = useMemo(() => ensureList(details?.templates), [details])
  const events = useMemo(() => ensureList(details?.events), [details])

  const fullAccessPerms = useMemo<GroupPermissionsView>(
    () => ({
      roleCode: 'admin',
      roleTitle: 'Администратор',
      permissions: {
        members_read: true,
        members_write: true,
        templates_manage: true,
        event_templates_manage: true,
        events_read: true,
        events_manage: true,
        polls_read: true,
        roles_manage: true,
        profile_read: true,
      },
    }),
    [],
  )

  const isAdmin = useMemo(() => {
    if (!authConfig?.enabled) {
      return true
    }
    return activePerms?.roleCode === 'admin'
  }, [authConfig?.enabled, activePerms?.roleCode])

  function can(perm: string): boolean {
    if (!authConfig?.enabled) {
      return true
    }
    return Boolean(activePerms?.permissions?.[perm])
  }

  const visibleSections = useMemo(() => {
    return sections.filter((s) => {
      if (s.id === 'overview') return isAdmin
      if (s.id === 'settings') return isAdmin || can('templates_manage') || can('event_templates_manage')
      if (s.id === 'billing') return isAdmin
      if (s.id === 'docs') return true
      if (s.id === 'events') return can('events_read')
      if (s.id === 'games') return can('events_read')
      if (s.id === 'polls') return can('polls_read')
      if (s.id === 'profile') return true
      if (s.id === 'members') return can('members_read')
      if (s.id === 'templates') return can('templates_manage')
      if (s.id === 'event_templates') return can('event_templates_manage')
      return false
    }).sort((a, b) => ['overview','events','members','games','polls','billing','templates','event_templates','profile','settings','docs'].indexOf(a.id) - ['overview','events','members','games','polls','billing','templates','event_templates','profile','settings','docs'].indexOf(b.id))
  }, [authConfig?.enabled, activePerms, isAdmin])
  const availableTemplateSections = visibleSections.filter(section => isTemplateSection(section.id)).map(section => section.id)
  const headerSectionIDs: Section[] = ['settings', 'profile']
  const headerSections = visibleSections.filter(section => headerSectionIDs.includes(section.id))
    .sort((a, b) => headerSectionIDs.indexOf(a.id) - headerSectionIDs.indexOf(b.id))
  const sidebarSections = sections.filter(section => section.id !== 'polls' && !isTemplateSection(section.id) && !headerSectionIDs.includes(section.id) && visibleSections.some(visible => visible.id === section.id))
    .sort((a, b) => ['overview', 'events', 'members', 'games', 'billing', 'docs'].indexOf(a.id)
      - ['overview', 'events', 'members', 'games', 'billing', 'docs'].indexOf(b.id))
  const activeSidebarSection = activeSection === 'polls' ? 'events' : activeSection
  const activeHeaderSection = isTemplateSection(activeSection) ? 'settings' : activeSection
  const templateCountedMap = useMemo(() => {
    const map = new Map<string, number>()
    for (const template of templates) {
      map.set(template.name, template.countedOptionsCount ?? 0)
    }
    return map
  }, [templates])

  const selectedEvent = useMemo(
    () => events.find((event) => event.id === activeEventID) ?? null,
    [events, activeEventID],
  )
  const selectedGroupPoll = useMemo(
    () => groupPolls.find((poll) => poll.postID === activePollPostID) ?? null,
    [groupPolls, activePollPostID],
  )
  const currentVotePostID = useMemo(() => {
    if (activeSection === 'polls') {
      return activePollPostID
    }
    if (activeSection === 'events') {
      return selectedHistoryPostID
    }
    return null
  }, [activePollPostID, activeSection, selectedHistoryPostID])
  const displayedEvents = useMemo(() => {
    const source = eventsMode === 'active' ? events : ensureList(archivedEvents)
    if (eventsTypeFilter === '') {
      return source
    }
    return source.filter((event) => event.eventType === eventsTypeFilter)
  }, [archivedEvents, events, eventsMode, eventsTypeFilter])
  const filteredEventHistory = useMemo(() => {
    if (historyStatusFilter === '') {
      return eventHistory
    }
    return eventHistory.filter((item) => item.status === historyStatusFilter)
  }, [eventHistory, historyStatusFilter])
  const historyPageCount = Math.max(1, Math.ceil(filteredEventHistory.length / historyPageSize))
  const currentHistoryPage = Math.min(historyPage, historyPageCount)
  const paginatedEventHistory = filteredEventHistory.slice((currentHistoryPage - 1) * historyPageSize, currentHistoryPage * historyPageSize)
  useEffect(() => { setHistoryPage(1) }, [activeChatID, historyStatusFilter, historyPageSize])
  useEffect(() => { setHistoryPage(page => Math.min(page, historyPageCount)) }, [historyPageCount])
  const membersRatingError = [membersRatingMin, membersRatingMax].some(value => value !== '' && (!Number.isFinite(Number(value)) || Number(value) < 1 || Number(value) > 10))
    ? 'Укажите рейтинг от 1 до 10.'
    : membersRatingMin !== '' && membersRatingMax !== '' && Number(membersRatingMin) > Number(membersRatingMax)
      ? 'Рейтинг «от» должен быть не больше рейтинга «до».' : ''
  const filteredMembers = useMemo(() => {
    const query = membersSearch.trim().toLowerCase()
    const list = members.filter((member) => {
      const profile = memberSkillProfiles[member.userTelegramID]
      const playerType = (profile?.playerType || member.playerType || '') as '' | 'attacker' | 'setter' | 'libero' | 'central'
      if (membersTypeFilter !== '' && playerType !== membersTypeFilter) {
        return false
      }
      if (membersRatingError) return false
      if (membersRatingMin !== '' || membersRatingMax !== '') {
        const rating = memberRating(profile ?? null, skillsCatalog)
        if (rating === null || (membersRatingMin !== '' && rating < Number(membersRatingMin)) || (membersRatingMax !== '' && rating > Number(membersRatingMax))) return false
      }
      if (query === '') {
        return true
      }
      const name = fullName(member).toLowerCase()
      const realName = (profile?.realName || member.realName || '').toLowerCase()
      const username = (member.username || '').toLowerCase()
      const usernameWithAt = username ? `@${username}` : ''
      const id = String(member.userTelegramID)
      return (
        name.includes(query) ||
        realName.includes(query) ||
        username.includes(query) ||
        usernameWithAt.includes(query) ||
        id.includes(query)
      )
    })
    if (membersSortCriteria.length === 0) {
      return list
    }

    const knownSkillCodes = new Set(skillsCatalog.map((skill) => skill.code))
    const valueCache = new Map<string, number>()
    const getSortValue = (member: GroupMember, code: string): number => {
      const cacheKey = `${member.userTelegramID}:${code}`
      const cached = valueCache.get(cacheKey)
      if (cached !== undefined) {
        return cached
      }
      const profile = memberSkillProfiles[member.userTelegramID] ?? null
      let value: number
      if (code === MEMBERS_SORT_AVERAGE_CODE) {
        const avg = memberRating(profile, skillsCatalog)
        value = avg === null ? -1 : avg
      } else if (knownSkillCodes.has(code)) {
        const skill = profileSkills(profile, skillsCatalog).find((item) => item.skillCode === code)
        value = typeof skill?.score === 'number' && Number.isFinite(skill.score) ? normalizedSkillScore(skill.score) : -1
      } else {
        value = -1
      }
      valueCache.set(cacheKey, value)
      return value
    }

    return [...list].sort((left, right) => {
      for (const criterion of membersSortCriteria) {
        const leftValue = getSortValue(left, criterion.code)
        const rightValue = getSortValue(right, criterion.code)
        if (leftValue !== rightValue) {
          if (leftValue < 0) return 1
          if (rightValue < 0) return -1
          return criterion.direction === 'asc' ? leftValue - rightValue : rightValue - leftValue
        }
      }
      const nameDiff = fullName(left).localeCompare(fullName(right), 'ru')
      if (nameDiff !== 0) {
        return nameDiff
      }
      return left.userTelegramID - right.userTelegramID
    })
  }, [
    memberSkillProfiles,
    members,
    membersSearch,
    membersSortCriteria,
    membersTypeFilter,
    membersRatingMin,
    membersRatingMax,
    membersRatingError,
    skillsCatalog,
  ])
  const membersPageCount = Math.max(1, Math.ceil(filteredMembers.length / membersPageSize))
  const currentMembersPage = Math.min(membersPage, membersPageCount)
  const paginatedMembers = filteredMembers.slice((currentMembersPage - 1) * membersPageSize, currentMembersPage * membersPageSize)
  useEffect(() => { setMembersPage(1) }, [activeChatID, membersSearch, membersTypeFilter, membersRatingMin, membersRatingMax, membersPageSize])
  useEffect(() => { setMembersPage(page => Math.min(page, membersPageCount)) }, [membersPageCount])
  const selectedHistoryEvent = useMemo(
    () => eventHistory.find((event) => event.instanceID === activeHistoryEventID) ?? null,
    [eventHistory, activeHistoryEventID],
  )

  useEffect(() => {
    const knownSkillCodes = new Set(skillsCatalog.map((skill) => skill.code))
    setMembersSortCriteria((prev) => {
      const next = prev.filter((item) => item.code === MEMBERS_SORT_AVERAGE_CODE || knownSkillCodes.has(item.code))
      return next.length === prev.length ? prev : next
    })
  }, [skillsCatalog])

  function onMembersSort(code: string, direction: 'asc' | 'desc') {
    setMembersPage(1)
    setMembersSortCriteria((prev) => {
      const idx = prev.findIndex((item) => item.code === code)
      if (idx === -1) {
        return [...prev, { code, direction }]
      }
      if (prev[idx].direction === direction) {
        return [...prev.slice(0, idx), ...prev.slice(idx + 1)]
      }
      const next = [...prev]
      next[idx] = { code, direction }
      return next
    })
  }

  useEffect(() => {
    if (activeChatID === null) {
      setActivePerms(null)
      return
    }
    if (!authConfig?.enabled) {
      setActivePerms(fullAccessPerms)
      return
    }
    if (!authUser) {
      setActivePerms(null)
      return
    }
    void (async () => {
      try {
        const view = await fetchGroupPermissions(activeChatID)
        setActivePerms(view)
      } catch (err) {
        setActivePerms(null)
      }
    })()
  }, [activeChatID, authConfig?.enabled, authUser, fullAccessPerms])

  useEffect(() => {
    if (activeSection !== 'profile') {
      return
    }
    if (activeChatID === null) {
      return
    }
    if (!authConfig?.enabled) {
      // local mode: show roles list as defaults (server returns full access anyway)
      return
    }
    if (!activePerms?.permissions?.roles_manage) {
      return
    }
    void (async () => {
      try {
        setGroupRolesError('')
        setGroupRolesLoading(true)
        const roles = await fetchGroupRoles(activeChatID)
        setGroupRoles(roles)
      } catch (err: any) {
        setGroupRolesError(err?.message || 'Ошибка загрузки ролей')
      } finally {
        setGroupRolesLoading(false)
      }
    })()
    if (members.length === 0) {
      void (async () => {
        try {
          const nextMembers = await fetchGroupMembers(activeChatID)
          setMembers(nextMembers)
        } catch {
          // ignore: roles UI will show empty dropdown if we can't read members
        }
      })()
    }
  }, [activeChatID, activePerms, activeSection, authConfig?.enabled, members.length])
  const memberByUserID = useMemo(() => {
    const map = new Map<number, GroupMember>()
    for (const member of members) {
      map.set(member.userTelegramID, member)
    }
    return map
  }, [members])

  const availableRelationMembers = useMemo(() => {
    if (!selectedMemberSkills) {
      return []
    }
    return members
      .filter((member) => member.userTelegramID !== selectedMemberSkills.userTelegramID)
      .sort((a, b) => fullName(a).localeCompare(fullName(b), 'ru'))
  }, [members, selectedMemberSkills])

  const relationMemberOptions = useMemo(() => {
    return availableRelationMembers.map((member) => {
      return {
        id: member.userTelegramID,
        label: (memberSkillProfiles[member.userTelegramID]?.realName || member.realName || '').trim() || fullName(member),
        username: (member.username || '').trim(),
      }
    })
  }, [availableRelationMembers, memberSkillProfiles])

  const eventEditorDirty = useMemo(() => {
    if (!selectedEvent || activeSection !== 'event_templates' || activeEventView !== 'edit') {
      return false
    }

    const selectedPublishWeekday = selectedEvent.pollPublishWeekday || selectedEvent.startWeekday
    const selectedAnnouncementLead = selectedEvent.announcementLeadMinutes || 60
    const currentCostValue = eventEditor.costAmount.trim()
    const parsedCurrentCost = currentCostValue === '' ? undefined : Number(currentCostValue)
    const selectedCost = selectedEvent.costAmount
    const costChanged =
      (parsedCurrentCost === undefined && selectedCost !== undefined) ||
      (parsedCurrentCost !== undefined && selectedCost === undefined) ||
      (parsedCurrentCost !== undefined && selectedCost !== undefined && Math.abs(parsedCurrentCost - selectedCost) > 0.000001)

    return (
      eventEditor.name.trim() !== selectedEvent.name ||
      eventEditor.eventType !== (selectedEvent.eventType || 'training') ||
      Number(eventEditor.weekday) !== selectedEvent.startWeekday ||
      Number(eventEditor.publishWeekday) !== selectedPublishWeekday ||
      eventEditor.publishAt !== toHourMinute(selectedEvent.pollPublishTime) ||
      eventEditor.startAt !== toHourMinute(selectedEvent.startTime) ||
      eventEditor.endAt !== toHourMinute(selectedEvent.endTime) ||
      !equalEventMentions(eventEditor.mentions, selectedEvent.mentions) ||
      eventEditor.announcementText.trim() !== (selectedEvent.announcementText || '') ||
      eventEditor.announcementEnabled !== selectedEvent.announcementEnabled ||
      Number(eventEditor.announcementLeadMinutes) !== selectedAnnouncementLead ||
      eventEditor.teamsAutoSplit !== selectedEvent.teamsAutoSplit ||
      eventEditor.teamsPublishList !== selectedEvent.teamsPublishList ||
      Number(eventEditor.teamSize) !== (selectedEvent.teamSize || 6) ||
      Number(eventEditor.maxPlaces) !== (selectedEvent.maxPlaces || 18) ||
      Number(eventEditor.minVotesToHold) !== (selectedEvent.minVotesToHold || 0) ||
      Number(eventEditor.cancelLeadMinutes) !== (selectedEvent.cancelLeadMinutes || 180) ||
      eventEditor.cancelNotifyEnabled !== (selectedEvent.cancelNotifyEnabled || false) ||
      eventEditor.settlementEnabled !== selectedEvent.settlementEnabled ||
      eventEditor.settlementPublishBefore !== selectedEvent.settlementPublishBefore ||
      eventEditor.settlementPublishAfter !== selectedEvent.settlementPublishAfter ||
      eventEditor.templateName !== (selectedEvent.pollTemplate || '') ||
      costChanged
    )
  }, [activeEventView, activeSection, eventEditor, selectedEvent])

  const eventEditorCanEnableSettlement = useMemo(() => {
    const templateName = eventEditor.templateName.trim()
    if (!templateName) {
      return false
    }
    return (templateCountedMap.get(templateName) ?? 0) > 0
  }, [eventEditor.templateName, templateCountedMap])
  const eventEditorHasTemplate = useMemo(() => eventEditor.templateName.trim() !== '', [eventEditor.templateName])

  function navigateTo(route: RouteState, replace = false) {
    const resolvedOrgKey =
      route.orgKey ??
      (route.chatID !== null ? makeOrgKey(groups.find((group) => group.chatID === route.chatID)?.title || '', route.chatID) : activeOrgKey)

    setActiveChatID(route.chatID)
    setActiveOrgKey(resolvedOrgKey ?? null)
    setActiveSection(route.section)
    setActiveMemberID(route.section === 'members' ? route.memberID ?? null : null)
    setActiveHistoryEventID(route.section === 'events' ? route.historyEventID ?? null : null)
    setActivePollPostID(route.section === 'polls' ? route.pollPostID ?? null : null)
    setActiveTemplateView(route.section === 'templates' ? route.templateView : 'list')
    setActiveTemplateName(route.section === 'templates' ? route.templateName : null)
    setActiveEventView(route.section === 'event_templates' ? route.eventView : 'list')
    setActiveEventID(route.section === 'event_templates' ? route.eventID : null)

    const nextPath = buildRoutePath({ ...route, orgKey: resolvedOrgKey })
    const stateOp = replace ? window.history.replaceState : window.history.pushState
    stateOp.call(window.history, {}, '', nextPath)
  }

  useEffect(() => {
    const onPopState = () => {
      const route = parseRoute(window.location.pathname)
      setActiveChatID(route.chatID)
      setActiveOrgKey(route.orgKey ?? null)
      setActiveSection(route.section)
      setActiveMemberID(route.section === 'members' ? route.memberID ?? null : null)
      setActiveHistoryEventID(route.section === 'events' ? route.historyEventID ?? null : null)
      setActivePollPostID(route.section === 'polls' ? route.pollPostID ?? null : null)
      setActiveTemplateView(route.templateView)
      setActiveTemplateName(route.templateName)
      setActiveEventView(route.eventView)
      setActiveEventID(route.eventID)
      setSuccess('')
      setError('')
    }

    window.addEventListener('popstate', onPopState)
    return () => window.removeEventListener('popstate', onPopState)
  }, [])

  useEffect(() => {
    void (async () => {
      setAuthLoading(true)
      try {
        const cfg = await fetchAuthConfig()
        setAuthConfig(cfg)
        if (cfg.enabled) {
          try {
            const me = await fetchAuthMe()
            setAuthUser(me.user)
          } catch {
            setAuthUser(null)
          }
        } else {
          setAuthUser({
            id: 0,
            username: '',
            firstName: 'Local',
            lastName: '',
            authDate: Math.floor(Date.now() / 1000),
          })
        }
      } catch (err) {
        setError((err as Error).message)
      } finally {
        setAuthLoading(false)
      }
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (authLoading || !authUser) {
      setGroupsLoaded(false)
      setGroups([])
      return
    }
    void (async () => {
      try {
        const loadedGroups = await fetchGroups()
        setGroups(loadedGroups)
        setGroupsLoaded(true)

        if (loadedGroups.length === 0) {
          return
        }

        let resolvedChatID = activeChatID
        if (resolvedChatID === null && activeOrgKey) {
          const byKey = loadedGroups.find((group) => makeOrgKey(group.title, group.chatID) === activeOrgKey)
          if (byKey) {
            resolvedChatID = byKey.chatID
            setActiveChatID(byKey.chatID)
            setActiveOrgKey(makeOrgKey(byKey.title, byKey.chatID))
          }
        }

        const hasActive = resolvedChatID !== null && loadedGroups.some((group) => group.chatID === resolvedChatID)
        if (!hasActive) {
          const first = loadedGroups[0]
          navigateTo(
            {
              chatID: first.chatID,
              orgKey: makeOrgKey(first.title, first.chatID),
              section: activeSection,
              memberID: activeSection === 'members' ? activeMemberID : null,
              historyEventID: activeSection === 'events' ? activeHistoryEventID : null,
              pollPostID: activeSection === 'polls' ? activePollPostID : null,
              templateView: activeSection === 'templates' ? activeTemplateView : 'list',
              templateName: activeSection === 'templates' ? activeTemplateName : null,
              eventView: activeSection === 'event_templates' ? activeEventView : 'list',
              eventID: activeSection === 'event_templates' ? activeEventID : null,
            },
            true,
          )
        }
      } catch (err) {
        setError((err as Error).message)
      }
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeChatID, activeOrgKey, authLoading, authUser])

  useEffect(() => {
    // Wait until we know whether auth is enabled to avoid missing the moment
    // when the login card (and the widget container) appears.
    if (authLoading) {
      return
    }
    if (!authConfig?.enabled || !authConfig.telegramLoginBot || authUser) {
      return
    }

    const container = document.getElementById('telegram-login-widget')
    if (!container) {
      return
    }
    container.innerHTML = ''
    setLoginWidgetError(false)

    window.onTelegramAuth = (payload) => {
      void (async () => {
        try {
          setError('')
          const result = await telegramAuthLogin(payload)
          setAuthUser(result.user)
        } catch (err) {
          setError((err as Error).message)
        }
      })()
    }

    const script = document.createElement('script')
    script.src = 'https://telegram.org/js/telegram-widget.js?22'
    script.async = true
    script.setAttribute('data-telegram-login', authConfig.telegramLoginBot)
    script.setAttribute('data-size', 'large')
    script.setAttribute('data-userpic', 'false')
    script.setAttribute('data-request-access', 'write')
    script.setAttribute('data-onauth', 'onTelegramAuth(user)')
    script.onerror = () => setLoginWidgetError(true)
    script.onload = () => setLoginWidgetError(false)
    container.appendChild(script)
    const widgetTimeout = window.setTimeout(() => {
      if (!container.querySelector('iframe')) setLoginWidgetError(true)
    }, 15000)

    return () => {
      window.clearTimeout(widgetTimeout)
      script.onerror = null
      script.onload = null
      delete window.onTelegramAuth
      container.innerHTML = ''
    }
  }, [authConfig, authUser, authLoading, loginWidgetAttempt])

  useEffect(() => {
    // Keep the requested detail route while authentication and organizations load.
    if (!groupsLoaded) return
    if (activeChatID === null) {
      setDetails(null)
      setMembers([])
      setMemberSkillProfiles({})
      setSelectedMemberSkills(null)
      setEventHistory([])
      setGroupPolls([])
      setGroupPollVotes([])
      setEventBilling(null)
      setGroupDebtSummary(null)
      setBillingDebtors([])
      setActiveHistoryEventID(null)
      setActivePollPostID(null)
      setArchivedEvents([])
      setMyProfile(null)
      return
    }
    if (!groups.some((group) => group.chatID === activeChatID)) {
      setDetails(null)
      setMembers([])
      setMemberSkillProfiles({})
      setSelectedMemberSkills(null)
      setEventHistory([])
      setGroupPolls([])
      setGroupPollVotes([])
      setEventBilling(null)
      setGroupDebtSummary(null)
      setBillingDebtors([])
      setActiveHistoryEventID(null)
      setActivePollPostID(null)
      setArchivedEvents([])
      setMyProfile(null)
      return
    }
    void reloadActiveOrganization(activeChatID)
  }, [activeChatID, groups, groupsLoaded])

  useEffect(() => {
    if (activeSection !== 'profile' || activeChatID === null || !authConfig?.enabled || !activePerms?.permissions?.profile_read) {
      setMyProfile(null)
      setMyProfileError('')
      return
    }
    let cancelled = false
    setMyProfileLoading(true)
    setMyProfileError('')
    void fetchMyGroupProfile(activeChatID)
      .then((profile) => {
        if (!cancelled) {
          setMyProfile(profile)
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setMyProfile(null)
          setMyProfileError((err as Error).message)
        }
      })
      .finally(() => {
        if (!cancelled) {
          setMyProfileLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, success, authConfig?.enabled, activePerms?.permissions?.profile_read])

  useEffect(() => {
    if (activeSection !== 'event_templates') {
      setEventsMode('active')
    }
  }, [activeSection])

  useEffect(() => {
    if ((activeSection !== 'events' && activeSection !== 'overview') || activeChatID === null) {
      return
    }
    let cancelled = false
    setEventHistoryLoading(true)
    setEventHistoryError('')
    void fetchEventHistory(activeChatID)
      .then((items) => {
        if (cancelled) {
          return
        }
        setEventHistory(items)
        if (items.length === 0) {
          setActiveHistoryEventID(null)
          return
        }
        if (activeHistoryEventID !== null && !items.some((item) => item.instanceID === activeHistoryEventID)) {
          setActiveHistoryEventID(items[0].instanceID)
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setEventHistory([])
          setEventHistoryError((err as Error).message)
        }
      })
      .finally(() => {
        if (!cancelled) {
          setEventHistoryLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, success])

  useEffect(() => {
    if (activeSection !== 'templates' || activeTemplateView !== 'edit' || !activeTemplateName || activeChatID === null) {
      setTemplateEditor({ name: '', question: '', options: ['', ''], counted: [false, false], weights: [1, 1] })
      setTemplateEditorError('')
      return
    }

    let cancelled = false
    setTemplateEditorLoading(true)
    setTemplateEditorError('')

    void fetchTemplate(activeChatID, activeTemplateName)
      .then((template) => {
        if (cancelled) {
          return
        }
        const preparedOptions = ensureAtLeastTwoOptions(template.options)
        const preparedWeights = (() => {
          const w = Array.isArray(template.optionWeights) ? [...template.optionWeights] : []
          while (w.length < preparedOptions.length) w.push(1)
          if (w.length > preparedOptions.length) return w.slice(0, preparedOptions.length)
          return w.map((x) => (Number.isFinite(x) && x > 0 ? Math.floor(x) : 1))
        })()
        setTemplateEditor({
          name: template.name,
          question: template.question,
          options: preparedOptions,
          counted: preparedOptions.map((_, idx) => template.countedOptions.includes(idx)),
          weights: preparedWeights,
        })
      })
      .catch((err) => {
        if (cancelled) {
          return
        }
        setTemplateEditorError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) {
          setTemplateEditorLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [activeChatID, activeSection, activeTemplateView, activeTemplateName])

  useEffect(() => {
    if (activeSection !== 'event_templates' || activeEventView !== 'edit' || !selectedEvent) {
      setEventEditor({
        name: '',
        eventType: 'training',
        weekday: '5',
        publishWeekday: '5',
        publishAt: '10:00',
        startAt: '19:00',
        endAt: '21:00',
        announcementText: '',
        mentions: emptyEventMentions(),
        announcementEnabled: false,
        announcementLeadMinutes: '60',
        teamsAutoSplit: false,
        teamsPublishList: false,
        teamSize: '6',
        maxPlaces: '18',
        minVotesToHold: '0',
        cancelLeadMinutes: '180',
        cancelNotifyEnabled: false,
        settlementEnabled: false,
        settlementPublishBefore: false,
        settlementPublishAfter: true,
        costAmount: '',
        templateName: '',
      })
      return
    }

    setEventEditor({
      name: selectedEvent.name,
      eventType: selectedEvent.eventType || 'training',
      weekday: String(selectedEvent.startWeekday),
      publishWeekday: String(selectedEvent.pollPublishWeekday || selectedEvent.startWeekday),
      publishAt: toHourMinute(selectedEvent.pollPublishTime),
      startAt: toHourMinute(selectedEvent.startTime),
      endAt: toHourMinute(selectedEvent.endTime),
      announcementText: selectedEvent.announcementText || '',
      mentions: selectedEvent.mentions ?? emptyEventMentions(),
      announcementEnabled: selectedEvent.announcementEnabled,
      announcementLeadMinutes: String(selectedEvent.announcementLeadMinutes || 60),
      teamsAutoSplit: selectedEvent.teamsAutoSplit,
      teamsPublishList: selectedEvent.teamsPublishList,
      teamSize: String(selectedEvent.teamSize || 6),
      maxPlaces: String(selectedEvent.maxPlaces || 18),
      minVotesToHold: String(selectedEvent.minVotesToHold || 0),
      cancelLeadMinutes: String(selectedEvent.cancelLeadMinutes || 180),
      cancelNotifyEnabled: selectedEvent.cancelNotifyEnabled || false,
      settlementEnabled: selectedEvent.settlementEnabled,
      settlementPublishBefore: selectedEvent.settlementPublishBefore,
      settlementPublishAfter: selectedEvent.settlementPublishAfter,
      costAmount: selectedEvent.costAmount === undefined ? '' : String(selectedEvent.costAmount),
      templateName: selectedEvent.pollTemplate || '',
    })
    setShowEventActivity(false)
    setEventActivity(null)
    setEventActivityError('')
    setEventPollHistory([])
    setEventPollHistoryError('')
    setSelectedHistoryPostID(null)
    setTeamSplit(null)
    setTeamSplitError('')
  }, [activeSection, activeEventView, selectedEvent])

  useEffect(() => {
    if (activeSection !== 'event_templates' || activeEventView !== 'edit') {
      return
    }
    if (activeChatID === null || activeEventID === null) {
      return
    }
    if (!details) {
      return
    }
    if (selectedEvent) {
      return
    }
    navigateTo(
      {
        chatID: activeChatID,
        section: 'event_templates',
        templateView: 'list',
        templateName: null,
        eventView: 'list',
        eventID: null,
      },
      true,
    )
  }, [activeSection, activeEventView, activeChatID, activeEventID, details, selectedEvent])

  useEffect(() => {
    if (activeSection !== 'events' || activeChatID === null || activeHistoryEventID === null) {
      return
    }
    let cancelled = false
    setEventPollHistoryLoading(true)
    setEventPollHistoryError('')
    void fetchEventPollHistoryForInstance(activeChatID, activeHistoryEventID)
      .then((items) => {
        if (cancelled) {
          return
        }
        setEventPollHistory(items)
        if (items.length === 0) {
          setSelectedHistoryPostID(null)
          setTeamSplit(null)
          return
        }
        setSelectedHistoryPostID((prev) => (prev && items.some((item) => item.postID === prev) ? prev : items[0].postID))
      })
      .catch((err) => {
        if (cancelled) {
          return
        }
        setEventPollHistory([])
        setEventPollHistoryError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) {
          setEventPollHistoryLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, activeHistoryEventID, success, attendanceRevision])

  useEffect(() => {
    if (activeSection !== 'polls' || activeChatID === null) {
      return
    }
    let cancelled = false
    setGroupPollsLoading(true)
    setGroupPollsError('')
    void fetchGroupPolls(activeChatID)
      .then((items) => {
        if (cancelled) {
          return
        }
        setGroupPolls(items)
        if (activePollPostID !== null && !items.some((item) => item.postID === activePollPostID)) {
          setActivePollPostID(null)
        }
      })
      .catch((err) => {
        if (cancelled) {
          return
        }
        setGroupPolls([])
        setGroupPollsError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) {
          setGroupPollsLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, activePollPostID, success])

  useEffect(() => {
    if (activeSection !== 'polls' || activeChatID === null || activePollPostID === null) {
      setGroupPollVotes([])
      setGroupPollVotesError('')
      setGroupPollVotesLoading(false)
      return
    }
    let cancelled = false
    setGroupPollVotesLoading(true)
    setGroupPollVotesError('')
    void fetchGroupPollVotes(activeChatID, activePollPostID)
      .then((items) => {
        if (cancelled) {
          return
        }
        setGroupPollVotes(items)
      })
      .catch((err) => {
        if (cancelled) {
          return
        }
        setGroupPollVotes([])
        setGroupPollVotesError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) {
          setGroupPollVotesLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, activePollPostID, success])

  useEffect(() => {
    setManualVoteDraft({
      userID: '',
      choice: '',
    })
  }, [currentVotePostID])

  useEffect(() => {
    if ((activeSection !== 'events' && activeSection !== 'polls') || activeChatID === null || currentVotePostID === null) {
      setPollOptions([])
      setPollOptionsError('')
      setPollOptionsLoading(false)
      return
    }
    let cancelled = false
    setPollOptionsLoading(true)
    setPollOptionsError('')
    void fetchGroupPollOptions(activeChatID, currentVotePostID)
      .then((items) => {
        if (cancelled) {
          return
        }
        setPollOptions(items)
        setManualVoteDraft((prev) => {
          if (prev.choice !== '' && items.some((item) => item.choice === prev.choice)) {
            return prev
          }
          return { ...prev, choice: '' }
        })
      })
      .catch((err) => {
        if (cancelled) {
          return
        }
        setPollOptions([])
        setPollOptionsError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) {
          setPollOptionsLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, currentVotePostID, success])

  useEffect(() => {
    if (activeSection !== 'events' || activeChatID === null || activeHistoryEventID === null) {
      setEventBilling(null)
      setEventBillingDraft({})
      return
    }
    let cancelled = false
    setEventBillingLoading(true)
    setEventBillingError('')
    void fetchEventBillingForInstance(activeChatID, activeHistoryEventID)
      .then((billing) => {
        if (cancelled) {
          return
        }
        setEventBilling(billing)
        const draft: Record<number, boolean> = {}
        if (billing?.players) {
          for (const player of billing.players) {
            draft[player.userID] = Boolean(player.isPaid)
          }
        }
        setEventBillingDraft(draft)
      })
      .catch((err) => {
        if (cancelled) {
          return
        }
        setEventBilling(null)
        setEventBillingDraft({})
        setEventBillingError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) {
          setEventBillingLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, activeHistoryEventID, success, attendanceRevision])

  useEffect(() => {
    if (activeSection !== 'billing' || activeChatID === null || !isAdmin) {
      setBillingDebtors([])
      setBillingLoading(false)
      setBillingError('')

      return
    }
    let cancelled = false
    setBillingLoading(true)
    setBillingError('')
    void fetchGroupDebtors(activeChatID)
      .then((items) => {
        if (cancelled) return
        const safe = Array.isArray(items) ? items : []
        setBillingDebtors(safe)

      })
      .catch((err) => {
        if (cancelled) return
        setBillingDebtors([])
        setBillingError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) setBillingLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, isAdmin, success])

  useEffect(() => {
    if (activeSection !== 'games' || activeChatID === null) {
      setGroupGames([])
      setGroupGamesLoading(false)
      setGroupGamesError('')
      setGamesExpandedRows({})
      setGamesRosterCache({})
      return
    }
    let cancelled = false
    setGroupGamesLoading(true)
    setGroupGamesError('')
    void fetchGroupGames(activeChatID)
      .then((rows) => {
        if (cancelled) return
        setGroupGames(Array.isArray(rows) ? rows : [])
      })
      .catch((err) => {
        if (cancelled) return
        setGroupGames([])
        setGroupGamesError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) setGroupGamesLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, success])

  useEffect(() => {
    if (activeSection !== 'events' || activeChatID === null || activeHistoryEventID === null) {
      setEventSetRowsDraft([])
      setEventSetsError('')
      setEventSetsLoading(false)
      return
    }
    let cancelled = false
    setEventSetsLoading(true)
    setEventSetsError('')
    void fetchEventSetRowsForInstance(activeChatID, activeHistoryEventID)
      .then((rows) => {
        if (cancelled) return
        const safe = Array.isArray(rows) ? rows : []
        setEventSetRowsDraft(
          safe
            .sort((a, b) => (a.ordinal ?? 0) - (b.ordinal ?? 0))
            .map((r) => ({
              left: (r.team1 as ActiveTeamCode) || 'A',
              right: (r.team2 as ActiveTeamCode) || 'B',
              leftScore: clampInt(r.score1 ?? 0, 0, 99),
              rightScore: clampInt(r.score2 ?? 0, 0, 99),
            })),
        )
      })
      .catch((err) => {
        if (cancelled) return
        setEventSetRowsDraft([])
        setEventSetsError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) setEventSetsLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, activeHistoryEventID, success, attendanceRevision])

  useEffect(() => {
    setHistoryDetailTab('distribution')
  }, [activeHistoryEventID])

  useEffect(() => {
    if (activeSection !== 'events' || activeChatID === null || selectedHistoryPostID === null) {
      setHistoryPollVotes([])
      setHistoryPollVotesError('')
      setHistoryPollVotesLoading(false)
      return
    }
    let cancelled = false
    setHistoryPollVotesLoading(true)
    setHistoryPollVotesError('')
    void fetchGroupPollVotes(activeChatID, selectedHistoryPostID)
      .then((items) => {
        if (cancelled) {
          return
        }
        setHistoryPollVotes(items)
      })
      .catch((err) => {
        if (cancelled) {
          return
        }
        setHistoryPollVotes([])
        setHistoryPollVotesError((err as Error).message)
      })
      .finally(() => {
        if (!cancelled) {
          setHistoryPollVotesLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, selectedHistoryPostID, success])

  useEffect(() => {
    if (
      activeSection !== 'events' ||
      activeChatID === null ||
      activeHistoryEventID === null ||
      selectedHistoryPostID === null ||
      !selectedHistoryEvent
    ) {
      return
    }
    let cancelled = false
    setTeamSplitLoading(true)
    setTeamSplitError('')
    void fetchEventTeamSplit(activeChatID, selectedHistoryEvent.eventID, selectedHistoryPostID)
      .then((state) => {
        if (!cancelled) {
          setTeamSplit(state)
          setTeamCEnabled(state.players.some((player) => player.team === 'C'))
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setTeamSplit(null)
          setTeamSplitError((err as Error).message)
        }
      })
      .finally(() => {
        if (!cancelled) {
          setTeamSplitLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [activeSection, activeChatID, activeHistoryEventID, selectedHistoryPostID, selectedHistoryEvent])

  useEffect(() => {
    if (historyDetailTab !== 'sets' || selectedHistoryPostID === null) {
      return
    }
    if (eventSetRowsDraft.length > 0) {
      return
    }
    const teams = teamSplit ? activeTeamCodes(teamSplit.players, teamCEnabled) : (['A', 'B'] as ActiveTeamCode[])
    if (teams.length < 2) {
      return
    }
    setEventSetRowsDraft([{ left: teams[0], right: teams[1], leftScore: 0, rightScore: 0 }])
  }, [eventSetRowsDraft.length, historyDetailTab, selectedHistoryPostID, teamCEnabled, teamSplit])

  useEffect(() => {
    if (activeSection !== 'members' || activeMemberID === null || activeChatID === null) {
      return
    }
    if (members.length > 0 && !members.some((member) => member.userTelegramID === activeMemberID)) {
      navigateTo(
        {
          chatID: activeChatID,
          section: 'members',
          memberID: null,
          templateView: 'list',
          templateName: null,
          eventView: 'list',
          eventID: null,
        },
        true,
      )
      return
    }
    void loadMemberProfile(activeMemberID)
  }, [activeSection, activeMemberID, activeChatID, members])

  useEffect(() => {
    setTemplateForm((prev) => ensureAtLeastTwoTemplateState(prev))
  }, [templateForm.options.length])

  useEffect(() => {
    setTemplateEditor((prev) => ensureAtLeastTwoTemplateState(prev))
  }, [templateEditor.options.length])

  useEffect(() => {
    if (eventEditorCanEnableSettlement) {
      return
    }
    setEventEditor((prev) => {
      if (!prev.settlementEnabled) {
        return prev
      }
      return { ...prev, settlementEnabled: false }
    })
  }, [eventEditorCanEnableSettlement])

  useEffect(() => {
    if (!success) {
      setSuccessVisible(false)
      return
    }

    setSuccessVisible(true)
    const hideTimer = window.setTimeout(() => setSuccessVisible(false), 2200)
    const clearTimer = window.setTimeout(() => setSuccess(''), 2900)

    return () => {
      window.clearTimeout(hideTimer)
      window.clearTimeout(clearTimer)
    }
  }, [success])

  useEffect(() => {
    const mql = window.matchMedia('(pointer: coarse)')
    const onChange = () => setTouchDragMode(mql.matches)
    onChange()
    mql.addEventListener('change', onChange)
    return () => mql.removeEventListener('change', onChange)
  }, [])

  useEffect(() => {
    const mql = window.matchMedia('(max-width: 900px)')
    const onChange = () => {
      if (!mql.matches) {
        setMobileNavOpen(false)
      }
    }
    onChange()
    mql.addEventListener('change', onChange)
    return () => mql.removeEventListener('change', onChange)
  }, [])

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setMobileNavOpen(false)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  async function reloadActiveOrganization(chatID: number, options?: { silent?: boolean }) {
    const silent = options?.silent === true
    if (!silent) {
      setLoading(true)
    }
    setError('')
    setMembersError('')
    try {
      const loadedDetails = await fetchGroupDetails(chatID)
      setDetails(loadedDetails)
      let loadedMembers: GroupMember[] = []
      try {
        const debt = await fetchGroupDebtSummary(chatID)
        setGroupDebtSummary(debt)
      } catch {
        setGroupDebtSummary(null)
      }
      try {
        const loadedArchived = await fetchArchivedEvents(chatID)
        setArchivedEvents(loadedArchived)
      } catch {
        setArchivedEvents([])
      }

      try {
        loadedMembers = await fetchGroupMembers(chatID)
        setMembers(loadedMembers)
      } catch (membersErr) {
        setMembers([])
        setMembersError((membersErr as Error).message)
        loadedMembers = []
      }
      try {
        const loadedSkills = await fetchSkillsCatalog(chatID)
        setSkillsCatalog(loadedSkills)
      } catch {
        setSkillsCatalog([])
      }
      try {
        if (loadedMembers.length === 0) {
          setMemberSkillProfiles({})
        } else {
          const profiles = await Promise.all(
            loadedMembers.map(async (member) => {
              try {
                return await fetchMemberSkills(chatID, member.userTelegramID)
              } catch {
                return null
              }
            }),
          )
          const map: Record<number, MemberSkillProfile> = {}
          for (const profile of profiles) {
            if (!profile) {
              continue
            }
            map[profile.userTelegramID] = profile
          }
          setMemberSkillProfiles(map)
        }
      } catch {
        setMemberSkillProfiles({})
      }

    } catch (err) {
      setError((err as Error).message)
    } finally {
      if (!silent) {
        setLoading(false)
      }
    }
  }

  async function refreshActiveOrganization() {
    if (!activeChatID || loading || refreshing) return
    setRefreshing(true)
    try {
      await reloadActiveOrganization(activeChatID, { silent: true })
    } finally {
      setRefreshing(false)
    }
  }

  async function runAction(action: () => Promise<void>, okMessage: string) {
    if (activeChatID === null) {
      return
    }
    setError('')
    setSuccess('')
    try {
      await action()
      await reloadActiveOrganization(activeChatID, { silent: true })
      setSuccess(okMessage)
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function onPublishRegistration() {
    if (activeChatID === null) {
      return
    }
    await runAction(() => publishRegistration(activeChatID), 'Регистрация опубликована')
  }

  async function onCreateTemplate(e: FormEvent) {
    e.preventDefault()
    if (activeChatID === null) {
      return
    }
    const nextTemplateName = templateForm.name.trim()
    const payload = buildTemplatePayload(templateForm.options, templateForm.counted, templateForm.weights)
    if (payload.options.length < 2) {
      setError('Для шаблона нужно минимум 2 варианта')
      return
    }
    await runAction(
      () =>
        createTemplate(activeChatID, {
          name: nextTemplateName,
          question: templateForm.question.trim(),
          options: payload.options,
          countedOptions: payload.countedOptions,
          optionWeights: payload.optionWeights,
        }),
      'Шаблон сохранен',
    )
    setTemplateForm({ name: '', question: '', options: ['', ''], counted: [false, false], weights: [1, 1] })
    navigateTo(
      {
        chatID: activeChatID,
        section: 'templates',
        templateView: 'edit',
        templateName: nextTemplateName,
        eventView: 'list',
        eventID: null,
      },
      true,
    )
  }

  async function onDeleteTemplate(templateName: string) {
    if (activeChatID === null) {
      return
    }
    if (templateName.trim().toLowerCase() === 'регистрация') {
      setError('Нельзя удалить системный шаблон \"Регистрация\"')
      return
    }
    await runAction(() => deleteTemplate(activeChatID, templateName), 'Шаблон удален')
  }

  function updateTemplateFormOption(index: number, value: string) {
    setTemplateForm((prev) => {
      const next = [...prev.options]
      next[index] = value
      return { ...prev, options: next }
    })
  }

  function updateTemplateFormWeight(index: number, value: string) {
    const parsed = Number.parseInt(value, 10)
    const nextWeight = Number.isFinite(parsed) && parsed > 0 ? parsed : 1
    setTemplateForm((prev) => {
      const next = [...prev.weights]
      next[index] = nextWeight
      return { ...prev, weights: next }
    })
  }

  function addTemplateFormOption() {
    setTemplateForm((prev) => ({ ...prev, options: [...prev.options, ''], counted: [...prev.counted, false], weights: [...prev.weights, 1] }))
  }

  function removeTemplateFormOption(index: number) {
    setTemplateForm((prev) => {
      if (prev.options.length <= 2) {
        return prev
      }
      return {
        ...prev,
        options: prev.options.filter((_, i) => i !== index),
        counted: prev.counted.filter((_, i) => i !== index),
        weights: prev.weights.filter((_, i) => i !== index),
      }
    })
  }

  function updateTemplateEditorOption(index: number, value: string) {
    setTemplateEditor((prev) => {
      const next = [...prev.options]
      next[index] = value
      return { ...prev, options: next }
    })
  }

  function updateTemplateEditorWeight(index: number, value: string) {
    const parsed = Number.parseInt(value, 10)
    const nextWeight = Number.isFinite(parsed) && parsed > 0 ? parsed : 1
    setTemplateEditor((prev) => {
      const next = [...prev.weights]
      next[index] = nextWeight
      return { ...prev, weights: next }
    })
  }

  function addTemplateEditorOption() {
    setTemplateEditor((prev) => ({ ...prev, options: [...prev.options, ''], counted: [...prev.counted, false], weights: [...prev.weights, 1] }))
  }

  function removeTemplateEditorOption(index: number) {
    setTemplateEditor((prev) => {
      if (prev.options.length <= 2) {
        return prev
      }
      return {
        ...prev,
        options: prev.options.filter((_, i) => i !== index),
        counted: prev.counted.filter((_, i) => i !== index),
        weights: prev.weights.filter((_, i) => i !== index),
      }
    })
  }

  function toggleTemplateFormCounted(index: number) {
    setTemplateForm((prev) => {
      const next = [...prev.counted]
      next[index] = !next[index]
      return { ...prev, counted: next }
    })
  }

  function toggleTemplateEditorCounted(index: number) {
    setTemplateEditor((prev) => {
      const next = [...prev.counted]
      next[index] = !next[index]
      return { ...prev, counted: next }
    })
  }

  async function onSaveTemplateEditor(e: FormEvent) {
    e.preventDefault()
    if (activeChatID === null || !activeTemplateName) {
      return
    }
    const payload = buildTemplatePayload(templateEditor.options, templateEditor.counted, templateEditor.weights)
    if (payload.options.length < 2) {
      setError('Для шаблона нужно минимум 2 варианта')
      return
    }

    setError('')
    setSuccess('')
    try {
      const updated = await updateTemplate(activeChatID, activeTemplateName, {
        name: templateEditor.name.trim(),
        question: templateEditor.question.trim(),
        options: payload.options,
        countedOptions: payload.countedOptions,
        optionWeights: payload.optionWeights,
      })

      await reloadActiveOrganization(activeChatID)
      setSuccess('Шаблон обновлен')

      navigateTo(
        {
          chatID: activeChatID,
          section: 'templates',
          templateView: 'edit',
          templateName: updated.name,
          eventView: 'list',
          eventID: null,
        },
        true,
      )
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function onDeleteTemplateEditor() {
    if (activeChatID === null || !activeTemplateName) {
      return
    }

    if (activeTemplateName.trim().toLowerCase() === 'регистрация') {
      setError('Нельзя удалить системный шаблон \"Регистрация\"')
      return
    }

    await runAction(() => deleteTemplate(activeChatID, activeTemplateName), 'Шаблон удален')
    navigateTo(
      {
        chatID: activeChatID,
        section: 'templates',
        templateView: 'list',
        templateName: null,
        eventView: 'list',
        eventID: null,
      },
      true,
    )
  }

  async function onCreateEvent(e: FormEvent) {
    e.preventDefault()
    if (activeChatID === null) {
      return
    }

    const weekday = Number(eventForm.weekday)
    const publishWeekday = Number(eventForm.publishWeekday)
    const eventType = eventForm.eventType
    const templateName = eventForm.templateName.trim()
    const announcementLeadMinutes = Number(eventForm.announcementLeadMinutes)
    const teamSize = Number(eventForm.teamSize)
    const maxPlaces = Number(eventForm.maxPlaces)
    const minVotesToHold = Number(eventForm.minVotesToHold)
    const cancelLeadMinutes = Number(eventForm.cancelLeadMinutes)
    const cancelNotifyEnabled = eventForm.cancelNotifyEnabled
    const announcementText = eventForm.announcementText.trim()
    const announcementEnabled = eventForm.announcementEnabled
    const teamsAutoSplit = eventForm.teamsAutoSplit
    const teamsPublishList = eventForm.teamsPublishList
    const settlementEnabled = eventForm.settlementEnabled
    const settlementPublishBefore = eventForm.settlementPublishBefore
    const settlementPublishAfter = eventForm.settlementPublishAfter
    const costAmount = eventForm.costAmount.trim() === '' ? undefined : Number(eventForm.costAmount)
    if (Number.isNaN(weekday) || Number.isNaN(publishWeekday)) {
      setError('Выбери дни недели для события и публикации')
      return
    }
    if (![60, 120, 1440].includes(announcementLeadMinutes)) {
      setError('Выбери интервал публикации анонса')
      return
    }
    if (eventType === 'training' && (Number.isNaN(teamSize) || teamSize < 2)) {
      setError('Количество игроков в команде должно быть не меньше 2')
      return
    }
    if (Number.isNaN(maxPlaces) || maxPlaces <= 0) {
      setError('Максимальное количество мест должно быть больше 0')
      return
    }
    if (Number.isNaN(minVotesToHold) || minVotesToHold < 0) {
      setError('Минимальное количество голосов не может быть отрицательным')
      return
    }
    if (Number.isNaN(cancelLeadMinutes) || cancelLeadMinutes <= 0) {
      setError('Время отмены должно быть больше 0 минут')
      return
    }
	    if (announcementEnabled && announcementText === '') {
	      setError('Заполни текст анонса или отключи публикацию')
	      return
	    }
	    const startMin = clockMinutes(eventForm.startAt)
	    const endMin = clockMinutes(eventForm.endAt)
	    const publishMin = clockMinutes(eventForm.publishAt)
	    if (startMin === null) {
	      setError('Время начала должно быть в формате ЧЧ:ММ, например 19:30')
	      return
	    }
	    if (endMin === null) {
	      setError('Время окончания должно быть в формате ЧЧ:ММ, например 19:30')
	      return
	    }
	    if (endMin <= startMin) {
	      setError('Время окончания должно быть позже времени начала')
	      return
	    }
	    if (publishMin === null) {
	      setError('Время публикации должно быть в формате ЧЧ:ММ, например 19:30')
	      return
	    }
	    if (publishWeekday === weekday && publishMin > startMin) {
	      setError('Время публикации опроса не может быть позже начала события')
	      return
	    }
	    if (settlementEnabled && !settlementPublishBefore && !settlementPublishAfter) {
	      setError('Для расчёта выбери хотя бы один режим: перед началом или после')
	      return
	    }
    if (settlementEnabled && !eventEditorCanEnableSettlement) {
      setError('Расчёт доступен только при привязанном шаблоне с отмеченными вариантами "Учёт"')
      return
    }
    if (settlementEnabled) {
      setError('Для нового события сначала привяжи шаблон с отмеченными вариантами "Учёт"')
      return
    }
    if (templateName === '') {
      setError('Выбери шаблон опроса')
      return
    }
    setError('')
    setSuccess('')

    let createdEventID: number | null = null
    try {
      const created = await createEvent(activeChatID, {
        name: eventForm.name.trim(),
        mentions: eventForm.mentions,
        eventType,
        templateName,
        weekday,
        publishWeekday,
        publishAt: eventForm.publishAt,
        startAt: eventForm.startAt,
        endAt: eventForm.endAt,
        announcementText,
        announcementEnabled,
        announcementLeadMinutes,
        teamsAutoSplit,
        teamsPublishList,
        teamSize,
        maxPlaces,
        minVotesToHold,
        cancelLeadMinutes,
        cancelNotifyEnabled,
        settlementEnabled,
        settlementPublishBefore,
        settlementPublishAfter,
        costAmount,
      })
      createdEventID = created.id
      await reloadActiveOrganization(activeChatID)
      setSuccess('Событие создано')
    } catch (err) {
      setError((err as Error).message)
      return
    }

    setEventForm({
      name: '',
      eventType: 'training',
      templateName: '',
      weekday: '5',
      publishWeekday: '5',
      publishAt: '10:00',
      startAt: '19:00',
      endAt: '21:00',
      announcementText: '',
      mentions: emptyEventMentions(),
      announcementEnabled: false,
      announcementLeadMinutes: '60',
      teamsAutoSplit: false,
      teamsPublishList: false,
      teamSize: '6',
      maxPlaces: '18',
      minVotesToHold: '0',
      cancelLeadMinutes: '180',
      cancelNotifyEnabled: false,
      settlementEnabled: false,
      settlementPublishBefore: false,
      settlementPublishAfter: true,
      costAmount: '',
    })

    if (createdEventID !== null) {
      navigateTo(
        {
          chatID: activeChatID,
          section: 'event_templates',
          templateView: 'list',
          templateName: null,
          eventView: 'edit',
          eventID: createdEventID,
        },
        true,
      )
    }
  }

  async function onSaveEventConfiguration(e: FormEvent) {
    e.preventDefault()
    if (activeChatID === null || activeEventID === null || !selectedEvent) {
      return
    }
    const name = eventEditor.name.trim()
    const eventType = eventEditor.eventType
    const weekday = Number(eventEditor.weekday)
    let publishWeekday = Number(eventEditor.publishWeekday)
    const announcementLeadMinutes = Number(eventEditor.announcementLeadMinutes)
    const teamSize = Number(eventEditor.teamSize)
    const maxPlaces = Number(eventEditor.maxPlaces)
    const minVotesToHold = Number(eventEditor.minVotesToHold)
    const cancelLeadMinutes = Number(eventEditor.cancelLeadMinutes)
    const cancelNotifyEnabled = eventEditor.cancelNotifyEnabled
    const announcementText = eventEditor.announcementText.trim()
    const announcementEnabled = eventEditor.announcementEnabled
    const teamsAutoSplit = eventEditor.teamsAutoSplit
    const teamsPublishList = eventEditor.teamsPublishList
    const settlementEnabled = eventEditor.settlementEnabled
    const settlementPublishBefore = eventEditor.settlementPublishBefore
    const settlementPublishAfter = eventEditor.settlementPublishAfter
    const templateName = eventEditor.templateName.trim()
    const hasTemplate = templateName !== ''
    let publishAt = eventEditor.publishAt.trim()
    const costValue = eventEditor.costAmount.trim()
    const costAmount = costValue === '' ? undefined : Number(costValue)

    if (name === '') {
      setError('Название события обязательно')
      return
    }
    if (Number.isNaN(weekday)) {
      setError('Выбери день недели события')
      return
    }
    if (hasTemplate && Number.isNaN(publishWeekday)) {
      setError('Выбери день публикации опроса')
      return
    }
	    if (hasTemplate && publishAt === '') {
	      setError('Укажи время публикации опроса')
	      return
	    }
    if (!hasTemplate) {
      if (Number.isNaN(publishWeekday)) {
        publishWeekday = weekday
      }
      if (publishAt === '') {
        publishAt = '10:00'
      }
    }
    if (![60, 120, 1440].includes(announcementLeadMinutes)) {
      setError('Выбери интервал публикации анонса')
      return
    }
    if (eventType === 'training' && (Number.isNaN(teamSize) || teamSize < 2)) {
      setError('Количество игроков в команде должно быть не меньше 2')
      return
    }
    if (Number.isNaN(maxPlaces) || maxPlaces <= 0) {
      setError('Максимальное количество мест должно быть больше 0')
      return
    }
    if (Number.isNaN(minVotesToHold) || minVotesToHold < 0) {
      setError('Минимальное количество голосов не может быть отрицательным')
      return
    }
    if (Number.isNaN(cancelLeadMinutes) || cancelLeadMinutes <= 0) {
      setError('Время отмены должно быть больше 0 минут')
      return
    }
	    if (announcementEnabled && announcementText === '') {
	      setError('Заполни текст анонса или отключи публикацию')
	      return
	    }
	    const startMin = clockMinutes(eventEditor.startAt)
	    const endMin = clockMinutes(eventEditor.endAt)
	    const publishMin = clockMinutes(publishAt)
	    if (startMin === null) {
	      setError('Время начала должно быть в формате ЧЧ:ММ, например 19:30')
	      return
	    }
	    if (endMin === null) {
	      setError('Время окончания должно быть в формате ЧЧ:ММ, например 19:30')
	      return
	    }
	    if (endMin <= startMin) {
	      setError('Время окончания должно быть позже времени начала')
	      return
	    }
	    if (publishMin === null) {
	      setError('Время публикации должно быть в формате ЧЧ:ММ, например 19:30')
	      return
	    }
	    if (publishWeekday === weekday && publishMin > startMin) {
	      setError('Время публикации опроса не может быть позже начала события')
	      return
	    }
	    if (settlementEnabled && !settlementPublishBefore && !settlementPublishAfter) {
	      setError('Для расчёта выбери хотя бы один режим: перед началом или после')
	      return
	    }
    if (costAmount !== undefined && (Number.isNaN(costAmount) || costAmount < 0)) {
      setError('Укажите стоимость числом, не меньше нуля')
      return
    }
    const selectedPublishWeekday = selectedEvent.pollPublishWeekday || selectedEvent.startWeekday
    const selectedAnnouncementLead = selectedEvent.announcementLeadMinutes || 60
    const detailsChanged =
      name !== selectedEvent.name ||
      eventType !== (selectedEvent.eventType || 'training') ||
      weekday !== selectedEvent.startWeekday ||
      publishWeekday !== selectedPublishWeekday ||
      publishAt !== toHourMinute(selectedEvent.pollPublishTime) ||
      eventEditor.startAt !== toHourMinute(selectedEvent.startTime) ||
      eventEditor.endAt !== toHourMinute(selectedEvent.endTime) ||
      !equalEventMentions(eventEditor.mentions, selectedEvent.mentions) ||
      announcementText !== (selectedEvent.announcementText || '') ||
      announcementEnabled !== selectedEvent.announcementEnabled ||
      announcementLeadMinutes !== selectedAnnouncementLead ||
      teamsAutoSplit !== selectedEvent.teamsAutoSplit ||
      teamsPublishList !== selectedEvent.teamsPublishList ||
      teamSize !== (selectedEvent.teamSize || 6) ||
      maxPlaces !== (selectedEvent.maxPlaces || 18) ||
      minVotesToHold !== (selectedEvent.minVotesToHold || 0) ||
      cancelLeadMinutes !== (selectedEvent.cancelLeadMinutes || 180) ||
      cancelNotifyEnabled !== (selectedEvent.cancelNotifyEnabled || false) ||
      settlementEnabled !== selectedEvent.settlementEnabled ||
      settlementPublishBefore !== selectedEvent.settlementPublishBefore ||
      settlementPublishAfter !== selectedEvent.settlementPublishAfter

    const selectedCost = selectedEvent.costAmount
    const costChanged =
      (costAmount === undefined && selectedCost !== undefined) ||
      (costAmount !== undefined && selectedCost === undefined) ||
      (costAmount !== undefined && selectedCost !== undefined && Math.abs(costAmount - selectedCost) > 0.000001)
    const templateChanged = templateName !== (selectedEvent.pollTemplate || '')

    await runAction(
      async () => {
        if (detailsChanged) {
          await updateEventDetails(activeChatID, activeEventID, {
            mentions: eventEditor.mentions,
            name,
            eventType,
            weekday,
            publishWeekday,
            publishAt,
            startAt: eventEditor.startAt,
            endAt: eventEditor.endAt,
            announcementText,
            announcementEnabled,
            announcementLeadMinutes,
            teamsAutoSplit,
            teamsPublishList,
            teamSize,
            maxPlaces,
            minVotesToHold,
            cancelLeadMinutes,
            cancelNotifyEnabled,
            settlementEnabled,
            settlementPublishBefore,
            settlementPublishAfter,
          })
        }
        if (costChanged) {
          await updateEventCost(activeChatID, activeEventID, costAmount)
        }
        if (templateChanged && templateName !== '') {
          await bindEvent(activeChatID, activeEventID, templateName)
        }
      },
      'Событие обновлено',
    )
  }

  async function onGenerateBillingForInstance(instanceID: number) {
    if (activeChatID === null || instanceID === 0) {
      return
    }
    await runAction(async () => {
      await generateEventBillingForInstance(activeChatID, instanceID)
    }, 'Расчёт сформирован')
  }

  async function onArchiveEvent(eventID: number) {
    if (activeChatID === null) {
      return
    }
    await runAction(() => archiveEvent(activeChatID, eventID), 'Событие отправлено в архив')
    if (activeEventID === eventID) {
      navigateTo(
        {
          chatID: activeChatID,
          section: 'event_templates',
          templateView: 'list',
          templateName: null,
          eventView: 'list',
          eventID: null,
        },
        true,
      )
    }
  }

  async function onUnarchiveEvent(eventID: number) {
    if (activeChatID === null) {
      return
    }
    await runAction(() => unarchiveEvent(activeChatID, eventID), 'Событие восстановлено из архива')
  }

  async function onToggleEventActivity() {
    if (activeChatID === null || activeEventID === null) {
      return
    }
    if (showEventActivity) {
      setShowEventActivity(false)
      return
    }
    setEventActivityLoading(true)
    setEventActivityError('')
    try {
      const summary = await fetchEventActivity(activeChatID, activeEventID)
      setEventActivity(summary)
      setShowEventActivity(true)
    } catch (err) {
      setEventActivityError((err as Error).message)
      setShowEventActivity(true)
    } finally {
      setEventActivityLoading(false)
    }
  }

  async function onToggleEventPublishEnabled() {
    if (activeChatID === null || activeEventID === null || !selectedEvent) {
      return
    }
    if (selectedEvent.publishEnabled) {
      await runAction(() => deactivateEventPublications(activeChatID, activeEventID), 'Публикации события отключены')
      return
    }
    await runAction(() => activateEventPublications(activeChatID, activeEventID), 'Публикации события включены')
  }

  function moveTeamPlayer(userID: number, team: 'unassigned' | 'A' | 'B' | 'C') {
    setTeamSplit((prev) => {
      if (!prev) {
        return prev
      }
      const players = prev.players.map((player, idx) => {
        if (player.userID !== userID) {
          return player
        }
        return {
          ...player,
          team,
          position: idx,
        }
      })
      return {
        ...prev,
        players,
        formations: undefined,
        chance: (() => {
          let a = 0
          let b = 0
          for (const player of players) {
            if (player.team === 'A') {
              a += player.rating
            } else if (player.team === 'B') {
              b += player.rating
            }
          }
          const total = a + b
          if (total <= 0) {
            return { teamAScore: a, teamBScore: b, teamAProb: 0.5, teamBProb: 0.5 }
          }
          return { teamAScore: a, teamBScore: b, teamAProb: a / total, teamBProb: b / total }
        })(),
      }
    })
  }

  function onDragStartTeamPlayer(userID: number) {
    setDraggedPlayerID(userID)
  }

  function onDropToTeam(team: 'unassigned' | 'A' | 'B' | 'C') {
    if (draggedPlayerID === null) {
      return
    }
    moveTeamPlayer(draggedPlayerID, team)
    setDraggedPlayerID(null)
  }

  function removeTeamC() {
    setTeamSplit((prev) => {
      if (!prev) {
        return prev
      }
      const players = prev.players.map((player, idx) =>
        player.team === 'C'
          ? {
              ...player,
              team: 'unassigned' as const,
              position: idx,
            }
          : player,
      )
      return {
        ...prev,
        players,
        formations: undefined,
      }
    })
    setTeamCEnabled(false)
  }

  async function onSaveTeamSplit() {
    if (activeChatID === null || activeHistoryEventID === null || selectedHistoryPostID === null || !teamSplit || !selectedHistoryEvent) {
      return
    }
    try {
      const saved = await saveEventTeamSplit(
        activeChatID,
        selectedHistoryEvent.eventID,
        selectedHistoryPostID,
        teamSplit.players.map((player, idx) => ({
          userID: player.userID,
          team: player.team,
          position: idx,
        })),
      )
      setTeamSplit(saved)
      setSuccess('Распределение сохранено, стартовые позиции и схема пересчитаны')
      await reloadActiveOrganization(activeChatID, { silent: true })
    } catch (err) {
      setTeamSplitError((err as Error).message)
    }
  }

  async function onAutoSplitTeam() {
    if (activeChatID === null || activeHistoryEventID === null || selectedHistoryPostID === null || !selectedHistoryEvent) {
      return
    }
    try {
      const next = await autoSplitEventTeams(activeChatID, selectedHistoryEvent.eventID, selectedHistoryPostID)
      setTeamSplit(next)
      setTeamCEnabled(next.players.some((player) => player.team === 'C'))
      setSuccess('Автораспределение выполнено')
      setTeamSplitError('')
    } catch (err) {
      setTeamSplitError((err as Error).message)
    }
  }

  async function onPublishTeamSplit() {
    if (activeChatID === null || activeHistoryEventID === null || selectedHistoryPostID === null || !selectedHistoryEvent) {
      return
    }
    await runAction(
      () => publishEventTeamSplit(activeChatID, selectedHistoryEvent.eventID, selectedHistoryPostID),
      'Состав команд опубликован',
    )
  }

  async function onAddManualPollVote(postID: number) {
    if (activeChatID === null) {
      return
    }
    const userID = Number(manualVoteDraft.userID)
    if (!Number.isInteger(userID) || userID <= 0) {
      setError('Выбери игрока для добавления голоса')
      return
    }
    const choice = manualVoteDraft.choice.trim()
    if (choice === '') {
      setError('Выбери вариант голоса')
      return
    }
    await runAction(
      () =>
        addGroupPollVoteForUser(activeChatID, postID, {
          userID,
          choice,
        }),
      'Голос добавлен, расчёт обновлён',
    )
    setManualVoteDraft((prev) => ({ ...prev, userID: '' }))
  }

  async function onCreateFromTemplate(eventID: number) {
    if (activeChatID === null) {
      return
    }
    try {
      const created = await createEventInstance(activeChatID, eventID)
      await reloadActiveOrganization(activeChatID, { silent: true })
      setSuccess('Событие создано')
      navigateTo({
        chatID: activeChatID,
        section: 'events',
        historyEventID: created.instanceID,
        templateView: 'list',
        templateName: null,
        eventView: 'list',
        eventID: null,
      })
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function onDeleteEventInstanceForTest() {
    if (activeChatID === null || activeHistoryEventID === null) {
      return
    }
    const confirmed = window.confirm(
      'Удалить это событие целиком?\n\nБудут удалены: голосование, голоса, распределение, расчёт/оплата и отметки публикаций.\n\nДействие необратимо.',
    )
    if (!confirmed) {
      return
    }
    await runAction(
      async () => {
        await deleteEventInstance(activeChatID, activeHistoryEventID)
        await reloadActiveOrganization(activeChatID, { silent: true })
        navigateTo({
          chatID: activeChatID,
          section: 'events',
          historyEventID: null,
          templateView: 'list',
          templateName: null,
          eventView: 'list',
          eventID: null,
        })
      },
      'Событие удалено',
    )
  }

  async function onSaveBilling() {
    if (activeChatID === null || activeHistoryEventID === null || !eventBilling) {
      return
    }
    try {
      const statuses = eventBilling.players.filter(player => !(player.passCovered && player.amountDue <= 0)).map((player) => ({
        userID: player.userID,
        paid: Boolean(eventBillingDraft[player.userID]),
      }))
      await saveEventBillingForInstance(activeChatID, activeHistoryEventID, statuses)
      setSuccess('Оплаты сохранены')
      await reloadActiveOrganization(activeChatID, { silent: true })
      const refreshed = await fetchEventBillingForInstance(activeChatID, activeHistoryEventID)
      setEventBilling(refreshed)
      const draft: Record<number, boolean> = {}
      if (refreshed?.players) {
        for (const player of refreshed.players) {
          draft[player.userID] = Boolean(player.isPaid)
        }
      }
      setEventBillingDraft(draft)
      setEventBillingError('')
    } catch (err) {
      setEventBillingError((err as Error).message)
    }
  }

  async function onSaveEventSets() {
    if (activeChatID === null || activeHistoryEventID === null) {
      return
    }
    try {
      const baseRows =
        eventSetRowsDraft.length > 0 ? eventSetRowsDraft : [{ left: 'A' as const, right: 'B' as const, leftScore: 0, rightScore: 0 }]

      const filtered = baseRows.filter((r) => r.left !== r.right)
      const source = filtered.length ? filtered : [{ left: 'A' as const, right: 'B' as const, leftScore: 0, rightScore: 0 }]
      const rowsToSave = source.map((r, idx) => ({
        ordinal: idx + 1,
        team1: r.left,
        score1: clampInt(r.leftScore, 0, 99),
        team2: r.right,
        score2: clampInt(r.rightScore, 0, 99),
      }))
      await saveEventSetRowsForInstance(activeChatID, activeHistoryEventID, rowsToSave)
      setSuccess('Партии сохранены')
      setEventSetsError('')
    } catch (err) {
      setEventSetsError((err as Error).message)
    }
  }

  async function onPublishEventSets() {
    if (activeChatID === null || activeHistoryEventID === null) {
      return
    }
    await runAction(async () => {
      const baseRows =
        eventSetRowsDraft.length > 0 ? eventSetRowsDraft : [{ left: 'A' as const, right: 'B' as const, leftScore: 0, rightScore: 0 }]
      const filtered = baseRows.filter((r) => r.left !== r.right)
      const source = filtered.length ? filtered : [{ left: 'A' as const, right: 'B' as const, leftScore: 0, rightScore: 0 }]
      const rowsToSave = source.map((r, idx) => ({
        ordinal: idx + 1,
        team1: r.left,
        score1: clampInt(r.leftScore, 0, 99),
        team2: r.right,
        score2: clampInt(r.rightScore, 0, 99),
      }))
      await saveEventSetRowsForInstance(activeChatID, activeHistoryEventID, rowsToSave)
      await publishEventSetRowsForInstance(activeChatID, activeHistoryEventID)
    }, 'Результаты партий опубликованы')
  }

  async function loadMemberProfile(userTelegramID: number) {
    if (activeChatID === null) {
      return
    }
    setMemberSkillLoading(true)
    setMemberSkillError('')
    setSelectedMemberSkills(null)
    setMemberRelations([])
    try {
      const [profile, relations] = await Promise.all([
        fetchMemberSkills(activeChatID, userTelegramID),
        fetchMemberRelations(activeChatID, userTelegramID),
      ])
      const draft: Record<string, string> = {}
      for (const skill of profileSkills(profile, skillsCatalog)) {
        draft[skill.skillCode] = String(normalizedSkillScore(skill.score))
      }
      setMemberSkillDraft(draft)
      setMemberPlayerTypeDraft(profile.playerType || '')
      setMemberRealNameDraft(profile.realName || '')
      setSelectedMemberSkills(profile)
      setMemberRelations(relations)
      setMemberRelationDraft({
        otherUserID: '',
        relationType: 'prefer_together',
        weight: '5',
      })
      setMemberSkillProfiles((prev) => ({ ...prev, [profile.userTelegramID]: profile }))
    } catch (err) {
      setMemberSkillError((err as Error).message)
    } finally {
      setMemberSkillLoading(false)
    }
  }

  async function onOpenMemberCard(member: GroupMember) {
    if (activeChatID === null) {
      return
    }
    navigateTo({
      chatID: activeChatID,
      section: 'members',
      memberID: member.userTelegramID,
      templateView: 'list',
      templateName: null,
      eventView: 'list',
      eventID: null,
    })
    await loadMemberProfile(member.userTelegramID)
  }

  async function onSaveMemberSkills() {
    if (activeChatID === null || !selectedMemberSkills) {
      return
    }
    const payload: Record<string, number> = {}
    for (const skill of profileSkills(selectedMemberSkills, skillsCatalog)) {
      const raw = (memberSkillDraft[skill.skillCode] ?? '5').trim()
      const value = Number(raw)
      if (!Number.isInteger(value) || value < 1 || value > 10) {
        setMemberSkillError(`Оценка для "${skill.skillName}" должна быть целым числом от 1 до 10`)
        return
      }
      payload[skill.skillCode] = value
    }
    setMemberSkillError('')
    try {
      await updateMemberProfile(activeChatID, selectedMemberSkills.userTelegramID, {
        playerType: memberPlayerTypeDraft,
        realName: memberRealNameDraft,
      })
      await updateMemberSkills(activeChatID, selectedMemberSkills.userTelegramID, payload)
      const refreshed = await fetchMemberSkills(activeChatID, selectedMemberSkills.userTelegramID)
      const nextDraft: Record<string, string> = {}
      for (const skill of profileSkills(refreshed, skillsCatalog)) {
        nextDraft[skill.skillCode] = String(normalizedSkillScore(skill.score))
      }
      setMemberSkillDraft(nextDraft)
      setMemberPlayerTypeDraft(refreshed.playerType || '')
      setMemberRealNameDraft(refreshed.realName || '')
      setSelectedMemberSkills(refreshed)
      setMemberSkillProfiles((prev) => ({ ...prev, [refreshed.userTelegramID]: refreshed }))
      setMembers((prev) =>
        prev.map((member) =>
          member.userTelegramID === refreshed.userTelegramID
            ? {
                ...member,
                realName: refreshed.realName || '',
                playerType: refreshed.playerType || '',
              }
            : member,
        ),
      )
      setSuccess('Профиль игрока сохранен')
    } catch (err) {
      setMemberSkillError((err as Error).message)
    }
  }

  function relationTypeLabel(type: 'prefer_together' | 'avoid_together') {
    return type === 'prefer_together' ? 'Играть вместе' : 'Не в одну команду'
  }

  function relationMemberOptionLabel(member: GroupMember) {
    const profile = memberSkillProfiles[member.userTelegramID]
    const realName = (profile?.realName || member.realName || '').trim()
    const username = (member.username || '').trim()
    const base = fullName(member)
    if (realName.length > 0) {
      return username.length > 0 ? `${realName} · @${username}` : realName
    }
    if (username.length > 0 && base !== `@${username}`) {
      return `${base} · @${username}`
    }
    return base
  }

  function relationUserName(relation: PlayerRelation) {
    const member = memberByUserID.get(relation.relatedUserID)
    if (member) {
      const profile = memberSkillProfiles[member.userTelegramID]
      const realName = (profile?.realName || member.realName || '').trim()
      if (realName.length > 0) {
        return realName
      }
      return fullName(member)
    }
    const full = `${relation.relatedFirstName ?? ''} ${relation.relatedLastName ?? ''}`.trim()
    if (full) {
      return full
    }
    if (relation.relatedUsername) {
      return `@${relation.relatedUsername}`
    }
    return `ID ${relation.relatedUserID}`
  }

  async function onAddMemberRelation() {
    if (activeChatID === null || !selectedMemberSkills) {
      return
    }
    const otherUserID = Number(memberRelationDraft.otherUserID)
    const weight = Number(memberRelationDraft.weight)
    if (Number.isNaN(otherUserID) || otherUserID <= 0) {
      setMemberSkillError('Выбери второго игрока для связи')
      return
    }
    if (Number.isNaN(weight) || weight < 1 || weight > 10) {
      setMemberSkillError('Принципиальность должна быть от 1 до 10')
      return
    }
    setMemberSkillError('')
    try {
      await upsertMemberRelation(activeChatID, selectedMemberSkills.userTelegramID, {
        otherUserID,
        relationType: memberRelationDraft.relationType,
        weight,
      })
      const next = await fetchMemberRelations(activeChatID, selectedMemberSkills.userTelegramID)
      setMemberRelations(next)
      setMemberRelationDraft((prev) => ({ ...prev, otherUserID: '', weight: '5' }))
      setSuccess('Связь сохранена')
    } catch (err) {
      setMemberSkillError((err as Error).message)
    }
  }

  async function onDeleteMemberRelation(relation: PlayerRelation) {
    if (activeChatID === null || !selectedMemberSkills) {
      return
    }
    setMemberSkillError('')
    try {
      await deleteMemberRelation(
        activeChatID,
        selectedMemberSkills.userTelegramID,
        relation.relatedUserID,
        relation.relationType,
      )
      const next = await fetchMemberRelations(activeChatID, selectedMemberSkills.userTelegramID)
      setMemberRelations(next)
      setSuccess('Связь удалена')
    } catch (err) {
      setMemberSkillError((err as Error).message)
    }
  }

  function openStudioSection(section: Section, extra: Partial<RouteState> = {}) {
    setMobileNavOpen(false)
    navigateTo({ chatID: activeChatID, section, templateView: 'list', templateName: null, eventView: 'list', eventID: null, ...extra })
  }

  function renderOverview() {
    if (!details) return <section className="content-card">Выберите организацию</section>
    return <StudioOverview details={details} members={members} debt={groupDebtSummary} history={eventHistory} loading={eventHistoryLoading} error={eventHistoryError} openEvent={id => openStudioSection('events', { historyEventID: id })} openMembers={() => openStudioSection('members')} openEvents={() => openStudioSection('events')} openBilling={() => openStudioSection('billing')} createEvent={() => openStudioSection('event_templates')}/>
  }

  function pollVoteDisplayName(vote: GroupPollVoteItem): string {
    const real = (vote.realName ?? '').trim()
    if (real) {
      return real
    }
    const full = `${vote.firstName ?? ''} ${vote.lastName ?? ''}`.trim()
    if (full) {
      return full
    }
    if (vote.username) {
      return `@${vote.username}`
    }
    return `ID ${vote.userID}`
  }

  function renderManualPollVoteForm(postID: number | null) {
    if (!isAdmin || postID === null) {
      return null
    }

    const sortedMembers = [...members].sort((left, right) => {
      const leftLabel = relationMemberOptionLabel(left)
      const rightLabel = relationMemberOptionLabel(right)
      const byName = leftLabel.localeCompare(rightLabel, 'ru')
      if (byName !== 0) {
        return byName
      }
      return left.userTelegramID - right.userTelegramID
    })
    const canSubmit =
      manualVoteDraft.userID.trim() !== '' &&
      manualVoteDraft.choice.trim() !== '' &&
      !pollOptionsLoading &&
      pollOptionsError === '' &&
      sortedMembers.length > 0 &&
      pollOptions.length > 0

    return (
      <section className="settings-group manual-vote-card">
        <p className="settings-group-title">Добавить голос игрока</p>
        {pollOptionsLoading ? <p className="muted">Загрузка вариантов голосования...</p> : null}

        {sortedMembers.length === 0 ? <p className="muted">Список игроков пуст. Добавь игроков в организацию.</p> : null}
        {!pollOptionsLoading && !pollOptionsError && pollOptions.length === 0 ? (
          <p className="muted">В этом голосовании не найдено вариантов.</p>
        ) : null}
        <div className="settings-row settings-row-3 manual-vote-row">
          <label className="field">
            <span>Игрок</span>
            <Select
              value={manualVoteDraft.userID}
              onChange={(e) => setManualVoteDraft((prev) => ({ ...prev, userID: e.target.value }))}
            >
              <option value="">Выбери игрока</option>
              {sortedMembers.map((member) => (
                <option key={member.userTelegramID} value={member.userTelegramID}>
                  {`${relationMemberOptionLabel(member)} · ID ${member.userTelegramID}`}
                </option>
              ))}
            </Select>
          </label>
          <label className="field">
            <span>Вариант</span>
            <Select
              value={manualVoteDraft.choice}
              onChange={(e) => setManualVoteDraft((prev) => ({ ...prev, choice: e.target.value }))}
            >
              <option value="">Выбери вариант</option>
              {pollOptions.map((option) => {
                const suffix: string[] = []
                if (!option.counted) {
                  suffix.push('без учёта')
                }
                if (option.choiceWeight > 1) {
                  suffix.push(`x${option.choiceWeight}`)
                }
                const label = suffix.length > 0 ? `${option.choiceLabel} (${suffix.join(', ')})` : option.choiceLabel
                return (
                  <option key={option.choice} value={option.choice}>
                    {label}
                  </option>
                )
              })}
            </Select>
          </label>
          <label className="field">
            <span className="field-label-placeholder" />
            <button type="button" onClick={() => void onAddManualPollVote(postID)} disabled={!canSubmit}>
              Добавить голос
            </button>
          </label>
        </div>
      </section>
    )
  }

  function renderBilling() {
    if (activeChatID === null) return <section className="content-card">Выберите организацию</section>
    return <BillingWorkspace key={activeChatID} chatID={activeChatID}><StudioBilling key={activeChatID} chatID={activeChatID} groupTitle={details?.group.title || 'Команда'} debtors={billingDebtors} loading={billingLoading} error={billingError} summaryChanged={setGroupDebtSummary} openEvent={id => openStudioSection('events', { historyEventID: id })}/></BillingWorkspace>
  }

  function renderMembers() {
    if (activeMemberID !== null) {
      return (
        <section className="content-card studio-player-profile">
          <div className="template-head">
            <button
              className="btn-secondary"
              onClick={() =>
                navigateTo({
                  chatID: activeChatID,
                  section: 'members',
                  memberID: null,
                  templateView: 'list',
                  templateName: null,
                  eventView: 'list',
                  eventID: null,
                })
              }
            >
              ← К списку игроков
            </button>
            <h3>Карточка игрока</h3>
          </div>
          {memberSkillLoading ? <p className="muted">Загрузка навыков...</p> : null}

          {!memberSkillLoading && !memberSkillError && !selectedMemberSkills ? <p className="muted">Игрок не найден</p> : null}
          {!memberSkillLoading && selectedMemberSkills ? (
            <div className="studio-player-profile-content">
              <div className="player-fifa-card">
                <div className="player-avatar">{initialsFromProfile(selectedMemberSkills)}</div>
                <div className="player-ident">
                  <p className="player-fio">
                    {selectedMemberSkills.realName?.trim() || fullName(selectedMemberSkills)}
                  </p>
                  <p className="player-meta">@{selectedMemberSkills.username || 'без_ника'}</p>
                  <p className="player-meta">ID: {selectedMemberSkills.userTelegramID}</p>
                </div>
                <div
                  className="player-overall"
                >
                  <span>Рейтинг</span>
                  <strong>{averageScore(selectedMemberSkills, skillsCatalog)?.toFixed(1) ?? '-'}</strong>
                </div>
              </div>

              {activeChatID !== null && isAdmin && <TrainingPasses key={`${activeChatID}-${activeMemberID}`} chatID={activeChatID} userID={activeMemberID ?? undefined} canManage/>}
<div className="studio-player-name-field">
                <label className="field">
                  <span>Реальное имя</span>
                  <input
                    placeholder="Например: Иван Иванов"
                    value={memberRealNameDraft}
                    onChange={(e) => setMemberRealNameDraft(e.target.value)}
                  />
                </label>
              </div>

              <section className="player-type-block" aria-labelledby="player-position-title">
                  <div className="player-type-head">
                    <h4 id="player-position-title">Амплуа</h4>
                    <span className="skill-slider-value">{playerTypeLabel(memberPlayerTypeDraft)}</span>
                  </div>
                  <div className="player-type-grid">
                    {playerTypeCards.map((typeCard) => (
                      <button
                        key={typeCard.value}
                        type="button"
                        className={`player-type-card ${memberPlayerTypeDraft === typeCard.value ? 'active' : ''}`}
                        data-player-type={typeCard.value}
                        aria-pressed={memberPlayerTypeDraft === typeCard.value}
                        onClick={() => setMemberPlayerTypeDraft(typeCard.value)}
                      >
                        <div className="player-type-image" aria-hidden>
                          {typeCard.image}
                        </div>
                        <strong>{typeCard.title}</strong>
                        <small>{typeCard.subtitle}</small>
                      </button>
                    ))}
                  </div>
              </section>
              <section className="studio-player-skills" aria-labelledby="player-skills-title">
                <h4 id="player-skills-title">Навыки</h4>
                <div className="skill-slider-grid">
                {profileSkills(selectedMemberSkills, skillsCatalog).map((skill) => (
                  <label className="skill-slider-row" key={skill.skillCode}>
                    <span className="skill-slider-name">{skill.skillName}</span>
                    <input
                      type="range"
                      min="1"
                      max="10"
                      step="1"
                      value={memberSkillDraft[skill.skillCode] ?? String(normalizedSkillScore(skill.score))}
                      onChange={(e) => setMemberSkillDraft((prev) => ({ ...prev, [skill.skillCode]: e.target.value }))}
                    />
                    <span className="skill-slider-value">{memberSkillDraft[skill.skillCode] ?? String(normalizedSkillScore(skill.score))}/10</span>
                  </label>
                ))}
                </div>
              </section>
              <section className="content-card relation-block">
                <h4>Связи игрока</h4>
                <div className="form-grid form-grid-3 relation-form-grid">
                  <div className="field relation-member-field">
                    <span>Игрок</span>
                    <PlayerPicker key={activeMemberID} options={relationMemberOptions} value={memberRelationDraft.otherUserID}
                      onChange={(optionID) => setMemberRelationDraft((prev) => ({ ...prev, otherUserID: String(optionID) }))}/>
                  </div>
                  <label className="field">
                    <span>Тип связи</span>
                    <Select
                      className="ui-select"
                      value={memberRelationDraft.relationType}
                      onChange={(e) =>
                        setMemberRelationDraft((prev) => ({
                          ...prev,
                          relationType: e.target.value as 'prefer_together' | 'avoid_together',
                        }))
                      }
                    >
                      <option value="prefer_together">Играть вместе</option>
                      <option value="avoid_together">Не в одну команду</option>
                    </Select>
                  </label>
                  <div className="field relation-priority-field">
                    <div className="relation-priority-label">
                      <label htmlFor="member-relation-priority">Принципиальность</label>
                      <HelpTip label="Как работает принципиальность">Модуль автораспределения учитывает пожелания игроков при подборе команд. Чем выше значение, тем важнее учесть пожелание: 1 — желательно, 10 — очень важно.</HelpTip>
                    </div>
                    <input
                      id="member-relation-priority"
                      type="number"
                      min="1"
                      max="10"
                      step="1"
                      value={memberRelationDraft.weight}
                      onChange={(e) => setMemberRelationDraft((prev) => ({ ...prev, weight: e.target.value }))}
                    />
                  </div>
                </div>
                <div className="manual-controls">
                  <button type="button" className="btn-secondary" onClick={() => void onAddMemberRelation()}>
                    Добавить связь
                  </button>
                </div>
                {memberRelations.length === 0 ? (
                  <p className="muted">Связей пока нет</p>
                ) : (
                  <div className="list-block relation-list">
                    {memberRelations.map((relation) => (
                      <div
                        className="list-row relation-row"
                        key={`${relation.relatedUserID}-${relation.relationType}`}
                      >
                        <div>
                          <strong>{relationUserName(relation)}</strong>
                          <p>
                            {relationTypeLabel(relation.relationType)} · принципиальность {relation.weight}/10
                          </p>
                        </div>
                        <div className="list-actions">
                          <button
                            type="button"
                            className="btn-danger"
                            onClick={() => void onDeleteMemberRelation(relation)}
                          >
                            Удалить
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </section>
              <div className="studio-player-save">
                <button type="button" onClick={() => void onSaveMemberSkills()}>
                  Сохранить
                </button>
              </div>
            </div>
          ) : null}
        </section>
      )
    }

    return (
      <section className="content-card studio-members-panel">
        <div className="template-head" ref={membersHeadingRef}>
          <h3>Состав команды</h3><div className="studio-view-switch" aria-label="Вид списка игроков"><button className={membersView==='cards'?'active':''} aria-pressed={membersView==='cards'} onClick={()=>setMembersView('cards')}><Icon name="grid" size={16}/>Карточки</button><button className={membersView==='list'?'active':''} aria-pressed={membersView==='list'} onClick={()=>setMembersView('list')}><Icon name="template" size={16}/>Список</button></div>
        </div>

        <div className="form-grid form-grid-3 members-filters">
          <label className="field">
            <span>Поиск</span>
            <input
              placeholder="Имя, ник, реальное имя или ID"
              value={membersSearch}
              onChange={(e) => setMembersSearch(e.target.value)}
            />
          </label>
          <label className="field">
            <span>Тип игрока</span>
            <Select
              value={membersTypeFilter}
              onChange={(e) => setMembersTypeFilter(e.target.value as '' | 'attacker' | 'setter' | 'libero' | 'central')}
            >
              <option value="">Все типы</option>
              <option value="attacker">Атакующий</option>
              <option value="setter">Пасующий</option>
              <option value="libero">Либеро</option>
              <option value="central">Центральный</option>
            </Select>
          </label>
          <div className="field">
            <span id="members-rating-label">Рейтинг</span>
            <div className="studio-rating-range" role="group" aria-labelledby="members-rating-label">
              <input type="number" inputMode="decimal" min="1" max="10" step="any" aria-label="Рейтинг от" placeholder="От 1" value={membersRatingMin} aria-invalid={Boolean(membersRatingError)} aria-describedby={membersRatingError ? 'members-rating-error' : undefined} onChange={e => setMembersRatingMin(e.target.value)}/>
              <span aria-hidden="true">—</span>
              <input type="number" inputMode="decimal" min="1" max="10" step="any" aria-label="Рейтинг до" placeholder="До 10" value={membersRatingMax} aria-invalid={Boolean(membersRatingError)} aria-describedby={membersRatingError ? 'members-rating-error' : undefined} onChange={e => setMembersRatingMax(e.target.value)}/>
            </div>
          </div>
        </div>
        <div className="studio-member-filter-status">
          <span role="status">Найдено {filteredMembers.length} из {members.length}</span>
          {(membersSearch || membersTypeFilter || membersRatingMin || membersRatingMax) && <button type="button" className="studio-text-button" onClick={() => { setMembersSearch(''); setMembersTypeFilter(''); setMembersRatingMin(''); setMembersRatingMax('') }}>Сбросить фильтры</button>}
        </div>
        {membersRatingError && <p id="members-rating-error" className="studio-rating-error" role="status">{membersRatingError}</p>}
        {members.length === 0 ? (
          <p className="muted">Пока нет данных об игроках</p>
        ) : filteredMembers.length === 0 ? (
          <p className="muted">По текущим фильтрам игроков не найдено</p>
        ) : membersView === 'cards' ? (
          <StudioMembers members={paginatedMembers} profiles={memberSkillProfiles} catalog={skillsCatalog} open={member => void onOpenMemberCard(member)}/>
        ) : (
          <MemberTable members={paginatedMembers} profiles={memberSkillProfiles} catalog={skillsCatalog} sortCriteria={membersSortCriteria} onSort={onMembersSort} open={member => void onOpenMemberCard(member)}/>
        )}
        {filteredMembers.length > 0 && <Pagination total={filteredMembers.length} page={currentMembersPage} pageSize={membersPageSize} navigationLabel="Страницы игроков" onPageChange={page => {
          setMembersPage(page)
          membersHeadingRef.current?.scrollIntoView({ block: 'start' })
        }} onPageSizeChange={size => { setMembersPageSize(size); setMembersPage(1) }}/>}
      </section>
    )
  }

  function renderTemplateEditor() {
    if (!activeTemplateName) {
      return null
    }

    return (
      <section className="content-card">
        <div className="template-head">
          <button
            className="btn-secondary"
            onClick={() =>
              navigateTo({
                chatID: activeChatID,
                section: 'templates',
                templateView: 'list',
                templateName: null,
                eventView: 'list',
                eventID: null,
              })
            }
          >
            ← К списку шаблонов
          </button>
          <h3>Редактирование: {activeTemplateName}</h3>
        </div>

        {templateEditorLoading ? <p className="muted">Загрузка шаблона...</p> : null}


        {!templateEditorLoading && !templateEditorError ? (
          <form className="form-grid" onSubmit={onSaveTemplateEditor}>
            <h4>Название</h4>
            <input
              value={templateEditor.name}
              onChange={(e) => setTemplateEditor((prev) => ({ ...prev, name: e.target.value }))}
              required
              disabled={activeTemplateName?.trim().toLowerCase() === 'регистрация'}
            />
            <h4>Вопрос</h4>
            <input
              value={templateEditor.question}
              onChange={(e) => setTemplateEditor((prev) => ({ ...prev, question: e.target.value }))}
              required
            />
            <h4>Варианты</h4>
            <div className="option-list">
              {templateEditor.options.map((option, index) => (
                <div className="option-row" key={`editor-option-${index}`}>
                  <input
                    value={option}
                    onChange={(e) => updateTemplateEditorOption(index, e.target.value)}
                    placeholder={`Вариант ${index + 1}`}
                  />
                  <label className="option-weight">
                    <span>K</span>
                    <input
                      type="number"
                      min="1"
                      step="1"
                      value={templateEditor.weights[index] ?? 1}
                      onChange={(e) => updateTemplateEditorWeight(index, e.target.value)}
                    />
                  </label>
                  <label className="option-accounting">
                    <Checkbox
                      checked={Boolean(templateEditor.counted[index])}
                      onChange={() => toggleTemplateEditorCounted(index)}
                    />
                    <span>Учёт</span>
                  </label>
                  <button
                    type="button"
                    className="icon-btn danger"
                    onClick={() => removeTemplateEditorOption(index)}
                    disabled={templateEditor.options.length <= 2}
                  >
                    -
                  </button>
                </div>
              ))}
            </div>
            <button type="button" className="icon-btn add" onClick={addTemplateEditorOption}>
              +
            </button>
            <div className="split-forms">
              <button type="submit">Сохранить изменения</button>
              {activeTemplateName?.trim().toLowerCase() === 'регистрация' ? null : (
                <button type="button" className="btn-danger" onClick={() => void onDeleteTemplateEditor()}>
                  Удалить шаблон
                </button>
              )}
            </div>
          </form>
        ) : null}
      </section>
    )
  }

  function renderTemplateCreate() {
    return (
      <section className="content-card">
        <div className="template-head">
          <button
            className="btn-secondary"
            onClick={() =>
              navigateTo({
                chatID: activeChatID,
                section: 'templates',
                templateView: 'list',
                templateName: null,
                eventView: 'list',
                eventID: null,
              })
            }
          >
            ← К списку шаблонов
          </button>
          <h3>Создать шаблон</h3>
        </div>

        <form className="form-grid" onSubmit={onCreateTemplate}>
          <h4>Название</h4>
          <input
            placeholder="Название"
            value={templateForm.name}
            onChange={(e) => setTemplateForm((prev) => ({ ...prev, name: e.target.value }))}
            required
          />
          <h4>Вопрос</h4>
          <input
            placeholder="Вопрос"
            value={templateForm.question}
            onChange={(e) => setTemplateForm((prev) => ({ ...prev, question: e.target.value }))}
            required
          />
          <h4>Варианты</h4>
          <div className="option-list">
            {templateForm.options.map((option, index) => (
              <div className="option-row" key={`create-option-${index}`}>
                <input
                  placeholder={`Вариант ${index + 1}`}
                  value={option}
                  onChange={(e) => updateTemplateFormOption(index, e.target.value)}
                />
                <label className="option-weight">
                  <span>K</span>
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={templateForm.weights[index] ?? 1}
                    onChange={(e) => updateTemplateFormWeight(index, e.target.value)}
                  />
                </label>
                <label className="option-accounting">
                  <Checkbox
                    checked={Boolean(templateForm.counted[index])}
                    onChange={() => toggleTemplateFormCounted(index)}
                  />
                  <span>Учёт</span>
                </label>
                <button
                  type="button"
                  className="icon-btn danger"
                  onClick={() => removeTemplateFormOption(index)}
                  disabled={templateForm.options.length <= 2}
                >
                  -
                </button>
              </div>
            ))}
          </div>
          <button type="button" className="icon-btn add" onClick={addTemplateFormOption}>
            +
          </button>
          <button type="submit">Сохранить шаблон</button>
        </form>
      </section>
    )
  }

  function renderTemplates() {
    if (activeTemplateView === 'edit' && activeTemplateName) {
      return renderTemplateEditor()
    }
    if (activeTemplateView === 'create') {
      return renderTemplateCreate()
    }

    return (
      <section className="content-card">
        <div className="template-head">
          <h3>Шаблоны голосований</h3>
          <button
            className="btn-secondary"
            onClick={() =>
              navigateTo({
                chatID: activeChatID,
                section: 'templates',
                templateView: 'create',
                templateName: null,
                eventView: 'list',
                eventID: null,
              })
            }
          >
            Создать шаблон
          </button>
        </div>

        <div className="list-block studio-poll-template-grid">
          {templates.length ? (
            templates.map((template) => (
              <div
                className="list-row clickable"
                key={template.name}
                role="button"
                tabIndex={0}
                aria-label={`Открыть шаблон: ${template.name}`}
                onClick={() =>
                  navigateTo({
                    chatID: activeChatID,
                    section: 'templates',
                    templateView: 'edit',
                    templateName: template.name,
                    eventView: 'list',
                    eventID: null,
                  })
                }
                onKeyDown={(e) => {
                  if (e.key !== 'Enter' && e.key !== ' ') return
                  e.preventDefault()
                  navigateTo({
                    chatID: activeChatID,
                    section: 'templates',
                    templateView: 'edit',
                    templateName: template.name,
                    eventView: 'list',
                    eventID: null,
                  })
                }}
              >
                <div><span className="studio-template-symbol"><Icon name="poll" size={24}/></span>
                  <strong>{template.name}</strong>
                  <p>{template.question}</p>
                  <p className="muted">Учёт вариантов: {template.countedOptionsCount}</p>
                </div>
                <div className="list-actions">
                  {template.name.trim().toLowerCase() === 'регистрация' ? null : (
                    <button
                      className="btn-danger"
                      onClick={(e) => {
                        e.stopPropagation()
                        onDeleteTemplate(template.name)
                      }}
                    >
                      Удалить
                    </button>
                  )}
                </div>
              </div>
            ))
          ) : (
            <p className="muted">Шаблонов пока нет</p>
          )}
        </div>
      </section>
    )
  }

  function renderEventCreate() {
    return (
      <section className="content-card">
        <div className="template-head">
          <button
            className="btn-secondary"
            onClick={() =>
              navigateTo({
                chatID: activeChatID,
                section: 'event_templates',
                templateView: 'list',
                templateName: null,
                eventView: 'list',
                eventID: null,
              })
            }
          >
            ← К шаблонам событий
          </button>
          <h3>Создать шаблон</h3>
        </div>

        <form className="event-editor-form" onSubmit={onCreateEvent}>
          <div className="event-editor-row event-editor-row-main">
            <label className="field">
              <span>Название</span>
              <input
                placeholder="Название события"
                value={eventForm.name}
                onChange={(e) => setEventForm((prev) => ({ ...prev, name: e.target.value }))}
                required
              />
            </label>
            <label className="field">
              <span>Тип события</span>
              <Select className="ui-select" value={eventForm.eventType} onChange={(e) => setEventForm((prev) => ({ ...prev, eventType: e.target.value as 'training' | 'activity' }))}>
                <option value="training">Тренировка</option>
                <option value="activity">Мероприятие</option>
              </Select>
            </label>
            <label className="field">
              <span>День недели</span>
              <Select
                className="ui-select"
                value={eventForm.weekday}
                onChange={(e) => setEventForm((prev) => ({ ...prev, weekday: e.target.value }))}
                required
              >
                {weekdayOptions.map((day) => (
                  <option key={day.value} value={day.value}>
                    {day.short} · {day.full}
                  </option>
                ))}
              </Select>
            </label>
            <label className="field">
              <span>Начало</span>
              <input
                type="time"
                value={eventForm.startAt}
                onChange={(e) => setEventForm((prev) => ({ ...prev, startAt: e.target.value }))}
                required
              />
            </label>
            <label className="field">
              <span>Конец</span>
              <input
                type="time"
                value={eventForm.endAt}
                onChange={(e) => setEventForm((prev) => ({ ...prev, endAt: e.target.value }))}
                required
              />
            </label>
            <label className="field">
              <span>Стоимость</span>
              <input
                type="number"
                min="0"
                step="0.01"
                placeholder="Например: 4000"
                value={eventForm.costAmount}
                onChange={(e) => setEventForm((prev) => ({ ...prev, costAmount: e.target.value }))}
              />
            </label>
          </div>
          <div className="event-editor-row event-editor-row-poll">
            <label className="field">
              <span>День публикации опроса</span>
              <Select
                className="ui-select"
                value={eventForm.publishWeekday}
                onChange={(e) => setEventForm((prev) => ({ ...prev, publishWeekday: e.target.value }))}
                required
              >
                {weekdayOptions.map((day) => (
                  <option key={day.value} value={day.value}>
                    {day.short} · {day.full}
                  </option>
                ))}
              </Select>
            </label>
            <label className="field">
              <span>Публикация опроса</span>
              <input
                type="time"
                value={eventForm.publishAt}
                onChange={(e) => setEventForm((prev) => ({ ...prev, publishAt: e.target.value }))}
                required
              />
            </label>
            <label className="field">
              <span>Шаблон опроса</span>
              <Select
                className="ui-select"
                value={eventForm.templateName}
                onChange={(e) => setEventForm((prev) => ({ ...prev, templateName: e.target.value }))}
                required
              >
                <option value="">Выбери шаблон</option>
                {templateNames.map((name) => (
                  <option key={name} value={name}>
                    {name}
                  </option>
                ))}
              </Select>
            </label>
            <label className="field">
              <span>Максимум мест</span>
              <input
                type="number"
                min="1"
                step="1"
                value={eventForm.maxPlaces}
                onChange={(e) => setEventForm((prev) => ({ ...prev, maxPlaces: e.target.value }))}
                required
              />
            </label>
          </div>
          <div className="event-editor-row event-editor-row-announcement-full">
            <label className="field announcement-text">
              <span>Анонс</span>
              <textarea
                placeholder="Текст анонса для публикации в группе"
                value={eventForm.announcementText}
                onChange={(e) => setEventForm((prev) => ({ ...prev, announcementText: e.target.value }))}
              />
            </label>
          </div>
          {activeChatID !== null && <EventMentionsEditor key={activeChatID} chatID={activeChatID} value={eventForm.mentions} onChange={mentions => setEventForm(prev => ({ ...prev, mentions }))}/>}
          <section className="settings-card settings-card-settings">
            <h4>Настройки</h4>
            <div className="settings-grid">
              <div className="settings-group">
                <p className="settings-group-title">Анонс</p>
                <div className="settings-row settings-row-2">
                  <label className="toggle-field toggle-field-inline">
                    <Checkbox
                      checked={eventForm.announcementEnabled}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, announcementEnabled: e.target.checked }))}
                    />
                    <span>Публиковать анонс</span>
                  </label>
                  <label className="field">
                    <span>Публиковать за</span>
                    <Select
                      className="ui-select"
                      value={eventForm.announcementLeadMinutes}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, announcementLeadMinutes: e.target.value }))}
                    >
                      {announcementLeadOptions.map((item) => (
                        <option key={item.value} value={item.value}>
                          {item.label}
                        </option>
                      ))}
                    </Select>
                  </label>
                </div>
              </div>
              <div className="settings-group">
                <p className="settings-group-title">Расчёт</p>
                <div className="settings-row settings-row-2">
                  <label className="toggle-field toggle-field-inline">
                    <Checkbox
                      checked={eventForm.settlementEnabled}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, settlementEnabled: e.target.checked }))}
                      disabled
                    />
                    <span>Публиковать расчёт</span>
                  </label>
                  <label className="toggle-field toggle-field-inline">
                    <Checkbox
                      checked={eventForm.settlementPublishBefore}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, settlementPublishBefore: e.target.checked }))}
                    />
                    <span>Перед началом</span>
                  </label>
                  <label className="toggle-field toggle-field-inline">
                    <Checkbox
                      checked={eventForm.settlementPublishAfter}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, settlementPublishAfter: e.target.checked }))}
                    />
                    <span>После</span>
                  </label>
                </div>
                <p className="muted settings-note">
                  Доступно после привязки шаблона с отмеченными вариантами «Учёт».
                </p>
              </div>
              <div className="settings-group">
                <p className="settings-group-title">Порог события</p>
                <div className="settings-row settings-row-2">
                  <label className="field">
                    <span>Минимальное количество голосов, чтобы событие состоялось</span>
                    <input
                      type="number"
                      min="0"
                      step="1"
                      value={eventForm.minVotesToHold}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, minVotesToHold: e.target.value }))}
                    />
                  </label>
                  <label className="field">
                    <span>Отменять за (минут до начала)</span>
                    <input
                      type="number"
                      min="1"
                      step="1"
                      value={eventForm.cancelLeadMinutes}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, cancelLeadMinutes: e.target.value }))}
                    />
                  </label>
                  <div className="field">
                    <span className="field-label-placeholder" aria-hidden="true">&nbsp;</span>
                    <label className="toggle-field toggle-field-inline">
                      <Checkbox
                        checked={eventForm.cancelNotifyEnabled}
                        onChange={(e) => setEventForm((prev) => ({ ...prev, cancelNotifyEnabled: e.target.checked }))}
                      />
                      <span>Уведомлять об отмене</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>
          </section>
          <button type="submit">Создать шаблон</button>
        </form>
      </section>
    )
  }

  function renderEventEditor() {
    if (!selectedEvent || activeEventID === null) {
      return (
        <section className="content-card">
          <div className="template-head">
            <button
              className="btn-secondary"
              onClick={() =>
                navigateTo({
                  chatID: activeChatID,
                  section: 'event_templates',
                  templateView: 'list',
                  templateName: null,
                  eventView: 'list',
                  eventID: null,
                })
              }
            >
              ← К шаблонам событий
            </button>
            <h3>Событие не найдено</h3>
          </div>
        </section>
      )
    }

    return (
      <section className="content-card">
        <div className="template-head">
          <button
            className="btn-secondary"
            onClick={() =>
              navigateTo({
                chatID: activeChatID,
                section: 'event_templates',
                templateView: 'list',
                templateName: null,
                eventView: 'list',
                eventID: null,
              })
            }
          >
            ← К шаблонам событий
          </button>
          <h3>
            Шаблон: {selectedEvent.name}
          </h3>
          <div className="list-actions">
            <button type="button" className="btn-secondary" onClick={() => void onCreateFromTemplate(selectedEvent.id)}>
              Создать встречу
            </button>
            <button type="button" className="btn-secondary" onClick={() => void onToggleEventActivity()}>
              Статистика
            </button>
            <button type="button" className="btn-secondary" onClick={() => void onToggleEventPublishEnabled()}>
              {selectedEvent.publishEnabled ? 'Деактивировать' : 'Активировать'}
            </button>
            <button type="button" className="btn-danger btn-icon" title="В архив" aria-label="В архив" onClick={() => void onArchiveEvent(selectedEvent.id)}>
              <span aria-hidden>🗃</span>
            </button>
          </div>
        </div>

        <div className="event-card">
          <p>Тип: {selectedEvent.eventType === 'training' ? 'Тренировка' : 'Мероприятие'}</p>
          <p>
            День события: {weekdayLabel(selectedEvent.startWeekday)} | Время: {toHourMinute(selectedEvent.startTime)} - {toHourMinute(selectedEvent.endTime)}
          </p>
          <p>
            Опрос: {selectedEvent.pollTemplate || 'не привязан'} | Публикация: {weekdayLabel(selectedEvent.pollPublishWeekday || selectedEvent.startWeekday)}{' '}
            {toHourMinute(selectedEvent.pollPublishTime)} | Стоимость: {formatMoney(selectedEvent.costAmount)}
          </p>
          <p>
            Анонс: {selectedEvent.announcementEnabled ? `включен (${announcementLeadLabel(selectedEvent.announcementLeadMinutes || 60)})` : 'выключен'}
          </p>
          <p>Публикации: {selectedEvent.publishEnabled ? 'активны' : 'отключены'}</p>
          <p>
            Расчёт: {selectedEvent.settlementEnabled ? 'включен' : 'выключен'} | Перед началом:{' '}
            {selectedEvent.settlementPublishBefore ? 'да' : 'нет'} | После: {selectedEvent.settlementPublishAfter ? 'да' : 'нет'}
          </p>
          <p>
            Отмена: минимум {selectedEvent.minVotesToHold || 0} голосов, проверка за {selectedEvent.cancelLeadMinutes || 180} мин | Уведомление:{' '}
            {selectedEvent.cancelNotifyEnabled ? 'да' : 'нет'}
          </p>
        </div>
        {showEventActivity ? (
          <section className="content-card">
            <h4>Статистика события</h4>
            {eventActivityLoading ? <p className="muted">Загрузка активности...</p> : null}

            {!eventActivityLoading && !eventActivityError && eventActivity ? (
              <div className="detail-grid">
                <div>
                  <span>Опросов опубликовано</span>
                  <strong>{eventActivity.pollsTotal}</strong>
                </div>
                <div>
                  <span>Голосов получено</span>
                  <strong>{eventActivity.votesTotal}</strong>
                </div>
                <div>
                  <span>Расчётов опубликовано</span>
                  <strong>{eventActivity.settlementsTotal}</strong>
                </div>
                <div>
                  <span>Анонсов опубликовано</span>
                  <strong>{eventActivity.announcementsTotal}</strong>
                </div>
              </div>
            ) : null}
          </section>
        ) : null}

        <form className="event-editor-form" onSubmit={onSaveEventConfiguration}>
          <div className="event-editor-row event-editor-row-main">
            <label className="field">
              <span>Название</span>
              <input value={eventEditor.name} onChange={(e) => setEventEditor((prev) => ({ ...prev, name: e.target.value }))} required />
            </label>
            <label className="field">
              <span>Тип события</span>
              <Select className="ui-select" value={eventEditor.eventType} onChange={(e) => setEventEditor((prev) => ({ ...prev, eventType: e.target.value as 'training' | 'activity' }))}>
                <option value="training">Тренировка</option>
                <option value="activity">Мероприятие</option>
              </Select>
            </label>
            <label className="field">
              <span>День недели</span>
              <Select
                className="ui-select"
                value={eventEditor.weekday}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, weekday: e.target.value }))}
                required
              >
                {weekdayOptions.map((day) => (
                  <option key={day.value} value={day.value}>
                    {day.short} · {day.full}
                  </option>
                ))}
              </Select>
            </label>
            <label className="field">
              <span>Начало</span>
              <input
                type="time"
                value={eventEditor.startAt}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, startAt: e.target.value }))}
                required
              />
            </label>
            <label className="field">
              <span>Конец</span>
              <input
                type="time"
                value={eventEditor.endAt}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, endAt: e.target.value }))}
                required
              />
            </label>
            <label className="field">
              <span>Стоимость</span>
              <input
                type="number"
                min="0"
                step="0.01"
                placeholder="необязательно"
                value={eventEditor.costAmount}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, costAmount: e.target.value }))}
              />
            </label>
          </div>

          <div className="event-editor-row event-editor-row-poll">
            <label className="field">
              <span>Шаблон опроса</span>
              <Select
                className="ui-select"
                value={eventEditor.templateName}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, templateName: e.target.value }))}
              >
                <option value="">Не выбран</option>
                {templateNames.map((name) => (
                  <option key={name} value={name}>
                    {name}
                  </option>
                ))}
              </Select>
            </label>
            <label className="field">
              <span>День публикации</span>
              <Select
                className="ui-select"
                value={eventEditor.publishWeekday}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, publishWeekday: e.target.value }))}
                required={eventEditorHasTemplate}
                disabled={!eventEditorHasTemplate}
              >
                {weekdayOptions.map((day) => (
                  <option key={day.value} value={day.value}>
                    {day.short} · {day.full}
                  </option>
                ))}
              </Select>
            </label>
            <label className="field">
              <span>Время публикации</span>
              <input
                type="time"
                value={eventEditor.publishAt}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, publishAt: e.target.value }))}
                required={eventEditorHasTemplate}
                disabled={!eventEditorHasTemplate}
              />
            </label>
            <label className="field">
              <span>Максимум мест</span>
              <input
                type="number"
                min="1"
                step="1"
                value={eventEditor.maxPlaces}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, maxPlaces: e.target.value }))}
                required
              />
            </label>
          </div>

          <div className="event-editor-row event-editor-row-announcement-full">
            <label className="field announcement-text">
              <span>Анонс</span>
              <textarea
                placeholder="Текст анонса для публикации в группе"
                value={eventEditor.announcementText}
                onChange={(e) => setEventEditor((prev) => ({ ...prev, announcementText: e.target.value }))}
              />
            </label>
          </div>
          {activeChatID !== null && <EventMentionsEditor key={activeChatID} chatID={activeChatID} value={eventEditor.mentions} onChange={mentions => setEventEditor(prev => ({ ...prev, mentions }))}/>}
          <section className="settings-card settings-card-settings">
            <h4>Настройки</h4>
            <div className="settings-grid">
              <div className="settings-group">
                <p className="settings-group-title">Анонс</p>
                <div className="settings-row settings-row-2">
                  <label className="field toggle-field">
                    <Checkbox
                      checked={eventEditor.announcementEnabled}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, announcementEnabled: e.target.checked }))}
                    />
                    <span>Публиковать анонс</span>
                  </label>
                  <label className="field">
                    <span>Публиковать за</span>
                    <Select
                      className="ui-select"
                      value={eventEditor.announcementLeadMinutes}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, announcementLeadMinutes: e.target.value }))}
                    >
                      {announcementLeadOptions.map((item) => (
                        <option key={item.value} value={item.value}>
                          {item.label}
                        </option>
                      ))}
                    </Select>
                  </label>
                </div>
              </div>
              <div className="settings-group">
                <p className="settings-group-title">Расчёт</p>
                <div className="settings-row settings-row-3">
                  <label className="field toggle-field">
                    <Checkbox
                      checked={eventEditor.settlementEnabled}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, settlementEnabled: e.target.checked }))}
                      disabled={!eventEditorCanEnableSettlement}
                    />
                    <span>Публиковать расчёт</span>
                  </label>
                  <label className="field toggle-field">
                    <Checkbox
                      checked={eventEditor.settlementPublishBefore}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, settlementPublishBefore: e.target.checked }))}
                    />
                    <span>Перед началом</span>
                  </label>
                  <label className="field toggle-field">
                    <Checkbox
                      checked={eventEditor.settlementPublishAfter}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, settlementPublishAfter: e.target.checked }))}
                    />
                    <span>После</span>
                  </label>
                </div>
                {!eventEditorCanEnableSettlement ? (
                  <p className="muted settings-note">Выбери шаблон и отметь минимум один вариант с флагом «Учёт».</p>
                ) : null}
              </div>
              <div className="settings-group">
                <p className="settings-group-title">Порог события</p>
                <div className="settings-row settings-row-3">
                  <label className="field">
                    <span>Минимальное количество голосов, чтобы событие состоялось</span>
                    <input
                      type="number"
                      min="0"
                      step="1"
                      value={eventEditor.minVotesToHold}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, minVotesToHold: e.target.value }))}
                    />
                  </label>
                  <label className="field">
                    <span>Отменять за (минут до начала)</span>
                    <input
                      type="number"
                      min="1"
                      step="1"
                      value={eventEditor.cancelLeadMinutes}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, cancelLeadMinutes: e.target.value }))}
                    />
                  </label>
                  <div className="field">
                    <span className="field-label-placeholder" aria-hidden="true">&nbsp;</span>
                    <label className="toggle-field toggle-field-inline">
                      <Checkbox
                        checked={eventEditor.cancelNotifyEnabled}
                        onChange={(e) => setEventEditor((prev) => ({ ...prev, cancelNotifyEnabled: e.target.checked }))}
                      />
                      <span>Уведомлять об отмене</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <button type="submit" disabled={!eventEditorDirty}>
            Сохранить
          </button>
        </form>
      </section>
    )
  }

  function renderEvents() {
    if (activeEventView === 'create') {
      return renderEventCreate()
    }
    if (activeEventView === 'edit') {
      return renderEventEditor()
    }

    return (
      <section className="content-card">
        <div className="template-head">
          <h3>Шаблоны событий</h3>
          <div className="list-actions">
            <button className="btn-secondary" onClick={() => setEventsMode(eventsMode === 'active' ? 'archived' : 'active')}>
              {eventsMode === 'active' ? 'Архив' : 'Активные'}
            </button>
            <button
              className="btn-secondary"
              onClick={() =>
                navigateTo({
                  chatID: activeChatID,
                  section: 'event_templates',
                  templateView: 'list',
                  templateName: null,
                  eventView: 'create',
                  eventID: null,
                })
              }
            >
              Создать шаблон
            </button>
          </div>
        </div>

        <div className="form-grid form-grid-3 members-filters">
          <label className="field">
            <span>Тип события</span>
            <Select value={eventsTypeFilter} onChange={(e) => setEventsTypeFilter(e.target.value as '' | 'training' | 'activity')}>
              <option value="">Все типы</option>
              <option value="training">Тренировка</option>
              <option value="activity">Мероприятие</option>
            </Select>
          </label>
          <label className="field">
            <span>Режим</span>
            <input value={eventsMode === 'active' ? 'Активные' : 'Архив'} readOnly />
          </label>
          <label className="field">
            <span>Результат</span>
            <input value={`${displayedEvents.length} шт.`} readOnly />
          </label>
        </div>

        <div className="list-block">
          {displayedEvents.length ? (
            <StudioEventTemplates items={displayedEvents} archived={eventsMode==='archived'} open={id=>openStudioSection('event_templates',{eventView:'edit',eventID:id})} create={id=>void onCreateFromTemplate(id)} archive={id=>void onArchiveEvent(id)} restore={id=>void onUnarchiveEvent(id)}/>
          ) : (
            <p className="muted">{eventsTypeFilter ? 'По выбранному типу событий нет' : eventsMode === 'active' ? 'Событий пока нет' : 'Архив пуст'}</p>
          )}
        </div>
      </section>
    )
  }

  function renderHistory() {
    if (activeChatID === null) {
      return <section className="content-card">Выбери организацию</section>
    }

    if (activeHistoryEventID === null) {
      return (
        <section className="content-card">
          <div className="template-head" ref={historyHeadingRef}>
            <h3>События</h3>
            <div className="studio-view-switch" role="group" aria-label="Вид событий">
              <button type="button" className={eventsView === 'cards' ? 'active' : ''} aria-pressed={eventsView === 'cards'} onClick={() => setEventsView('cards')}><Icon name="grid" size={16}/>Плитки</button>
              <button type="button" className={eventsView === 'list' ? 'active' : ''} aria-pressed={eventsView === 'list'} onClick={() => setEventsView('list')}><Icon name="list" size={16}/>Список</button>
            </div>
          </div>
          <div className="form-grid form-grid-3 members-filters">
            <label className="field">
              <span>Статус</span>
              <Select
                value={historyStatusFilter}
                onChange={(e) =>
                  setHistoryStatusFilter(e.target.value as '' | 'in_voting' | 'on_distribution' | 'on_review' | 'completed' | 'not_held')
                }
              >
                <option value="">Все статусы</option>
                <option value="in_voting">В голосовании</option>
                <option value="on_distribution">На распределении</option>
                <option value="on_review">На проверке</option>
                <option value="not_held">Не состоялось</option>
                <option value="completed">Завершено</option>
              </Select>
            </label>
            <label className="field">
              <span>Период</span>
              <input value="Последние события" readOnly />
            </label>
            <label className="field">
              <span>Результат</span>
              <input value={`${filteredEventHistory.length} шт.`} readOnly />
            </label>
          </div>
          {eventHistoryLoading ? <p className="muted">Загрузка истории событий...</p> : null}

          {!eventHistoryLoading && !eventHistoryError && eventHistory.length === 0 ? <p className="muted">Событий пока нет</p> : null}

          {!eventHistoryLoading && filteredEventHistory.length > 0 ? <StudioEvents items={paginatedEventHistory} view={eventsView} open={id=>openStudioSection('events',{historyEventID:id})}/> : null}
          {!eventHistoryLoading && !eventHistoryError && filteredEventHistory.length > 0 ? <Pagination total={filteredEventHistory.length} page={currentHistoryPage} pageSize={historyPageSize} onPageChange={page => {
            setHistoryPage(page)
            historyHeadingRef.current?.scrollIntoView({ block: 'start' })
          }} onPageSizeChange={size => { setHistoryPageSize(size); setHistoryPage(1) }}/>: null}
          {!eventHistoryLoading && eventHistory.length > 0 && filteredEventHistory.length === 0 ? (
            <p className="muted">По выбранному статусу событий нет</p>
          ) : null}
        </section>
      )
    }

    return (
      <section className="content-card studio-event-detail">
        <div className="studio-event-backbar">
          <button
            className="studio-event-back"
            onClick={() =>
              navigateTo({
                chatID: activeChatID,
                section: 'events',
                historyEventID: null,
                templateView: 'list',
                templateName: null,
                eventView: 'list',
                eventID: null,
              })
            }
          >
            <Icon name="back" size={17}/>Все события
          </button>
          {selectedHistoryEvent && <button type="button" className="btn-danger studio-event-delete" onClick={() => void onDeleteEventInstanceForTest()}>Удалить событие</button>}
        </div>
        {eventHistoryLoading ? <p className="muted">Загрузка истории событий...</p> : null}


        {!eventHistoryLoading && !selectedHistoryEvent ? <p className="muted">Событие не найдено</p> : null}

	        {selectedHistoryEvent ? (
	          <section className="team-split-board studio-event-board">
            <EventDetailHeader event={selectedHistoryEvent}/>

            <div className="history-tabs studio-event-tabs" role="group" aria-label="Разделы события">
              <button
                type="button"
                className={historyDetailTab === 'distribution' ? 'history-tab active' : 'history-tab'}
                aria-pressed={historyDetailTab === 'distribution'}
                onClick={() => setHistoryDetailTab('distribution')}
              >
                <Icon name="users" size={17}/>Состав
              </button>
              <button
                type="button"
                className={historyDetailTab === 'votes' ? 'history-tab active' : 'history-tab'}
                aria-pressed={historyDetailTab === 'votes'}
                onClick={() => setHistoryDetailTab('votes')}
                disabled={selectedHistoryPostID === null}
                title={selectedHistoryPostID === null ? 'Сначала выбери опрос' : undefined}
              >
                <Icon name="poll" size={17}/>Голоса
              </button>
              {isAdmin && selectedHistoryEvent.eventType === 'training' && <button type="button" className={historyDetailTab === 'attendance' ? 'history-tab active' : 'history-tab'} aria-pressed={historyDetailTab === 'attendance'} onClick={() => setHistoryDetailTab('attendance')}><Icon name="users" size={17}/>Посещаемость</button>}
<button
                type="button"
                className={historyDetailTab === 'billing' ? 'history-tab active' : 'history-tab'}
                aria-pressed={historyDetailTab === 'billing'}
                onClick={() => setHistoryDetailTab('billing')}
              >
                <Icon name="wallet" size={17}/>Оплата
              </button>
              <button
                type="button"
                className={historyDetailTab === 'sets' ? 'history-tab active' : 'history-tab'}
                aria-pressed={historyDetailTab === 'sets'}
                onClick={() => setHistoryDetailTab('sets')}
                disabled={selectedHistoryPostID === null}
                title={selectedHistoryPostID === null ? 'Сначала выбери опрос' : undefined}
              >
                <Icon name="score" size={17}/>Партии
              </button>
            </div>

            {historyDetailTab !== 'billing' && historyDetailTab !== 'attendance' && !eventPollHistoryError && <EventPollContext
              polls={eventPollHistory}
              selectedPostID={selectedHistoryPostID}
              loading={eventPollHistoryLoading}
              onSelect={setSelectedHistoryPostID}
              mode={historyDetailTab === 'votes' ? 'votes' : 'roster'}
            />}

            {historyDetailTab === 'distribution' ? (
              <>
                {selectedHistoryPostID !== null ? (
              <>
                {teamSplitLoading ? <p className="muted">Загружаю участников...</p> : null}

                {!teamSplitLoading && teamSplit ? (
                  <>
                    {(() => {
                      const activeTeams = activeTeamCodes(teamSplit.players, teamCEnabled)
                      const formationRows = activeTeams
                        .map((teamCode) => ({
                          teamCode,
                          formation: teamSplit.formations?.[teamCode],
                        }))
                        .filter(
                          (
                            item,
                          ): item is {
                            teamCode: ActiveTeamCode
                            formation: {
                              scheme: string
                              analysis: string
                              indicators?: Array<{
                                code: string
                                name: string
                                result: string
                                reference: string
                                passed: boolean
                              }>
                            }
                          } =>
                            Boolean(
                              item.formation &&
                                (item.formation.scheme || item.formation.analysis || (item.formation.indicators?.length ?? 0) > 0),
                            ),
                        )
                      const selectedTouchPlayer =
                        draggedPlayerID !== null ? teamSplit.players.find((p) => p.userID === draggedPlayerID) ?? null : null
                      const lineupLabel = (index: number) => (index < 6 ? String(index + 1) : 'З')
                      const unassignedPlayers = teamSplit.players.filter((player) => player.team === 'unassigned')
                      const renderTeamColumn = (teamCode: ActiveTeamCode) => (
                        <div
                          className={`team-column ${draggedPlayerID !== null ? 'team-column-drop' : ''}`}
                          data-team={teamCode}
                          onClick={() => {
                            if (!touchDragMode || draggedPlayerID === null) return
                            onDropToTeam(teamCode)
                          }}
                          onDragOver={(e) => {
                            e.preventDefault()
                          }}
                          onDrop={(e) => {
                            e.preventDefault()
                            onDropToTeam(teamCode)
                          }}
                        >
                          <div className="team-column-head">
                            <div>
                              <h5><span className="studio-team-letter" aria-hidden="true">{teamCode}</span>{`Команда ${teamCode}`}</h5>
                              {teamSplit.formations?.[teamCode]?.scheme ? (
                                <small className="muted" title={teamSplit.formations?.[teamCode]?.analysis || ''}>
                                  Схема: {teamSplit.formations?.[teamCode]?.scheme}
                                </small>
                              ) : null}
                            </div>
                            <span className="studio-team-count">{playerCountLabel(teamSplit.players.filter(player => player.team === teamCode).length)}</span>
                            {teamCode === 'C' ? (
                              <button type="button" className="team-remove-btn" onClick={removeTeamC} title="Убрать команду C" aria-label="Убрать команду C">
                                −
                              </button>
                            ) : null}
                          </div>
                          <div className="team-players">
                            {(() => {
                              const teamPlayers = teamSplit.players.filter((player) => player.team === teamCode)
                              return teamPlayers.map((player, idx) => (
                                <article
                                  className={draggedPlayerID === player.userID ? 'team-player-card selected' : 'team-player-card'}
                                  key={`${teamCode}-${player.userID}`}
                                  draggable={!touchDragMode}
                                  onDragStart={() => onDragStartTeamPlayer(player.userID)}
                                  onDragEnd={() => setDraggedPlayerID(null)}
                                  onClick={() => {
                                    if (!touchDragMode) return
                                    setDraggedPlayerID((prev) => (prev === player.userID ? null : player.userID))
                                  }}
                                >
                                  <p>{playerDisplayName(player)}</p>
                                  <small>
                                    {player.choiceLabel} · рейтинг {player.rating.toFixed(1)} · позиция {lineupLabel(idx)}
                                  </small>
                                </article>
                              ))
                            })()}
                            {teamSplit.players.filter((player) => player.team === teamCode).length === 0 ? <p className="studio-team-empty">{touchDragMode ? 'Выберите игрока и нажмите на эту команду' : 'Перетащите сюда игроков'}</p> : null}
                          </div>
                        </div>
                      )

                      return (
                        <>
                          {touchDragMode && selectedTouchPlayer ? (
                            <p className="muted" style={{ margin: '6px 0 10px' }}>
                              Выбрано: <strong>{playerDisplayName(selectedTouchPlayer)}</strong>. Нажми на колонку команды, чтобы переместить.
                            </p>
                          ) : null}
                          <div className="team-layout">
                            <div
                              className={`team-column team-column-unassigned ${draggedPlayerID !== null ? 'team-column-drop' : ''}`}
                              onClick={() => {
                                if (!touchDragMode || draggedPlayerID === null) return
                                onDropToTeam('unassigned')
                              }}
                              onDragOver={(e) => {
                                e.preventDefault()
                              }}
                              onDrop={(e) => {
                                e.preventDefault()
                                onDropToTeam('unassigned')
                              }}
                            >
                              <div className="team-column-head"><h5>Без команды</h5><span className="studio-team-count">{playerCountLabel(unassignedPlayers.length)}</span></div>
                              <div className="team-players team-players-unassigned">
                                {unassignedPlayers.map((player) => (
                                  <article
                                    className={draggedPlayerID === player.userID ? 'team-player-card selected' : 'team-player-card'}
                                    key={`unassigned-${player.userID}`}
                                    draggable={!touchDragMode}
                                    onDragStart={() => onDragStartTeamPlayer(player.userID)}
                                    onDragEnd={() => setDraggedPlayerID(null)}
                                    onClick={() => {
                                      if (!touchDragMode) return
                                      setDraggedPlayerID((prev) => (prev === player.userID ? null : player.userID))
                                    }}
                                  >
                                    <p>{playerDisplayName(player)}</p>
                                    <small>{player.choiceLabel} · рейтинг {player.rating.toFixed(1)} · позиция -</small>
                                  </article>
                                ))}
                                {unassignedPlayers.length === 0 ? <p className="studio-unassigned-empty">Все игроки распределены</p> : null}
                              </div>
                            </div>

                            <div className="studio-roster-toolbar"><span>Составы команд</span>{activeTeams.length === 2 && !activeTeams.includes('C') ? (
                              <button type="button" className="btn-secondary" onClick={() => setTeamCEnabled(true)}><Icon name="plus" size={15}/>Добавить команду C</button>
                            ) : null}</div>
                            <div className={activeTeams.length === 2 ? 'team-double-row' : 'team-triple-row'}>
                              {activeTeams.map((teamCode) => (
                                <div key={teamCode}>{renderTeamColumn(teamCode)}</div>
                              ))}
                            </div>
                          </div>

                          {formationRows.length > 0 ? (
                            <LineupAnalysis>
                              <div className="team-analysis-list">
                                {formationRows.map(({ teamCode, formation }) => (
                                  <article className="team-analysis-card" key={`analysis-${teamCode}`}>
                                    <div className="team-analysis-head">
                                      <strong style={{ color: teamColor(teamCode) }}>{`Команда ${teamCode}`}</strong>
                                      <span className="team-analysis-scheme">{formation.scheme || '-'}</span>
                                    </div>
                                    {formation.analysis ? <p>{formation.analysis}</p> : null}
                                    {formation.indicators && formation.indicators.length > 0 ? (
                                      <div className="team-analysis-grid">
                                        <div className="team-analysis-grid-head">
                                          <span>Показатель</span>
                                          <span>Результат</span>
                                          <span>Ориентир</span>
                                          <span>Статус</span>
                                        </div>
                                        {formation.indicators.map((item) => (
                                          <div className="team-analysis-grid-row" key={`${teamCode}-${item.code}`}>
                                            <span className="team-analysis-grid-name">{item.name}</span>
                                            <span>{item.result}</span>
                                            <span>{item.reference}</span>
                                            <span className={item.passed ? 'team-analysis-status ok' : 'team-analysis-status warn'}>
                                              {item.passed ? 'В норме' : 'Ниже ориентира'}
                                            </span>
                                          </div>
                                        ))}
                                      </div>
                                    ) : null}
                                    {!formation.analysis && (!formation.indicators || formation.indicators.length === 0) ? (
                                      <p>Нет данных анализа</p>
                                    ) : null}
                                  </article>
                                ))}
                              </div>
                            </LineupAnalysis>
                          ) : null}

                          <TeamForecast players={teamSplit.players} includeC={teamCEnabled}/>
                        </>
                      )
                    })()}
                    <div className="manual-controls team-save-controls">
                      <button type="button" className="btn-secondary" onClick={() => void onAutoSplitTeam()}>
                        Автораспределить
                      </button>
                      <button type="button" onClick={() => void onSaveTeamSplit()}>
                        Сохранить
                      </button>
                      <button type="button" className="btn-secondary" onClick={() => void onPublishTeamSplit()}>
                        Опубликовать состав
                      </button>
                    </div>
                  </>
                ) : null}
              </>
                ) : null}
              </>
            ) : null}

            {historyDetailTab === 'votes' ? (
              <section className="content-card studio-event-votes">
                {renderManualPollVoteForm(selectedHistoryPostID)}
                <div className="template-head">
                  <h4>Голоса</h4>
                </div>
                {selectedHistoryPostID === null ? <p className="muted">Выбери опрос в списке выше.</p> : null}
                {historyPollVotesLoading ? <p className="muted">Загрузка голосов...</p> : null}

                {!historyPollVotesLoading && !historyPollVotesError && selectedHistoryPostID !== null && historyPollVotes.length === 0 ? (
                  <p className="muted">По этому опросу пока нет голосов.</p>
                ) : null}
                {!historyPollVotesLoading && !historyPollVotesError && historyPollVotes.length > 0 ? (
                  <div className="table-wrap">
                    <table className="table">
                      <thead>
                        <tr>
                          <th>Игрок</th>
                          <th>Выбор</th>
                          <th>Учет</th>
                          <th>Источник</th>
                          <th>Время</th>
                          {isAdmin ? <th className="col-center">Удалить</th> : null}
                        </tr>
                      </thead>
                      <tbody>
                        {historyPollVotes.map((vote) => (
                          <tr key={`${vote.userID}-${vote.choice}-${vote.votedAt}`}>
                            <td>{pollVoteDisplayName(vote)}</td>
                            <td>{vote.choiceLabel || vote.choice}</td>
                            <td>{vote.counted ? 'да' : 'нет'}</td>
                            <td>{vote.source || '-'}</td>
                            <td>{formatDateTime(vote.votedAt)}</td>
                            {isAdmin ? (
                              <td className="col-center">
                                <button
                                  type="button"
                                  className="btn-danger btn-icon"
                                  title="Удалить голос"
                                  aria-label="Удалить голос"
                                  onClick={() => {
                                    if (activeChatID === null || selectedHistoryPostID === null) return
                                    const name = pollVoteDisplayName(vote)
                                    setConfirmModal({
                                      open: true,
                                      title: 'Удалить голос игрока?',
                                      message:
                                        `Вы точно хотите удалить голос игрока?\n\n` +
                                        `Игрок: ${name}\n` +
                                        `Выбор: ${vote.choiceLabel || vote.choice}\n\n` +
                                        `Действие безвозвратно и приведет к перерасчёту стоимости тренировки.`,
                                      confirmLabel: 'Удалить',
                                      cancelLabel: 'Отмена',
                                      danger: true,
                                      onConfirm: async () => {
                                        await deleteGroupPollVotesForUser(activeChatID, selectedHistoryPostID, vote.userID, vote.choice)
                                      },
                                    })
                                  }}
                                >
                                  −
                                </button>
                              </td>
                            ) : null}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : null}
              </section>
            ) : null}

            {historyDetailTab === 'sets' ? (
              <EventSetsEditor
                rows={eventSetRowsDraft}
                setRows={setEventSetRowsDraft}
                teams={teamSplit ? activeTeamCodes(teamSplit.players, teamCEnabled) : ['A', 'B']}
                loading={eventSetsLoading}
                available={selectedHistoryPostID !== null}
                onSave={() => void onSaveEventSets()}
                onPublish={() => void onPublishEventSets()}
              />
            ) : null}

            {historyDetailTab === 'attendance' && activeChatID !== null && isAdmin && <TrainingAttendance key={`${activeChatID}-${activeHistoryEventID}`} chatID={activeChatID} instanceID={selectedHistoryEvent.instanceID} cancelled={selectedHistoryEvent.status === 'not_held'} startAt={selectedHistoryEvent.nextStartAt} onChanged={() => setAttendanceRevision(value => value + 1)}/>}
{historyDetailTab === 'billing' ? (
              <section className="content-card">
                <div className="studio-event-billing-heading">
                  <div><h4>Оплата события</h4><p>Взносы и задолженность только за эту тренировку.</p></div>
                  {(!authConfig?.enabled || activePerms?.permissions.billing_manage) && <EventDebtReminder
                    key={`${activeChatID}:${selectedHistoryEvent.instanceID}`}
                    chatID={activeChatID}
                    instanceID={selectedHistoryEvent.instanceID}
                    eventName={selectedHistoryEvent.name}
                    groupTitle={details?.group.title || 'Группа команды'}
                    billing={eventBilling}
                    pendingChanges={Boolean(eventBilling?.players.some(player => Boolean(eventBillingDraft[player.userID]) !== player.isPaid))}
                    unavailable={eventBillingLoading || Boolean(eventBillingError)}
                    onPublished={() => setSuccess('Должники по этому событию опубликованы в Telegram')}
                  />}
                </div>
                {eventBillingLoading ? <p className="muted">Загружаю оплаты...</p> : null}

                {!eventBillingLoading && !eventBillingError && !eventBilling ? (
                  <>
                    <p className="muted">Расчет по событию пока не создан.</p>
                    {selectedHistoryEvent ? (
                      <div className="manual-controls">
                        <button type="button" className="btn-secondary" onClick={() => void onGenerateBillingForInstance(selectedHistoryEvent.instanceID)}>
                          Сформировать расчет
                        </button>
                      </div>
                    ) : null}
                  </>
                ) : null}
                {eventBilling ? (
                  <>
                    <div className="detail-grid">
                      <div>
                        <span>Дата расчёта</span>
                        <strong>{formatDateTime(eventBilling.localDate)}</strong>
                      </div>
                      <div>
                        <span>Участников</span>
                        <strong>{eventBilling.participantsCount}</strong>
                      </div>
                      <div>
                        <span>На человека</span>
                        <strong>{formatMoney(eventBilling.amountPerPerson)}</strong>
                      </div>
                      <div>
                        <span>Оплачено</span>
                        <strong>{eventBilling.paidCount}</strong>
                      </div>
                      <div>
                        <span>Не оплачено</span>
                        <strong>{eventBilling.unpaidCount}</strong>
                      </div>
                      <div>
                        <span>Осталось оплатить</span>
                        <strong>{formatMoney(eventBilling.debtAmount)}</strong>
                      </div>
                    </div>
                    <div className="table-wrap table-wrap-spaced">
                      <table className="table">
                        <thead>
                          <tr>
                            <th>Игрок</th>
                            <th>Сумма</th>
                            <th className="col-center">Оплатил</th>
                          </tr>
                        </thead>
                        <tbody>
                          {eventBilling.players.map((player) => (
                            <tr key={player.userID}>
                              <td>{player.realName?.trim() || `${player.firstName ?? ''} ${player.lastName ?? ''}`.trim() || (player.username ? `@${player.username}` : `ID ${player.userID}`)}</td>
                              <td>{formatMoney(player.amountDue)}{player.passCovered && <small style={{display:"block",color:"var(--accent)"}}>Абонемент · 1 место</small>}</td>
                              <td className="col-center">
                                <label className="toggle-field toggle-field-only">
                                  <Checkbox
                                    aria-label={`Оплата: ${player.realName?.trim() || `${player.firstName ?? ''} ${player.lastName ?? ''}`.trim() || player.username || `Игрок ${player.userID}`}`}
                                    disabled={Boolean(player.passCovered && player.amountDue <= 0)} checked={Boolean(eventBillingDraft[player.userID])}
                                    onChange={(e) =>
                                      setEventBillingDraft((prev) => ({
                                        ...prev,
                                        [player.userID]: e.target.checked,
                                      }))
                                    }
                                  />
                                </label>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    <div className="manual-controls">
                      <button
                        type="button"
                        className="btn-secondary"
                        onClick={() => void onGenerateBillingForInstance(selectedHistoryEvent.instanceID)}
                      >
                        Сформировать расчет
                      </button>
                      <button type="button" onClick={() => void onSaveBilling()}>
                        Сохранить оплаты
                      </button>
                    </div>
                  </>
                ) : null}
              </section>
            ) : null}
          </section>
        ) : null}
      </section>
    )
  }

  function renderPolls() {
    if (activeChatID === null) {
      return <section className="content-card">Выбери организацию</section>
    }

    if (activePollPostID !== null) {
      return (
        <section className="content-card">
          <div className="template-head">
            <button
              className="btn-secondary"
              onClick={() => openStudioSection('events', { historyEventID: selectedGroupPoll?.instanceID ?? null })}
            >
              {selectedGroupPoll?.instanceID ? '← К событию' : '← К событиям'}
            </button>
            <h3>Голосование #{activePollPostID}</h3>
          </div>

          {selectedGroupPoll ? (
            <div className="detail-grid">
              <div>
                <span>Событие</span>
                <strong>{selectedGroupPoll.eventName || 'Без события'}</strong>
              </div>
              <div>
                <span>Шаблон</span>
                <strong>{selectedGroupPoll.templateName || '-'}</strong>
              </div>
              <div>
                <span>Статус</span>
                <strong>{selectedGroupPoll.status === 'open' ? 'Открыто' : selectedGroupPoll.status === 'closed' ? 'Завершено' : selectedGroupPoll.status || '—'}</strong>
              </div>
              <div>
                <span>Опубликовано</span>
                <strong>{formatDateTime(selectedGroupPoll.publishedAt)}</strong>
              </div>
              <div>
                <span>Учет/Всего</span>
                <strong>
                  {selectedGroupPoll.countedVotes} / {selectedGroupPoll.totalVotes}
                </strong>
              </div>
              <div>
                <span>Вопрос</span>
                <strong>{selectedGroupPoll.question || '-'}</strong>
              </div>
            </div>
          ) : (
            <p className="muted">Карточка голосования не найдена в текущем списке.</p>
          )}

          {groupPollVotesLoading ? <p className="muted">Загрузка голосов...</p> : null}

          {!groupPollVotesLoading && !groupPollVotesError && groupPollVotes.length === 0 ? <p className="muted">По этому опросу пока нет голосов</p> : null}
          {!groupPollVotesLoading && groupPollVotes.length > 0 ? (
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Игрок</th>
                    <th>Выбор</th>
                    <th>Учет</th>
                    <th>Источник</th>
                    <th>Время</th>
                  </tr>
                </thead>
                <tbody>
                  {groupPollVotes.map((vote) => (
                    <tr key={`${vote.userID}-${vote.choice}-${vote.votedAt}`}>
                      <td>{pollVoteDisplayName(vote)}</td>
                      <td>{vote.choiceLabel || vote.choice}</td>
                      <td>{vote.counted ? 'да' : 'нет'}</td>
                      <td>{vote.source || '-'}</td>
                      <td>{formatDateTime(vote.votedAt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
          {renderManualPollVoteForm(activePollPostID)}
        </section>
      )
    }

    return (
      <section className="content-card">
        <div className="template-head">
          <h3>Голосования</h3>
        </div>
        {groupPollsLoading ? <p className="muted">Загрузка голосований...</p> : null}

        {!groupPollsLoading && !groupPollsError && groupPolls.length === 0 ? <p className="muted">Голосований пока нет</p> : null}
        {!groupPollsLoading && groupPolls.length > 0 ? <StudioPolls items={groupPolls} open={id=>openStudioSection('polls',{pollPostID:id})}/> : null}
      </section>
    )
  }

  function renderGames() {
    if (activeChatID === null) {
      return <section className="content-card">Выбери организацию</section>
    }

    const toggleRow = async (row: GroupGameRow) => {
      const key = `${row.instanceID}:${row.ordinal}:${row.team1}:${row.team2}`
      setGamesExpandedRows((prev) => ({ ...prev, [key]: !prev[key] }))
      if (gamesRosterCache[key]) {
        return
      }
      try {
        const roster = await fetchGameRoster(activeChatID, row.instanceID, row.team1, row.team2)
        setGamesRosterCache((prev) => ({ ...prev, [key]: roster }))
      } catch {
        // keep collapsed/expanded state; roster will show "нет данных" placeholder
      }
    }

    return (
      <section className="content-card">
        <div className="template-head">
          <h3>Игры</h3>
        </div>

        {groupGamesLoading ? <p className="muted">Загрузка игр...</p> : null}

        {!groupGamesLoading && !groupGamesError && groupGames.length === 0 ? <p className="muted">Пока нет сохранённых партий</p> : null}

        {!groupGamesLoading && !groupGamesError && groupGames.length > 0 ? (
          <div className="table-wrap">
            <table className="table games-table">
              <thead>
                <tr>
                  <th>Команда</th>
                  <th className="col-center">Счёт</th>
                  <th>Команда</th>
                  <th>Дата</th>
                </tr>
              </thead>
              <tbody>
                {groupGames.map((row) => {
                  const key = `${row.instanceID}:${row.ordinal}:${row.team1}:${row.team2}`
                  const expanded = Boolean(gamesExpandedRows[key])
                  const roster = gamesRosterCache[key]
                  const team1Players = roster?.team1Players ?? []
                  const team2Players = roster?.team2Players ?? []
                  const hasRoster = Boolean(roster && (team1Players.length > 0 || team2Players.length > 0))

                  return (
                    <Fragment key={key}>
                      <tr className="member-row game-row" onClick={() => void toggleRow(row)}>
                        <td>
                          <div className="person-cell">
                            <strong>{`Команда ${row.team1}`}</strong>
                            <span className="muted">{row.eventName || `Событие #${row.instanceID}`}</span>
                          </div>
                        </td>
                        <td className="col-center">
                          <button type="button" className="game-score" aria-expanded={expanded} aria-label={`Составы команд: ${row.eventName}, партия ${row.ordinal}, счёт ${row.score1}:${row.score2}`} onClick={(event) => { event.stopPropagation(); void toggleRow(row) }}>{`${clampInt(row.score1, 0, 99)}:${clampInt(row.score2, 0, 99)}`}</button>
                        </td>
                        <td>
                          <strong>{`Команда ${row.team2}`}</strong>
                        </td>
                        <td>{formatDateTime(row.startAt)}</td>
                      </tr>

                      {expanded ? (
                        <tr className="games-expand-row">
                          <td colSpan={4}>
                            {!roster ? (
                              <p className="muted game-expand-empty">Не удалось загрузить составы команд</p>
                            ) : !hasRoster ? (
                              <p className="muted game-expand-empty">
                                Нет сохранённого распределения команд для этого события (или команды не заполнены).
                              </p>
                            ) : (
                              <div className="game-roster">
                                <div className="game-roster-col">
                                  <p className="muted game-roster-title">{`Команда ${row.team1}`}</p>
                                  <ul className="game-roster-list">
                                    {team1Players.length ? (
                                      team1Players.map((p) => <li key={`t1-${key}-${p.userID}`}>{playerDisplayName(p)}</li>)
                                    ) : (
                                      <li className="muted">Пусто</li>
                                    )}
                                  </ul>
                                </div>
                                <div className="game-roster-col">
                                  <p className="muted game-roster-title">{`Команда ${row.team2}`}</p>
                                  <ul className="game-roster-list">
                                    {team2Players.length ? (
                                      team2Players.map((p) => <li key={`t2-${key}-${p.userID}`}>{playerDisplayName(p)}</li>)
                                    ) : (
                                      <li className="muted">Пусто</li>
                                    )}
                                  </ul>
                                </div>
                              </div>
                            )}
                          </td>
                        </tr>
                      ) : null}
                    </Fragment>
                  )
                })}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>
    )
  }

  function renderContent() {
    if (activeSection === 'profile') return renderProfile()
    if (!groupsLoaded && !error) return <section className="content-card" role="status">Загружаем команды…</section>
    if (!loading && groups.length === 0) {
      return (
        <section className="content-card">
          <h3>Организации не найдены</h3>
          <p className="muted">Добавь бота в группу Telegram и сделай первичную настройку, после этого группа появится здесь.</p>
        </section>
      )
    }
    if (!details) {
      return loading ? null : <section className="content-card">Выбери организацию</section>
    }

    switch (activeSection) {
      case 'overview':
        if (!isAdmin) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Раздел «Обзор» доступен только администраторам.</p>
            </section>
          )
        }
        return renderOverview()
      case 'settings':
        if (!isAdmin) return availableTemplateSections.length ? <div className="studio-org-settings"><OrganizationTemplates openEvents={can('event_templates_manage') ? () => openStudioSection('event_templates') : undefined} openPolls={can('templates_manage') ? () => openStudioSection('templates') : undefined}/></div> : <section className="content-card"><h3>Недостаточно прав</h3><p className="muted">Организация недоступна.</p></section>
        return <OrganizationSettings key={activeChatID} group={details.group} members={loading || membersError ? null : members} roleTitle={activePerms?.roleTitle || 'Администратор'} canPublish={can('templates_manage')} publishRegistration={onPublishRegistration} openMembers={can('members_read') ? () => openStudioSection('members') : undefined} openDocumentation={() => openStudioSection('docs')} openEventTemplates={can('event_templates_manage') ? () => openStudioSection('event_templates') : undefined} openPollTemplates={can('templates_manage') ? () => openStudioSection('templates') : undefined}/>
      case 'billing':
        if (!isAdmin) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Раздел «Взносы» доступен только администраторам.</p>
            </section>
          )
        }
        return renderBilling()
      case 'members':
        if (!can('members_read')) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Раздел «Игроки» доступен только администраторам.</p>
            </section>
          )
        }
        return renderMembers()
      case 'templates':
        if (!can('templates_manage')) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Шаблоны доступны только администраторам.</p>
            </section>
          )
        }
        return renderTemplates()
      case 'polls':
        if (!can('polls_read')) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Раздел «Голосования» недоступен.</p>
            </section>
          )
        }
        return renderPolls()
      case 'events':
        if (!can('events_read')) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Раздел «События» недоступен.</p>
            </section>
          )
        }
        return renderHistory()
      case 'games':
        if (!can('events_read')) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Раздел «Игры» недоступен.</p>
            </section>
          )
        }
        return renderGames()
      case 'event_templates':
        if (!can('event_templates_manage')) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Шаблоны событий доступны только администраторам.</p>
            </section>
          )
        }
        return renderEvents()
      case 'docs':
        return <Documentation />
      default:
        return null
    }
  }

  function renderProfile() {
    return <>
      <ThemeSwitcher theme={productTheme} choose={chooseProductTheme}/>
      {authConfig?.enabled && (can('profile_read')
        ? renderProfileDetails()
        : <section className="content-card"><h3>Профиль игрока</h3><p className="muted">Нет доступа к личной статистике игрока.</p></section>)}
    </>
  }

  function renderProfileDetails() {
    if (activeChatID === null) {
      return <section className="content-card">Выбери организацию</section>
    }
    if (myProfileLoading) {
      return <section className="content-card">Загрузка профиля...</section>
    }
    if (myProfileError) {
      return (
        <section className="content-card">
          <h3>Профиль</h3>
          <p className="muted">Данные профиля недоступны.</p>
        </section>
      )
    }
    if (!myProfile) {
      return (
        <section className="content-card">
          <h3>Профиль</h3>
          <p className="muted">Нет данных</p>
        </section>
      )
    }

    return (
      <section className="content-card">
        <div className="studio-profile-hero"><span className="studio-profile-avatar">{memberInitials(authUser ? `${authUser.firstName} ${authUser.lastName}` : 'Team Time')}</span><div><p className="studio-eyebrow">ВАШЕ МЕСТО В КОМАНДЕ</p><h2>{authUser ? `${authUser.firstName} ${authUser.lastName}` : 'Мой профиль'}</h2><p>{authUser?.username ? `@${authUser.username}` : details?.group.title}</p><span className="badge">{myProfile.roleTitle || myProfile.roleCode}</span></div><Icon name="ball" size={150}/></div>
        <div className="metrics-grid studio-profile-metrics"><article className="metric-card"><span className="metric-label">Встреч в истории<Icon name="calendar" size={19}/></span><strong className="metric-value">{myProfile.trainings.length}</strong></article><article className="metric-card"><span className="metric-label">Осталось оплатить<Icon name="wallet" size={19}/></span><strong className="metric-value">{formatMoney(myProfile.debtAmount)}</strong></article><article className="metric-card"><span className="metric-label">Оплаченных встреч<Icon name="check" size={19}/></span><strong className="metric-value">{myProfile.trainings.filter(item=>item.isPaid).length}</strong></article></div>

        {activeChatID !== null && <TrainingPasses key={activeChatID} chatID={activeChatID} mine/>}
<h4 style={{ marginTop: 18 }}>История</h4>
        {myProfile.trainings.length ? (
          <div className="table-wrap table-wrap-spaced">
            <table className="table">
              <thead>
                <tr>
                  <th>Дата</th>
                  <th>Событие</th>
                  <th>Статус</th>
                  <th>Сумма</th>
                  <th className="col-center">Оплачено</th>
                </tr>
              </thead>
              <tbody>
                {myProfile.trainings.map((item) => (
                  <tr key={item.instanceID}>
                    <td>{formatDateTime(item.startAt)}</td>
                    <td>{item.name}</td>
                    <td>{historyStatusLabel(item.status as any)}</td>
                    <td>{formatMoney(item.amountDue)}</td>
                    <td className="col-center">{item.isPaid ? 'да' : 'нет'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="muted">Пока нет тренировок</p>
        )}

        {authConfig?.enabled && activePerms?.permissions?.roles_manage ? (
          <>
            <h4 style={{ marginTop: 22 }}>Роли</h4>
            <p className="muted" style={{ marginTop: 6 }}>
              Назначай дополнительные роли (тренер/капитан) участникам группы. Telegram-админам назначать роли не нужно.
            </p>
            <div className="form-grid form-grid-3 members-filters" style={{ marginTop: 12 }}>
              <label className="field">
                <span>Участник</span>
                <Select className="ui-select" value={roleAssignUserID} onChange={(e) => setRoleAssignUserID(e.target.value)}>
                  <option value="">Выбери участника</option>
                  {members.map((m) => (
                    <option key={m.userTelegramID} value={m.userTelegramID}>
                      {`${fullName(m)}${m.username ? ` (@${m.username})` : ''} · ${m.role}`}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="field">
                <span>Роль</span>
                <Select className="ui-select" value={roleAssignCode} onChange={(e) => setRoleAssignCode(e.target.value)}>
                  <option value="member">Участник</option>
                  <option value="captain">Капитан</option>
                  <option value="trainer">Тренер</option>
                </Select>
              </label>
              <label className="field">
                <span>&nbsp;</span>
                <button
                  type="button"
                  className="primary-btn"
                  disabled={!roleAssignUserID}
                  onClick={() =>
                    void (async () => {
                      if (!activeChatID) return
                      try {
                        setGroupRolesError('')
                        await assignGroupRole(activeChatID, { userTelegramID: Number(roleAssignUserID), roleCode: roleAssignCode })
                        setSuccess('Роль обновлена')
                      } catch (err: any) {
                        setGroupRolesError(err?.message || 'Ошибка назначения роли')
                      }
                    })()
                  }
                >
                  Сохранить роль
                </button>
              </label>
            </div>
            {groupRolesLoading ? <p className="muted">Загрузка ролей...</p> : null}

            {groupRoles.length ? (
              <div className="table-wrap table-wrap-spaced">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Код</th>
                      <th>Название</th>
                      <th>Доступы</th>
                    </tr>
                  </thead>
                  <tbody>
                    {groupRoles.map((r) => (
                      <tr key={r.code}>
                        <td>{r.code}</td>
                        <td>{r.title}</td>
                        <td>{Object.keys(r.permissions || {}).filter((k) => r.permissions[k]).join(', ') || '-'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : null}
          </>
        ) : null}
      </section>
    )
  }

  if (authLoading || (authConfig?.enabled && !authUser)) {
    return <>
      <ErrorNotifications/>
      <LoginPage checking={authLoading} configured={Boolean(authConfig?.telegramLoginBot)} widgetError={loginWidgetError} onRetry={() => setLoginWidgetAttempt(attempt => attempt + 1)}>
        <div id="telegram-login-widget"/>
      </LoginPage>
    </>
  }

  return (
    <div className={`console-shell studio-product${mobileNavOpen ? ' nav-open' : ''}${sidebarCollapsed ? ' sidebar-collapsed' : ''}`} onInvalid={event => {
      if (event.defaultPrevented) return
      event.preventDefault()
      const field = event.target as HTMLInputElement
      const label = field.labels?.[0]?.querySelector('span')?.textContent || field.getAttribute('aria-label') || field.getAttribute('placeholder')
      const message = field.validity.valueMissing ? (label ? `Заполните поле «${label}».` : 'Заполните обязательные поля.')
        : field.validity.rangeUnderflow ? `Укажите значение не меньше ${field.min}${label ? ` в поле «${label}»` : ''}.`
        : field.validity.rangeOverflow ? `Укажите значение не больше ${field.max}${label ? ` в поле «${label}»` : ''}.`
        : label ? `Проверьте значение в поле «${label}».` : 'Проверьте заполненные поля.'
      notifyError(message)
      field.focus()
    }}>
      <ErrorNotifications/>
      <div className="mobile-topbar">
        <button
          type="button"
          className="btn-icon mobile-menu-btn"
          aria-label={mobileNavOpen ? 'Закрыть меню' : 'Открыть меню'}
          aria-expanded={mobileNavOpen}
          onClick={() => setMobileNavOpen((v) => !v)}
        >
          <span aria-hidden="true">{mobileNavOpen ? '✕' : '☰'}</span>
        </button>
        <div className="mobile-topbar-text">
          <strong className="mobile-topbar-title">{details?.group.title || 'TeamTime'}</strong>
          <span className="mobile-topbar-subtitle">{sections.find((s) => s.id === activeSection)?.title || 'Навигация'}</span>
        </div>
        <button
          type="button"
          className="btn-icon mobile-refresh-btn"
          aria-label="Обновить"
          title="Обновить данные команды"
          disabled={!activeChatID || loading || refreshing}
          onClick={() => void refreshActiveOrganization()}
        >
          <Icon name="refresh" size={20}/>
        </button>
        <HeaderProgress active={loading || refreshing}/>
      </div>
      <div className="sidebar-overlay" role="presentation" onClick={() => setMobileNavOpen(false)} />
      <aside className="console-sidebar">
        <div className="brand-block">
          <button className="brand-mark" aria-label="На главную TeamTime" onClick={()=>openStudioSection('overview')}><img src="/icon.png" alt="" width="64" height="64"/></button>
          <div>
            <p className="brand-title">teamtime.</p>
            <p className="brand-subtitle">Время быть командой</p>
          </div>
          <button
            type="button"
            className="btn-icon sidebar-close-btn"
            aria-label="Закрыть меню"
            onClick={() => setMobileNavOpen(false)}
          >
            <span aria-hidden="true">✕</span>
          </button>
        </div>



        <nav className="nav-menu" id="team-sidebar-nav" aria-label="Разделы команды">
          <p className="section-caption">Справочники</p>
          {sidebarSections.map((section) => (
            <button
              key={section.id}
              className={activeSidebarSection === section.id ? 'nav-item active' : 'nav-item'}
              aria-current={activeSidebarSection === section.id ? 'page' : undefined}
              aria-label={section.title}
              title={section.title}
              onClick={() => {
                setMobileNavOpen(false)
                navigateTo({
                  chatID: activeChatID,
                  section: section.id,
                  templateView: 'list',
                  templateName: null,
                  eventView: 'list',
                  eventID: null,
                })
              }}
            >
              <span className="nav-icon"><Icon name={({overview:'grid',events:'calendar',members:'users',games:'score',polls:'poll',billing:'wallet',templates:'template',event_templates:'repeat',profile:'profile',settings:'organization',docs:'template'} as Record<Section,string>)[section.id]} size={22}/></span>
              <span>
                <strong>{section.title}</strong>
                <small>{section.subtitle}</small>
              </span>
            </button>
          ))}
        </nav>

        <div className="studio-sidebar-footer">
        {authConfig?.enabled && authUser ? (
          <button
            className="studio-sidebar-exit"
            aria-label="Выйти"
            title="Выйти"
            onClick={() =>
              void (async () => {
                setMobileNavOpen(false)
                await logoutAuth()
                setAuthUser(null)
                setDetails(null)
                setGroups([])
              })()
            }
          >
            <Icon name="logout" size={20}/><span>Выйти</span>
          </button>
        ) : null}
        <button type="button" className="studio-sidebar-toggle"
          aria-label={sidebarCollapsed ? 'Развернуть меню' : 'Свернуть меню'}
          title={sidebarCollapsed ? 'Развернуть меню' : 'Свернуть меню'}
          aria-expanded={!sidebarCollapsed} aria-controls="team-sidebar-nav"
          onClick={() => {
            const next = !sidebarCollapsed
            setSidebarCollapsed(next)
            try { localStorage.setItem('teamtime-sidebar-collapsed', String(next)) } catch { /* The toggle still works without storage. */ }
          }}>
          <Icon name="chevron" size={20}/><span>Свернуть</span>
        </button>
        </div>
      </aside>

      <main className="console-main">
        <header className="console-header"><a className="studio-wordmark" href="/">teamtime.</a>
        <section className="studio-org-selector">
          <span className="studio-org-label">Ваша команда</span>
          <Select
            className="ui-select"
            aria-label="Организация"
            value={activeChatID ?? ''}
            onChange={(e) => {
              const value = e.target.value
              const nextChatID = value ? Number(value) : null
              setMobileNavOpen(false)
              navigateTo({
                chatID: nextChatID,
                section: activeSection,
                templateView: 'list',
                templateName: null,
                eventView: 'list',
                eventID: null,
              })
              setSuccess('')
              setError('')
            }}
          >
            <option value="">Выбери организацию</option>
            {groups.map((group) => (
              <option key={group.chatID} value={group.chatID}>
                {group.title}
              </option>
            ))}
          </Select>
        </section>
          <div className="studio-header-actions">
            <button type="button" className="btn-icon btn-secondary studio-header-refresh" aria-label="Обновить данные команды" title="Обновить данные команды" disabled={!activeChatID || loading || refreshing} onClick={()=>void refreshActiveOrganization()}><Icon name="refresh" size={20}/></button>
            <nav className="studio-header-nav" aria-label="Личный кабинет и помощь">
              {headerSections.map(section => (
                <button
                  key={section.id}
                  type="button"
                  className={activeHeaderSection === section.id ? 'studio-header-link active' : 'studio-header-link'}
                  aria-current={activeHeaderSection === section.id ? 'page' : undefined}
                  aria-label={section.id === 'profile' ? 'Профиль' : section.title}
                  title={section.id === 'profile' ? 'Профиль' : section.title}
                  onClick={() => openStudioSection(section.id)}
                >
                  {section.id === 'profile' ? <Icon name="profile" size={22}/> : section.icon}
                </button>
              ))}
            </nav>
          </div>
          <HeaderProgress active={loading || refreshing}/>
        </header>
        <div className="studio-scroll-area">
        <div className="studio-workspace" data-section={activeSection}>
          <div className="studio-page-content">
          <h1 className="studio-visually-hidden">{activeSection === 'overview' ? 'Обзор команды' : sections.find(section => section.id === activeSection)?.title}</h1>

        {success ? <section className={`toast toast-success ${successVisible ? 'show' : 'hide'}`}>{success}</section> : null}

        {isTemplateSection(activeSection) ? (
          <section className="content-card studio-template-workspace">
            <OrganizationTemplateNavigation active={activeSection} available={availableTemplateSections} open={openStudioSection}/>
            <div className="studio-template-content">{renderContent()}</div>
          </section>
        ) : renderContent()}
          </div>
        <footer className="studio-page-footer"><span>TeamTime · Время быть командой.</span></footer>
        </div>
        </div>
      </main>

      {confirmModal.open ? (
        <div
          className="modal-overlay"
          role="presentation"
          onClick={() => {
            if (confirmBusy) return
            setConfirmModal((prev) => ({ ...prev, open: false, onConfirm: null }))
          }}
        >
          <div
            className="modal-card"
            role="dialog"
            aria-modal="true"
            aria-label={confirmModal.title}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-head">
              <h4>{confirmModal.title}</h4>
              <button
                type="button"
                className="btn-icon btn-secondary modal-close"
                aria-label="Закрыть"
                disabled={confirmBusy}
                onClick={() => setConfirmModal((prev) => ({ ...prev, open: false, onConfirm: null }))}
              >
                <span aria-hidden="true">✕</span>
              </button>
            </div>
            <p className="muted modal-body" style={{ whiteSpace: 'pre-line' }}>
              {confirmModal.message}
            </p>
            <div className="modal-actions">
              <button
                type="button"
                className="btn-secondary"
                disabled={confirmBusy}
                onClick={() => setConfirmModal((prev) => ({ ...prev, open: false, onConfirm: null }))}
              >
                {confirmModal.cancelLabel}
              </button>
              <button
                type="button"
                className={confirmModal.danger ? 'btn-danger' : ''}
                disabled={confirmBusy}
                onClick={() =>
                  void (async () => {
                    if (!confirmModal.onConfirm) {
                      setConfirmModal((prev) => ({ ...prev, open: false }))
                      return
                    }
                    setConfirmBusy(true)
                    try {
                      // Close immediately to avoid double-clicks; errors will show via runAction.
                      setConfirmModal((prev) => ({ ...prev, open: false }))
                      await runAction(confirmModal.onConfirm, 'Голос удалён, расчёт пересчитан')
                    } finally {
                      setConfirmBusy(false)
                      setConfirmModal((prev) => ({ ...prev, onConfirm: null }))
                    }
                  })()
                }
              >
                {confirmModal.confirmLabel}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}

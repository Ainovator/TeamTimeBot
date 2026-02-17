import { FormEvent, Fragment, useEffect, useMemo, useRef, useState } from 'react'
import {
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
  publishEventSettlement,
  publishEventTeamSplit,
  publishGroupDebtors,
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
  teamPairProbabilities,
  type ActiveTeamCode,
} from './features/history/historyUtils'
import {
  averageScore,
  fullName,
  fullNamePartsFromProfile,
  initialsFromProfile,
  normalizedSkillScore,
  playerTypeCards,
  playerTypeLabel,
  profileSkills,
  scoreColor,
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

const docsNavItems: Array<{ id: string; title: string }> = [
  { id: 'docs-quickstart', title: 'Быстрый старт' },
  { id: 'docs-templates-polls', title: 'Шаблоны голосований' },
  { id: 'docs-templates-events', title: 'Шаблоны событий' },
  { id: 'docs-workflow', title: 'Жизненный цикл тренировки' },
  { id: 'docs-votes', title: 'Голоса и учёт' },
  { id: 'docs-teams', title: 'Команды и распределение' },
  { id: 'docs-sets', title: 'Партии (сеты)' },
  { id: 'docs-billing', title: 'Оплата и перерасчёт' },
  { id: 'docs-debts', title: 'Задолженности' },
  { id: 'docs-roles', title: 'Роли и доступы' },
]

const MEMBERS_SORT_AVERAGE_CODE = '__avg__'
type MembersSortCriterion = { code: string; direction: 'asc' | 'desc' }

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

type SetRowDraft = {
  left: ActiveTeamCode
  right: ActiveTeamCode
  leftScore: number
  rightScore: number
}

function clampInt(value: unknown, min: number, max: number): number {
  const n = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(n)) return min
  return Math.max(min, Math.min(max, Math.trunc(n)))
}

export default function App() {
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null)
  const [authUser, setAuthUser] = useState<AuthUser | null>(null)
  const [authLoading, setAuthLoading] = useState(true)
  const [groups, setGroups] = useState<Group[]>([])
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

  const [details, setDetails] = useState<GroupDetails | null>(null)
  const [members, setMembers] = useState<GroupMember[]>([])
  const [membersError, setMembersError] = useState('')
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
  const [memberRelationPickerQuery, setMemberRelationPickerQuery] = useState('')
  const [memberRelationPickerOpen, setMemberRelationPickerOpen] = useState(false)
  const [membersSearch, setMembersSearch] = useState('')
  const [membersTypeFilter, setMembersTypeFilter] = useState<'' | 'attacker' | 'setter' | 'libero' | 'central'>('')
  const [membersSortCriteria, setMembersSortCriteria] = useState<MembersSortCriterion[]>([])
  const [memberSkillLoading, setMemberSkillLoading] = useState(false)
  const [memberSkillError, setMemberSkillError] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
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
    announcementEnabled: false,
    announcementLeadMinutes: '60',
    teamsAutoSplit: false,
    teamsPublishList: false,
    teamSize: '6',
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
  const [templateEditorError, setTemplateEditorError] = useState('')
  const [eventEditor, setEventEditor] = useState({
    name: '',
    eventType: 'training' as 'training' | 'activity',
    weekday: '5',
    publishWeekday: '5',
    publishAt: '10:00',
    startAt: '19:00',
    endAt: '21:00',
    announcementText: '',
    announcementEnabled: false,
    announcementLeadMinutes: '60',
    teamsAutoSplit: false,
    teamsPublishList: false,
    teamSize: '6',
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
  const [eventActivityError, setEventActivityError] = useState('')
  const [showEventActivity, setShowEventActivity] = useState(false)
  const [eventHistory, setEventHistory] = useState<EventHistoryItem[]>([])
  const [historyStatusFilter, setHistoryStatusFilter] = useState<'' | 'in_voting' | 'on_distribution' | 'on_review' | 'completed' | 'not_held'>('')
  const [historyDetailTab, setHistoryDetailTab] = useState<'distribution' | 'votes' | 'billing' | 'sets'>('distribution')
  const [eventHistoryLoading, setEventHistoryLoading] = useState(false)
  const [eventHistoryError, setEventHistoryError] = useState('')
  const [eventPollHistory, setEventPollHistory] = useState<EventPollHistoryItem[]>([])
  const [groupPolls, setGroupPolls] = useState<GroupPollItem[]>([])
  const [groupPollsLoading, setGroupPollsLoading] = useState(false)
  const [groupPollsError, setGroupPollsError] = useState('')
  const [groupPollVotes, setGroupPollVotes] = useState<GroupPollVoteItem[]>([])
  const [groupPollVotesLoading, setGroupPollVotesLoading] = useState(false)
  const [groupPollVotesError, setGroupPollVotesError] = useState('')
  const [eventPollHistoryLoading, setEventPollHistoryLoading] = useState(false)
  const [eventPollHistoryError, setEventPollHistoryError] = useState('')
  const [selectedHistoryPostID, setSelectedHistoryPostID] = useState<number | null>(null)
  const [historyPollVotes, setHistoryPollVotes] = useState<GroupPollVoteItem[]>([])
  const [historyPollVotesLoading, setHistoryPollVotesLoading] = useState(false)
  const [historyPollVotesError, setHistoryPollVotesError] = useState('')
  const [teamSplit, setTeamSplit] = useState<EventTeamSplitState | null>(null)
  const [teamSplitLoading, setTeamSplitLoading] = useState(false)
  const [teamSplitError, setTeamSplitError] = useState('')
  const [draggedPlayerID, setDraggedPlayerID] = useState<number | null>(null)
  const [touchDragMode, setTouchDragMode] = useState(false)
  const [teamCEnabled, setTeamCEnabled] = useState(false)
  const [eventBilling, setEventBilling] = useState<EventBilling | null>(null)
  const [eventBillingLoading, setEventBillingLoading] = useState(false)
  const [eventBillingError, setEventBillingError] = useState('')
  const [eventBillingDraft, setEventBillingDraft] = useState<Record<number, boolean>>({})
  const [eventSetRowsDraft, setEventSetRowsDraft] = useState<SetRowDraft[]>([])
  const [eventSetsLoading, setEventSetsLoading] = useState(false)
  const [eventSetsError, setEventSetsError] = useState('')
  const [groupDebtSummary, setGroupDebtSummary] = useState<GroupDebtSummary | null>(null)
  const [billingDebtors, setBillingDebtors] = useState<GroupDebtor[]>([])
  const [billingLoading, setBillingLoading] = useState(false)
  const [billingError, setBillingError] = useState('')
  const [billingExpanded, setBillingExpanded] = useState<Record<number, boolean>>({})
  const [billingPublishDraft, setBillingPublishDraft] = useState<Record<number, boolean>>({})
  const [groupGames, setGroupGames] = useState<GroupGameRow[]>([])
  const [groupGamesLoading, setGroupGamesLoading] = useState(false)
  const [groupGamesError, setGroupGamesError] = useState('')
  const [gamesExpandedRows, setGamesExpandedRows] = useState<Record<string, boolean>>({})
  const [gamesRosterCache, setGamesRosterCache] = useState<Record<string, GameRosterResponse>>({})
  const [myProfile, setMyProfile] = useState<UserGroupProfile | null>(null)
  const [myProfileLoading, setMyProfileLoading] = useState(false)
  const [myProfileError, setMyProfileError] = useState('')
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
  const memberRelationPickerRef = useRef<HTMLDivElement | null>(null)
  const [confirmBusy, setConfirmBusy] = useState(false)
  const [activePerms, setActivePerms] = useState<GroupPermissionsView | null>(null)
  const [groupRoles, setGroupRoles] = useState<GroupRoleView[]>([])
  const [groupRolesLoading, setGroupRolesLoading] = useState(false)
  const [groupRolesError, setGroupRolesError] = useState('')
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
      if (s.id === 'billing') return isAdmin
      if (s.id === 'docs') return true
      if (s.id === 'events') return can('events_read')
      if (s.id === 'games') return can('events_read')
      if (s.id === 'polls') return can('polls_read')
      if (s.id === 'profile') return can('profile_read')
      if (s.id === 'members') return can('members_read')
      if (s.id === 'templates') return can('templates_manage')
      if (s.id === 'event_templates') return can('event_templates_manage')
      return false
    })
  }, [authConfig?.enabled, activePerms, isAdmin])
  const templateCountedMap = useMemo(() => {
    const map = new Map<string, number>()
    for (const template of templates) {
      map.set(template.name, template.countedOptionsCount ?? 0)
    }
    return map
  }, [templates])

  const [docsSideNavVisible, setDocsSideNavVisible] = useState(false)
  const [docsActiveAnchor, setDocsActiveAnchor] = useState<string>(docsNavItems[0]?.id ?? 'docs-quickstart')
  const [docsSideNavCollapsed, setDocsSideNavCollapsed] = useState<boolean>(() => {
    try {
      return window.localStorage.getItem('docsSideNavCollapsed') === '1'
    } catch {
      return false
    }
  })
  const selectedEvent = useMemo(
    () => events.find((event) => event.id === activeEventID) ?? null,
    [events, activeEventID],
  )
  const selectedGroupPoll = useMemo(
    () => groupPolls.find((poll) => poll.postID === activePollPostID) ?? null,
    [groupPolls, activePollPostID],
  )
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
  const filteredMembers = useMemo(() => {
    const query = membersSearch.trim().toLowerCase()
    const list = members.filter((member) => {
      const profile = memberSkillProfiles[member.userTelegramID]
      const playerType = (profile?.playerType || member.playerType || '') as '' | 'attacker' | 'setter' | 'libero' | 'central'
      if (membersTypeFilter !== '' && playerType !== membersTypeFilter) {
        return false
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
        const avg = averageScore(profile, skillsCatalog)
        value = avg === null ? -1 : avg
      } else if (knownSkillCodes.has(code)) {
        const skill = profileSkills(profile, skillsCatalog).find((item) => item.skillCode === code)
        value = normalizedSkillScore(skill?.score)
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
    skillsCatalog,
  ])
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

  const membersSortOrderByCode = useMemo(() => {
    const map = new Map<string, number>()
    membersSortCriteria.forEach((item, idx) => map.set(item.code, idx + 1))
    return map
  }, [membersSortCriteria])

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
        label: `${relationMemberOptionLabel(member)} · ID ${member.userTelegramID}`,
      }
    })
  }, [availableRelationMembers, memberSkillProfiles])

  const selectedRelationMemberOption = useMemo(() => {
    const selectedID = Number(memberRelationDraft.otherUserID)
    if (!Number.isInteger(selectedID) || selectedID <= 0) {
      return null
    }
    return relationMemberOptions.find((item) => item.id === selectedID) ?? null
  }, [memberRelationDraft.otherUserID, relationMemberOptions])

  const filteredRelationMemberOptions = useMemo(() => {
    const query = memberRelationPickerQuery.trim().toLowerCase()
    if (query === '') {
      return relationMemberOptions
    }
    return relationMemberOptions.filter((item) => item.label.toLowerCase().includes(query) || String(item.id).includes(query))
  }, [memberRelationPickerQuery, relationMemberOptions])

  useEffect(() => {
    if (!memberRelationPickerOpen) {
      return
    }
    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node | null
      if (target && memberRelationPickerRef.current && !memberRelationPickerRef.current.contains(target)) {
        setMemberRelationPickerOpen(false)
      }
    }
    document.addEventListener('mousedown', handlePointerDown)
    return () => document.removeEventListener('mousedown', handlePointerDown)
  }, [memberRelationPickerOpen])
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
      eventEditor.announcementText.trim() !== (selectedEvent.announcementText || '') ||
      eventEditor.announcementEnabled !== selectedEvent.announcementEnabled ||
      Number(eventEditor.announcementLeadMinutes) !== selectedAnnouncementLead ||
      eventEditor.teamsAutoSplit !== selectedEvent.teamsAutoSplit ||
      eventEditor.teamsPublishList !== selectedEvent.teamsPublishList ||
      Number(eventEditor.teamSize) !== (selectedEvent.teamSize || 6) ||
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
      setGroups([])
      return
    }
    void (async () => {
      try {
        const loadedGroups = await fetchGroups()
        setGroups(loadedGroups)

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
    container.appendChild(script)

    return () => {
      delete window.onTelegramAuth
      container.innerHTML = ''
    }
  }, [authConfig, authUser, authLoading])

  useEffect(() => {
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
      setBillingExpanded({})
      setBillingPublishDraft({})
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
      setBillingExpanded({})
      setBillingPublishDraft({})
      setActiveHistoryEventID(null)
      setActivePollPostID(null)
      setArchivedEvents([])
      setMyProfile(null)
      return
    }
    void reloadActiveOrganization(activeChatID)
  }, [activeChatID, groups])

  useEffect(() => {
    if (activeSection !== 'profile' || activeChatID === null) {
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
  }, [activeSection, activeChatID, success])

  useEffect(() => {
    if (activeSection !== 'event_templates') {
      setEventsMode('active')
    }
  }, [activeSection])

  useEffect(() => {
    if (activeSection !== 'events' || activeChatID === null) {
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
        announcementEnabled: false,
        announcementLeadMinutes: '60',
        teamsAutoSplit: false,
        teamsPublishList: false,
        teamSize: '6',
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
      announcementEnabled: selectedEvent.announcementEnabled,
      announcementLeadMinutes: String(selectedEvent.announcementLeadMinutes || 60),
      teamsAutoSplit: selectedEvent.teamsAutoSplit,
      teamsPublishList: selectedEvent.teamsPublishList,
      teamSize: String(selectedEvent.teamSize || 6),
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
  }, [activeSection, activeChatID, activeHistoryEventID, success])

  useEffect(() => {
    if (activeSection !== 'docs') {
      setDocsSideNavVisible(false)
      return
    }
    let cancelled = false
    const toc = document.querySelector('.docs-toc')
    if (!toc) {
      return
    }

    const io = new IntersectionObserver(
      (entries) => {
        if (cancelled) return
        const e = entries[0]
        // Show the side nav when the main TOC is not visible.
        setDocsSideNavVisible(!e.isIntersecting)
      },
      { threshold: 0.05 },
    )
    io.observe(toc)

    const sectionEls = docsNavItems
      .map((it) => document.getElementById(it.id))
      .filter((el): el is HTMLElement => Boolean(el))

    const sectionIO = new IntersectionObserver(
      (entries) => {
        if (cancelled) return
        const visible = entries
          .filter((x) => x.isIntersecting)
          .sort((a, b) => (b.intersectionRatio ?? 0) - (a.intersectionRatio ?? 0))
        if (visible.length > 0) {
          const id = (visible[0].target as HTMLElement).id
          if (id) setDocsActiveAnchor(id)
        }
      },
      // Consider section active when its heading reaches upper half.
      { threshold: [0.15, 0.35, 0.55] },
    )
    for (const el of sectionEls) {
      sectionIO.observe(el)
    }

    return () => {
      cancelled = true
      io.disconnect()
      sectionIO.disconnect()
    }
  }, [activeSection])

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
  }, [activeSection, activeChatID, activeHistoryEventID, success])

  useEffect(() => {
    if (activeSection !== 'billing' || activeChatID === null || !isAdmin) {
      setBillingDebtors([])
      setBillingLoading(false)
      setBillingError('')
      setBillingPublishDraft({})
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
        setBillingPublishDraft((prev) => {
          const next: Record<number, boolean> = { ...prev }
          const seen = new Set<number>()
          for (const d of safe) {
            const uid = Number(d.userID)
            if (!Number.isFinite(uid) || uid === 0) continue
            seen.add(uid)
            if (typeof next[uid] !== 'boolean') next[uid] = true
          }
          for (const k of Object.keys(next)) {
            const uid = Number(k)
            if (!seen.has(uid)) delete next[uid]
          }
          return next
        })
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
  }, [activeSection, activeChatID, activeHistoryEventID, success])

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
    if (Number.isNaN(minVotesToHold) || minVotesToHold < 0) {
      setError('Минимальное количество голосов должно быть >= 0')
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
	      setError('Время начала должно быть в формате HH:MM')
	      return
	    }
	    if (endMin === null) {
	      setError('Время окончания должно быть в формате HH:MM')
	      return
	    }
	    if (endMin <= startMin) {
	      setError('Время окончания должно быть позже времени начала')
	      return
	    }
	    if (publishMin === null) {
	      setError('Время публикации должно быть в формате HH:MM')
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
      announcementEnabled: false,
      announcementLeadMinutes: '60',
      teamsAutoSplit: false,
      teamsPublishList: false,
      teamSize: '6',
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
    if (Number.isNaN(minVotesToHold) || minVotesToHold < 0) {
      setError('Минимальное количество голосов должно быть >= 0')
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
	      setError('Время начала должно быть в формате HH:MM')
	      return
	    }
	    if (endMin === null) {
	      setError('Время окончания должно быть в формате HH:MM')
	      return
	    }
	    if (endMin <= startMin) {
	      setError('Время окончания должно быть позже времени начала')
	      return
	    }
	    if (publishMin === null) {
	      setError('Время публикации должно быть в формате HH:MM')
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
      setError('Стоимость должна быть числом >= 0')
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
      announcementText !== (selectedEvent.announcementText || '') ||
      announcementEnabled !== selectedEvent.announcementEnabled ||
      announcementLeadMinutes !== selectedAnnouncementLead ||
      teamsAutoSplit !== selectedEvent.teamsAutoSplit ||
      teamsPublishList !== selectedEvent.teamsPublishList ||
      teamSize !== (selectedEvent.teamSize || 6) ||
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

  async function onPublishSettlementForEvent(eventID: number) {
    if (activeChatID === null || eventID === 0) {
      return
    }
    await runAction(() => publishEventSettlement(activeChatID, eventID), 'Расчёт опубликован')
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
      const statuses = eventBilling.players.map((player) => ({
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
      setMemberRelationPickerQuery('')
      setMemberRelationPickerOpen(false)
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
        setMemberSkillError(`Оценка для "${skill.skillName}" должна быть целым числом 1..10`)
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

  function onRelationMemberPickerChange(value: string) {
    setMemberRelationPickerQuery(value)
  }

  function onRelationMemberSelect(optionID: number) {
    const option = relationMemberOptions.find((item) => item.id === optionID)
    setMemberRelationDraft((prev) => ({ ...prev, otherUserID: String(optionID) }))
    setMemberRelationPickerQuery(option?.label || '')
    setMemberRelationPickerOpen(false)
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
      setMemberSkillError('Вес связи должен быть от 1 до 10')
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
      setMemberRelationPickerQuery('')
      setMemberRelationPickerOpen(false)
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

  function renderOverview() {
    if (!details) {
      return <section className="content-card">Выбери организацию слева</section>
    }

    return (
      <>
        <section className="metrics-grid">
          <article className="metric-card">
            <p className="metric-label">Подписчики</p>
            <p className="metric-value">{members.length}</p>
          </article>
          <article className="metric-card">
            <p className="metric-label">Шаблоны</p>
            <p className="metric-value">{templates.length}</p>
          </article>
          <article className="metric-card">
            <p className="metric-label">События</p>
            <p className="metric-value">{events.length}</p>
          </article>
          <article className="metric-card">
            <p className="metric-label">Общий долг группы</p>
            <p className="metric-value">{formatMoney(groupDebtSummary?.totalDebt ?? 0)}</p>
          </article>
        </section>

        <section className="content-card">
          <h3>Организация</h3>
          <div className="detail-grid">
            <div>
              <span>Название</span>
              <strong>{details.group.title}</strong>
            </div>
            <div>
              <span>chat_id</span>
              <strong>{details.group.chatID}</strong>
            </div>
            <div>
              <span>Таймзона</span>
              <strong>{details.group.timezone}</strong>
            </div>
          </div>
          <div className="manual-controls">
            <button type="button" className="btn-secondary" onClick={() => void onPublishRegistration()}>
              Опубликовать регистрацию
            </button>
          </div>
        </section>
      </>
    )
  }

  function debtorDisplayName(d: GroupDebtor): string {
    const full = `${d.firstName ?? ''} ${d.lastName ?? ''}`.trim()
    if (full) {
      return full
    }
    if (d.username) {
      return `@${d.username}`
    }
    return `ID ${d.userID}`
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

  function scrollDocsTo(id: string) {
    const el = document.getElementById(id)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }

  function toggleDocsSideNav(next: boolean) {
    setDocsSideNavCollapsed(next)
    try {
      window.localStorage.setItem('docsSideNavCollapsed', next ? '1' : '0')
    } catch {
      // ignore
    }
  }

  function renderDocs() {
    if (activeChatID === null) {
      return <section className="content-card">Выбери организацию</section>
    }

    return (
      <section className="docs-page">
        <section className="docs-hero">
          <div className="docs-hero-text">
            <p className="docs-kicker">TeamTime Console</p>
            <h2>Документация</h2>
            <p className="muted">
              Полный workflow продукта: от шаблонов и публикаций до распределения команд, партий, расчёта оплаты и контроля задолженностей.
            </p>
            <div className="docs-hero-actions">
              <button type="button" className="btn-secondary" onClick={() => scrollDocsTo('docs-quickstart')}>
                Быстрый старт
              </button>
              <button type="button" onClick={() => scrollDocsTo('docs-workflow')}>
                Посмотреть workflow
              </button>
            </div>
          </div>
          <div className="docs-hero-figure" aria-hidden="true">
            <img src="/docs/hero.svg" alt="" />
          </div>
        </section>

        <section className="docs-toc">
          <div className="docs-toc-head">
            <h3>Оглавление</h3>
            <p className="muted">Кликни по пункту, чтобы перейти к разделу.</p>
          </div>
          <div className="docs-toc-grid">
            {docsNavItems.map(({ id, title }) => (
              <button key={id} type="button" className="docs-toc-item" onClick={() => scrollDocsTo(id)}>
                <span>{title}</span>
                <span className="docs-toc-arrow" aria-hidden="true">
                  →
                </span>
              </button>
            ))}
          </div>
        </section>

        {docsSideNavCollapsed ? (
          <button
            type="button"
            className="docs-sidenav-toggle docs-sidenav-fab"
            aria-label="Показать боковую навигацию документации"
            onClick={() => toggleDocsSideNav(false)}
          >
            <span className="hamburger" aria-hidden="true">
              <span />
              <span />
              <span />
            </span>
          </button>
        ) : null}

        <aside
          className={docsSideNavVisible && !docsSideNavCollapsed ? 'docs-sidenav show' : 'docs-sidenav'}
          aria-label="Навигация по документации"
        >
          <div className="docs-sidenav-head">
            <div>
              <strong>Документация</strong>
            </div>
            <button
              type="button"
              className="docs-sidenav-toggle"
              aria-label="Скрыть боковую навигацию документации"
              onClick={() => toggleDocsSideNav(true)}
            >
              <span className="hamburger" aria-hidden="true">
                <span />
                <span />
                <span />
              </span>
            </button>
          </div>
          <div className="docs-sidenav-list">
            {docsNavItems.map((it) => (
              <button
                key={`side-${it.id}`}
                type="button"
                className={docsActiveAnchor === it.id ? 'docs-sidenav-item active' : 'docs-sidenav-item'}
                onClick={() => scrollDocsTo(it.id)}
              >
                {it.title}
              </button>
            ))}
          </div>
        </aside>

        <section id="docs-quickstart" className="docs-section">
          <div className="docs-section-head">
            <h3>Быстрый старт</h3>
            <p className="muted">Минимальный путь, чтобы провести первую тренировку с голосованием и оплатой.</p>
          </div>
          <div className="docs-grid">
            <article className="docs-card">
              <h4>1) Создай шаблон голосования</h4>
              <p className="muted">
                В разделе <strong>Шаблоны голосований</strong> задай вопрос и варианты ответов.
              </p>
              <ul className="docs-list">
                <li>Отметь варианты, которые участвуют в учёте посещения.</li>
                <li>Если нужно, настрой веса вариантов (например, +2 места).</li>
              </ul>
            </article>
            <article className="docs-card">
              <h4>2) Создай шаблон события</h4>
              <p className="muted">
                В разделе <strong>Шаблоны событий</strong> настрой расписание, публикацию и стоимость.
              </p>
              <ul className="docs-list">
                <li>Привяжи созданный шаблон голосования.</li>
                <li>Укажи стоимость тренировки и настройки расчёта.</li>
              </ul>
            </article>
            <article className="docs-card">
              <h4>3) Проведи тренировку</h4>
              <p className="muted">
                В <strong>События</strong> открой нужную тренировку, проверь голоса, распределение и оплату.
              </p>
              <ul className="docs-list">
                <li>При необходимости удали лишний голос (например, игрок не пришёл).</li>
                <li>Нажми «Сформировать расчёт» и сохрани оплаты.</li>
              </ul>
            </article>
          </div>
        </section>

        <section id="docs-templates-polls" className="docs-section">
          <div className="docs-section-head">
            <h3>Шаблоны голосований</h3>
            <p className="muted">Здесь задаётся вопрос и набор вариантов, которые дальше используются в тренировках.</p>
          </div>
          <div className="docs-media">
            <div className="docs-ui-preview docs-ui-preview-compact" aria-label="Пример шаблона голосования">
              <div className="template-head">
                <h4>Шаблон голосования</h4>
                <button type="button" className="btn-secondary" disabled>
                  Создать шаблон
                </button>
              </div>
              <form className="form-grid">
                <h4>Название</h4>
                <input value="Регистрация на тренировку" disabled />
                <h4>Вопрос</h4>
                <input value="Сколько человек придёт?" disabled />
                <h4>Варианты</h4>
                <div className="option-list">
                  <div className="option-row">
                    <input value="+1 (приду)" disabled />
                    <label className="option-weight">
                      <span>K</span>
                      <input type="number" value={1} disabled />
                    </label>
                    <label className="option-accounting">
                      <input type="checkbox" checked readOnly />
                      <span>Учёт</span>
                    </label>
                    <button type="button" className="icon-btn danger" disabled>
                      -
                    </button>
                  </div>
                  <div className="option-row">
                    <input value="+2 (я и друг)" disabled />
                    <label className="option-weight">
                      <span>K</span>
                      <input type="number" value={2} disabled />
                    </label>
                    <label className="option-accounting">
                      <input type="checkbox" checked readOnly />
                      <span>Учёт</span>
                    </label>
                    <button type="button" className="icon-btn danger" disabled>
                      -
                    </button>
                  </div>
                  <div className="option-row">
                    <input value="Не смогу" disabled />
                    <label className="option-weight">
                      <span>K</span>
                      <input type="number" value={1} disabled />
                    </label>
                    <label className="option-accounting">
                      <input type="checkbox" readOnly />
                      <span>Учёт</span>
                    </label>
                    <button type="button" className="icon-btn danger" disabled>
                      -
                    </button>
                  </div>
                </div>
                <button type="button" className="icon-btn add" disabled>
                  +
                </button>
                <div className="split-forms">
                  <button type="button" disabled>
                    Сохранить шаблон
                  </button>
                  <button type="button" className="btn-danger" disabled>
                    Удалить шаблон
                  </button>
                </div>
              </form>
            </div>
          </div>
          <div className="docs-callout">
            <strong>Важно:</strong> в расчёт оплаты попадают только те варианты, которые помечены как «учёт».
          </div>
        </section>

        <section id="docs-templates-events" className="docs-section">
          <div className="docs-section-head">
            <h3>Шаблоны событий</h3>
            <p className="muted">Расписание, публикации, стоимость и настройки расчёта для тренировки. Ниже описано, за что отвечает каждая настройка.</p>
          </div>
          <div className="docs-grid docs-grid-2">
            <article className="docs-card">
              <h4>Расписание</h4>
              <p className="muted">Когда старт и конец, а также когда публикуется голосование.</p>
            </article>
            <article className="docs-card">
              <h4>Стоимость</h4>
              <p className="muted">Используется для «на человека» и перерасчёта при изменении состава.</p>
            </article>
          </div>

          <div className="docs-setting-table">
            <div className="docs-setting-row docs-setting-head">
              <div>Настройка</div>
              <div>Что делает</div>
              <div>На что влияет</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Название</strong>
                <small className="muted">имя события</small>
              </div>
              <div>Название тренировки/мероприятия, отображается в консоли и в публикациях.</div>
              <div>Списки событий, сообщения в Telegram, отчёты.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Тип события</strong>
                <small className="muted">тренировка / мероприятие</small>
              </div>
              <div>Категория события для фильтров и аналитики.</div>
              <div>Фильтры в разделе «События», отображение в истории.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>День недели</strong>
                <small className="muted">когда проходит</small>
              </div>
              <div>Определяет день проведения события по локальной таймзоне группы.</div>
              <div>Планирование экземпляров, расчёт «следующей даты».</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Начало</strong>
                <small className="muted">время старта</small>
              </div>
              <div>Время начала тренировки.</div>
              <div>Переход статусов, отображение «Дата начала».</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Окончание</strong>
                <small className="muted">время конца</small>
              </div>
              <div>Время окончания тренировки.</div>
              <div>Переход на «На проверке» после окончания, отображение «Дата окончания».</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Шаблон голосования</strong>
                <small className="muted">привязка</small>
              </div>
              <div>Какой шаблон использовать для публикации опроса.</div>
              <div>Список вариантов, учёт (counted options), веса вариантов.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>День публикации опроса</strong>
                <small className="muted">pollPublishWeekday</small>
              </div>
              <div>В какой день недели публиковать голосование (можно отличать от дня тренировки).</div>
              <div>Автопубликация опроса, когда начинается сбор голосов.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Время публикации опроса</strong>
                <small className="muted">pollPublishTime</small>
              </div>
              <div>Во сколько публиковать голосование.</div>
              <div>Автопубликация опроса, «окно голосования».</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Публикации активны</strong>
                <small className="muted">publishEnabled</small>
              </div>
              <div>Глобальный переключатель авто-публикаций для этого события.</div>
              <div>Автопубликация опросов/уведомлений, появление события в расписании.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Анонс</strong>
                <small className="muted">announcementEnabled</small>
              </div>
              <div>Включает/выключает сообщение-анонс перед тренировкой (если настроено).</div>
              <div>Отправку анонса в группу Telegram.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Текст анонса</strong>
                <small className="muted">announcementText</small>
              </div>
              <div>Текст, который будет отправлен в группу как анонс.</div>
              <div>Контент сообщения, формат публикации.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>За сколько минут</strong>
                <small className="muted">announcementLeadMinutes</small>
              </div>
              <div>За какое время до начала тренировки отправлять анонс.</div>
              <div>Тайминг анонса.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Автораспределение команд</strong>
                <small className="muted">teamsAutoSplit</small>
              </div>
              <div>Если включено, система может автоматически распределять игроков по командам (по рейтингу/правилам).</div>
              <div>Вкладку «Распределение», кнопки автосплита и расчёт баланса.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Публиковать список команд</strong>
                <small className="muted">teamsPublishList</small>
              </div>
              <div>Определяет, будет ли публиковаться состав команд в группу.</div>
              <div>Кнопку/действие «Опубликовать состав» и текст публикации.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Размер команды</strong>
                <small className="muted">teamSize</small>
              </div>
              <div>Целевое количество игроков в команде (используется в распределении).</div>
              <div>Автораспределение, подсказки по заполнению команд.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Минимум голосов для проведения</strong>
                <small className="muted">minVotesToHold</small>
              </div>
              <div>Минимальный порог “учтённых” голосов, чтобы тренировка считалась состоявшейся.</div>
              <div>Логику статусов и уведомлений об отмене (если включено).</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Отмена: за сколько минут</strong>
                <small className="muted">cancelLeadMinutes</small>
              </div>
              <div>За сколько минут до начала проверять порог голосов и при необходимости отменять.</div>
              <div>Переход в «Не состоялось» и/или уведомления.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Уведомлять об отмене</strong>
                <small className="muted">cancelNotifyEnabled</small>
              </div>
              <div>Включает сообщение в группу, если тренировка отменена по порогу голосов.</div>
              <div>Отправку уведомления в Telegram.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Стоимость</strong>
                <small className="muted">costAmount</small>
              </div>
              <div>Сумма, которая делится на количество учтённых “мест” (с учётом весов) и даёт «на человека».</div>
              <div>Расчёт оплат, задолженности, публикации расчёта.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Расчёт включен</strong>
                <small className="muted">settlementEnabled</small>
              </div>
              <div>Глобальный переключатель, делать ли расчёт оплаты для события.</div>
              <div>Доступность вкладки «Оплата» и публикаций расчёта.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Публиковать расчёт до</strong>
                <small className="muted">settlementPublishBefore</small>
              </div>
              <div>Если включено, можно публиковать расчёт до тренировки (предварительный).</div>
              <div>Тайминг публикации и сценарий “предоплаты”.</div>
            </div>

            <div className="docs-setting-row">
              <div>
                <strong>Публиковать расчёт после</strong>
                <small className="muted">settlementPublishAfter</small>
              </div>
              <div>Если включено, можно публиковать расчёт после тренировки.</div>
              <div>Сценарий “оплата по факту”, контроль задолженностей.</div>
            </div>
          </div>
        </section>

        <section id="docs-workflow" className="docs-section">
          <div className="docs-section-head">
            <h3>Жизненный цикл тренировки</h3>
            <p className="muted">Статус меняется автоматически по времени начала/окончания тренировки и по состоянию оплат.</p>
          </div>
          <div className="docs-media">
            <img className="docs-figure" src="/docs/workflow.svg" alt="" />
          </div>
          <div className="docs-grid docs-grid-2">
            <article className="docs-card">
              <h4>В голосовании</h4>
              <p className="muted">
                Активный сбор голосов. Длится до момента <strong>за 30 минут до старта</strong> (старт распределения).
              </p>
              <ul className="docs-list">
                <li>В этот период Telegram-изменения (поставили/сняли голос) учитываются автоматически.</li>
                <li>В расчёты и распределение попадают только «учтённые» варианты из шаблона голосования.</li>
              </ul>
            </article>
            <article className="docs-card">
              <h4>На распределении</h4>
              <p className="muted">
                Начинается <strong>за 30 минут до старта</strong> и длится до <strong>времени окончания</strong>.
              </p>
              <ul className="docs-list">
                <li>В этот момент «окно голосования» закрыто: поздние изменения из Telegram больше не меняют учёт.</li>
                <li>Админ распределяет игроков по командам, сохраняет и при необходимости публикует состав.</li>
              </ul>
            </article>
            <article className="docs-card">
              <h4>На проверке</h4>
              <p className="muted">Этап контроля оплат после окончания тренировки.</p>
              <ul className="docs-list">
                <li>Появляется после окончания, если есть <strong>неоплаченные</strong> суммы по расчёту.</li>
                <li>Админ может пересчитать «Сформировать расчёт» (если менялись голоса) и сохранить оплаты.</li>
              </ul>
            </article>
            <article className="docs-card">
              <h4>Завершено</h4>
              <p className="muted">Финальный статус: долгов по расчёту не осталось.</p>
              <ul className="docs-list">
                <li>Ставится автоматически после окончания, если неоплаченных сумм нет.</li>
                <li>Также станет «Завершено», когда все оплаты отмечены.</li>
              </ul>
            </article>
          </div>
        </section>

        <section id="docs-votes" className="docs-section">
          <div className="docs-section-head">
            <h3>Голоса и учёт</h3>
            <p className="muted">Внутри тренировки доступна вкладка «Голоса» со списком выборов.</p>
          </div>
          <div className="docs-callout">
            <strong>Админская правка:</strong> можно удалить конкретный голос кнопкой «−». Это приведёт к перерасчёту оплаты для тренировки.
          </div>
        </section>

        <section id="docs-teams" className="docs-section">
          <div className="docs-section-head">
            <h3>Команды и распределение</h3>
            <p className="muted">Кто попадает в распределение и как работает автораспределение.</p>
          </div>
          <div className="docs-media">
            <img className="docs-figure" src="/docs/teams.svg" alt="" />
          </div>
          <div className="docs-grid docs-grid-2">
            <article className="docs-card">
              <h4>Кто участвует</h4>
              <p className="muted">Распределение строится на базе «учтённых» вариантов голосования.</p>
              <ul className="docs-list">
                <li>Берутся только варианты, отмеченные как <strong>учёт</strong> в шаблоне голосования.</li>
                <li>У каждого варианта может быть <strong>вес</strong>: он добавляет «места» (например, +2).</li>
                <li>Если у голоса есть дополнительные места, создаются «гости» как отдельные слоты.</li>
              </ul>
            </article>
            <article className="docs-card">
              <h4>Рейтинг игрока</h4>
              <p className="muted">Нужен для баланса сил команд.</p>
              <ul className="docs-list">
                <li>Рейтинг считается как <strong>среднее</strong> по навыкам игрока.</li>
                <li>Если у игрока нет оценок, используется нейтральная база <strong>5.0</strong>.</li>
                <li>Гостевые слоты всегда идут с рейтингом <strong>5.0</strong>.</li>
              </ul>
            </article>
          </div>

          <div className="docs-callout">
            <strong>Важно:</strong> автораспределение не “угадывает идеал”, оно даёт устойчивую базу, которую можно вручную донастроить drag-and-drop и сохранить.
          </div>

          <div className="docs-grid docs-grid-2">
            <article className="docs-card">
              <h4>Сколько команд и вместимость</h4>
              <p className="muted">Система сама определяет A/B или A/B/C.</p>
              <ul className="docs-list">
                <li>По умолчанию распределяем в <strong>2 команды</strong>: A и B.</li>
                <li>Команда C появляется, если игроков <strong>больше 14</strong> или если ранее уже была сохранена команда C.</li>
                <li>Вместимость команд считается равномерно: разница максимум 1 игрок.</li>
              </ul>
            </article>
            <article className="docs-card">
              <h4>Как выбирается команда (встроенный алгоритм)</h4>
              <p className="muted">Алгоритм пытается минимизировать дисбаланс рейтингов и учесть роли/связи.</p>
              <ul className="docs-list">
                <li>Сначала распределяются <strong>связующие</strong> (setter), затем <strong>либеро</strong>, затем все остальные.</li>
                <li>Внутри каждой группы игроки идут по убыванию рейтинга.</li>
                <li>Для каждого игрока выбирается команда с “лучшей” метрикой с учётом текущей силы, ролей и связей.</li>
              </ul>
            </article>
          </div>

          <div className="docs-grid docs-grid-2">
            <article className="docs-card">
              <h4>Роли (player type)</h4>
              <p className="muted">Роли учитываются, чтобы не сложить всех ключевых игроков в одну команду.</p>
              <ul className="docs-list">
                <li>Поддерживаются роли типа <strong>setter</strong> и <strong>libero</strong> (если выставлены игрокам в профиле).</li>
                <li>Алгоритм добавляет штраф за перекос по ролям, чтобы распределять их равномернее.</li>
              </ul>
            </article>
            <article className="docs-card">
              <h4>Связи игроков</h4>
              <p className="muted">Связи влияют на выбор команды через штрафы/бонусы.</p>
              <ul className="docs-list">
                <li><strong>prefer_together</strong>: стараемся держать вместе (штраф если в разных командах, бонус если в одной).</li>
                <li><strong>avoid_together</strong>: стараемся разводить по разным командам (штраф если попали вместе).</li>
                <li>Вес связи усиливает эффект.</li>
              </ul>
            </article>
          </div>

          <div className="docs-callout">
            <strong>Расширенный режим:</strong> если задан переменный окружения <strong>TEAM_SPLIT_SERVICE_URL</strong>, система сначала пробует внешний сервис распределения и при недоступности откатывается на встроенный алгоритм.
          </div>
        </section>

        <section id="docs-sets" className="docs-section">
          <div className="docs-section-head">
            <h3>Партии (сеты)</h3>
            <p className="muted">Фиксируй счёт партий и публикуй результаты в группу.</p>
          </div>
          <div className="docs-grid">
            <article className="docs-card">
              <h4>2–3 команды</h4>
              <p className="muted">Для каждой партии выбирается, какие команды играют, и задаётся счёт.</p>
            </article>
            <article className="docs-card">
              <h4>Публикация</h4>
              <p className="muted">Один клик, чтобы отправить итог по партиям в Telegram-группу.</p>
            </article>
            <article className="docs-card">
              <h4>История</h4>
              <p className="muted">Данные хранятся построчно, чтобы потом строить статистику.</p>
            </article>
          </div>
        </section>

        <section id="docs-billing" className="docs-section">
          <div className="docs-section-head">
            <h3>Оплата и перерасчёт</h3>
            <p className="muted">Вкладка «Оплата» показывает начисления и позволяет отметить оплативших.</p>
          </div>
          <div className="docs-callout">
            <strong>Сформировать расчёт</strong> всегда пересчитывает стоимость на основании актуальных «учтённых» голосов (с учётом весов и настроек).
          </div>
          <div className="docs-media">
            <div className="docs-ui-preview" aria-label="Пример вкладки оплаты">
              <div className="template-head">
                <h4>Оплата события</h4>
              </div>
              <div className="detail-grid">
                <div>
                  <span>Дата расчёта</span>
                  <strong>20.02.2026, 19:00</strong>
                </div>
                <div>
                  <span>Участников</span>
                  <strong>12</strong>
                </div>
                <div>
                  <span>На человека</span>
                  <strong>350.00 ₽</strong>
                </div>
                <div>
                  <span>Оплачено</span>
                  <strong>7</strong>
                </div>
                <div>
                  <span>Не оплачено</span>
                  <strong>5</strong>
                </div>
                <div>
                  <span>Долг по событию</span>
                  <strong>1750.00 ₽</strong>
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
                    <tr>
                      <td>Михаил Козлов</td>
                      <td>350.00 ₽</td>
                      <td className="col-center">
                        <label className="toggle-field toggle-field-only">
                          <input type="checkbox" checked readOnly />
                        </label>
                      </td>
                    </tr>
                    <tr>
                      <td>Анна Смирнова</td>
                      <td>350.00 ₽</td>
                      <td className="col-center">
                        <label className="toggle-field toggle-field-only">
                          <input type="checkbox" readOnly />
                        </label>
                      </td>
                    </tr>
                    <tr>
                      <td>@player_nick</td>
                      <td>700.00 ₽</td>
                      <td className="col-center">
                        <label className="toggle-field toggle-field-only">
                          <input type="checkbox" readOnly />
                        </label>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div className="manual-controls">
                <button type="button" className="btn-secondary" disabled>
                  Опубликовать расчёт
                </button>
                <button type="button" className="btn-secondary" disabled>
                  Сформировать расчет
                </button>
                <button type="button" disabled>
                  Сохранить оплаты
                </button>
              </div>
            </div>
          </div>
        </section>

        <section id="docs-debts" className="docs-section">
          <div className="docs-section-head">
            <h3>Задолженности</h3>
            <p className="muted">Список игроков, кто должен деньги, с деревом тренировок. Есть публикация в группу с чекбоксами.</p>
          </div>
          <div className="docs-media">
            <div className="docs-ui-preview docs-ui-preview-compact" aria-label="Пример вкладки задолженностей">
              <div className="template-head">
                <h4>Задолженности</h4>
                <button type="button" disabled>
                  Опубликовать
                </button>
              </div>

              <div className="table-wrap">
                <table className="table billing-debtors-table">
                  <colgroup>
                    <col style={{ width: '44%' }} />
                    <col style={{ width: '28%' }} />
                    <col style={{ width: '20%' }} />
                    <col style={{ width: '8%' }} />
                  </colgroup>
                  <thead>
                    <tr>
                      <th>Игрок</th>
                      <th>Реальное ФИО</th>
                      <th className="col-center">Долг за все тренировки</th>
                      <th className="col-center">Публиковать</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr className="member-row">
                      <td>
                        <div className="person-cell">
                          <strong>Михаил Дмитриевич</strong>
                          <span>@iamjq1</span>
                        </div>
                      </td>
                      <td>Михаил Козлов</td>
                      <td className="col-center">
                        <span className="badge badge-debt debt-badge">4000.00 ₽</span>
                      </td>
                      <td className="col-center">
                        <label className="toggle-field toggle-field-only">
                          <input type="checkbox" checked readOnly />
                        </label>
                      </td>
                    </tr>

                    <tr className="billing-expand-row">
                      <td colSpan={4}>
                        <div className="list-block debt-trainings" style={{ marginTop: 10 }}>
                          <div className="billing-training-row clickable tree-first">
                            <div className="billing-training-info">
                              <div className="tree-gutter" aria-hidden="true">
                                <span className="tree-elbow" />
                              </div>
                              <div className="billing-training-text">
                                <strong>Волейбол в пятницу</strong>
                                <p className="muted">20.02.2026, 19:00</p>
                              </div>
                            </div>
                            <div className="billing-training-debt">
                              <span className="badge badge-debt debt-badge">4000.00 ₽</span>
                            </div>
                            <div className="billing-training-spacer" aria-hidden="true" />
                          </div>
                          <div className="billing-training-row clickable tree-last">
                            <div className="billing-training-info">
                              <div className="tree-gutter" aria-hidden="true">
                                <span className="tree-elbow" />
                              </div>
                              <div className="billing-training-text">
                                <strong>Тренировка в воскресенье</strong>
                                <p className="muted">23.02.2026, 11:00</p>
                              </div>
                            </div>
                            <div className="billing-training-debt">
                              <span className="badge badge-debt debt-badge">0.00 ₽</span>
                            </div>
                            <div className="billing-training-spacer" aria-hidden="true" />
                          </div>
                        </div>
                      </td>
                    </tr>

                    <tr className="member-row">
                      <td>
                        <div className="person-cell">
                          <strong>@player_nick</strong>
                          <span>ID 123456</span>
                        </div>
                      </td>
                      <td>-</td>
                      <td className="col-center">
                        <span className="badge badge-debt debt-badge">700.00 ₽</span>
                      </td>
                      <td className="col-center">
                        <label className="toggle-field toggle-field-only">
                          <input type="checkbox" checked readOnly />
                        </label>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </section>

        <section id="docs-roles" className="docs-section">
          <div className="docs-section-head">
            <h3>Роли и доступы</h3>
            <p className="muted">Админ видит больше разделов и может выполнять действия, влияющие на расчёты и публикации.</p>
          </div>
          <div className="docs-grid docs-grid-2">
            <article className="docs-card">
              <h4>Администратор</h4>
              <p className="muted">Управляет шаблонами, правами, расчётами, публикациями и ручными корректировками.</p>
            </article>
            <article className="docs-card">
              <h4>Участник</h4>
              <p className="muted">Видит профиль и историю, в рамках выданных прав.</p>
            </article>
          </div>
        </section>

        <section className="docs-footer">
          <p className="muted">Версия документации привязана к текущей версии консоли.</p>
        </section>
      </section>
    )
  }

  function renderBilling() {
    if (activeChatID === null) {
      return <section className="content-card">Выбери организацию</section>
    }

    const publishIncluded = Object.keys(billingPublishDraft)
      .filter((k) => billingPublishDraft[Number(k)])
      .map((k) => Number(k))
      .filter((n) => Number.isFinite(n) && n !== 0)

    return (
      <section className="content-card">
        <div className="template-head">
          <h3>Задолженности</h3>
          <button
            type="button"
            disabled={billingLoading || publishIncluded.length === 0}
            onClick={() =>
              void runAction(
                async () => {
                  if (activeChatID === null) return
                  await publishGroupDebtors(activeChatID, publishIncluded)
                },
                'Задолженности опубликованы',
              )
            }
          >
            Опубликовать
          </button>
        </div>

        {billingLoading ? <p className="muted">Загрузка задолженностей...</p> : null}
        {billingError ? <p className="muted">Ошибка: {billingError}</p> : null}

        {!billingLoading && !billingError && billingDebtors.length === 0 ? <p className="muted">Задолженностей нет</p> : null}

        {!billingLoading && !billingError && billingDebtors.length > 0 ? (
          <div className="table-wrap">
            <table className="table billing-debtors-table">
              <colgroup>
                <col style={{ width: '44%' }} />
                <col style={{ width: '28%' }} />
                <col style={{ width: '20%' }} />
                <col style={{ width: '8%' }} />
              </colgroup>
              <thead>
                <tr>
                  <th>Игрок</th>
                  <th>Реальное ФИО</th>
                  <th className="col-center">Долг за все тренировки</th>
                  <th className="col-center">Публиковать</th>
                </tr>
              </thead>
              <tbody>
                {billingDebtors.flatMap((debtor) => {
                  const expanded = Boolean(billingExpanded[debtor.userID])
                  const rows: JSX.Element[] = []
                  rows.push(
                    <tr
                      key={`debtor-${debtor.userID}`}
                      className="member-row"
                      onClick={() =>
                        setBillingExpanded((prev) => ({
                          ...prev,
                          [debtor.userID]: !prev[debtor.userID],
                        }))
                      }
                    >
                      <td>
                        <div className="person-cell">
                          <strong>{debtorDisplayName(debtor)}</strong>
                          <span>{debtor.username ? `@${debtor.username}` : `ID ${debtor.userID}`}</span>
                        </div>
                      </td>
                      <td>{debtor.realName || '-'}</td>
                      <td className="col-center">
                        <span className="badge badge-debt debt-badge">{formatMoney(debtor.totalDebt)}</span>
                      </td>
                      <td className="col-center">
                        <label
                          className="toggle-field toggle-field-only"
                          onClick={(e) => {
                            e.stopPropagation()
                          }}
                        >
                          <input
                            type="checkbox"
                            checked={Boolean(billingPublishDraft[debtor.userID])}
                            onChange={(e) =>
                              setBillingPublishDraft((prev) => ({
                                ...prev,
                                [debtor.userID]: e.target.checked,
                              }))
                            }
                          />
                        </label>
                      </td>
                    </tr>,
                  )

                  if (expanded) {
                    rows.push(
                      <tr key={`debtor-expand-${debtor.userID}`} className="billing-expand-row">
                        <td colSpan={4}>
                          {debtor.trainings?.length ? (
                            <div className="list-block debt-trainings" style={{ marginTop: 10 }}>
                              {debtor.trainings.map((t, idx) => {
                                const isFirst = idx === 0
                                const isLast = idx === debtor.trainings.length - 1
                                return (
                                <div
                                  key={`${debtor.userID}-${t.instanceID}`}
                                  className={`billing-training-row clickable ${isFirst ? 'tree-first' : ''} ${isLast ? 'tree-last' : ''}`}
                                  role="button"
                                  tabIndex={0}
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    navigateTo({
                                      chatID: activeChatID,
                                      section: 'events',
                                      historyEventID: t.instanceID,
                                      templateView: 'list',
                                      templateName: null,
                                      eventView: 'list',
                                      eventID: null,
                                    })
                                  }}
                                  onKeyDown={(e) => {
                                    if (e.key !== 'Enter' && e.key !== ' ') return
                                    e.preventDefault()
                                    e.stopPropagation()
                                    navigateTo({
                                      chatID: activeChatID,
                                      section: 'events',
                                      historyEventID: t.instanceID,
                                      templateView: 'list',
                                      templateName: null,
                                      eventView: 'list',
                                      eventID: null,
                                    })
                                  }}
                                >
                                  <div className="billing-training-info">
                                    <div className="tree-gutter" aria-hidden="true">
                                      <span className="tree-elbow" />
                                    </div>
                                    <div className="billing-training-text">
                                      <strong>{t.eventName || `Событие #${t.instanceID}`}</strong>
                                      <p className="muted">{formatDateTime(t.startAt)}</p>
                                    </div>
                                  </div>
                                  <div className="billing-training-debt">
                                    <span className="badge badge-debt debt-badge">{formatMoney(t.amountDue)}</span>
                                  </div>
                                  <div className="billing-training-spacer" aria-hidden="true" />
                                </div>
                                )
                              })}
                            </div>
                          ) : (
                            <p className="muted" style={{ marginTop: 10 }}>
                              Нет привязанных тренировок
                            </p>
                          )}
                        </td>
                      </tr>,
                    )
                  }
                  return rows
                })}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>
    )
  }

  function renderMembers() {
    if (activeMemberID !== null) {
      return (
        <section className="content-card">
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
          {memberSkillError ? <p className="muted">Ошибка: {memberSkillError}</p> : null}
          {!memberSkillLoading && !memberSkillError && !selectedMemberSkills ? <p className="muted">Игрок не найден</p> : null}
          {!memberSkillLoading && selectedMemberSkills ? (
            <>
              <div className="player-fifa-card">
                <div className="player-avatar">{initialsFromProfile(selectedMemberSkills)}</div>
                <div className="player-ident">
                  <p className="player-fio">
                    {(() => {
                      const fio = fullNamePartsFromProfile(selectedMemberSkills)
                      return `${fio.lastName} ${fio.firstName} ${fio.patronymic}`
                    })()}
                  </p>
                  <p className="player-meta">@{selectedMemberSkills.username || 'без_ника'}</p>
                  <p className="player-meta">ID: {selectedMemberSkills.userTelegramID}</p>
                </div>
                <div
                  className="player-overall"
                  style={{
                    borderColor: scoreColor(averageScore(selectedMemberSkills, skillsCatalog)),
                    color: scoreColor(averageScore(selectedMemberSkills, skillsCatalog)),
                  }}
                >
                  <span>OVERALL</span>
                  <strong>{averageScore(selectedMemberSkills, skillsCatalog)?.toFixed(1) ?? '-'}</strong>
                </div>
              </div>

              <div className="form-grid form-grid-3" style={{ marginTop: 16 }}>
                <label className="field" style={{ gridColumn: '1 / -1' }}>
                  <span>Реальное имя</span>
                  <input
                    placeholder="Например: Иван Иванов"
                    value={memberRealNameDraft}
                    onChange={(e) => setMemberRealNameDraft(e.target.value)}
                  />
                </label>
              </div>

              <div className="skill-slider-grid">
                <div className="player-type-block">
                  <div className="player-type-head">
                    <span className="skill-slider-name">Тип игрока</span>
                    <span className="skill-slider-value">{playerTypeLabel(memberPlayerTypeDraft)}</span>
                  </div>
                  <div className="player-type-grid">
                    {playerTypeCards.map((typeCard) => (
                      <button
                        key={typeCard.value}
                        type="button"
                        className={`player-type-card ${memberPlayerTypeDraft === typeCard.value ? 'active' : ''}`}
                        data-player-type={typeCard.value}
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
                </div>
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
              <section className="content-card relation-block">
                <h4>Связи игрока</h4>
                <div className="form-grid form-grid-3 relation-form-grid">
                  <label className="field relation-member-field">
                    <span>Игрок</span>
                    <div className={`relation-member-picker ${memberRelationPickerOpen ? 'open' : ''}`} ref={memberRelationPickerRef}>
                      <button
                        type="button"
                        className={`relation-picker-trigger ${selectedRelationMemberOption ? '' : 'empty'}`}
                        onClick={() => {
                          setMemberRelationPickerOpen((prev) => !prev)
                          setMemberRelationPickerQuery('')
                        }}
                        aria-haspopup="listbox"
                        aria-expanded={memberRelationPickerOpen}
                      >
                        <span>{selectedRelationMemberOption?.label || 'Выбери игрока для связи'}</span>
                        <span className="relation-picker-caret" aria-hidden="true">
                          {memberRelationPickerOpen ? '▴' : '▾'}
                        </span>
                      </button>
                      {memberRelationPickerOpen ? (
                        <div className="relation-picker-dropdown">
                          <div className="relation-picker-search">
                            <input
                              autoFocus
                              placeholder="Поиск по реальному имени, нику или ID"
                              value={memberRelationPickerQuery}
                              onChange={(e) => onRelationMemberPickerChange(e.target.value)}
                              onKeyDown={(e) => {
                                if (e.key === 'Escape') {
                                  setMemberRelationPickerOpen(false)
                                }
                                if (e.key === 'Enter' && filteredRelationMemberOptions.length > 0) {
                                  e.preventDefault()
                                  onRelationMemberSelect(filteredRelationMemberOptions[0].id)
                                }
                              }}
                            />
                          </div>
                          <div className="relation-picker-options" role="listbox">
                            {filteredRelationMemberOptions.length === 0 ? (
                              <p className="relation-picker-empty muted">Игрок не найден</p>
                            ) : (
                              filteredRelationMemberOptions.map((option) => (
                                <button
                                  key={option.id}
                                  type="button"
                                  className={`relation-picker-option ${String(option.id) === memberRelationDraft.otherUserID ? 'active' : ''}`}
                                  onClick={() => onRelationMemberSelect(option.id)}
                                >
                                  {option.label}
                                </button>
                              ))
                            )}
                          </div>
                        </div>
                      ) : null}
                    </div>
                  </label>
                  <label className="field">
                    <span>Тип связи</span>
                    <select
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
                    </select>
                  </label>
                  <label className="field">
                    <span>Вес (1-10)</span>
                    <input
                      type="number"
                      min="1"
                      max="10"
                      step="1"
                      value={memberRelationDraft.weight}
                      onChange={(e) => setMemberRelationDraft((prev) => ({ ...prev, weight: e.target.value }))}
                    />
                  </label>
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
                            {relationTypeLabel(relation.relationType)} · вес {relation.weight}
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
              <div className="manual-controls">
                <button type="button" onClick={() => void onSaveMemberSkills()}>
                  Сохранить
                </button>
              </div>
            </>
          ) : null}
        </section>
      )
    }

    return (
      <section className="content-card">
        <div className="template-head">
          <h3>Игроки группы</h3>
        </div>
        {membersError ? <p className="muted">Подписчики временно недоступны: {membersError}</p> : null}
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
            <select
              value={membersTypeFilter}
              onChange={(e) => setMembersTypeFilter(e.target.value as '' | 'attacker' | 'setter' | 'libero' | 'central')}
            >
              <option value="">Все типы</option>
              <option value="attacker">Атакующий</option>
              <option value="setter">Пасующий</option>
              <option value="libero">Либеро</option>
              <option value="central">Центральный</option>
            </select>
          </label>
          <label className="field">
            <span>Результат</span>
            <input value={`${filteredMembers.length} из ${members.length}`} readOnly />
          </label>
        </div>
        {members.length === 0 ? (
          <p className="muted">Пока нет данных об игроках</p>
        ) : filteredMembers.length === 0 ? (
          <p className="muted">По текущим фильтрам игроков не найдено</p>
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Игрок</th>
                  <th>Реальное имя</th>
                  <th>Тип игрока</th>
                  <th>
                    <div className="members-head-title">Характеристики</div>
                    <div className="members-skill-sort-grid">
                      {skillsCatalog.map((skill) => (
                        <div className="members-skill-sort-item" key={`sort-${skill.code}`}>
                          <span>
                            {skill.name}
                            {membersSortOrderByCode.has(skill.code) ? (
                              <em className="members-sort-order">{membersSortOrderByCode.get(skill.code)}</em>
                            ) : null}
                          </span>
                          <div className="members-sort-arrows">
                            <button
                              type="button"
                              className={`members-sort-arrow ${
                                membersSortCriteria.some((item) => item.code === skill.code && item.direction === 'asc') ? 'active' : ''
                              }`}
                              onClick={() => onMembersSort(skill.code, 'asc')}
                              aria-label={`Сортировать по ${skill.name} по возрастанию`}
                              title={`Сортировать по ${skill.name} по возрастанию`}
                            >
                              ↑
                            </button>
                            <button
                              type="button"
                              className={`members-sort-arrow ${
                                membersSortCriteria.some((item) => item.code === skill.code && item.direction === 'desc') ? 'active' : ''
                              }`}
                              onClick={() => onMembersSort(skill.code, 'desc')}
                              aria-label={`Сортировать по ${skill.name} по убыванию`}
                              title={`Сортировать по ${skill.name} по убыванию`}
                            >
                              ↓
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </th>
                  <th>
                    <div className="members-average-head">
                      <span>Средняя</span>
                      <div className="members-sort-arrows">
                        <button
                          type="button"
                          className={`members-sort-arrow ${
                            membersSortCriteria.some(
                              (item) => item.code === MEMBERS_SORT_AVERAGE_CODE && item.direction === 'asc',
                            )
                              ? 'active'
                              : ''
                          }`}
                          onClick={() => onMembersSort(MEMBERS_SORT_AVERAGE_CODE, 'asc')}
                          aria-label="Сортировать по средней по возрастанию"
                          title="Сортировать по средней по возрастанию"
                        >
                          ↑
                        </button>
                        <button
                          type="button"
                          className={`members-sort-arrow ${
                            membersSortCriteria.some(
                              (item) => item.code === MEMBERS_SORT_AVERAGE_CODE && item.direction === 'desc',
                            )
                              ? 'active'
                              : ''
                          }`}
                          onClick={() => onMembersSort(MEMBERS_SORT_AVERAGE_CODE, 'desc')}
                          aria-label="Сортировать по средней по убыванию"
                          title="Сортировать по средней по убыванию"
                        >
                          ↓
                        </button>
                      </div>
                    </div>
                  </th>
                </tr>
              </thead>
              <tbody>
                {filteredMembers.map((member) => (
                  <tr key={member.userTelegramID} className="member-row" onClick={() => void onOpenMemberCard(member)}>
                    <td>
                      <div className="person-cell">
                        <strong>{fullName(member)}</strong>
                        <span>ID: {member.userTelegramID}</span>
                      </div>
                    </td>
                    <td>
                      {(memberSkillProfiles[member.userTelegramID]?.realName || member.realName || '').trim() ? (
                        (memberSkillProfiles[member.userTelegramID]?.realName || member.realName || '').trim()
                      ) : (
                        <span className="muted">—</span>
                      )}
                    </td>
                    <td>
                      <span className="skill-chip">
                        {playerTypeLabel(
                          (memberSkillProfiles[member.userTelegramID]?.playerType || member.playerType || '') as
                            | ''
                            | 'attacker'
                            | 'setter'
                            | 'libero'
                            | 'central',
                        )}
                      </span>
                    </td>
                    <td>
                      <div className="skill-chip-wrap">
                        {profileSkills(memberSkillProfiles[member.userTelegramID] ?? null, skillsCatalog).map((skill) => (
                          <span className="skill-chip" key={`${member.userTelegramID}-${skill.skillCode}`}>
                            {skill.skillName}: {normalizedSkillScore(skill.score)}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td>
                      {(() => {
                        const avg = averageScore(memberSkillProfiles[member.userTelegramID] ?? null, skillsCatalog)
                        const color = scoreColor(avg)
                        return (
                          <span className="score-pill" style={{ borderColor: color, color }}>
                            {avg === null ? '-' : avg.toFixed(1)}
                          </span>
                        )
                      })()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
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
        {templateEditorError ? <p className="muted">Ошибка загрузки: {templateEditorError}</p> : null}

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
                    <input
                      type="checkbox"
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
                  <input
                    type="checkbox"
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

        <div className="list-block">
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
                <div>
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
            ← К списку событий
          </button>
          <h3>Создать событие</h3>
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
              <select className="ui-select" value={eventForm.eventType} onChange={(e) => setEventForm((prev) => ({ ...prev, eventType: e.target.value as 'training' | 'activity' }))}>
                <option value="training">Тренировка</option>
                <option value="activity">Мероприятие</option>
              </select>
            </label>
            <label className="field">
              <span>День недели</span>
              <select
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
              </select>
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
              <select
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
              </select>
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
              <select
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
              </select>
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
          <section className="settings-card settings-card-settings">
            <h4>Настройки</h4>
            <div className="settings-grid">
              <div className="settings-group">
                <p className="settings-group-title">Анонс</p>
                <div className="settings-row settings-row-2">
                  <label className="toggle-field toggle-field-inline">
                    <input
                      type="checkbox"
                      checked={eventForm.announcementEnabled}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, announcementEnabled: e.target.checked }))}
                    />
                    <span>Публиковать анонс</span>
                  </label>
                  <label className="field">
                    <span>Публиковать за</span>
                    <select
                      className="ui-select"
                      value={eventForm.announcementLeadMinutes}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, announcementLeadMinutes: e.target.value }))}
                    >
                      {announcementLeadOptions.map((item) => (
                        <option key={item.value} value={item.value}>
                          {item.label}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>
              </div>
              <div className="settings-group">
                <p className="settings-group-title">Расчёт</p>
                <div className="settings-row settings-row-2">
                  <label className="toggle-field toggle-field-inline">
                    <input
                      type="checkbox"
                      checked={eventForm.settlementEnabled}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, settlementEnabled: e.target.checked }))}
                      disabled
                    />
                    <span>Публиковать расчёт</span>
                  </label>
                  <label className="toggle-field toggle-field-inline">
                    <input
                      type="checkbox"
                      checked={eventForm.settlementPublishBefore}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, settlementPublishBefore: e.target.checked }))}
                    />
                    <span>Перед началом</span>
                  </label>
                  <label className="toggle-field toggle-field-inline">
                    <input
                      type="checkbox"
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
                      <input
                        type="checkbox"
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
          {eventForm.eventType === 'training' ? (
            <section className="settings-card settings-card-teams">
              <h4>Модуль деления на команды</h4>
              <div className="settings-grid">
                <div className="settings-row settings-row-3">
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventForm.teamsAutoSplit}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, teamsAutoSplit: e.target.checked }))}
                    />
                    <span>Автоделение на команды</span>
                  </label>
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventForm.teamsPublishList}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, teamsPublishList: e.target.checked }))}
                    />
                    <span>Публиковать список</span>
                  </label>
                  {/*
                  <label className="field">
                    <span>Игроков в команде</span>
                    <input
                      type="number"
                      min="2"
                      step="1"
                      value={eventForm.teamSize}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, teamSize: e.target.value }))}
                    />
                  </label>
                  */}
                </div>
              </div>
            </section>
          ) : null}
          <button type="submit">Создать событие</button>
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
              ← К списку событий
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
            ← К списку событий
          </button>
          <h3>
            Редактирование события #{selectedEvent.id}
          </h3>
          <div className="list-actions">
            <button type="button" className="btn-secondary" onClick={() => void onCreateFromTemplate(selectedEvent.id)}>
              Создать
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
            {eventActivityError ? <p className="muted">Ошибка: {eventActivityError}</p> : null}
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
              <select className="ui-select" value={eventEditor.eventType} onChange={(e) => setEventEditor((prev) => ({ ...prev, eventType: e.target.value as 'training' | 'activity' }))}>
                <option value="training">Тренировка</option>
                <option value="activity">Мероприятие</option>
              </select>
            </label>
            <label className="field">
              <span>День недели</span>
              <select
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
              </select>
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
              <select
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
              </select>
            </label>
            <label className="field">
              <span>День публикации</span>
              <select
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
              </select>
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

          <section className="settings-card settings-card-settings">
            <h4>Настройки</h4>
            <div className="settings-grid">
              <div className="settings-group">
                <p className="settings-group-title">Анонс</p>
                <div className="settings-row settings-row-2">
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventEditor.announcementEnabled}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, announcementEnabled: e.target.checked }))}
                    />
                    <span>Публиковать анонс</span>
                  </label>
                  <label className="field">
                    <span>Публиковать за</span>
                    <select
                      className="ui-select"
                      value={eventEditor.announcementLeadMinutes}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, announcementLeadMinutes: e.target.value }))}
                    >
                      {announcementLeadOptions.map((item) => (
                        <option key={item.value} value={item.value}>
                          {item.label}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>
              </div>
              <div className="settings-group">
                <p className="settings-group-title">Расчёт</p>
                <div className="settings-row settings-row-3">
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventEditor.settlementEnabled}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, settlementEnabled: e.target.checked }))}
                      disabled={!eventEditorCanEnableSettlement}
                    />
                    <span>Публиковать расчёт</span>
                  </label>
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventEditor.settlementPublishBefore}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, settlementPublishBefore: e.target.checked }))}
                    />
                    <span>Перед началом</span>
                  </label>
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
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
                      <input
                        type="checkbox"
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
          {eventEditor.eventType === 'training' ? (
            <section className="settings-card settings-card-teams">
              <h4>Модуль деления на команды</h4>
              <div className="settings-grid">
                <div className="settings-row settings-row-3">
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventEditor.teamsAutoSplit}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, teamsAutoSplit: e.target.checked }))}
                    />
                    <span>Автоделение на команды</span>
                  </label>
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventEditor.teamsPublishList}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, teamsPublishList: e.target.checked }))}
                    />
                    <span>Публиковать список</span>
                  </label>
                  {/*
                  <label className="field">
                    <span>Игроков в команде</span>
                    <input
                      type="number"
                      min="2"
                      step="1"
                      value={eventEditor.teamSize}
                      onChange={(e) => setEventEditor((prev) => ({ ...prev, teamSize: e.target.value }))}
                    />
                  </label>
                  */}
                </div>
              </div>
            </section>
          ) : null}

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
          <h3>События</h3>
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
              Создать событие
            </button>
          </div>
        </div>

        <div className="form-grid form-grid-3 members-filters">
          <label className="field">
            <span>Тип события</span>
            <select value={eventsTypeFilter} onChange={(e) => setEventsTypeFilter(e.target.value as '' | 'training' | 'activity')}>
              <option value="">Все типы</option>
              <option value="training">Тренировка</option>
              <option value="activity">Мероприятие</option>
            </select>
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
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>ID / Название</th>
                    <th>Тип</th>
                    <th>Расписание</th>
                    <th>Опрос</th>
                    <th>Статус</th>
                    <th>Стоимость</th>
                    <th>Действие</th>
                  </tr>
                </thead>
                <tbody>
                  {displayedEvents.map((event) => (
                    <tr
                      key={event.id}
                      className={eventsMode === 'active' ? 'member-row' : undefined}
                      onClick={
                        eventsMode === 'active'
                          ? () =>
                              navigateTo({
                                chatID: activeChatID,
                                section: 'event_templates',
                                templateView: 'list',
                                templateName: null,
                                eventView: 'edit',
                                eventID: event.id,
                              })
                          : undefined
                      }
                    >
                      <td>
                        <div className="person-cell">
                          <strong>#{event.id} {event.name}</strong>
                          <span>
                            {event.announcementEnabled
                              ? `Анонс: ${announcementLeadLabel(event.announcementLeadMinutes || 60)}`
                              : 'Анонс: выключен'}
                          </span>
                        </div>
                      </td>
                      <td>{event.eventType === 'training' ? 'Тренировка' : 'Мероприятие'}</td>
                      <td>
                        <div className="person-cell">
                          <strong>
                            {weekdayLabel(event.startWeekday)} · {toHourMinute(event.startTime)} - {toHourMinute(event.endTime)}
                          </strong>
                          <span>
                            Публикация: {weekdayLabel(event.pollPublishWeekday || event.startWeekday)} {toHourMinute(event.pollPublishTime)}
                          </span>
                        </div>
                      </td>
                      <td>{event.pollTemplate || 'не привязан'}</td>
                      <td>
                        <div className="person-cell">
                          <strong>{event.publishEnabled ? 'Публикации активны' : 'Публикации отключены'}</strong>
                          <span>
                            Расчёт: {event.settlementEnabled ? 'вкл' : 'выкл'} · до: {event.settlementPublishBefore ? 'да' : 'нет'} · после:{' '}
                            {event.settlementPublishAfter ? 'да' : 'нет'}
                          </span>
                        </div>
                      </td>
                      <td>{formatMoney(event.costAmount)}</td>
                      <td>
                        {eventsMode === 'active' ? (
                          <div className="list-actions">
                            <button
                              className="btn-secondary"
                              onClick={(e) => {
                                e.stopPropagation()
                                void onCreateFromTemplate(event.id)
                              }}
                            >
                              Создать
                            </button>
                            <button
                              className="btn-danger btn-icon"
                              title="В архив"
                              aria-label="В архив"
                              onClick={(e) => {
                                e.stopPropagation()
                                void onArchiveEvent(event.id)
                              }}
                            >
                              <span aria-hidden>🗃</span>
                            </button>
                          </div>
                        ) : (
                          <button className="btn-secondary" onClick={() => void onUnarchiveEvent(event.id)}>
                            Восстановить
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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
          <div className="template-head">
            <h3>События</h3>
          </div>
          <div className="form-grid form-grid-3 members-filters">
            <label className="field">
              <span>Статус</span>
              <select
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
              </select>
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
          {eventHistoryError ? <p className="muted">Ошибка: {eventHistoryError}</p> : null}
          {!eventHistoryLoading && !eventHistoryError && eventHistory.length === 0 ? <p className="muted">Событий пока нет</p> : null}

          {!eventHistoryLoading && filteredEventHistory.length > 0 ? (
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>ID / Название</th>
                    <th>Тип</th>
                    <th>Статус</th>
                    <th>Дата начала</th>
                    <th>Дата окончания</th>
                    <th>ID голосования</th>
                    <th>Долг</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredEventHistory.map((item) => (
                    <tr
                      key={item.instanceID}
                      className="member-row"
                      onClick={() =>
                        navigateTo({
                          chatID: activeChatID,
                          section: 'events',
                          historyEventID: item.instanceID,
                          templateView: 'list',
                          templateName: null,
                          eventView: 'list',
                          eventID: null,
                        })
                      }
                    >
                      <td>
                        <div className="person-cell">
                          <strong>#{item.instanceID} · {item.name}</strong>
                          <span>{item.publishEnabled ? 'Публикации активны' : 'Публикации отключены'}</span>
                        </div>
                      </td>
                      <td>{item.eventType === 'training' ? 'Тренировка' : 'Мероприятие'}</td>
                      <td>{historyStatusLabel(item.status)}</td>
                      <td>{formatDateTime(item.nextStartAt)}</td>
                      <td>{formatDateTime(item.endAt)}</td>
                      <td>{item.latestPostID ? `#${item.latestPostID}` : 'не создано'}</td>
                      <td>{formatMoney(item.debtAmount ?? 0)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
          {!eventHistoryLoading && eventHistory.length > 0 && filteredEventHistory.length === 0 ? (
            <p className="muted">По выбранному статусу событий нет</p>
          ) : null}
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
                section: 'events',
                historyEventID: null,
                templateView: 'list',
                templateName: null,
                eventView: 'list',
                eventID: null,
              })
            }
          >
            ← К списку событий
          </button>
          <h3>Событие #{activeHistoryEventID}</h3>
        </div>
        {eventHistoryLoading ? <p className="muted">Загрузка истории событий...</p> : null}
        {eventHistoryError ? <p className="muted">Ошибка: {eventHistoryError}</p> : null}

        {!eventHistoryLoading && !selectedHistoryEvent ? <p className="muted">Событие не найдено</p> : null}

	        {selectedHistoryEvent ? (
	          <section className="team-split-board">
	            <div className="team-split-head">
	              <div>
	                <strong>Распределение по командам: {selectedHistoryEvent.name}</strong>
	                <p className="muted">
	                  Статус: {historyStatusLabel(selectedHistoryEvent.status)} · Долг: {formatMoney(selectedHistoryEvent.debtAmount ?? 0)}
	                  <br />
	                  Начало: {formatDateTime(selectedHistoryEvent.nextStartAt)} · Окончание: {formatDateTime(selectedHistoryEvent.endAt)}
	                </p>
	              </div>
	              <div className="list-actions">
	                <button type="button" className="btn-danger" onClick={() => void onDeleteEventInstanceForTest()}>
	                  Удалить событие
	                </button>
	              </div>
	            </div>

            <div className="history-tabs">
              <button
                type="button"
                className={historyDetailTab === 'distribution' ? 'history-tab active' : 'history-tab'}
                onClick={() => setHistoryDetailTab('distribution')}
              >
                Распределение
              </button>
              <button
                type="button"
                className={historyDetailTab === 'votes' ? 'history-tab active' : 'history-tab'}
                onClick={() => setHistoryDetailTab('votes')}
                disabled={selectedHistoryPostID === null}
                title={selectedHistoryPostID === null ? 'Сначала выбери опрос' : undefined}
              >
                Голоса
              </button>
              <button
                type="button"
                className={historyDetailTab === 'billing' ? 'history-tab active' : 'history-tab'}
                onClick={() => setHistoryDetailTab('billing')}
              >
                Оплата
              </button>
              <button
                type="button"
                className={historyDetailTab === 'sets' ? 'history-tab active' : 'history-tab'}
                onClick={() => setHistoryDetailTab('sets')}
                disabled={selectedHistoryPostID === null}
                title={selectedHistoryPostID === null ? 'Сначала выбери опрос' : undefined}
              >
                Партии
              </button>
            </div>

            {historyDetailTab === 'distribution' ? (
              <>
                {eventPollHistoryLoading ? <p className="muted">Загружаю историю опросов...</p> : null}
                {eventPollHistoryError ? <p className="muted">Ошибка: {eventPollHistoryError}</p> : null}
                {!eventPollHistoryLoading && !eventPollHistoryError && eventPollHistory.length === 0 ? (
                  <p className="muted">Пока нет опубликованных опросов для этого события.</p>
                ) : null}

                {!eventPollHistoryLoading && eventPollHistory.length > 0 ? (
                  <div className="list-block">
                    {eventPollHistory.map((item) => (
                      <button
                        key={item.postID}
                        type="button"
                        className={selectedHistoryPostID === item.postID ? 'history-poll-btn active' : 'history-poll-btn'}
                        onClick={() => {
                          setSelectedHistoryPostID(item.postID)
                          setHistoryDetailTab('votes')
                        }}
                      >
                        <span>#{item.postID} · {formatDateTime(item.publishedAt)}</span>
                        <span>Учет: {item.countedVotes} / Всего: {item.totalVotes}</span>
                        <span>{item.teamsConfigured ? 'Команды сохранены' : 'Команды не сохранены'}</span>
                      </button>
                    ))}
                  </div>
                ) : null}

                {selectedHistoryPostID !== null ? (
              <>
                {teamSplitLoading ? <p className="muted">Загружаю участников...</p> : null}
                {teamSplitError ? <p className="muted">Ошибка: {teamSplitError}</p> : null}
                {!teamSplitLoading && teamSplit ? (
                  <>
                    {(() => {
                      const activeTeams = activeTeamCodes(teamSplit.players, teamCEnabled)
                      const pairRows = teamPairProbabilities(teamSplit.players, teamCEnabled).filter((pair) =>
                        activeTeams.includes(pair.left) && activeTeams.includes(pair.right),
                      )
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
                              <h5>{`Команда ${teamCode}`}</h5>
                              {teamSplit.formations?.[teamCode]?.scheme ? (
                                <small className="muted" title={teamSplit.formations?.[teamCode]?.analysis || ''}>
                                  Схема: {teamSplit.formations?.[teamCode]?.scheme}
                                </small>
                              ) : null}
                            </div>
                            {teamCode === 'C' ? (
                              <button type="button" className="team-remove-btn" onClick={removeTeamC} title="Убрать команду C">
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
                            {teamSplit.players.filter((player) => player.team === teamCode).length === 0 ? <p className="muted">Пусто</p> : null}
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
                              <h5>Нераспределенные</h5>
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
                                {unassignedPlayers.length === 0 ? <p className="muted">Пусто</p> : null}
                              </div>
                            </div>

                            <div className={activeTeams.length === 2 ? 'team-double-row' : 'team-triple-row'}>
                              {activeTeams.map((teamCode) => (
                                <div key={teamCode}>{renderTeamColumn(teamCode)}</div>
                              ))}
                              {activeTeams.length === 2 && !activeTeams.includes('C') ? (
                                <div className="team-add-slot">
                                  <button type="button" className="team-add-btn" onClick={() => setTeamCEnabled(true)} title="Добавить команду C">
                                    +
                                  </button>
                                </div>
                              ) : null}
                            </div>
                          </div>

                          {formationRows.length > 0 ? (
                            <>
                              <h5 className="team-analysis-title">Аналитика расстановки</h5>
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
                            </>
                          ) : null}

                          <h5 className="team-forecast-title">Прогноз</h5>
                          <div className="team-prob-list">
                            {(() => {
                              let orderedPairs = pairRows
                              const hasABC = activeTeams.length === 3 && activeTeams.includes('A') && activeTeams.includes('B') && activeTeams.includes('C')
                              if (hasABC) {
                                const getPair = (left: ActiveTeamCode, right: ActiveTeamCode) => {
                                  const direct = pairRows.find((pair) => pair.left === left && pair.right === right)
                                  if (direct) {
                                    return direct
                                  }
                                  const reverse = pairRows.find((pair) => pair.left === right && pair.right === left)
                                  if (reverse) {
                                    return { left, right, leftProb: reverse.rightProb, rightProb: reverse.leftProb }
                                  }
                                  return null
                                }
                                orderedPairs = [getPair('A', 'B'), getPair('C', 'A'), getPair('B', 'C')].filter(
                                  (item): item is { left: ActiveTeamCode; right: ActiveTeamCode; leftProb: number; rightProb: number } => item !== null,
                                )
                              }
                              return orderedPairs.map((pair) => {
                              return (
                                <div className="team-balance-row" key={`${pair.left}-${pair.right}`}>
                                  <span className="team-balance-label team-balance-label-left" style={{ color: teamColor(pair.left) }}>
                                    <strong>{`Команда ${pair.left}`}</strong>
                                    <small className="team-balance-percent">{`${(pair.leftProb * 100).toFixed(0)}%`}</small>
                                  </span>
                                  <div className="team-balance-bar">
                                    <div
                                      className="team-balance-fill team-balance-fill-left"
                                      style={{ width: `${(pair.leftProb * 100).toFixed(2)}%`, background: teamColor(pair.left) }}
                                    />
                                    <div
                                      className="team-balance-fill team-balance-fill-right"
                                      style={{ width: `${(pair.rightProb * 100).toFixed(2)}%`, background: teamColor(pair.right) }}
                                    />
                                    <div className="team-balance-marker" />
                                  </div>
                                  <span className="team-balance-label team-balance-label-right" style={{ color: teamColor(pair.right) }}>
                                    <strong>{`Команда ${pair.right}`}</strong>
                                    <small className="team-balance-percent">{`${(pair.rightProb * 100).toFixed(0)}%`}</small>
                                  </span>
                                </div>
                              )
                              })
                            })()}
                          </div>
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
              <section className="content-card">
                <div className="template-head">
                  <h4>Голоса</h4>
                </div>
                {selectedHistoryPostID === null ? <p className="muted">Выбери опрос в списке выше.</p> : null}
                {historyPollVotesLoading ? <p className="muted">Загрузка голосов...</p> : null}
                {historyPollVotesError ? <p className="muted">Ошибка: {historyPollVotesError}</p> : null}
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
                {selectedHistoryPostID !== null ? (
                  <div className="manual-controls">
                    <button
                      type="button"
                      className="btn-secondary"
                      onClick={() =>
                        navigateTo({
                          chatID: activeChatID,
                          section: 'polls',
                          pollPostID: selectedHistoryPostID,
                          templateView: 'list',
                          templateName: null,
                          eventView: 'list',
                          eventID: null,
                        })
                      }
                    >
                      Открыть голосование
                    </button>
                  </div>
                ) : null}
              </section>
            ) : null}

            {historyDetailTab === 'sets' ? (
              <section className="content-card">
                <div className="template-head">
                  <h4>Партии</h4>
                </div>
                {selectedHistoryPostID === null ? <p className="muted">Выбери опрос в списке выше.</p> : null}
                {eventSetsLoading ? <p className="muted">Загрузка партий...</p> : null}
                {eventSetsError ? <p className="muted">Ошибка: {eventSetsError}</p> : null}

                {selectedHistoryPostID !== null ? (
                  <>
                    {(() => {
                      const teams = teamSplit ? activeTeamCodes(teamSplit.players, teamCEnabled) : (['A', 'B'] as ActiveTeamCode[])
                      const availableTeams = teams.length >= 2 ? teams : (['A', 'B'] as ActiveTeamCode[])
                      const ensureOtherTeam = (t: ActiveTeamCode) => availableTeams.find((x) => x !== t) ?? availableTeams[0]

                      return (
                        <>
                          <div className="set-list">
                            {(eventSetRowsDraft.length ? eventSetRowsDraft : [{ left: availableTeams[0], right: availableTeams[1], leftScore: 0, rightScore: 0 }]).map(
                              (row, idx) => (
                                <div className="set-row" key={`set-row-${idx}`}>
                                  <span className="muted">{`Партия ${idx + 1}`}</span>
                                  <div className="set-row-main">
                                    {availableTeams.length > 2 ? (
                                      <select
                                        className="set-team"
                                        value={row.left}
                                        onChange={(e) => {
                                          const nextLeft = e.target.value as ActiveTeamCode
                                          setEventSetRowsDraft((prev) =>
                                            prev.map((r, j) => {
                                              if (j !== idx) return r
                                              const nextRight = nextLeft === r.right ? ensureOtherTeam(nextLeft) : r.right
                                              return { ...r, left: nextLeft, right: nextRight }
                                            }),
                                          )
                                        }}
                                      >
                                        {availableTeams.map((t) => (
                                          <option key={`left-${idx}-${t}`} value={t}>
                                            {t}
                                          </option>
                                        ))}
                                      </select>
                                    ) : (
                                      <strong className="set-team">{row.left}</strong>
                                    )}

                                    <div className="set-score">
                                      <input
                                        type="number"
                                        min="0"
                                        max="99"
                                        value={row.leftScore}
                                        onChange={(e) => {
                                          const value = Number(e.target.value)
                                          setEventSetRowsDraft((prev) => prev.map((r, j) => (j === idx ? { ...r, leftScore: value } : r)))
                                        }}
                                      />
                                      <span>:</span>
                                      <input
                                        type="number"
                                        min="0"
                                        max="99"
                                        value={row.rightScore}
                                        onChange={(e) => {
                                          const value = Number(e.target.value)
                                          setEventSetRowsDraft((prev) => prev.map((r, j) => (j === idx ? { ...r, rightScore: value } : r)))
                                        }}
                                      />
                                    </div>

                                    {availableTeams.length > 2 ? (
                                      <select
                                        className="set-team"
                                        value={row.right}
                                        onChange={(e) => {
                                          const nextRight = e.target.value as ActiveTeamCode
                                          setEventSetRowsDraft((prev) =>
                                            prev.map((r, j) => {
                                              if (j !== idx) return r
                                              const nextLeft = nextRight === r.left ? ensureOtherTeam(nextRight) : r.left
                                              return { ...r, left: nextLeft, right: nextRight }
                                            }),
                                          )
                                        }}
                                      >
                                        {availableTeams.map((t) => (
                                          <option key={`right-${idx}-${t}`} value={t}>
                                            {t}
                                          </option>
                                        ))}
                                      </select>
                                    ) : (
                                      <strong className="set-team">{row.right}</strong>
                                    )}
                                  </div>
                                  <button
                                    type="button"
                                    className="icon-btn danger set-del-btn"
                                    title="Удалить партию"
                                    disabled={(eventSetRowsDraft.length || 1) <= 1}
                                    onClick={() => setEventSetRowsDraft((prev) => prev.filter((_, j) => j !== idx))}
                                  >
                                    −
                                  </button>
                                </div>
                              ),
                            )}

                            <div className="set-add-row">
                              <button
                                type="button"
                                className="set-add-btn"
                                aria-label="Добавить партию"
                                title="Добавить партию"
                                onClick={() =>
                                  setEventSetRowsDraft((prev) => [
                                    ...prev,
                                    {
                                      left: prev[prev.length - 1]?.left ?? availableTeams[0],
                                      right: prev[prev.length - 1]?.right ?? availableTeams[1],
                                      leftScore: 0,
                                      rightScore: 0,
                                    },
                                  ])
                                }
                              >
                                +
                              </button>
                            </div>
                          </div>

                          <div className="manual-controls">
                            {availableTeams.length > 2 ? (
                              <button
                                type="button"
                                className="btn-secondary"
                                onClick={() => setEventSetRowsDraft([])}
                              >
                                Сбросить
                              </button>
                            ) : null}
                            <button type="button" onClick={() => void onSaveEventSets()}>
                              Сохранить
                            </button>
                            <button type="button" className="btn-secondary" onClick={() => void onPublishEventSets()}>
                              Опубликовать
                            </button>
                          </div>
                        </>
                      )
                    })()}
                  </>
                ) : null}
              </section>
            ) : null}

            {historyDetailTab === 'billing' ? (
              <section className="content-card">
                <div className="template-head">
                  <h4>Оплата события</h4>
                </div>
                {eventBillingLoading ? <p className="muted">Загружаю оплаты...</p> : null}
                {eventBillingError ? <p className="muted">Ошибка: {eventBillingError}</p> : null}
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
                        <span>Долг по событию</span>
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
                              <td>{formatMoney(player.amountDue)}</td>
                              <td className="col-center">
                                <label className="toggle-field toggle-field-only">
                                  <input
                                    type="checkbox"
                                    checked={Boolean(eventBillingDraft[player.userID])}
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
                        onClick={() => void onPublishSettlementForEvent(selectedHistoryEvent.eventID)}
                        disabled={
                          (selectedHistoryEvent.pollTemplate || '').trim() === '' ||
                          (templateCountedMap.get(selectedHistoryEvent.pollTemplate || '') ?? 0) === 0
                        }
                      >
                        Опубликовать расчёт
                      </button>
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
              onClick={() =>
                navigateTo({
                  chatID: activeChatID,
                  section: 'polls',
                  pollPostID: null,
                  templateView: 'list',
                  templateName: null,
                  eventView: 'list',
                  eventID: null,
                })
              }
            >
              ← К списку голосований
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
                <strong>{selectedGroupPoll.status || '-'}</strong>
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
          {groupPollVotesError ? <p className="muted">Ошибка: {groupPollVotesError}</p> : null}
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
                    <tr key={vote.userID}>
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
        </section>
      )
    }

    return (
      <section className="content-card">
        <div className="template-head">
          <h3>Голосования</h3>
        </div>
        {groupPollsLoading ? <p className="muted">Загрузка голосований...</p> : null}
        {groupPollsError ? <p className="muted">Ошибка: {groupPollsError}</p> : null}
        {!groupPollsLoading && !groupPollsError && groupPolls.length === 0 ? <p className="muted">Голосований пока нет</p> : null}
        {!groupPollsLoading && groupPolls.length > 0 ? (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Событие</th>
                  <th>Шаблон</th>
                  <th>Вопрос</th>
                  <th>Статус</th>
                  <th>Учет/Всего</th>
                  <th>Опубликовано</th>
                </tr>
              </thead>
              <tbody>
                {groupPolls.map((poll) => (
                  <tr
                    key={poll.postID}
                    className="member-row"
                    onClick={() =>
                      navigateTo({
                        chatID: activeChatID,
                        section: 'polls',
                        pollPostID: poll.postID,
                        templateView: 'list',
                        templateName: null,
                        eventView: 'list',
                        eventID: null,
                      })
                    }
                  >
                    <td>#{poll.postID}</td>
                    <td>
                      {poll.eventName || 'Без события'}
                      {poll.instanceID ? ` · instance #${poll.instanceID}` : ''}
                    </td>
                    <td>{poll.templateName || '-'}</td>
                    <td>{poll.question || '-'}</td>
                    <td>{poll.status || '-'}</td>
                    <td>
                      {poll.countedVotes} / {poll.totalVotes}
                    </td>
                    <td>{formatDateTime(poll.publishedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
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
        {groupGamesError ? <p className="muted">Ошибка: {groupGamesError}</p> : null}
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
                          <span className="game-score">{`${clampInt(row.score1, 0, 99)}:${clampInt(row.score2, 0, 99)}`}</span>
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
    if (!loading && groups.length === 0) {
      return (
        <section className="content-card">
          <h3>Организации не найдены</h3>
          <p className="muted">Добавь бота в группу Telegram и сделай первичную настройку, после этого группа появится здесь.</p>
        </section>
      )
    }
    if (!details) {
      return <section className="content-card">Выбери организацию</section>
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
      case 'billing':
        if (!isAdmin) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Раздел «Задолженности» доступен только администраторам.</p>
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
      case 'profile':
        if (!can('profile_read')) {
          return (
            <section className="content-card">
              <h3>Недостаточно прав</h3>
              <p className="muted">Раздел «Профиль» недоступен.</p>
            </section>
          )
        }
        return renderProfile()
      case 'docs':
        return renderDocs()
      default:
        return null
    }
  }

  function renderProfile() {
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
          <p className="muted">Ошибка: {myProfileError}</p>
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
        <div className="template-head">
          <h3>Профиль</h3>
        </div>
        <div className="form-grid form-grid-3 members-filters">
          <label className="field">
            <span>Роль</span>
            <input value={myProfile.roleTitle || myProfile.roleCode} readOnly />
          </label>
          <label className="field">
            <span>Задолженность</span>
            <input value={formatMoney(myProfile.debtAmount)} readOnly />
          </label>
          <label className="field">
            <span>Тренировок</span>
            <input value={`${myProfile.trainings.length} шт.`} readOnly />
          </label>
        </div>

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
                <select className="ui-select" value={roleAssignUserID} onChange={(e) => setRoleAssignUserID(e.target.value)}>
                  <option value="">Выбери участника</option>
                  {members.map((m) => (
                    <option key={m.userTelegramID} value={m.userTelegramID}>
                      {`${fullName(m)}${m.username ? ` (@${m.username})` : ''} · ${m.role}`}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field">
                <span>Роль</span>
                <select className="ui-select" value={roleAssignCode} onChange={(e) => setRoleAssignCode(e.target.value)}>
                  <option value="member">Участник</option>
                  <option value="captain">Капитан</option>
                  <option value="trainer">Тренер</option>
                </select>
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
            {groupRolesError ? <p className="muted">Ошибка: {groupRolesError}</p> : null}
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

  return (
    <div className={mobileNavOpen ? 'console-shell nav-open' : 'console-shell'}>
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
          <strong className="mobile-topbar-title">{details?.group.title || 'TeamTime Console'}</strong>
          <span className="mobile-topbar-subtitle">{sections.find((s) => s.id === activeSection)?.title || 'Навигация'}</span>
        </div>
        <button
          type="button"
          className="btn-icon mobile-refresh-btn"
          aria-label="Обновить"
          disabled={!activeChatID}
          onClick={() => (activeChatID ? void reloadActiveOrganization(activeChatID) : undefined)}
        >
          <span aria-hidden="true">⟳</span>
        </button>
      </div>
      <div className="sidebar-overlay" role="presentation" onClick={() => setMobileNavOpen(false)} />
      <aside className="console-sidebar">
        <div className="brand-block">
          <div className="brand-mark">TT</div>
          <div>
            <p className="brand-title">TeamTime Console</p>
            <p className="brand-subtitle">Управление организацией</p>
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

        <section className="org-selector">
          <p className="section-caption">Организация</p>
          <select
            className="ui-select"
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
          </select>
        </section>

        <nav className="nav-menu" aria-label="Справочники">
          <p className="section-caption">Справочники</p>
          {visibleSections.map((section) => (
            <button
              key={section.id}
              className={activeSection === section.id ? 'nav-item active' : 'nav-item'}
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
              <span className="nav-icon">{section.icon}</span>
              <span>
                <strong>{section.title}</strong>
                <small>{section.subtitle}</small>
              </span>
            </button>
          ))}
        </nav>

        {authConfig?.enabled && authUser ? (
          <button
            className="refresh-btn"
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
            Выйти
          </button>
        ) : null}
      </aside>

      <main className="console-main">
        <header className="console-header">
          <div>
            <h1>{details?.group.title || 'TeamTime'}</h1>
            <p>
              {details ? `chat_id: ${details.group.chatID} • timezone: ${details.group.timezone}` : 'Выбери организацию в левом меню'}
            </p>
          </div>
        </header>

        {loading ? <section className="content-card">Загрузка...</section> : null}
        {error ? <section className="content-card alert error">{error}</section> : null}
        {success ? <section className={`toast toast-success ${successVisible ? 'show' : 'hide'}`}>{success}</section> : null}

        {authLoading ? (
          <section className="content-card">Проверка авторизации...</section>
        ) : authConfig?.enabled && !authUser ? (
          <section className="content-card">
            <h3>Вход через Telegram</h3>
            <p className="muted">Авторизуйся через Telegram, чтобы видеть только свои админские группы.</p>
            <div id="telegram-login-widget" />
          </section>
        ) : (
          renderContent()
        )}
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

import { FormEvent, useEffect, useMemo, useState } from 'react'
import {
  activateEventPublications,
  archiveEvent,
  bindEvent,
  createEvent,
  createEventInstance,
  createTemplate,
  deactivateEventPublications,
  deleteTemplate,
  fetchAuthConfig,
  fetchAuthMe,
  fetchArchivedEvents,
  fetchEventActivity,
  fetchEventBillingForInstance,
  fetchEventHistory,
  fetchGroupPolls,
  fetchGroupPollVotes,
  publishRegistration,
  generateEventBillingForInstance,
  fetchEventPollHistoryForInstance,
  fetchEventTeamSplit,
  fetchGroupDebtSummary,
  fetchGroupDetails,
  fetchGroupMembers,
  fetchGroups,
  fetchMemberSkills,
  fetchMemberRelations,
  fetchSkillsCatalog,
  fetchTemplate,
  autoSplitEventTeams,
  publishEventSettlement,
  publishEventTeamSplit,
  saveEventTeamSplit,
  saveEventBillingForInstance,
  telegramAuthLogin,
  unarchiveEvent,
  updateMemberProfile,
  updateMemberSkills,
  upsertMemberRelation,
  deleteMemberRelation,
  updateEventCost,
  updateEventDetails,
  updateTemplate,
  logoutAuth,
} from './api'
import { announcementLeadLabel, announcementLeadOptions, ensureList, formatMoney, toHourMinute, weekdayLabel, weekdayOptions } from './app/constants'
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
  Group,
  GroupDebtSummary,
  GroupDetails,
  GroupPollItem,
  GroupPollVoteItem,
  GroupMember,
  MemberSkillProfile,
  PlayerRelation,
  SkillCatalogItem,
  AuthConfig,
  AuthUser,
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

  const [details, setDetails] = useState<GroupDetails | null>(null)
  const [members, setMembers] = useState<GroupMember[]>([])
  const [membersError, setMembersError] = useState('')
  const [skillsCatalog, setSkillsCatalog] = useState<SkillCatalogItem[]>([])
  const [memberSkillProfiles, setMemberSkillProfiles] = useState<Record<number, MemberSkillProfile>>({})
  const [selectedMemberSkills, setSelectedMemberSkills] = useState<MemberSkillProfile | null>(null)
  const [memberSkillDraft, setMemberSkillDraft] = useState<Record<string, string>>({})
  const [memberPlayerTypeDraft, setMemberPlayerTypeDraft] = useState<'' | 'attacker' | 'setter' | 'libero'>('')
  const [memberRelations, setMemberRelations] = useState<PlayerRelation[]>([])
  const [memberRelationDraft, setMemberRelationDraft] = useState({
    otherUserID: '',
    relationType: 'prefer_together' as 'prefer_together' | 'avoid_together',
    weight: '5',
  })
  const [membersSearch, setMembersSearch] = useState('')
  const [membersTypeFilter, setMembersTypeFilter] = useState<'' | 'attacker' | 'setter' | 'libero'>('')
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
  })
  const [eventForm, setEventForm] = useState({
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
  })

  const [templateEditor, setTemplateEditor] = useState<TemplateFormState>({
    name: '',
    question: '',
    options: ['', ''],
    counted: [false, false],
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
  const [historyDetailTab, setHistoryDetailTab] = useState<'distribution' | 'votes' | 'billing'>('distribution')
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
  const [teamCEnabled, setTeamCEnabled] = useState(false)
  const [eventBilling, setEventBilling] = useState<EventBilling | null>(null)
  const [eventBillingLoading, setEventBillingLoading] = useState(false)
  const [eventBillingError, setEventBillingError] = useState('')
  const [eventBillingDraft, setEventBillingDraft] = useState<Record<number, boolean>>({})
  const [groupDebtSummary, setGroupDebtSummary] = useState<GroupDebtSummary | null>(null)

  const templateNames = useMemo(() => ensureList(details?.templateNames), [details])
  const templates = useMemo(() => ensureList(details?.templates), [details])
  const events = useMemo(() => ensureList(details?.events), [details])
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
    return members.filter((member) => {
      const profile = memberSkillProfiles[member.userTelegramID]
      const playerType = (profile?.playerType || member.playerType || '') as '' | 'attacker' | 'setter' | 'libero'
      if (membersTypeFilter !== '' && playerType !== membersTypeFilter) {
        return false
      }
      if (query === '') {
        return true
      }
      const name = fullName(member).toLowerCase()
      const username = (member.username || '').toLowerCase()
      const id = String(member.userTelegramID)
      return name.includes(query) || username.includes(query) || id.includes(query)
    })
  }, [memberSkillProfiles, members, membersSearch, membersTypeFilter])
  const selectedHistoryEvent = useMemo(
    () => eventHistory.find((event) => event.instanceID === activeHistoryEventID) ?? null,
    [eventHistory, activeHistoryEventID],
  )
  const availableRelationMembers = useMemo(() => {
    if (!selectedMemberSkills) {
      return []
    }
    return members
      .filter((member) => member.userTelegramID !== selectedMemberSkills.userTelegramID)
      .sort((a, b) => fullName(a).localeCompare(fullName(b), 'ru'))
  }, [members, selectedMemberSkills])
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
  }, [authConfig, authUser])

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
      setActiveHistoryEventID(null)
      setActivePollPostID(null)
      setArchivedEvents([])
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
      setActiveHistoryEventID(null)
      setActivePollPostID(null)
      setArchivedEvents([])
      return
    }
    void reloadActiveOrganization(activeChatID)
  }, [activeChatID, groups])

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
      setTemplateEditor({ name: '', question: '', options: ['', ''], counted: [false, false] })
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
        setTemplateEditor({
          name: template.name,
          question: template.question,
          options: preparedOptions,
          counted: preparedOptions.map((_, idx) => template.countedOptions.includes(idx)),
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
    const payload = buildTemplatePayload(templateForm.options, templateForm.counted)
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
        }),
      'Шаблон сохранен',
    )
    setTemplateForm({ name: '', question: '', options: ['', ''], counted: [false, false] })
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

  function addTemplateFormOption() {
    setTemplateForm((prev) => ({ ...prev, options: [...prev.options, ''], counted: [...prev.counted, false] }))
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

  function addTemplateEditorOption() {
    setTemplateEditor((prev) => ({ ...prev, options: [...prev.options, ''], counted: [...prev.counted, false] }))
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
    const payload = buildTemplatePayload(templateEditor.options, templateEditor.counted)
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
    setError('')
    setSuccess('')

    let createdEventID: number | null = null
    try {
      const created = await createEvent(activeChatID, {
        name: eventForm.name.trim(),
        eventType,
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
      setSuccess('Распределение команд сохранено')
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
        setMemberSkillError(`Оценка для "${skill.skillName}" должна быть целым числом 1..10`)
        return
      }
      payload[skill.skillCode] = value
    }
    setMemberSkillError('')
    try {
      await updateMemberProfile(activeChatID, selectedMemberSkills.userTelegramID, { playerType: memberPlayerTypeDraft })
      await updateMemberSkills(activeChatID, selectedMemberSkills.userTelegramID, payload)
      const refreshed = await fetchMemberSkills(activeChatID, selectedMemberSkills.userTelegramID)
      const nextDraft: Record<string, string> = {}
      for (const skill of profileSkills(refreshed, skillsCatalog)) {
        nextDraft[skill.skillCode] = String(normalizedSkillScore(skill.score))
      }
      setMemberSkillDraft(nextDraft)
      setMemberPlayerTypeDraft(refreshed.playerType || '')
      setSelectedMemberSkills(refreshed)
      setMemberSkillProfiles((prev) => ({ ...prev, [refreshed.userTelegramID]: refreshed }))
      setSuccess('Профиль игрока сохранен')
    } catch (err) {
      setMemberSkillError((err as Error).message)
    }
  }

  function relationTypeLabel(type: 'prefer_together' | 'avoid_together') {
    return type === 'prefer_together' ? 'Играть вместе' : 'Не в одну команду'
  }

  function relationUserName(relation: PlayerRelation) {
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
                <div className="form-grid form-grid-3">
                  <label className="field">
                    <span>Игрок</span>
                    <select
                      className="ui-select"
                      value={memberRelationDraft.otherUserID}
                      onChange={(e) => setMemberRelationDraft((prev) => ({ ...prev, otherUserID: e.target.value }))}
                    >
                      <option value="">Выбери игрока</option>
                      {availableRelationMembers.map((member) => (
                        <option key={member.userTelegramID} value={member.userTelegramID}>
                          {fullName(member)}
                        </option>
                      ))}
                    </select>
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
              placeholder="Имя, username или ID"
              value={membersSearch}
              onChange={(e) => setMembersSearch(e.target.value)}
            />
          </label>
          <label className="field">
            <span>Тип игрока</span>
            <select
              value={membersTypeFilter}
              onChange={(e) => setMembersTypeFilter(e.target.value as '' | 'attacker' | 'setter' | 'libero')}
            >
              <option value="">Все типы</option>
              <option value="attacker">Атакующий</option>
              <option value="setter">Пасующий</option>
              <option value="libero">Либеро</option>
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
                  <th>Тип игрока</th>
                  <th>Характеристики</th>
                  <th>Средняя</th>
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
                      <span className="skill-chip">
                        {playerTypeLabel((memberSkillProfiles[member.userTelegramID]?.playerType || member.playerType || '') as '' | 'attacker' | 'setter' | 'libero')}
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
          <h3>Шаблоны опросов</h3>
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
              <div className="list-row" key={template.name}>
                <div>
                  <strong>{template.name}</strong>
                  <p>{template.question}</p>
                  <p className="muted">Учёт вариантов: {template.countedOptionsCount}</p>
                </div>
                <div className="list-actions">
                  <button
                    className="btn-secondary"
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
                  >
                    Открыть
                  </button>
                  {template.name.trim().toLowerCase() === 'регистрация' ? null : (
                    <button className="btn-danger" onClick={() => onDeleteTemplate(template.name)}>
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
            <span />
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
                </p>
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
                      const renderTeamColumn = (teamCode: ActiveTeamCode) => (
                            <div
                              className={`team-column ${draggedPlayerID !== null ? 'team-column-drop' : ''}`}
                              data-team={teamCode}
                          onDragOver={(e) => {
                            e.preventDefault()
                          }}
                          onDrop={(e) => {
                            e.preventDefault()
                            onDropToTeam(teamCode)
                          }}
                          >
                            <div className="team-column-head">
                              <h5>{`Команда ${teamCode}`}</h5>
                              {teamCode === 'C' ? (
                                <button type="button" className="team-remove-btn" onClick={removeTeamC} title="Убрать команду C">
                                  −
                                </button>
                              ) : null}
                            </div>
                            <div className="team-players">
                            {teamSplit.players
                              .filter((player) => player.team === teamCode)
                              .map((player) => (
                                <article
                                  className="team-player-card"
                                  key={`${teamCode}-${player.userID}`}
                                  draggable
                                  onDragStart={() => onDragStartTeamPlayer(player.userID)}
                                  onDragEnd={() => setDraggedPlayerID(null)}
                                >
                                  <p>{playerDisplayName(player)}</p>
                                  <small>{player.choiceLabel} · рейтинг {player.rating.toFixed(1)}</small>
                                </article>
                              ))}
                            {teamSplit.players.filter((player) => player.team === teamCode).length === 0 ? <p className="muted">Пусто</p> : null}
                          </div>
                        </div>
                      )

                      return (
                        <>
                          <div className="team-layout">
                            <div
                              className={`team-column team-column-unassigned ${draggedPlayerID !== null ? 'team-column-drop' : ''}`}
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
                                {teamSplit.players
                                  .filter((player) => player.team === 'unassigned')
                                  .map((player) => (
                                    <article
                                      className="team-player-card"
                                      key={`unassigned-${player.userID}`}
                                      draggable
                                      onDragStart={() => onDragStartTeamPlayer(player.userID)}
                                      onDragEnd={() => setDraggedPlayerID(null)}
                                    >
                                      <p>{playerDisplayName(player)}</p>
                                      <small>{player.choiceLabel} · рейтинг {player.rating.toFixed(1)}</small>
                                    </article>
                                  ))}
                                {teamSplit.players.filter((player) => player.team === 'unassigned').length === 0 ? <p className="muted">Пусто</p> : null}
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
                        </tr>
                      </thead>
                      <tbody>
                        {historyPollVotes.map((vote) => (
                          <tr key={vote.userID}>
                            <td>
                              {`${vote.firstName || ''} ${vote.lastName || ''}`.trim() ||
                                (vote.username ? `@${vote.username}` : `ID ${vote.userID}`)}
                            </td>
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
                              <td>{`${player.firstName ?? ''} ${player.lastName ?? ''}`.trim() || (player.username ? `@${player.username}` : `ID ${player.userID}`)}</td>
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
                      <td>
                        {`${vote.firstName || ''} ${vote.lastName || ''}`.trim() || (vote.username ? `@${vote.username}` : `ID ${vote.userID}`)}
                      </td>
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
        return renderOverview()
      case 'members':
        return renderMembers()
      case 'templates':
        return renderTemplates()
      case 'polls':
        return renderPolls()
      case 'events':
        return renderHistory()
      case 'event_templates':
        return renderEvents()
      default:
        return null
    }
  }

  return (
    <div className="console-shell">
      <aside className="console-sidebar">
        <div className="brand-block">
          <div className="brand-mark">TT</div>
          <div>
            <p className="brand-title">TeamTime Console</p>
            <p className="brand-subtitle">Управление организацией</p>
          </div>
        </div>

        <section className="org-selector">
          <p className="section-caption">Организация</p>
          <select
            className="ui-select"
            value={activeChatID ?? ''}
            onChange={(e) => {
              const value = e.target.value
              const nextChatID = value ? Number(value) : null
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
          {sections.map((section) => (
            <button
              key={section.id}
              className={activeSection === section.id ? 'nav-item active' : 'nav-item'}
              onClick={() =>
                navigateTo({
                  chatID: activeChatID,
                  section: section.id,
                  templateView: 'list',
                  templateName: null,
                  eventView: 'list',
                  eventID: null,
                })
              }
            >
              <span className="nav-icon">{section.icon}</span>
              <span>
                <strong>{section.title}</strong>
                <small>{section.subtitle}</small>
              </span>
            </button>
          ))}
        </nav>

        <button className="refresh-btn" onClick={() => activeChatID !== null && reloadActiveOrganization(activeChatID)}>
          Обновить данные
        </button>
        {authConfig?.enabled && authUser ? (
          <button
            className="refresh-btn"
            onClick={() =>
              void (async () => {
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
    </div>
  )
}

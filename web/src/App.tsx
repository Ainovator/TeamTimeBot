import { FormEvent, useEffect, useMemo, useState } from 'react'
import {
  activateEventPublications,
  archiveEvent,
  bindEvent,
  createEvent,
  createTemplate,
  deactivateEventPublications,
  deleteTemplate,
  fetchArchivedEvents,
  fetchEventActivity,
  fetchEventHistory,
  fetchEventPollHistory,
  fetchEventTeamSplit,
  fetchGroupDetails,
  fetchGroupMembers,
  fetchGroups,
  fetchMemberSkills,
  fetchSkillsCatalog,
  fetchTemplate,
  publishEventAnnouncement,
  publishEventPoll,
  publishEventSettlement,
  saveEventTeamSplit,
  unarchiveEvent,
  updateMemberProfile,
  updateMemberSkills,
  updateEventCost,
  updateEventDetails,
  updateTemplate,
} from './api'
import { announcementLeadLabel, announcementLeadOptions, ensureList, formatMoney, toHourMinute, weekdayLabel, weekdayOptions } from './app/constants'
import { sections } from './app/navigation'
import { buildRoutePath, parseRoute, type RouteState, type Section } from './app/router'
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
  EventPollHistoryItem,
  EventTeamSplitState,
  EventView,
  Group,
  GroupDetails,
  GroupMember,
  MemberSkillProfile,
  SkillCatalogItem,
} from './types'

const initialRoute = parseRoute(window.location.pathname)
export default function App() {
  const [groups, setGroups] = useState<Group[]>([])
  const [activeChatID, setActiveChatID] = useState<number | null>(initialRoute.chatID)
  const [activeSection, setActiveSection] = useState<Section>(initialRoute.section)
  const [activeMemberID, setActiveMemberID] = useState<number | null>(initialRoute.memberID ?? null)
  const [activeHistoryEventID, setActiveHistoryEventID] = useState<number | null>(initialRoute.historyEventID ?? null)
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
  const [historyStatusFilter, setHistoryStatusFilter] = useState<'' | 'in_voting' | 'on_distribution' | 'completed'>('')
  const [eventHistoryLoading, setEventHistoryLoading] = useState(false)
  const [eventHistoryError, setEventHistoryError] = useState('')
  const [eventPollHistory, setEventPollHistory] = useState<EventPollHistoryItem[]>([])
  const [eventPollHistoryLoading, setEventPollHistoryLoading] = useState(false)
  const [eventPollHistoryError, setEventPollHistoryError] = useState('')
  const [selectedHistoryPostID, setSelectedHistoryPostID] = useState<number | null>(null)
  const [teamSplit, setTeamSplit] = useState<EventTeamSplitState | null>(null)
  const [teamSplitLoading, setTeamSplitLoading] = useState(false)
  const [teamSplitError, setTeamSplitError] = useState('')
  const [draggedPlayerID, setDraggedPlayerID] = useState<number | null>(null)
  const [teamCEnabled, setTeamCEnabled] = useState(false)

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
    () => eventHistory.find((event) => event.eventID === activeHistoryEventID) ?? null,
    [eventHistory, activeHistoryEventID],
  )
  const eventEditorDirty = useMemo(() => {
    if (!selectedEvent || activeSection !== 'events' || activeEventView !== 'edit') {
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
    setActiveChatID(route.chatID)
    setActiveSection(route.section)
    setActiveMemberID(route.section === 'members' ? route.memberID ?? null : null)
    setActiveHistoryEventID(route.section === 'history' ? route.historyEventID ?? null : null)
    setActiveTemplateView(route.section === 'templates' ? route.templateView : 'list')
    setActiveTemplateName(route.section === 'templates' ? route.templateName : null)
    setActiveEventView(route.section === 'events' ? route.eventView : 'list')
    setActiveEventID(route.section === 'events' ? route.eventID : null)

    const nextPath = buildRoutePath(route)
    const stateOp = replace ? window.history.replaceState : window.history.pushState
    stateOp.call(window.history, {}, '', nextPath)
  }

  useEffect(() => {
    const onPopState = () => {
      const route = parseRoute(window.location.pathname)
      setActiveChatID(route.chatID)
      setActiveSection(route.section)
      setActiveMemberID(route.section === 'members' ? route.memberID ?? null : null)
      setActiveHistoryEventID(route.section === 'history' ? route.historyEventID ?? null : null)
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
      try {
        const loadedGroups = await fetchGroups()
        setGroups(loadedGroups)

        if (loadedGroups.length === 0) {
          return
        }

        const hasActive = activeChatID !== null && loadedGroups.some((group) => group.chatID === activeChatID)
        if (!hasActive) {
          navigateTo(
            {
              chatID: loadedGroups[0].chatID,
              section: 'overview',
              templateView: 'list',
              templateName: null,
              eventView: 'list',
              eventID: null,
            },
            true,
          )
        }
      } catch (err) {
        setError((err as Error).message)
      }
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (activeChatID === null) {
      setDetails(null)
      setMembers([])
      setMemberSkillProfiles({})
      setSelectedMemberSkills(null)
      setEventHistory([])
      setActiveHistoryEventID(null)
      setArchivedEvents([])
      return
    }
    if (!groups.some((group) => group.chatID === activeChatID)) {
      setDetails(null)
      setMembers([])
      setMemberSkillProfiles({})
      setSelectedMemberSkills(null)
      setEventHistory([])
      setActiveHistoryEventID(null)
      setArchivedEvents([])
      return
    }
    void reloadActiveOrganization(activeChatID)
  }, [activeChatID, groups])

  useEffect(() => {
    if (activeSection !== 'events') {
      setEventsMode('active')
    }
  }, [activeSection])

  useEffect(() => {
    if (activeSection !== 'history' || activeChatID === null) {
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
        if (activeHistoryEventID !== null && !items.some((item) => item.eventID === activeHistoryEventID)) {
          setActiveHistoryEventID(items[0].eventID)
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
    if (activeSection !== 'events' || activeEventView !== 'edit' || !selectedEvent) {
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
    if (activeSection !== 'events' || activeEventView !== 'edit') {
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
        section: 'events',
        templateView: 'list',
        templateName: null,
        eventView: 'list',
        eventID: null,
      },
      true,
    )
  }, [activeSection, activeEventView, activeChatID, activeEventID, details, selectedEvent])

  useEffect(() => {
    if (activeSection !== 'history' || activeChatID === null || activeHistoryEventID === null) {
      return
    }
    let cancelled = false
    setEventPollHistoryLoading(true)
    setEventPollHistoryError('')
    void fetchEventPollHistory(activeChatID, activeHistoryEventID)
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
    if (activeSection !== 'history' || activeChatID === null || activeHistoryEventID === null || selectedHistoryPostID === null) {
      return
    }
    let cancelled = false
    setTeamSplitLoading(true)
    setTeamSplitError('')
    void fetchEventTeamSplit(activeChatID, activeHistoryEventID, selectedHistoryPostID)
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
  }, [activeSection, activeChatID, activeHistoryEventID, selectedHistoryPostID])

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
      settlementEnabled: false,
      settlementPublishBefore: false,
      settlementPublishAfter: true,
      costAmount: '',
    })

    if (createdEventID !== null) {
      navigateTo(
        {
          chatID: activeChatID,
          section: 'events',
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

  async function onManualPublishAnnouncement() {
    if (activeChatID === null || activeEventID === null) {
      return
    }
    await runAction(() => publishEventAnnouncement(activeChatID, activeEventID), 'Анонс опубликован')
  }

  async function onManualPublishPoll() {
    if (activeChatID === null || activeEventID === null) {
      return
    }
    await runAction(() => publishEventPoll(activeChatID, activeEventID), 'Опрос опубликован')
  }

  async function onManualPublishSettlement() {
    if (activeChatID === null || activeEventID === null) {
      return
    }
    await runAction(() => publishEventSettlement(activeChatID, activeEventID), 'Расчёт опубликован')
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
          section: 'events',
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
    if (activeChatID === null || activeHistoryEventID === null || selectedHistoryPostID === null || !teamSplit) {
      return
    }
    try {
      const saved = await saveEventTeamSplit(
        activeChatID,
        activeHistoryEventID,
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

  async function loadMemberProfile(userTelegramID: number) {
    if (activeChatID === null) {
      return
    }
    setMemberSkillLoading(true)
    setMemberSkillError('')
    setSelectedMemberSkills(null)
    try {
      const profile = await fetchMemberSkills(activeChatID, userTelegramID)
      const draft: Record<string, string> = {}
      for (const skill of profileSkills(profile, skillsCatalog)) {
        draft[skill.skillCode] = String(normalizedSkillScore(skill.score))
      }
      setMemberSkillDraft(draft)
      setMemberPlayerTypeDraft(profile.playerType || '')
      setSelectedMemberSkills(profile)
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
              <button type="button" className="btn-danger" onClick={() => void onDeleteTemplateEditor()}>
                Удалить шаблон
              </button>
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
                  <button className="btn-danger" onClick={() => onDeleteTemplate(template.name)}>
                    Удалить
                  </button>
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
                section: 'events',
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

        <form className="form-grid form-grid-3" onSubmit={onCreateEvent}>
          <h4>Название</h4>
          <input
            placeholder="Название события"
            value={eventForm.name}
            onChange={(e) => setEventForm((prev) => ({ ...prev, name: e.target.value }))}
            required
          />
          <h4>Тип события</h4>
          <select className="ui-select" value={eventForm.eventType} onChange={(e) => setEventForm((prev) => ({ ...prev, eventType: e.target.value as 'training' | 'activity' }))}>
            <option value="training">Тренировка</option>
            <option value="activity">Мероприятие</option>
          </select>
          <h4>День недели</h4>
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
          <h4>День публикации опроса</h4>
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
          <h4>Публикация опроса</h4>
          <input
            type="time"
            value={eventForm.publishAt}
            onChange={(e) => setEventForm((prev) => ({ ...prev, publishAt: e.target.value }))}
            required
          />
          <h4>Начало</h4>
          <input
            type="time"
            value={eventForm.startAt}
            onChange={(e) => setEventForm((prev) => ({ ...prev, startAt: e.target.value }))}
            required
          />
          <h4>Конец</h4>
          <input
            type="time"
            value={eventForm.endAt}
            onChange={(e) => setEventForm((prev) => ({ ...prev, endAt: e.target.value }))}
            required
          />
          <h4>Стоимость (опционально)</h4>
          <input
            type="number"
            min="0"
            step="0.01"
            placeholder="Например: 4000"
            value={eventForm.costAmount}
            onChange={(e) => setEventForm((prev) => ({ ...prev, costAmount: e.target.value }))}
          />
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
          {eventForm.eventType === 'training' ? (
            <section className="settings-card">
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
                </div>
              </div>
            </section>
          ) : null}
          <section className="settings-card">
            <h4>Настройки</h4>
            <div className="settings-grid">
              <div className="settings-group">
                <p className="settings-group-title">Анонс</p>
                <div className="settings-row settings-row-2">
                  <label className="field toggle-field">
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
                <div className="settings-row settings-row-3">
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventForm.settlementEnabled}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, settlementEnabled: e.target.checked }))}
                      disabled
                    />
                    <span>Публиковать расчёт</span>
                  </label>
                  <label className="field toggle-field">
                    <input
                      type="checkbox"
                      checked={eventForm.settlementPublishBefore}
                      onChange={(e) => setEventForm((prev) => ({ ...prev, settlementPublishBefore: e.target.checked }))}
                    />
                    <span>Перед началом</span>
                  </label>
                  <label className="field toggle-field">
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
                </div>
              </div>
            </div>
          </section>
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
                  section: 'events',
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
                section: 'events',
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

          {eventEditor.eventType === 'training' ? (
            <section className="settings-card">
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
                </div>
              </div>
            </section>
          ) : null}

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

          <section className="settings-card">
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
                <div className="settings-row settings-row-2">
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
                </div>
              </div>
            </div>
          </section>

          <section className="settings-card">
            <h4>Ручное управление</h4>
            <div className="manual-controls">
              <button
                type="button"
                className="btn-secondary"
                onClick={() => void onManualPublishAnnouncement()}
                disabled={(selectedEvent.announcementText || '').trim() === ''}
              >
                Опубликовать анонс
              </button>
              <button
                type="button"
                className="btn-secondary"
                onClick={() => void onManualPublishPoll()}
                disabled={(selectedEvent.pollTemplate || '').trim() === ''}
              >
                Опубликовать опрос
              </button>
              <button
                type="button"
                className="btn-secondary"
                onClick={() => void onManualPublishSettlement()}
                disabled={
                  (selectedEvent.pollTemplate || '').trim() === '' ||
                  (templateCountedMap.get(selectedEvent.pollTemplate || '') ?? 0) === 0
                }
              >
                Опубликовать расчёт
              </button>
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
                  section: 'events',
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
                                section: 'events',
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
            <h3>История событий</h3>
          </div>
          <div className="form-grid form-grid-3 members-filters">
            <label className="field">
              <span>Статус</span>
              <select
                value={historyStatusFilter}
                onChange={(e) =>
                  setHistoryStatusFilter(e.target.value as '' | 'in_voting' | 'on_distribution' | 'completed')
                }
              >
                <option value="">Все статусы</option>
                <option value="in_voting">В голосовании</option>
                <option value="on_distribution">На распределении</option>
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
                    <th>Следующий старт</th>
                    <th>Опрос</th>
                    <th>Последняя публикация</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredEventHistory.map((item) => (
                    <tr
                      key={item.eventID}
                      className="member-row"
                      onClick={() =>
                        navigateTo({
                          chatID: activeChatID,
                          section: 'history',
                          historyEventID: item.eventID,
                          templateView: 'list',
                          templateName: null,
                          eventView: 'list',
                          eventID: null,
                        })
                      }
                    >
                      <td>
                        <div className="person-cell">
                          <strong>#{item.eventID} · {item.name}</strong>
                          <span>{item.publishEnabled ? 'Публикации активны' : 'Публикации отключены'}</span>
                        </div>
                      </td>
                      <td>{item.eventType === 'training' ? 'Тренировка' : 'Мероприятие'}</td>
                      <td>{historyStatusLabel(item.status)}</td>
                      <td>{formatDateTime(item.nextStartAt)}</td>
                      <td>{item.pollTemplate || 'не привязан'}</td>
                      <td>{item.latestPollAt ? formatDateTime(item.latestPollAt) : 'нет'}</td>
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
                section: 'history',
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
                <p className="muted">Статус: {historyStatusLabel(selectedHistoryEvent.status)}</p>
              </div>
            </div>

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
                    onClick={() => setSelectedHistoryPostID(item.postID)}
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
                      <button type="button" onClick={() => void onSaveTeamSplit()}>
                        Сохранить
                      </button>
                    </div>
                  </>
                ) : null}
              </>
            ) : null}
          </section>
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
      case 'events':
        return renderEvents()
      case 'history':
        return renderHistory()
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

        {renderContent()}
      </main>
    </div>
  )
}

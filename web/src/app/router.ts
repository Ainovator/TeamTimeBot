export type Section = 'overview' | 'members' | 'templates' | 'polls' | 'event_templates' | 'events'

export type RouteState = {
  chatID: number | null
  orgKey?: string | null
  section: Section
  memberID?: number | null
  historyEventID?: number | null
  pollPostID?: number | null
  templateView: 'list' | 'create' | 'edit'
  templateName: string | null
  eventView: 'list' | 'create' | 'edit'
  eventID: number | null
}

export function makeOrgKey(title: string, chatID: number): string {
  const normalized = (title || '')
    .trim()
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, '-')
    .replace(/^-+|-+$/g, '')
  const prefix = normalized || 'org'
  return `${prefix}--${Math.abs(chatID)}`
}

export function parseRoute(pathname: string): RouteState {
  const route: RouteState = {
    chatID: null,
    orgKey: null,
    section: 'overview',
    memberID: null,
    historyEventID: null,
    pollPostID: null,
    templateView: 'list',
    templateName: null,
    eventView: 'list',
    eventID: null,
  }

  const segments = pathname.split('/').filter(Boolean)
  if (segments.length < 2 || segments[0] !== 'org') {
    return route
  }

  const orgSegment = decodeURIComponent(segments[1])
  route.orgKey = orgSegment
  const chatID = Number(orgSegment)
  if (!Number.isNaN(chatID)) {
    route.chatID = chatID
  }

  if (segments.length >= 3) {
    const section = segments[2]
    if (section === 'overview' || section === 'members' || section === 'templates' || section === 'polls' || section === 'events') {
      route.section = section
    }
    if (section === 'history') {
      route.section = 'events'
    }
    if (section === 'event-templates') {
      route.section = 'event_templates'
    }
  }

  if (route.section === 'templates') {
    if (segments.length >= 4) {
      if (segments[3] === 'new') {
        route.templateView = 'create'
      } else {
        route.templateView = 'edit'
        route.templateName = decodeURIComponent(segments[3])
      }
    } else {
      route.templateView = 'list'
    }
  }

  if (route.section === 'members' && segments.length >= 4) {
    const memberID = Number(segments[3])
    if (!Number.isNaN(memberID)) {
      route.memberID = memberID
    }
  }

  if (route.section === 'polls' && segments.length >= 4) {
    const postID = Number(segments[3])
    if (!Number.isNaN(postID)) {
      route.pollPostID = postID
    }
  }

  if (route.section === 'events' && segments.length >= 4) {
    const eventID = Number(segments[3])
    if (!Number.isNaN(eventID)) {
      route.historyEventID = eventID
    }
  }

  if (route.section === 'event_templates') {
    if (segments.length >= 4) {
      if (segments[3] === 'new') {
        route.eventView = 'create'
      } else {
        const eventID = Number(segments[3])
        if (!Number.isNaN(eventID)) {
          route.eventView = 'edit'
          route.eventID = eventID
        }
      }
    } else {
      route.eventView = 'list'
    }
  }

  return route
}

export function buildRoutePath(route: RouteState): string {
  const orgSegment = route.orgKey || (route.chatID ? String(route.chatID) : '')
  if (!orgSegment) {
    return '/'
  }

  if (route.section === 'overview') {
    return `/org/${encodeURIComponent(orgSegment)}`
  }

  if (route.section === 'templates') {
    if (route.templateView === 'create') {
      return `/org/${encodeURIComponent(orgSegment)}/templates/new`
    }
    if (route.templateView === 'edit' && route.templateName) {
      return `/org/${encodeURIComponent(orgSegment)}/templates/${encodeURIComponent(route.templateName)}`
    }
    return `/org/${encodeURIComponent(orgSegment)}/templates`
  }

  if (route.section === 'members') {
    if (route.memberID) {
      return `/org/${encodeURIComponent(orgSegment)}/members/${route.memberID}`
    }
    return `/org/${encodeURIComponent(orgSegment)}/members`
  }

  if (route.section === 'polls') {
    if (route.pollPostID) {
      return `/org/${encodeURIComponent(orgSegment)}/polls/${route.pollPostID}`
    }
    return `/org/${encodeURIComponent(orgSegment)}/polls`
  }

  if (route.section === 'events') {
    if (route.historyEventID) {
      return `/org/${encodeURIComponent(orgSegment)}/events/${route.historyEventID}`
    }
    return `/org/${encodeURIComponent(orgSegment)}/events`
  }

  if (route.section === 'event_templates') {
    if (route.eventView === 'create') {
      return `/org/${encodeURIComponent(orgSegment)}/event-templates/new`
    }
    if (route.eventView === 'edit' && route.eventID) {
      return `/org/${encodeURIComponent(orgSegment)}/event-templates/${route.eventID}`
    }
    return `/org/${encodeURIComponent(orgSegment)}/event-templates`
  }

  return `/org/${encodeURIComponent(orgSegment)}/${route.section}`
}

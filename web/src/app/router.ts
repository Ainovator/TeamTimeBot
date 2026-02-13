export type Section = 'overview' | 'members' | 'templates' | 'events' | 'history'

export type RouteState = {
  chatID: number | null
  orgKey?: string | null
  section: Section
  memberID?: number | null
  historyEventID?: number | null
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
    if (section === 'overview' || section === 'members' || section === 'templates' || section === 'events' || section === 'history') {
      route.section = section
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

  if (route.section === 'history' && segments.length >= 4) {
    const eventID = Number(segments[3])
    if (!Number.isNaN(eventID)) {
      route.historyEventID = eventID
    }
  }

  if (route.section === 'events') {
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

  if (route.section === 'history') {
    if (route.historyEventID) {
      return `/org/${encodeURIComponent(orgSegment)}/history/${route.historyEventID}`
    }
    return `/org/${encodeURIComponent(orgSegment)}/history`
  }

  if (route.section === 'events') {
    if (route.eventView === 'create') {
      return `/org/${encodeURIComponent(orgSegment)}/events/new`
    }
    if (route.eventView === 'edit' && route.eventID) {
      return `/org/${encodeURIComponent(orgSegment)}/events/${route.eventID}`
    }
    return `/org/${encodeURIComponent(orgSegment)}/events`
  }

  return `/org/${encodeURIComponent(orgSegment)}/${route.section}`
}

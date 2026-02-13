export type Section = 'overview' | 'members' | 'templates' | 'events' | 'history'

export type RouteState = {
  chatID: number | null
  section: Section
  memberID?: number | null
  historyEventID?: number | null
  templateView: 'list' | 'create' | 'edit'
  templateName: string | null
  eventView: 'list' | 'create' | 'edit'
  eventID: number | null
}

export function parseRoute(pathname: string): RouteState {
  const route: RouteState = {
    chatID: null,
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

  const chatID = Number(segments[1])
  if (Number.isNaN(chatID)) {
    return route
  }
  route.chatID = chatID

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
  if (!route.chatID) {
    return '/'
  }

  if (route.section === 'overview') {
    return `/org/${route.chatID}`
  }

  if (route.section === 'templates') {
    if (route.templateView === 'create') {
      return `/org/${route.chatID}/templates/new`
    }
    if (route.templateView === 'edit' && route.templateName) {
      return `/org/${route.chatID}/templates/${encodeURIComponent(route.templateName)}`
    }
    return `/org/${route.chatID}/templates`
  }

  if (route.section === 'members') {
    if (route.memberID) {
      return `/org/${route.chatID}/members/${route.memberID}`
    }
    return `/org/${route.chatID}/members`
  }

  if (route.section === 'history') {
    if (route.historyEventID) {
      return `/org/${route.chatID}/history/${route.historyEventID}`
    }
    return `/org/${route.chatID}/history`
  }

  if (route.section === 'events') {
    if (route.eventView === 'create') {
      return `/org/${route.chatID}/events/new`
    }
    if (route.eventView === 'edit' && route.eventID) {
      return `/org/${route.chatID}/events/${route.eventID}`
    }
    return `/org/${route.chatID}/events`
  }

  return `/org/${route.chatID}/${route.section}`
}

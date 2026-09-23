# Graph Report - .  (2026-09-12)

## Corpus Check
- 125 files · ~303,390 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1118 nodes · 2428 edges · 71 communities (65 shown, 6 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 152 edges (avg confidence: 0.81)
- Token cost: unavailable (the host did not expose measured agent usage)

## Community Hubs (Navigation)
- [[_COMMUNITY_Web API and authentication|Web API and authentication]]
- [[_COMMUNITY_Database and event billing|Database and event billing]]
- [[_COMMUNITY_Frontend API client|Frontend API client]]
- [[_COMMUNITY_Telegram bot framework|Telegram bot framework]]
- [[_COMMUNITY_Telegram command handlers|Telegram command handlers]]
- [[_COMMUNITY_Team balancing orchestration|Team balancing orchestration]]
- [[_COMMUNITY_Team optimization service|Team optimization service]]
- [[_COMMUNITY_React product interface|React product interface]]
- [[_COMMUNITY_Telegram data types|Telegram data types]]
- [[_COMMUNITY_Scheduled poll publication|Scheduled poll publication]]
- [[_COMMUNITY_Frontend domain models|Frontend domain models]]
- [[_COMMUNITY_Registration poll commands|Registration poll commands]]
- [[_COMMUNITY_Recurring event scheduling|Recurring event scheduling]]
- [[_COMMUNITY_Product and deployment documentation|Product and deployment documentation]]
- [[_COMMUNITY_Team assignment persistence|Team assignment persistence]]
- [[_COMMUNITY_Poll template management|Poll template management]]
- [[_COMMUNITY_Telegram message objects|Telegram message objects]]
- [[_COMMUNITY_Application configuration|Application configuration]]
- [[_COMMUNITY_Frontend package dependencies|Frontend package dependencies]]
- [[_COMMUNITY_TypeScript compiler configuration|TypeScript compiler configuration]]
- [[_COMMUNITY_Inline result types|Inline result types]]
- [[_COMMUNITY_Event template storage|Event template storage]]
- [[_COMMUNITY_Player skills and preferences|Player skills and preferences]]
- [[_COMMUNITY_Groups and administrators|Groups and administrators]]
- [[_COMMUNITY_Poll counting and weights|Poll counting and weights]]
- [[_COMMUNITY_Event instances and history|Event instances and history]]
- [[_COMMUNITY_Telegram vote processing|Telegram vote processing]]
- [[_COMMUNITY_Telegram API transport|Telegram API transport]]
- [[_COMMUNITY_File handling and tests|File handling and tests]]
- [[_COMMUNITY_Inline query handling|Inline query handling]]
- [[_COMMUNITY_Training announcement scheduler|Training announcement scheduler]]
- [[_COMMUNITY_Group roles and permissions|Group roles and permissions]]
- [[_COMMUNITY_Frontend routes and navigation|Frontend routes and navigation]]
- [[_COMMUNITY_Input message content|Input message content]]
- [[_COMMUNITY_Match history utilities|Match history utilities]]
- [[_COMMUNITY_Event cancellation scheduler|Event cancellation scheduler]]
- [[_COMMUNITY_Telegram conversation sessions|Telegram conversation sessions]]
- [[_COMMUNITY_Framework handler dispatch|Framework handler dispatch]]
- [[_COMMUNITY_Event settlement scheduler|Event settlement scheduler]]
- [[_COMMUNITY_Group database models|Group database models]]
- [[_COMMUNITY_Event database models|Event database models]]
- [[_COMMUNITY_Role database models|Role database models]]
- [[_COMMUNITY_Template database models|Template database models]]
- [[_COMMUNITY_Message sending options|Message sending options]]
- [[_COMMUNITY_Poll database models|Poll database models]]
- [[_COMMUNITY_Team database models|Team database models]]
- [[_COMMUNITY_Vote persistence and membership|Vote persistence and membership]]
- [[_COMMUNITY_Inline article serialization|Inline article serialization]]
- [[_COMMUNITY_Web response data structures|Web response data structures]]
- [[_COMMUNITY_Telegram protocol constants|Telegram protocol constants]]
- [[_COMMUNITY_Middle blocker imagery|Middle blocker imagery]]
- [[_COMMUNITY_Payment interface illustration|Payment interface illustration]]
- [[_COMMUNITY_Training lifecycle illustration|Training lifecycle illustration]]
- [[_COMMUNITY_Event instance deletion|Event instance deletion]]
- [[_COMMUNITY_Dashboard visual language|Dashboard visual language]]
- [[_COMMUNITY_Team grouping illustration|Team grouping illustration]]
- [[_COMMUNITY_Template interface illustration|Template interface illustration]]
- [[_COMMUNITY_Attacker vector artwork|Attacker vector artwork]]
- [[_COMMUNITY_Attacker volleyball photograph|Attacker volleyball photograph]]
- [[_COMMUNITY_Libero vector artwork|Libero vector artwork]]
- [[_COMMUNITY_Libero volleyball photograph|Libero volleyball photograph]]
- [[_COMMUNITY_Setter vector artwork|Setter vector artwork]]
- [[_COMMUNITY_Personal training profiles|Personal training profiles]]
- [[_COMMUNITY_Setter volleyball photograph|Setter volleyball photograph]]
- [[_COMMUNITY_Legacy Travis configuration|Legacy Travis configuration]]
- [[_COMMUNITY_Go module identity|Go module identity]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 88 edges
2. `Store` - 71 edges
3. `Context` - 70 edges
4. `Server` - 41 edges
5. `parseResponse()` - 37 edges
6. `Bot` - 36 edges
7. `Time` - 30 edges
8. `HandleCallback()` - 27 edges
9. `ResponseWriter` - 26 edges
10. `Request` - 25 edges

## Surprising Connections (you probably didn't know these)
- `NewBot()` --calls--> `defaultHTTPClient()`  [INFERRED]
  bot.go → client.go
- `main()` --calls--> `NewBot()`  [INFERRED]
  cmd/teamtimebot/main.go → bot.go
- `TestBot()` --calls--> `NewBot()`  [INFERRED]
  telebot_test.go → bot.go
- `main()` --calls--> `NewBot()`  [INFERRED]
  cmd/web/main.go → bot.go
- `main()` --calls--> `WithProxy()`  [INFERRED]
  cmd/teamtimebot/main.go → client.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Group activity administration** — docs_project_overview_ru_telegram_groups, docs_project_overview_ru_poll_templates, docs_project_overview_ru_poll_schedules, docs_project_overview_ru_training_slots, docs_web_admin_ru_events, docs_web_admin_ru_event_cost [INFERRED 0.85]
- **Local Docker Compose application stack** — docker_compose_postgresql, docker_compose_bot, docker_compose_ml, docker_compose_web, docker_compose_frontend [EXTRACTED 1.00]
- **Последовательность статусов тренировки** — docs_workflow_voting, docs_workflow_distribution, docs_workflow_payment_review, docs_workflow_completed [EXTRACTED 1.00]

## Communities (71 total, 6 thin omitted)

### Community 0 - "Web API and authentication"
Cohesion: 0.08
Nodes (51): authConfig, Bot, Context, GroupPermissionsView, Store, teamFormationIndicator, TeamSplitPlayer, Time (+43 more)

### Community 1 - "Database and event billing"
Cohesion: 0.08
Nodes (22): Context, JSON, Store, Time, EventPollPost, DebtorTrainingDebt, EventActivitySummary, EventBillingParticipant (+14 more)

### Community 2 - "Frontend API client"
Cohesion: 0.05
Nodes (60): activateEventPublications(), addGroupPollVoteForUser(), archiveEvent(), assignGroupRole(), autoSplitEventTeams(), bindEvent(), createEvent(), createEventInstance() (+52 more)

### Community 3 - "Telegram bot framework"
Cohesion: 0.07
Nodes (30): SendOptions, Audio, Callback, Chat, Client, Document, Duration, File (+22 more)

### Community 4 - "Telegram command handlers"
Cohesion: 0.09
Nodes (49): Bot, Callback, Chat, Context, EventView, Store, Time, User (+41 more)

### Community 5 - "Team balancing orchestration"
Cohesion: 0.09
Nodes (49): AdminGroup, AdminMember, DueSchedule, EventSetScore, EventSetsMatch, EventView, GroupMemberView, GroupPermissionsView (+41 more)

### Community 6 - "Team optimization service"
Cohesion: 0.12
Nodes (42): BaseHTTPRequestHandler, _assign_unit_to_bucket(), _bench_score(), _bucket_attacker_load(), _bucket_central_load(), _bucket_setter_quality(), _build_role_targets(), _build_stat_targets() (+34 more)

### Community 7 - "React product interface"
Cohesion: 0.09
Nodes (31): announcementLeadLabel(), announcementLeadOptions, clockMinutes(), ensureList(), formatMoney(), toHourMinute(), weekdayLabel(), weekdayOptions (+23 more)

### Community 8 - "Telegram data types"
Cohesion: 0.09
Nodes (30): Callback, File, Message, PollAnswer, Query, ChatPhoto, ChatType, EntityType (+22 more)

### Community 9 - "Scheduled poll publication"
Cohesion: 0.10
Nodes (24): Time, T, Bot, Context, Store, Time, Bot, Context (+16 more)

### Community 10 - "Frontend domain models"
Cohesion: 0.06
Nodes (32): ApiError, AuthConfig, AuthUser, DebtorTrainingDebt, EventActivitySummary, EventBilling, EventBillingPlayer, EventPollHistoryItem (+24 more)

### Community 11 - "Registration poll commands"
Cohesion: 0.14
Nodes (27): Bot, Chat, Context, Duration, Location, Mutex, Store, Time (+19 more)

### Community 12 - "Recurring event scheduling"
Cohesion: 0.12
Nodes (19): Time, Weekday, Context, Store, ScheduleView, Time, DueSchedule, PollSchedule (+11 more)

### Community 13 - "Product and deployment documentation"
Cohesion: 0.11
Nodes (29): Telegram bot service, Node 20 Vite frontend on port 5173, Team split service on port 9000, PostgreSQL 16 persistent database, Bundled production web assets, Production PostgreSQL loopback binding, Telegram login and web session configuration, Go web container on port 8080 (+21 more)

### Community 14 - "Team assignment persistence"
Cohesion: 0.16
Nodes (15): TeamFormationIndicator, TeamSplitPlayer, EventTeamSplitState, GameRosterResponse, calculateTeamChance(), enforceGuestOwnerAssignments(), guestSlotUserID(), keysOfMap() (+7 more)

### Community 15 - "Poll template management"
Cohesion: 0.24
Nodes (8): Context, PollTemplate, Store, ScheduleView, TemplateView, isSystemTemplateName(), normalizeOptionWeights(), TemplateDetails

### Community 16 - "Telegram message objects"
Cohesion: 0.11
Nodes (12): Audio, Chat, Document, Location, Sticker, Time, User, Video (+4 more)

### Community 17 - "Application configuration"
Cohesion: 0.16
Nodes (13): Client, Bot, Duration, Config, defaultPostgresDSN(), getEnvOrDefault(), Load(), main() (+5 more)

### Community 18 - "Frontend package dependencies"
Cohesion: 0.11
Nodes (17): dependencies, react, react-dom, devDependencies, @types/react, @types/react-dom, typescript, vite (+9 more)

### Community 19 - "TypeScript compiler configuration"
Cohesion: 0.11
Nodes (17): compilerOptions, allowImportingTsExtensions, isolatedModules, jsx, lib, module, moduleResolution, noEmit (+9 more)

### Community 20 - "Inline result types"
Cohesion: 0.35
Nodes (14): InlineKeyboardMarkup, InputMessageContent, InlineQueryResultArticle, InlineQueryResultAudio, InlineQueryResultBase, InlineQueryResultContact, InlineQueryResultDocument, InlineQueryResultGif (+6 more)

### Community 21 - "Event template storage"
Cohesion: 0.23
Nodes (7): Context, EventView, Store, GroupEvent, clockMinutes(), normalizeAnnouncementLeadMinutes(), normalizeEventType()

### Community 22 - "Player skills and preferences"
Cohesion: 0.20
Nodes (7): Context, Store, GroupMemberView, MemberSkillProfile, PlayerRelationView, normalizeRelationType(), SkillCatalogItem

### Community 23 - "Groups and administrators"
Cohesion: 0.27
Nodes (6): AdminGroup, AdminMember, Context, GroupView, Store, TelegramGroup

### Community 24 - "Poll counting and weights"
Cohesion: 0.21
Nodes (7): EventPollHistoryItem, GroupPollItem, GroupPollOptionItem, GroupPollVoteItem, normalizeCountedOptionIndexes(), normalizeOptionWeightsLen(), parsePollOptionChoice()

### Community 25 - "Event instances and history"
Cohesion: 0.26
Nodes (8): PollTemplate, EventInstance, EventHistoryItem, EventHistoryStatus, clampInstanceStatus(), ensureEventInstanceTx(), nextWeekdayTime(), parseClockTime()

### Community 26 - "Telegram vote processing"
Cohesion: 0.29
Nodes (11): PollAnswer, Store, Time, User, pendingVoteClearEntry, pendingVoteClearKey, cancelPendingVoteClear(), HandlePollAnswer() (+3 more)

### Community 27 - "Telegram API transport"
Cohesion: 0.25
Nodes (5): Duration, Bot, User, wrapSystem(), Update

### Community 28 - "File handling and tests"
Cohesion: 0.25
Nodes (7): T, File, NewFile(), TestBot(), TestChat(), TestFile(), TestRecipient()

### Community 29 - "Inline query handling"
Cohesion: 0.22
Nodes (9): Location, User, InlineQueryResults, inferIQR(), InlineQueryResult, InlineQueryResults, Query, QueryResponse (+1 more)

### Community 30 - "Training announcement scheduler"
Cohesion: 0.31
Nodes (8): Bot, Context, Store, Time, NewEventAnnouncementScheduler(), nextEventStartLocal(), normalizeAnnouncementLead(), EventAnnouncementScheduler

### Community 31 - "Group roles and permissions"
Cohesion: 0.31
Nodes (6): Context, GroupPermissionsView, Store, GroupRoleView, defaultMemberPermissions(), mergePermissions()

### Community 32 - "Frontend routes and navigation"
Cohesion: 0.22
Nodes (7): SectionItem, sections, buildRoutePath(), makeOrgKey(), parseRoute(), RouteState, Section

### Community 33 - "Input message content"
Cohesion: 0.20
Nodes (5): InputContactMessageContent, InputLocationMessageContent, InputMessageContent, InputTextMessageContent, InputVenueMessageContent

### Community 34 - "Match history utilities"
Cohesion: 0.22
Nodes (9): ActiveTeamCode, activeTeamCodes(), formatDateTime(), historyStatusLabel(), playerDisplayName(), teamColor(), teamPairProbabilities(), EventHistoryItem (+1 more)

### Community 35 - "Event cancellation scheduler"
Cohesion: 0.39
Nodes (6): Bot, Context, Store, NewEventCancellationScheduler(), normalizeCancelLeadMinutes(), EventCancellationScheduler

### Community 36 - "Telegram conversation sessions"
Cohesion: 0.36
Nodes (5): Mutex, action, newSessionStore(), sessionState, sessionStore

### Community 37 - "Framework handler dispatch"
Cohesion: 0.29
Nodes (5): Bot, Message, Bot, Context, Handler

### Community 38 - "Event settlement scheduler"
Cohesion: 0.43
Nodes (5): Bot, Context, Store, NewEventSettlementScheduler(), EventSettlementScheduler

### Community 39 - "Group database models"
Cohesion: 0.32
Nodes (4): Time, GroupMember, TelegramGroup, TelegramUser

### Community 40 - "Event database models"
Cohesion: 0.33
Nodes (4): JSON, Time, EventInstance, GroupEvent

### Community 41 - "Role database models"
Cohesion: 0.33
Nodes (4): JSON, Time, GroupRole, GroupRoleAssignment

### Community 42 - "Template database models"
Cohesion: 0.33
Nodes (4): JSON, Time, PollSchedule, PollTemplate

### Community 43 - "Message sending options"
Cohesion: 0.29
Nodes (6): Message, ParseMode, KeyboardButton, ReplyMarkup, ReplyMarkup, SendOptions

### Community 44 - "Poll database models"
Cohesion: 0.40
Nodes (3): Time, EventPollPost, EventPollVote

### Community 45 - "Team database models"
Cohesion: 0.40
Nodes (3): Time, EventTeamAssignment, EventTeamSession

### Community 46 - "Vote persistence and membership"
Cohesion: 0.60
Nodes (4): DB, ensureDefaultMemberSkillsTx(), upsertGroupMemberTx(), upsertTelegramUserTx()

### Community 48 - "Web response data structures"
Cohesion: 0.40
Nodes (5): EventView, GroupView, ScheduleView, TemplateView, GroupDetails

### Community 49 - "Telegram protocol constants"
Cohesion: 0.40
Nodes (4): ChatAction, ChatType, EntityType, ParseMode

### Community 50 - "Middle blocker imagery"
Cohesion: 0.50
Nodes (5): Central volleyball player illustration, Central player type illustration, Middle blocker player archetype, Volleyball block at the net, Stylized player with both arms raised

### Community 51 - "Payment interface illustration"
Cohesion: 0.50
Nodes (4): Dark navy card with rounded rows and colored status pills, Debt status badges, Event payment interface illustration, Event payment table with two rows

### Community 52 - "Training lifecycle illustration"
Cohesion: 0.50
Nodes (4): Завершено, На распределении, На проверке, В голосовании

### Community 54 - "Dashboard visual language"
Cohesion: 0.67
Nodes (3): Rounded cards with a list, avatar, status bars and action button, Dark navy interface with blue glow and muted pink and green accents, Hero illustration of a dark product interface

### Community 55 - "Team grouping illustration"
Cohesion: 1.00
Nodes (3): Connections between teams, Color differentiated team cards, Team grouping illustration

### Community 56 - "Template interface illustration"
Cohesion: 1.00
Nodes (3): Шаблон события, Poll and event template illustration, Шаблон голосования

### Community 57 - "Attacker vector artwork"
Cohesion: 0.67
Nodes (3): Attacker player type artwork, Abstract athletic silhouette in motion, Diagonal blue and pink motion bands

### Community 58 - "Attacker volleyball photograph"
Cohesion: 0.67
Nodes (3): Volleyball attacking role, Attacker role photograph: volleyball spike against a block, Net attack opposed by a two-player block

### Community 59 - "Libero vector artwork"
Cohesion: 0.67
Nodes (3): Defensive player movement, Teal flowing motion on dark background, Libero player type illustration

### Community 60 - "Libero volleyball photograph"
Cohesion: 0.67
Nodes (3): Libero defensive player role, Low forearm reception of a volleyball, Libero volleyball player photograph

### Community 61 - "Setter vector artwork"
Cohesion: 0.67
Nodes (3): Setter player type illustration, Setter volleyball role, Raised-arm player silhouette with diagonal blue and teal bands

## Knowledge Gaps
- **198 isolated node(s):** `SendOptions`, `User`, `Duration`, `Update`, `Tree` (+193 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Database and event billing` to `Web API and authentication`, `Telegram command handlers`, `Team balancing orchestration`, `Registration poll commands`, `Recurring event scheduling`, `Vote persistence and membership`, `Poll template management`, `Team assignment persistence`, `Application configuration`, `Groups and administrators`, `Event template storage`, `Event instance deletion`, `Player skills and preferences`, `Poll counting and weights`, `Event instances and history`, `Telegram API transport`, `Group roles and permissions`?**
  _High betweenness centrality (0.232) - this node is a cross-community bridge._
- **Why does `main()` connect `Telegram command handlers` to `Database and event billing`, `Event cancellation scheduler`, `Telegram bot framework`, `Event settlement scheduler`, `Scheduled poll publication`, `Application configuration`, `Telegram vote processing`, `Training announcement scheduler`?**
  _High betweenness centrality (0.117) - this node is a cross-community bridge._
- **Why does `NewBot()` connect `Telegram bot framework` to `Application configuration`, `Telegram command handlers`, `File handling and tests`?**
  _High betweenness centrality (0.071) - this node is a cross-community bridge._
- **Are the 46 inferred relationships involving `New()` (e.g. with `BuildScheduleExpr()` and `ComputeNextRunAt()`) actually correct?**
  _`New()` has 46 INFERRED edges - model-reasoned connections that need verification._
- **What connects `SendOptions`, `User`, `Duration` to the rest of the system?**
  _199 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Web API and authentication` be split into smaller, more focused modules?**
  _Cohesion score 0.08411703239289446 - nodes in this community are weakly interconnected._
- **Should `Database and event billing` be split into smaller, more focused modules?**
  _Cohesion score 0.07769973661106233 - nodes in this community are weakly interconnected._

## Audit notes

- Scope: product snapshot before the design-lab implementation, 125 detected files.
- Semantic extraction completed for all 14 chunks (11 documents and 13 individual illustrations).
- Token usage and monetary cost: unavailable. The host did not provide measured per-agent usage; no estimated counts are presented as actual usage.
- Some documentation is historical; verify product behavior against current source.
- An initial Windows multiprocessing attempt failed; the AST extraction was rerun successfully with parallel=False for all 101 code files.

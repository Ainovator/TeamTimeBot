package telegram

import "sync"

type action string

const (
	actionNone                 action = ""
	actionSelectTemplate       action = "select_template"
	actionPollName             action = "poll_name"
	actionPollQuestion         action = "poll_question"
	actionPollOptions          action = "poll_options"
	actionTemplateEditQuestion action = "template_edit_question"
	actionScheduleName         action = "schedule_name"
	actionSchedulePickDays     action = "schedule_pick_days"
	actionScheduleDays         action = "schedule_days"
	actionScheduleTime         action = "schedule_time"
	actionScheduleEditTime     action = "schedule_edit_time"
	actionEventName            action = "event_name"
	actionEventPublishTime     action = "event_publish_time"
	actionEventStartTime       action = "event_start_time"
	actionEventEndTime         action = "event_end_time"
	actionEventCost            action = "event_cost"
	actionEventBindTemplate    action = "event_bind_template"
	actionEventCountedOptions  action = "event_counted_options"
	actionEventEditCost        action = "event_edit_cost"
)

type sessionState struct {
	Action              action
	MenuSection         string
	TargetGroupChatID   int64
	SelectedTemplate    string
	AvailableTemplates  []string
	PollName            string
	PollQuestion        string
	PollOptions         []string
	ScheduleName        string
	ScheduleDays        []int
	ScheduleTimes       map[int]string
	SchedulePendingDay  int
	ManageScheduleID    uint64
	EventName           string
	EventStartDay       int
	EventPublishTime    string
	EventStartTime      string
	EventEndTime        string
	EventCostAmount     *float64
	EventBindEventID    uint64
	EventOptionCount    int
	EventCountedOptions []int
}

type sessionStore struct {
	mu    sync.Mutex
	state map[int64]sessionState
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		state: make(map[int64]sessionState),
	}
}

func (s *sessionStore) get(chatID int64) sessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state[chatID]
}

func (s *sessionStore) set(chatID int64, value sessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[chatID] = value
}

func (s *sessionStore) clear(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state, chatID)
}

package scheduler

import (
	"context"
	"log"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

type EventPollScheduler struct {
	bot   *tele.Bot
	store *postgres.Store
}

func NewEventPollScheduler(bot *tele.Bot, store *postgres.Store) *EventPollScheduler {
	return &EventPollScheduler{
		bot:   bot,
		store: store,
	}
}

func (s *EventPollScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *EventPollScheduler) tick(ctx context.Context) {
	events, err := s.store.ListActiveEventsWithGroups(ctx)
	if err != nil {
		log.Printf("event_poll: list active events failed: %v", err)
		return
	}

	nowUTC := time.Now().UTC()
	for _, event := range events {
		if event.PollTemplate == "" {
			continue
		}
		loc, err := time.LoadLocation(event.Timezone)
		if err != nil {
			continue
		}

		nowLocal := nowUTC.In(loc)
		if isoWeekday(nowLocal.Weekday()) != event.StartWeekday {
			continue
		}

		publishHour, publishMin, err := parseClockHourMinute(event.PollPublishTime)
		if err != nil {
			continue
		}
		publishLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), publishHour, publishMin, 0, 0, loc)
		if nowLocal.Before(publishLocal) {
			continue
		}

		localDate := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)
		fromUTC := localDate.UTC()
		toUTC := localDate.Add(24 * time.Hour).UTC()
		existing, err := s.store.GetLatestEventPollPostForRange(ctx, event.EventID, fromUTC, toUTC)
		if err != nil {
			log.Printf("event_poll: lookup existing post failed for event %d: %v", event.EventID, err)
			continue
		}
		if existing != nil {
			continue
		}

		template, err := s.store.GetEventTemplateDetails(ctx, event.ChatID, event.EventID)
		if err != nil {
			log.Printf("event_poll: load template failed for event %d: %v", event.EventID, err)
			continue
		}
		if len(template.TemplateOptions) < 2 {
			log.Printf("event_poll: template has <2 options for event %d", event.EventID)
			continue
		}

		chat := tele.Chat{ID: event.ChatID, Type: tele.ChatGroup}
		sent, err := s.bot.SendPollWithMeta(chat, template.TemplateQuestion, template.TemplateOptions, nil)
		if err != nil {
			log.Printf("event_poll: send poll failed for event %d chat %d: %v", event.EventID, event.ChatID, err)
			continue
		}
		if sent != nil {
			eventID := event.EventID
			if _, err := s.store.CreateEventPollPost(ctx, event.ChatID, &eventID, template.TemplateName, sent.MessageID, sent.PollID); err != nil {
				log.Printf("event_poll: save post failed for event %d: %v", event.EventID, err)
			}
		}
	}
}

func parseClockHourMinute(value string) (int, int, error) {
	layouts := []string{"15:04:05", "15:04"}
	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, value)
		if err == nil {
			return parsed.Hour(), parsed.Minute(), nil
		}
	}
	return 0, 0, err
}

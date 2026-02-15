package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

type EventAnnouncementScheduler struct {
	bot   *tele.Bot
	store *postgres.Store
}

func NewEventAnnouncementScheduler(bot *tele.Bot, store *postgres.Store) *EventAnnouncementScheduler {
	return &EventAnnouncementScheduler{
		bot:   bot,
		store: store,
	}
}

func (s *EventAnnouncementScheduler) Start(ctx context.Context) {
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

func (s *EventAnnouncementScheduler) tick(ctx context.Context) {
	events, err := s.store.ListActiveEventsWithGroups(ctx)
	if err != nil {
		log.Printf("event_announcement: list active events failed: %v", err)
		return
	}

	nowUTC := time.Now().UTC()
	for _, event := range events {
		if !event.AnnouncementEnabled {
			continue
		}
		text := strings.TrimSpace(event.AnnouncementText)
		if text == "" {
			continue
		}

		loc, err := time.LoadLocation(event.Timezone)
		if err != nil {
			log.Printf("event_announcement: invalid timezone %q for chat %d event %d: %v", event.Timezone, event.ChatID, event.EventID, err)
			continue
		}

		startHour, startMinute, err := parseClockHourMinute(event.StartTime)
		if err != nil {
			continue
		}

		lead := normalizeAnnouncementLead(event.AnnouncementLeadMinutes)
		if lead == 0 {
			continue
		}

		nowLocal := nowUTC.In(loc)
		eventStartLocal := nextEventStartLocal(nowLocal, event.StartWeekday, startHour, startMinute)
		if eventStartLocal.IsZero() {
			continue
		}
		announceAt := eventStartLocal.Add(-time.Duration(lead) * time.Minute)

		// Publish only in the pre-event window.
		if nowLocal.Before(announceAt) || !nowLocal.Before(eventStartLocal) {
			continue
		}

		eventDate := time.Date(eventStartLocal.Year(), eventStartLocal.Month(), eventStartLocal.Day(), 0, 0, 0, 0, loc)
		exists, err := s.store.HasEventAnnouncement(ctx, event.EventID, eventDate)
		if err != nil {
			log.Printf("event_announcement: check existence failed for event %d: %v", event.EventID, err)
			continue
		}
		if exists {
			continue
		}

		message := fmt.Sprintf(
			"Анонс события \"%s\"\n%s\nКогда: %s (%s)",
			event.Name,
			text,
			eventStartLocal.Format("02.01.2006 15:04"),
			event.Timezone,
		)

		chat := tele.Chat{ID: event.ChatID, Type: tele.ChatGroup}
		if err := s.bot.SendMessage(chat, message, nil); err != nil {
			log.Printf("event_announcement: send message failed for event %d chat %d: %v", event.EventID, event.ChatID, err)
			continue
		}
		if err := s.store.CreateEventAnnouncement(ctx, event.GroupID, event.EventID, eventDate); err != nil {
			log.Printf("event_announcement: save sent flag failed for event %d: %v", event.EventID, err)
		}
	}
}

func normalizeAnnouncementLead(value int) int {
	switch value {
	case 60, 120, 1440:
		return value
	case 0:
		return 60
	default:
		return 0
	}
}

func nextEventStartLocal(nowLocal time.Time, eventWeekday int, hour int, minute int) time.Time {
	if eventWeekday < 1 || eventWeekday > 7 {
		return time.Time{}
	}

	daysAhead := eventWeekday - isoWeekday(nowLocal.Weekday())
	if daysAhead < 0 {
		daysAhead += 7
	}

	candidateDate := nowLocal.AddDate(0, 0, daysAhead)
	candidate := time.Date(
		candidateDate.Year(),
		candidateDate.Month(),
		candidateDate.Day(),
		hour,
		minute,
		0,
		0,
		nowLocal.Location(),
	)

	if !candidate.After(nowLocal) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

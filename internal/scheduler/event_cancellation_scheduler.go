package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

type EventCancellationScheduler struct {
	bot   *tele.Bot
	store *postgres.Store
}

func NewEventCancellationScheduler(bot *tele.Bot, store *postgres.Store) *EventCancellationScheduler {
	return &EventCancellationScheduler{
		bot:   bot,
		store: store,
	}
}

func (s *EventCancellationScheduler) Start(ctx context.Context) {
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

func (s *EventCancellationScheduler) tick(ctx context.Context) {
	events, err := s.store.ListActiveEventsWithGroups(ctx)
	if err != nil {
		log.Printf("event_cancellation: list active events failed: %v", err)
		return
	}

	nowUTC := time.Now().UTC()
	for _, event := range events {
		if event.MinVotesToHold <= 0 || !event.CancelNotifyEnabled {
			continue
		}

		loc, err := time.LoadLocation(event.Timezone)
		if err != nil {
			continue
		}

		startHour, startMinute, err := parseClockHourMinute(event.StartTime)
		if err != nil {
			continue
		}

		nowLocal := nowUTC.In(loc)
		eventStartLocal := nextEventStartLocal(nowLocal, event.StartWeekday, startHour, startMinute)
		if eventStartLocal.IsZero() {
			continue
		}
		cancelLead := normalizeCancelLeadMinutes(event.CancelLeadMinutes)
		cancelAt := eventStartLocal.Add(-time.Duration(cancelLead) * time.Minute)
		if nowLocal.Before(cancelAt) || !nowLocal.Before(eventStartLocal) {
			continue
		}

		eventDate := time.Date(eventStartLocal.Year(), eventStartLocal.Month(), eventStartLocal.Day(), 0, 0, 0, 0, loc)
		exists, err := s.store.HasEventCancellation(ctx, event.EventID, eventDate)
		if err != nil {
			log.Printf("event_cancellation: check existence failed for event %d: %v", event.EventID, err)
			continue
		}
		if exists {
			continue
		}

		countedVotes := 0
		cycleStart := eventStartLocal.AddDate(0, 0, -7)
		post, err := s.store.GetLatestEventPollPostForRange(ctx, event.EventID, cycleStart.UTC(), eventStartLocal.UTC())
		if err != nil {
			log.Printf("event_cancellation: load latest poll post failed for event %d: %v", event.EventID, err)
			continue
		}
		if post != nil {
			countedOptions, err := s.store.GetEventTemplateCountedOptions(ctx, event.EventID)
			if err != nil {
				log.Printf("event_cancellation: load counted options failed for event %d: %v", event.EventID, err)
				continue
			}
			if len(countedOptions) > 0 {
				choices := make([]string, 0, len(countedOptions))
				for _, idx := range countedOptions {
					choices = append(choices, fmt.Sprintf("option_%d", idx))
				}
				countedVotes, err = s.store.CountVotesForPostChoices(ctx, post.ID, choices)
				if err != nil {
					log.Printf("event_cancellation: count votes failed for event %d: %v", event.EventID, err)
					continue
				}
			}
		}

		if countedVotes >= event.MinVotesToHold {
			continue
		}

		message := fmt.Sprintf(
			"Событие \"%s\" отменено.\nНедостаточно подтверждений: %d из %d.\nПлановое начало: %s (%s)",
			event.Name,
			countedVotes,
			event.MinVotesToHold,
			eventStartLocal.Format("02.01.2006 15:04"),
			event.Timezone,
		)
		chat := tele.Chat{ID: event.ChatID, Type: tele.ChatGroup}
		if err := s.bot.SendMessage(chat, message, nil); err != nil {
			log.Printf("event_cancellation: send message failed for event %d chat %d: %v", event.EventID, event.ChatID, err)
			continue
		}
		if err := s.store.CreateEventCancellation(ctx, event.GroupID, event.EventID, eventDate); err != nil {
			log.Printf("event_cancellation: save sent flag failed for event %d: %v", event.EventID, err)
		}
	}
}

func normalizeCancelLeadMinutes(value int) int {
	if value <= 0 {
		return 180
	}
	return value
}

package scheduler

import (
	"context"
	"encoding/json"
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
	instances, err := s.store.ListEventInstancesForCancellation(ctx, time.Now().UTC())
	if err != nil {
		log.Printf("event_cancellation: list instances failed: %v", err)
		return
	}

	nowUTC := time.Now().UTC()
	for _, inst := range instances {
		loc, err := time.LoadLocation(inst.Timezone)
		if err != nil {
			log.Printf("event_cancellation: invalid timezone %q for chat %d event %d: %v", inst.Timezone, inst.ChatID, inst.EventID, err)
			continue
		}

		nowLocal := nowUTC.In(loc)
		eventStartLocal := inst.PlannedStartAt.In(loc)
		cancelLead := normalizeCancelLeadMinutes(inst.CancelLeadMinutes)
		cancelAt := eventStartLocal.Add(-time.Duration(cancelLead) * time.Minute)
		if nowLocal.Before(cancelAt) || !nowLocal.Before(eventStartLocal) {
			continue
		}

		eventDate := time.Date(eventStartLocal.Year(), eventStartLocal.Month(), eventStartLocal.Day(), 0, 0, 0, 0, loc)
		exists, err := s.store.HasEventCancellation(ctx, inst.EventID, eventDate)
		if err != nil {
			log.Printf("event_cancellation: check existence failed for event %d: %v", inst.EventID, err)
			continue
		}
		if exists {
			continue
		}

		countedVotes := 0
		if inst.PollPostID != nil {
			var countedOptions []int
			_ = json.Unmarshal(inst.PollCountedOptions, &countedOptions)
			if len(countedOptions) > 0 {
				var optionWeights []int
				_ = json.Unmarshal(inst.PollOptionWeights, &optionWeights)

				choices := make([]string, 0, len(countedOptions))
				weightByChoice := make(map[string]int, len(countedOptions))
				for _, idx := range countedOptions {
					choice := fmt.Sprintf("option_%d", idx)
					choices = append(choices, choice)
					w := 1
					if idx >= 0 && idx < len(optionWeights) {
						w = optionWeights[idx]
					}
					if w <= 0 {
						w = 1
					}
					weightByChoice[choice] = w
				}

				byChoice, err := s.store.CountVotesForPostChoicesByChoice(ctx, *inst.PollPostID, choices)
				if err != nil {
					log.Printf("event_cancellation: count votes failed for event %d: %v", inst.EventID, err)
					continue
				}
				for choice, c := range byChoice {
					w := weightByChoice[choice]
					if w <= 0 {
						w = 1
					}
					countedVotes += c * w
				}
			}
		}

		if countedVotes >= inst.MinVotesToHold {
			continue
		}

		message := fmt.Sprintf(
			"Событие \"%s\" отменено.\nНедостаточно подтверждений: %d из %d мест.\nПлановое начало: %s (%s)",
			inst.EventName,
			countedVotes,
			inst.MinVotesToHold,
			eventStartLocal.Format("02.01.2006 15:04"),
			inst.Timezone,
		)
		chat := tele.Chat{ID: inst.ChatID, Type: tele.ChatGroup}
		if err := s.bot.SendMessage(chat, message, nil); err != nil {
			log.Printf("event_cancellation: send message failed for event %d chat %d: %v", inst.EventID, inst.ChatID, err)
			continue
		}
		if err := s.store.CreateEventCancellation(ctx, inst.GroupID, inst.EventID, eventDate); err != nil {
			log.Printf("event_cancellation: save sent flag failed for event %d: %v", inst.EventID, err)
		}
		if err := s.store.SetEventInstanceStatusByEventDate(ctx, inst.GroupID, inst.EventID, eventDate, string(postgres.EventHistoryStatusNotHeld)); err != nil {
			log.Printf("event_cancellation: set instance status failed for event %d: %v", inst.EventID, err)
		}
	}
}

func normalizeCancelLeadMinutes(value int) int {
	if value <= 0 {
		return 180
	}
	return value
}

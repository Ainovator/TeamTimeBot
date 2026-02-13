package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

const defaultTrainingTotalAmount = 4000.0

type EventSettlementScheduler struct {
	bot   *tele.Bot
	store *postgres.Store
}

func NewEventSettlementScheduler(bot *tele.Bot, store *postgres.Store) *EventSettlementScheduler {
	return &EventSettlementScheduler{
		bot:   bot,
		store: store,
	}
}

func (s *EventSettlementScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
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

func (s *EventSettlementScheduler) tick(ctx context.Context) {
	events, err := s.store.ListActiveEventsWithGroups(ctx)
	if err != nil {
		log.Printf("event_settlement: list active events failed: %v", err)
		return
	}

	nowUTC := time.Now().UTC()
	for _, event := range events {
		loc, err := time.LoadLocation(event.Timezone)
		if err != nil {
			continue
		}
		nowLocal := nowUTC.In(loc)
		if isoWeekday(nowLocal.Weekday()) != event.StartWeekday {
			continue
		}

		endParsed, err := time.Parse("15:04", event.EndTime)
		if err != nil {
			continue
		}
		eventEndLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), endParsed.Hour(), endParsed.Minute(), 0, 0, loc)
		if nowLocal.Before(eventEndLocal) {
			continue
		}

		localDate := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)
		exists, err := s.store.HasEventSettlement(ctx, event.EventID, localDate)
		if err != nil || exists {
			continue
		}

		dayStartUTC := localDate.UTC()
		dayEndUTC := localDate.Add(24 * time.Hour).UTC()
		post, err := s.store.GetLatestEventPollPostForRange(ctx, event.EventID, dayStartUTC, dayEndUTC)
		if err != nil {
			log.Printf("event_settlement: get latest post failed for event %d: %v", event.EventID, err)
			continue
		}

		participants := 0
		var postID *uint64
		if post != nil {
			postID = &post.ID
			countedOptions, err := s.store.GetEventCountedOptions(ctx, event.EventID)
			if err != nil {
				log.Printf("event_settlement: get counted options failed for event %d: %v", event.EventID, err)
				continue
			}
			if len(countedOptions) == 0 {
				countedOptions = []int{0}
			}
			choices := make([]string, 0, len(countedOptions))
			for _, idx := range countedOptions {
				choices = append(choices, fmt.Sprintf("option_%d", idx))
			}
			count, err := s.store.CountVotesForPostChoices(ctx, post.ID, choices)
			if err != nil {
				log.Printf("event_settlement: count votes failed for post %d: %v", post.ID, err)
				continue
			}
			participants = count
		}

		totalAmount := defaultTrainingTotalAmount
		if event.CostAmount != nil {
			totalAmount = *event.CostAmount
		}

		perPerson := 0.0
		if participants > 0 {
			perPerson = totalAmount / float64(participants)
		}

		if err := s.store.CreateEventSettlement(ctx, event.GroupID, event.EventID, postID, localDate, totalAmount, participants, perPerson); err != nil {
			log.Printf("event_settlement: create settlement failed for event %d: %v", event.EventID, err)
			continue
		}

		message := fmt.Sprintf(
			"Итоги тренировки \"%s\" за %s\nУчастников (вариант #1): %d\nСтоимость на человека: %.2f ₽",
			event.Name,
			localDate.Format("02.01.2006"),
			participants,
			perPerson,
		)
		if participants == 0 {
			message = fmt.Sprintf(
				"Итоги тренировки \"%s\" за %s\nНет голосов по варианту #1, стоимость на человека не рассчитана.",
				event.Name,
				localDate.Format("02.01.2006"),
			)
		}
		chat := tele.Chat{ID: event.ChatID, Type: tele.ChatGroup}
		if err := s.bot.SendMessage(chat, message, nil); err != nil {
			log.Printf("event_settlement: send summary failed for event %d chat %d: %v", event.EventID, event.ChatID, err)
		}
	}
}

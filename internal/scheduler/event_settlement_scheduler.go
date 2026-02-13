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
		if !event.SettlementEnabled {
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

		startParsed, err := time.Parse("15:04", event.StartTime)
		if err != nil {
			continue
		}
		endParsed, err := time.Parse("15:04", event.EndTime)
		if err != nil {
			continue
		}
		eventStartLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), startParsed.Hour(), startParsed.Minute(), 0, 0, loc)
		eventEndLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), endParsed.Hour(), endParsed.Minute(), 0, 0, loc)

		localDate := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)

		var participants int
		var perPerson float64
		var postID *uint64
		dataLoaded := false
		loadData := func() bool {
			if dataLoaded {
				return true
			}
			p, pp, pid, loadErr := s.calculateSettlement(ctx, event, localDate)
			if loadErr != nil {
				log.Printf("event_settlement: calculate failed for event %d: %v", event.EventID, loadErr)
				return false
			}
			participants = p
			perPerson = pp
			postID = pid
			dataLoaded = true
			return true
		}

		if event.SettlementPublishBefore && !nowLocal.Before(eventStartLocal) && nowLocal.Before(eventEndLocal) {
			sentBefore, err := s.store.HasEventSettlementNotice(ctx, event.EventID, localDate, "before")
			if err != nil {
				log.Printf("event_settlement: check before notice failed for event %d: %v", event.EventID, err)
				continue
			}
			if !sentBefore && loadData() {
				message := fmt.Sprintf(
					"Предварительный расчет \"%s\" за %s\nУчастников: %d\nСтоимость на человека: %.2f ₽",
					event.Name,
					localDate.Format("02.01.2006"),
					participants,
					perPerson,
				)
				if participants == 0 {
					message = fmt.Sprintf(
						"Предварительный расчет \"%s\" за %s\nПока нет голосов для расчета.",
						event.Name,
						localDate.Format("02.01.2006"),
					)
				}
				chat := tele.Chat{ID: event.ChatID, Type: tele.ChatGroup}
				if err := s.bot.SendMessage(chat, message, nil); err != nil {
					log.Printf("event_settlement: send before summary failed for event %d chat %d: %v", event.EventID, event.ChatID, err)
				} else if err := s.store.CreateEventSettlementNotice(ctx, event.GroupID, event.EventID, localDate, "before"); err != nil {
					log.Printf("event_settlement: save before notice failed for event %d: %v", event.EventID, err)
				}
			}
		}

		if event.SettlementPublishAfter && !nowLocal.Before(eventEndLocal) {
			sentAfter, err := s.store.HasEventSettlementNotice(ctx, event.EventID, localDate, "after")
			if err != nil {
				log.Printf("event_settlement: check after notice failed for event %d: %v", event.EventID, err)
				continue
			}
			if sentAfter {
				continue
			}
			if !loadData() {
				continue
			}

			exists, err := s.store.HasEventSettlement(ctx, event.EventID, localDate)
			if err != nil {
				log.Printf("event_settlement: check settlement exists failed for event %d: %v", event.EventID, err)
				continue
			}
			if !exists {
				if err := s.store.CreateEventSettlement(ctx, event.GroupID, event.EventID, postID, localDate, settlementTotalAmount(event), participants, perPerson); err != nil {
					log.Printf("event_settlement: create settlement failed for event %d: %v", event.EventID, err)
					continue
				}
			}

			message := fmt.Sprintf(
				"Итоги тренировки \"%s\" за %s\nУчастников: %d\nСтоимость на человека: %.2f ₽",
				event.Name,
				localDate.Format("02.01.2006"),
				participants,
				perPerson,
			)
			if participants == 0 {
				message = fmt.Sprintf(
					"Итоги тренировки \"%s\" за %s\nНет голосов, стоимость на человека не рассчитана.",
					event.Name,
					localDate.Format("02.01.2006"),
				)
			}
			chat := tele.Chat{ID: event.ChatID, Type: tele.ChatGroup}
			if err := s.bot.SendMessage(chat, message, nil); err != nil {
				log.Printf("event_settlement: send after summary failed for event %d chat %d: %v", event.EventID, event.ChatID, err)
			} else if err := s.store.CreateEventSettlementNotice(ctx, event.GroupID, event.EventID, localDate, "after"); err != nil {
				log.Printf("event_settlement: save after notice failed for event %d: %v", event.EventID, err)
			}
		}
	}
}

func settlementTotalAmount(event postgres.EventWithGroupView) float64 {
	totalAmount := defaultTrainingTotalAmount
	if event.CostAmount != nil {
		totalAmount = *event.CostAmount
	}
	return totalAmount
}

func (s *EventSettlementScheduler) calculateSettlement(
	ctx context.Context,
	event postgres.EventWithGroupView,
	localDate time.Time,
) (participants int, perPerson float64, postID *uint64, err error) {
	dayStartUTC := localDate.UTC()
	dayEndUTC := localDate.Add(24 * time.Hour).UTC()
	post, err := s.store.GetLatestEventPollPostForRange(ctx, event.EventID, dayStartUTC, dayEndUTC)
	if err != nil {
		return 0, 0, nil, err
	}

	if post != nil {
		postID = &post.ID
		countedOptions, err := s.store.GetEventTemplateCountedOptions(ctx, event.EventID)
		if err != nil {
			return 0, 0, nil, err
		}
		if len(countedOptions) == 0 {
			return 0, 0, postID, nil
		}
		choices := make([]string, 0, len(countedOptions))
		for _, idx := range countedOptions {
			choices = append(choices, fmt.Sprintf("option_%d", idx))
		}
		count, err := s.store.CountVotesForPostChoices(ctx, post.ID, choices)
		if err != nil {
			return 0, 0, nil, err
		}
		participants = count
	}

	totalAmount := settlementTotalAmount(event)
	if participants > 0 {
		perPerson = totalAmount / float64(participants)
	}
	return participants, perPerson, postID, nil
}

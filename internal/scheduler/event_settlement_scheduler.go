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
			log.Printf("event_settlement: invalid timezone %q for chat %d event %d: %v", event.Timezone, event.ChatID, event.EventID, err)
			continue
		}
		nowLocal := nowUTC.In(loc)

		// Work with an instance snapshot for today. If instance doesn't exist yet, create it once.
		localDate := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)
		inst, err := s.store.GetEventInstanceSnapshotByEventDate(ctx, event.GroupID, event.EventID, localDate)
		if err != nil {
			log.Printf("event_settlement: load instance failed for event %d: %v", event.EventID, err)
			continue
		}
		if inst == nil {
			_, err := s.store.EnsureEventInstanceForDate(
				ctx,
				event.GroupID,
				event.EventID,
				localDate,
				event.StartTime,
				event.EndTime,
				string(postgres.EventHistoryStatusInVoting),
				event.Timezone,
			)
			if err != nil {
				log.Printf("event_settlement: ensure instance failed for event %d: %v", event.EventID, err)
				continue
			}
			inst, err = s.store.GetEventInstanceSnapshotByEventDate(ctx, event.GroupID, event.EventID, localDate)
			if err != nil || inst == nil {
				continue
			}
		}

		if !inst.SettlementEnabled {
			continue
		}
		if isoWeekday(nowLocal.Weekday()) != event.StartWeekday {
			// Keep old guard for now; instance schedule is still based on the template's weekday.
			continue
		}
		eventStartLocal := inst.PlannedStartAt.In(loc)
		eventEndLocal := inst.PlannedEndAt.In(loc)

		var participants int
		var perPerson float64
		var postID *uint64
		dataLoaded := false
		loadData := func() bool {
			if dataLoaded {
				return true
			}
			participants = 0
			perPerson = 0
			postID = nil

			if inst.PollPostID != nil {
				postID = inst.PollPostID
				var countedOptions []int
				_ = json.Unmarshal(inst.PollCountedOptions, &countedOptions)
				if len(countedOptions) > 0 {
					choices := make([]string, 0, len(countedOptions))
					for _, idx := range countedOptions {
						choices = append(choices, fmt.Sprintf("option_%d", idx))
					}
					count, err := s.store.CountVotesForPostChoices(ctx, *inst.PollPostID, choices)
					if err != nil {
						log.Printf("event_settlement: count votes failed for event %d: %v", event.EventID, err)
						return false
					}
					participants = count
				}
			}

			totalAmount := defaultTrainingTotalAmount
			if inst.CostAmount != nil {
				totalAmount = *inst.CostAmount
			}
			if participants > 0 {
				perPerson = totalAmount / float64(participants)
			}
			dataLoaded = true
			return true
		}

		if inst.SettlementPublishBefore && !nowLocal.Before(eventStartLocal) && nowLocal.Before(eventEndLocal) {
			sentBefore, err := s.store.HasEventSettlementNotice(ctx, event.EventID, localDate, "before")
			if err != nil {
				log.Printf("event_settlement: check before notice failed for event %d: %v", event.EventID, err)
				continue
			}
			if !sentBefore && loadData() {
				message := fmt.Sprintf(
					"Предварительный расчет \"%s\" за %s\nУчастников: %d\nСтоимость на человека: %.2f ₽",
					inst.EventName,
					localDate.Format("02.01.2006"),
					participants,
					perPerson,
				)
				if participants == 0 {
					message = fmt.Sprintf(
						"Предварительный расчет \"%s\" за %s\nПока нет голосов для расчета.",
						inst.EventName,
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

		if inst.SettlementPublishAfter && !nowLocal.Before(eventEndLocal) {
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
				instanceID := inst.InstanceID
				totalAmount := defaultTrainingTotalAmount
				if inst.CostAmount != nil {
					totalAmount = *inst.CostAmount
				}
				if err := s.store.CreateEventSettlement(ctx, event.GroupID, event.EventID, &instanceID, postID, localDate, totalAmount, participants, perPerson); err != nil {
					log.Printf("event_settlement: create settlement failed for event %d: %v", event.EventID, err)
					continue
				}
				if err := s.store.SetEventInstanceStatusByEventDate(ctx, event.GroupID, event.EventID, localDate, string(postgres.EventHistoryStatusOnReview)); err != nil {
					log.Printf("event_settlement: set instance status failed for event %d: %v", event.EventID, err)
				}
			}

			message := fmt.Sprintf(
				"Итоги тренировки \"%s\" за %s\nУчастников: %d\nСтоимость на человека: %.2f ₽",
				inst.EventName,
				localDate.Format("02.01.2006"),
				participants,
				perPerson,
			)
			if participants == 0 {
				message = fmt.Sprintf(
					"Итоги тренировки \"%s\" за %s\nНет голосов, стоимость на человека не рассчитана.",
					inst.EventName,
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

// Legacy helper methods removed: settlement calculations are now based on instance snapshots.

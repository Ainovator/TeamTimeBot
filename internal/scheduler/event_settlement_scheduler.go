package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
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

		// Do not create/ensure instances on non-event days.
		// Otherwise we may accidentally create "today" instances that later get cancelled.
		if isoWeekday(nowLocal.Weekday()) != event.StartWeekday {
			continue
		}

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
		eventStartLocal := inst.PlannedStartAt.In(loc)
		eventEndLocal := inst.PlannedEndAt.In(loc)

		var seats int
		var pricePerSeat float64
		var postID *uint64
		var payers []postgres.PollSeatCountItem
		dataLoaded := false
		loadData := func() bool {
			if dataLoaded {
				return true
			}
			seats = 0
			pricePerSeat = 0
			postID = nil
			payers = nil

			if inst.PollPostID != nil {
				postID = inst.PollPostID
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

					items, totalSeats, err := s.store.ListSeatCountsForPostChoices(ctx, *inst.PollPostID, choices, weightByChoice)
					if err != nil {
						log.Printf("event_settlement: count votes failed for event %d: %v", event.EventID, err)
						return false
					}
					payers = items
					seats = totalSeats
				}
			}

			totalAmount := defaultTrainingTotalAmount
			if inst.CostAmount != nil {
				totalAmount = *inst.CostAmount
			}
			if seats > 0 {
				pricePerSeat = math.Ceil(totalAmount / float64(seats))
			}
			dataLoaded = true
			return true
		}

		buildMessage := func(prefix string) string {
			if seats == 0 {
				return fmt.Sprintf("%s \"%s\" за %s\nПока нет голосов для расчета.", prefix, inst.EventName, localDate.Format("02.01.2006"))
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("%s \"%s\" за %s\n", prefix, inst.EventName, localDate.Format("02.01.2006")))
			b.WriteString(fmt.Sprintf("Мест: %d\n", seats))
			b.WriteString(fmt.Sprintf("Цена за место: %.0f ₽\n\n", pricePerSeat))
			for _, p := range payers {
				if p.Seats <= 0 {
					continue
				}
				name := strings.TrimSpace(p.RealName)
				if name == "" {
					name = strings.TrimSpace(strings.TrimSpace(p.FirstName + " " + p.LastName))
				}
				if name == "" && strings.TrimSpace(p.Username) != "" {
					name = "@" + strings.TrimSpace(p.Username)
				}
				if name == "" {
					name = fmt.Sprintf("id:%d", p.UserID)
				}
				amount := pricePerSeat * float64(p.Seats)
				b.WriteString(fmt.Sprintf("%s — %.0f ₽\n", name, amount))
			}
			return strings.TrimSpace(b.String())
		}

		if inst.SettlementPublishBefore && !nowLocal.Before(eventStartLocal) && nowLocal.Before(eventEndLocal) {
			sentBefore, err := s.store.HasEventSettlementNotice(ctx, event.EventID, localDate, "before")
			if err != nil {
				log.Printf("event_settlement: check before notice failed for event %d: %v", event.EventID, err)
				continue
			}
			if !sentBefore && loadData() {
				message := buildMessage("Предварительный расчет")
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
				if err := s.store.CreateEventSettlement(ctx, event.GroupID, event.EventID, &instanceID, postID, localDate, totalAmount, seats, pricePerSeat); err != nil {
					log.Printf("event_settlement: create settlement failed for event %d: %v", event.EventID, err)
					continue
				}
				if err := s.store.SetEventInstanceStatusByEventDate(ctx, event.GroupID, event.EventID, localDate, string(postgres.EventHistoryStatusOnReview)); err != nil {
					log.Printf("event_settlement: set instance status failed for event %d: %v", event.EventID, err)
				}
			}

			message := buildMessage("Итоги тренировки")
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

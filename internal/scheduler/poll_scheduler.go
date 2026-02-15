package scheduler

import (
	"context"
	"log"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

type PollScheduler struct {
	bot   *tele.Bot
	store *postgres.Store
}

func NewPollScheduler(bot *tele.Bot, store *postgres.Store) *PollScheduler {
	return &PollScheduler{
		bot:   bot,
		store: store,
	}
}

func (s *PollScheduler) Start(ctx context.Context) {
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

func (s *PollScheduler) tick(ctx context.Context) {
	nowUTC := time.Now().UTC()

	schedules, err := s.store.ListDueSchedules(ctx, nowUTC, 100)
	if err != nil {
		log.Printf("scheduler: list due schedules failed: %v", err)
		return
	}

	for _, schedule := range schedules {
		nextRunAt, calcErr := postgres.ComputeNextRunAt(schedule.Timezone, schedule.ScheduleExpr, nowUTC)
		if calcErr != nil {
			log.Printf("scheduler: next run calc failed for schedule %d: %v", schedule.ScheduleID, calcErr)
			_ = s.store.SetNextRunAt(ctx, schedule.ScheduleID, nowUTC.Add(10*time.Minute))
			continue
		}

		chat := tele.Chat{ID: schedule.ChatID, Type: tele.ChatGroup}
		sent, err := s.bot.SendPollWithMeta(chat, schedule.Question, schedule.Options, nil)
		if err != nil {
			log.Printf("scheduler: send poll failed for schedule %d chat %d: %v", schedule.ScheduleID, schedule.ChatID, err)
			_ = s.store.SetNextRunAt(ctx, schedule.ScheduleID, nowUTC.Add(2*time.Minute))
			continue
		}
		if sent != nil {
			var eventID *uint64
			loc, locErr := time.LoadLocation(schedule.Timezone)
			if locErr != nil {
				log.Printf("scheduler: invalid timezone %q for schedule %d chat %d: %v", schedule.Timezone, schedule.ScheduleID, schedule.ChatID, locErr)
			} else {
				localWeekday := isoWeekday(nowUTC.In(loc).Weekday())
				foundEventID, findErr := s.store.FindBoundEventIDByTemplateAndWeekday(ctx, schedule.ChatID, schedule.TemplateName, localWeekday)
				if findErr != nil {
					log.Printf("scheduler: find bound event failed for schedule %d: %v", schedule.ScheduleID, findErr)
				} else {
					eventID = foundEventID
				}
			}
			if _, err := s.store.CreateEventPollPost(
				ctx,
				schedule.ChatID,
				eventID,
				schedule.TemplateName,
				sent.MessageID,
				sent.PollID,
			); err != nil {
				log.Printf("scheduler: post persistence failed for schedule %d chat %d: %v", schedule.ScheduleID, schedule.ChatID, err)
			}
		}

		if err := s.store.SetNextRunAt(ctx, schedule.ScheduleID, nextRunAt); err != nil {
			log.Printf("scheduler: update next run failed for schedule %d: %v", schedule.ScheduleID, err)
		}
	}
}

func isoWeekday(day time.Weekday) int {
	if day == time.Sunday {
		return 7
	}
	return int(day)
}

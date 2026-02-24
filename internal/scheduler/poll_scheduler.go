package scheduler

import (
	"context"
	"log"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/polls"
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

		pollQuestion := schedule.Question
		var eventID *uint64
		loc, locErr := time.LoadLocation(schedule.Timezone)
		if locErr != nil {
			log.Printf("scheduler: invalid timezone %q for schedule %d chat %d: %v", schedule.Timezone, schedule.ScheduleID, schedule.ChatID, locErr)
		} else {
			nowLocal := nowUTC.In(loc)
			localWeekday := isoWeekday(nowLocal.Weekday())
			foundEventID, findErr := s.store.FindBoundEventIDByTemplateAndWeekday(ctx, schedule.ChatID, schedule.TemplateName, localWeekday)
			if findErr != nil {
				log.Printf("scheduler: find bound event failed for schedule %d: %v", schedule.ScheduleID, findErr)
			} else {
				eventID = foundEventID
			}
			if eventID != nil {
				event, eventErr := s.store.GetEventByID(ctx, schedule.ChatID, *eventID)
				if eventErr != nil {
					log.Printf("scheduler: load event failed for schedule %d event %d: %v", schedule.ScheduleID, *eventID, eventErr)
				} else {
					startHour, startMinute, startErr := parseClockHourMinute(event.StartTime)
					if startErr != nil {
						log.Printf("scheduler: invalid event start time %q for event %d: %v", event.StartTime, event.ID, startErr)
					} else {
						targetStartLocal := nextWeekdayTimeLocal(nowLocal, event.StartWeekday, startHour, startMinute)
						pollQuestion = polls.WithEventDate(schedule.Question, targetStartLocal)
					}
				}
			}
		}

		chat := tele.Chat{ID: schedule.ChatID, Type: tele.ChatGroup}
		sent, err := s.bot.SendPollWithMeta(chat, pollQuestion, schedule.Options, nil)
		if err != nil {
			log.Printf("scheduler: send poll failed for schedule %d chat %d: %v", schedule.ScheduleID, schedule.ChatID, err)
			_ = s.store.SetNextRunAt(ctx, schedule.ScheduleID, nowUTC.Add(2*time.Minute))
			continue
		}
		if sent != nil {
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

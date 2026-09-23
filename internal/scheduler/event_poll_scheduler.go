package scheduler

import (
	"context"
	"log"
	"time"

	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/internal/notifications"

	"gopkg.in/telebot.v4/internal/polls"
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
	debug := schedulerDebugEnabled()
	testMode := schedulerTestModeEnabled()
	if debug {
		log.Printf("event_poll: tick nowUTC=%s active_events=%d test_mode=%v", nowUTC.Format(time.RFC3339), len(events), testMode)
	}
	for _, event := range events {
		if event.PollTemplate == "" {
			if debug {
				log.Printf("event_poll: skip event %d chat %d: no poll template", event.EventID, event.ChatID)
			}
			continue
		}
		loc, err := time.LoadLocation(event.Timezone)
		if err != nil {
			log.Printf("event_poll: invalid timezone %q for chat %d event %d: %v", event.Timezone, event.ChatID, event.EventID, err)
			continue
		}

		nowLocal := nowUTC.In(loc)
		publishWeekday := event.PollPublishWeekday
		if publishWeekday == 0 {
			publishWeekday = event.StartWeekday
		}
		nowW := isoWeekday(nowLocal.Weekday())
		if nowW != publishWeekday {
			if debug {
				log.Printf("event_poll: skip event %d chat %d: weekday now=%d want=%d nowLocal=%s", event.EventID, event.ChatID, nowW, publishWeekday, nowLocal.Format(time.RFC3339))
			}
			continue
		}

		publishHour, publishMin, err := parseClockHourMinute(event.PollPublishTime)
		if err != nil {
			log.Printf("event_poll: invalid publish time %q for chat %d event %d: %v", event.PollPublishTime, event.ChatID, event.EventID, err)
			continue
		}
		publishLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), publishHour, publishMin, 0, 0, loc)
		if nowLocal.Before(publishLocal) {
			if debug {
				log.Printf("event_poll: skip event %d chat %d: not time yet publishLocal=%s nowLocal=%s", event.EventID, event.ChatID, publishLocal.Format(time.RFC3339), nowLocal.Format(time.RFC3339))
			}
			continue
		}

		startHour, startMin, err := parseClockHourMinute(event.StartTime)
		if err != nil {
			log.Printf("event_poll: invalid start time %q for chat %d event %d: %v", event.StartTime, event.ChatID, event.EventID, err)
			continue
		}
		targetStartLocal := nextWeekdayTimeLocal(nowLocal, event.StartWeekday, startHour, startMin)
		targetEventDate := time.Date(targetStartLocal.Year(), targetStartLocal.Month(), targetStartLocal.Day(), 0, 0, 0, 0, loc)

		if !testMode {
			localDate := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)
			fromUTC := localDate.UTC()
			toUTC := localDate.Add(24 * time.Hour).UTC()
			existing, err := s.store.GetLatestEventPollPostForRange(ctx, event.EventID, fromUTC, toUTC)
			if err != nil {
				log.Printf("event_poll: lookup existing post failed for event %d: %v", event.EventID, err)
				continue
			}
			if existing != nil {
				if debug {
					log.Printf("event_poll: skip event %d chat %d: already posted today (post_id=%d published_at=%s)", event.EventID, event.ChatID, existing.ID, existing.PublishedAt.Format(time.RFC3339))
				}
				continue
			}

			instance, err := s.store.GetEventInstanceSnapshotByEventDate(ctx, event.GroupID, event.EventID, targetEventDate)
			if err != nil {
				log.Printf("event_poll: lookup instance failed for event %d date %s: %v", event.EventID, targetEventDate.Format("2006-01-02"), err)
				continue
			}
			if instance != nil {
				if instance.PollPostID != nil {
					if debug {
						log.Printf("event_poll: skip event %d chat %d: instance %d date %s already has poll_post_id=%d", event.EventID, event.ChatID, instance.InstanceID, targetEventDate.Format("2006-01-02"), *instance.PollPostID)
					}
					continue
				}
				hasPollPost, err := s.store.HasEventPollPostForInstance(ctx, instance.InstanceID)
				if err != nil {
					log.Printf("event_poll: lookup poll post by instance failed for event %d instance %d: %v", event.EventID, instance.InstanceID, err)
					continue
				}
				if hasPollPost {
					if debug {
						log.Printf("event_poll: skip event %d chat %d: instance %d date %s already has poll post", event.EventID, event.ChatID, instance.InstanceID, targetEventDate.Format("2006-01-02"))
					}
					continue
				}
			}
		} else if debug {
			log.Printf("event_poll: test_mode bypass: allow duplicate polls for event %d chat %d", event.EventID, event.ChatID)
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

		recipients, err := s.store.GetEventMentionRecipients(ctx, event.ChatID, event.EventID, "poll")
		if err != nil {
			log.Printf("event_poll: load mention recipients for event %d: %v", event.EventID, err)
			continue
		}
		chat := tele.Chat{ID: event.ChatID, Type: tele.ChatGroup}
		pollQuestion := polls.WithEventDate(template.TemplateQuestion, targetEventDate)
		if debug {
			log.Printf("event_poll: posting event %d chat %d template=%q question_len=%d options=%d", event.EventID, event.ChatID, template.TemplateName, len(pollQuestion), len(template.TemplateOptions))
		}
		sent, err := s.bot.SendPollWithMeta(chat, pollQuestion, template.TemplateOptions, nil)
		if err != nil {
			log.Printf("event_poll: send poll failed for event %d chat %d: %v", event.EventID, event.ChatID, err)
			continue
		}
		if sent != nil {
			eventID := event.EventID
			if _, err := s.store.CreateEventPollPost(ctx, event.ChatID, &eventID, template.TemplateName, sent.MessageID, sent.PollID); err != nil {
				log.Printf("event_poll: save post failed for event %d: %v", event.EventID, err)
			} else if debug {
				log.Printf("event_poll: saved post for event %d chat %d telegram_message_id=%d poll_id=%s", event.EventID, event.ChatID, sent.MessageID, sent.PollID)
			}
			if err := notifications.SendHTML(s.bot, chat, notifications.PollInvitation(pollQuestion, recipients), sent.MessageID); err != nil {
				log.Printf("event_poll: poll published but mentions failed for event %d: %v", event.EventID, err)
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

func nextWeekdayTimeLocal(now time.Time, weekday int, hour int, minute int) time.Time {
	if weekday < 1 || weekday > 7 {
		return now
	}
	currentWeekday := isoWeekday(now.Weekday())
	delta := weekday - currentWeekday
	if delta < 0 {
		delta += 7
	}
	candidateDate := now.AddDate(0, 0, delta)
	candidate := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(), hour, minute, 0, 0, now.Location())
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

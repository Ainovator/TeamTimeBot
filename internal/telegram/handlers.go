package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

func RegisterHandlers(bot *tele.Bot, store *postgres.Store) {
	bot.Handle("/start", func(c tele.Context) error {
		return c.Send("TeamTimeBot готов. Команды: /setgroup, /setpoll, /setschedule")
	})

	bot.Handle("/setgroup", func(c tele.Context) error {
		payload := strings.TrimSpace(c.Message().Payload)
		if payload == "" {
			return c.Send("Использование: /setgroup Europe/Moscow")
		}

		if _, err := time.LoadLocation(payload); err != nil {
			return c.Send("Некорректный timezone. Пример: Europe/Moscow")
		}

		chat := c.Chat()
		title := chat.Title
		if title == "" {
			title = fmt.Sprintf("chat_%d", chat.ID)
		}

		group, err := store.UpsertGroup(context.Background(), chat.ID, title, payload)
		if err != nil {
			return c.Send("Ошибка сохранения группы: " + err.Error())
		}

		return c.Send(fmt.Sprintf("Группа сохранена: %s (%s)", group.Title, group.Timezone))
	})

	bot.Handle("/setpoll", func(c tele.Context) error {
		payload := strings.TrimSpace(c.Message().Payload)
		if payload == "" {
			return c.Send("Использование: /setpoll name|question|Да,Нет")
		}

		name, question, options, err := postgres.ParsePollSpec(payload)
		if err != nil {
			return c.Send(err.Error())
		}

		_, err = store.UpsertPollTemplate(context.Background(), c.Chat().ID, name, question, options)
		if err != nil {
			return c.Send("Ошибка сохранения шаблона: " + err.Error())
		}

		return c.Send(fmt.Sprintf("Шаблон `%s` сохранен. Опций: %d", name, len(options)))
	})

	bot.Handle("/setschedule", func(c tele.Context) error {
		payload := strings.TrimSpace(c.Message().Payload)
		if payload == "" {
			return c.Send("Использование: /setschedule template_name|0 18 * * 2,4")
		}

		templateName, cronExpr, err := postgres.ParseScheduleSpec(payload)
		if err != nil {
			return c.Send(err.Error())
		}

		_, err = store.UpsertSchedule(context.Background(), c.Chat().ID, templateName, cronExpr)
		if err != nil {
			return c.Send("Ошибка сохранения расписания: " + err.Error())
		}

		return c.Send(fmt.Sprintf("Расписание для `%s` сохранено: %s", templateName, cronExpr))
	})
}

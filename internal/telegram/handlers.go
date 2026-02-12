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
	bot.Handle("/start", func(c tele.Context) {
		_ = c.Bot.SendMessage(c.Message.Chat, "TeamTimeBot готов. Команды: /setgroup, /setpoll, /setschedule", nil)
	})

	bot.Handle("/setgroup", func(c tele.Context) {
		payload := extractCommandPayload(c.Message.Text, "/setgroup")
		if payload == "" {
			_ = c.Bot.SendMessage(c.Message.Chat, "Использование: /setgroup Europe/Moscow", nil)
			return
		}

		if _, err := time.LoadLocation(payload); err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, "Некорректный timezone. Пример: Europe/Moscow", nil)
			return
		}

		chat := c.Message.Chat
		title := chat.Title
		if title == "" {
			title = fmt.Sprintf("chat_%d", chat.ID)
		}

		group, err := store.UpsertGroup(context.Background(), chat.ID, title, payload)
		if err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, "Ошибка сохранения группы: "+err.Error(), nil)
			return
		}

		_ = c.Bot.SendMessage(c.Message.Chat, fmt.Sprintf("Группа сохранена: %s (%s)", group.Title, group.Timezone), nil)
	})

	bot.Handle("/setpoll", func(c tele.Context) {
		payload := extractCommandPayload(c.Message.Text, "/setpoll")
		if payload == "" {
			_ = c.Bot.SendMessage(c.Message.Chat, "Использование: /setpoll name|question|Да,Нет", nil)
			return
		}

		name, question, options, err := postgres.ParsePollSpec(payload)
		if err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, err.Error(), nil)
			return
		}

		_, err = store.UpsertPollTemplate(context.Background(), c.Message.Chat.ID, name, question, options)
		if err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, "Ошибка сохранения шаблона: "+err.Error(), nil)
			return
		}

		_ = c.Bot.SendMessage(c.Message.Chat, fmt.Sprintf("Шаблон %s сохранен. Опций: %d", name, len(options)), nil)
	})

	bot.Handle("/setschedule", func(c tele.Context) {
		payload := extractCommandPayload(c.Message.Text, "/setschedule")
		if payload == "" {
			_ = c.Bot.SendMessage(c.Message.Chat, "Использование: /setschedule template_name|0 18 * * 2,4", nil)
			return
		}

		templateName, cronExpr, err := postgres.ParseScheduleSpec(payload)
		if err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, err.Error(), nil)
			return
		}

		_, err = store.UpsertSchedule(context.Background(), c.Message.Chat.ID, templateName, cronExpr)
		if err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, "Ошибка сохранения расписания: "+err.Error(), nil)
			return
		}

		_ = c.Bot.SendMessage(c.Message.Chat, fmt.Sprintf("Расписание для %s сохранено: %s", templateName, cronExpr), nil)
	})
}

func extractCommandPayload(text, command string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, command) {
		return ""
	}

	rest := strings.TrimSpace(text[len(command):])
	if strings.HasPrefix(rest, "@") {
		parts := strings.Fields(rest)
		if len(parts) <= 1 {
			return ""
		}
		return strings.Join(parts[1:], " ")
	}

	return rest
}

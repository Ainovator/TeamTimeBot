package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

var sessions = newSessionStore()

const defaultTimezone = "Europe/Moscow"

func RegisterHandlers(bot *tele.Bot, store *postgres.Store) {
	bot.Handle("/start", func(c tele.Context) {
		_ = c.Bot.SendMessage(
			c.Message.Chat,
			"TeamTimeBot готов.\nОткрой /help для пошаговой настройки.",
			nil,
		)
	})

	bot.Handle("/help", func(c tele.Context) {
		_ = c.Bot.SendMessage(
			c.Message.Chat,
			"Как начать:\n"+
				"1) Добавь бота в нужную группу.\n"+
				"2) В группе выполни /setgroup (сохраняет группу и админов).\n"+
				"3) В личке боту выполни /settings.\n"+
				"4) Выбери группу -> выбери/создай шаблон -> настрой расписание.\n\n"+
				"Если ты новый админ и группа не видна в /settings:\n"+
				"- Выполни /updategroup в группе (или попроси другого админа сделать это).\n\n"+
				"Команды:\n"+
				"/start - краткий старт\n"+
				"/help - эта справка\n"+
				"/settings - меню настройки\n"+
				"/setgroup - подключить группу (только в группе)\n"+
				"/updategroup - обновить список админов (только в группе)",
			nil,
		)
	})

	bot.Handle("/settings", func(c tele.Context) {
		state := sessions.get(c.Message.Chat.ID)
		state.Action = actionNone
		state.MenuSection = ""
		state.SelectedTemplate = ""
		state.AvailableTemplates = nil
		state.ScheduleName = ""
		state.ScheduleDays = nil
		state.ScheduleTimes = nil
		state.SchedulePendingDay = 0
		state.ManageScheduleID = 0
		state.EventName = ""
		state.EventStartDay = 0
		state.EventPublishTime = ""
		state.EventStartTime = ""
		state.EventEndTime = ""
		state.EventCostAmount = nil
		state.EventOptionCount = 0
		sessions.set(c.Message.Chat.ID, state)

		if isPrivateChat(c.Message.Chat) {
			sendGroupSelectorMenu(c.Bot, store, c.Message.Chat, c.Message.Sender)
			return
		}
		if !ensureGroupAdmin(c.Bot, c.Message) {
			return
		}

		setTargetGroup(c.Message.Chat.ID, c.Message.Chat.ID)
		sendSettingsMenu(c.Bot, c.Message.Chat, formatGroupTitle(c.Message.Chat.ID, c.Message.Chat.Title))
	})

	bot.Handle("/setgroup", func(c tele.Context) {
		if isPrivateChat(c.Message.Chat) {
			_ = c.Bot.SendMessage(c.Message.Chat, "Команду /setgroup нужно запускать в группе.", nil)
			return
		}
		if !ensureGroupAdmin(c.Bot, c.Message) {
			return
		}

		chat := c.Message.Chat
		title := formatGroupTitle(chat.ID, chat.Title)

		group, err := store.UpsertGroup(context.Background(), chat.ID, title, defaultTimezone)
		if err != nil {
			_ = c.Bot.SendMessage(chat, "Ошибка сохранения группы: "+err.Error(), nil)
			return
		}

		if err := syncGroupAdmins(c.Bot, store, chat); err != nil {
			_ = c.Bot.SendMessage(chat, "Группа сохранена, но не удалось синхронизировать админов: "+err.Error(), nil)
			return
		}

		if _, err := store.EnsureDefaultRegistrationTemplate(context.Background(), chat.ID); err != nil {
			_ = c.Bot.SendMessage(chat, "Группа сохранена, но не удалось создать шаблон \"Регистрация\": "+err.Error(), nil)
			return
		}

		_ = c.Bot.SendMessage(chat, fmt.Sprintf("Группа сохранена: %s (%s). Админы синхронизированы.", group.Title, group.Timezone), nil)
	})

	bot.Handle("/updategroup", func(c tele.Context) {
		if isPrivateChat(c.Message.Chat) {
			_ = c.Bot.SendMessage(c.Message.Chat, "Команду /updategroup нужно запускать в группе.", nil)
			return
		}
		if !ensureGroupAdmin(c.Bot, c.Message) {
			return
		}
		if err := syncGroupAdmins(c.Bot, store, c.Message.Chat); err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, "Ошибка синхронизации админов: "+err.Error(), nil)
			return
		}
		_ = c.Bot.SendMessage(c.Message.Chat, "Список админов обновлен.", nil)
	})

	bot.Handle("/setpoll", func(c tele.Context) {
		if !isPrivateChat(c.Message.Chat) && !ensureGroupAdmin(c.Bot, c.Message) {
			return
		}
		targetChatID := resolveTargetGroupChatID(c.Message)
		if targetChatID == 0 {
			_ = c.Bot.SendMessage(c.Message.Chat, "Сначала выбери группу через /settings.", nil)
			return
		}

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

		_, err = store.UpsertPollTemplate(context.Background(), targetChatID, name, question, options)
		if err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, "Ошибка сохранения шаблона: "+err.Error(), nil)
			return
		}

		_ = c.Bot.SendMessage(c.Message.Chat, fmt.Sprintf("Шаблон %s сохранен. Опций: %d", name, len(options)), nil)
	})

	bot.Handle("/setschedule", func(c tele.Context) {
		if !isPrivateChat(c.Message.Chat) && !ensureGroupAdmin(c.Bot, c.Message) {
			return
		}
		targetChatID := resolveTargetGroupChatID(c.Message)
		if targetChatID == 0 {
			_ = c.Bot.SendMessage(c.Message.Chat, "Сначала выбери группу через /settings.", nil)
			return
		}

		payload := extractCommandPayload(c.Message.Text, "/setschedule")
		if payload == "" {
			_ = c.Bot.SendMessage(c.Message.Chat, "Использование: /setschedule template_name|HH:MM или /setschedule template_name|1,3,5|HH:MM", nil)
			return
		}

		templateName, cronExpr, err := postgres.ParseScheduleSpec(payload)
		if err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, err.Error(), nil)
			return
		}

		_, err = store.UpsertSchedule(context.Background(), targetChatID, templateName, cronExpr)
		if err != nil {
			_ = c.Bot.SendMessage(c.Message.Chat, "Ошибка сохранения расписания: "+err.Error(), nil)
			return
		}

		_ = c.Bot.SendMessage(c.Message.Chat, fmt.Sprintf("Расписание для %s сохранено: %s", templateName, postgres.FormatScheduleExprForDisplay(cronExpr)), nil)
	})

	bot.Handle(tele.Default, func(c tele.Context) {
		handleStatefulText(c.Bot, store, c.Message)
	})
}

func handleStatefulText(bot *tele.Bot, store *postgres.Store, message tele.Message) {
	chat := message.Chat
	text := strings.TrimSpace(message.Text)
	if text == "" {
		return
	}

	state := sessions.get(chat.ID)
	targetChatID := resolveTargetGroupChatID(message)
	if state.TargetGroupChatID != 0 {
		targetChatID = state.TargetGroupChatID
	}

	switch state.Action {
	case actionSelectTemplate:
		_ = bot.SendMessage(chat, "Выбери шаблон кнопкой из списка в сообщении выше.", nil)

	case actionTemplateEditQuestion:
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу через /settings.", nil)
			return
		}
		if strings.TrimSpace(state.SelectedTemplate) == "" {
			_ = bot.SendMessage(chat, "Сначала выбери шаблон.", nil)
			return
		}
		template, err := store.GetTemplateByName(context.Background(), targetChatID, state.SelectedTemplate)
		if err != nil {
			_ = bot.SendMessage(chat, "Ошибка чтения шаблона: "+err.Error(), nil)
			return
		}
		_, err = store.UpsertPollTemplate(context.Background(), targetChatID, template.Name, text, template.Options)
		if err != nil {
			_ = bot.SendMessage(chat, "Ошибка сохранения вопроса: "+err.Error(), nil)
			return
		}
		state.Action = actionNone
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Вопрос шаблона обновлен.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, message.Sender))

	case actionPollName:
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу через /settings.", nil)
			return
		}
		state.Action = actionPollQuestion
		state.PollName = text
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи вопрос голосования:", nil)

	case actionPollQuestion:
		state.Action = actionPollOptions
		state.PollQuestion = text
		state.PollOptions = nil
		sessions.set(chat.ID, state)
		sendPollOptionsPrompt(bot, chat, state.PollOptions)

	case actionPollOptions:
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу через /settings.", nil)
			return
		}
		option := strings.TrimSpace(text)
		if option == "" {
			_ = bot.SendMessage(chat, "Вариант не должен быть пустым.", nil)
			return
		}
		state.PollOptions = append(state.PollOptions, option)
		sessions.set(chat.ID, state)
		sendPollOptionsPrompt(bot, chat, state.PollOptions)

	case actionSchedulePickDays:
		_ = bot.SendMessage(chat, "Выбери дни недели кнопками и нажми «Дни готовы».", nil)

	case actionScheduleTime:
		processScheduleTimeInput(bot, store, chat, text, state)

	case actionScheduleEditTime:
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу через /settings.", nil)
			return
		}
		if _, err := time.Parse("15:04", text); err != nil {
			_ = bot.SendMessage(chat, "Неверный формат времени. Используй HH:MM (например: 19:30).", nil)
			return
		}
		if state.ManageScheduleID == 0 || len(state.ScheduleDays) == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери расписание для изменения.", nil)
			return
		}
		scheduleExpr, err := postgres.BuildScheduleExpr(state.ScheduleDays, text)
		if err != nil {
			_ = bot.SendMessage(chat, "Ошибка формата расписания: "+err.Error(), nil)
			return
		}
		if err := store.UpdateScheduleExpr(context.Background(), targetChatID, state.ManageScheduleID, scheduleExpr); err != nil {
			_ = bot.SendMessage(chat, "Ошибка изменения расписания: "+err.Error(), nil)
			return
		}
		state.Action = actionNone
		state.ManageScheduleID = 0
		state.ScheduleDays = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Расписание изменено: "+postgres.FormatScheduleExprForDisplay(scheduleExpr), nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, message.Sender))

	case actionEventName:
		name := strings.TrimSpace(text)
		if name == "" {
			_ = bot.SendMessage(chat, "Название события не должно быть пустым.", nil)
			return
		}
		state.EventName = name
		state.EventStartDay = 0
		state.EventPublishTime = ""
		state.EventStartTime = ""
		state.Action = actionNone
		sessions.set(chat.ID, state)
		sendEventStartDayPicker(bot, chat)

	case actionEventPublishTime:
		if _, err := time.Parse("15:04", text); err != nil {
			_ = bot.SendMessage(chat, "Неверный формат времени. Используй HH:MM (например: 18:00).", nil)
			return
		}
		state.EventPublishTime = text
		state.Action = actionEventStartTime
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи время начала события в формате HH:MM:", nil)

	case actionEventStartTime:
		if _, err := time.Parse("15:04", text); err != nil {
			_ = bot.SendMessage(chat, "Неверный формат времени. Используй HH:MM (например: 19:30).", nil)
			return
		}
		state.EventStartTime = text
		state.Action = actionEventEndTime
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи время окончания события в формате HH:MM:", nil)

	case actionEventEndTime:
		endParsed, err := time.Parse("15:04", text)
		if err != nil {
			_ = bot.SendMessage(chat, "Неверный формат времени. Используй HH:MM (например: 21:00).", nil)
			return
		}
		startParsed, _ := time.Parse("15:04", state.EventStartTime)
		if !endParsed.After(startParsed) {
			_ = bot.SendMessage(chat, "Время окончания должно быть позже начала в тот же день.", nil)
			return
		}
		state.EventEndTime = text
		state.Action = actionEventCost
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи стоимость события в рублях (например: 4000). Это необязательно: отправь \"-\" чтобы пропустить.", nil)

	case actionEventCost:
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу через /settings.", nil)
			return
		}
		var cost *float64
		if strings.TrimSpace(text) != "-" {
			parsed, err := parseEventCost(text)
			if err != nil {
				_ = bot.SendMessage(chat, err.Error(), nil)
				return
			}
			cost = &parsed
		}

		if _, err := store.CreateEvent(
			context.Background(),
			targetChatID,
			state.EventName,
			"training",
			state.EventStartDay,
			state.EventStartDay,
			state.EventPublishTime,
			state.EventStartTime,
			state.EventEndTime,
			"",
			false,
			60,
			false,
			false,
			6,
			0,
			180,
			false,
			true,
			false,
			true,
			cost,
		); err != nil {
			_ = bot.SendMessage(chat, "Ошибка сохранения события: "+err.Error(), nil)
			return
		}

		createdMsg := fmt.Sprintf(
			"Событие сохранено: %s\nДень: %s\nПубликация опроса: %s\nНачало: %s\nКонец: %s",
			state.EventName,
			weekdayLabel(state.EventStartDay),
			state.EventPublishTime,
			state.EventStartTime,
			state.EventEndTime,
		)
		if cost != nil {
			createdMsg += fmt.Sprintf("\nСтоимость: %.2f ₽", *cost)
		} else {
			createdMsg += "\nСтоимость: не указана"
		}
		state.Action = actionNone
		state.EventName = ""
		state.EventStartDay = 0
		state.EventPublishTime = ""
		state.EventStartTime = ""
		state.EventEndTime = ""
		state.EventCostAmount = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, createdMsg, nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, message.Sender))

	case actionEventCountedOptions:
		_ = bot.SendMessage(chat, "Используй кнопки выбора вариантов и «Сохранить».", nil)

	case actionEventEditCost:
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу через /settings.", nil)
			return
		}
		if state.EventBindEventID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери событие в «Редактировать событие».", nil)
			return
		}
		var cost *float64
		if strings.TrimSpace(text) != "-" {
			parsed, err := parseEventCost(text)
			if err != nil {
				_ = bot.SendMessage(chat, err.Error(), nil)
				return
			}
			cost = &parsed
		}
		if err := store.UpdateEventCostAmount(context.Background(), targetChatID, state.EventBindEventID, cost); err != nil {
			_ = bot.SendMessage(chat, "Ошибка сохранения стоимости: "+err.Error(), nil)
			return
		}

		changed := "не указана"
		if cost != nil {
			changed = fmt.Sprintf("%.2f ₽", *cost)
		}
		state.Action = actionNone
		state.EventBindEventID = 0
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Стоимость события обновлена: "+changed, nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, message.Sender))
	}
}

func processScheduleTimeInput(bot *tele.Bot, _ *postgres.Store, chat tele.Chat, text string, state sessionState) {
	if _, err := time.Parse("15:04", text); err != nil {
		_ = bot.SendMessage(chat, "Неверный формат времени. Используй HH:MM (например: 19:30).", nil)
		return
	}
	if state.SchedulePendingDay < 1 || state.SchedulePendingDay > 7 {
		_ = bot.SendMessage(chat, "Сначала выбери день недели кнопкой.", nil)
		return
	}
	if !containsInt(state.ScheduleDays, state.SchedulePendingDay) {
		_ = bot.SendMessage(chat, "Этот день не выбран. Выбери день еще раз кнопкой.", nil)
		return
	}

	if state.ScheduleTimes == nil {
		state.ScheduleTimes = make(map[int]string)
	}
	state.ScheduleTimes[state.SchedulePendingDay] = text
	state.SchedulePendingDay = 0
	state.Action = actionSchedulePickDays
	sessions.set(chat.ID, state)
	sendScheduleDaysPicker(bot, chat, state.ScheduleDays, state.ScheduleTimes)
}

func sendSettingsMenu(bot *tele.Bot, chat tele.Chat, selectedGroupTitle string) {
	state := sessions.get(chat.ID)
	selectedTemplate := state.SelectedTemplate
	if state.MenuSection == "" {
		options := &tele.SendOptions{
			ReplyMarkup: tele.ReplyMarkup{
				InlineKeyboard: [][]tele.KeyboardButton{
					{
						{Text: "Сменить группу", Data: "settings:group_selector"},
					},
					{
						{Text: "Опросы", Data: "settings:section_polls"},
						{Text: "События", Data: "settings:section_events"},
					},
				},
			},
		}
		_ = bot.SendMessage(chat, fmt.Sprintf("Главное меню\nГруппа: %s", selectedGroupTitle), options)
		return
	}

	if state.MenuSection == "events" {
		sendEventsMenu(bot, chat, selectedGroupTitle)
		return
	}

	if selectedTemplate == "" {
		options := &tele.SendOptions{
			ReplyMarkup: tele.ReplyMarkup{
				InlineKeyboard: [][]tele.KeyboardButton{
					{
						{Text: "Выбрать шаблон", Data: "settings:template_pick"},
						{Text: "Добавить шаблон", Data: "settings:poll_new"},
					},
					{
						{Text: "Показать настройки", Data: "settings:list"},
					},
					{
						{Text: "Главное меню", Data: "settings:main_menu"},
					},
				},
			},
		}
		_ = bot.SendMessage(chat, fmt.Sprintf("Опросы\nГруппа: %s", selectedGroupTitle), options)
		return
	}

	options := &tele.SendOptions{
		ReplyMarkup: tele.ReplyMarkup{
			InlineKeyboard: [][]tele.KeyboardButton{
				{
					{Text: "Редактировать шаблон", Data: "settings:template_edit"},
				},
				{
					{Text: "Отправить опрос", Data: "settings:poll_send_now"},
				},
				{
					{Text: "Новое расписание", Data: "settings:schedule_new"},
				},
				{
					{Text: "Управление расписаниями", Data: "settings:schedule_manage"},
				},
				{
					{Text: "Удалить шаблон", Data: "settings:template_delete"},
				},
				{
					{Text: "Главное меню", Data: "settings:main_menu"},
				},
			},
		},
	}

	_ = bot.SendMessage(chat, fmt.Sprintf("Опросы: шаблон\nГруппа: %s\nШаблон: %s", selectedGroupTitle, selectedTemplate), options)
}

func sendEventsMenu(bot *tele.Bot, chat tele.Chat, selectedGroupTitle string) {
	options := &tele.SendOptions{
		ReplyMarkup: tele.ReplyMarkup{
			InlineKeyboard: [][]tele.KeyboardButton{
				{
					{Text: "Создать событие", Data: "settings:event_new"},
				},
				{
					{Text: "Редактировать событие", Data: "settings:event_edit"},
				},
				{
					{Text: "Главное меню", Data: "settings:main_menu"},
				},
			},
		},
	}
	_ = bot.SendMessage(chat, fmt.Sprintf("События\nГруппа: %s", selectedGroupTitle), options)
}

func sendGroupSelectorMenu(bot *tele.Bot, store *postgres.Store, recipient tele.Chat, user tele.User) {
	groups, err := store.ListGroupsForAdmin(context.Background(), int64(user.ID))
	if err != nil {
		_ = bot.SendMessage(recipient, "Не удалось получить список групп: "+err.Error(), nil)
		return
	}
	if len(groups) == 0 {
		_ = bot.SendMessage(
			recipient,
			"Нет доступных групп.\nЕсли ты новый админ, попроси любого текущего админа в группе выполнить /updategroup.\nЕсли группа еще не подключена, выполни в группе /setgroup.",
			nil,
		)
		return
	}

	keyboard := make([][]tele.KeyboardButton, 0, len(groups))
	for _, group := range groups {
		keyboard = append(keyboard, []tele.KeyboardButton{{
			Text: formatGroupTitle(group.ChatID, group.Title),
			Data: fmt.Sprintf("settings:pick_group:%d", group.ChatID),
		}})
	}

	options := &tele.SendOptions{ReplyMarkup: tele.ReplyMarkup{InlineKeyboard: keyboard}}
	_ = bot.SendMessage(recipient, "Выбери группу для настройки:", options)
}

func parseOptions(payload string) []string {
	rawOptions := strings.Split(payload, ",")
	options := make([]string, 0, len(rawOptions))
	for _, option := range rawOptions {
		option = strings.TrimSpace(option)
		if option != "" {
			options = append(options, option)
		}
	}
	return options
}

func sendPollOptionsPrompt(bot *tele.Bot, chat tele.Chat, options []string) {
	text := "Вводи варианты по одному сообщению.\nПример: отправь \"Буду\", затем отдельным сообщением \"Не буду\"."
	if len(options) > 0 {
		text += fmt.Sprintf("\n\nСейчас вариантов: %d", len(options))
	}

	menu := &tele.SendOptions{
		ReplyMarkup: tele.ReplyMarkup{
			InlineKeyboard: [][]tele.KeyboardButton{
				{
					{Text: "Готово", Data: "settings:poll_options_done"},
					{Text: "Отмена", Data: "settings:poll_options_cancel"},
				},
			},
		},
	}
	_ = bot.SendMessage(chat, text, menu)
}

func sendScheduleDaysPicker(bot *tele.Bot, chat tele.Chat, selectedDays []int, dayTimes map[int]string) {
	text := "Выбери дни недели. Нажимай для включения/выключения, затем «Дни готовы»."
	text += "\nСейчас: " + formatSelectedDaysWithTimes(selectedDays, dayTimes)

	menu := &tele.SendOptions{
		ReplyMarkup: tele.ReplyMarkup{
			InlineKeyboard: [][]tele.KeyboardButton{
				{
					{Text: dayButtonLabel(1, selectedDays, "Пн"), Data: "settings:schedule_day:1"},
					{Text: dayButtonLabel(2, selectedDays, "Вт"), Data: "settings:schedule_day:2"},
					{Text: dayButtonLabel(3, selectedDays, "Ср"), Data: "settings:schedule_day:3"},
					{Text: dayButtonLabel(4, selectedDays, "Чт"), Data: "settings:schedule_day:4"},
				},
				{
					{Text: dayButtonLabel(5, selectedDays, "Пт"), Data: "settings:schedule_day:5"},
					{Text: dayButtonLabel(6, selectedDays, "Сб"), Data: "settings:schedule_day:6"},
					{Text: dayButtonLabel(7, selectedDays, "Вс"), Data: "settings:schedule_day:7"},
				},
				{
					{Text: "Дни готовы", Data: "settings:schedule_days_done"},
					{Text: "Отмена", Data: "settings:schedule_days_cancel"},
				},
			},
		},
	}
	_ = bot.SendMessage(chat, text, menu)
}

func sendEventStartDayPicker(bot *tele.Bot, chat tele.Chat) {
	menu := &tele.SendOptions{
		ReplyMarkup: tele.ReplyMarkup{
			InlineKeyboard: [][]tele.KeyboardButton{
				{
					{Text: "Пн", Data: "settings:event_start_day:1"},
					{Text: "Вт", Data: "settings:event_start_day:2"},
					{Text: "Ср", Data: "settings:event_start_day:3"},
					{Text: "Чт", Data: "settings:event_start_day:4"},
				},
				{
					{Text: "Пт", Data: "settings:event_start_day:5"},
					{Text: "Сб", Data: "settings:event_start_day:6"},
					{Text: "Вс", Data: "settings:event_start_day:7"},
				},
				{
					{Text: "Отмена", Data: "settings:event_cancel"},
				},
			},
		},
	}
	_ = bot.SendMessage(chat, "Выбери день недели для начала события:", menu)
}

func formatSelectedDaysWithTimes(selectedDays []int, dayTimes map[int]string) string {
	if len(selectedDays) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(selectedDays))
	for _, day := range selectedDays {
		label := weekdayLabel(day)
		if t := strings.TrimSpace(dayTimes[day]); t != "" {
			label += " " + t
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, ", ")
}

func dayButtonLabel(day int, selected []int, label string) string {
	if containsInt(selected, day) {
		return "✅ " + label
	}
	return label
}

func weekdayLabel(day int) string {
	switch day {
	case 1:
		return "пн"
	case 2:
		return "вт"
	case 3:
		return "ср"
	case 4:
		return "чт"
	case 5:
		return "пт"
	case 6:
		return "сб"
	case 7:
		return "вс"
	default:
		return fmt.Sprintf("day_%d", day)
	}
}

func isoWeekday(day time.Weekday) int {
	if day == time.Sunday {
		return 7
	}
	return int(day)
}

func parseWeekdaysInput(payload string) ([]int, error) {
	normalized := strings.NewReplacer(
		"понедельник", "1",
		"пн", "1",
		"вторник", "2",
		"вт", "2",
		"среда", "3",
		"ср", "3",
		"четверг", "4",
		"чт", "4",
		"пятница", "5",
		"пт", "5",
		"суббота", "6",
		"сб", "6",
		"воскресенье", "7",
		"вс", "7",
		" ", "",
	).Replace(strings.ToLower(payload))

	return postgres.ParseWeekdaysCSV(normalized)
}

func parseEventCost(value string) (float64, error) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", ".")
	cost, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("неверный формат стоимости. Пример: 4000 или 3500.50")
	}
	if cost < 0 {
		return 0, fmt.Errorf("стоимость не может быть отрицательной")
	}
	return cost, nil
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

func resolveTargetGroupChatID(message tele.Message) int64 {
	if isPrivateChat(message.Chat) {
		return sessions.get(message.Chat.ID).TargetGroupChatID
	}
	return message.Chat.ID
}

func setTargetGroup(sessionChatID, targetGroupChatID int64) {
	state := sessions.get(sessionChatID)
	state.TargetGroupChatID = targetGroupChatID
	sessions.set(sessionChatID, state)
}

func isPrivateChat(chat tele.Chat) bool {
	return chat.Type == tele.ChatPrivate
}

func formatGroupTitle(chatID int64, title string) string {
	if strings.TrimSpace(title) != "" {
		return title
	}
	return fmt.Sprintf("chat_%d", chatID)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func syncGroupAdmins(bot *tele.Bot, store *postgres.Store, chat tele.Chat) error {
	admins, err := bot.GetChatAdministrators(chat)
	if err != nil {
		return err
	}

	input := make([]postgres.AdminMember, 0, len(admins))
	for _, admin := range admins {
		if admin.User.IsBot {
			continue
		}
		input = append(input, postgres.AdminMember{
			UserID:    int64(admin.User.ID),
			Username:  admin.User.Username,
			FirstName: admin.User.FirstName,
			LastName:  admin.User.LastName,
			Role:      admin.Status,
		})
	}

	return store.SyncGroupAdmins(context.Background(), chat.ID, input)
}

func ensureGroupAdmin(bot *tele.Bot, message tele.Message) bool {
	if isPrivateChat(message.Chat) {
		return true
	}

	admins, err := bot.GetChatAdministrators(message.Chat)
	if err != nil {
		_ = bot.SendMessage(message.Chat, "Не удалось проверить права администратора.", nil)
		return false
	}

	for _, admin := range admins {
		if admin.User.ID == message.Sender.ID {
			return true
		}
	}

	_ = bot.SendMessage(message.Chat, "Эта команда доступна только администраторам группы.", nil)
	return false
}

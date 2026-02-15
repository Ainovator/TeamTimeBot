package telegram

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

func HandleCallback(bot *tele.Bot, store *postgres.Store, callback tele.Callback) {
	_ = bot.AnswerCallbackQuery(&callback, &tele.CallbackResponse{})

	chat := callback.Message.Chat
	data := strings.TrimSpace(callback.Data)

	switch {
	case data == "settings:main_menu":
		state := sessions.get(chat.ID)
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
		state.EventBindEventID = 0
		state.EventOptionCount = 0
		state.EventCountedOptions = nil
		sessions.set(chat.ID, state)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:template_menu":
		state := sessions.get(chat.ID)
		state.Action = actionNone
		state.MenuSection = "polls"
		state.EventBindEventID = 0
		state.AvailableTemplates = nil
		sessions.set(chat.ID, state)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:section_polls":
		state := sessions.get(chat.ID)
		state.Action = actionNone
		state.MenuSection = "polls"
		state.EventName = ""
		state.EventStartDay = 0
		state.EventPublishTime = ""
		state.EventStartTime = ""
		state.EventEndTime = ""
		state.EventCostAmount = nil
		state.EventBindEventID = 0
		state.EventOptionCount = 0
		state.EventCountedOptions = nil
		sessions.set(chat.ID, state)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:section_events":
		state := sessions.get(chat.ID)
		state.Action = actionNone
		state.MenuSection = "events"
		state.SelectedTemplate = ""
		state.EventStartDay = 0
		state.EventPublishTime = ""
		state.EventStartTime = ""
		state.EventEndTime = ""
		state.EventCostAmount = nil
		state.EventBindEventID = 0
		state.EventOptionCount = 0
		state.EventCountedOptions = nil
		sessions.set(chat.ID, state)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:group_selector":
		sendGroupSelectorMenu(bot, store, chat, callback.Sender)

	case strings.HasPrefix(data, "settings:pick_group:"):
		if !isPrivateChat(chat) {
			_ = bot.SendMessage(chat, "Выбор группы доступен из личного чата с ботом.", nil)
			return
		}

		chatIDText := strings.TrimPrefix(data, "settings:pick_group:")
		targetChatID, err := strconv.ParseInt(chatIDText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный идентификатор группы.", nil)
			return
		}

		groups, err := store.ListGroupsForAdmin(context.Background(), int64(callback.Sender.ID))
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось проверить доступ к группам: "+err.Error(), nil)
			return
		}

		selectedTitle := ""
		for _, group := range groups {
			if group.ChatID == targetChatID {
				selectedTitle = formatGroupTitle(group.ChatID, group.Title)
				break
			}
		}
		if selectedTitle == "" {
			_ = bot.SendMessage(chat, "У тебя нет доступа к этой группе. Выполни /updategroup в группе и попробуй снова.", nil)
			return
		}

		setTargetGroup(chat.ID, targetChatID)
		state := sessions.get(chat.ID)
		state.Action = actionNone
		state.MenuSection = ""
		state.SelectedTemplate = ""
		state.AvailableTemplates = nil
		sessions.set(chat.ID, state)
		sendSettingsMenu(bot, chat, selectedTitle)

	case data == "settings:template_pick":
		state := sessions.get(chat.ID)
		state.MenuSection = "polls"
		sessions.set(chat.ID, state)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		templateNames, err := store.ListTemplateNamesByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить шаблоны: "+err.Error(), nil)
			return
		}
		if len(templateNames) == 0 {
			_ = bot.SendMessage(chat, "В этой группе пока нет шаблонов. Нажми «Добавить шаблон».", nil)
			sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))
			return
		}

		state.Action = actionSelectTemplate
		state.AvailableTemplates = templateNames
		sessions.set(chat.ID, state)

		keyboard := make([][]tele.KeyboardButton, 0, len(templateNames))
		for idx, name := range templateNames {
			keyboard = append(keyboard, []tele.KeyboardButton{
				{Text: name, Data: fmt.Sprintf("settings:pick_template:%d", idx)},
			})
		}
		options := &tele.SendOptions{
			ReplyMarkup: tele.ReplyMarkup{
				InlineKeyboard: keyboard,
			},
		}
		_ = bot.SendMessage(chat, "Выбери шаблон:", options)

	case strings.HasPrefix(data, "settings:pick_template:"):
		state := sessions.get(chat.ID)
		if len(state.AvailableTemplates) == 0 {
			_ = bot.SendMessage(chat, "Список шаблонов устарел. Нажми «Выбрать шаблон» еще раз.", nil)
			return
		}

		idxText := strings.TrimPrefix(data, "settings:pick_template:")
		idx, err := strconv.Atoi(idxText)
		if err != nil || idx < 0 || idx >= len(state.AvailableTemplates) {
			_ = bot.SendMessage(chat, "Некорректный выбор шаблона.", nil)
			return
		}

		state.Action = actionNone
		state.SelectedTemplate = state.AvailableTemplates[idx]
		state.AvailableTemplates = nil
		sessions.set(chat.ID, state)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:poll_new":
		state := sessions.get(chat.ID)
		if isPrivateChat(chat) && state.TargetGroupChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		state.MenuSection = "polls"
		state.Action = actionPollName
		state.PollName = ""
		state.PollQuestion = ""
		state.PollOptions = nil
		state.AvailableTemplates = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Название шаблона (например: weekly):", nil)

	case data == "settings:poll_send_now":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		if strings.TrimSpace(state.SelectedTemplate) == "" {
			_ = bot.SendMessage(chat, "Сначала выбери шаблон.", nil)
			return
		}

		template, err := store.GetTemplateByName(context.Background(), targetChatID, state.SelectedTemplate)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось загрузить шаблон: "+err.Error(), nil)
			return
		}
		if len(template.Options) < 2 {
			_ = bot.SendMessage(chat, "У шаблона должно быть минимум 2 варианта.", nil)
			return
		}

		target := tele.Chat{ID: targetChatID, Type: tele.ChatGroup}
		sent, err := bot.SendPollWithMeta(target, template.Question, template.Options, nil)
		if err != nil {
			_ = bot.SendMessage(chat, "Ошибка отправки опроса: "+err.Error(), nil)
			return
		}
		if sent != nil {
			var eventID *uint64
			group, groupErr := store.GetGroupByChatID(context.Background(), targetChatID)
			if groupErr == nil {
				loc, locErr := time.LoadLocation(group.Timezone)
				if locErr == nil {
					weekday := isoWeekday(time.Now().UTC().In(loc).Weekday())
					foundEventID, findErr := store.FindBoundEventIDByTemplateAndWeekday(context.Background(), targetChatID, template.Name, weekday)
					if findErr == nil {
						eventID = foundEventID
					}
				}
			}
			if _, err := store.CreateEventPollPost(
				context.Background(),
				targetChatID,
				eventID,
				template.Name,
				sent.MessageID,
				sent.PollID,
			); err != nil {
				_ = bot.SendMessage(chat, "Опрос отправлен, но не удалось сохранить публикацию: "+err.Error(), nil)
				sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))
				return
			}
		}
		_ = bot.SendMessage(chat, "Опрос отправлен в группу.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:poll_options_cancel":
		state := sessions.get(chat.ID)
		state.Action = actionNone
		state.PollName = ""
		state.PollQuestion = ""
		state.PollOptions = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Настройка шаблона отменена.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:poll_options_done":
		state := sessions.get(chat.ID)
		if state.Action != actionPollOptions {
			_ = bot.SendMessage(chat, "Сейчас нет активной настройки шаблона.", nil)
			return
		}
		if len(state.PollOptions) < 2 {
			_ = bot.SendMessage(chat, "Нужно минимум 2 варианта. Добавь еще варианты и нажми «Готово».", nil)
			return
		}

		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		_, err := store.UpsertPollTemplate(context.Background(), targetChatID, state.PollName, state.PollQuestion, state.PollOptions)
		if err != nil {
			_ = bot.SendMessage(chat, "Ошибка сохранения шаблона: "+err.Error(), nil)
			return
		}

		state.Action = actionNone
		state.SelectedTemplate = state.PollName
		state.PollName = ""
		state.PollQuestion = ""
		state.PollOptions = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, fmt.Sprintf("Шаблон %s сохранен и выбран.", state.SelectedTemplate), nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:event_new":
		state := sessions.get(chat.ID)
		if isPrivateChat(chat) && state.TargetGroupChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		state.MenuSection = "events"
		state.Action = actionEventName
		state.EventName = ""
		state.EventStartDay = 0
		state.EventStartTime = ""
		state.EventEndTime = ""
		state.EventCostAmount = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Название события:", nil)

	case strings.HasPrefix(data, "settings:event_start_day:"):
		state := sessions.get(chat.ID)
		if state.Action != actionNone || strings.TrimSpace(state.EventName) == "" {
			_ = bot.SendMessage(chat, "Сначала нажми «Создать событие».", nil)
			return
		}
		dayText := strings.TrimPrefix(data, "settings:event_start_day:")
		day, err := strconv.Atoi(dayText)
		if err != nil || day < 1 || day > 7 {
			_ = bot.SendMessage(chat, "Некорректный день недели.", nil)
			return
		}
		state.Action = actionEventPublishTime
		state.EventStartDay = day
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи время публикации опроса в формате HH:MM (например: 18:00):", nil)

	case data == "settings:event_cancel":
		state := sessions.get(chat.ID)
		state.Action = actionNone
		state.EventName = ""
		state.EventStartDay = 0
		state.EventPublishTime = ""
		state.EventStartTime = ""
		state.EventEndTime = ""
		state.EventCostAmount = nil
		state.EventBindEventID = 0
		state.EventOptionCount = 0
		state.EventCountedOptions = nil
		state.AvailableTemplates = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Операция отменена.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:event_bind_poll":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		events, err := store.ListEventsByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить события: "+err.Error(), nil)
			return
		}
		if len(events) == 0 {
			_ = bot.SendMessage(chat, "Сначала создай событие.", nil)
			sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))
			return
		}

		keyboard := make([][]tele.KeyboardButton, 0, len(events)+1)
		for _, event := range events {
			keyboard = append(keyboard, []tele.KeyboardButton{
				{Text: fmt.Sprintf("#%d %s (%s %s-%s)", event.ID, event.Name, weekdayLabel(event.StartWeekday), event.StartTime, event.EndTime), Data: fmt.Sprintf("settings:event_bind_pick:%d", event.ID)},
			})
		}
		keyboard = append(keyboard, []tele.KeyboardButton{
			{Text: "Отмена", Data: "settings:event_cancel"},
		})
		options := &tele.SendOptions{ReplyMarkup: tele.ReplyMarkup{InlineKeyboard: keyboard}}
		_ = bot.SendMessage(chat, "Выбери событие для привязки опроса:", options)

	case data == "settings:event_edit":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		events, err := store.ListEventsByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить события: "+err.Error(), nil)
			return
		}
		if len(events) == 0 {
			_ = bot.SendMessage(chat, "Пока нет событий. Нажми «Создать событие».", nil)
			sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))
			return
		}

		keyboard := make([][]tele.KeyboardButton, 0, len(events)+1)
		for _, event := range events {
			poll := "не привязан"
			if strings.TrimSpace(event.PollTemplate) != "" {
				poll = event.PollTemplate
			}
			keyboard = append(keyboard, []tele.KeyboardButton{
				{Text: fmt.Sprintf("#%d %s (%s публ:%s %s-%s) | %s", event.ID, event.Name, weekdayLabel(event.StartWeekday), event.PollPublishTime, event.StartTime, event.EndTime, poll), Data: fmt.Sprintf("settings:event_edit_pick:%d", event.ID)},
			})
		}
		keyboard = append(keyboard, []tele.KeyboardButton{{Text: "Назад", Data: "settings:section_events"}})
		options := &tele.SendOptions{ReplyMarkup: tele.ReplyMarkup{InlineKeyboard: keyboard}}
		_ = bot.SendMessage(chat, "Выбери событие для редактирования:", options)

	case strings.HasPrefix(data, "settings:event_edit_pick:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		idText := strings.TrimPrefix(data, "settings:event_edit_pick:")
		eventID, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный идентификатор события.", nil)
			return
		}
		events, err := store.ListEventsByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить событие: "+err.Error(), nil)
			return
		}
		var selected *postgres.EventView
		for i := range events {
			if events[i].ID == eventID {
				selected = &events[i]
				break
			}
		}
		if selected == nil {
			_ = bot.SendMessage(chat, "Событие не найдено.", nil)
			return
		}
		state.EventBindEventID = eventID
		state.Action = actionNone
		sessions.set(chat.ID, state)
		sendEventEditMenu(bot, chat, *selected)

	case strings.HasPrefix(data, "settings:event_edit_bind_poll:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		idText := strings.TrimPrefix(data, "settings:event_edit_bind_poll:")
		eventID, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный идентификатор события.", nil)
			return
		}
		templateNames, err := store.ListTemplateNamesByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить шаблоны: "+err.Error(), nil)
			return
		}
		if len(templateNames) == 0 {
			_ = bot.SendMessage(chat, "В этой группе пока нет шаблонов. Сначала добавь шаблон в разделе «Опросы».", nil)
			return
		}
		state.Action = actionEventBindTemplate
		state.EventBindEventID = eventID
		state.AvailableTemplates = templateNames
		sessions.set(chat.ID, state)
		keyboard := make([][]tele.KeyboardButton, 0, len(templateNames)+1)
		for idx, name := range templateNames {
			keyboard = append(keyboard, []tele.KeyboardButton{
				{Text: name, Data: fmt.Sprintf("settings:event_bind_template:%d", idx)},
			})
		}
		keyboard = append(keyboard, []tele.KeyboardButton{{Text: "Отмена", Data: "settings:event_cancel"}})
		options := &tele.SendOptions{ReplyMarkup: tele.ReplyMarkup{InlineKeyboard: keyboard}}
		_ = bot.SendMessage(chat, "Выбери шаблон опроса для события:", options)

	case strings.HasPrefix(data, "settings:event_edit_counted_options:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		idText := strings.TrimPrefix(data, "settings:event_edit_counted_options:")
		eventID, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный идентификатор события.", nil)
			return
		}
		details, err := loadEventTemplateDetails(context.Background(), store, targetChatID, eventID)
		if err != nil {
			_ = bot.SendMessage(chat, "Сначала привяжи шаблон к событию, затем настраивай учет голосов.", nil)
			return
		}
		current, err := store.GetEventCountedOptions(context.Background(), eventID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить текущую настройку: "+err.Error(), nil)
			return
		}
		state.Action = actionEventCountedOptions
		state.EventBindEventID = eventID
		state.EventOptionCount = len(details.TemplateOptions)
		state.EventCountedOptions = current
		sessions.set(chat.ID, state)
		sendEventCountedOptionsPicker(bot, chat, details, current)

	case strings.HasPrefix(data, "settings:event_edit_cost:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		idText := strings.TrimPrefix(data, "settings:event_edit_cost:")
		eventID, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный идентификатор события.", nil)
			return
		}
		state.Action = actionEventEditCost
		state.EventBindEventID = eventID
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи новую стоимость события в рублях (например: 4000). Отправь \"-\" чтобы очистить.", nil)

	case strings.HasPrefix(data, "settings:event_calc:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		idText := strings.TrimPrefix(data, "settings:event_calc:")
		eventID, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный идентификатор события.", nil)
			return
		}

		events, err := store.ListEventsByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить событие: "+err.Error(), nil)
			return
		}
		var selected *postgres.EventView
		for i := range events {
			if events[i].ID == eventID {
				selected = &events[i]
				break
			}
		}
		if selected == nil {
			_ = bot.SendMessage(chat, "Событие не найдено.", nil)
			return
		}
		if strings.TrimSpace(selected.PollTemplate) == "" {
			_ = bot.SendMessage(chat, "Сначала привяжи опрос к событию.", nil)
			return
		}

		post, err := store.GetLatestEventPollPost(context.Background(), eventID)
		if err != nil {
			_ = bot.SendMessage(chat, "Ошибка при поиске последнего опроса: "+err.Error(), nil)
			return
		}
		if post == nil {
			_ = bot.SendMessage(chat, "Для этого события пока нет опубликованных опросов.", nil)
			return
		}

		counted, err := store.GetEventCountedOptions(context.Background(), eventID)
		if err != nil {
			_ = bot.SendMessage(chat, "Ошибка чтения настройки учитываемых голосов: "+err.Error(), nil)
			return
		}
		if len(counted) == 0 {
			counted = []int{0}
		}
		choices := make([]string, 0, len(counted))
		human := make([]string, 0, len(counted))
		for _, idx := range counted {
			choices = append(choices, fmt.Sprintf("option_%d", idx))
			human = append(human, strconv.Itoa(idx+1))
		}
		participants, err := store.CountVotesForPostChoices(context.Background(), post.ID, choices)
		if err != nil {
			_ = bot.SendMessage(chat, "Ошибка подсчета голосов: "+err.Error(), nil)
			return
		}

		total := 4000.0
		if selected.CostAmount != nil {
			total = *selected.CostAmount
		}
		perPerson := 0.0
		if participants > 0 {
			perPerson = math.Ceil(total / float64(participants))
		}

		text := fmt.Sprintf(
			"Тестовый расчет для события \"%s\"\nПоследний опрос: #%d (%s)\nУчитываем варианты: %s\nМест: %d\nСтоимость события: %.2f ₽\nЦена за место: %.0f ₽",
			selected.Name,
			post.ID,
			post.PublishedAt.Format("02.01.2006 15:04"),
			strings.Join(human, ","),
			participants,
			total,
			perPerson,
		)
		if participants == 0 {
			text += "\nНет участников по выбранным вариантам."
		}
		_ = bot.SendMessage(chat, text, nil)

	case data == "settings:event_counted_options":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		events, err := store.ListEventsByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить события: "+err.Error(), nil)
			return
		}
		if len(events) == 0 {
			_ = bot.SendMessage(chat, "Сначала создай событие.", nil)
			sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))
			return
		}

		keyboard := make([][]tele.KeyboardButton, 0, len(events)+1)
		for _, event := range events {
			keyboard = append(keyboard, []tele.KeyboardButton{
				{Text: fmt.Sprintf("#%d %s (%s %s-%s)", event.ID, event.Name, weekdayLabel(event.StartWeekday), event.StartTime, event.EndTime), Data: fmt.Sprintf("settings:event_count_pick:%d", event.ID)},
			})
		}
		keyboard = append(keyboard, []tele.KeyboardButton{{Text: "Отмена", Data: "settings:event_cancel"}})
		options := &tele.SendOptions{ReplyMarkup: tele.ReplyMarkup{InlineKeyboard: keyboard}}
		_ = bot.SendMessage(chat, "Выбери событие для настройки учитываемых голосов:", options)

	case strings.HasPrefix(data, "settings:event_count_pick:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		idText := strings.TrimPrefix(data, "settings:event_count_pick:")
		eventID, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный идентификатор события.", nil)
			return
		}

		details, err := loadEventTemplateDetails(context.Background(), store, targetChatID, eventID)
		if err != nil {
			_ = bot.SendMessage(chat, "Сначала привяжи шаблон к событию, затем настраивай учет голосов.", nil)
			return
		}

		current, err := store.GetEventCountedOptions(context.Background(), eventID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить текущую настройку: "+err.Error(), nil)
			return
		}

		state.Action = actionEventCountedOptions
		state.EventBindEventID = eventID
		state.EventOptionCount = len(details.TemplateOptions)
		state.EventCountedOptions = current
		sessions.set(chat.ID, state)
		sendEventCountedOptionsPicker(bot, chat, details, current)

	case strings.HasPrefix(data, "settings:event_count_toggle:"):
		state := sessions.get(chat.ID)
		if state.Action != actionEventCountedOptions || state.EventBindEventID == 0 || state.EventOptionCount <= 0 {
			_ = bot.SendMessage(chat, "Сначала выбери событие в «Учитывать голоса».", nil)
			return
		}
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		idxText := strings.TrimPrefix(data, "settings:event_count_toggle:")
		idx, err := strconv.Atoi(idxText)
		if err != nil || idx < 0 || idx >= state.EventOptionCount {
			_ = bot.SendMessage(chat, "Некорректный вариант.", nil)
			return
		}

		if containsInt(state.EventCountedOptions, idx) {
			next := make([]int, 0, len(state.EventCountedOptions))
			for _, value := range state.EventCountedOptions {
				if value != idx {
					next = append(next, value)
				}
			}
			state.EventCountedOptions = next
		} else {
			state.EventCountedOptions = append(state.EventCountedOptions, idx)
		}
		sessions.set(chat.ID, state)

		details, err := loadEventTemplateDetails(context.Background(), store, targetChatID, state.EventBindEventID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось загрузить шаблон события: "+err.Error(), nil)
			return
		}
		sendEventCountedOptionsPicker(bot, chat, details, state.EventCountedOptions)

	case data == "settings:event_count_save":
		state := sessions.get(chat.ID)
		if state.Action != actionEventCountedOptions || state.EventBindEventID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери событие в «Учитывать голоса».", nil)
			return
		}
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		if err := store.ReplaceEventCountedOptions(context.Background(), targetChatID, state.EventBindEventID, state.EventCountedOptions); err != nil {
			_ = bot.SendMessage(chat, "Ошибка сохранения настройки: "+err.Error(), nil)
			return
		}

		shown := "по умолчанию: 1"
		if len(state.EventCountedOptions) > 0 {
			human := make([]string, 0, len(state.EventCountedOptions))
			for _, idx := range state.EventCountedOptions {
				human = append(human, strconv.Itoa(idx+1))
			}
			shown = strings.Join(human, ",")
		}

		state.Action = actionNone
		state.EventBindEventID = 0
		state.EventOptionCount = 0
		state.EventCountedOptions = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Настройка сохранена. Учитываем варианты: "+shown, nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case strings.HasPrefix(data, "settings:event_bind_pick:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		idText := strings.TrimPrefix(data, "settings:event_bind_pick:")
		eventID, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный идентификатор события.", nil)
			return
		}

		templateNames, err := store.ListTemplateNamesByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить шаблоны: "+err.Error(), nil)
			return
		}
		if len(templateNames) == 0 {
			_ = bot.SendMessage(chat, "В этой группе пока нет шаблонов. Сначала добавь шаблон в разделе «Опросы».", nil)
			sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))
			return
		}

		state.Action = actionEventBindTemplate
		state.EventBindEventID = eventID
		state.AvailableTemplates = templateNames
		sessions.set(chat.ID, state)

		keyboard := make([][]tele.KeyboardButton, 0, len(templateNames)+1)
		for idx, name := range templateNames {
			keyboard = append(keyboard, []tele.KeyboardButton{
				{Text: name, Data: fmt.Sprintf("settings:event_bind_template:%d", idx)},
			})
		}
		keyboard = append(keyboard, []tele.KeyboardButton{
			{Text: "Отмена", Data: "settings:event_cancel"},
		})
		options := &tele.SendOptions{ReplyMarkup: tele.ReplyMarkup{InlineKeyboard: keyboard}}
		_ = bot.SendMessage(chat, "Выбери шаблон опроса для события:", options)

	case strings.HasPrefix(data, "settings:event_bind_template:"):
		state := sessions.get(chat.ID)
		if state.Action != actionEventBindTemplate || state.EventBindEventID == 0 || len(state.AvailableTemplates) == 0 {
			_ = bot.SendMessage(chat, "Сначала нажми «Привязать опрос».", nil)
			return
		}
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		idxText := strings.TrimPrefix(data, "settings:event_bind_template:")
		idx, err := strconv.Atoi(idxText)
		if err != nil || idx < 0 || idx >= len(state.AvailableTemplates) {
			_ = bot.SendMessage(chat, "Некорректный выбор шаблона.", nil)
			return
		}

		templateName := state.AvailableTemplates[idx]
		if err := store.BindEventToTemplate(context.Background(), targetChatID, state.EventBindEventID, templateName); err != nil {
			_ = bot.SendMessage(chat, "Не удалось привязать опрос: "+err.Error(), nil)
			return
		}

		state.Action = actionNone
		state.EventBindEventID = 0
		state.AvailableTemplates = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Опрос успешно привязан к событию.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:event_list":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		events, err := store.ListEventsByChatID(context.Background(), targetChatID)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось получить события: "+err.Error(), nil)
			return
		}
		if len(events) == 0 {
			_ = bot.SendMessage(chat, "Пока нет событий. Нажми «Создать событие».", nil)
			sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))
			return
		}
		var b strings.Builder
		b.WriteString("Список событий:\n")
		for _, event := range events {
			poll := "не привязан"
			if strings.TrimSpace(event.PollTemplate) != "" {
				poll = event.PollTemplate
			}
			counted, err := store.GetEventCountedOptions(context.Background(), event.ID)
			countedText := "1"
			if err == nil && len(counted) > 0 {
				human := make([]string, 0, len(counted))
				for _, idx := range counted {
					human = append(human, strconv.Itoa(idx+1))
				}
				countedText = strings.Join(human, ",")
			}
			costText := "не указана"
			if event.CostAmount != nil {
				costText = fmt.Sprintf("%.2f ₽", *event.CostAmount)
			}
			b.WriteString(fmt.Sprintf("- #%d %s: %s публ:%s %s-%s | стоимость: %s | опрос: %s | учитывать: %s\n", event.ID, event.Name, weekdayLabel(event.StartWeekday), event.PollPublishTime, event.StartTime, event.EndTime, costText, poll, countedText))
		}
		_ = bot.SendMessage(chat, strings.TrimSpace(b.String()), nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:schedule_new":
		state := sessions.get(chat.ID)
		if isPrivateChat(chat) && state.TargetGroupChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if strings.TrimSpace(state.SelectedTemplate) == "" {
			templateNames, err := store.ListTemplateNamesByChatID(context.Background(), targetChatID)
			if err == nil && len(templateNames) > 0 {
				_ = bot.SendMessage(chat, "Сначала выбери шаблон кнопкой «Выбрать шаблон». Доступные: "+strings.Join(templateNames, ", "), nil)
			} else {
				_ = bot.SendMessage(chat, "Сначала добавь и выбери шаблон.", nil)
			}
			return
		}

		state.Action = actionSchedulePickDays
		state.ScheduleName = state.SelectedTemplate
		state.ScheduleDays = nil
		state.ScheduleTimes = make(map[int]string)
		state.SchedulePendingDay = 0
		sessions.set(chat.ID, state)
		sendScheduleDaysPicker(bot, chat, state.ScheduleDays, state.ScheduleTimes)

	case strings.HasPrefix(data, "settings:schedule_day:"):
		state := sessions.get(chat.ID)
		if state.Action == actionScheduleTime && state.SchedulePendingDay != 0 {
			_ = bot.SendMessage(chat, "Сначала введи время HH:MM для "+weekdayLabel(state.SchedulePendingDay)+".", nil)
			return
		}
		if state.Action != actionSchedulePickDays {
			_ = bot.SendMessage(chat, "Сначала нажми «Новое расписание».", nil)
			return
		}
		dayText := strings.TrimPrefix(data, "settings:schedule_day:")
		day, err := strconv.Atoi(dayText)
		if err != nil || day < 1 || day > 7 {
			_ = bot.SendMessage(chat, "Некорректный день недели.", nil)
			return
		}

		if state.ScheduleTimes == nil {
			state.ScheduleTimes = make(map[int]string)
		}
		if containsInt(state.ScheduleDays, day) {
			next := make([]int, 0, len(state.ScheduleDays))
			for _, d := range state.ScheduleDays {
				if d != day {
					next = append(next, d)
				}
			}
			state.ScheduleDays = next
			delete(state.ScheduleTimes, day)
			if state.SchedulePendingDay == day {
				state.SchedulePendingDay = 0
			}
			sessions.set(chat.ID, state)
			sendScheduleDaysPicker(bot, chat, state.ScheduleDays, state.ScheduleTimes)
			return
		}

		state.ScheduleDays = append(state.ScheduleDays, day)
		state.SchedulePendingDay = day
		state.Action = actionScheduleTime
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи время HH:MM для "+weekdayLabel(day)+":", nil)

	case data == "settings:schedule_days_done":
		state := sessions.get(chat.ID)
		if state.Action == actionScheduleTime && state.SchedulePendingDay != 0 {
			_ = bot.SendMessage(chat, "Сначала введи время HH:MM для "+weekdayLabel(state.SchedulePendingDay)+".", nil)
			return
		}
		if state.Action != actionSchedulePickDays {
			_ = bot.SendMessage(chat, "Сначала нажми «Новое расписание».", nil)
			return
		}
		if len(state.ScheduleDays) == 0 {
			_ = bot.SendMessage(chat, "Выбери хотя бы один день недели.", nil)
			return
		}
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		if state.ScheduleTimes == nil {
			_ = bot.SendMessage(chat, "Сначала выбери день и задай время.", nil)
			return
		}
		for _, day := range state.ScheduleDays {
			if strings.TrimSpace(state.ScheduleTimes[day]) == "" {
				_ = bot.SendMessage(chat, "Не задано время для "+weekdayLabel(day)+". Нажми на день и введи время.", nil)
				return
			}
		}

		created := make([]string, 0, len(state.ScheduleDays))
		for _, day := range state.ScheduleDays {
			scheduleExpr, err := postgres.BuildScheduleExpr([]int{day}, state.ScheduleTimes[day])
			if err != nil {
				_ = bot.SendMessage(chat, "Ошибка формата расписания: "+err.Error(), nil)
				return
			}
			if _, err := store.CreateSchedule(context.Background(), targetChatID, state.ScheduleName, scheduleExpr); err != nil {
				templateNames, namesErr := store.ListTemplateNamesByChatID(context.Background(), targetChatID)
				if namesErr == nil && len(templateNames) > 0 {
					_ = bot.SendMessage(chat, "Ошибка сохранения расписания: "+err.Error()+"\nДоступные шаблоны: "+strings.Join(templateNames, ", "), nil)
				} else {
					_ = bot.SendMessage(chat, "Ошибка сохранения расписания: "+err.Error(), nil)
				}
				return
			}
			created = append(created, postgres.FormatScheduleExprForDisplay(scheduleExpr))
		}

		template := state.ScheduleName
		state.Action = actionNone
		state.ScheduleDays = nil
		state.ScheduleTimes = nil
		state.SchedulePendingDay = 0
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, fmt.Sprintf("Расписания для %s добавлены: %s", template, strings.Join(created, "; ")), nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:schedule_days_cancel":
		state := sessions.get(chat.ID)
		state.Action = actionNone
		state.ScheduleDays = nil
		state.ScheduleTimes = nil
		state.ScheduleName = ""
		state.SchedulePendingDay = 0
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Настройка расписания отменена.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:schedule_manage":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		if strings.TrimSpace(state.SelectedTemplate) == "" {
			_ = bot.SendMessage(chat, "Сначала выбери шаблон.", nil)
			return
		}
		sendScheduleManagePicker(bot, store, chat, targetChatID, state.SelectedTemplate, callback.Sender)

	case strings.HasPrefix(data, "settings:schedule_select:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}

		idText := strings.TrimPrefix(data, "settings:schedule_select:")
		id, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный ID расписания.", nil)
			return
		}
		details, err := store.GetScheduleDetails(context.Background(), targetChatID, id)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось найти расписание: "+err.Error(), nil)
			return
		}

		state.ManageScheduleID = id
		sessions.set(chat.ID, state)

		options := &tele.SendOptions{
			ReplyMarkup: tele.ReplyMarkup{
				InlineKeyboard: [][]tele.KeyboardButton{
					{
						{Text: "Изменить время", Data: fmt.Sprintf("settings:schedule_edit_time:%d", id)},
						{Text: "Удалить", Data: fmt.Sprintf("settings:schedule_delete:%d", id)},
					},
					{
						{Text: "Назад к расписаниям", Data: "settings:schedule_manage"},
					},
				},
			},
		}
		_ = bot.SendMessage(
			chat,
			fmt.Sprintf("Расписание #%d: %s", id, postgres.FormatScheduleExprForDisplay(details.Schedule)),
			options,
		)

	case data == "settings:template_edit":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
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

		preview := "Редактирование шаблона " + state.SelectedTemplate
		preview += "\n\nТекущий вопрос:\n" + template.Question
		preview += "\n\nТекущие варианты:"
		for i, option := range template.Options {
			preview += fmt.Sprintf("\n%d. %s", i+1, option)
		}
		options := &tele.SendOptions{
			ReplyMarkup: tele.ReplyMarkup{
				InlineKeyboard: [][]tele.KeyboardButton{
					{
						{Text: "Изменить вопрос", Data: "settings:template_edit_question"},
						{Text: "Изменить варианты", Data: "settings:template_edit_options"},
					},
					{
						{Text: "Отмена", Data: "settings:main_menu"},
					},
				},
			},
		}
		_ = bot.SendMessage(chat, preview, options)

	case data == "settings:template_edit_question":
		state := sessions.get(chat.ID)
		if strings.TrimSpace(state.SelectedTemplate) == "" {
			_ = bot.SendMessage(chat, "Сначала выбери шаблон.", nil)
			return
		}
		state.Action = actionTemplateEditQuestion
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи новый вопрос шаблона:", nil)

	case data == "settings:template_edit_options":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
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

		state.Action = actionPollOptions
		state.PollName = template.Name
		state.PollQuestion = template.Question
		state.PollOptions = nil
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Текущие варианты будут заменены. Добавляй новые варианты по одному сообщению и нажми «Готово».", nil)
		sendPollOptionsPrompt(bot, chat, state.PollOptions)

	case data == "settings:template_delete":
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		if strings.TrimSpace(state.SelectedTemplate) == "" {
			_ = bot.SendMessage(chat, "Сначала выбери шаблон.", nil)
			return
		}

		if err := store.DeleteTemplateByName(context.Background(), targetChatID, state.SelectedTemplate); err != nil {
			_ = bot.SendMessage(chat, "Ошибка удаления шаблона: "+err.Error(), nil)
			return
		}

		deleted := state.SelectedTemplate
		state.Action = actionNone
		state.SelectedTemplate = ""
		state.AvailableTemplates = nil
		state.ScheduleName = ""
		state.ScheduleDays = nil
		state.ScheduleTimes = nil
		state.SchedulePendingDay = 0
		state.ManageScheduleID = 0
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Шаблон удален: "+deleted, nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case strings.HasPrefix(data, "settings:schedule_edit_time:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		idText := strings.TrimPrefix(data, "settings:schedule_edit_time:")
		id, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный ID расписания.", nil)
			return
		}
		details, err := store.GetScheduleDetails(context.Background(), targetChatID, id)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось найти расписание: "+err.Error(), nil)
			return
		}
		spec, err := postgres.ParseScheduleExpr(details.Schedule)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось прочитать расписание: "+err.Error(), nil)
			return
		}
		state.Action = actionScheduleEditTime
		state.ManageScheduleID = id
		state.ScheduleDays = spec.Weekdays
		sessions.set(chat.ID, state)
		_ = bot.SendMessage(chat, "Введи новое время HH:MM для расписания #"+idText+":", nil)

	case strings.HasPrefix(data, "settings:schedule_toggle:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		idText := strings.TrimPrefix(data, "settings:schedule_toggle:")
		id, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный ID расписания.", nil)
			return
		}
		details, err := store.GetScheduleDetails(context.Background(), targetChatID, id)
		if err != nil {
			_ = bot.SendMessage(chat, "Не удалось найти расписание: "+err.Error(), nil)
			return
		}
		if err := store.SetScheduleActive(context.Background(), targetChatID, id, !details.IsActive); err != nil {
			_ = bot.SendMessage(chat, "Ошибка изменения статуса: "+err.Error(), nil)
			return
		}
		_ = bot.SendMessage(chat, "Статус расписания обновлен.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case strings.HasPrefix(data, "settings:schedule_delete:"):
		state := sessions.get(chat.ID)
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = state.TargetGroupChatID
		}
		idText := strings.TrimPrefix(data, "settings:schedule_delete:")
		id, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			_ = bot.SendMessage(chat, "Некорректный ID расписания.", nil)
			return
		}
		if err := store.DeleteSchedule(context.Background(), targetChatID, id); err != nil {
			_ = bot.SendMessage(chat, "Ошибка удаления расписания: "+err.Error(), nil)
			return
		}
		_ = bot.SendMessage(chat, "Расписание удалено.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	case data == "settings:schedule_noop":
		// noop button for visually grouping schedule entries
		return

	case data == "settings:list":
		targetChatID := chat.ID
		if isPrivateChat(chat) {
			targetChatID = sessions.get(chat.ID).TargetGroupChatID
		}
		if targetChatID == 0 {
			_ = bot.SendMessage(chat, "Сначала выбери группу.", nil)
			return
		}
		sendCurrentSettings(bot, store, chat, targetChatID)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, callback.Sender))

	default:
		_ = bot.SendMessage(chat, "Неизвестное действие меню.", nil)
	}
}

func sendScheduleManagePicker(
	bot *tele.Bot,
	store *postgres.Store,
	chat tele.Chat,
	targetChatID int64,
	templateName string,
	sender tele.User,
) {
	schedules, err := store.ListSchedulesByTemplate(context.Background(), targetChatID, templateName)
	if err != nil {
		_ = bot.SendMessage(chat, "Ошибка загрузки расписаний: "+err.Error(), nil)
		return
	}
	if len(schedules) == 0 {
		_ = bot.SendMessage(chat, "Для этого шаблона пока нет расписаний.", nil)
		sendSettingsMenu(bot, chat, selectedGroupTitleForChat(store, chat, sender))
		return
	}

	keyboard := make([][]tele.KeyboardButton, 0, len(schedules)+1)
	for _, schedule := range schedules {
		status := "🟢"
		if !schedule.IsActive {
			status = "⚪"
		}
		keyboard = append(keyboard, []tele.KeyboardButton{
			{
				Text: fmt.Sprintf("%s %s", status, schedule.SendAt),
				Data: fmt.Sprintf("settings:schedule_select:%d", schedule.ID),
			},
		})
	}
	keyboard = append(keyboard, []tele.KeyboardButton{
		{Text: "Назад в меню шаблона", Data: "settings:template_menu"},
	})

	options := &tele.SendOptions{ReplyMarkup: tele.ReplyMarkup{InlineKeyboard: keyboard}}
	_ = bot.SendMessage(chat, "Выбери расписание:", options)
}

func sendCurrentSettings(bot *tele.Bot, store *postgres.Store, recipient tele.Chat, targetChatID int64) {
	templates, schedules, err := store.GetGroupSnapshot(context.Background(), targetChatID)
	if err != nil {
		_ = bot.SendMessage(recipient, "Не удалось получить настройки: "+err.Error(), nil)
		return
	}

	message := fmt.Sprintf("Текущие настройки для чата %d:\n\nШаблоны:\n", targetChatID)
	if len(templates) == 0 {
		message += "- пусто\n"
	} else {
		for _, template := range templates {
			message += fmt.Sprintf("- %s: %s\n", template.Name, template.Question)
		}
	}

	message += "\nРасписания:\n"
	if len(schedules) == 0 {
		message += "- пусто"
	} else {
		for _, schedule := range schedules {
			message += fmt.Sprintf("- %s: %s\n", schedule.TemplateName, schedule.SendAt)
		}
	}

	_ = bot.SendMessage(recipient, message, nil)
}

func loadEventTemplateDetails(ctx context.Context, store *postgres.Store, targetChatID int64, eventID uint64) (*postgres.EventTemplateDetails, error) {
	events, err := store.ListEventsByChatID(ctx, targetChatID)
	if err != nil {
		return nil, err
	}

	var selected *postgres.EventView
	for i := range events {
		if events[i].ID == eventID {
			selected = &events[i]
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("event not found")
	}
	if strings.TrimSpace(selected.PollTemplate) == "" {
		return nil, fmt.Errorf("poll template is not bound")
	}

	template, err := store.GetTemplateByName(ctx, targetChatID, selected.PollTemplate)
	if err != nil {
		return nil, err
	}

	return &postgres.EventTemplateDetails{
		EventID:          eventID,
		EventName:        selected.Name,
		TemplateName:     template.Name,
		TemplateQuestion: template.Question,
		TemplateOptions:  template.Options,
	}, nil
}

func sendEventEditMenu(bot *tele.Bot, chat tele.Chat, event postgres.EventView) {
	costText := "не указана"
	if event.CostAmount != nil {
		costText = fmt.Sprintf("%.2f ₽", *event.CostAmount)
	}
	pollText := "не привязан"
	if strings.TrimSpace(event.PollTemplate) != "" {
		pollText = event.PollTemplate
	}

	text := fmt.Sprintf(
		"Редактирование события #%d\n%s\n%s\nПубликация: %s\nНачало: %s\nКонец: %s\nОпрос: %s\nСтоимость: %s",
		event.ID,
		event.Name,
		weekdayLabel(event.StartWeekday),
		event.PollPublishTime,
		event.StartTime,
		event.EndTime,
		pollText,
		costText,
	)
	options := &tele.SendOptions{
		ReplyMarkup: tele.ReplyMarkup{
			InlineKeyboard: [][]tele.KeyboardButton{
				{
					{Text: "Привязать опрос", Data: fmt.Sprintf("settings:event_edit_bind_poll:%d", event.ID)},
				},
				{
					{Text: "Учитывать голоса", Data: fmt.Sprintf("settings:event_edit_counted_options:%d", event.ID)},
				},
				{
					{Text: "Изменить стоимость", Data: fmt.Sprintf("settings:event_edit_cost:%d", event.ID)},
				},
				{
					{Text: "Посчитать", Data: fmt.Sprintf("settings:event_calc:%d", event.ID)},
				},
				{
					{Text: "Назад к событиям", Data: "settings:section_events"},
				},
			},
		},
	}
	_ = bot.SendMessage(chat, text, options)
}

func sendEventCountedOptionsPicker(bot *tele.Bot, chat tele.Chat, details *postgres.EventTemplateDetails, selected []int) {
	var b strings.Builder
	b.WriteString("Событие: ")
	b.WriteString(details.EventName)
	b.WriteString("\nШаблон: ")
	b.WriteString(details.TemplateName)
	b.WriteString("\nВыбери варианты, которые учитывать в расчете суммы:")

	keyboard := make([][]tele.KeyboardButton, 0, len(details.TemplateOptions)+2)
	for idx, option := range details.TemplateOptions {
		prefix := "⬜"
		if containsInt(selected, idx) {
			prefix = "✅"
		}
		keyboard = append(keyboard, []tele.KeyboardButton{
			{Text: fmt.Sprintf("%s %d) %s", prefix, idx+1, option), Data: fmt.Sprintf("settings:event_count_toggle:%d", idx)},
		})
	}
	keyboard = append(keyboard, []tele.KeyboardButton{
		{Text: "Сохранить", Data: "settings:event_count_save"},
		{Text: "Отмена", Data: "settings:event_cancel"},
	})

	menu := &tele.SendOptions{ReplyMarkup: tele.ReplyMarkup{InlineKeyboard: keyboard}}
	_ = bot.SendMessage(chat, b.String(), menu)
}

func selectedGroupTitleForChat(store *postgres.Store, chat tele.Chat, sender tele.User) string {
	if !isPrivateChat(chat) {
		return formatGroupTitle(chat.ID, chat.Title)
	}

	targetChatID := sessions.get(chat.ID).TargetGroupChatID
	if targetChatID == 0 {
		return "не выбрана"
	}

	groups, err := store.ListGroupsForAdmin(context.Background(), int64(sender.ID))
	if err != nil {
		return formatGroupTitle(targetChatID, "")
	}

	for _, group := range groups {
		if group.ChatID == targetChatID {
			return formatGroupTitle(group.ChatID, group.Title)
		}
	}

	return formatGroupTitle(targetChatID, "")
}

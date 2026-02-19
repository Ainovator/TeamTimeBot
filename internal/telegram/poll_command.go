package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

const pollCommandCooldown = 10 * time.Minute

type pollCommandState struct {
	mu         sync.Mutex
	nextByChat map[int64]time.Time
	pinByChat  map[int64]int64
}

func newPollCommandState() *pollCommandState {
	return &pollCommandState{
		nextByChat: make(map[int64]time.Time),
		pinByChat:  make(map[int64]int64),
	}
}

var pollState = newPollCommandState()

func HandlePollCommand(store *postgres.Store, c tele.Context) {
	chat := c.Message.Chat
	now := time.Now().UTC()
	if remaining, blocked := pollState.reserveCooldown(chat.ID, now); blocked {
		sendPollCommandMessage(c.Bot, chat, "Команда /poll доступна раз в 10 минут.\nДо следующего вызова: "+formatCooldownRemaining(remaining))
		return
	}

	postID, err := store.GetLatestGroupPollByChatID(context.Background(), chat.ID)
	if err != nil {
		sendPollCommandMessage(c.Bot, chat, "Не удалось получить последний опрос: "+err.Error())
		return
	}
	if postID == nil {
		sendPollCommandMessage(c.Bot, chat, "В группе пока нет опубликованных опросов.")
		return
	}

	votes, err := store.ListGroupPollVotesByPostID(context.Background(), chat.ID, *postID)
	if err != nil {
		sendPollCommandMessage(c.Bot, chat, "Не удалось получить голоса последнего опроса: "+err.Error())
		return
	}

	queue := make([]postgres.GroupPollVoteItem, 0, len(votes))
	for _, vote := range votes {
		if vote.Counted {
			queue = append(queue, vote)
		}
	}
	if len(queue) == 0 {
		sendPollCommandMessage(c.Bot, chat, "В последнем опросе нет голосов по вариантам с опцией «учёт».")
		return
	}

	loc := time.UTC
	if group, err := store.GetGroupByChatID(context.Background(), chat.ID); err == nil {
		if loaded, locErr := time.LoadLocation(group.Timezone); locErr == nil {
			loc = loaded
		}
	}

	var b strings.Builder
	b.WriteString("Текущая очередь:\n\n")
	for i, vote := range queue {
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". ")
		b.WriteString(pollVoteDisplayName(vote))
		b.WriteString(" (")
		b.WriteString(strings.TrimSpace(vote.ChoiceLabel))
		b.WriteString(") - ")
		b.WriteString(vote.VotedAt.In(loc).Format("02.01 15:04"))
		if i+1 < len(queue) {
			b.WriteByte('\n')
		}
	}

	messageID, err := sendPollReportWithMessageID(c.Bot, chat, b.String())
	if err != nil {
		log.Printf("poll_command: failed to send poll report in chat %d: %v", chat.ID, err)
		sendPollCommandMessage(c.Bot, chat, "Не удалось опубликовать список: "+err.Error())
		return
	}

	prevPinned := pollState.getPinned(chat.ID)
	if prevPinned == 0 {
		if currentChat, chatErr := c.Bot.GetChat(chat); chatErr == nil && currentChat.PinnedMessage != nil {
			prevPinned = int64(currentChat.PinnedMessage.ID)
		}
	}
	if prevPinned > 0 && prevPinned != messageID {
		if err := unpinChatMessage(c.Bot, chat.ID, prevPinned); err != nil {
			log.Printf("poll_command: failed to unpin previous message %d in chat %d: %v", prevPinned, chat.ID, err)
		}
	}
	if err := pinChatMessage(c.Bot, chat.ID, messageID); err != nil {
		log.Printf("poll_command: failed to pin message %d in chat %d: %v", messageID, chat.ID, err)
		return
	}
	pollState.setPinned(chat.ID, messageID)
}

func pollVoteDisplayName(vote postgres.GroupPollVoteItem) string {
	if rn := strings.TrimSpace(vote.RealName); rn != "" {
		return rn
	}
	full := strings.TrimSpace(strings.TrimSpace(vote.FirstName + " " + vote.LastName))
	if full != "" {
		return full
	}
	if u := strings.TrimSpace(vote.Username); u != "" {
		return "@" + u
	}
	return strconv.FormatInt(vote.UserID, 10)
}

func (s *pollCommandState) reserveCooldown(chatID int64, now time.Time) (time.Duration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.nextByChat[chatID]
	if now.Before(next) {
		return next.Sub(now), true
	}
	s.nextByChat[chatID] = now.Add(pollCommandCooldown)
	return 0, false
}

func (s *pollCommandState) getPinned(chatID int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pinByChat[chatID]
}

func (s *pollCommandState) setPinned(chatID int64, messageID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pinByChat[chatID] = messageID
}

func formatCooldownRemaining(remaining time.Duration) string {
	if remaining <= 0 {
		return "00:00"
	}
	seconds := int(remaining / time.Second)
	if remaining%time.Second != 0 {
		seconds++
	}
	if seconds < 0 {
		seconds = 0
	}
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

func sendPollCommandMessage(bot *tele.Bot, chat tele.Chat, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	// Telegram text hard limit is 4096 chars. Keep margin for safety.
	const maxLen = 3900
	if len(text) <= maxLen {
		if err := bot.SendMessage(chat, text, nil); err != nil {
			log.Printf("poll_command: failed to send message to chat %d: %v", chat.ID, err)
		}
		return
	}

	chunks := splitByMaxLen(text, maxLen)
	for _, payload := range chunks {
		if err := bot.SendMessage(chat, payload, nil); err != nil {
			log.Printf("poll_command: failed to send chunk to chat %d: %v", chat.ID, err)
			return
		}
	}
}

func splitByMaxLen(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	lines := strings.Split(text, "\n")
	chunks := make([]string, 0, 2)
	var chunk strings.Builder
	flush := func() {
		payload := strings.TrimSpace(chunk.String())
		if payload == "" {
			return
		}
		chunks = append(chunks, payload)
		chunk.Reset()
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if chunk.Len() > 0 && chunk.Len()+1+len(line) > maxLen {
			flush()
		}
		if chunk.Len() > 0 {
			chunk.WriteByte('\n')
		}
		chunk.WriteString(line)
	}
	flush()
	return chunks
}

func sendPollReportWithMessageID(bot *tele.Bot, chat tele.Chat, text string) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, errors.New("empty report text")
	}

	const maxLen = 3900
	chunks := splitByMaxLen(text, maxLen)
	firstMessageID := int64(0)
	for _, chunk := range chunks {
		messageID, err := sendMessageWithMeta(bot, chat.ID, chunk)
		if err != nil {
			return firstMessageID, err
		}
		if firstMessageID == 0 {
			firstMessageID = messageID
		}
	}
	if firstMessageID == 0 {
		return 0, errors.New("telegram did not return message id")
	}
	return firstMessageID, nil
}

func sendMessageWithMeta(bot *tele.Bot, chatID int64, text string) (int64, error) {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	var response struct {
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	if err := callTelegramAPI(bot.Token, "sendMessage", payload, &response); err != nil {
		return 0, err
	}
	return response.Result.MessageID, nil
}

func pinChatMessage(bot *tele.Bot, chatID int64, messageID int64) error {
	payload := map[string]interface{}{
		"chat_id":              chatID,
		"message_id":           messageID,
		"disable_notification": true,
	}
	return callTelegramAPI(bot.Token, "pinChatMessage", payload, nil)
}

func unpinChatMessage(bot *tele.Bot, chatID int64, messageID int64) error {
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	return callTelegramAPI(bot.Token, "unpinChatMessage", payload, nil)
}

func callTelegramAPI(token, method string, payload interface{}, result interface{}) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", &body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var envelope struct {
		Ok          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if !envelope.Ok {
		if strings.TrimSpace(envelope.Description) == "" {
			return errors.New("telegram api error")
		}
		return errors.New(envelope.Description)
	}
	if result == nil {
		return nil
	}
	if len(envelope.Result) == 0 {
		return errors.New("telegram api returned empty result")
	}
	return json.Unmarshal(envelope.Result, result)
}

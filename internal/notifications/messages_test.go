package notifications

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/internal/storage/postgres"
)

func TestMentionsEscapeNamesAndWorkWithoutUsername(t *testing.T) {
	got := Mention(123, `Анна <b> & "команда"`)
	if !strings.Contains(got, `href="tg://user?id=123"`) || strings.Contains(got, "<b>") || !strings.Contains(got, "&amp;") {
		t.Fatalf("unsafe or missing mention: %s", got)
	}
	if strings.Contains(Mention(0, "Гость"), "tg://") {
		t.Fatal("non-Telegram guests must not produce an invalid mention")
	}
	members := []postgres.GroupMemberView{{UserTelegramID: 123, RealName: "Анна"}, {UserTelegramID: 456, Username: "volley"}}
	message := Announcement("Тренировка <вечер> & друзья", members)
	if strings.Contains(message, "<вечер>") || strings.Count(message, "tg://user?id=") != 2 {
		t.Fatalf("invalid announcement: %s", message)
	}
	if PollInvitation("Вопрос", nil) != "" {
		t.Fatal("no selected recipients should produce no extra poll message")
	}
}

type recordingSender struct {
	parts   []string
	options []*tele.SendOptions
	fail    bool
}

func (sender *recordingSender) SendMessage(_ tele.Recipient, message string, options *tele.SendOptions) error {
	if sender.fail {
		return errors.New("Telegram unavailable")
	}
	sender.parts = append(sender.parts, message)
	sender.options = append(sender.options, options)
	return nil
}

func TestLongRecipientListKeepsEveryMentionAndReply(t *testing.T) {
	var lines []string
	for i := 1; i <= 180; i++ {
		lines = append(lines, Mention(int64(i), fmt.Sprintf("🏐 Игрок %d %s", i, strings.Repeat("Я", 40))))
	}
	sender := &recordingSender{}
	if err := SendHTML(sender, tele.Chat{ID: -100}, strings.Join(lines, "\n"), 321); err != nil {
		t.Fatal(err)
	}
	if len(sender.parts) < 2 {
		t.Fatal("long list was not split")
	}
	count := 0
	for i, part := range sender.parts {
		if textLength(part) > 3500 || strings.Count(part, "<a ") != strings.Count(part, "</a>") {
			t.Fatalf("broken message part: %s", part)
		}
		count += strings.Count(part, "tg://user?id=")
		if sender.options[i].ParseMode != tele.ModeHTML || sender.options[i].ReplyTo.ID != 321 || sender.options[i].DisableNotification {
			t.Fatal("mention send options were lost")
		}
	}
	if count != len(lines) {
		t.Fatalf("got %d mentions, want %d", count, len(lines))
	}
	if err := SendHTML(&recordingSender{fail: true}, tele.Chat{ID: -100}, "Сообщение", 0); err == nil {
		t.Fatal("send error was ignored")
	}
}

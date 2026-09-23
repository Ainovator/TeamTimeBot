package notifications

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf16"

	tele "gopkg.in/telebot.v4"
)

// Mention targets the Telegram account even when it has no public username.
func Mention(userID int64, name string) string {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		name = fmt.Sprintf("Игрок %d", userID)
	}
	if userID <= 0 {
		return html.EscapeString(name)
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, userID, html.EscapeString(name))
}

type Sender interface {
	SendMessage(tele.Recipient, string, *tele.SendOptions) error
}

var htmlTags = regexp.MustCompile(`<[^>]*>`)

func textLength(value string) int {
	return len(utf16.Encode([]rune(html.UnescapeString(htmlTags.ReplaceAllString(value, "")))))
}

// Split on complete lines so an inline mention never gets cut in half.
func Messages(message string) []string {
	var result []string
	current := ""
	for _, line := range strings.Split(strings.TrimSpace(message), "\n") {
		candidate := line
		if current != "" {
			candidate = current + "\n" + line
		}
		if current != "" && textLength(candidate) > 3500 {
			result = append(result, strings.TrimSpace(current))
			current = line
		} else {
			current = candidate
		}
	}
	if strings.TrimSpace(current) != "" {
		result = append(result, strings.TrimSpace(current))
	}
	return result
}

func SendHTML(sender Sender, chat tele.Recipient, message string, replyTo int64) error {
	options := &tele.SendOptions{ParseMode: tele.ModeHTML, DisableWebPagePreview: true}
	if replyTo != 0 {
		options.ReplyTo.ID = int(replyTo)
	}
	for _, part := range Messages(message) {
		if err := sender.SendMessage(chat, part, options); err != nil {
			return err
		}
	}
	return nil
}

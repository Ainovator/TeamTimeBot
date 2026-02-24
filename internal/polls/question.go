package polls

import (
	"strings"
	"time"
	"unicode/utf8"
)

const telegramPollQuestionMaxRunes = 300

// WithEventDate appends the event date to a poll question and keeps Telegram limits.
func WithEventDate(question string, eventDate time.Time) string {
	base := strings.TrimSpace(question)
	if eventDate.IsZero() {
		return truncateRunes(base, telegramPollQuestionMaxRunes)
	}

	dateText := eventDate.Format("02.01.2006")
	if strings.Contains(base, dateText) {
		return truncateRunes(base, telegramPollQuestionMaxRunes)
	}

	suffix := " (" + dateText + ")"
	suffixLen := utf8.RuneCountInString(suffix)
	if suffixLen >= telegramPollQuestionMaxRunes {
		return truncateRunes(dateText, telegramPollQuestionMaxRunes)
	}

	base = truncateRunes(base, telegramPollQuestionMaxRunes-suffixLen)
	base = strings.TrimSpace(base)
	if base == "" {
		return dateText
	}
	return base + suffix
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

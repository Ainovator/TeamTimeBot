package polls

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestWithEventDate_AppendsDate(t *testing.T) {
	date := time.Date(2026, time.March, 1, 18, 0, 0, 0, time.UTC)
	got := WithEventDate("Who joins training?", date)
	want := "Who joins training? (01.03.2026)"
	if got != want {
		t.Fatalf("unexpected question: got %q want %q", got, want)
	}
}

func TestWithEventDate_AvoidsDuplicateDate(t *testing.T) {
	date := time.Date(2026, time.March, 1, 18, 0, 0, 0, time.UTC)
	input := "Training poll (01.03.2026)"
	got := WithEventDate(input, date)
	if got != input {
		t.Fatalf("date should not be duplicated: got %q want %q", got, input)
	}
}

func TestWithEventDate_TruncatesToTelegramLimit(t *testing.T) {
	date := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	base := strings.Repeat("a", 400)
	got := WithEventDate(base, date)
	if utf8.RuneCountInString(got) > 300 {
		t.Fatalf("question is too long: %d runes", utf8.RuneCountInString(got))
	}
	if !strings.HasSuffix(got, "(01.03.2026)") {
		t.Fatalf("missing date suffix: %q", got)
	}
}

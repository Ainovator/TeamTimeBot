package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

func HandlePollAnswer(store *postgres.Store, answer tele.PollAnswer) {
	pollID := strings.TrimSpace(answer.PollID)
	if pollID == "" {
		return
	}

	post, err := store.GetEventPollPostByTelegramPollID(context.Background(), pollID)
	if err != nil {
		// Poll can be unrelated to persisted posts; ignore silently.
		return
	}

	choice := "none"
	if len(answer.OptionIDs) > 0 {
		parts := make([]string, 0, len(answer.OptionIDs))
		for _, idx := range answer.OptionIDs {
			parts = append(parts, fmt.Sprintf("option_%d", idx))
		}
		choice = strings.Join(parts, ",")
	}

	if err := store.UpsertEventPollVote(
		context.Background(),
		post.ID,
		int64(answer.User.ID),
		answer.User.Username,
		answer.User.FirstName,
		answer.User.LastName,
		choice,
		"poll",
		time.Now().UTC(),
	); err != nil {
		log.Printf("poll_answer: failed to save vote for post %d user %d: %v", post.ID, answer.User.ID, err)
	}
}

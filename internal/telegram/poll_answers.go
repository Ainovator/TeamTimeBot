package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

const emptyVoteGracePeriod = 60 * time.Second

type pendingVoteClearKey struct {
	PostID uint64
	UserID int64
}

type pendingVoteClearEntry struct {
	timer    *time.Timer
	issuedAt time.Time
	user     tele.User
}

var pendingVoteClears = struct {
	mu      sync.Mutex
	entries map[pendingVoteClearKey]*pendingVoteClearEntry
}{
	entries: make(map[pendingVoteClearKey]*pendingVoteClearEntry),
}

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

	choices := make([]string, 0, len(answer.OptionIDs))
	for _, idx := range answer.OptionIDs {
		choices = append(choices, fmt.Sprintf("option_%d", idx))
	}

	if len(choices) == 0 {
		// Telegram multi-select can force users to clear all choices first.
		// Delay destructive clear so quick re-vote can preserve queue position
		// for still-selected options (e.g. keep "+" while removing "+1").
		schedulePendingVoteClear(store, post.ID, answer.User)
		return
	}
	cancelPendingVoteClear(post.ID, int64(answer.User.ID))

	if err := store.ReplaceEventPollVotes(
		context.Background(),
		post.ID,
		int64(answer.User.ID),
		answer.User.Username,
		answer.User.FirstName,
		answer.User.LastName,
		choices,
		"poll",
		time.Now().UTC(),
	); err != nil {
		log.Printf("poll_answer: failed to save vote for post %d user %d: %v", post.ID, answer.User.ID, err)
	}
}

func schedulePendingVoteClear(store *postgres.Store, postID uint64, user tele.User) {
	key := pendingVoteClearKey{
		PostID: postID,
		UserID: int64(user.ID),
	}
	issuedAt := time.Now().UTC()
	timer := time.AfterFunc(emptyVoteGracePeriod, func() {
		runPendingVoteClear(store, key, issuedAt)
	})

	pendingVoteClears.mu.Lock()
	if prev := pendingVoteClears.entries[key]; prev != nil {
		prev.timer.Stop()
	}
	pendingVoteClears.entries[key] = &pendingVoteClearEntry{
		timer:    timer,
		issuedAt: issuedAt,
		user:     user,
	}
	pendingVoteClears.mu.Unlock()
}

func cancelPendingVoteClear(postID uint64, userID int64) {
	key := pendingVoteClearKey{
		PostID: postID,
		UserID: userID,
	}
	pendingVoteClears.mu.Lock()
	entry := pendingVoteClears.entries[key]
	if entry != nil {
		delete(pendingVoteClears.entries, key)
	}
	pendingVoteClears.mu.Unlock()
	if entry != nil {
		entry.timer.Stop()
	}
}

func runPendingVoteClear(store *postgres.Store, key pendingVoteClearKey, issuedAt time.Time) {
	pendingVoteClears.mu.Lock()
	entry := pendingVoteClears.entries[key]
	if entry == nil || !entry.issuedAt.Equal(issuedAt) {
		pendingVoteClears.mu.Unlock()
		return
	}
	delete(pendingVoteClears.entries, key)
	user := entry.user
	pendingVoteClears.mu.Unlock()

	if err := store.ReplaceEventPollVotes(
		context.Background(),
		key.PostID,
		key.UserID,
		user.Username,
		user.FirstName,
		user.LastName,
		nil,
		"poll",
		time.Now().UTC(),
	); err != nil {
		log.Printf("poll_answer: failed to clear votes for post %d user %d after grace period: %v", key.PostID, key.UserID, err)
	}
}

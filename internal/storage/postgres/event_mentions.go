package postgres

import (
	"context"
	"errors"
	"sort"

	"github.com/lib/pq"
)

type EventMentionSettings struct {
	UserIDs        pq.Int64Array `json:"userIDs" gorm:"column:mention_user_ids;type:bigint[]"`
	OnPoll         bool          `json:"onPoll" gorm:"column:mention_on_poll"`
	OnAnnouncement bool          `json:"onAnnouncement" gorm:"column:mention_on_announcement"`
}

func normalizeMentionIDs(ids []int64) (pq.Int64Array, error) {
	unique := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errors.New("mention user ID must be positive")
		}
		unique[id] = true
	}
	result := make(pq.Int64Array, 0, len(unique))
	for id := range unique {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func (s *Store) validateEventMentions(ctx context.Context, groupID uint64, settings *EventMentionSettings) error {
	ids, err := normalizeMentionIDs(settings.UserIDs)
	if err != nil {
		return err
	}
	settings.UserIDs = ids
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&GroupMember{}).
		Where("group_id = ? AND user_telegram_id IN ? AND is_active = TRUE AND status = 'active'", groupID, []int64(ids)).
		Count(&count).Error; err != nil {
		return err
	}
	if int(count) != len(ids) {
		return errors.New("mention recipients must be active members of this organization")
	}
	return nil
}

// Resolve current membership at send time; departed players are not pinged.
func (s *Store) GetEventMentionRecipients(ctx context.Context, chatID int64, eventID uint64, publication string) ([]GroupMemberView, error) {
	event, err := s.GetEventByID(ctx, chatID, eventID)
	if err != nil {
		return nil, err
	}
	settings := event.Mentions
	if (publication != "poll" || !settings.OnPoll) && (publication != "announcement" || !settings.OnAnnouncement) {
		return nil, nil
	}
	if len(settings.UserIDs) == 0 {
		return nil, nil
	}
	members, err := s.ListGroupMembersByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	wanted := make(map[int64]bool, len(settings.UserIDs))
	for _, id := range settings.UserIDs {
		wanted[id] = true
	}
	var result []GroupMemberView
	for _, member := range members {
		if wanted[member.UserTelegramID] && member.Status == "active" {
			result = append(result, member)
		}
	}
	return result, nil
}

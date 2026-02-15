package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// DeleteEventInstance permanently deletes an event instance and all derived data for that day,
// so a new instance can be created again for the same (event_id, local_date).
//
// This is primarily useful for testing. It removes:
// - poll post for the instance (cascades votes, team split)
// - settlements for the instance/day (cascades payments)
// - sent flags (announcements/cancellations/settlement notices)
// - the instance row itself
func (s *Store) DeleteEventInstance(ctx context.Context, chatID int64, instanceID uint64) error {
	group, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	if instanceID == 0 {
		return errors.New("instance id is required")
	}

	type instRow struct {
		ID         uint64
		GroupID    uint64
		EventID    uint64
		LocalDate  time.Time
		PollPostID *uint64
	}
	var inst instRow
	if err := s.db.WithContext(ctx).
		Table("event_instances").
		Select("id, group_id, event_id, local_date, poll_post_id").
		Where("id = ? AND group_id = ?", instanceID, group.ID).
		Take(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("event instance not found")
		}
		return err
	}
	localDateStr := inst.LocalDate.Format("2006-01-02")

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Resolve the instance poll post (prefer direct link, fallback to old link on posts).
		pollPostID := inst.PollPostID
		if pollPostID == nil {
			var id uint64
			if err := tx.Table("event_poll_posts").
				Select("id").
				Where("instance_id = ?", inst.ID).
				Order("published_at DESC, id DESC").
				Limit(1).
				Scan(&id).Error; err != nil {
				return err
			}
			if id != 0 {
				pollPostID = &id
			}
		}

		// Delete poll post first so instance->poll and post->instance FKs don't block.
		// Votes and team split cascade from the post.
		if pollPostID != nil {
			if err := tx.Exec("DELETE FROM event_poll_posts WHERE id = ?", *pollPostID).Error; err != nil {
				return err
			}
		}

		// Delete settlements for this instance/day so the unique (event_id, local_date) does not block recreation.
		// Payments cascade from settlements.
		if err := tx.Exec(
			"DELETE FROM event_settlements WHERE instance_id = ? OR (event_id = ? AND local_date = ?)",
			inst.ID,
			inst.EventID,
			localDateStr,
		).Error; err != nil {
			return err
		}

		// Cleanup sent flags.
		if err := tx.Exec("DELETE FROM event_settlement_notices WHERE event_id = ? AND local_date = ?", inst.EventID, localDateStr).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM event_announcements WHERE event_id = ? AND event_date = ?", inst.EventID, localDateStr).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM event_cancellations WHERE event_id = ? AND event_date = ?", inst.EventID, localDateStr).Error; err != nil {
			return err
		}

		// Finally, delete the instance so it can be recreated for the same date.
		if err := tx.Exec("DELETE FROM event_instances WHERE id = ?", inst.ID).Error; err != nil {
			return err
		}

		return nil
	})
}

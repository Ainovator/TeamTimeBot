package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TrainingPass struct {
	ID             uint64                `json:"id"`
	GroupID        uint64                `json:"-"`
	UserTelegramID int64                 `json:"userTelegramID"`
	Name           string                `json:"name"`
	TotalVisits    *int                  `json:"totalVisits"`
	StartsOn       time.Time             `json:"startsOn"`
	EndsOn         time.Time             `json:"endsOn"`
	PriceKopecks   int64                 `json:"priceKopecks"`
	IsPaid         bool                  `json:"isPaid"`
	FrozenAt       *time.Time            `json:"frozenAt"`
	IsClosed       bool                  `json:"isClosed"`
	Version        int                   `json:"version"`
	RequestID      string                `json:"-"`
	CreatedAt      time.Time             `json:"createdAt"`
	UsedVisits     int                   `json:"usedVisits" gorm:"->;-:migration"`
	MemberName     string                `json:"memberName" gorm:"->;-:migration"`
	State          string                `json:"state" gorm:"-"`
	History        []TrainingPassHistory `json:"history" gorm:"-"`
}
type TrainingPassHistory struct {
	ID         uint64    `json:"id"`
	PassID     uint64    `json:"-"`
	InstanceID *uint64   `json:"instanceID"`
	ActorID    int64     `json:"actorID"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (TrainingPassHistory) TableName() string { return "training_pass_history" }

type IssueTrainingPass struct {
	UserTelegramID int64  `json:"userTelegramID"`
	Name           string `json:"name"`
	TotalVisits    *int   `json:"totalVisits"`
	StartsOn       string `json:"startsOn"`
	EndsOn         string `json:"endsOn"`
	PriceKopecks   int64  `json:"priceKopecks"`
	IsPaid         bool   `json:"isPaid"`
	RequestID      string `json:"requestID"`
}
type TrainingPassAction struct {
	Action  string `json:"action"`
	Version int    `json:"version"`
	Days    int    `json:"days"`
}
type TrainingAttendance struct {
	ID             uint64    `json:"id"`
	GroupID        uint64    `json:"-"`
	InstanceID     uint64    `json:"instanceID"`
	UserTelegramID int64     `json:"userTelegramID"`
	PassID         *uint64   `json:"passID"`
	Status         string    `json:"status"`
	ActorID        int64     `json:"actorID"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (TrainingAttendance) TableName() string { return "training_attendance" }

func passDate(t time.Time) string { return t.Format("2006-01-02") }
func groupToday(g *TelegramGroup) string {
	loc, err := time.LoadLocation(g.Timezone)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}
func trainingPassState(p TrainingPass, today string) string {
	switch {
	case p.IsClosed:
		return "closed"
	case p.FrozenAt != nil:
		return "frozen"
	case passDate(p.EndsOn) < today:
		return "expired"
	case !p.IsPaid:
		return "unpaid"
	case passDate(p.StartsOn) > today:
		return "scheduled"
	case p.TotalVisits != nil && p.UsedVisits >= *p.TotalVisits:
		return "exhausted"
	default:
		return "active"
	}
}
func usedPassVisits(tx *gorm.DB, id uint64) (int, error) {
	var n int64
	err := tx.Model(&TrainingAttendance{}).Where("pass_id = ? AND status = 'attended'", id).Count(&n).Error
	return int(n), err
}
func passHistory(tx *gorm.DB, id uint64, instance *uint64, actor int64, note string) error {
	return tx.Create(&TrainingPassHistory{PassID: id, InstanceID: instance, ActorID: actor, Note: note}).Error
}

// The group lock serializes issuing/changing passes and concurrent attendance for the same organization.
func (s *Store) passTransaction(ctx context.Context, chatID int64, fn func(*gorm.DB, *TelegramGroup) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var g TelegramGroup
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("chat_id = ?", chatID).Take(&g).Error; err != nil {
			return passProblem("Организация не найдена")
		}
		return fn(tx, &g)
	})
}

func (s *Store) ListTrainingPasses(ctx context.Context, chatID int64, userID int64) ([]TrainingPass, error) {
	g, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	out := make([]TrainingPass, 0)
	q := s.db.WithContext(ctx).Table("training_passes p").Select(`p.*,
 COALESCE(NULLIF(gm.real_name,''),NULLIF(TRIM(CONCAT(tu.first_name,' ',tu.last_name)),''),NULLIF(tu.username,''),'Игрок') AS member_name,
 (SELECT COUNT(*) FROM training_attendance a WHERE a.pass_id=p.id AND a.status='attended') AS used_visits`).
		Joins("LEFT JOIN group_members gm ON gm.group_id=p.group_id AND gm.user_telegram_id=p.user_telegram_id").
		Joins("LEFT JOIN telegram_users tu ON tu.telegram_id=p.user_telegram_id").Where("p.group_id = ?", g.ID)
	if userID != 0 {
		q = q.Where("p.user_telegram_id = ?", userID)
	}
	if err = q.Order("p.created_at DESC, p.id DESC").Scan(&out).Error; err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(out))
	for _, p := range out {
		ids = append(ids, p.ID)
	}
	history := make([]TrainingPassHistory, 0)
	if len(ids) > 0 {
		if err = s.db.WithContext(ctx).Where("pass_id IN ?", ids).Order("id DESC").Find(&history).Error; err != nil {
			return nil, err
		}
	}
	byID := map[uint64][]TrainingPassHistory{}
	for _, h := range history {
		byID[h.PassID] = append(byID[h.PassID], h)
	}
	for i := range out {
		out[i].State = trainingPassState(out[i], groupToday(g))
		out[i].History = byID[out[i].ID]
		if out[i].History == nil {
			out[i].History = []TrainingPassHistory{}
		}
	}
	return out, nil
}

func (s *Store) IssueTrainingPass(ctx context.Context, chatID, actor int64, req IssueTrainingPass) (uint64, error) {
	start, e1 := time.Parse("2006-01-02", req.StartsOn)
	end, e2 := time.Parse("2006-01-02", req.EndsOn)
	name := strings.TrimSpace(req.Name)
	if e1 != nil || e2 != nil || end.Before(start) || end.After(start.AddDate(5, 0, 0)) {
		return 0, passProblem("Укажите корректный срок абонемента, не более пяти лет")
	}
	if len([]rune(name)) < 1 || len([]rune(name)) > 100 {
		return 0, passProblem("Название должно содержать от 1 до 100 символов")
	}
	if req.TotalVisits != nil && (*req.TotalVisits < 1 || *req.TotalVisits > 1000) {
		return 0, passProblem("Укажите от 1 до 1000 занятий")
	}
	if req.PriceKopecks < 0 || req.PriceKopecks > 100000000 {
		return 0, passProblem("Укажите стоимость от 0 до 1 000 000 ₽")
	}
	if _, err := uuid.Parse(req.RequestID); err != nil {
		return 0, passProblem("Обновите форму выдачи абонемента")
	}
	var id uint64
	err := s.passTransaction(ctx, chatID, func(tx *gorm.DB, g *TelegramGroup) error {
		var existing TrainingPass
		e := tx.Where("group_id = ? AND request_id = ?", g.ID, req.RequestID).Take(&existing).Error
		if e == nil {
			id = existing.ID
			return nil
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var count int64
		if e = tx.Model(&GroupMember{}).Where("group_id = ? AND user_telegram_id = ? AND is_active", g.ID, req.UserTelegramID).Count(&count).Error; e != nil {
			return e
		}
		if count == 0 {
			return passProblem("Выберите действующего игрока этой организации")
		}
		p := TrainingPass{GroupID: g.ID, UserTelegramID: req.UserTelegramID, Name: name, TotalVisits: req.TotalVisits, StartsOn: start, EndsOn: end, PriceKopecks: req.PriceKopecks, IsPaid: req.IsPaid, Version: 1, RequestID: req.RequestID}
		if e = tx.Create(&p).Error; e != nil {
			return e
		}
		id = p.ID
		return passHistory(tx, p.ID, nil, actor, "Абонемент выдан")
	})
	return id, err
}

func (s *Store) ChangeTrainingPass(ctx context.Context, chatID, actor int64, id uint64, req TrainingPassAction) error {
	return s.passTransaction(ctx, chatID, func(tx *gorm.DB, g *TelegramGroup) error {
		var p TrainingPass
		if err := tx.Where("id = ? AND group_id = ?", id, g.ID).Take(&p).Error; err != nil {
			return passProblem("Абонемент не найден в этой организации")
		}
		if p.Version != req.Version {
			return passProblem("Абонемент уже изменён. Обновите список и повторите действие")
		}
		if p.IsClosed {
			return passProblem("Абонемент закрыт. Выдайте новый")
		}
		note := ""
		changes := map[string]interface{}{"version": p.Version + 1}
		switch req.Action {
		case "paid":
			changes["is_paid"] = true
			note = "Оплата отмечена вручную"
		case "unpaid":
			used, err := usedPassVisits(tx, p.ID)
			if err != nil {
				return err
			}
			if used > 0 {
				return passProblem("Нельзя снять оплату с абонемента, по которому уже посещали занятия")
			}
			changes["is_paid"] = false
			note = "Отметка оплаты отменена"
		case "freeze":
			if p.FrozenAt != nil || passDate(p.EndsOn) < groupToday(g) {
				return passProblem("Можно заморозить только неистёкший абонемент")
			}
			changes["frozen_at"] = time.Now()
			note = "Абонемент заморожен"
		case "resume":
			if p.FrozenAt == nil {
				return passProblem("Абонемент не заморожен")
			}
			loc, err := time.LoadLocation(g.Timezone)
			if err != nil {
				loc = time.UTC
			}
			from, _ := time.Parse("2006-01-02", p.FrozenAt.In(loc).Format("2006-01-02"))
			to, _ := time.Parse("2006-01-02", groupToday(g))
			days := int(to.Sub(from).Hours() / 24)
			if days < 0 {
				days = 0
			}
			changes["frozen_at"] = nil
			changes["ends_on"] = p.EndsOn.AddDate(0, 0, days)
			note = fmt.Sprintf("Заморозка снята, срок продлён на %d дн.", days)
		case "extend":
			if req.Days < 1 || req.Days > 365 {
				return passProblem("Продлить срок можно на 1–365 дней")
			}
			changes["ends_on"] = p.EndsOn.AddDate(0, 0, req.Days)
			note = fmt.Sprintf("Срок продлён на %d дн.", req.Days)
		case "close":
			changes["is_closed"] = true
			note = "Абонемент закрыт, история сохранена"
		default:
			return passProblem("Неизвестное действие с абонементом")
		}
		if err := tx.Model(&TrainingPass{}).Where("id = ?", p.ID).Updates(changes).Error; err != nil {
			return err
		}
		return passHistory(tx, p.ID, nil, actor, note)
	})
}

func (s *Store) ListTrainingAttendance(ctx context.Context, chatID int64, instanceID uint64) ([]TrainingAttendance, error) {
	g, err := s.getGroupByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	var e EventInstance
	if err = s.db.WithContext(ctx).Where("id = ? AND group_id = ?", instanceID, g.ID).Take(&e).Error; err != nil {
		return nil, passProblem("Событие не найдено в этой организации")
	}
	rows := make([]TrainingAttendance, 0)
	err = s.db.WithContext(ctx).Where("group_id = ? AND instance_id = ?", g.ID, instanceID).Order("user_telegram_id").Find(&rows).Error
	return rows, err
}

func (s *Store) SetTrainingAttendance(ctx context.Context, chatID, actor int64, instanceID uint64, userID int64, status string) error {
	if status != "attended" && status != "absent" && status != "unmarked" {
		return passProblem("Неизвестная отметка посещаемости")
	}
	return s.passTransaction(ctx, chatID, func(tx *gorm.DB, g *TelegramGroup) error {
		// Lock the event too: a concurrent scheduler cancellation must run after this transaction.
		var event EventInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND group_id = ?", instanceID, g.ID).Take(&event).Error; err != nil {
			return passProblem("Событие не найдено в этой организации")
		}
		if !event.IsActive || event.Status == "not_held" {
			return passProblem("Событие отменено. Занятия по абонементам возвращены")
		}
		if event.EventType != "training" {
			return passProblem("Абонементы действуют на тренировки")
		}
		if event.PlannedStartAt.After(time.Now()) {
			return passProblem("Посещаемость можно отметить после начала тренировки")
		}
		var n int64
		if err := tx.Model(&GroupMember{}).Where("group_id = ? AND user_telegram_id = ? AND is_active", g.ID, userID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return passProblem("Игрок не найден в этой организации")
		}
		var old TrainingAttendance
		err := tx.Where("instance_id = ? AND user_telegram_id = ?", instanceID, userID).Take(&old).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if old.Status == status {
			return nil
		}
		var passID *uint64
		if status == "attended" {
			passes := []TrainingPass{}
			// Never fall back silently to a one-off charge if a valid pass exists but is unpaid/frozen.
			date := passDate(event.LocalDate)
			if err = tx.Where("group_id = ? AND user_telegram_id = ? AND starts_on <= ? AND ends_on >= ? AND NOT is_closed", g.ID, userID, date, date).Order("ends_on ASC, id ASC").Find(&passes).Error; err != nil {
				return err
			}
			blocked := false
			for _, p := range passes {
				used, e := usedPassVisits(tx, p.ID)
				if e != nil {
					return e
				}
				if p.TotalVisits != nil && used >= *p.TotalVisits {
					continue
				}
				if !p.IsPaid || p.FrozenAt != nil {
					blocked = true
					continue
				}
				pid := p.ID
				passID = &pid
				break
			}
			if passID == nil && blocked {
				return passProblem("Абонемент не оплачен или заморожен. Сначала измените его статус")
			}
			// Cash and pass payment must not both cover the same attendance.
			if passID != nil {
				if err = tx.Table("event_settlement_payments p").Joins("JOIN event_settlements s ON s.id=p.settlement_id").Where("s.instance_id = ? AND s.group_id = ? AND p.user_id = ? AND COALESCE(p.cash_amount_paid, CASE WHEN p.is_paid THEN p.amount_due ELSE 0 END) >= p.amount_due AND p.amount_due > 0", instanceID, g.ID, userID).Count(&n).Error; err != nil {
					return err
				}
				if n > 0 {
					return passProblem("За эту тренировку уже отмечена разовая оплата. Сначала отмените её во взносах")
				}
			}
		}
		row := TrainingAttendance{GroupID: g.ID, InstanceID: instanceID, UserTelegramID: userID, Status: status, PassID: passID, ActorID: actor, UpdatedAt: time.Now()}
		if err = tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "instance_id"}, {Name: "user_telegram_id"}}, DoUpdates: clause.AssignmentColumns([]string{"status", "pass_id", "actor_id", "updated_at"})}).Create(&row).Error; err != nil {
			return err
		}
		if old.PassID != nil && old.Status == "attended" {
			if err = passHistory(tx, *old.PassID, &instanceID, actor, "Занятие возвращено: отметка присутствия отменена"); err != nil {
				return err
			}
		}
		if passID != nil {
			return passHistory(tx, *passID, &instanceID, actor, "Посещение тренировки по абонементу")
		}
		return nil
	})
}

type PassValidationError struct{ Message string }

func (e *PassValidationError) Error() string { return e.Message }
func passProblem(message string) error       { return &PassValidationError{Message: message} }

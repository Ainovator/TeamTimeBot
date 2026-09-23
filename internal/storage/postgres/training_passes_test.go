package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm/logger"
)

func TestTrainingPassState(t *testing.T) {
	n := 4
	today := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	p := TrainingPass{TotalVisits: &n, StartsOn: today, EndsOn: today, IsPaid: true}
	if got := trainingPassState(p, "2026-09-13"); got != "active" {
		t.Fatalf("inclusive end date: %s", got)
	}
	if got := trainingPassState(p, "2026-09-14"); got != "expired" {
		t.Fatal(got)
	}
	p.UsedVisits = 4
	if got := trainingPassState(p, "2026-09-13"); got != "exhausted" {
		t.Fatal(got)
	}
	p.TotalVisits = nil
	if got := trainingPassState(p, "2026-09-13"); got != "active" {
		t.Fatal(got)
	}
	p.FrozenAt = &today
	if got := trainingPassState(p, "2026-09-13"); got != "frozen" {
		t.Fatal(got)
	}
}

func TestTrainingPassLifecycleIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "tt_pass_test_" + fmt.Sprint(time.Now().UnixNano())
	if _, err = admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec("DROP SCHEMA " + schema + " CASCADE")
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	db, err := sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err = goose.Up(db, "../../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	s, err := New(parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	s.db.Logger = logger.Default.LogMode(logger.Silent)
	pool, _ := s.db.DB()
	defer pool.Close()
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	exec := func(query string, args ...interface{}) { t.Helper(); must(s.db.Exec(query, args...).Error) }
	exec("INSERT INTO telegram_groups(id,chat_id,title,timezone) VALUES (1,-101,'Test','Europe/Moscow'),(2,-102,'Other','Europe/Moscow')")
	exec("INSERT INTO telegram_users(telegram_id,first_name) VALUES (11,'Игрок'),(12,'Безлимит')")
	exec("INSERT INTO group_members(group_id,user_telegram_id) VALUES (1,11),(1,12)")
	exec("INSERT INTO group_events(id,group_id,name,start_weekday,start_time,end_time,poll_publish_weekday,poll_publish_time,announcement_lead_minutes) VALUES (1,1,'Test',1,'18:00','20:00',1,'12:00',60)")
	for i := 1; i <= 4; i++ {
		exec("INSERT INTO event_instances(id,group_id,event_id,local_date,planned_start_at,planned_end_at,status,event_type,event_name) VALUES (?,1,1,CURRENT_DATE - ?::integer,NOW()-INTERVAL '4 days',NOW()-INTERVAL '3 days','completed','training','Training')", i, i)
	}
	one := 1
	req := IssueTrainingPass{UserTelegramID: 11, Name: "1 занятие", TotalVisits: &one, StartsOn: time.Now().AddDate(0, 0, -10).Format("2006-01-02"), EndsOn: time.Now().AddDate(0, 0, 30).Format("2006-01-02"), IsPaid: true, PriceKopecks: 60000, RequestID: uuid.NewString()}
	pid, err := s.IssueTrainingPass(ctx, -101, 99, req)
	must(err)
	pid2, err := s.IssueTrainingPass(ctx, -101, 99, req)
	must(err)
	if pid != pid2 {
		t.Fatal("duplicate issue")
	}
	if _, err = s.IssueTrainingPass(ctx, -102, 99, req); err == nil {
		t.Fatal("cross-group issue allowed")
	}
	if err = s.ChangeTrainingPass(ctx, -102, 99, pid, TrainingPassAction{Action: "close", Version: 1}); err == nil {
		t.Fatal("cross-group change allowed")
	}
	if err = s.SetTrainingAttendance(ctx, -102, 99, 1, 11, "attended"); err == nil {
		t.Fatal("cross-group attendance allowed")
	}
	// Concurrent marking must consume the last visit at most once, including different events.
	var wg sync.WaitGroup
	for _, id := range []uint64{1, 1, 2, 2} {
		wg.Add(1)
		go func(event uint64) {
			defer wg.Done()
			if err := s.SetTrainingAttendance(ctx, -101, 99, event, 11, "attended"); err != nil {
				t.Error(err)
			}
		}(id)
	}
	wg.Wait()
	list, err := s.ListTrainingPasses(ctx, -101, 11)
	must(err)
	if len(list) != 1 || list[0].UsedVisits != 1 {
		t.Fatalf("overspent: %+v", list)
	}
	var attendance TrainingAttendance
	must(s.db.Where("pass_id = ? AND status='attended'", pid).Take(&attendance).Error)
	eventID := attendance.InstanceID
	exec("INSERT INTO event_settlements(id,group_id,event_id,instance_id,local_date,total_amount,participants_count,amount_per_person) SELECT 1,1,1,id,local_date,1200,2,600 FROM event_instances WHERE id=?", eventID)
	exec("INSERT INTO event_settlement_payments(settlement_id,user_id,amount_due) VALUES (1,11,1200)")
	bill, err := s.GetEventBillingByInstance(ctx, -101, eventID)
	must(err)
	if bill.Players[0].AmountDue != 600 || !bill.Players[0].PassCovered {
		t.Fatalf("guest should owe 600: %+v", bill)
	}
	must(s.SaveEventBillingPaymentsByInstance(ctx, -101, eventID, map[int64]bool{11: true}))
	must(s.SetTrainingAttendance(ctx, -101, 99, eventID, 11, "unmarked"))
	debt, err := s.GetGroupDebtSummary(ctx, -101)
	must(err)
	if debt.TotalDebt != 600 {
		t.Fatalf("cash for guest lost after undo: %+v", debt)
	}
	list, err = s.ListTrainingPasses(ctx, -101, 11)
	must(err)
	if list[0].UsedVisits != 0 {
		t.Fatal("undo did not return visit")
	}
	// A cancelled event returns the visit even if cancellation did not use the HTTP handler.
	must(s.SetTrainingAttendance(ctx, -101, 99, eventID, 11, "attended"))
	exec("UPDATE event_instances SET status='not_held' WHERE id=?", eventID)
	list, err = s.ListTrainingPasses(ctx, -101, 11)
	must(err)
	if list[0].UsedVisits != 0 {
		t.Fatal("cancel did not return visit")
	}
	if err = s.SetTrainingAttendance(ctx, -101, 99, eventID, 11, "attended"); err == nil {
		t.Fatal("cancelled event accepted")
	}
	must(s.ChangeTrainingPass(ctx, -101, 99, pid, TrainingPassAction{Action: "freeze", Version: 1}))
	if err = s.SetTrainingAttendance(ctx, -101, 99, 3, 11, "attended"); err == nil {
		t.Fatal("frozen pass accepted")
	}
	if err = s.ChangeTrainingPass(ctx, -101, 99, pid, TrainingPassAction{Action: "extend", Version: 1, Days: 5}); err == nil {
		t.Fatal("stale version accepted")
	}
	exec("UPDATE training_passes SET frozen_at=NOW()-INTERVAL '2 days' WHERE id=?", pid)
	must(s.ChangeTrainingPass(ctx, -101, 99, pid, TrainingPassAction{Action: "resume", Version: 2}))
	list, err = s.ListTrainingPasses(ctx, -101, 11)
	must(err)
	wantEnd, _ := time.Parse("2006-01-02", req.EndsOn)
	if passDate(list[0].EndsOn) != passDate(wantEnd.AddDate(0, 0, 2)) {
		t.Fatal("freeze duration not restored")
	}
	must(s.ChangeTrainingPass(ctx, -101, 99, pid, TrainingPassAction{Action: "unpaid", Version: 3}))
	if err = s.SetTrainingAttendance(ctx, -101, 99, 3, 11, "attended"); err == nil {
		t.Fatal("unpaid pass accepted")
	}
	must(s.ChangeTrainingPass(ctx, -101, 99, pid, TrainingPassAction{Action: "paid", Version: 4}))
	must(s.SetTrainingAttendance(ctx, -101, 99, 3, 11, "attended"))
	if err = s.ChangeTrainingPass(ctx, -101, 99, pid, TrainingPassAction{Action: "unpaid", Version: 5}); err == nil {
		t.Fatal("used pass payment removed")
	}
	must(s.SetTrainingAttendance(ctx, -101, 99, 3, 11, "absent"))
	list, err = s.ListTrainingPasses(ctx, -101, 11)
	must(err)
	if list[0].UsedVisits != 0 || len(list[0].History) < 7 {
		t.Fatal("missing refund or audit")
	}
	req.UserTelegramID = 12
	req.TotalVisits = nil
	req.RequestID = uuid.NewString()
	_, err = s.IssueTrainingPass(ctx, -101, 99, req)
	must(err)
	for _, id := range []uint64{3, 4} {
		must(s.SetTrainingAttendance(ctx, -101, 99, id, 12, "attended"))
	}
	list, err = s.ListTrainingPasses(ctx, -101, 12)
	must(err)
	if list[0].UsedVisits != 2 || list[0].State != "active" {
		t.Fatal("unlimited pass exhausted")
	}
	// Single seat, covered by a pass, cannot turn into a cash payment when saving all checkboxes.
	exec("INSERT INTO event_settlements(id,group_id,event_id,instance_id,local_date,total_amount,participants_count,amount_per_person) SELECT 2,1,1,id,local_date,600,1,600 FROM event_instances WHERE id=3")
	exec("INSERT INTO event_settlement_payments(settlement_id,user_id,amount_due) VALUES (2,12,600)")
	must(s.SaveEventBillingPaymentsByInstance(ctx, -101, 3, map[int64]bool{12: true}))
	must(s.SetTrainingAttendance(ctx, -101, 99, 3, 12, "unmarked"))
	bill, err = s.GetEventBillingByInstance(ctx, -101, 3)
	must(err)
	if bill.Players[0].IsPaid || bill.Players[0].AmountDue != 600 {
		t.Fatal("pass converted into cash")
	}
}

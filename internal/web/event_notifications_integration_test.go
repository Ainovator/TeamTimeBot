package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/pressly/goose/v3"
	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/internal/storage/postgres"
)

type telegramCapture struct{ calls []map[string]any }

func (capture *telegramCapture) RoundTrip(request *http.Request) (*http.Response, error) {
	var payload map[string]any
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = make(map[string]any)
	}
	method := request.URL.Path[strings.LastIndex(request.URL.Path, "/")+1:]
	payload["method"] = method
	response := `{"ok":true,"result":{"id":999,"first_name":"Test bot","username":"test_bot"}}`
	if method != "getMe" {
		capture.calls = append(capture.calls, payload)
		response = fmt.Sprintf(`{"ok":true,"result":{"message_id":%d,"poll":{"id":"test-poll-%d"}}}`, len(capture.calls)+700, len(capture.calls))
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response)), Request: request}, nil
}

func TestEventNotificationsIntegration(t *testing.T) {
	dsn := os.Getenv("TEAMTIME_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEAMTIME_TEST_DATABASE_URL to an isolated PostgreSQL database")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("event_notifications_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP SCHEMA " + pq.QuoteIdentifier(schema) + " CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	testDSN := parsed.String()
	db, err := sql.Open("postgres", testDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	goose.SetLogger(log.New(io.Discard, "", 0))
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	exec := func(statement string, args ...any) {
		t.Helper()
		if _, err := db.Exec(statement, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO telegram_groups(id,chat_id,title,timezone) VALUES (1,-100,'Команда <A>','UTC'),(2,-200,'Другая команда','UTC');
		INSERT INTO telegram_users(telegram_id,first_name) VALUES (101,'Анна & Оля'),(102,'Борис'),(103,'Другой'),(999,'Чужой');
		INSERT INTO group_members(group_id,user_telegram_id,real_name) VALUES (1,101,'Анна <b> & Оля'),(1,102,'Борис'),(1,103,'Другой'),(2,999,'Чужой');
		INSERT INTO poll_templates(group_id,name,question,options,counted_options) VALUES (1,'Запись','Кто играет?','["Да","Нет"]','[0]');`)
	store, err := postgres.New(testDSN)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	settings := postgres.EventMentionSettings{UserIDs: pq.Int64Array{102, 101, 101}, OnPoll: true, OnAnnouncement: true}
	create := func(mentions *postgres.EventMentionSettings) (*postgres.GroupEvent, error) {
		return store.CreateEvent(ctx, -100, "Тренировка <вечер>", "training", "Запись", 1, 1, "10:00", "19:00", "21:00", "Приходите <все> & друзья", true, 60, false, false, 6, 18, 0, 180, false, false, false, false, nil, mentions)
	}
	event, err := create(&settings)
	if err != nil {
		t.Fatal(err)
	}
	update := func(mentions *postgres.EventMentionSettings) error {
		return store.UpdateEventDetails(ctx, -100, event.ID, "Тренировка <вечер>", "training", 1, 1, "10:00", "19:00", "21:00", "Приходите <все> & друзья", true, 60, false, false, 6, 18, 0, 180, false, false, false, false, mentions)
	}
	capture := &telegramCapture{}
	bot, err := tele.NewBot("fixture-token", tele.WithHTTPClient(&http.Client{Transport: capture}))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, bot, Config{})
	post := func(path, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		return response
	}
	t.Run("settings persist, validate organization and preserve omitted fields", func(t *testing.T) {
		view, err := store.GetEventByID(ctx, -100, event.ID)
		if err != nil || len(view.Mentions.UserIDs) != 2 || !view.Mentions.OnPoll || !view.Mentions.OnAnnouncement {
			t.Fatalf("settings not saved: %#v, %v", view, err)
		}
		if _, err := create(&postgres.EventMentionSettings{UserIDs: pq.Int64Array{999}, OnPoll: true}); err == nil {
			t.Fatal("accepted a member from another group")
		}
		if _, err := create(&postgres.EventMentionSettings{UserIDs: pq.Int64Array{-1}, OnPoll: true}); err == nil {
			t.Fatal("accepted invalid Telegram ID")
		}
		changed := postgres.EventMentionSettings{UserIDs: pq.Int64Array{101}, OnAnnouncement: true}
		if err := update(&changed); err != nil {
			t.Fatal(err)
		}
		if err := update(nil); err != nil {
			t.Fatal(err)
		}
		view, err = store.GetEventByID(ctx, -100, event.ID)
		if err != nil || len(view.Mentions.UserIDs) != 1 || view.Mentions.OnPoll || !view.Mentions.OnAnnouncement {
			t.Fatalf("update lost settings: %#v, %v", view, err)
		}
		if err := update(&settings); err != nil {
			t.Fatal(err)
		}
	})
	exec(`INSERT INTO event_instances(id,group_id,event_id,local_date,planned_start_at,planned_end_at,status,event_name) VALUES
		(101,1,$1,'2026-09-10','2026-09-10 19:00Z','2026-09-10 21:00Z','on_review','Выбранная <тренировка>'),
		(102,1,$1,'2026-09-11','2026-09-11 19:00Z','2026-09-11 21:00Z','on_review','Другая тренировка')`, event.ID)
	exec(`INSERT INTO event_settlements(id,group_id,event_id,instance_id,local_date,total_amount,participants_count,amount_per_person) VALUES
		(201,1,$1,101,'2026-09-10',1800,3,600),(202,1,$1,102,'2026-09-11',9000,1,9000)`, event.ID)
	exec(`INSERT INTO event_settlement_payments(settlement_id,user_id,first_name,amount_due,is_paid) VALUES
		(201,101,'Анна',600,FALSE),(201,102,'Борис',600,TRUE),(201,103,'Другой',600,FALSE),(202,101,'Анна',9000,FALSE)`)
	t.Run("event debt uses selected instance, selected debtors and HTML mentions", func(t *testing.T) {
		capture.calls = nil
		response := post("/api/groups/-100/events/history/101/billing/publish", `{"userIDs":[101,102,999]}`)
		if response.Code != http.StatusOK {
			t.Fatalf("HTTP %d: %s", response.Code, response.Body.String())
		}
		if len(capture.calls) != 1 {
			t.Fatalf("unexpected calls: %#v", capture.calls)
		}
		payload := capture.calls[0]
		text := payload["text"].(string)
		for _, want := range []string{"tg://user?id=101", "600 ₽", "2026-09-10", "&lt;тренировка&gt;", "&lt;b&gt;"} {
			if !strings.Contains(text, want) {
				t.Fatalf("missing %q in %s", want, text)
			}
		}
		for _, unwanted := range []string{"tg://user?id=102", "tg://user?id=103", "tg://user?id=999", "9000", "<b>"} {
			if strings.Contains(text, unwanted) {
				t.Fatalf("included %q in %s", unwanted, text)
			}
		}
		if payload["parse_mode"] != "HTML" || payload["chat_id"] != "-100" {
			t.Fatalf("invalid send options: %#v", payload)
		}
		before := len(capture.calls)
		response = post("/api/groups/-200/events/history/101/billing/publish", `{"userIDs":[101]}`)
		if response.Code == http.StatusOK || len(capture.calls) != before {
			t.Fatal("published another organization's debt")
		}
		exec("UPDATE event_settlement_payments SET is_paid=TRUE WHERE settlement_id=201 AND user_id=101")
		response = post("/api/groups/-100/events/history/101/billing/publish", `{"userIDs":[101,102]}`)
		if response.Code == http.StatusOK || len(capture.calls) != before {
			t.Fatal("published debt for paid players")
		}
		exec("UPDATE event_settlement_payments SET is_paid=FALSE WHERE settlement_id=201 AND user_id=101")
	})
	t.Run("group debt mentions selected players and totals their trainings", func(t *testing.T) {
		capture.calls = nil
		if err := server.publishGroupDebtorsNow(ctx, -100, []int64{101}); err != nil {
			t.Fatal(err)
		}
		text := capture.calls[0]["text"].(string)
		if !strings.Contains(text, "9600 ₽") || !strings.Contains(text, "tg://user?id=101") || strings.Contains(text, "tg://user?id=103") || strings.Contains(text, "<A>") {
			t.Fatalf("wrong group debt: %s", text)
		}
	})
	t.Run("manual poll and announcement both mention configured members", func(t *testing.T) {
		capture.calls = nil
		if err := server.publishEventPollNow(ctx, -100, event.ID); err != nil {
			t.Fatal(err)
		}
		if len(capture.calls) != 2 || capture.calls[0]["method"] != "sendPoll" || capture.calls[1]["reply_to_message_id"] != "701" {
			t.Fatalf("poll invitation not attached: %#v", capture.calls)
		}
		if strings.Count(capture.calls[1]["text"].(string), "tg://user?id=") != 2 {
			t.Fatal("poll recipients missing")
		}
		if err := server.publishAnnouncementNow(ctx, -100, event.ID); err != nil {
			t.Fatal(err)
		}
		text := capture.calls[2]["text"].(string)
		if strings.Count(text, "tg://user?id=") != 2 || strings.Contains(text, "<все>") {
			t.Fatalf("invalid announcement: %s", text)
		}
	})
	t.Run("departed members and disabled modes are not mentioned", func(t *testing.T) {
		exec("UPDATE group_members SET status='left' WHERE group_id=1 AND user_telegram_id=102")
		members, err := store.GetEventMentionRecipients(ctx, -100, event.ID, "poll")
		if err != nil || len(members) != 1 || members[0].UserTelegramID != 101 {
			t.Fatalf("wrong recipients: %#v, %v", members, err)
		}
		if err := update(&postgres.EventMentionSettings{UserIDs: pq.Int64Array{101}, OnAnnouncement: true}); err != nil {
			t.Fatal(err)
		}
		members, err = store.GetEventMentionRecipients(ctx, -100, event.ID, "poll")
		if err != nil || len(members) != 0 {
			t.Fatal("disabled poll mentions still active")
		}
		if err := update(&postgres.EventMentionSettings{UserIDs: pq.Int64Array{}, OnPoll: true, OnAnnouncement: true}); err != nil {
			t.Fatal(err)
		}
		view, err := store.GetEventByID(ctx, -100, event.ID)
		if err != nil || view.Mentions.UserIDs == nil || len(view.Mentions.UserIDs) != 0 {
			t.Fatalf("clearing selection failed: %#v, %v", view, err)
		}
	})
	t.Run("migration rollback and reapply", func(t *testing.T) {
		if err := goose.DownTo(db, "../../db/migrations", 38); err != nil {
			t.Fatal(err)
		}
		if err := goose.Up(db, "../../db/migrations"); err != nil {
			t.Fatal(err)
		}
		if _, err := store.GetEventByID(ctx, -100, event.ID); err != nil {
			t.Fatal(err)
		}
	})
}

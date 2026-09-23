package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTrainingPassRoutesRequireAuthentication(t *testing.T) {
	s := NewServer(nil, nil, Config{TelegramLoginBotUsername: "test", TelegramBotToken: "test", SessionSecret: "test"})
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/groups/-1/passes"}, {"GET", "/api/groups/-1/passes/mine"},
		{"POST", "/api/groups/-1/passes"}, {"PATCH", "/api/groups/-1/passes/1"},
		{"GET", "/api/groups/-1/attendance/1"}, {"PUT", "/api/groups/-1/attendance/1"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, r)
			if w.Code != 401 {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

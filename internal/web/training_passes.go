package web

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

func (s *Server) handleTrainingPassRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	// Only the current user's pass list is available without financial administration rights.
	own := len(parts) == 1 && parts[0] == "mine" && r.Method == http.MethodGet
	perm := "roles_manage"
	if own {
		perm = "profile_read"
	}
	user, _, ok := s.requireGroupPermission(w, r, chatID, perm)
	if !ok {
		return
	}
	if (len(parts) == 0 || own) && r.Method == http.MethodGet {
		var uid int64
		if own {
			uid = user.ID
			if uid == 0 {
				writeJSON(w, http.StatusOK, []postgres.TrainingPass{})
				return
			}
		} else if v := r.URL.Query().Get("userID"); v != "" {
			var err error
			uid, err = strconv.ParseInt(v, 10, 64)
			if err != nil || uid <= 0 {
				writeErrorMessage(w, 400, "Некорректный игрок")
				return
			}
		}
		out, err := s.store.ListTrainingPasses(r.Context(), chatID, uid)
		if err != nil {
			writeErrorMessage(w, 500, "Не удалось загрузить абонементы. Повторите попытку")
			return
		}
		writeJSON(w, 200, out)
		return
	}
	if len(parts) == 0 && r.Method == http.MethodPost {
		var req postgres.IssueTrainingPass
		if err := decodeJSON(r, &req); err != nil {
			writeErrorMessage(w, 400, "Проверьте данные абонемента")
			return
		}
		id, err := s.store.IssueTrainingPass(r.Context(), chatID, user.ID, req)
		if err != nil {
			writePassError(w, err)
			return
		}
		writeJSON(w, 201, map[string]uint64{"id": id})
		return
	}
	if len(parts) == 1 && r.Method == http.MethodPatch {
		id, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil || id == 0 {
			writeErrorMessage(w, 400, "Некорректный абонемент")
			return
		}
		var req postgres.TrainingPassAction
		if err = decodeJSON(r, &req); err != nil {
			writeErrorMessage(w, 400, "Проверьте действие с абонементом")
			return
		}
		if err = s.store.ChangeTrainingPass(r.Context(), chatID, user.ID, id, req); err != nil {
			writePassError(w, err)
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
		return
	}
	writeMethodNotAllowed(w)
}

func (s *Server) handleTrainingAttendanceRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	user, _, ok := s.requireGroupPermission(w, r, chatID, "roles_manage")
	if !ok {
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeErrorMessage(w, 400, "Некорректное событие")
		return
	}
	if r.Method == http.MethodGet {
		rows, err := s.store.ListTrainingAttendance(r.Context(), chatID, id)
		if err != nil {
			writePassError(w, err)
			return
		}
		writeJSON(w, 200, rows)
		return
	}
	if r.Method == http.MethodPut {
		var req struct {
			UserTelegramID int64  `json:"userTelegramID"`
			Status         string `json:"status"`
		}
		if err = decodeJSON(r, &req); err != nil {
			writeErrorMessage(w, 400, "Проверьте отметку посещения")
			return
		}
		if err = s.store.SetTrainingAttendance(r.Context(), chatID, user.ID, id, req.UserTelegramID, req.Status); err != nil {
			writePassError(w, err)
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
		return
	}
	writeMethodNotAllowed(w)
}

func writePassError(w http.ResponseWriter, err error) {
	var problem *postgres.PassValidationError
	if errors.As(err, &problem) {
		writeErrorMessage(w, http.StatusBadRequest, problem.Message)
		return
	}
	log.Printf("training passes: %v", err)
	writeErrorMessage(w, http.StatusInternalServerError, "Не удалось сохранить изменение. Обновите страницу и повторите попытку")
}

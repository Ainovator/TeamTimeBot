package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/internal/storage/postgres"
)

type Config struct {
	StaticDir string
}

type Server struct {
	store     *postgres.Store
	staticDir string
	bot       *tele.Bot
}

type GroupDetails struct {
	Group         postgres.GroupView      `json:"group"`
	TemplateNames []string                `json:"templateNames"`
	Templates     []postgres.TemplateView `json:"templates"`
	Schedules     []postgres.ScheduleView `json:"schedules"`
	Events        []postgres.EventView    `json:"events"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewServer(store *postgres.Store, bot *tele.Bot, cfg Config) *Server {
	return &Server{store: store, bot: bot, staticDir: strings.TrimSpace(cfg.StaticDir)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/groups", s.handleGroups)
	mux.HandleFunc("/api/groups/", s.handleGroupRoutes)
	mux.HandleFunc("/", s.handleStaticOrInfo)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	groups, err := s.store.ListActiveGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (s *Server) handleGroupRoutes(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(rest, "/")
	chatID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		s.handleGroupDetails(w, r, chatID)
		return
	}

	switch parts[1] {
	case "members":
		s.handleMemberRoutes(w, r, chatID, parts[2:])
	case "templates":
		s.handleTemplateRoutes(w, r, chatID, parts[2:])
	case "schedules":
		s.handleScheduleRoutes(w, r, chatID, parts[2:])
	case "events":
		s.handleEventRoutes(w, r, chatID, parts[2:])
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleGroupDetails(w http.ResponseWriter, r *http.Request, chatID int64) {
	group, err := s.store.GetGroupByChatID(r.Context(), chatID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	templates, schedules, err := s.store.GetGroupSnapshot(r.Context(), chatID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	templateNames, err := s.store.ListTemplateNamesByChatID(r.Context(), chatID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	events, err := s.store.ListEventsByChatID(r.Context(), chatID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if templates == nil {
		templates = make([]postgres.TemplateView, 0)
	}
	if schedules == nil {
		schedules = make([]postgres.ScheduleView, 0)
	}
	if templateNames == nil {
		templateNames = make([]string, 0)
	}
	if events == nil {
		events = make([]postgres.EventView, 0)
	}

	writeJSON(w, http.StatusOK, GroupDetails{
		Group: postgres.GroupView{
			ChatID:   group.ChatID,
			Title:    group.Title,
			Timezone: group.Timezone,
		},
		TemplateNames: templateNames,
		Templates:     templates,
		Schedules:     schedules,
		Events:        events,
	})
}

func (s *Server) handleMemberRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if len(parts) == 0 {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		members, err := s.store.ListGroupMembersByChatID(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, members)
		return
	}

	if len(parts) == 1 && parts[0] == "skills" && r.Method == http.MethodGet {
		skills, err := s.store.ListSkillsCatalog(r.Context())
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if skills == nil {
			skills = make([]postgres.SkillCatalogItem, 0)
		}
		writeJSON(w, http.StatusOK, skills)
		return
	}

	if len(parts) == 2 && parts[1] == "skills" {
		userTelegramID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid user id")
			return
		}

		if r.Method == http.MethodGet {
			profile, err := s.store.GetMemberSkillProfile(r.Context(), chatID, userTelegramID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, profile)
			return
		}
		if r.Method == http.MethodPut {
			var req struct {
				Scores map[string]int `json:"scores"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if req.Scores == nil {
				req.Scores = map[string]int{}
			}
			if err := s.store.UpsertMemberSkills(r.Context(), chatID, userTelegramID, req.Scores); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
	}

	if len(parts) == 2 && parts[1] == "profile" && r.Method == http.MethodPut {
		userTelegramID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid user id")
			return
		}
		var req struct {
			PlayerType string `json:"playerType"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.store.UpdateMemberPlayerType(r.Context(), chatID, userTelegramID, req.PlayerType); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[1] == "relations" {
		userTelegramID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid user id")
			return
		}

		if r.Method == http.MethodGet {
			relations, err := s.store.ListPlayerRelationsByUser(r.Context(), chatID, userTelegramID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if relations == nil {
				relations = make([]postgres.PlayerRelationView, 0)
			}
			writeJSON(w, http.StatusOK, relations)
			return
		}

		if r.Method == http.MethodPost {
			var req struct {
				OtherUserID  int64  `json:"otherUserID"`
				RelationType string `json:"relationType"`
				Weight       int    `json:"weight"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if err := s.store.UpsertPlayerRelation(r.Context(), chatID, userTelegramID, req.OtherUserID, req.RelationType, req.Weight); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
	}

	if len(parts) == 4 && parts[1] == "relations" && r.Method == http.MethodDelete {
		userTelegramID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid user id")
			return
		}
		otherUserID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid related user id")
			return
		}
		relationType, err := urlPathUnescape(parts[3])
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid relation type")
			return
		}
		if err := s.store.DeletePlayerRelation(r.Context(), chatID, userTelegramID, otherUserID, relationType); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	writeMethodNotAllowed(w)
}

func (s *Server) handleTemplateRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if len(parts) == 0 {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req struct {
			Name           string   `json:"name"`
			Question       string   `json:"question"`
			Options        []string `json:"options"`
			CountedOptions []int    `json:"countedOptions"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if len(req.Options) < 2 {
			writeErrorMessage(w, http.StatusBadRequest, "need at least 2 options")
			return
		}
		if _, err := s.store.UpsertPollTemplateWithCounted(
			r.Context(),
			chatID,
			strings.TrimSpace(req.Name),
			strings.TrimSpace(req.Question),
			sanitizeOptions(req.Options),
			req.CountedOptions,
		); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 1 {
		templateName, err := urlPathUnescape(parts[0])
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid template name")
			return
		}

		switch r.Method {
		case http.MethodGet:
			template, err := s.store.GetTemplateByName(r.Context(), chatID, templateName)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeJSON(w, http.StatusOK, template)
			return
		case http.MethodPut:
			var req struct {
				Name           string   `json:"name"`
				Question       string   `json:"question"`
				Options        []string `json:"options"`
				CountedOptions []int    `json:"countedOptions"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			options := sanitizeOptions(req.Options)
			if strings.TrimSpace(req.Question) == "" {
				writeErrorMessage(w, http.StatusBadRequest, "question is required")
				return
			}
			if len(options) < 2 {
				writeErrorMessage(w, http.StatusBadRequest, "need at least 2 options")
				return
			}
			updated, err := s.store.UpdateTemplateByNameWithCounted(
				r.Context(),
				chatID,
				templateName,
				strings.TrimSpace(req.Name),
				strings.TrimSpace(req.Question),
				options,
				req.CountedOptions,
			)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, updated)
			return
		case http.MethodDelete:
			if err := s.store.DeleteTemplateByName(r.Context(), chatID, templateName); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		default:
			writeMethodNotAllowed(w)
			return
		}
	}

	writeMethodNotAllowed(w)
}

func (s *Server) handleScheduleRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if len(parts) == 0 {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req struct {
			TemplateName string `json:"templateName"`
			WeekdaysCSV  string `json:"weekdaysCsv"`
			SendAt       string `json:"sendAt"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		weekdays, err := postgres.ParseWeekdaysCSV(strings.TrimSpace(req.WeekdaysCSV))
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		expr, err := postgres.BuildScheduleExpr(weekdays, strings.TrimSpace(req.SendAt))
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if _, err := s.store.CreateSchedule(r.Context(), chatID, strings.TrimSpace(req.TemplateName), expr); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 1 && r.Method == http.MethodDelete {
		id, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid schedule id")
			return
		}
		if err := s.store.DeleteSchedule(r.Context(), chatID, id); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	writeMethodNotAllowed(w)
}

func (s *Server) handleEventRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if len(parts) == 0 {
		if r.Method == http.MethodGet {
			events, err := s.store.ListEventsByChatID(r.Context(), chatID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if events == nil {
				events = make([]postgres.EventView, 0)
			}
			writeJSON(w, http.StatusOK, events)
			return
		}
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req struct {
			Name                    string   `json:"name"`
			EventType               string   `json:"eventType"`
			Weekday                 int      `json:"weekday"`
			PublishWeekday          int      `json:"publishWeekday"`
			PublishAt               string   `json:"publishAt"`
			StartAt                 string   `json:"startAt"`
			EndAt                   string   `json:"endAt"`
			AnnouncementText        string   `json:"announcementText"`
			AnnouncementEnabled     *bool    `json:"announcementEnabled"`
			AnnouncementLeadMinutes int      `json:"announcementLeadMinutes"`
			TeamsAutoSplit          *bool    `json:"teamsAutoSplit"`
			TeamsPublishList        *bool    `json:"teamsPublishList"`
			TeamSize                int      `json:"teamSize"`
			MinVotesToHold          int      `json:"minVotesToHold"`
			CancelLeadMinutes       int      `json:"cancelLeadMinutes"`
			CancelNotifyEnabled     *bool    `json:"cancelNotifyEnabled"`
			SettlementEnabled       *bool    `json:"settlementEnabled"`
			SettlementPublishBefore *bool    `json:"settlementPublishBefore"`
			SettlementPublishAfter  *bool    `json:"settlementPublishAfter"`
			CostAmount              *float64 `json:"costAmount"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		publishWeekday := req.PublishWeekday
		if publishWeekday == 0 {
			publishWeekday = req.Weekday
		}
		announcementEnabled := false
		if req.AnnouncementEnabled != nil {
			announcementEnabled = *req.AnnouncementEnabled
		}
		settlementEnabled := true
		if req.SettlementEnabled != nil {
			settlementEnabled = *req.SettlementEnabled
		}
		settlementPublishBefore := false
		if req.SettlementPublishBefore != nil {
			settlementPublishBefore = *req.SettlementPublishBefore
		}
		settlementPublishAfter := true
		if req.SettlementPublishAfter != nil {
			settlementPublishAfter = *req.SettlementPublishAfter
		}
		teamsAutoSplit := false
		if req.TeamsAutoSplit != nil {
			teamsAutoSplit = *req.TeamsAutoSplit
		}
		teamsPublishList := false
		if req.TeamsPublishList != nil {
			teamsPublishList = *req.TeamsPublishList
		}
		teamSize := req.TeamSize
		if teamSize == 0 {
			teamSize = 6
		}
		cancelNotifyEnabled := false
		if req.CancelNotifyEnabled != nil {
			cancelNotifyEnabled = *req.CancelNotifyEnabled
		}
		created, err := s.store.CreateEvent(
			r.Context(),
			chatID,
			strings.TrimSpace(req.Name),
			req.EventType,
			req.Weekday,
			publishWeekday,
			strings.TrimSpace(req.PublishAt),
			strings.TrimSpace(req.StartAt),
			strings.TrimSpace(req.EndAt),
			req.AnnouncementText,
			announcementEnabled,
			req.AnnouncementLeadMinutes,
			teamsAutoSplit,
			teamsPublishList,
			teamSize,
			req.MinVotesToHold,
			req.CancelLeadMinutes,
			cancelNotifyEnabled,
			settlementEnabled,
			settlementPublishBefore,
			settlementPublishAfter,
			req.CostAmount,
		)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		event, err := s.store.GetEventByID(r.Context(), chatID, created.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, event)
		return
	}

	if len(parts) == 1 && parts[0] == "archived" && r.Method == http.MethodGet {
		events, err := s.store.ListArchivedEventsByChatID(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if events == nil {
			events = make([]postgres.EventView, 0)
		}
		writeJSON(w, http.StatusOK, events)
		return
	}

	if len(parts) == 1 && parts[0] == "history" && r.Method == http.MethodGet {
		history, err := s.store.ListEventHistory(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if history == nil {
			history = make([]postgres.EventHistoryItem, 0)
		}
		writeJSON(w, http.StatusOK, history)
		return
	}

	if len(parts) == 1 {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}

		switch r.Method {
		case http.MethodGet:
			event, err := s.store.GetEventByID(r.Context(), chatID, eventID)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeJSON(w, http.StatusOK, event)
			return
		case http.MethodPut:
			var req struct {
				Name string `json:"name"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if err := s.store.UpdateEventName(r.Context(), chatID, eventID, req.Name); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			event, err := s.store.GetEventByID(r.Context(), chatID, eventID)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeJSON(w, http.StatusOK, event)
			return
		default:
			writeMethodNotAllowed(w)
			return
		}
	}

	if len(parts) == 2 && parts[1] == "details" && r.Method == http.MethodPut {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}

		var req struct {
			Name                    string `json:"name"`
			EventType               string `json:"eventType"`
			Weekday                 int    `json:"weekday"`
			PublishWeekday          int    `json:"publishWeekday"`
			PublishAt               string `json:"publishAt"`
			StartAt                 string `json:"startAt"`
			EndAt                   string `json:"endAt"`
			AnnouncementText        string `json:"announcementText"`
			AnnouncementEnabled     *bool  `json:"announcementEnabled"`
			AnnouncementLeadMinutes int    `json:"announcementLeadMinutes"`
			TeamsAutoSplit          *bool  `json:"teamsAutoSplit"`
			TeamsPublishList        *bool  `json:"teamsPublishList"`
			TeamSize                int    `json:"teamSize"`
			MinVotesToHold          int    `json:"minVotesToHold"`
			CancelLeadMinutes       int    `json:"cancelLeadMinutes"`
			CancelNotifyEnabled     *bool  `json:"cancelNotifyEnabled"`
			SettlementEnabled       *bool  `json:"settlementEnabled"`
			SettlementPublishBefore *bool  `json:"settlementPublishBefore"`
			SettlementPublishAfter  *bool  `json:"settlementPublishAfter"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		publishWeekday := req.PublishWeekday
		if publishWeekday == 0 {
			publishWeekday = req.Weekday
		}
		announcementEnabled := false
		if req.AnnouncementEnabled != nil {
			announcementEnabled = *req.AnnouncementEnabled
		}
		settlementEnabled := true
		if req.SettlementEnabled != nil {
			settlementEnabled = *req.SettlementEnabled
		}
		settlementPublishBefore := false
		if req.SettlementPublishBefore != nil {
			settlementPublishBefore = *req.SettlementPublishBefore
		}
		settlementPublishAfter := true
		if req.SettlementPublishAfter != nil {
			settlementPublishAfter = *req.SettlementPublishAfter
		}
		teamsAutoSplit := false
		if req.TeamsAutoSplit != nil {
			teamsAutoSplit = *req.TeamsAutoSplit
		}
		teamsPublishList := false
		if req.TeamsPublishList != nil {
			teamsPublishList = *req.TeamsPublishList
		}
		teamSize := req.TeamSize
		if teamSize == 0 {
			teamSize = 6
		}
		cancelNotifyEnabled := false
		if req.CancelNotifyEnabled != nil {
			cancelNotifyEnabled = *req.CancelNotifyEnabled
		}

		if err := s.store.UpdateEventDetails(
			r.Context(),
			chatID,
			eventID,
			req.Name,
			req.EventType,
			req.Weekday,
			publishWeekday,
			req.PublishAt,
			req.StartAt,
			req.EndAt,
			req.AnnouncementText,
			announcementEnabled,
			req.AnnouncementLeadMinutes,
			teamsAutoSplit,
			teamsPublishList,
			teamSize,
			req.MinVotesToHold,
			req.CancelLeadMinutes,
			cancelNotifyEnabled,
			settlementEnabled,
			settlementPublishBefore,
			settlementPublishAfter,
		); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		event, err := s.store.GetEventByID(r.Context(), chatID, eventID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, event)
		return
	}

	if len(parts) == 2 && parts[1] == "bind" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		var req struct {
			TemplateName string `json:"templateName"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.store.BindEventToTemplate(r.Context(), chatID, eventID, strings.TrimSpace(req.TemplateName)); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[1] == "archive" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		if err := s.store.SetEventArchived(r.Context(), chatID, eventID, true); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[1] == "unarchive" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		if err := s.store.SetEventArchived(r.Context(), chatID, eventID, false); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[1] == "activity" && r.Method == http.MethodGet {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		summary, err := s.store.GetEventActivitySummary(r.Context(), chatID, eventID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, summary)
		return
	}

	if len(parts) == 2 && parts[1] == "polls" && r.Method == http.MethodGet {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		history, err := s.store.ListEventPollHistory(r.Context(), chatID, eventID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if history == nil {
			history = make([]postgres.EventPollHistoryItem, 0)
		}
		writeJSON(w, http.StatusOK, history)
		return
	}

	if len(parts) == 4 && parts[1] == "polls" && parts[3] == "teams" {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		postID, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid post id")
			return
		}

		if r.Method == http.MethodGet {
			state, err := s.store.GetEventTeamSplitState(r.Context(), chatID, eventID, postID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, state)
			return
		}

		if r.Method == http.MethodPut {
			var req struct {
				Assignments []postgres.TeamSplitAssignmentInput `json:"assignments"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if err := s.store.SaveEventTeamSplit(r.Context(), chatID, eventID, postID, req.Assignments); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			state, err := s.store.GetEventTeamSplitState(r.Context(), chatID, eventID, postID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, state)
			return
		}
	}

	if len(parts) == 5 && parts[1] == "polls" && parts[3] == "teams" && parts[4] == "publish" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		postID, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid post id")
			return
		}
		if s.bot == nil {
			writeErrorMessage(w, http.StatusBadRequest, "manual controls are unavailable: bot is not configured")
			return
		}
		if err := s.publishEventTeamSplitNow(r.Context(), chatID, eventID, postID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 5 && parts[1] == "polls" && parts[3] == "teams" && parts[4] == "autosplit" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		postID, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid post id")
			return
		}
		state, err := s.store.AutoSplitEventTeams(r.Context(), chatID, eventID, postID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, state)
		return
	}

	if len(parts) == 2 && parts[1] == "activate" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		if err := s.store.SetEventPublishEnabled(r.Context(), chatID, eventID, true); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[1] == "deactivate" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		if err := s.store.SetEventPublishEnabled(r.Context(), chatID, eventID, false); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[1] == "cost" && (r.Method == http.MethodPut || r.Method == http.MethodPost) {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		var req struct {
			CostAmount *float64 `json:"costAmount"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.store.UpdateEventCostAmount(r.Context(), chatID, eventID, req.CostAmount); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 3 && parts[1] == "manual" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		if s.bot == nil {
			writeErrorMessage(w, http.StatusBadRequest, "manual controls are unavailable: bot is not configured")
			return
		}

		switch parts[2] {
		case "announcement":
			if err := s.publishAnnouncementNow(r.Context(), chatID, eventID); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
		case "poll":
			if err := s.publishEventPollNow(r.Context(), chatID, eventID); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
		case "settlement":
			if err := s.publishSettlementNow(r.Context(), chatID, eventID); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
		default:
			writeMethodNotAllowed(w)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	writeMethodNotAllowed(w)
}

func (s *Server) publishAnnouncementNow(ctx context.Context, chatID int64, eventID uint64) error {
	event, err := s.store.GetEventByID(ctx, chatID, eventID)
	if err != nil {
		return err
	}
	if !event.PublishEnabled {
		return errors.New("event publications are disabled")
	}
	text := strings.TrimSpace(event.AnnouncementText)
	if text == "" {
		return errors.New("announcement text is empty")
	}

	group, err := s.store.GetGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(group.Timezone)
	if err != nil {
		loc = time.UTC
	}
	nowLocal := time.Now().In(loc)
	startHour, startMin, err := parseHourMinute(event.StartTime)
	if err != nil {
		return err
	}
	nextStart := nextEventStartLocal(nowLocal, event.StartWeekday, startHour, startMin)
	message := fmt.Sprintf(
		"Анонс события \"%s\"\n%s\nКогда: %s (%s)",
		event.Name,
		text,
		nextStart.Format("02.01.2006 15:04"),
		group.Timezone,
	)
	chat := tele.Chat{ID: chatID, Type: tele.ChatGroup}
	if err := s.bot.SendMessage(chat, message, nil); err != nil {
		return err
	}
	return nil
}

func (s *Server) publishEventPollNow(ctx context.Context, chatID int64, eventID uint64) error {
	event, err := s.store.GetEventByID(ctx, chatID, eventID)
	if err != nil {
		return err
	}
	if !event.PublishEnabled {
		return errors.New("event publications are disabled")
	}
	if strings.TrimSpace(event.PollTemplate) == "" {
		return errors.New("no poll template is bound to event")
	}

	template, err := s.store.GetEventTemplateDetails(ctx, chatID, eventID)
	if err != nil {
		return err
	}
	if len(template.TemplateOptions) < 2 {
		return errors.New("template requires at least 2 options")
	}

	chat := tele.Chat{ID: chatID, Type: tele.ChatGroup}
	sent, err := s.bot.SendPollWithMeta(chat, template.TemplateQuestion, template.TemplateOptions, nil)
	if err != nil {
		return err
	}
	if sent != nil {
		eid := eventID
		if _, err := s.store.CreateEventPollPost(ctx, chatID, &eid, template.TemplateName, sent.MessageID, sent.PollID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) publishSettlementNow(ctx context.Context, chatID int64, eventID uint64) error {
	event, err := s.store.GetEventByID(ctx, chatID, eventID)
	if err != nil {
		return err
	}
	if !event.PublishEnabled {
		return errors.New("event publications are disabled")
	}
	if strings.TrimSpace(event.PollTemplate) == "" {
		return errors.New("no poll template is bound to event")
	}
	countedOptions, err := s.store.GetEventTemplateCountedOptions(ctx, eventID)
	if err != nil {
		return err
	}
	if len(countedOptions) == 0 {
		return errors.New("template has no options marked with accounting flag")
	}

	post, err := s.store.GetLatestEventPollPost(ctx, eventID)
	if err != nil {
		return err
	}
	if post == nil {
		return errors.New("no published poll found for this event")
	}

	choices := make([]string, 0, len(countedOptions))
	for _, idx := range countedOptions {
		choices = append(choices, fmt.Sprintf("option_%d", idx))
	}
	participants, err := s.store.CountVotesForPostChoices(ctx, post.ID, choices)
	if err != nil {
		return err
	}

	totalAmount := 4000.0
	if event.CostAmount != nil {
		totalAmount = *event.CostAmount
	}
	perPerson := 0.0
	if participants > 0 {
		perPerson = totalAmount / float64(participants)
	}

	message := fmt.Sprintf(
		"Итоги тренировки \"%s\"\nУчастников: %d\nСтоимость на человека: %.2f ₽",
		event.Name,
		participants,
		perPerson,
	)
	if participants == 0 {
		message = fmt.Sprintf("Итоги тренировки \"%s\"\nНет голосов для расчета.", event.Name)
	}

	chat := tele.Chat{ID: chatID, Type: tele.ChatGroup}
	if err := s.bot.SendMessage(chat, message, nil); err != nil {
		return err
	}
	return nil
}

func (s *Server) publishEventTeamSplitNow(ctx context.Context, chatID int64, eventID, postID uint64) error {
	event, err := s.store.GetEventByID(ctx, chatID, eventID)
	if err != nil {
		return err
	}
	state, err := s.store.GetEventTeamSplitState(ctx, chatID, eventID, postID)
	if err != nil {
		return err
	}
	if state == nil || len(state.Players) == 0 {
		return errors.New("no players to publish")
	}

	teams := map[string][]postgres.TeamSplitPlayer{
		"A":          {},
		"B":          {},
		"C":          {},
		"unassigned": {},
	}
	for _, player := range state.Players {
		key := strings.ToUpper(strings.TrimSpace(player.Team))
		if key == "" {
			key = "unassigned"
		}
		if key != "A" && key != "B" && key != "C" {
			key = "unassigned"
		}
		teams[key] = append(teams[key], player)
	}
	for key := range teams {
		sort.SliceStable(teams[key], func(i, j int) bool {
			return teams[key][i].Position < teams[key][j].Position
		})
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Состав команд: %q\n", event.Name)
	fmt.Fprintf(&b, "Опрос #%d\n\n", postID)

	writeTeamBlock := func(code string) {
		list := teams[code]
		if len(list) == 0 {
			return
		}
		fmt.Fprintf(&b, "Команда %s (%d):\n", code, len(list))
		for idx, p := range list {
			fmt.Fprintf(&b, "%d. %s\n", idx+1, teamPlayerDisplayName(p))
		}
		b.WriteString("\n")
	}

	writeTeamBlock("A")
	writeTeamBlock("B")
	writeTeamBlock("C")

	if reserve := teams["unassigned"]; len(reserve) > 0 {
		fmt.Fprintf(&b, "Резерв (%d):\n", len(reserve))
		for idx, p := range reserve {
			fmt.Fprintf(&b, "%d. %s\n", idx+1, teamPlayerDisplayName(p))
		}
	}

	pairs := buildTeamForecastPairs(teams)
	if len(pairs) > 0 {
		b.WriteString("\n\nПрогноз:\n")
		for _, pair := range pairs {
			fmt.Fprintf(&b, "Команда %s %d%% / %d%% Команда %s\n", pair.Left, pair.LeftPercent, pair.RightPercent, pair.Right)
		}
	}

	message := strings.TrimSpace(b.String())
	if message == "" {
		return errors.New("nothing to publish")
	}

	chat := tele.Chat{ID: chatID, Type: tele.ChatGroup}
	return s.bot.SendMessage(chat, message, nil)
}

func teamPlayerDisplayName(p postgres.TeamSplitPlayer) string {
	full := strings.TrimSpace(strings.TrimSpace(p.FirstName + " " + p.LastName))
	if full != "" {
		return full
	}
	if strings.TrimSpace(p.Username) != "" {
		return "@" + strings.TrimSpace(p.Username)
	}
	return strconv.FormatInt(p.UserID, 10)
}

type teamForecastPair struct {
	Left         string
	Right        string
	LeftPercent  int
	RightPercent int
}

func buildTeamForecastPairs(teams map[string][]postgres.TeamSplitPlayer) []teamForecastPair {
	active := make([]string, 0, 3)
	for _, code := range []string{"A", "B", "C"} {
		if len(teams[code]) > 0 {
			active = append(active, code)
		}
	}
	if len(active) < 2 {
		return nil
	}

	scores := map[string]float64{"A": 0, "B": 0, "C": 0}
	for code, players := range teams {
		for _, p := range players {
			scores[code] += p.Rating
		}
	}

	makePair := func(left, right string) teamForecastPair {
		leftScore := scores[left]
		rightScore := scores[right]
		total := leftScore + rightScore
		leftProb := 0.5
		if total > 0 {
			leftProb = leftScore / total
		}
		leftPercent := int(leftProb * 100)
		if leftPercent < 0 {
			leftPercent = 0
		}
		if leftPercent > 100 {
			leftPercent = 100
		}
		return teamForecastPair{
			Left:         left,
			Right:        right,
			LeftPercent:  leftPercent,
			RightPercent: 100 - leftPercent,
		}
	}

	if len(active) == 2 {
		return []teamForecastPair{makePair(active[0], active[1])}
	}

	return []teamForecastPair{
		makePair("A", "B"),
		makePair("C", "A"),
		makePair("B", "C"),
	}
}

func parseHourMinute(value string) (int, int, error) {
	layouts := []string{"15:04:05", "15:04"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, strings.TrimSpace(value))
		if err == nil {
			return parsed.Hour(), parsed.Minute(), nil
		}
	}
	return 0, 0, errors.New("invalid time format")
}

func isoWeekday(day time.Weekday) int {
	if day == time.Sunday {
		return 7
	}
	return int(day)
}

func nextEventStartLocal(nowLocal time.Time, eventWeekday int, hour int, minute int) time.Time {
	if eventWeekday < 1 || eventWeekday > 7 {
		return nowLocal
	}
	daysAhead := eventWeekday - isoWeekday(nowLocal.Weekday())
	if daysAhead < 0 {
		daysAhead += 7
	}
	candidateDate := nowLocal.AddDate(0, 0, daysAhead)
	candidate := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(), hour, minute, 0, 0, nowLocal.Location())
	if !candidate.After(nowLocal) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return candidate
}

func (s *Server) handleStaticOrInfo(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	if s.staticDir == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("TeamTimeBot web API is running. Configure WEB_STATIC_DIR or start frontend dev server."))
		return
	}

	requested := filepath.Clean(r.URL.Path)
	if requested == "." || requested == "/" {
		s.serveIndex(w, r)
		return
	}

	fullPath := filepath.Join(s.staticDir, requested)
	if fileInfo, err := os.Stat(fullPath); err == nil && !fileInfo.IsDir() {
		http.ServeFile(w, r, fullPath)
		return
	}
	// SPA fallback
	s.serveIndex(w, r)
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(s.staticDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		writeErrorMessage(w, http.StatusNotFound, "frontend static files not found")
		return
	}
	http.ServeFile(w, r, indexPath)
}

func sanitizeOptions(options []string) []string {
	result := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option != "" {
			result = append(result, option)
		}
	}
	return result
}

func decodeJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeError(w http.ResponseWriter, code int, err error) {
	if err == nil {
		writeErrorMessage(w, code, "unknown error")
		return
	}
	writeErrorMessage(w, code, err.Error())
}

func writeErrorMessage(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, ErrorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func urlPathUnescape(value string) (string, error) {
	return url.PathUnescape(value)
}

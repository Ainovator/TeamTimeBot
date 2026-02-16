package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
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
	StaticDir                string
	TelegramLoginBotUsername string
	TelegramBotToken         string
	SessionSecret            string
}

type Server struct {
	store     *postgres.Store
	staticDir string
	bot       *tele.Bot
	auth      authConfig
}

type authConfig struct {
	enabled    bool
	loginBot   string
	botToken   string
	cookieName string
	secret     []byte
}

type authContextKey string

const authUserContextKey authContextKey = "auth_user"

type AuthUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	AuthDate  int64  `json:"authDate"`
}

type sessionClaims struct {
	UserID    int64  `json:"userId"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Exp       int64  `json:"exp"`
}

type telegramAuthPayload struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
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

func (s *Server) requireGroupPermission(w http.ResponseWriter, r *http.Request, chatID int64, perm string) (AuthUser, *postgres.GroupPermissionsView, bool) {
	if !s.auth.enabled {
		return AuthUser{}, &postgres.GroupPermissionsView{
			RoleCode:  "admin",
			RoleTitle: "Администратор",
			Permissions: map[string]bool{
				"members_read":           true,
				"members_write":          true,
				"templates_manage":       true,
				"event_templates_manage": true,
				"events_read":            true,
				"events_manage":          true,
				"polls_read":             true,
				"roles_manage":           true,
				"profile_read":           true,
			},
		}, true
	}
	user, ok := s.authUserFromRequest(r)
	if !ok {
		writeErrorMessage(w, http.StatusUnauthorized, "unauthorized")
		return AuthUser{}, nil, false
	}
	view, err := s.store.GetGroupPermissionsForUser(r.Context(), chatID, user.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return AuthUser{}, nil, false
	}
	if view == nil || !view.Permissions[perm] {
		writeErrorMessage(w, http.StatusForbidden, "forbidden")
		return AuthUser{}, view, false
	}
	return user, view, true
}

func NewServer(store *postgres.Store, bot *tele.Bot, cfg Config) *Server {
	loginBot := strings.TrimSpace(cfg.TelegramLoginBotUsername)
	botToken := strings.TrimSpace(cfg.TelegramBotToken)
	secret := strings.TrimSpace(cfg.SessionSecret)
	if secret == "" {
		secret = botToken
	}

	auth := authConfig{
		enabled:    loginBot != "" && botToken != "" && secret != "",
		loginBot:   loginBot,
		botToken:   botToken,
		cookieName: "tt_session",
		secret:     []byte(secret),
	}
	return &Server{
		store:     store,
		bot:       bot,
		staticDir: strings.TrimSpace(cfg.StaticDir),
		auth:      auth,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/auth/config", s.handleAuthConfig)
	mux.HandleFunc("/api/auth/me", s.handleAuthMe)
	mux.HandleFunc("/api/auth/telegram", s.handleAuthTelegram)
	mux.HandleFunc("/api/auth/logout", s.handleAuthLogout)
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
	if !s.auth.enabled {
		groups, err := s.store.ListActiveGroups(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, groups)
		return
	}
	authUser, ok := s.authUserFromRequest(r)
	if !ok {
		writeErrorMessage(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// When auth is enabled, allow both admins and regular members to log in.
	// We will gate sensitive endpoints separately based on role.
	groups, err := s.store.ListActiveGroupsForUser(r.Context(), authUser.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (s *Server) handleAuthConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":           s.auth.enabled,
		"telegramLoginBot":  s.auth.loginBot,
		"sessionCookieName": s.auth.cookieName,
	})
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	if !s.auth.enabled {
		writeJSON(w, http.StatusOK, map[string]interface{}{"enabled": false, "user": nil})
		return
	}
	user, ok := s.authUserFromRequest(r)
	if !ok {
		writeErrorMessage(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"enabled": true, "user": user})
}

func (s *Server) handleAuthTelegram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if !s.auth.enabled {
		writeErrorMessage(w, http.StatusBadRequest, "telegram auth is not configured")
		return
	}
	var req telegramAuthPayload
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, err := s.verifyTelegramLogin(req)
	if err != nil {
		writeErrorMessage(w, http.StatusUnauthorized, "telegram auth verification failed")
		return
	}
	token, err := s.buildSessionToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.auth.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "user": user})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.auth.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	case "me":
		s.handleMeRoutes(w, r, chatID, parts[2:])
	case "permissions":
		s.handlePermissionsRoutes(w, r, chatID, parts[2:])
	case "roles":
		s.handleRolesRoutes(w, r, chatID, parts[2:])
	case "members":
		s.handleMemberRoutes(w, r, chatID, parts[2:])
	case "templates":
		s.handleTemplateRoutes(w, r, chatID, parts[2:])
	case "polls":
		s.handlePollRoutes(w, r, chatID, parts[2:])
	case "registration":
		s.handleRegistrationRoutes(w, r, chatID, parts[2:])
	case "schedules":
		s.handleScheduleRoutes(w, r, chatID, parts[2:])
	case "events":
		s.handleEventRoutes(w, r, chatID, parts[2:])
	case "billing":
		s.handleBillingRoutes(w, r, chatID, parts[2:])
	case "games":
		s.handleGamesRoutes(w, r, chatID, parts[2:])
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleGamesRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if _, _, ok := s.requireGroupPermission(w, r, chatID, "events_read"); !ok {
		return
	}

	// GET /api/groups/:chatID/games
	if len(parts) == 0 {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		rows, err := s.store.ListGroupGames(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if rows == nil {
			rows = make([]postgres.GroupGameRow, 0)
		}
		writeJSON(w, http.StatusOK, rows)
		return
	}

	// GET /api/groups/:chatID/games/:instanceID/roster?team1=A&team2=B
	if len(parts) == 2 && parts[1] == "roster" && r.Method == http.MethodGet {
		instanceID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil || instanceID == 0 {
			writeErrorMessage(w, http.StatusBadRequest, "invalid instance id")
			return
		}
		team1 := strings.TrimSpace(r.URL.Query().Get("team1"))
		team2 := strings.TrimSpace(r.URL.Query().Get("team2"))
		if team1 == "" || team2 == "" {
			writeErrorMessage(w, http.StatusBadRequest, "team1 and team2 are required")
			return
		}
		out, err := s.store.GetGameRosterByInstance(r.Context(), chatID, instanceID, team1, team2)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if out == nil {
			out = &postgres.GameRosterResponse{
				Team1:        team1,
				Team2:        team2,
				Team1Players: []postgres.TeamSplitPlayer{},
				Team2Players: []postgres.TeamSplitPlayer{},
			}
		}
		writeJSON(w, http.StatusOK, out)
		return
	}

	writeMethodNotAllowed(w)
}

func (s *Server) handlePermissionsRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if r.Method != http.MethodGet || len(parts) != 0 {
		writeMethodNotAllowed(w)
		return
	}
	if !s.auth.enabled {
		writeJSON(w, http.StatusOK, postgres.GroupPermissionsView{
			RoleCode:  "admin",
			RoleTitle: "Администратор",
			Permissions: map[string]bool{
				"members_read":           true,
				"members_write":          true,
				"templates_manage":       true,
				"event_templates_manage": true,
				"events_read":            true,
				"events_manage":          true,
				"polls_read":             true,
				"roles_manage":           true,
				"profile_read":           true,
			},
		})
		return
	}
	user, ok := s.authUserFromRequest(r)
	if !ok {
		writeErrorMessage(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	view, err := s.store.GetGroupPermissionsForUser(r.Context(), chatID, user.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleRolesRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if !s.auth.enabled {
		writeErrorMessage(w, http.StatusBadRequest, "auth is not configured")
		return
	}
	user, ok := s.authUserFromRequest(r)
	if !ok {
		writeErrorMessage(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	perms, err := s.store.GetGroupPermissionsForUser(r.Context(), chatID, user.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if perms == nil || !perms.Permissions["roles_manage"] {
		writeErrorMessage(w, http.StatusForbidden, "forbidden")
		return
	}

	if len(parts) == 0 && r.Method == http.MethodGet {
		roles, err := s.store.ListGroupRoles(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if roles == nil {
			roles = make([]postgres.GroupRoleView, 0)
		}
		writeJSON(w, http.StatusOK, roles)
		return
	}

	if len(parts) == 1 && parts[0] == "assign" && r.Method == http.MethodPut {
		var req struct {
			UserTelegramID int64  `json:"userTelegramID"`
			RoleCode       string `json:"roleCode"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if req.UserTelegramID == 0 {
			writeErrorMessage(w, http.StatusBadRequest, "userTelegramID is required")
			return
		}
		if err := s.store.AssignGroupRole(r.Context(), chatID, req.UserTelegramID, req.RoleCode); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	writeMethodNotAllowed(w)
}

func (s *Server) handleMeRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if !s.auth.enabled {
		writeErrorMessage(w, http.StatusBadRequest, "auth is not configured")
		return
	}
	user, ok := s.authUserFromRequest(r)
	if !ok {
		writeErrorMessage(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	perms, err := s.store.GetGroupPermissionsForUser(r.Context(), chatID, user.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if perms == nil || !perms.Permissions["profile_read"] {
		writeErrorMessage(w, http.StatusForbidden, "forbidden")
		return
	}
	if len(parts) == 0 && r.Method == http.MethodGet {
		profile, err := s.store.GetUserGroupProfile(r.Context(), chatID, user.ID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, profile)
		return
	}
	writeMethodNotAllowed(w)
}

func (s *Server) handleRegistrationRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if _, _, ok := s.requireGroupPermission(w, r, chatID, "templates_manage"); !ok {
		return
	}

	if len(parts) == 1 && parts[0] == "publish" && r.Method == http.MethodPost {
		if s.bot == nil {
			writeErrorMessage(w, http.StatusBadRequest, "manual controls are unavailable: bot is not configured")
			return
		}
		template, err := s.store.EnsureDefaultRegistrationTemplate(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		chat := tele.Chat{ID: chatID, Type: tele.ChatGroup}
		sent, err := s.bot.SendPollWithMeta(chat, template.Question, template.Options, nil)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if sent == nil {
			writeErrorMessage(w, http.StatusBadRequest, "telegram did not return poll metadata")
			return
		}
		if _, err := s.store.CreateEventPollPost(r.Context(), chatID, nil, template.Name, sent.MessageID, sent.PollID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	writeMethodNotAllowed(w)
}

func (s *Server) handlePollRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if _, _, ok := s.requireGroupPermission(w, r, chatID, "polls_read"); !ok {
		return
	}

	if len(parts) == 0 && r.Method == http.MethodGet {
		items, err := s.store.ListGroupPollsByChatID(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if items == nil {
			items = make([]postgres.GroupPollItem, 0)
		}
		writeJSON(w, http.StatusOK, items)
		return
	}
	if len(parts) == 2 && parts[1] == "votes" && r.Method == http.MethodGet {
		postID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid post id")
			return
		}
		votes, err := s.store.ListGroupPollVotesByPostID(r.Context(), chatID, postID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if votes == nil {
			votes = make([]postgres.GroupPollVoteItem, 0)
		}
		writeJSON(w, http.StatusOK, votes)
		return
	}
	if len(parts) == 3 && parts[1] == "votes" && r.Method == http.MethodDelete {
		if _, _, ok := s.requireGroupPermission(w, r, chatID, "roles_manage"); !ok {
			return
		}
		postID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid post id")
			return
		}
		userID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid user id")
			return
		}
		choice := strings.TrimSpace(r.URL.Query().Get("choice"))
		var delErr error
		if choice != "" {
			delErr = s.store.DeleteGroupPollVoteChoiceByPostUser(r.Context(), chatID, postID, userID, choice)
		} else {
			// Backward-compat: if choice isn't provided, delete all choices for this user in this poll.
			delErr = s.store.DeleteGroupPollVotesByPostAndUser(r.Context(), chatID, postID, userID)
		}
		if delErr != nil {
			writeError(w, http.StatusBadRequest, delErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	writeMethodNotAllowed(w)
}

func (s *Server) handleBillingRoutes(w http.ResponseWriter, r *http.Request, chatID int64, parts []string) {
	if _, _, ok := s.requireGroupPermission(w, r, chatID, "roles_manage"); !ok {
		return
	}

	if len(parts) == 1 && parts[0] == "summary" && r.Method == http.MethodGet {
		summary, err := s.store.GetGroupDebtSummary(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, summary)
		return
	}
	if len(parts) == 1 && parts[0] == "debtors" && r.Method == http.MethodGet {
		debtors, err := s.store.ListGroupDebtors(r.Context(), chatID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if debtors == nil {
			debtors = make([]postgres.GroupDebtor, 0)
		}
		writeJSON(w, http.StatusOK, debtors)
		return
	}
	if len(parts) == 1 && parts[0] == "publish" && r.Method == http.MethodPost {
		var req struct {
			UserIDs []int64 `json:"userIDs"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.publishGroupDebtorsNow(r.Context(), chatID, req.UserIDs); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	writeMethodNotAllowed(w)
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
	if _, _, ok := s.requireGroupPermission(w, r, chatID, "members_read"); !ok {
		return
	}

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
			PlayerType string  `json:"playerType"`
			RealName   *string `json:"realName,omitempty"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.store.UpdateMemberPlayerType(r.Context(), chatID, userTelegramID, req.PlayerType); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if req.RealName != nil {
			if err := s.store.UpdateMemberRealName(r.Context(), chatID, userTelegramID, *req.RealName); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
	if _, _, ok := s.requireGroupPermission(w, r, chatID, "templates_manage"); !ok {
		return
	}

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
			OptionWeights  []int    `json:"optionWeights"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if len(req.Options) < 2 {
			writeErrorMessage(w, http.StatusBadRequest, "need at least 2 options")
			return
		}
		if _, err := s.store.UpsertPollTemplateWithCountedAndWeights(
			r.Context(),
			chatID,
			strings.TrimSpace(req.Name),
			strings.TrimSpace(req.Question),
			sanitizeOptions(req.Options),
			req.CountedOptions,
			req.OptionWeights,
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
				OptionWeights  []int    `json:"optionWeights"`
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
			updated, err := s.store.UpdateTemplateByNameWithCountedAndWeights(
				r.Context(),
				chatID,
				templateName,
				strings.TrimSpace(req.Name),
				strings.TrimSpace(req.Question),
				options,
				req.CountedOptions,
				req.OptionWeights,
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
	if _, _, ok := s.requireGroupPermission(w, r, chatID, "templates_manage"); !ok {
		return
	}

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
	if len(parts) > 0 && parts[0] == "history" {
		needed := "events_read"
		if r.Method != http.MethodGet {
			needed = "events_manage"
		}
		if _, _, ok := s.requireGroupPermission(w, r, chatID, needed); !ok {
			return
		}
	} else {
		if _, _, ok := s.requireGroupPermission(w, r, chatID, "event_templates_manage"); !ok {
			return
		}
	}

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
			TemplateName            string   `json:"templateName"`
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
			strings.TrimSpace(req.TemplateName),
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

	if len(parts) == 3 && parts[0] == "history" && parts[2] == "polls" && r.Method == http.MethodGet {
		instanceID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid instance id")
			return
		}
		history, err := s.store.ListEventPollHistoryByInstance(r.Context(), chatID, instanceID)
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

	if len(parts) == 4 && parts[0] == "history" && parts[2] == "billing" && parts[3] == "publish" && r.Method == http.MethodPost {
		instanceID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid instance id")
			return
		}
		var req struct {
			UserIDs []int64 `json:"userIDs"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.publishEventBillingDebtorsNow(r.Context(), chatID, instanceID, req.UserIDs); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 3 && parts[0] == "history" && parts[2] == "billing" {
		instanceID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid instance id")
			return
		}
		if r.Method == http.MethodGet {
			billing, err := s.store.GetEventBillingByInstance(r.Context(), chatID, instanceID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if billing == nil {
				writeJSON(w, http.StatusOK, nil)
				return
			}
			writeJSON(w, http.StatusOK, billing)
			return
		}
		if r.Method == http.MethodPost {
			billing, err := s.store.EnsureEventBillingByInstance(r.Context(), chatID, instanceID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, billing)
			return
		}
		if r.Method == http.MethodPut {
			var req struct {
				Statuses []struct {
					UserID int64 `json:"userID"`
					Paid   bool  `json:"paid"`
				} `json:"statuses"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			statuses := make(map[int64]bool, len(req.Statuses))
			for _, item := range req.Statuses {
				if item.UserID == 0 {
					continue
				}
				statuses[item.UserID] = item.Paid
			}
			if err := s.store.SaveEventBillingPaymentsByInstance(r.Context(), chatID, instanceID, statuses); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
	}

	if len(parts) == 3 && parts[0] == "history" && parts[2] == "sets" {
		instanceID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid instance id")
			return
		}
		if r.Method == http.MethodGet {
			rows, err := s.store.GetEventSetRowsByInstance(r.Context(), chatID, instanceID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if rows == nil {
				rows = make([]postgres.EventSetRow, 0)
			}
			writeJSON(w, http.StatusOK, rows)
			return
		}
		if r.Method == http.MethodPut {
			var req struct {
				Rows []postgres.EventSetRow `json:"rows"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if err := s.store.SaveEventSetRowsByInstance(r.Context(), chatID, instanceID, req.Rows); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
	}

	if len(parts) == 4 && parts[0] == "history" && parts[2] == "sets" && parts[3] == "publish" && r.Method == http.MethodPost {
		instanceID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid instance id")
			return
		}
		if err := s.publishEventSetRowsNow(r.Context(), chatID, instanceID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if len(parts) == 2 && parts[0] == "history" && r.Method == http.MethodDelete {
		instanceID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid instance id")
			return
		}
		if err := s.store.DeleteEventInstance(r.Context(), chatID, instanceID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

	if len(parts) == 2 && parts[1] == "instances" && r.Method == http.MethodPost {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		var req struct {
			LocalDate string `json:"localDate"`
		}
		if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		instance, err := s.store.CreateEventInstanceFromTemplate(r.Context(), chatID, eventID, req.LocalDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if s.bot == nil {
			writeErrorMessage(w, http.StatusBadRequest, "bot is not configured")
			return
		}
		if err := s.publishEventPollForInstanceNow(r.Context(), chatID, eventID, instance.ID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]uint64{"instanceID": instance.ID})
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

	if len(parts) == 2 && parts[1] == "billing" {
		eventID, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "invalid event id")
			return
		}
		if r.Method == http.MethodGet {
			billing, err := s.store.GetEventBilling(r.Context(), chatID, eventID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if billing == nil {
				writeJSON(w, http.StatusOK, nil)
				return
			}
			writeJSON(w, http.StatusOK, billing)
			return
		}
		if r.Method == http.MethodPut {
			var req struct {
				Statuses []struct {
					UserID int64 `json:"userID"`
					Paid   bool  `json:"paid"`
				} `json:"statuses"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			statuses := make(map[int64]bool, len(req.Statuses))
			for _, item := range req.Statuses {
				if item.UserID == 0 {
					continue
				}
				statuses[item.UserID] = item.Paid
			}
			if err := s.store.SaveEventBillingPayments(r.Context(), chatID, eventID, statuses); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
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

func (s *Server) publishEventPollForInstanceNow(ctx context.Context, chatID int64, eventID, instanceID uint64) error {
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
	if sent == nil {
		return errors.New("telegram did not return poll metadata")
	}
	if _, err := s.store.CreateEventPollPostForInstance(ctx, chatID, eventID, instanceID, template.TemplateName, sent.MessageID, sent.PollID); err != nil {
		return err
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
	countedOptions, optionWeights, err := s.store.GetEventTemplateCountedOptionsAndWeights(ctx, eventID)
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
	weightByChoice := make(map[string]int, len(countedOptions))
	for _, idx := range countedOptions {
		choice := fmt.Sprintf("option_%d", idx)
		choices = append(choices, choice)
		w := 1
		if idx >= 0 && idx < len(optionWeights) {
			w = optionWeights[idx]
		}
		if w <= 0 {
			w = 1
		}
		weightByChoice[choice] = w
	}
	payers, seats, err := s.store.ListSeatCountsForPostChoices(ctx, post.ID, choices, weightByChoice)
	if err != nil {
		return err
	}

	totalAmount := 4000.0
	if event.CostAmount != nil {
		totalAmount = *event.CostAmount
	}
	pricePerSeat := 0.0
	if seats > 0 {
		pricePerSeat = math.Ceil(totalAmount / float64(seats))
	}

	var message string
	if seats == 0 {
		message = fmt.Sprintf("Итоги тренировки \"%s\"\nНет голосов для расчета.", event.Name)
	} else {
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Итоги тренировки \"%s\"\n", event.Name))
		b.WriteString(fmt.Sprintf("Мест: %d\n", seats))
		b.WriteString(fmt.Sprintf("Цена за место: %.0f ₽\n\n", pricePerSeat))
		for _, p := range payers {
			if p.Seats <= 0 {
				continue
			}
			name := strings.TrimSpace(p.RealName)
			if name == "" {
				name = strings.TrimSpace(strings.TrimSpace(p.FirstName + " " + p.LastName))
			}
			if name == "" && strings.TrimSpace(p.Username) != "" {
				name = "@" + strings.TrimSpace(p.Username)
			}
			if name == "" {
				name = fmt.Sprintf("id:%d", p.UserID)
			}
			amount := pricePerSeat * float64(p.Seats)
			b.WriteString(fmt.Sprintf("%s — %.0f ₽\n", name, amount))
		}
		message = strings.TrimSpace(b.String())
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
			label := strconv.Itoa(idx + 1)
			if idx >= 6 {
				label = "З"
			}
			fmt.Fprintf(&b, "%s. %s\n", label, teamPlayerDisplayName(p))
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

	activeTeamCount := 0
	for _, code := range []string{"A", "B", "C"} {
		if len(teams[code]) > 0 {
			activeTeamCount++
		}
	}
	pairs := buildTeamForecastPairs(teams)
	if len(pairs) > 0 {
		if activeTeamCount > 2 {
			if first := chooseOpeningPair(pairs); first != nil {
				fmt.Fprintf(&b, "\n\nПервая игра: Команда %s vs Команда %s\n", first.Left, first.Right)
			} else {
				b.WriteString("\n\n")
			}
		} else {
			b.WriteString("\n\n")
		}
		b.WriteString("Прогноз:\n")
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

func (s *Server) publishEventSetRowsNow(ctx context.Context, chatID int64, instanceID uint64) error {
	name, localDate, err := s.store.GetEventInstanceHeader(ctx, chatID, instanceID)
	if err != nil {
		return err
	}
	rows, err := s.store.GetEventSetRowsByInstance(ctx, chatID, instanceID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return errors.New("no sets to publish")
	}

	type pairKey struct {
		A string
		B string
	}
	type pairScore struct {
		AWins int
		BWins int
	}
	summary := map[pairKey]pairScore{}

	var b strings.Builder
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Событие #%d", instanceID)
	}
	fmt.Fprintf(&b, "Партии: %q\n", name)
	if !localDate.IsZero() {
		fmt.Fprintf(&b, "Дата: %s\n", localDate.Format("2006-01-02"))
	}
	b.WriteString("\n")

	for _, r := range rows {
		t1 := strings.TrimSpace(r.Team1)
		t2 := strings.TrimSpace(r.Team2)
		if t1 == "" || t2 == "" || t1 == t2 {
			continue
		}
		fmt.Fprintf(&b, "%d) Команда %s %d:%d Команда %s\n", r.Ordinal, t1, r.Score1, r.Score2, t2)

		// Pair summary: normalize order (A,B) == (B,A)
		k := pairKey{A: t1, B: t2}
		swap := false
		if k.B < k.A {
			k.A, k.B = k.B, k.A
			swap = true
		}
		ps := summary[k]
		leftScore := r.Score1
		rightScore := r.Score2
		if swap {
			leftScore, rightScore = rightScore, leftScore
		}
		if leftScore > rightScore {
			ps.AWins++
		} else if rightScore > leftScore {
			ps.BWins++
		}
		summary[k] = ps
	}

	if len(summary) > 0 {
		b.WriteString("\nИтог по матчам:\n")
		keys := make([]pairKey, 0, len(summary))
		for k := range summary {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].A != keys[j].A {
				return keys[i].A < keys[j].A
			}
			return keys[i].B < keys[j].B
		})
		for _, k := range keys {
			ps := summary[k]
			fmt.Fprintf(&b, "Команда %s %d:%d Команда %s\n", k.A, ps.AWins, ps.BWins, k.B)
		}
	}

	message := strings.TrimSpace(b.String())
	if message == "" {
		return errors.New("nothing to publish")
	}

	chat := tele.Chat{ID: chatID, Type: tele.ChatGroup}
	return s.bot.SendMessage(chat, message, nil)
}

func billingPlayerDisplayName(p postgres.EventBillingParticipant) string {
	if rn := strings.TrimSpace(p.RealName); rn != "" {
		return rn
	}
	full := strings.TrimSpace(strings.TrimSpace(p.FirstName + " " + p.LastName))
	if full != "" {
		return full
	}
	if strings.TrimSpace(p.Username) != "" {
		return "@" + strings.TrimSpace(p.Username)
	}
	return strconv.FormatInt(p.UserID, 10)
}

func (s *Server) publishEventBillingDebtorsNow(ctx context.Context, chatID int64, instanceID uint64, includeUserIDs []int64) error {
	name, localDate, err := s.store.GetEventInstanceHeader(ctx, chatID, instanceID)
	if err != nil {
		return err
	}
	billing, err := s.store.GetEventBillingByInstance(ctx, chatID, instanceID)
	if err != nil {
		return err
	}
	if billing == nil {
		return errors.New("billing not found")
	}

	include := map[int64]bool(nil)
	if len(includeUserIDs) > 0 {
		include = make(map[int64]bool, len(includeUserIDs))
		for _, uid := range includeUserIDs {
			if uid != 0 {
				include[uid] = true
			}
		}
	}

	type item struct {
		Name   string
		Amount float64
	}
	list := make([]item, 0, len(billing.Players))
	total := 0.0
	for _, p := range billing.Players {
		if p.IsPaid || p.AmountDue <= 0 {
			continue
		}
		if include != nil && !include[p.UserID] {
			continue
		}
		name := billingPlayerDisplayName(p)
		list = append(list, item{Name: name, Amount: p.AmountDue})
		total += p.AmountDue
	}
	if len(list) == 0 {
		return errors.New("no debtors to publish")
	}

	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Amount != list[j].Amount {
			return list[i].Amount > list[j].Amount
		}
		return list[i].Name < list[j].Name
	})

	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Событие #%d", instanceID)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Задолженности за %q\n", name)
	if !localDate.IsZero() {
		fmt.Fprintf(&b, "Дата: %s\n", localDate.Format("2006-01-02"))
	}
	b.WriteString("\n")
	for _, it := range list {
		fmt.Fprintf(&b, "%s — %.0f ₽\n", it.Name, it.Amount)
	}
	fmt.Fprintf(&b, "\nИтого: %.0f ₽", total)

	message := strings.TrimSpace(b.String())
	chat := tele.Chat{ID: chatID, Type: tele.ChatGroup}
	return s.bot.SendMessage(chat, message, nil)
}

func groupDebtorDisplayName(d postgres.GroupDebtor) string {
	full := strings.TrimSpace(strings.TrimSpace(d.FirstName + " " + d.LastName))
	if full != "" {
		return full
	}
	if strings.TrimSpace(d.Username) != "" {
		return "@" + strings.TrimSpace(d.Username)
	}
	return strconv.FormatInt(d.UserID, 10)
}

func (s *Server) publishGroupDebtorsNow(ctx context.Context, chatID int64, includeUserIDs []int64) error {
	group, err := s.store.GetGroupByChatID(ctx, chatID)
	if err != nil {
		return err
	}
	debtors, err := s.store.ListGroupDebtors(ctx, chatID)
	if err != nil {
		return err
	}

	include := map[int64]bool(nil)
	if len(includeUserIDs) > 0 {
		include = make(map[int64]bool, len(includeUserIDs))
		for _, uid := range includeUserIDs {
			if uid != 0 {
				include[uid] = true
			}
		}
	}

	type item struct {
		Name   string
		Amount float64
	}
	list := make([]item, 0, len(debtors))
	total := 0.0
	for _, d := range debtors {
		if d.TotalDebt <= 0 {
			continue
		}
		if include != nil && !include[d.UserID] {
			continue
		}
		name := groupDebtorDisplayName(d)
		if rn := strings.TrimSpace(d.RealName); rn != "" && rn != name {
			name = fmt.Sprintf("%s (%s)", name, rn)
		}
		list = append(list, item{Name: name, Amount: d.TotalDebt})
		total += d.TotalDebt
	}
	if len(list) == 0 {
		return errors.New("no debtors to publish")
	}

	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Amount != list[j].Amount {
			return list[i].Amount > list[j].Amount
		}
		return list[i].Name < list[j].Name
	})

	var b strings.Builder
	title := strings.TrimSpace(group.Title)
	if title == "" {
		title = strconv.FormatInt(chatID, 10)
	}
	fmt.Fprintf(&b, "Задолженности · %s\n\n", title)
	for _, it := range list {
		fmt.Fprintf(&b, "%s — %.0f ₽\n", it.Name, it.Amount)
	}
	fmt.Fprintf(&b, "\nИтого: %.0f ₽", total)

	message := strings.TrimSpace(b.String())
	chat := tele.Chat{ID: chatID, Type: tele.ChatGroup}
	return s.bot.SendMessage(chat, message, nil)
}

func teamPlayerDisplayName(p postgres.TeamSplitPlayer) string {
	if rn := strings.TrimSpace(p.RealName); rn != "" {
		return rn
	}
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

func chooseOpeningPair(pairs []teamForecastPair) *teamForecastPair {
	if len(pairs) < 2 {
		return nil
	}
	out := pairs[rand.IntN(len(pairs))]
	return &out
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

func (s *Server) authUserFromRequest(r *http.Request) (AuthUser, bool) {
	if !s.auth.enabled {
		return AuthUser{}, false
	}
	cookie, err := r.Cookie(s.auth.cookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return AuthUser{}, false
	}
	claims, err := s.parseSessionToken(cookie.Value)
	if err != nil {
		return AuthUser{}, false
	}
	return AuthUser{
		ID:        claims.UserID,
		Username:  claims.Username,
		FirstName: claims.FirstName,
		LastName:  claims.LastName,
		AuthDate:  time.Now().Unix(),
	}, true
}

func (s *Server) verifyTelegramLogin(req telegramAuthPayload) (AuthUser, error) {
	if req.ID == 0 || req.AuthDate == 0 || strings.TrimSpace(req.Hash) == "" {
		return AuthUser{}, errors.New("invalid telegram auth payload")
	}
	authTime := time.Unix(req.AuthDate, 0)
	if authTime.Before(time.Now().Add(-24 * time.Hour)) {
		return AuthUser{}, errors.New("telegram auth payload expired")
	}

	values := map[string]string{
		"auth_date":  strconv.FormatInt(req.AuthDate, 10),
		"first_name": strings.TrimSpace(req.FirstName),
		"id":         strconv.FormatInt(req.ID, 10),
		"last_name":  strings.TrimSpace(req.LastName),
		"photo_url":  strings.TrimSpace(req.PhotoURL),
		"username":   strings.TrimSpace(req.Username),
	}
	keys := make([]string, 0, len(values))
	for k, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+values[key])
	}
	dataCheckString := strings.Join(lines, "\n")

	botSecretHash := sha256.Sum256([]byte(s.auth.botToken))
	mac := hmac.New(sha256.New, botSecretHash[:])
	_, _ = mac.Write([]byte(dataCheckString))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(req.Hash))) {
		return AuthUser{}, errors.New("invalid telegram hash")
	}

	return AuthUser{
		ID:        req.ID,
		Username:  strings.TrimSpace(req.Username),
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		AuthDate:  req.AuthDate,
	}, nil
}

func (s *Server) buildSessionToken(user AuthUser) (string, error) {
	claims := sessionClaims{
		UserID:    user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Exp:       time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.auth.secret)
	_, _ = mac.Write([]byte(payloadEncoded))
	signature := hex.EncodeToString(mac.Sum(nil))
	return payloadEncoded + "." + signature, nil
}

func (s *Server) parseSessionToken(token string) (sessionClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return sessionClaims{}, errors.New("invalid session token")
	}
	payloadEncoded := strings.TrimSpace(parts[0])
	signature := strings.TrimSpace(parts[1])
	if payloadEncoded == "" || signature == "" {
		return sessionClaims{}, errors.New("invalid session token parts")
	}
	mac := hmac.New(sha256.New, s.auth.secret)
	_, _ = mac.Write([]byte(payloadEncoded))
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(signature)) {
		return sessionClaims{}, errors.New("invalid session signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadEncoded)
	if err != nil {
		return sessionClaims{}, err
	}
	var claims sessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return sessionClaims{}, err
	}
	if claims.UserID == 0 || claims.Exp <= 0 || time.Now().Unix() > claims.Exp {
		return sessionClaims{}, errors.New("session expired")
	}
	return claims, nil
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
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

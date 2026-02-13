package main

import (
	"log"
	"net/http"
	"os"

	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/internal/config"
	"gopkg.in/telebot.v4/internal/storage/postgres"
	"gopkg.in/telebot.v4/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	store, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}

	var bot *tele.Bot
	if cfg.BotToken != "" {
		loadedBot, err := tele.NewBot(cfg.BotToken)
		if err != nil {
			log.Fatalf("failed to init bot for manual controls: %v", err)
		}
		bot = loadedBot
	}

	srv := web.NewServer(store, bot, web.Config{
		StaticDir:                os.Getenv("WEB_STATIC_DIR"),
		TelegramLoginBotUsername: os.Getenv("WEB_TELEGRAM_LOGIN_BOT"),
		TelegramBotToken:         cfg.BotToken,
		SessionSecret:            os.Getenv("WEB_SESSION_SECRET"),
	})

	addr := os.Getenv("WEB_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("web admin started on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("web server stopped: %v", err)
	}
}

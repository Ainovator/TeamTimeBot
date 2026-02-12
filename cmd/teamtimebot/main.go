package main

import (
	"log"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/config"
	"gopkg.in/telebot.v4/internal/storage/postgres"
	"gopkg.in/telebot.v4/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.BotToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	store, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	bot, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: cfg.PollerTimeout},
	})
	if err != nil {
		log.Fatal(err)
	}

	telegram.RegisterHandlers(bot, store)
	log.Println("TeamTimeBot started")
	bot.Start()
}

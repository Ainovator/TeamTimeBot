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

	bot, err := tele.NewBot(cfg.BotToken)
	if err != nil {
		log.Fatal(err)
	}

	messages := make(chan tele.Message, 100)
	bot.Listen(messages, cfg.PollerTimeout)

	telegram.RegisterHandlers(bot, store)
	log.Println("TeamTimeBot started")

	for message := range messages {
		bot.Serve(message)
	}
}

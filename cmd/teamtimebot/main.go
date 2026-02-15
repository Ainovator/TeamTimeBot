package main

import (
	"context"
	"log"

	tele "gopkg.in/telebot.v4"

	"gopkg.in/telebot.v4/internal/config"
	"gopkg.in/telebot.v4/internal/scheduler"
	"gopkg.in/telebot.v4/internal/storage/postgres"
	"gopkg.in/telebot.v4/internal/telegram"
	_ "gopkg.in/telebot.v4/internal/tzdata"
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
	bot.Errors = make(chan error, 100)

	messages := make(chan tele.Message, 100)
	bot.Messages = messages

	telegram.RegisterHandlers(bot, store)
	log.Println("TeamTimeBot started")
	go scheduler.NewPollScheduler(bot, store).Start(context.Background())
	go scheduler.NewEventPollScheduler(bot, store).Start(context.Background())
	go scheduler.NewEventAnnouncementScheduler(bot, store).Start(context.Background())
	go scheduler.NewEventCancellationScheduler(bot, store).Start(context.Background())
	go scheduler.NewEventSettlementScheduler(bot, store).Start(context.Background())

	callbacks := make(chan tele.Callback, 100)
	bot.Callbacks = callbacks
	pollAnswers := make(chan tele.PollAnswer, 100)
	bot.PollAnswers = pollAnswers
	go bot.Start(cfg.PollerTimeout)

	for {
		select {
		case message := <-messages:
			bot.Serve(message)
		case callback := <-callbacks:
			telegram.HandleCallback(bot, store, callback)
		case pollAnswer := <-pollAnswers:
			telegram.HandlePollAnswer(store, pollAnswer)
		case botErr := <-bot.Errors:
			log.Printf("telegram poller error: %v", botErr)
		}
	}
}

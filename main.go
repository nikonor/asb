package main

import (
	"context"
	"os"
	"time"

	"github.com/txix-open/isp-kit/log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nikonor/asb/controller"
	"github.com/nikonor/asb/repo"
	"github.com/nikonor/asb/service"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := log.New(log.WithLevel(log.DebugLevel))
	if err != nil {
		panic(err)
	}
	token, ok := os.LookupEnv("TLG_TOKEN")
	if !ok {
		logger.Fatal(ctx, "Не удалось получить токен из env::TLG_TOKEN")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Fatal(ctx, err)
	}
	// TODO: webhook

	bot.Debug = true

	logger.Debug(ctx, "Authorized on account "+bot.Self.UserName)

	// TODO: сделать конфинг
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	r, err := repo.New(ctx, logger)
	if err != nil {
		logger.Fatal(ctx, err)
	}
	s := service.New(logger, r)
	c := controller.New(logger, s)

	updates := bot.GetUpdatesChan(u)

	// TODO: переделать на воркеров
	for update := range updates {
		go func(update tgbotapi.Update) {
			newCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			needSend, msg, err := c.Message(newCtx, update)
			if err != nil {
				logger.Warn(ctx, err)
				return
			}

			if needSend {
				_, err = bot.Send(msg)
				if err != nil {
					logger.Warn(ctx, err)
				}
			}
		}(update)
	}
}

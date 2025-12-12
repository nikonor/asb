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
	ch := make(chan tgbotapi.Update, 1_000_000)

	for range 3 {
		go worker(ctx, logger, c, bot, ch)
	}

	// TODO: переделать на воркеров
	for update := range updates {
		ch <- update
	}
}

func worker(ctx context.Context, logger log.Logger, c *controller.Controller, bot *tgbotapi.BotAPI, ch chan tgbotapi.Update) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-ch:
			func(update tgbotapi.Update) {
				newCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				defer cancel()

				needSend, msg, idForDel, err := c.Message(newCtx, update)
				if err != nil {
					logger.Warn(ctx, err)
					return
				}

				if needSend {
					resp, err := bot.Send(msg)
					if err != nil {
						logger.Warn(ctx, err)
						return
					}
					if msg.ReplyMarkup != nil {
						c.SaveMessageLink(update, resp)
					}
				}

				if idForDel != 0 {
					delMsg := tgbotapi.NewDeleteMessage(getChatId(update), idForDel)
					_, _ = bot.Send(delMsg)
				}

			}(update)
		}
	}
}

func getChatId(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID
	}
	return 0
}

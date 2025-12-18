package main

import (
	"context"
	"os"
	"time"

	"github.com/nikonor/asb/domain"
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

	bot.Debug = false

	logger.Debug(ctx, "Authorized on account "+bot.Self.UserName)

	// TODO: сделать конфинг
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	receiverCh := make(chan tgbotapi.Update, 1_000_000)
	senderCh := make(chan domain.SendingObject, 1_000_000)

	r, err := repo.New(ctx, logger, senderCh)
	if err != nil {
		logger.Fatal(ctx, err)
	}
	s := service.New(logger, r)
	c := controller.New(logger, s)

	for range 3 { // TODO: cfg
		go receiverWorker(ctx, logger, c, receiverCh, senderCh)
	}

	for range 3 { // TODO: cfg
		go senderWorker(ctx, logger, bot, c, senderCh)
	}

	// TODO: переделать на воркеров
	for update := range updates {
		receiverCh <- update
	}
}

func senderWorker(ctx context.Context, logger log.Logger, bot *tgbotapi.BotAPI, c *controller.Controller, senderChan <-chan domain.SendingObject) {
	logger.Debug(ctx, "sender worker start")
	for {
		select {
		case <-ctx.Done():
			logger.Debug(ctx, "Sender worker done")
			return
		case msg := <-senderChan:
			logger.Debug(ctx, "Sender worker got message")
			// TODO: перепосылка
			resp, err := bot.Send(msg.Msg)
			if err != nil {
				logger.Warn(ctx, "error on send message::"+err.Error())
				// TODO: тут какая-то проблема при отравке delete
			}
			if msg.NeedSave {
				c.SaveMessageLink(msg.Update, resp)
			}
		}
	}
}

func receiverWorker(ctx context.Context, logger log.Logger, c *controller.Controller,
	receiverChan chan tgbotapi.Update, senderChan chan domain.SendingObject) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-receiverChan:
			func(update tgbotapi.Update) {
				newCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				defer cancel()

				needSend, msg, idForDel, err := c.HandleMessage(newCtx, update)
				if err != nil {
					logger.Warn(ctx, err)
					return
				}

				if needSend {
					s := domain.SendingObject{Msg: msg, Update: update}
					if msg.ReplyMarkup != nil {
						s.NeedSave = true
					}
					senderChan <- s
				}

				if idForDel != 0 {
					senderChan <- domain.SendingObject{Msg: tgbotapi.NewDeleteMessage(getChatId(update), idForDel)}
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

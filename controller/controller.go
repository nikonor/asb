package controller

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Service interface {
	IsExistUser(from int64) (bool, error)
}

type Controller struct {
	srv Service
}

func New(srv Service) *Controller {
	return &Controller{srv: srv}
}

func (c *Controller) Message(msg tgbotapi.Update) (tgbotapi.MessageConfig, error) {
	c.srv.IsExistUser(msg.Message.Chat.ID)

	out := tgbotapi.NewMessage(msg.Message.Chat.ID, msg.Message.Text)
	out.ReplyToMessageID = msg.Message.MessageID

	return out, nil
}

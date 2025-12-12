package controller

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nikonor/asb/domain"
	"github.com/txix-open/isp-kit/log"
)

type Service interface {
	IsUserExists(ctx context.Context, userId int64) (string, error)
	ValidateNewUser(ctx context.Context, userId int64, data string) (bool, error)
}

type Controller struct {
	logger log.Logger
	srv    Service
}

func New(logger log.Logger, srv Service) *Controller {
	return &Controller{logger: logger, srv: srv}
}

func (c *Controller) Message(ctx context.Context, msg tgbotapi.Update) (bool, tgbotapi.MessageConfig, error) {
	if msg.Message != nil {
		exist, err := c.srv.IsUserExists(ctx, msg.Message.From.ID)
		switch {
		case err != nil:
			return false, tgbotapi.MessageConfig{}, err
		case exist != domain.Exist:
			out := c.newMsg(msg.Message.Chat.ID,
				fmt.Sprintf("%s, %s!",
					msg.Message.From.FirstName+" "+msg.Message.From.LastName,
					"это ваше первое сообщение, подтвердите, что вы не бот"),
				msg.Message.MessageID)
			out = c.addButton(out, exist)

			return true, out, nil
		}
	}

	// а если это нажатие на кнопку?
	if msg.CallbackQuery != nil {
		ok, _ := c.srv.ValidateNewUser(ctx, msg.CallbackQuery.From.ID, msg.CallbackQuery.Data)
		if ok {
			// TODO: welcome to cfg
			return true,
				c.newMsg(msg.CallbackQuery.Message.Chat.ID, "welcome", msg.CallbackQuery.Message.MessageID), nil
		}
	}

	return false, tgbotapi.MessageConfig{}, nil
}

func (c *Controller) newMsg(chatId int64, txt string, replTo int) tgbotapi.MessageConfig {
	out := tgbotapi.NewMessage(chatId, txt)
	if replTo != 0 {
		out.ReplyToMessageID = replTo
	}
	return out
}

func (c *Controller) addButton(out tgbotapi.MessageConfig, key string) tgbotapi.MessageConfig {
	out.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(
			"Я не бот", // TODO: to cfg
			key),
		),
	)
	return out
}

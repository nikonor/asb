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
	ValidateNewUser(ctx context.Context, userId int64, data string) (bool, int, error)
	SaveMessageLink(userId int64, messageID int)
	SaveToQuery(ctx context.Context, chatId int64, userId int64, messageId int) error
}

type Controller struct {
	logger log.Logger
	srv    Service
	// TODO: очередь на удаление сообщений (храним id + время удаления. ФП каждую секунду до места в очереди, где время больше текущего)
	//		плюс она же должна удалять первое сообщение от бота
}

func New(logger log.Logger, srv Service) *Controller {
	return &Controller{logger: logger, srv: srv}
}

func (c *Controller) Message(ctx context.Context, msg tgbotapi.Update) (bool, tgbotapi.MessageConfig, int, error) {
	if msg.Message != nil {
		exist, err := c.srv.IsUserExists(ctx, msg.Message.From.ID)
		switch {
		case err != nil:
			return false, tgbotapi.MessageConfig{}, 0, err
		case exist == domain.Baned || exist == domain.TmpUser:
			return false, tgbotapi.MessageConfig{}, msg.Message.MessageID, nil
		case exist != domain.Exist:
			// TODO: что делать с повторым сообщением?
			if err = c.srv.SaveToQuery(ctx, msg.Message.Chat.ID, msg.Message.From.ID, msg.Message.MessageID); err != nil {
				c.logger.Warn(ctx, "error SaveToQuery::"+err.Error())
			}
			out := c.newMsg(msg.Message.Chat.ID,
				fmt.Sprintf("%s, %s!",
					msg.Message.From.FirstName+" "+msg.Message.From.LastName,
					"это ваше первое сообщение, подтвердите, что вы не бот"),
				msg.Message.MessageID)
			out = c.addButton(out, exist)

			return true, out, 0, nil
		}
	}

	// а если это нажатие на кнопку?
	if msg.CallbackQuery != nil {
		ok, qMessageId, _ := c.srv.ValidateNewUser(ctx, msg.CallbackQuery.From.ID, msg.CallbackQuery.Data)
		if ok {

			// TODO: welcome to cfg
			return true,
				c.newMsg(msg.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%s, %s!",
					msg.CallbackQuery.From.FirstName+" "+msg.CallbackQuery.From.LastName, "welcome"), 0),
				qMessageId, nil
		}
	}

	return false, tgbotapi.MessageConfig{}, 0, nil
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

func (c *Controller) SaveMessageLink(update tgbotapi.Update, resp tgbotapi.Message) {
	c.srv.SaveMessageLink(update.Message.From.ID, resp.MessageID)
}

package controller

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nikonor/asb/domain"
	"github.com/txix-open/isp-kit/log"
)

type Service interface {
	GetUserStatus(ctx context.Context, userId int64) string
	HandleCallback(ctx context.Context, userId int64, data string) int
	SaveMessageLink(userId int64, messageID int)
	SaveToPending(ctx context.Context, chatId int64, userId int64, messageId int) error
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

func (c *Controller) HandleMessage(ctx context.Context, msg tgbotapi.Update) (bool, tgbotapi.MessageConfig, int, error) {
	switch {
	case msg.Message != nil:
		return c.handleMessage(ctx, msg)
	case msg.CallbackQuery != nil:
		return c.handleCallbackQuery(ctx, msg)
		// TODO: остальные типы сообщений
	}

	return false, tgbotapi.MessageConfig{}, 0, nil
}

func newMsg(chatId int64, txt string, replTo int) tgbotapi.MessageConfig {
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

// needSend, msg, idForDel, err
func (c *Controller) handleMessage(ctx context.Context, msg tgbotapi.Update) (bool, tgbotapi.MessageConfig, int, error) {
	exist := c.srv.GetUserStatus(ctx, msg.Message.From.ID)
	switch {
	case exist == domain.Baned || exist == domain.TmpUser:
		return false, tgbotapi.MessageConfig{}, msg.Message.MessageID, nil
	case exist != domain.Exist:
		if err := c.srv.SaveToPending(ctx, msg.Message.Chat.ID, msg.Message.From.ID,
			msg.Message.MessageID); err != nil {
			c.logger.Warn(ctx, "error SaveToQuery::"+err.Error())
		}
		out := newMsg(msg.Message.Chat.ID,
			fmt.Sprintf("%s, %s!",
				msg.Message.From.FirstName+" "+msg.Message.From.LastName,
				"это ваше первое сообщение, подтвердите, что вы не бот"), // TODO: cfg
			msg.Message.MessageID)
		out = c.addButton(out, exist)

		return true, out, 0, nil
	default:
		return false, tgbotapi.MessageConfig{}, 0, nil
	}
}

// needSend, msg, idForDel, err
func (c *Controller) handleCallbackQuery(ctx context.Context, msg tgbotapi.Update) (bool,
	tgbotapi.MessageConfig, int, error) {
	qMessageId := c.srv.HandleCallback(ctx, msg.CallbackQuery.From.ID, msg.CallbackQuery.Data)
	if qMessageId != 0 {
		return true,
			newMsg(msg.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%s, %s!",
				msg.CallbackQuery.From.FirstName+" "+msg.CallbackQuery.From.LastName, "welcome"), 0),
			qMessageId, nil
	}
	return false, tgbotapi.MessageConfig{}, 0, nil
}

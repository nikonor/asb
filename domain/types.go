package domain

import (
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type SendingObject struct {
	Msg      tgbotapi.Chattable
	NeedSave bool
	Update   tgbotapi.Update
}

package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nikonor/asb/controller"
	"github.com/nikonor/asb/repo"
	"github.com/nikonor/asb/service"
)

type Controller interface {
	Message(msg tgbotapi.Update) (tgbotapi.MessageConfig, error)
}

func main() {
	token, ok := os.LookupEnv("TLG_TOKEN")
	if !ok {
		log.Panic("Не удалось получить токен из env::TLG_TOKEN")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}
	// TODO: webhook

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// TODO: сделать конфинг
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	r := repo.New()
	s := service.New(r)
	c := controller.New(s)

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %#v", update.Message.From.UserName, update)
			msg, err := c.Message(update)
			if err != nil {
				log.Println(err)
				continue
			}
			bot.Send(msg)
		}
	}
}

package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"log"
	telegram "workbot/internal"
)

func main() {
	bot, err := tgbotapi.NewBotAPI("6455469274:AAGPBBQ9HrWnGp6HdfP3OE2l6mGPFW-ngj4")
	if err != nil {
		panic(err)
	}
	bot.Debug = true

	telegramBot := telegram.NewBot(bot)
	if err := telegramBot.Start(); err != nil {
		log.Fatal(err)
	}
}

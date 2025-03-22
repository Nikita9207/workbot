package main

import (
	"database/sql"
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

	db, err := sql.Open("postgres", "user=postgres password=gospel dbname=postgres sslmode=disable")
	if err != nil {
		log.Println("Ошибка при открытии соединения с базой данных:", err)
		log.Panic(err)
	}
	defer db.Close()

	telegramBot := telegram.NewBot(bot, db)
	if err := telegramBot.Start(); err != nil {
		log.Fatal(err)
	}

}

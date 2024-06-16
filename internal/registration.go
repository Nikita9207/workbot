package internal

import (
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"log"
)

func (b *Bot) RegisterUser(update tgbotapi.Update, db *sql.DB, bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	db, err := sql.Open("postgres", "user=postgres password=simplepassword dbname=postgres sslmode=disable")
	if err != nil {
		log.Println("Ошибка при открытии соединения с базой данных:", err)
		log.Panic(err)
	}
	defer db.Close()

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите ваше имя:")
	bot.Send(msg)

	var name, surname, phone string
	var nameSet, surnameSet, phoneSet bool

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			continue
		}

		if !nameSet {
			name = update.Message.Text
			nameSet = true
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите вашу фамилию:")
			bot.Send(msg)
			continue
		}

		if !surnameSet {
			surname = update.Message.Text
			surnameSet = true
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите ваш номер телефона:")
			bot.Send(msg)
			continue
		}

		if !phoneSet {
			phone = update.Message.Text
			phoneSet = true
			break
		}
	}

	id := uuid.New().String()
	_, err = db.Exec("INSERT INTO registration.clients (id, name, surname, phone, created_at, updated_at) VALUES (ID, $1, $2, $3, NOW(), NOW())", id, name, surname, phone)
	if err != nil {
		log.Panic(err)
	}
	return
}

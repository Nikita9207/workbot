package internal

import (
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

const (
	commandStart = "start"
	commandInfo  = "info"
)

func (b *Bot) handelCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case commandStart:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Добро пожаловать!")

		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Регистрация"),
				tgbotapi.NewKeyboardButton("Записаться на тренировку"),
			),
		)
		msg.ReplyMarkup = keyboard

		if _, err := b.bot.Send(msg); err != nil {
			panic(err)
		}
	case commandInfo:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Здесь будет таблица всех!")
		if _, err := b.bot.Send(msg); err != nil {
			panic(err)
		}
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Пока я такого не умею =(")
		if _, err := b.bot.Send(msg); err != nil {
			panic(err)
		}
	}
}

func (b *Bot) handelMessage(message *tgbotapi.Message, update tgbotapi.Update, bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Регистрация"),
			tgbotapi.NewKeyboardButton("Записаться на тренировку"),
		),
	)
	msg.ReplyMarkup = keyboard

	switch message.Text {
	case "Регистрация":
		b.handleRegistration(update, bot, updates)
	case "Записаться на тренировку":
		//b.handleTrainingRegistration(message)
	default:
		if _, err := b.bot.Send(msg); err != nil {
			panic(err)
		}
	}

	if _, err := b.bot.Send(msg); err != nil {
		panic(err)
	}
}

func (b *Bot) handleRegistration(update tgbotapi.Update, bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {

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

	_, err = db.Exec("INSERT INTO public.clients (name, surname, phone, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())", name, surname, phone)
	if err != nil {
		log.Panic(err)
	}
	return
}

/*func (b *Bot) handleTrainingRegistration(message *tgbotapi.Message) {

}*/

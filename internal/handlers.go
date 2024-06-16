package internal

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const commandStart = "start"

func (b *Bot) handelCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case commandStart:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Добро пожаловать!")
		if _, err := b.bot.Send(msg); err != nil {
			panic(err)
		}
	case "Регистрация":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Почти, еще чуть чуть)")
		if _, err := b.bot.Send(msg); err != nil {
			panic(err)
		}
	case "Записаться на тренирвоку":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Почти разобрался как это сделать =)")
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

func (b *Bot) handelMessage(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)

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
}

func (b *Bot) handleKeyboard() {

}

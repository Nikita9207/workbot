package internal

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
	"time"
)

type Clients struct {
	Id        int
	Name      string
	Surname   string
	Phone     int
	CreatedAt *time.Time
}

const (
	commandStart = "start"
	commandInfo  = "info"
)

func (b *Bot) handelCommand(message *tgbotapi.Message, update tgbotapi.Update, bot *tgbotapi.BotAPI, db *sql.DB) {
	cs := make([]Clients, 0)
	s := make([]string, 0)
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
		result, err := b.db.Query("select id, name, surname, phone from public.clients")
		if err != nil {
			log.Fatal(err)
		}
		for result.Next() {
			var c Clients
			err = result.Scan(&c.Id, &c.Name, &c.Surname, &c.Phone)
			if err != nil {
				log.Fatal(err)
			}
			cs = append(cs, c)
			s = append(s, fmt.Sprintf("ID: %d\nИмя: %s\nФамилия: %s\nТелефон: %d\n", c.Id, c.Name, c.Surname, c.Phone))
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, strings.Join(s, ""))
		_, err = bot.Send(msg)
		if err != nil {
			log.Fatalf("Failed to send message: %v", err)
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

	_, err := b.db.Exec("INSERT INTO public.clients (name, surname, phone, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())", name, surname, phone)
	if err != nil {
		log.Panic(err)
	}
	return
}

/*func (b *Bot) handleTrainingRegistration(message *tgbotapi.Message) {

}*/

package internal

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"
	"net/http"
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

type KoboldResult struct {
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

type KoboldResponse struct {
	Results []KoboldResult `json:"results"`
}

//type Client struct {
//	Id int "json:'id'"
//	Username string "json:'username'"
//	FirstName string "json:'firstName'"
//	LastName string "json:'lastName'"
//	Email string "json:'email'"
//	Password string "json:'password'"
//	Phone int "json:'phone'"
//	UserStatus int "json:'userStatus'"
//}

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
				tgbotapi.NewKeyboardButton("Войти"),
				tgbotapi.NewKeyboardButton("Регистрация"),
				tgbotapi.NewKeyboardButton("AI"),
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

	switch message.Text {
	case "Войти":
		b.handlerComeIn(message, bot)
	case "Регистрация":
		b.handleRegistration(update, bot, updates)
	case "AI":
		b.kobold(bot, message)
	default:
		b.kobold(bot, message)
		//if _, err := b.bot.Send(msg); err != nil {
		//	panic(err)
		//}
	}

	if _, err := b.bot.Send(msg); err != nil {
		panic(err)
	}
}

func (b *Bot) kobold(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	url := "http://localhost:5001/api/v1/generate"
	fmt.Println("URL:>", url)

	var d = fmt.Sprintf(`{
  "max_context_length": 2048,
  "max_length": 100,
  "prompt": "%s",
  "quiet": false,
  "rep_pen": 1.1,
  "rep_pen_range": 256,
  "rep_pen_slope": 1,
  "temperature": 0.5,
  "tfs": 1,
  "top_a": 0,
  "top_k": 100,
  "top_p": 0.9,
  "typical": 1
}`, message.Text)
	var jsonStr = []byte(d)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	data := KoboldResponse{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("response Body:", string(body))
	msg := tgbotapi.NewMessage(message.Chat.ID, data.Results[0].Text)
	bot.Send(msg)
	//response, err := http.Post("http://localhost:5001/api/v1/generate")
	//if err != nil {
	//	log.Println(err)

	//	return
	//}
	//defer response.Body.Close()
	//
	//msg := tgbotapi.NewMessage(message.Chat.ID, response.Status)
	//bot.Send(msg)
}

func (b *Bot) handlerComeIn(message *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	msgText := fmt.Sprintf("Здравствуйте")
	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("AI"),
		),
	)
	msg.ReplyMarkup = keyboard
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

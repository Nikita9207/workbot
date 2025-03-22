package kobold

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"net/http"
)

type AiRequest struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float64
}

func kobold() {
	response, err := http.Get("http://localhost:5001")
	if err != nil {
		log.Println(err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка при обращении к API Kobold!")
		bot.Send(msg)
		return
	}
	defer response.Body.Close()

	msg := tgbotapi.NewMessage(message.Chat.ID, response.Status)
	bot.Send(msg)
}

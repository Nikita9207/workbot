package internal

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron"
)

func (b *Bot) handleTrainingRegistration(message *tgbotapi.Message) {
	// Создаем новый календарь
	c := cron.New()

	// Добавляем функцию, которая будет вызываться при выборе даты
	c.AddFunc("@every 1m", func() {
		// Здесь Вы можете обработать выбор даты пользователем
		// и записать его в базу данных или отправить сообщение
		fmt.Println("Пользователь выбрал дату для тренировки")
	})

	// Запускаем календарь
	c.Start()

	// Отправляем сообщение с календарем
	/*msg := tgbotapi.NewMessage(message.Chat.ID, "Выберите дату для тренировки:")
	if _, err := b.bot.Send(msg); err != nil {
		panic(err)
	}*/
}

package bot

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"workbot/internal/excel"
	"workbot/internal/models"
	"workbot/internal/training"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	commandStart = "start"
	commandInfo  = "info"
)

var userStates = struct {
	sync.RWMutex
	states map[int64]string
}{states: make(map[int64]string)}

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	switch message.Command() {
	case commandStart:
		// Проверяем, зарегистрирован ли пользователь
		var clientID int
		var name, surname string
		err := b.db.QueryRow("SELECT id, name, surname FROM public.clients WHERE telegram_id = $1", chatID).
			Scan(&clientID, &name, &surname)

		if err == nil {
			// Пользователь зарегистрирован — показываем меню клиента
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Добро пожаловать, %s!", name))
			keyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Записаться на тренировку"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Мои записи"),
					tgbotapi.NewKeyboardButton("Мои тренировки"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Экспорт в календарь"),
				),
			)
			msg.ReplyMarkup = keyboard
			if _, err := b.api.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
		} else {
			// Пользователь не зарегистрирован — показываем меню регистрации
			msg := tgbotapi.NewMessage(chatID, "Добро пожаловать!")
			keyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Регистрация"),
				),
			)
			msg.ReplyMarkup = keyboard
			if _, err := b.api.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
		}

	case commandInfo:
		b.handleInfoCommand(message)

	default:
		msg := tgbotapi.NewMessage(chatID, "Пока я такого не умею =(")
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
	}
}

func (b *Bot) handleInfoCommand(message *tgbotapi.Message) {
	rows, err := b.db.Query("SELECT id, name, surname, phone, COALESCE(birth_date, '') FROM public.clients")
	if err != nil {
		log.Println("Ошибка запроса клиентов:", err)
		b.sendError(message.Chat.ID, "Ошибка получения списка клиентов", err)
		return
	}
	defer rows.Close()

	var clients []string
	for rows.Next() {
		var c models.Client
		err = rows.Scan(&c.ID, &c.Name, &c.Surname, &c.Phone, &c.BirthDate)
		if err != nil {
			log.Println("Ошибка чтения данных:", err)
			continue
		}
		clients = append(clients, fmt.Sprintf("━━━━━━━━━━━━━━━\nID: %d\n%s %s\n%s\n%s\n",
			c.ID, c.Name, c.Surname, c.Phone, c.BirthDate))
	}

	if err := rows.Err(); err != nil {
		log.Println("Ошибка итерации по rows:", err)
	}

	if len(clients) == 0 {
		b.sendMessage(message.Chat.ID, "Список клиентов пуст")
		return
	}

	b.sendMessage(message.Chat.ID, "Список клиентов:\n"+strings.Join(clients, ""))
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.RLock()
	state := userStates.states[chatID]
	userStates.RUnlock()

	// Обработка состояний регистрации
	if strings.HasPrefix(state, "reg_") {
		b.processRegistration(message, state)
		return
	}

	// Обработка состояний бронирования
	if strings.HasPrefix(state, "booking_") {
		b.processBooking(message, state)
		return
	}

	if state == "awaiting_training" {
		b.handleTrainingInput(message)
		return
	}

	switch message.Text {
	case "Регистрация":
		b.startRegistration(message)
	case "Записаться на тренировку":
		b.handleBookTraining(message)
	case "Мои записи":
		b.handleMyAppointments(message)
	case "Мои тренировки":
		b.handleMyTrainings(message)
	case "Экспорт в календарь":
		b.handleExportCalendar(message)
	case "Отмена":
		b.handleCancel(message)
	case "Назад":
		b.restoreMainMenu(chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте /start для начала.")
		if _, err := b.api.Send(msg); err != nil {
			log.Println("Ошибка отправки сообщения:", err)
		}
	}
}

// handleMyTrainings показывает тренировки пользователя
func (b *Bot) handleMyTrainings(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Получаем ID клиента
	var clientID int
	err := b.db.QueryRow("SELECT id FROM public.clients WHERE telegram_id = $1", chatID).Scan(&clientID)
	if err != nil {
		b.sendMessage(chatID, "Вы не зарегистрированы. Используйте /start для регистрации.")
		return
	}

	// Получаем последние тренировки из Excel
	trainings, err := excel.GetClientTrainings(excel.FilePath, clientID, 5)
	if err != nil {
		b.sendError(chatID, "Ошибка загрузки тренировок.", err)
		return
	}

	if len(trainings) == 0 {
		b.sendMessage(chatID, "У вас пока нет записанных тренировок.")
		return
	}

	var result strings.Builder
	result.WriteString("Ваши последние тренировки:\n\n")
	for _, t := range trainings {
		result.WriteString(t)
		result.WriteString("\n")
	}

	b.sendMessage(chatID, result.String())
}

func (b *Bot) handleTrainingStart(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	var clientID int
	err := b.db.QueryRow("SELECT id FROM public.clients WHERE telegram_id = $1", chatID).Scan(&clientID)
	if err != nil {
		b.sendMessage(chatID, "Вы не зарегистрированы. Нажмите 'Регистрация' для начала.")
		return
	}

	setState(chatID, "awaiting_training")

	helpText := `Введите тренировку в формате:

Жим лежа 4x10x60
Присед 5x5x100
Подтягивания 3x12
Планка 3x60сек

Формат: Упражнение ПодходыxПовторыxВес

Можно указать дату в первой строке:
13.01.2026
Жим лежа 4x10x60
...

Если дата не указана — используется сегодня.`

	b.sendMessageWithKeyboard(chatID, helpText, createCancelKeyboard())
}

func (b *Bot) handleCancel(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	clearState(chatID)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Войти"),
			tgbotapi.NewKeyboardButton("Регистрация"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Записать тренировку"),
		),
	)
	b.sendMessageWithKeyboard(chatID, "Операция отменена.", keyboard)
}

func (b *Bot) handleTrainingInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleCancel(message)
		return
	}

	clearState(chatID)

	exercises, trainingDate, err := training.Parse(text)
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("Ошибка разбора тренировки: %v", err))
		b.restoreMainMenu(chatID)
		return
	}

	if len(exercises) == 0 {
		b.sendMessage(chatID, "Не найдено ни одного упражнения. Проверьте формат.")
		b.restoreMainMenu(chatID)
		return
	}

	var clientID int
	var name, surname string
	err = b.db.QueryRow("SELECT id, name, surname FROM public.clients WHERE telegram_id = $1", chatID).
		Scan(&clientID, &name, &surname)
	if err != nil {
		b.sendMessage(chatID, "Ошибка: клиент не найден.")
		b.restoreMainMenu(chatID)
		return
	}

	err = excel.SaveTrainingToExcel(excel.FilePath, b.db, clientID, name, surname, trainingDate, exercises)
	if err != nil {
		b.sendError(chatID, fmt.Sprintf("Ошибка сохранения: %v", err), err)
		b.restoreMainMenu(chatID)
		return
	}

	if err := excel.UpdateAllDashboards(excel.FilePath, b.db); err != nil {
		log.Printf("Ошибка обновления dashboard: %v", err)
	}

	confirmText := training.FormatConfirmation(exercises, trainingDate)
	b.sendMessage(chatID, confirmText)
	b.restoreMainMenu(chatID)
}

func (b *Bot) restoreMainMenu(chatID int64) {
	// Проверяем, зарегистрирован ли пользователь
	var exists bool
	if err := b.db.QueryRow("SELECT EXISTS(SELECT 1 FROM public.clients WHERE telegram_id = $1)", chatID).Scan(&exists); err != nil {
		log.Printf("Ошибка проверки клиента: %v", err)
		exists = false
	}

	var keyboard tgbotapi.ReplyKeyboardMarkup
	if exists {
		// Пользователь зарегистрирован — меню клиента
		keyboard = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Записаться на тренировку"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Мои записи"),
				tgbotapi.NewKeyboardButton("Мои тренировки"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Экспорт в календарь"),
			),
		)
	} else {
		// Пользователь не зарегистрирован — меню регистрации
		keyboard = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Регистрация"),
			),
		)
	}

	b.sendMessageWithKeyboard(chatID, "Выберите действие:", keyboard)
}

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
			msg := tgbotapi.NewMessage(chatID, b.tf("welcome_name", chatID, name))
			keyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(b.t("btn_book_training", chatID)),
					tgbotapi.NewKeyboardButton(b.t("btn_feedback", chatID)),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(b.t("btn_my_appointments", chatID)),
					tgbotapi.NewKeyboardButton(b.t("btn_my_trainings", chatID)),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(b.t("btn_my_progress", chatID)),
					tgbotapi.NewKeyboardButton(b.t("btn_export_calendar", chatID)),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(b.t("btn_settings", chatID)),
				),
			)
			msg.ReplyMarkup = keyboard
			if _, err := b.api.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
		} else {
			// Пользователь не зарегистрирован — показываем меню регистрации
			msg := tgbotapi.NewMessage(chatID, b.t("welcome", chatID))
			keyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(b.t("btn_registration", chatID)),
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

	// Обработка состояний обратной связи
	if strings.HasPrefix(state, "feedback_") {
		switch state {
		case "feedback_select_training":
			b.handleFeedbackSelectTraining(message)
		case "feedback_awaiting_input":
			b.handleFeedbackInput(message)
		}
		return
	}

	// Обработка состояний трекера прогресса
	if strings.HasPrefix(state, "progress_") {
		b.processProgressState(message, state)
		return
	}

	// Обработка состояний тренировки (ввод веса)
	if strings.HasPrefix(state, "workout_weight_") {
		exerciseIDStr := strings.TrimPrefix(state, "workout_weight_")
		b.handleWorkoutWeightInput(message, exerciseIDStr)
		return
	}

	switch message.Text {
	case "Регистрация", "Registration":
		b.startRegistration(message)
	case "Записаться на тренировку", "Book a training":
		b.handleBookTraining(message)
	case "Обратная связь", "Feedback":
		b.handleFeedbackStart(message)
	case "Мои записи", "My appointments":
		b.handleMyAppointments(message)
	case "Мои тренировки", "My trainings":
		b.handleMyTrainings(message)
	case "Экспорт в календарь", "Export to calendar":
		b.handleExportCalendar(message)
	case "Мой прогресс", "My progress":
		b.handleProgressMenu(message)
	case "📝 Записать прогресс", "📝 Record progress":
		b.handleStartProgress(chatID)
	case "📊 Мой прогресс", "📊 My progress":
		b.handleViewProgress(chatID)
	case "📈 Динамика веса", "📈 Weight dynamics":
		b.handleWeightDynamics(chatID)
	case "📏 Динамика замеров", "📏 Measurements dynamics":
		b.handleMeasurementsDynamics(chatID)
	case "⚙️ Настройки", "⚙️ Settings":
		b.handleSettingsMenu(message)
	case "Отмена", "Cancel":
		b.handleCancel(message)
	case "Назад", "Back":
		b.restoreMainMenu(chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, b.t("unknown_command_start", chatID))
		if _, err := b.api.Send(msg); err != nil {
			log.Println("Ошибка отправки сообщения:", err)
		}
	}
}

// handleMyTrainings показывает тренировки пользователя
func (b *Bot) handleMyTrainings(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Получаем ID клиента
	clientID, err := b.repo.Program.GetClientByTelegramID(chatID)
	if err != nil || clientID == 0 {
		b.sendMessage(chatID, b.t("reg_not_registered", chatID))
		return
	}

	// Сначала проверяем есть ли активная программа с тренировками
	program, err := b.repo.Program.GetActiveProgram(clientID)
	if err == nil && program != nil {
		// Показываем следующую тренировку из программы
		b.handleMyWorkouts(message)
		return
	}

	// Fallback: показываем из Excel
	trainings, err := excel.GetClientTrainings(excel.FilePath, clientID, 5)
	if err != nil {
		b.sendError(chatID, b.t("error", chatID), err)
		return
	}

	if len(trainings) == 0 {
		b.sendMessage(chatID, b.t("trainings_empty", chatID))
		return
	}

	var result strings.Builder
	result.WriteString(b.t("trainings_title", chatID) + "\n\n")
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

	// Возвращаем в главное меню
	b.restoreMainMenu(chatID)
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
		// Пользователь зарегистрирован — меню клиента с локализацией
		keyboard = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton(b.t("btn_book_training", chatID)),
				tgbotapi.NewKeyboardButton(b.t("btn_feedback", chatID)),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton(b.t("btn_my_appointments", chatID)),
				tgbotapi.NewKeyboardButton(b.t("btn_my_trainings", chatID)),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton(b.t("btn_my_progress", chatID)),
				tgbotapi.NewKeyboardButton(b.t("btn_export_calendar", chatID)),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton(b.t("btn_settings", chatID)),
			),
		)
	} else {
		// Пользователь не зарегистрирован — меню регистрации
		keyboard = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton(b.t("btn_registration", chatID)),
			),
		)
	}

	b.sendMessageWithKeyboard(chatID, b.t("choose_action", chatID), keyboard)
}

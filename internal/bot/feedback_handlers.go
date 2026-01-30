package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"workbot/internal/excel"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// feedbackState хранит состояние обратной связи для клиента
type feedbackState struct {
	TrainingIndex int    // индекс выбранной тренировки
	TrainingDate  string // дата тренировки
}

var clientFeedbackStates = make(map[int64]*feedbackState)

// handleFeedbackStart начинает процесс обратной связи - показывает список тренировок
func (b *Bot) handleFeedbackStart(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Получаем ID клиента
	var clientID int
	var name, surname string
	err := b.db.QueryRow("SELECT id, name, surname FROM public.clients WHERE telegram_id = $1", chatID).
		Scan(&clientID, &name, &surname)
	if err != nil {
		b.sendMessage(chatID, b.t("reg_not_registered", chatID))
		return
	}

	// Получаем последние тренировки из Excel
	trainings, err := excel.GetClientTrainings(excel.FilePath, clientID, 10)
	if err != nil {
		b.sendError(chatID, b.t("error", chatID), err)
		return
	}

	if len(trainings) == 0 {
		b.sendMessage(chatID, b.t("feedback_no_trainings", chatID))
		b.restoreMainMenu(chatID)
		return
	}

	// Создаём кнопки с тренировками
	var buttons [][]tgbotapi.KeyboardButton
	for i, t := range trainings {
		// Извлекаем дату из строки тренировки (первая строка содержит дату)
		lines := strings.Split(t, "\n")
		dateStr := "Training"
		if len(lines) > 0 {
			dateStr = strings.TrimSpace(lines[0])
		}
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("%d. %s", i+1, dateStr)),
		))
	}
	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(b.t("cancel", chatID)),
	))

	setState(chatID, "feedback_select_training")

	msg := tgbotapi.NewMessage(chatID, b.t("feedback_select_training", chatID))
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// handleFeedbackSelectTraining обрабатывает выбор тренировки
func (b *Bot) handleFeedbackSelectTraining(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" || text == "Cancel" {
		clearState(chatID)
		delete(clientFeedbackStates, chatID)
		b.restoreMainMenu(chatID)
		return
	}

	// Парсим номер тренировки
	parts := strings.SplitN(text, ".", 2)
	if len(parts) < 2 {
		b.sendMessage(chatID, "Выберите тренировку из списка.")
		return
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil || index < 1 {
		b.sendMessage(chatID, "Выберите тренировку из списка.")
		return
	}

	// Сохраняем выбор
	dateStr := strings.TrimSpace(parts[1])
	clientFeedbackStates[chatID] = &feedbackState{
		TrainingIndex: index - 1,
		TrainingDate:  dateStr,
	}

	setState(chatID, "feedback_awaiting_input")

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📅 %s\n\n%s", dateStr, b.t("feedback_enter", chatID)))
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(b.t("cancel", chatID)),
		),
	)
	b.api.Send(msg)
}

// handleFeedbackInput обрабатывает ввод обратной связи (текст)
func (b *Bot) handleFeedbackInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" || text == "Cancel" {
		clearState(chatID)
		delete(clientFeedbackStates, chatID)
		b.restoreMainMenu(chatID)
		return
	}

	b.saveFeedback(chatID, text)
}


// saveFeedback сохраняет обратную связь и отправляет тренеру
func (b *Bot) saveFeedback(chatID int64, feedbackText string) {
	state := clientFeedbackStates[chatID]
	if state == nil {
		b.sendMessage(chatID, "Ошибка: не выбрана тренировка.")
		b.restoreMainMenu(chatID)
		return
	}

	// Получаем данные клиента
	var clientID int
	var name, surname string
	err := b.db.QueryRow("SELECT id, name, surname FROM public.clients WHERE telegram_id = $1", chatID).
		Scan(&clientID, &name, &surname)
	if err != nil {
		b.sendMessage(chatID, "Ошибка: клиент не найден.")
		clearState(chatID)
		delete(clientFeedbackStates, chatID)
		b.restoreMainMenu(chatID)
		return
	}

	// Сохраняем обратную связь в Excel
	err = excel.SaveFeedback(excel.FilePath, clientID, state.TrainingDate, feedbackText)
	if err != nil {
		log.Printf("Ошибка сохранения feedback: %v", err)
		// Продолжаем — отправим тренеру даже если не сохранилось в Excel
	}

	// Отправляем тренеру (всем админам)
	b.notifyTrainersAboutFeedback(clientID, name, surname, state.TrainingDate, feedbackText)

	// Очищаем состояние
	clearState(chatID)
	delete(clientFeedbackStates, chatID)

	b.sendMessage(chatID, b.t("feedback_saved", chatID))
	b.restoreMainMenu(chatID)
}

// notifyTrainersAboutFeedback отправляет уведомление тренерам
func (b *Bot) notifyTrainersAboutFeedback(clientID int, name, surname, trainingDate, feedback string) {
	// Получаем всех админов
	rows, err := b.db.Query("SELECT telegram_id FROM public.admins")
	if err != nil {
		log.Printf("Ошибка получения админов: %v", err)
		return
	}
	defer rows.Close()

	notification := fmt.Sprintf(
		"Обратная связь от клиента\n\n"+
			"Клиент: %s %s\n"+
			"Тренировка: %s\n"+
			"Время: %s\n\n"+
			"Сообщение:\n%s",
		name, surname,
		trainingDate,
		time.Now().Format("02.01.2006 15:04"),
		feedback,
	)

	for rows.Next() {
		var adminTelegramID int64
		if err := rows.Scan(&adminTelegramID); err != nil {
			continue
		}

		msg := tgbotapi.NewMessage(adminTelegramID, notification)
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки уведомления админу %d: %v", adminTelegramID, err)
		}
	}
}

package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"workbot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// WorkoutSession хранит состояние текущей тренировки пользователя
type WorkoutSession struct {
	WorkoutID       int
	CurrentExercise int
	StartTime       time.Time
	Exercises       []models.WorkoutExercise
	CompletedCount  int
	SkippedCount    int
}

var workoutSessions = struct {
	sync.RWMutex
	sessions map[int64]*WorkoutSession
}{sessions: make(map[int64]*WorkoutSession)}

// getWorkoutSession возвращает сессию тренировки пользователя
func getWorkoutSession(chatID int64) *WorkoutSession {
	workoutSessions.RLock()
	defer workoutSessions.RUnlock()
	return workoutSessions.sessions[chatID]
}

// setWorkoutSession устанавливает сессию тренировки
func setWorkoutSession(chatID int64, session *WorkoutSession) {
	workoutSessions.Lock()
	defer workoutSessions.Unlock()
	workoutSessions.sessions[chatID] = session
}

// clearWorkoutSession очищает сессию тренировки
func clearWorkoutSession(chatID int64) {
	workoutSessions.Lock()
	defer workoutSessions.Unlock()
	delete(workoutSessions.sessions, chatID)
}

// handleSendWorkoutToClient отправляет следующую тренировку клиенту (для тренера)
func (b *Bot) handleSendWorkoutToClient(clientID int, adminChatID int64) {
	// Получаем telegram_id клиента
	var telegramID int64
	err := b.db.QueryRow("SELECT telegram_id FROM public.clients WHERE id = $1", clientID).Scan(&telegramID)
	if err != nil {
		b.sendMessage(adminChatID, "Ошибка: клиент не найден или не имеет telegram_id")
		return
	}

	// Получаем следующую ожидающую тренировку
	workout, err := b.repo.Program.GetNextPendingWorkout(clientID)
	if err != nil {
		b.sendError(adminChatID, "Ошибка получения тренировки", err)
		return
	}

	if workout == nil {
		b.sendMessage(adminChatID, "Нет ожидающих тренировок для этого клиента")
		return
	}

	// Отправляем клиенту
	b.sendWorkoutToClient(telegramID, workout)

	// Отмечаем как отправленную
	if err := b.repo.Program.MarkWorkoutSent(workout.ID); err != nil {
		log.Printf("Ошибка отметки тренировки как отправленной: %v", err)
	}

	b.sendMessage(adminChatID, fmt.Sprintf("✅ Тренировка \"%s\" отправлена клиенту", workout.Name))
}

// sendWorkoutToClient отправляет тренировку клиенту с inline-кнопками
func (b *Bot) sendWorkoutToClient(chatID int64, workout *models.Workout) {
	// Рассчитываем примерную длительность (2.5 мин на упражнение)
	estimatedDuration := len(workout.Exercises) * 3

	// Формируем сообщение
	text := b.tf("workout_info", chatID,
		workout.Name,
		workout.WeekNum,
		workout.DayNum,
		len(workout.Exercises),
		estimatedDuration,
	)

	// Кнопки
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.t("workout_start_btn", chatID),
				fmt.Sprintf("workout_start_%d", workout.ID),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.t("workout_later_btn", chatID),
				"workout_later",
			),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки тренировки клиенту: %v", err)
	}
}

// handleWorkoutCallback обрабатывает callback-запросы связанные с тренировкой
func (b *Bot) handleWorkoutCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// Отвечаем на callback
	callbackResponse := tgbotapi.NewCallback(callback.ID, "")
	b.api.Request(callbackResponse)

	switch {
	case strings.HasPrefix(data, "workout_start_"):
		// Начать тренировку
		workoutIDStr := strings.TrimPrefix(data, "workout_start_")
		workoutID, _ := strconv.Atoi(workoutIDStr)
		b.startWorkoutSession(chatID, workoutID, callback.Message.MessageID)

	case data == "workout_later":
		// Отложить
		b.editMessage(chatID, callback.Message.MessageID, b.t("cancelled", chatID), nil)

	case strings.HasPrefix(data, "workout_done_"):
		// Упражнение выполнено
		exerciseIDStr := strings.TrimPrefix(data, "workout_done_")
		exerciseID, _ := strconv.Atoi(exerciseIDStr)
		b.markExerciseDone(chatID, exerciseID, callback.Message.MessageID)

	case strings.HasPrefix(data, "workout_skip_"):
		// Упражнение пропущено
		exerciseIDStr := strings.TrimPrefix(data, "workout_skip_")
		exerciseID, _ := strconv.Atoi(exerciseIDStr)
		b.markExerciseSkipped(chatID, exerciseID, callback.Message.MessageID)

	case strings.HasPrefix(data, "workout_weight_"):
		// Изменить вес
		exerciseIDStr := strings.TrimPrefix(data, "workout_weight_")
		exerciseID, _ := strconv.Atoi(exerciseIDStr)
		b.askForWeight(chatID, exerciseID)

	case strings.HasPrefix(data, "workout_next_"):
		// Следующее упражнение
		b.showNextExercise(chatID, callback.Message.MessageID)

	case strings.HasPrefix(data, "workout_prev_"):
		// Предыдущее упражнение
		b.showPrevExercise(chatID, callback.Message.MessageID)

	case data == "workout_finish":
		// Завершить тренировку
		b.finishWorkout(chatID, callback.Message.MessageID)

	case strings.HasPrefix(data, "workout_feeling_"):
		// Самочувствие
		feeling := strings.TrimPrefix(data, "workout_feeling_")
		b.saveWorkoutFeeling(chatID, feeling, callback.Message.MessageID)

	case strings.HasPrefix(data, "workout_rpe_"):
		// RPE
		rpeStr := strings.TrimPrefix(data, "workout_rpe_")
		rpe, _ := strconv.Atoi(rpeStr)
		b.saveWorkoutRPE(chatID, rpe, callback.Message.MessageID)
	}
}

// startWorkoutSession начинает сессию тренировки
func (b *Bot) startWorkoutSession(chatID int64, workoutID int, messageID int) {
	workout, err := b.repo.Program.GetWorkoutByID(workoutID)
	if err != nil || workout == nil {
		b.sendMessage(chatID, b.t("error", chatID))
		return
	}

	// Создаём сессию
	session := &WorkoutSession{
		WorkoutID:       workoutID,
		CurrentExercise: 0,
		StartTime:       time.Now(),
		Exercises:       workout.Exercises,
		CompletedCount:  0,
		SkippedCount:    0,
	}
	setWorkoutSession(chatID, session)

	// Показываем первое упражнение
	b.showCurrentExercise(chatID, messageID)
}

// showCurrentExercise показывает текущее упражнение
func (b *Bot) showCurrentExercise(chatID int64, messageID int) {
	session := getWorkoutSession(chatID)
	if session == nil || session.CurrentExercise >= len(session.Exercises) {
		b.finishWorkout(chatID, messageID)
		return
	}

	exercise := session.Exercises[session.CurrentExercise]
	total := len(session.Exercises)
	current := session.CurrentExercise + 1

	// Формируем текст упражнения
	var text strings.Builder
	text.WriteString(b.tf("workout_exercise_title", chatID, current, total))
	text.WriteString("\n\n")
	text.WriteString(b.tf("workout_exercise_name", chatID, exercise.ExerciseName))
	text.WriteString("\n\n")
	text.WriteString(b.tf("workout_exercise_sets", chatID, exercise.Sets))
	text.WriteString("\n")
	text.WriteString(b.tf("workout_exercise_reps", chatID, exercise.Reps))

	if exercise.Weight > 0 {
		text.WriteString("\n")
		weightText := b.tf("workout_exercise_weight", chatID, exercise.Weight)
		if exercise.WeightPercent > 0 {
			weightText += b.tf("workout_exercise_weight_percent", chatID, int(exercise.WeightPercent))
		}
		text.WriteString(weightText)
	}

	if exercise.RestSeconds > 0 {
		text.WriteString("\n")
		text.WriteString(b.tf("workout_exercise_rest", chatID, exercise.RestSeconds))
	}

	if exercise.RPE > 0 {
		text.WriteString("\n")
		text.WriteString(b.tf("workout_exercise_rpe", chatID, exercise.RPE))
	}

	if exercise.Tempo != "" {
		text.WriteString("\n")
		text.WriteString(b.tf("workout_exercise_tempo", chatID, exercise.Tempo))
	}

	if exercise.Notes != "" {
		text.WriteString("\n\n")
		text.WriteString(b.tf("workout_exercise_notes", chatID, exercise.Notes))
	}

	// Кнопки
	var rows [][]tgbotapi.InlineKeyboardButton

	// Основные кнопки действий
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			b.t("workout_btn_done", chatID),
			fmt.Sprintf("workout_done_%d", exercise.ID),
		),
		tgbotapi.NewInlineKeyboardButtonData(
			b.t("workout_btn_skip", chatID),
			fmt.Sprintf("workout_skip_%d", exercise.ID),
		),
	))

	// Кнопка изменения веса
	if exercise.Weight > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.t("workout_btn_change_weight", chatID),
				fmt.Sprintf("workout_weight_%d", exercise.ID),
			),
		))
	}

	// Навигация
	var navRow []tgbotapi.InlineKeyboardButton
	if session.CurrentExercise > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(
			b.t("workout_btn_back", chatID),
			"workout_prev_",
		))
	}
	if session.CurrentExercise < len(session.Exercises)-1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(
			b.t("workout_btn_next", chatID),
			"workout_next_",
		))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	// Кнопка завершения
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			b.t("workout_btn_finish", chatID),
			"workout_finish",
		),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.editMessage(chatID, messageID, text.String(), &keyboard)
}

// markExerciseDone отмечает упражнение как выполненное
func (b *Bot) markExerciseDone(chatID int64, exerciseID int, messageID int) {
	session := getWorkoutSession(chatID)
	if session == nil {
		return
	}

	// Отмечаем в БД
	if err := b.repo.Program.MarkExerciseCompleted(exerciseID); err != nil {
		log.Printf("Ошибка отметки упражнения: %v", err)
	}

	session.CompletedCount++
	session.CurrentExercise++
	setWorkoutSession(chatID, session)

	// Показываем следующее упражнение
	b.showCurrentExercise(chatID, messageID)
}

// markExerciseSkipped отмечает упражнение как пропущенное
func (b *Bot) markExerciseSkipped(chatID int64, exerciseID int, messageID int) {
	session := getWorkoutSession(chatID)
	if session == nil {
		return
	}

	// Отмечаем в БД
	if err := b.repo.Program.MarkExerciseSkipped(exerciseID); err != nil {
		log.Printf("Ошибка отметки пропуска: %v", err)
	}

	session.SkippedCount++
	session.CurrentExercise++
	setWorkoutSession(chatID, session)

	// Показываем следующее упражнение
	b.showCurrentExercise(chatID, messageID)
}

// askForWeight запрашивает ввод веса
func (b *Bot) askForWeight(chatID int64, exerciseID int) {
	setState(chatID, fmt.Sprintf("workout_weight_%d", exerciseID))
	b.sendMessage(chatID, b.t("workout_enter_weight", chatID))
}

// handleWorkoutWeightInput обрабатывает ввод веса
func (b *Bot) handleWorkoutWeightInput(message *tgbotapi.Message, exerciseIDStr string) {
	chatID := message.Chat.ID

	weight, err := strconv.ParseFloat(strings.TrimSpace(message.Text), 64)
	if err != nil || weight <= 0 {
		b.sendMessage(chatID, b.t("progress_invalid_number", chatID))
		return
	}

	exerciseID, _ := strconv.Atoi(exerciseIDStr)

	// Обновляем вес в БД (используем фактический вес)
	if err := b.repo.Program.UpdateExerciseResult(exerciseID, 0, 0, weight, 0); err != nil {
		log.Printf("Ошибка обновления веса: %v", err)
	}

	clearState(chatID)
	b.sendMessage(chatID, b.tf("workout_weight_saved", chatID, weight))
}

// showNextExercise показывает следующее упражнение
func (b *Bot) showNextExercise(chatID int64, messageID int) {
	session := getWorkoutSession(chatID)
	if session == nil {
		return
	}

	if session.CurrentExercise < len(session.Exercises)-1 {
		session.CurrentExercise++
		setWorkoutSession(chatID, session)
	}

	b.showCurrentExercise(chatID, messageID)
}

// showPrevExercise показывает предыдущее упражнение
func (b *Bot) showPrevExercise(chatID int64, messageID int) {
	session := getWorkoutSession(chatID)
	if session == nil {
		return
	}

	if session.CurrentExercise > 0 {
		session.CurrentExercise--
		setWorkoutSession(chatID, session)
	}

	b.showCurrentExercise(chatID, messageID)
}

// finishWorkout завершает тренировку
func (b *Bot) finishWorkout(chatID int64, messageID int) {
	session := getWorkoutSession(chatID)
	if session == nil {
		return
	}

	duration := int(time.Since(session.StartTime).Minutes())
	total := len(session.Exercises)

	// Формируем сообщение о завершении
	var text strings.Builder
	text.WriteString(b.t("workout_post_title", chatID))
	text.WriteString("\n\n")
	text.WriteString(b.tf("workout_post_stats", chatID, session.CompletedCount, total, duration))
	text.WriteString("\n\n")
	text.WriteString(b.t("workout_post_feeling", chatID))

	// Кнопки самочувствия
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.t("workout_post_feeling_great", chatID),
				"workout_feeling_great",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				b.t("workout_post_feeling_good", chatID),
				"workout_feeling_good",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.t("workout_post_feeling_tired", chatID),
				"workout_feeling_tired",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				b.t("workout_post_feeling_bad", chatID),
				"workout_feeling_bad",
			),
		),
	)

	b.editMessage(chatID, messageID, text.String(), &keyboard)
}

// saveWorkoutFeeling сохраняет самочувствие и запрашивает RPE
func (b *Bot) saveWorkoutFeeling(chatID int64, feeling string, messageID int) {
	session := getWorkoutSession(chatID)
	if session == nil {
		return
	}

	// Сохраняем в сессию для итогового feedback
	// Запрашиваем RPE
	text := b.t("workout_post_rpe", chatID)

	// Кнопки RPE (1-10)
	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("1", "workout_rpe_1"),
		tgbotapi.NewInlineKeyboardButtonData("2", "workout_rpe_2"),
		tgbotapi.NewInlineKeyboardButtonData("3", "workout_rpe_3"),
		tgbotapi.NewInlineKeyboardButtonData("4", "workout_rpe_4"),
		tgbotapi.NewInlineKeyboardButtonData("5", "workout_rpe_5"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("6", "workout_rpe_6"),
		tgbotapi.NewInlineKeyboardButtonData("7", "workout_rpe_7"),
		tgbotapi.NewInlineKeyboardButtonData("8", "workout_rpe_8"),
		tgbotapi.NewInlineKeyboardButtonData("9", "workout_rpe_9"),
		tgbotapi.NewInlineKeyboardButtonData("10", "workout_rpe_10"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	// Сохраняем feeling в state для последующего использования
	setState(chatID, fmt.Sprintf("workout_feeling_%s", feeling))

	b.editMessage(chatID, messageID, text, &keyboard)
}

// saveWorkoutRPE сохраняет RPE и завершает тренировку
func (b *Bot) saveWorkoutRPE(chatID int64, rpe int, messageID int) {
	session := getWorkoutSession(chatID)
	if session == nil {
		return
	}

	// Получаем feeling из state
	userStates.RLock()
	state := userStates.states[chatID]
	userStates.RUnlock()

	feeling := ""
	if strings.HasPrefix(state, "workout_feeling_") {
		feeling = strings.TrimPrefix(state, "workout_feeling_")
	}

	// Формируем feedback
	feelingText := map[string]string{
		"great": "💪 Отлично",
		"good":  "👍 Хорошо",
		"tired": "😓 Устал",
		"bad":   "😞 Плохо",
	}

	feedback := fmt.Sprintf("RPE: %d/10\nСамочувствие: %s", rpe, feelingText[feeling])

	// Отмечаем тренировку как завершённую
	if err := b.repo.Program.MarkWorkoutCompleted(session.WorkoutID, feedback); err != nil {
		log.Printf("Ошибка завершения тренировки: %v", err)
	}

	// Очищаем сессию
	clearWorkoutSession(chatID)
	clearState(chatID)

	// Итоговое сообщение
	b.editMessage(chatID, messageID, b.t("workout_saved", chatID), nil)

	// Восстанавливаем главное меню
	b.restoreMainMenu(chatID)
}

// handleMyWorkouts показывает текущие тренировки клиента из программы
func (b *Bot) handleMyWorkouts(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Получаем ID клиента
	clientID, err := b.repo.Program.GetClientByTelegramID(chatID)
	if err != nil || clientID == 0 {
		b.sendMessage(chatID, b.t("reg_not_registered", chatID))
		return
	}

	// Получаем активную программу
	program, err := b.repo.Program.GetActiveProgram(clientID)
	if err != nil {
		b.sendError(chatID, b.t("error", chatID), err)
		return
	}

	if program == nil {
		b.sendMessage(chatID, b.t("workout_no_active_program", chatID))
		return
	}

	// Получаем следующую тренировку
	nextWorkout, err := b.repo.Program.GetNextPendingWorkout(clientID)
	if err != nil {
		b.sendError(chatID, b.t("error", chatID), err)
		return
	}

	if nextWorkout == nil {
		b.sendMessage(chatID, b.t("workout_no_pending", chatID))
		return
	}

	// Отправляем тренировку клиенту
	b.sendWorkoutToClient(chatID, nextWorkout)
}

// editMessage редактирует сообщение
func (b *Bot) editMessage(chatID int64, messageID int, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = "Markdown"
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка редактирования сообщения: %v", err)
	}
}

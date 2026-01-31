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

// showProgramProgress показывает прогресс по программе клиента (для тренера)
func (b *Bot) showProgramProgress(clientID int, adminChatID int64) {
	progress, err := b.repo.Program.GetProgramProgress(clientID)
	if err != nil {
		b.sendError(adminChatID, "Ошибка получения прогресса", err)
		return
	}

	if progress == nil {
		b.sendMessage(adminChatID, "У клиента нет активной программы")
		return
	}

	// Получаем имя клиента
	client, _ := b.repo.Client.GetByID(clientID)
	clientName := "Клиент"
	if client != nil {
		clientName = fmt.Sprintf("%s %s", client.Name, client.Surname)
	}

	// Формируем прогресс-бар
	progressBar := makeProgressBar(progress.ProgressPercent, 10)

	// Формируем текст
	var text strings.Builder
	text.WriteString(fmt.Sprintf("📊 *Прогресс программы*\n\n"))
	text.WriteString(fmt.Sprintf("👤 *Клиент:* %s\n", clientName))
	text.WriteString(fmt.Sprintf("📋 *Программа:* %s\n", progress.ProgramName))
	if progress.Goal != "" {
		text.WriteString(fmt.Sprintf("🎯 *Цель:* %s\n", progress.Goal))
	}
	text.WriteString("\n")
	text.WriteString(fmt.Sprintf("📅 *Неделя:* %d из %d\n", progress.CurrentWeek, progress.TotalWeeks))
	text.WriteString(fmt.Sprintf("🏋️ *Тренировок:* %d в неделю\n", progress.DaysPerWeek))
	text.WriteString("\n")
	text.WriteString(fmt.Sprintf("*Статистика:*\n"))
	text.WriteString(fmt.Sprintf("✅ Выполнено: %d\n", progress.CompletedCount))
	text.WriteString(fmt.Sprintf("📤 Отправлено: %d\n", progress.SentCount))
	text.WriteString(fmt.Sprintf("⏳ Ожидает: %d\n", progress.PendingCount))
	if progress.SkippedCount > 0 {
		text.WriteString(fmt.Sprintf("⏭️ Пропущено: %d\n", progress.SkippedCount))
	}
	text.WriteString("\n")
	text.WriteString(fmt.Sprintf("*Прогресс:* %.0f%%\n", progress.ProgressPercent))
	text.WriteString(progressBar)

	if progress.NextWorkout != nil {
		text.WriteString(fmt.Sprintf("\n\n📌 *Следующая:* %s (Нед.%d, День %d)",
			progress.NextWorkout.Name, progress.NextWorkout.WeekNum, progress.NextWorkout.DayNum))
	}

	// Inline-кнопки для действий
	var rows [][]tgbotapi.InlineKeyboardButton

	if progress.NextWorkout != nil {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"👁️ Превью тренировки",
				fmt.Sprintf("prog_preview_%d", progress.NextWorkout.ID),
			),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"📤 Отправить тренировку",
				fmt.Sprintf("prog_send_%d_%d", clientID, progress.NextWorkout.ID),
			),
		))
	}

	if progress.PendingCount > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("📦 Отправить неделю %d", progress.CurrentWeek),
				fmt.Sprintf("prog_send_week_%d_%d_%d", clientID, progress.ProgramID, progress.CurrentWeek),
			),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"🔔 Напомнить клиенту",
			fmt.Sprintf("prog_remind_%d", clientID),
		),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := tgbotapi.NewMessage(adminChatID, text.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки прогресса: %v", err)
	}
}

// handleProgramCallback обрабатывает callback-запросы связанные с программой (для тренера)
func (b *Bot) handleProgramCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// Отвечаем на callback
	callbackResponse := tgbotapi.NewCallback(callback.ID, "")
	b.api.Request(callbackResponse)

	switch {
	case strings.HasPrefix(data, "prog_preview_"):
		// Превью тренировки
		workoutIDStr := strings.TrimPrefix(data, "prog_preview_")
		workoutID, _ := strconv.Atoi(workoutIDStr)
		b.showWorkoutPreview(chatID, workoutID, callback.Message.MessageID)

	case strings.HasPrefix(data, "prog_send_week_"):
		// Отправить всю неделю
		parts := strings.Split(strings.TrimPrefix(data, "prog_send_week_"), "_")
		if len(parts) == 3 {
			clientID, _ := strconv.Atoi(parts[0])
			programID, _ := strconv.Atoi(parts[1])
			weekNum, _ := strconv.Atoi(parts[2])
			b.sendWeekWorkouts(chatID, clientID, programID, weekNum)
		}

	case strings.HasPrefix(data, "prog_send_"):
		// Отправить одну тренировку
		parts := strings.Split(strings.TrimPrefix(data, "prog_send_"), "_")
		if len(parts) == 2 {
			clientID, _ := strconv.Atoi(parts[0])
			workoutID, _ := strconv.Atoi(parts[1])
			b.sendSpecificWorkout(chatID, clientID, workoutID)
		}

	case strings.HasPrefix(data, "prog_remind_"):
		// Напомнить клиенту
		clientIDStr := strings.TrimPrefix(data, "prog_remind_")
		clientID, _ := strconv.Atoi(clientIDStr)
		b.sendWorkoutReminder(chatID, clientID)

	case strings.HasPrefix(data, "prog_back_"):
		// Вернуться к прогрессу
		clientIDStr := strings.TrimPrefix(data, "prog_back_")
		clientID, _ := strconv.Atoi(clientIDStr)
		b.showProgramProgress(clientID, chatID)
	}
}

// showWorkoutPreview показывает превью тренировки перед отправкой
func (b *Bot) showWorkoutPreview(chatID int64, workoutID int, messageID int) {
	workout, err := b.repo.Program.GetWorkoutByID(workoutID)
	if err != nil || workout == nil {
		b.sendMessage(chatID, "Ошибка: тренировка не найдена")
		return
	}

	// Получаем client_id для кнопки "назад"
	clientID, _ := b.repo.Program.GetClientIDByWorkout(workoutID)

	var text strings.Builder
	text.WriteString(fmt.Sprintf("👁️ *Превью тренировки*\n\n"))
	text.WriteString(fmt.Sprintf("📋 *%s*\n", workout.Name))
	text.WriteString(fmt.Sprintf("📅 Неделя %d, День %d\n", workout.WeekNum, workout.DayNum))
	text.WriteString(fmt.Sprintf("🏋️ Упражнений: %d\n\n", len(workout.Exercises)))

	for i, ex := range workout.Exercises {
		text.WriteString(fmt.Sprintf("*%d. %s*\n", i+1, ex.ExerciseName))
		text.WriteString(fmt.Sprintf("   %d×%s", ex.Sets, ex.Reps))
		if ex.Weight > 0 {
			text.WriteString(fmt.Sprintf(" @ %.0fкг", ex.Weight))
			if ex.WeightPercent > 0 {
				text.WriteString(fmt.Sprintf(" (%.0f%%)", ex.WeightPercent))
			}
		}
		if ex.RPE > 0 {
			text.WriteString(fmt.Sprintf(" RPE %.1f", ex.RPE))
		}
		text.WriteString("\n")
		if ex.RestSeconds > 0 {
			text.WriteString(fmt.Sprintf("   ⏱️ Отдых: %d сек\n", ex.RestSeconds))
		}
		if ex.Notes != "" {
			text.WriteString(fmt.Sprintf("   📝 %s\n", ex.Notes))
		}
	}

	// Кнопки
	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"📤 Отправить клиенту",
			fmt.Sprintf("prog_send_%d_%d", clientID, workoutID),
		),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"◀️ Назад к прогрессу",
			fmt.Sprintf("prog_back_%d", clientID),
		),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	if messageID > 0 {
		b.editMessage(chatID, messageID, text.String(), &keyboard)
	} else {
		msg := tgbotapi.NewMessage(chatID, text.String())
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}

// sendWeekWorkouts отправляет все тренировки недели клиенту
func (b *Bot) sendWeekWorkouts(adminChatID int64, clientID int, programID int, weekNum int) {
	// Получаем telegram_id клиента
	var telegramID int64
	err := b.db.QueryRow("SELECT telegram_id FROM public.clients WHERE id = $1", clientID).Scan(&telegramID)
	if err != nil || telegramID == 0 {
		b.sendMessage(adminChatID, "Ошибка: клиент не найден или не имеет telegram_id")
		return
	}

	// Получаем неотправленные тренировки недели
	workouts, err := b.repo.Program.GetPendingWorkoutsForWeek(programID, weekNum)
	if err != nil {
		b.sendError(adminChatID, "Ошибка получения тренировок", err)
		return
	}

	if len(workouts) == 0 {
		b.sendMessage(adminChatID, fmt.Sprintf("Нет неотправленных тренировок на неделе %d", weekNum))
		return
	}

	// Отправляем каждую тренировку
	sentCount := 0
	for i := range workouts {
		workout := &workouts[i]
		b.sendWorkoutToClient(telegramID, workout)
		if err := b.repo.Program.MarkWorkoutSent(workout.ID); err != nil {
			log.Printf("Ошибка отметки тренировки %d: %v", workout.ID, err)
		}
		sentCount++
		// Небольшая пауза между сообщениями
		time.Sleep(500 * time.Millisecond)
	}

	b.sendMessage(adminChatID, fmt.Sprintf("✅ Отправлено %d тренировок (Неделя %d)", sentCount, weekNum))

	// Показываем обновлённый прогресс
	b.showProgramProgress(clientID, adminChatID)
}

// sendSpecificWorkout отправляет конкретную тренировку клиенту
func (b *Bot) sendSpecificWorkout(adminChatID int64, clientID int, workoutID int) {
	// Получаем telegram_id клиента
	var telegramID int64
	err := b.db.QueryRow("SELECT telegram_id FROM public.clients WHERE id = $1", clientID).Scan(&telegramID)
	if err != nil || telegramID == 0 {
		b.sendMessage(adminChatID, "Ошибка: клиент не найден или не имеет telegram_id")
		return
	}

	// Получаем тренировку
	workout, err := b.repo.Program.GetWorkoutByID(workoutID)
	if err != nil || workout == nil {
		b.sendError(adminChatID, "Ошибка получения тренировки", err)
		return
	}

	// Отправляем клиенту
	b.sendWorkoutToClient(telegramID, workout)

	// Отмечаем как отправленную
	if err := b.repo.Program.MarkWorkoutSent(workout.ID); err != nil {
		log.Printf("Ошибка отметки тренировки как отправленной: %v", err)
	}

	b.sendMessage(adminChatID, fmt.Sprintf("✅ Тренировка \"%s\" отправлена клиенту", workout.Name))

	// Показываем обновлённый прогресс
	b.showProgramProgress(clientID, adminChatID)
}

// sendWorkoutReminder отправляет напоминание клиенту о тренировке
func (b *Bot) sendWorkoutReminder(adminChatID int64, clientID int) {
	// Получаем telegram_id клиента
	var telegramID int64
	var name string
	err := b.db.QueryRow("SELECT telegram_id, name FROM public.clients WHERE id = $1", clientID).Scan(&telegramID, &name)
	if err != nil || telegramID == 0 {
		b.sendMessage(adminChatID, "Ошибка: клиент не найден или не имеет telegram_id")
		return
	}

	// Получаем прогресс
	progress, err := b.repo.Program.GetProgramProgress(clientID)
	if err != nil || progress == nil {
		b.sendMessage(adminChatID, "Ошибка получения прогресса программы")
		return
	}

	// Формируем напоминание
	var text strings.Builder
	text.WriteString(fmt.Sprintf("👋 Привет, %s!\n\n", name))

	if progress.SentCount > 0 && progress.NextWorkout != nil && progress.NextWorkout.Status == "sent" {
		// Есть отправленная, но не выполненная тренировка
		text.WriteString("🏋️ У тебя есть запланированная тренировка!\n\n")
		text.WriteString(fmt.Sprintf("📋 *%s*\n", progress.NextWorkout.Name))
		text.WriteString(fmt.Sprintf("Неделя %d, День %d\n\n", progress.NextWorkout.WeekNum, progress.NextWorkout.DayNum))
		text.WriteString("Напиши /workouts чтобы начать 💪")
	} else if progress.PendingCount > 0 {
		// Есть неотправленные тренировки
		text.WriteString("📅 Готов к следующей тренировке?\n\n")
		text.WriteString(fmt.Sprintf("Прогресс программы: %.0f%% (%d/%d)\n\n",
			progress.ProgressPercent, progress.CompletedCount, progress.TotalWorkouts))
		text.WriteString("Напиши /workouts чтобы получить тренировку 💪")
	} else {
		text.WriteString("🎉 Все тренировки программы выполнены!\n")
		text.WriteString("Отличная работа! 🏆")
	}

	msg := tgbotapi.NewMessage(telegramID, text.String())
	msg.ParseMode = "Markdown"

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки напоминания: %v", err)
		b.sendMessage(adminChatID, "❌ Ошибка отправки напоминания")
		return
	}

	b.sendMessage(adminChatID, fmt.Sprintf("✅ Напоминание отправлено клиенту %s", name))
}

// makeProgressBar создаёт текстовый прогресс-бар
func makeProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("▓", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s]", bar)
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

	// Отправляем уведомление тренеру
	duration := int(time.Since(session.StartTime).Minutes())
	b.notifyTrainerWorkoutCompleted(session.WorkoutID, chatID, duration, rpe, feeling)

	// Очищаем сессию
	clearWorkoutSession(chatID)
	clearState(chatID)

	// Итоговое сообщение
	b.editMessage(chatID, messageID, b.t("workout_saved", chatID), nil)

	// Восстанавливаем главное меню
	b.restoreMainMenu(chatID)
}

// notifyTrainerWorkoutCompleted отправляет тренеру уведомление о завершении тренировки
func (b *Bot) notifyTrainerWorkoutCompleted(workoutID int, clientChatID int64, duration, rpe int, feeling string) {
	// Получаем тренера (первого админа)
	trainerID, err := b.repo.Admin.GetFirst()
	if err != nil {
		log.Printf("Ошибка получения тренера: %v", err)
		return
	}

	// Получаем данные тренировки
	workout, err := b.repo.Program.GetWorkoutByID(workoutID)
	if err != nil || workout == nil {
		log.Printf("Ошибка получения тренировки: %v", err)
		return
	}

	// Получаем статистику (тоннаж, выполнение)
	stats, err := b.repo.Program.GetWorkoutStats(workoutID)
	if err != nil {
		log.Printf("Ошибка получения статистики: %v", err)
		return
	}

	// Получаем имя клиента
	clientID, _ := b.repo.Program.GetClientIDByWorkout(workoutID)
	client, _ := b.repo.Client.GetByID(clientID)
	clientName := "Клиент"
	if client != nil {
		clientName = fmt.Sprintf("%s %s", client.Name, client.Surname)
	}

	// Формируем сообщение
	feelingEmoji := map[string]string{
		"great": "💪",
		"good":  "👍",
		"tired": "😓",
		"bad":   "😞",
	}

	// Индикатор выполнения плана
	complianceIndicator := "✅"
	if stats.ComplianceRate < 100 {
		complianceIndicator = "⚠️"
	}
	if stats.ComplianceRate < 70 {
		complianceIndicator = "❌"
	}

	// Форматируем тоннаж
	tonnageStr := formatTonnage(stats.Tonnage)

	text := fmt.Sprintf(`🏋️ *Тренировка завершена!*

👤 *Клиент:* %s
📋 *Тренировка:* %s (Неделя %d, День %d)

📊 *Статистика:*
• Тоннаж: *%s*
• Выполнено: %d/%d упражнений %s
• Выполнение плана: %.0f%%
• Длительность: %d мин

💭 *Обратная связь:*
• RPE: %d/10
• Самочувствие: %s`,
		clientName,
		workout.Name, workout.WeekNum, workout.DayNum,
		tonnageStr,
		stats.Completed, stats.TotalExercises, complianceIndicator,
		stats.ComplianceRate,
		duration,
		rpe,
		feelingEmoji[feeling],
	)

	msg := tgbotapi.NewMessage(trainerID, text)
	msg.ParseMode = "Markdown"

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки уведомления тренеру: %v", err)
	}
}

// formatTonnage форматирует тоннаж в читаемый вид
func formatTonnage(tonnage float64) string {
	if tonnage >= 1000 {
		return fmt.Sprintf("%.1f т", tonnage/1000)
	}
	return fmt.Sprintf("%.0f кг", tonnage)
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

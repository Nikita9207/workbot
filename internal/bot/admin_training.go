package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"workbot/internal/excel"
	"workbot/internal/gsheets"
	"workbot/internal/models"
	"workbot/internal/training"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// getTrainingNumber возвращает номер тренировки для клиента
func (b *Bot) getTrainingNumber(clientID int) int {
	var count int
	if err := b.db.QueryRow("SELECT COUNT(*) FROM trainings WHERE client_id = $1", clientID).Scan(&count); err != nil {
		log.Printf("Ошибка подсчёта тренировок клиента %d: %v", clientID, err)
		return 1
	}
	return count + 1
}

// showClientsForSending показывает список клиентов для отправки тренировки
func (b *Bot) showClientsForSending(message *tgbotapi.Message) {
	rows, err := b.db.Query(`
		SELECT c.id, c.name, c.surname, COALESCE(c.telegram_id, 0)
		FROM public.clients c
		LEFT JOIN public.admins a ON c.telegram_id = a.telegram_id
		WHERE a.telegram_id IS NULL AND c.deleted_at IS NULL
		ORDER BY c.name`)
	if err != nil {
		b.sendError(message.Chat.ID, "Ошибка загрузки клиентов", err)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id int
		var name, surname string
		var telegramID int64
		if err := rows.Scan(&id, &name, &surname, &telegramID); err != nil {
			log.Printf("Ошибка сканирования клиента: %v", err)
			continue
		}

		// Проверяем тренировки из программы (training_programs)
		pendingWorkouts, _ := b.repo.Program.GetPendingWorkoutsForClient(id)
		programWorkoutsCount := len(pendingWorkouts)

		// Проверяем тренировки из Excel
		unsentGroups, _ := excel.GetUnsentTrainings(excel.FilePath, b.db, id)
		excelWorkoutsCount := len(unsentGroups)

		totalCount := programWorkoutsCount + excelWorkoutsCount
		if totalCount == 0 {
			continue
		}

		status := "нет telegram"
		if telegramID > 0 {
			if programWorkoutsCount > 0 && excelWorkoutsCount > 0 {
				status = fmt.Sprintf("%d из прог. + %d из Excel", programWorkoutsCount, excelWorkoutsCount)
			} else if programWorkoutsCount > 0 {
				status = fmt.Sprintf("%d из программы", programWorkoutsCount)
			} else {
				status = fmt.Sprintf("%d из Excel", excelWorkoutsCount)
			}
		}

		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("Отправить: %s %s [%d] (%s)", name, surname, id, status)),
		))
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка итерации по клиентам: %v", err)
	}

	if len(buttons) == 0 {
		b.sendMessage(message.Chat.ID, "Нет неотправленных тренировок")
		b.handleAdminStart(message)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Назад"),
	))

	keyboard := tgbotapi.NewReplyKeyboard(buttons...)
	b.sendMessageWithKeyboard(message.Chat.ID, "Выберите клиента для отправки тренировки:", keyboard)
}

// handleSendTrainingSelection обрабатывает отправку тренировки клиенту
func (b *Bot) handleSendTrainingSelection(message *tgbotapi.Message, text string) {
	chatID := message.Chat.ID

	clientID := parseIDFromBrackets(text)
	if clientID == 0 {
		b.sendMessage(chatID, "Ошибка выбора клиента")
		return
	}

	var telegramID int64
	var name, surname string
	err := b.db.QueryRow("SELECT telegram_id, name, surname FROM public.clients WHERE id = $1", clientID).
		Scan(&telegramID, &name, &surname)
	if err != nil {
		b.sendError(chatID, "Клиент не найден", err)
		return
	}

	if telegramID == 0 {
		b.sendMessage(chatID, fmt.Sprintf("У клиента %s %s нет telegram. Тренировка не может быть отправлена.", name, surname))
		b.handleAdminStart(message)
		return
	}

	sentCount := 0

	// 1. Сначала отправляем тренировки из программы (training_programs)
	nextWorkout, err := b.repo.Program.GetNextPendingWorkout(clientID)
	if err == nil && nextWorkout != nil {
		// Отправляем тренировку с inline-кнопками
		b.sendWorkoutToClient(telegramID, nextWorkout)

		// Отмечаем как отправленную
		if err := b.repo.Program.MarkWorkoutSent(nextWorkout.ID); err != nil {
			log.Printf("Ошибка отметки тренировки: %v", err)
		}
		sentCount++
	}

	// 2. Затем отправляем из Excel (для обратной совместимости)
	unsentGroups, err := excel.GetUnsentTrainings(excel.FilePath, b.db, clientID)
	if err == nil && len(unsentGroups) > 0 {
		for _, group := range unsentGroups {
			msgText := excel.FormatTrainingMessage(&group)
			notification := tgbotapi.NewMessage(telegramID, msgText)
			if _, err := b.api.Send(notification); err != nil {
				log.Printf("Ошибка отправки клиенту %d: %v", clientID, err)
				continue
			}

			if err := excel.MarkTrainingGroupAsSent(excel.FilePath, &group); err != nil {
				log.Printf("Ошибка пометки тренировки: %v", err)
			}
			sentCount++
		}
	}

	if sentCount == 0 {
		b.sendMessage(chatID, "Нет неотправленных тренировок для этого клиента")
	} else {
		b.sendMessage(chatID, fmt.Sprintf("✅ Отправлено %d тренировок клиенту %s %s", sentCount, name, surname))
	}
	b.handleAdminStart(message)
}

// startTrainingInput начинает ввод тренировки
func (b *Bot) startTrainingInput(chatID int64, clientID int) {
	var name, surname string
	if err := b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).
		Scan(&name, &surname); err != nil {
		log.Printf("Ошибка получения данных клиента %d: %v", clientID, err)
		b.sendMessage(chatID, "Ошибка получения данных клиента")
		return
	}

	setState(chatID, "admin_awaiting_training")

	helpText := fmt.Sprintf("Клиент: %s %s\n\n"+
		"Введите тренировку в формате:\n"+
		"Упражнение подходы/повторы вес\n\n"+
		"Пример:\n"+
		"Жим лежа 4/8 60\n"+
		"Присед 4/6 80\n\n"+
		"Можно указать дату первой строкой (ДД.ММ.ГГГГ)",
		name, surname)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	b.sendMessageWithKeyboard(chatID, helpText, keyboard)
}

// handleAdminTrainingInput обрабатывает ввод тренировки от админа
func (b *Bot) handleAdminTrainingInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	// Кнопка "Назад" - возврат к профилю клиента
	if text == "Назад" {
		adminStates.RLock()
		clientID := adminStates.selectedClient[chatID]
		adminStates.RUnlock()

		setState(chatID, "viewing_client")

		if clientID > 0 {
			b.showClientProfile(chatID, clientID)
		} else {
			b.handleAdminStart(message)
		}
		return
	}

	adminStates.Lock()
	clientID := adminStates.selectedClient[chatID]
	delete(adminStates.selectedClient, chatID)
	adminStates.Unlock()

	clearState(chatID)

	if clientID == 0 {
		b.sendMessage(chatID, "Ошибка: клиент не выбран")
		b.handleAdminStart(message)
		return
	}

	exercises, trainingDate, err := training.Parse(text)
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("Ошибка разбора: %v", err))
		b.handleAdminStart(message)
		return
	}

	if len(exercises) == 0 {
		b.sendMessage(chatID, "Не найдено упражнений. Проверьте формат.")
		b.handleAdminStart(message)
		return
	}

	var name, surname string
	var telegramID int64
	err = b.db.QueryRow("SELECT name, surname, COALESCE(telegram_id, 0) FROM public.clients WHERE id = $1", clientID).
		Scan(&name, &surname, &telegramID)
	if err != nil {
		b.sendError(chatID, "Ошибка: клиент не найден", err)
		b.handleAdminStart(message)
		return
	}

	err = excel.SaveTrainingToExcel(excel.FilePath, b.db, clientID, name, surname, trainingDate, exercises)
	if err != nil {
		b.sendError(chatID, fmt.Sprintf("Ошибка сохранения: %v", err), err)
		b.handleAdminStart(message)
		return
	}

	if err := excel.UpdateAllDashboards(excel.FilePath, b.db); err != nil {
		log.Printf("Ошибка обновления dashboard: %v", err)
	}

	// Сохраняем в Google Sheets
	if b.sheetsClient != nil {
		var sheetID string
		if err := b.db.QueryRow("SELECT google_sheet_id FROM clients WHERE id = $1", clientID).Scan(&sheetID); err != nil {
			log.Printf("Ошибка получения google_sheet_id: %v", err)
		}
		if sheetID != "" {
			trainingNum := b.getTrainingNumber(clientID)
			gsExercises := make([]gsheets.TrainingExercise, len(exercises))
			for i, ex := range exercises {
				gsExercises[i] = gsheets.TrainingExercise{
					Name:   ex.Name,
					Sets:   ex.Sets,
					Reps:   ex.Reps,
					Weight: ex.Weight,
				}
			}
			if err := b.sheetsClient.AddTraining(sheetID, trainingDate, trainingNum, gsExercises); err != nil {
				log.Printf("Ошибка записи в Google Sheets: %v", err)
			}
		}
	}

	confirmText := fmt.Sprintf("Тренировка для %s %s сохранена!\n\n", name, surname)
	confirmText += training.FormatConfirmation(exercises, trainingDate)

	b.sendMessage(chatID, confirmText)

	if telegramID > 0 {
		clientMsg := formatTrainingNotification(exercises, trainingDate)
		notification := tgbotapi.NewMessage(telegramID, clientMsg)
		if _, err := b.api.Send(notification); err != nil {
			log.Printf("Ошибка отправки клиенту %d: %v", clientID, err)
		} else {
			b.sendMessage(chatID, "Уведомление отправлено клиенту")
		}
	}

	b.handleAdminStart(message)
}

// startSetGoal начинает установку цели
func (b *Bot) startSetGoal(chatID int64, clientID int) {
	setState(chatID, "setting_goal")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Набор массы"),
			tgbotapi.NewKeyboardButton("Похудение"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сила"),
			tgbotapi.NewKeyboardButton("Выносливость"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	b.sendMessageWithKeyboard(chatID, "Введите цель клиента:\n\n"+
		"Например:\n"+
		"- Набор мышечной массы 5кг за 3 месяца\n"+
		"- Похудение на 10кг\n"+
		"- Подготовка к соревнованиям\n"+
		"- Общая физическая подготовка", keyboard)
}

// handleSetGoal сохраняет цель клиента
func (b *Bot) handleSetGoal(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		setState(chatID, "viewing_client")
		adminStates.RLock()
		clientID := adminStates.selectedClient[chatID]
		adminStates.RUnlock()
		b.showClientProfile(chatID, clientID)
		return
	}

	adminStates.RLock()
	clientID := adminStates.selectedClient[chatID]
	adminStates.RUnlock()

	_, err := b.db.Exec("UPDATE public.clients SET goal = $1 WHERE id = $2", text, clientID)
	if err != nil {
		b.sendError(chatID, "Ошибка сохранения цели", err)
	} else {
		if _, err := b.db.Exec(`INSERT INTO public.client_goals (client_id, goal) VALUES ($1, $2)`, clientID, text); err != nil {
			log.Printf("Ошибка записи истории целей: %v", err)
		}
		b.sendMessage(chatID, fmt.Sprintf("Цель установлена: %s", text))
	}

	setState(chatID, "viewing_client")
	b.showClientProfile(chatID, clientID)
}

// startCreatePlan начинает создание плана
func (b *Bot) startCreatePlan(chatID int64, clientID int) {
	setState(chatID, "creating_plan")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сгенерировать AI"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	b.sendMessageWithKeyboard(chatID, "Введите план тренировок:\n\n"+
		"Можете описать:\n"+
		"- Периодизацию (подготовительный, соревновательный период)\n"+
		"- Частоту тренировок в неделю\n"+
		"- Основные упражнения и прогрессию\n"+
		"- Любые заметки по плану\n\n"+
		"Или используйте AI для автоматической генерации.", keyboard)
}

// handleCreatePlan сохраняет план
func (b *Bot) handleCreatePlan(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		setState(chatID, "viewing_client")
		adminStates.RLock()
		clientID := adminStates.selectedClient[chatID]
		adminStates.RUnlock()
		b.showClientProfile(chatID, clientID)
		return
	}

	if text == "Сгенерировать AI" {
		adminStates.RLock()
		clientID := adminStates.selectedClient[chatID]
		adminStates.RUnlock()
		b.handleAIClientPlan(chatID, clientID)
		return
	}

	adminStates.RLock()
	clientID := adminStates.selectedClient[chatID]
	adminStates.RUnlock()

	_, err := b.db.Exec("UPDATE public.clients SET training_plan = $1 WHERE id = $2", text, clientID)
	if err != nil {
		b.sendError(chatID, "Ошибка сохранения плана", err)
	} else {
		b.sendMessage(chatID, "План сохранён!")
	}

	setState(chatID, "viewing_client")
	b.showClientProfile(chatID, clientID)
}

// handleAIClientPlan генерирует план через AI
func (b *Bot) handleAIClientPlan(chatID int64, clientID int) {
	var name, surname string
	var goal sql.NullString
	if err := b.db.QueryRow("SELECT name, surname, goal FROM public.clients WHERE id = $1", clientID).
		Scan(&name, &surname, &goal); err != nil {
		b.sendError(chatID, "Ошибка получения данных клиента", err)
		return
	}

	if !goal.Valid || goal.String == "" {
		b.sendMessage(chatID, "Сначала задайте цель клиента!")
		b.startSetGoal(chatID, clientID)
		return
	}

	aiClientStore.Lock()
	aiClientStore.data[chatID] = clientID
	aiClientStore.Unlock()

	setState(chatID, "ai_awaiting_params")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Силовая"),
			tgbotapi.NewKeyboardButton("Кардио"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Смешанная"),
			tgbotapi.NewKeyboardButton("Функциональная"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	b.sendMessageWithKeyboard(chatID, fmt.Sprintf(
		"Генерация плана для %s %s\n"+
			"Цель: %s\n\n"+
			"Выберите тип тренировки:",
		name, surname, goal.String), keyboard)
}

// showClientHistory показывает историю тренировок клиента
func (b *Bot) showClientHistory(chatID int64, clientID int) {
	trainings, err := excel.GetClientTrainings(excel.FilePath, clientID, 10)
	if err != nil {
		b.sendError(chatID, "Ошибка загрузки истории", err)
		return
	}

	if len(trainings) == 0 {
		b.sendMessage(chatID, "У клиента пока нет записанных тренировок.")
		return
	}

	var result strings.Builder
	result.WriteString("История тренировок:\n\n")
	for _, t := range trainings {
		result.WriteString(t)
		result.WriteString("\n")
	}

	b.sendMessage(chatID, result.String())
}

// formatTrainingNotification форматирует уведомление о тренировке для клиента
func formatTrainingNotification(exercises []models.ExerciseInput, trainingDate time.Time) string {
	var totalTonnage float64
	var exerciseList strings.Builder

	exerciseList.WriteString(fmt.Sprintf("Тренировка на %s\n\n", trainingDate.Format("02.01.2006")))

	for i, ex := range exercises {
		tonnage := float64(ex.Sets) * float64(ex.Reps) * ex.Weight
		totalTonnage += tonnage
		exerciseList.WriteString(fmt.Sprintf("%d. %s\n   %d/%d", i+1, ex.Name, ex.Sets, ex.Reps))
		if ex.Weight > 0 {
			exerciseList.WriteString(fmt.Sprintf(" %.0fкг", ex.Weight))
		}
		exerciseList.WriteString("\n")
	}

	if totalTonnage > 0 {
		exerciseList.WriteString(fmt.Sprintf("\nОбщий тоннаж: %.0f кг", totalTonnage))
	}

	exerciseList.WriteString("\n\nУдачной тренировки!")

	return exerciseList.String()
}

// handleAIState обрабатывает состояния AI плана
func (b *Bot) handleAIState(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		clearState(chatID)
		aiClientStore.Lock()
		delete(aiClientStore.data, chatID)
		aiClientStore.Unlock()
		// Возвращаемся к профилю клиента
		adminStates.RLock()
		clientID := adminStates.selectedClient[chatID]
		adminStates.RUnlock()
		if clientID > 0 {
			b.showClientProfile(chatID, clientID)
		} else {
			b.handleAdminStart(message)
		}
		return
	}

	switch state {
	case "ai_awaiting_params":
		b.handleAIParamsSelect(message)
	}
}

// handleAIParamsSelect обрабатывает выбор типа тренировки для AI
func (b *Bot) handleAIParamsSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	aiClientStore.RLock()
	clientID := aiClientStore.data[chatID]
	aiClientStore.RUnlock()

	if clientID == 0 {
		b.sendMessage(chatID, "Ошибка: клиент не выбран")
		b.handleAdminStart(message)
		return
	}

	// Получаем данные клиента
	var name, surname string
	var goal sql.NullString
	if err := b.db.QueryRow("SELECT name, surname, goal FROM public.clients WHERE id = $1", clientID).
		Scan(&name, &surname, &goal); err != nil {
		b.sendError(chatID, "Ошибка получения данных клиента", err)
		return
	}

	var trainingType string
	switch text {
	case "Силовая":
		trainingType = "силовая"
	case "Кардио":
		trainingType = "кардио"
	case "Смешанная":
		trainingType = "смешанная"
	case "Функциональная":
		trainingType = "функциональная"
	default:
		b.sendMessage(chatID, "Выберите тип тренировки из меню")
		return
	}

	// Генерируем простой план на основе типа
	plan := b.generateSimplePlan(name, surname, goal.String, trainingType)

	// Сохраняем в БД
	_, err := b.db.Exec("UPDATE public.clients SET training_plan = $1 WHERE id = $2", plan, clientID)
	if err != nil {
		b.sendError(chatID, "Ошибка сохранения плана", err)
		return
	}

	b.sendMessage(chatID, fmt.Sprintf("✅ План сгенерирован!\n\n%s", plan))

	// Очищаем состояние
	clearState(chatID)
	aiClientStore.Lock()
	delete(aiClientStore.data, chatID)
	aiClientStore.Unlock()

	b.showClientProfile(chatID, clientID)
}

// generateSimplePlan генерирует простой план тренировок
func (b *Bot) generateSimplePlan(name, surname, goal, trainingType string) string {
	var plan strings.Builder

	plan.WriteString(fmt.Sprintf("План для %s %s\n", name, surname))
	plan.WriteString(fmt.Sprintf("Цель: %s\n", goal))
	plan.WriteString(fmt.Sprintf("Тип: %s тренировка\n\n", trainingType))

	switch trainingType {
	case "силовая":
		plan.WriteString("День 1 (Верх):\n")
		plan.WriteString("1. Жим штанги лёжа 4x8-10\n")
		plan.WriteString("2. Тяга штанги в наклоне 4x8-10\n")
		plan.WriteString("3. Жим гантелей сидя 3x10-12\n")
		plan.WriteString("4. Подтягивания 3xmax\n")
		plan.WriteString("5. Разгибания на трицепс 3x12-15\n\n")
		plan.WriteString("День 2 (Низ):\n")
		plan.WriteString("1. Приседания со штангой 4x8-10\n")
		plan.WriteString("2. Румынская тяга 4x8-10\n")
		plan.WriteString("3. Жим ногами 3x10-12\n")
		plan.WriteString("4. Сгибания ног 3x12-15\n")
		plan.WriteString("5. Подъём на носки 4x15-20\n")

	case "кардио":
		plan.WriteString("День 1:\n")
		plan.WriteString("1. Разминка 5 мин\n")
		plan.WriteString("2. Интервальный бег 20 мин (30с быстро/60с медленно)\n")
		plan.WriteString("3. Велотренажёр 15 мин\n")
		plan.WriteString("4. Заминка 5 мин\n\n")
		plan.WriteString("День 2:\n")
		plan.WriteString("1. Разминка 5 мин\n")
		plan.WriteString("2. Эллипс 25 мин\n")
		plan.WriteString("3. Скакалка 3x3 мин\n")
		plan.WriteString("4. Заминка 5 мин\n")

	case "смешанная":
		plan.WriteString("День 1:\n")
		plan.WriteString("1. Приседания 4x10\n")
		plan.WriteString("2. Жим гантелей лёжа 3x12\n")
		plan.WriteString("3. Тяга верхнего блока 3x12\n")
		plan.WriteString("4. Интервальное кардио 15 мин\n\n")
		plan.WriteString("День 2:\n")
		plan.WriteString("1. Становая тяга 4x8\n")
		plan.WriteString("2. Жим плечами 3x12\n")
		plan.WriteString("3. Тяга нижнего блока 3x12\n")
		plan.WriteString("4. HIIT 15 мин\n")

	case "функциональная":
		plan.WriteString("День 1:\n")
		plan.WriteString("1. Берпи 3x10\n")
		plan.WriteString("2. Махи гирей 4x15\n")
		plan.WriteString("3. Выпады с гантелями 3x12 на ногу\n")
		plan.WriteString("4. Планка 3x45 сек\n")
		plan.WriteString("5. Box jump 3x10\n\n")
		plan.WriteString("День 2:\n")
		plan.WriteString("1. Турецкий подъём 3x5 на сторону\n")
		plan.WriteString("2. Рывок гири 4x10\n")
		plan.WriteString("3. Приседания с гирей 3x12\n")
		plan.WriteString("4. Farmer's walk 3x30м\n")
		plan.WriteString("5. Скручивания 3x20\n")
	}

	return plan.String()
}

package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"workbot/internal/models"
	"workbot/internal/training"
)

// States for 1PM flow
const (
	state1PMSelectClient   = "1pm_select_client"
	state1PMSelectExercise = "1pm_select_exercise"
	state1PMInputMethod    = "1pm_input_method"
	state1PMManualInput    = "1pm_manual_input"
	state1PMCalcWeight     = "1pm_calc_weight"
	state1PMCalcReps       = "1pm_calc_reps"
	state1PMAddExercise    = "1pm_add_exercise"
	state1PMConfirm        = "1pm_confirm"
)

// onePMStore stores temporary data for 1PM recording
var onePMStore = struct {
	sync.RWMutex
	clientID    map[int64]int
	exerciseID  map[int64]int
	calcWeight  map[int64]float64
	calcReps    map[int64]int
	calcMethod  map[int64]string
	calculated1PM map[int64]float64
}{
	clientID:      make(map[int64]int),
	exerciseID:    make(map[int64]int),
	calcWeight:    make(map[int64]float64),
	calcReps:      make(map[int64]int),
	calcMethod:    make(map[int64]string),
	calculated1PM: make(map[int64]float64),
}

// handle1PMMenu shows the 1PM testing menu
func (b *Bot) handle1PMMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = state1PMSelectClient
	userStates.Unlock()

	b.showClientsFor1PM(chatID, "Выберите клиента для записи 1ПМ:")
}

// showClientsFor1PM shows client list for 1PM recording
func (b *Bot) showClientsFor1PM(chatID int64, text string) {
	rows, err := b.db.Query(`
		SELECT id, name, surname
		FROM public.clients
		WHERE deleted_at IS NULL
		ORDER BY name, surname`)
	if err != nil {
		log.Printf("Ошибка получения клиентов: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки списка клиентов")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id int
		var name, surname string
		if err := rows.Scan(&id, &name, &surname); err != nil {
			continue
		}
		buttonText := fmt.Sprintf("1PM>> %s %s [%d]", name, surname, id)
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет клиентов. Сначала добавьте клиента через меню клиентов.")
		b.api.Send(msg)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// handle1PMClientSelect handles client selection for 1PM
func (b *Bot) handle1PMClientSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	// Parse client ID from "1PM>> Name Surname [ID]"
	clientID := parse1PMClientID(text)
	if clientID == 0 {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора клиента")
		b.api.Send(msg)
		return
	}

	onePMStore.Lock()
	onePMStore.clientID[chatID] = clientID
	onePMStore.Unlock()

	userStates.Lock()
	userStates.states[chatID] = state1PMSelectExercise
	userStates.Unlock()

	b.show1PMExerciseList(chatID, clientID)
}

// show1PMExerciseList shows exercise selection for 1PM
func (b *Bot) show1PMExerciseList(chatID int64, clientID int) {
	// Get exercises with existing 1PM for this client
	rows, err := b.db.Query(`
		SELECT e.id, e.name, e.muscle_group,
			COALESCE((SELECT one_pm_kg FROM public.exercise_1pm
				WHERE client_id = $1 AND exercise_id = e.id
				ORDER BY test_date DESC LIMIT 1), 0) as current_1pm
		FROM public.exercises e
		ORDER BY
			CASE WHEN e.movement_type = 'compound' THEN 0 ELSE 1 END,
			e.muscle_group, e.name
	`, clientID)
	if err != nil {
		log.Printf("Ошибка получения упражнений: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки упражнений")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id int
		var name, muscleGroup string
		var current1PM float64
		if err := rows.Scan(&id, &name, &muscleGroup, &current1PM); err != nil {
			continue
		}

		buttonText := fmt.Sprintf("EX>> %s", name)
		if current1PM > 0 {
			buttonText += fmt.Sprintf(" (%.1f кг)", current1PM)
		}
		buttonText += fmt.Sprintf(" [%d]", id)

		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		))
	}

	// Add option to create new exercise
	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("➕ Добавить упражнение"),
	))
	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите упражнение для записи 1ПМ:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// handle1PMExerciseSelect handles exercise selection
func (b *Bot) handle1PMExerciseSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clear1PMState(chatID)
		b.handleAdminCancel(message)
		return
	}

	if text == "➕ Добавить упражнение" {
		userStates.Lock()
		userStates.states[chatID] = state1PMAddExercise
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите название нового упражнения:")
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
		return
	}

	// Parse exercise ID from "EX>> Name (1PM) [ID]"
	exerciseID := parse1PMExerciseID(text)
	if exerciseID == 0 {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора упражнения")
		b.api.Send(msg)
		return
	}

	onePMStore.Lock()
	onePMStore.exerciseID[chatID] = exerciseID
	onePMStore.Unlock()

	b.show1PMInputMethod(chatID)
}

// show1PMInputMethod shows method selection (manual or calculated)
func (b *Bot) show1PMInputMethod(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = state1PMInputMethod
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Как записать 1ПМ?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Ввести 1ПМ вручную"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Рассчитать по подходу (Бжицки)"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Рассчитать по подходу (Эпли)"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Рассчитать (среднее)"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handle1PMInputMethod handles method selection
func (b *Bot) handle1PMInputMethod(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clear1PMState(chatID)
		b.handleAdminCancel(message)
		return
	}

	switch text {
	case "Ввести 1ПМ вручную":
		userStates.Lock()
		userStates.states[chatID] = state1PMManualInput
		userStates.Unlock()

		onePMStore.Lock()
		onePMStore.calcMethod[chatID] = "manual"
		onePMStore.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите 1ПМ в килограммах (например: 100 или 102.5):")
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)

	case "Рассчитать по подходу (Бжицки)":
		onePMStore.Lock()
		onePMStore.calcMethod[chatID] = "brzycki"
		onePMStore.Unlock()
		b.ask1PMCalcWeight(chatID)

	case "Рассчитать по подходу (Эпли)":
		onePMStore.Lock()
		onePMStore.calcMethod[chatID] = "epley"
		onePMStore.Unlock()
		b.ask1PMCalcWeight(chatID)

	case "Рассчитать (среднее)":
		onePMStore.Lock()
		onePMStore.calcMethod[chatID] = "average"
		onePMStore.Unlock()
		b.ask1PMCalcWeight(chatID)

	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите вариант из меню")
		b.api.Send(msg)
	}
}

// ask1PMCalcWeight asks for weight used in set
func (b *Bot) ask1PMCalcWeight(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = state1PMCalcWeight
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Введите вес, с которым был выполнен подход (кг):\n\nНапример: 80 или 82.5")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handle1PMCalcWeight handles weight input for calculation
func (b *Bot) handle1PMCalcWeight(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clear1PMState(chatID)
		b.handleAdminCancel(message)
		return
	}

	weight, err := strconv.ParseFloat(strings.Replace(text, ",", ".", 1), 64)
	if err != nil || weight <= 0 {
		msg := tgbotapi.NewMessage(chatID, "Введите корректный вес (положительное число)")
		b.api.Send(msg)
		return
	}

	onePMStore.Lock()
	onePMStore.calcWeight[chatID] = weight
	onePMStore.Unlock()

	userStates.Lock()
	userStates.states[chatID] = state1PMCalcReps
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Сколько повторений было выполнено?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("1"),
			tgbotapi.NewKeyboardButton("2"),
			tgbotapi.NewKeyboardButton("3"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4"),
			tgbotapi.NewKeyboardButton("5"),
			tgbotapi.NewKeyboardButton("6"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("7"),
			tgbotapi.NewKeyboardButton("8"),
			tgbotapi.NewKeyboardButton("10"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handle1PMCalcReps handles reps input and calculates 1PM
func (b *Bot) handle1PMCalcReps(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clear1PMState(chatID)
		b.handleAdminCancel(message)
		return
	}

	reps, err := strconv.Atoi(text)
	if err != nil || reps <= 0 || reps > 30 {
		msg := tgbotapi.NewMessage(chatID, "Введите корректное количество повторений (1-30)")
		b.api.Send(msg)
		return
	}

	onePMStore.Lock()
	weight := onePMStore.calcWeight[chatID]
	method := onePMStore.calcMethod[chatID]
	onePMStore.calcReps[chatID] = reps
	onePMStore.Unlock()

	// Calculate 1PM
	calculated1PM := training.Calculate1PM(weight, reps, method)

	onePMStore.Lock()
	onePMStore.calculated1PM[chatID] = calculated1PM
	onePMStore.Unlock()

	methodName := training.CalcMethodName(method)

	b.show1PMConfirmation(chatID, calculated1PM, weight, reps, methodName)
}

// handle1PMManualInput handles manual 1PM input
func (b *Bot) handle1PMManualInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clear1PMState(chatID)
		b.handleAdminCancel(message)
		return
	}

	onePM, err := strconv.ParseFloat(strings.Replace(text, ",", ".", 1), 64)
	if err != nil || onePM <= 0 {
		msg := tgbotapi.NewMessage(chatID, "Введите корректный вес (положительное число)")
		b.api.Send(msg)
		return
	}

	onePMStore.Lock()
	onePMStore.calculated1PM[chatID] = onePM
	onePMStore.Unlock()

	b.show1PMConfirmation(chatID, onePM, 0, 0, "Ручной ввод")
}

// show1PMConfirmation shows confirmation before saving
func (b *Bot) show1PMConfirmation(chatID int64, onePM, weight float64, reps int, method string) {
	userStates.Lock()
	userStates.states[chatID] = state1PMConfirm
	userStates.Unlock()

	onePMStore.RLock()
	exerciseID := onePMStore.exerciseID[chatID]
	clientID := onePMStore.clientID[chatID]
	onePMStore.RUnlock()

	// Get exercise and client names
	var exerciseName, clientName string
	b.db.QueryRow("SELECT name FROM public.exercises WHERE id = $1", exerciseID).Scan(&exerciseName)
	b.db.QueryRow("SELECT name || ' ' || surname FROM public.clients WHERE id = $1", clientID).Scan(&clientName)

	text := fmt.Sprintf("📊 Подтверждение записи 1ПМ\n\n"+
		"👤 Клиент: %s\n"+
		"🏋️ Упражнение: %s\n"+
		"💪 1ПМ: %.1f кг\n"+
		"📐 Метод: %s\n",
		clientName, exerciseName, onePM, method)

	if weight > 0 && reps > 0 {
		text += fmt.Sprintf("📝 Исходные данные: %.1f кг × %d повт.\n", weight, reps)
	}

	text += "\nСохранить?"

	msg := tgbotapi.NewMessage(chatID, text)
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ Сохранить"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔄 Пересчитать"),
			tgbotapi.NewKeyboardButton("❌ Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handle1PMConfirm handles confirmation
func (b *Bot) handle1PMConfirm(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	switch text {
	case "✅ Сохранить":
		b.save1PM(chatID, message)
	case "🔄 Пересчитать":
		b.show1PMInputMethod(chatID)
	case "❌ Отмена":
		b.clear1PMState(chatID)
		b.handleAdminCancel(message)
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите действие из меню")
		b.api.Send(msg)
	}
}

// save1PM saves 1PM to database
func (b *Bot) save1PM(chatID int64, message *tgbotapi.Message) {
	onePMStore.RLock()
	clientID := onePMStore.clientID[chatID]
	exerciseID := onePMStore.exerciseID[chatID]
	onePM := onePMStore.calculated1PM[chatID]
	method := onePMStore.calcMethod[chatID]
	weight := onePMStore.calcWeight[chatID]
	reps := onePMStore.calcReps[chatID]
	onePMStore.RUnlock()

	// Insert into database
	_, err := b.db.Exec(`
		INSERT INTO public.exercise_1pm
			(client_id, exercise_id, one_pm_kg, test_date, calc_method, source_weight, source_reps, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		clientID, exerciseID, onePM, time.Now(), method, weight, reps, chatID)

	if err != nil {
		log.Printf("Ошибка сохранения 1ПМ: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения. Попробуйте снова.")
		b.api.Send(msg)
		return
	}

	// Get history
	history := b.get1PMHistory(clientID, exerciseID)

	responseText := fmt.Sprintf("✅ 1ПМ сохранён: %.1f кг\n\n", onePM)
	if len(history) > 1 {
		responseText += "📈 История:\n"
		for i, h := range history {
			if i >= 5 {
				break
			}
			responseText += fmt.Sprintf("• %s: %.1f кг\n", h.TestDate.Format("02.01.2006"), h.OnePMKg)
		}

		// Show progress
		if len(history) >= 2 {
			gain := history[0].OnePMKg - history[len(history)-1].OnePMKg
			gainPercent := (gain / history[len(history)-1].OnePMKg) * 100
			responseText += fmt.Sprintf("\n📊 Прогресс: %+.1f кг (%+.1f%%)\n", gain, gainPercent)
		}
	}

	msg := tgbotapi.NewMessage(chatID, responseText)
	b.api.Send(msg)

	b.clear1PMState(chatID)
	b.handle1PMMenu(message)
}

// handle1PMAddExercise handles adding new exercise
func (b *Bot) handle1PMAddExercise(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		onePMStore.RLock()
		clientID := onePMStore.clientID[chatID]
		onePMStore.RUnlock()

		userStates.Lock()
		userStates.states[chatID] = state1PMSelectExercise
		userStates.Unlock()

		b.show1PMExerciseList(chatID, clientID)
		return
	}

	// Normalize name
	nameNormalized := strings.ToLower(strings.TrimSpace(text))
	name := strings.TrimSpace(text)

	// Check if exists
	var existingID int
	err := b.db.QueryRow("SELECT id FROM public.exercises WHERE name_normalized = $1", nameNormalized).Scan(&existingID)
	if err == nil {
		// Exercise exists, use it
		onePMStore.Lock()
		onePMStore.exerciseID[chatID] = existingID
		onePMStore.Unlock()

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Упражнение \"%s\" уже существует. Используем его.", name))
		b.api.Send(msg)

		b.show1PMInputMethod(chatID)
		return
	}

	// Insert new exercise
	var newID int
	err = b.db.QueryRow(`
		INSERT INTO public.exercises (name, name_normalized, movement_type, is_trackable_1pm)
		VALUES ($1, $2, 'compound', true)
		RETURNING id`, name, nameNormalized).Scan(&newID)

	if err != nil {
		log.Printf("Ошибка добавления упражнения: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка добавления упражнения")
		b.api.Send(msg)
		return
	}

	onePMStore.Lock()
	onePMStore.exerciseID[chatID] = newID
	onePMStore.Unlock()

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Упражнение \"%s\" добавлено", name))
	b.api.Send(msg)

	b.show1PMInputMethod(chatID)
}

// get1PMHistory returns 1PM history for client and exercise
func (b *Bot) get1PMHistory(clientID, exerciseID int) []models.Exercise1PM {
	rows, err := b.db.Query(`
		SELECT id, one_pm_kg, test_date, COALESCE(calc_method, 'manual'),
		       COALESCE(source_weight, 0), COALESCE(source_reps, 0)
		FROM public.exercise_1pm
		WHERE client_id = $1 AND exercise_id = $2
		ORDER BY test_date DESC
		LIMIT 10`, clientID, exerciseID)
	if err != nil {
		log.Printf("Ошибка получения истории 1ПМ: %v", err)
		return nil
	}
	defer rows.Close()

	var history []models.Exercise1PM
	for rows.Next() {
		var h models.Exercise1PM
		if err := rows.Scan(&h.ID, &h.OnePMKg, &h.TestDate, &h.CalcMethod, &h.SourceWeight, &h.SourceReps); err != nil {
			log.Printf("Ошибка сканирования 1ПМ записи: %v", err)
			continue
		}
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Ошибка итерации по истории 1ПМ: %v", err)
	}
	return history
}

// clear1PMState clears all temporary 1PM data
func (b *Bot) clear1PMState(chatID int64) {
	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	onePMStore.Lock()
	delete(onePMStore.clientID, chatID)
	delete(onePMStore.exerciseID, chatID)
	delete(onePMStore.calcWeight, chatID)
	delete(onePMStore.calcReps, chatID)
	delete(onePMStore.calcMethod, chatID)
	delete(onePMStore.calculated1PM, chatID)
	onePMStore.Unlock()
}

// parse1PMClientID extracts client ID from button text
func parse1PMClientID(text string) int {
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		return 0
	}
	id, _ := strconv.Atoi(text[start+1 : end])
	return id
}

// parse1PMExerciseID extracts exercise ID from button text
func parse1PMExerciseID(text string) int {
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		return 0
	}
	id, _ := strconv.Atoi(text[start+1 : end])
	return id
}

// handle1PMState routes 1PM states
func (b *Bot) handle1PMState(message *tgbotapi.Message, state string) {
	switch state {
	case state1PMSelectClient:
		b.handle1PMClientSelect(message)
	case state1PMSelectExercise:
		b.handle1PMExerciseSelect(message)
	case state1PMInputMethod:
		b.handle1PMInputMethod(message)
	case state1PMManualInput:
		b.handle1PMManualInput(message)
	case state1PMCalcWeight:
		b.handle1PMCalcWeight(message)
	case state1PMCalcReps:
		b.handle1PMCalcReps(message)
	case state1PMAddExercise:
		b.handle1PMAddExercise(message)
	case state1PMConfirm:
		b.handle1PMConfirm(message)
	}
}

package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"workbot/clients/ai"
	"workbot/internal/excel"
	"workbot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// programStates хранит состояния создания программ
var programStates = struct {
	sync.RWMutex
	data map[int64]*programCreationState
}{data: make(map[int64]*programCreationState)}

type programCreationState struct {
	ClientID    int
	ClientName  string
	Goal        string
	DaysPerWeek int
	TotalWeeks  int
	Experience  string
	Equipment   string
	Injuries    string
	OnePMData   map[string]float64
}

// handleProgramsMenu показывает меню программ
func (b *Bot) handleProgramsMenu(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "Управление программами тренировок:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Создать программу"),
			tgbotapi.NewKeyboardButton("Активные программы"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отправить тренировку"),
			tgbotapi.NewKeyboardButton("Отметить выполнение"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleCreateProgram начинает создание программы
func (b *Bot) handleCreateProgram(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Показываем список клиентов для выбора
	rows, err := b.db.Query(`
		SELECT c.id, c.name, c.surname
		FROM public.clients c
		LEFT JOIN public.admins a ON c.telegram_id = a.telegram_id
		WHERE a.telegram_id IS NULL AND c.deleted_at IS NULL
		ORDER BY c.name`)
	if err != nil {
		log.Printf("Ошибка получения клиентов: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки клиентов")
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
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("Программа>> %s %s [%d]", name, surname, id)),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет клиентов для создания программы")
		b.api.Send(msg)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите клиента для создания программы:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "program_select_client"
	userStates.Unlock()
}

// handleProgramClientSelection обрабатывает выбор клиента для программы
func (b *Bot) handleProgramClientSelection(message *tgbotapi.Message, text string) {
	chatID := message.Chat.ID

	// Парсим ID клиента
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 {
		return
	}

	clientID, _ := strconv.Atoi(text[start+1 : end])
	if clientID == 0 {
		return
	}

	// Получаем данные клиента
	var name, surname string
	var goal sql.NullString
	b.db.QueryRow("SELECT name, surname, goal FROM public.clients WHERE id = $1", clientID).
		Scan(&name, &surname, &goal)

	// Инициализируем состояние создания программы
	state := &programCreationState{
		ClientID:   clientID,
		ClientName: fmt.Sprintf("%s %s", name, surname),
		OnePMData:  make(map[string]float64),
	}

	if goal.Valid {
		state.Goal = goal.String
	}

	// Получаем 1ПМ клиента
	rows, _ := b.db.Query(`
		SELECT e.name, opm.weight
		FROM public.exercise_1pm opm
		JOIN public.exercises e ON opm.exercise_id = e.id
		WHERE opm.client_id = $1
		AND opm.test_date = (
			SELECT MAX(test_date) FROM public.exercise_1pm
			WHERE client_id = $1 AND exercise_id = opm.exercise_id
		)`, clientID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var exName string
			var weight float64
			if rows.Scan(&exName, &weight) == nil {
				state.OnePMData[exName] = weight
			}
		}
	}

	programStates.Lock()
	programStates.data[chatID] = state
	programStates.Unlock()

	// Спрашиваем цель
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Создание программы для %s\n\nВведите цель тренировок:",
		state.ClientName))

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
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "program_set_goal"
	userStates.Unlock()
}

// handleProgramState обрабатывает состояния создания программы
func (b *Bot) handleProgramState(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	programStates.RLock()
	progState := programStates.data[chatID]
	programStates.RUnlock()

	if progState == nil {
		b.handleAdminStart(message)
		return
	}

	switch state {
	case "program_set_goal":
		progState.Goal = text
		b.askProgramDays(chatID)

	case "program_set_days":
		days, _ := strconv.Atoi(text)
		if days < 1 || days > 7 {
			msg := tgbotapi.NewMessage(chatID, "Укажите число от 1 до 7")
			b.api.Send(msg)
			return
		}
		progState.DaysPerWeek = days
		b.askProgramWeeks(chatID)

	case "program_set_weeks":
		weeks, _ := strconv.Atoi(text)
		if weeks < 1 || weeks > 52 {
			msg := tgbotapi.NewMessage(chatID, "Укажите число от 1 до 52")
			b.api.Send(msg)
			return
		}
		progState.TotalWeeks = weeks
		b.askProgramExperience(chatID)

	case "program_set_experience":
		switch text {
		case "Новичок":
			progState.Experience = "beginner"
		case "Средний":
			progState.Experience = "intermediate"
		case "Продвинутый":
			progState.Experience = "advanced"
		default:
			progState.Experience = "intermediate"
		}
		b.askProgramEquipment(chatID)

	case "program_set_equipment":
		progState.Equipment = text
		b.generateAndSaveProgram(chatID, progState)
	}
}

func (b *Bot) askProgramDays(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Сколько дней в неделю будут тренировки?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("2"),
			tgbotapi.NewKeyboardButton("3"),
			tgbotapi.NewKeyboardButton("4"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("5"),
			tgbotapi.NewKeyboardButton("6"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "program_set_days"
	userStates.Unlock()
}

func (b *Bot) askProgramWeeks(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "На сколько недель программа?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4"),
			tgbotapi.NewKeyboardButton("8"),
			tgbotapi.NewKeyboardButton("12"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("16"),
			tgbotapi.NewKeyboardButton("24"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "program_set_weeks"
	userStates.Unlock()
}

func (b *Bot) askProgramExperience(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Уровень подготовки клиента:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Новичок"),
			tgbotapi.NewKeyboardButton("Средний"),
			tgbotapi.NewKeyboardButton("Продвинутый"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "program_set_experience"
	userStates.Unlock()
}

func (b *Bot) askProgramEquipment(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Какое оборудование доступно?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Полный зал"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Только штанга и гантели"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Домашние тренировки"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "program_set_equipment"
	userStates.Unlock()
}

func (b *Bot) generateAndSaveProgram(chatID int64, state *programCreationState) {
	// Отправляем сообщение о генерации
	waitMsg := tgbotapi.NewMessage(chatID, "Генерирую программу через AI... Это может занять минуту.")
	b.api.Send(waitMsg)

	// Создаём запрос к AI
	req := ai.ProgramRequest{
		ClientName:  state.ClientName,
		Goal:        state.Goal,
		Experience:  state.Experience,
		DaysPerWeek: state.DaysPerWeek,
		TotalWeeks:  state.TotalWeeks,
		Equipment:   state.Equipment,
		Injuries:    state.Injuries,
		OnePMData:   state.OnePMData,
	}

	// Генерируем программу
	program, err := b.aiClient.GenerateFullProgram(req)
	if err != nil {
		log.Printf("Ошибка генерации программы: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка генерации: %v", err))
		b.api.Send(msg)
		b.handleAdminStart(&tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}})
		return
	}

	program.ClientID = state.ClientID

	// Сохраняем в Excel
	pm := excel.NewProgramManager(b.config.ClientsDir, b.config.JournalPath)
	filePath, err := pm.CreateProgramFile(program)
	if err != nil {
		log.Printf("Ошибка сохранения программы: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Программа создана, но ошибка сохранения файла: %v", err))
		b.api.Send(msg)
	} else {
		program.FilePath = filePath
	}

	// Сохраняем в БД
	err = b.saveProgramToDB(program)
	if err != nil {
		log.Printf("Ошибка сохранения в БД: %v", err)
	}

	// Обновляем журнал
	b.updateClientJournal(state.ClientID, program)

	// Очищаем состояние
	programStates.Lock()
	delete(programStates.data, chatID)
	programStates.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	// Показываем результат
	resultMsg := fmt.Sprintf(
		"Программа создана!\n\n"+
			"Клиент: %s\n"+
			"Цель: %s\n"+
			"Длительность: %d недель\n"+
			"Тренировок в неделю: %d\n"+
			"Всего тренировок: %d\n\n"+
			"Файл сохранён:\n%s",
		program.ClientName,
		program.Goal,
		program.TotalWeeks,
		program.DaysPerWeek,
		len(program.Workouts),
		filePath)

	msg := tgbotapi.NewMessage(chatID, resultMsg)
	b.api.Send(msg)

	// Предлагаем отправить первую тренировку
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отправить первую тренировку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	nextMsg := tgbotapi.NewMessage(chatID, "Хотите отправить первую тренировку клиенту?")
	nextMsg.ReplyMarkup = keyboard
	b.api.Send(nextMsg)
}

// handleSendNextWorkout отправляет следующую тренировку клиенту
func (b *Bot) handleSendNextWorkout(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Показываем клиентов с активными программами
	rows, err := b.db.Query(`
		SELECT DISTINCT c.id, c.name, c.surname, p.name as program_name
		FROM public.clients c
		JOIN public.training_programs p ON c.id = p.client_id
		WHERE p.status = 'active' AND c.deleted_at IS NULL
		ORDER BY c.name`)
	if err != nil {
		log.Printf("Ошибка получения программ: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки программ")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id int
		var name, surname, programName string
		if err := rows.Scan(&id, &name, &surname, &programName); err != nil {
			continue
		}
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("Трен>> %s %s [%d]", name, surname, id)),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет клиентов с активными программами")
		b.api.Send(msg)
		b.handleAdminStart(message)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите клиента для отправки тренировки:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "send_workout_select"
	userStates.Unlock()
}

// handleSendWorkoutSelection отправляет тренировку выбранному клиенту
func (b *Bot) handleSendWorkoutSelection(message *tgbotapi.Message, text string) {
	chatID := message.Chat.ID

	// Парсим ID клиента
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 {
		return
	}

	clientID, _ := strconv.Atoi(text[start+1 : end])
	if clientID == 0 {
		return
	}

	// Получаем telegram_id клиента
	var telegramID int64
	var name, surname string
	err := b.db.QueryRow("SELECT telegram_id, name, surname FROM public.clients WHERE id = $1", clientID).
		Scan(&telegramID, &name, &surname)
	if err != nil || telegramID == 0 {
		msg := tgbotapi.NewMessage(chatID, "У клиента нет Telegram для отправки")
		b.api.Send(msg)
		return
	}

	// Получаем следующую тренировку из файла
	var filePath string
	b.db.QueryRow(`SELECT file_path FROM public.training_programs WHERE client_id = $1 AND status = 'active' LIMIT 1`, clientID).
		Scan(&filePath)

	if filePath == "" {
		msg := tgbotapi.NewMessage(chatID, "Файл программы не найден")
		b.api.Send(msg)
		return
	}

	pm := excel.NewProgramManager(b.config.ClientsDir, b.config.JournalPath)
	workout, err := pm.GetNextWorkoutFromFile(filePath)
	if err != nil || workout == nil {
		msg := tgbotapi.NewMessage(chatID, "Нет невыполненных тренировок или ошибка чтения файла")
		b.api.Send(msg)
		return
	}

	// Форматируем и отправляем тренировку клиенту
	workoutMsg := ai.FormatWorkoutMessage(workout, workout.WeekNum)
	notification := tgbotapi.NewMessage(telegramID, workoutMsg)
	if _, err := b.api.Send(notification); err != nil {
		log.Printf("Ошибка отправки клиенту: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка отправки: %v", err))
		b.api.Send(msg)
		return
	}

	// Обновляем статус в файле
	pm.UpdateWorkoutStatus(filePath, workout.WeekNum, workout.Name, models.WorkoutStatusSent)

	// Уведомляем тренера
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Тренировка отправлена!\n\nКлиент: %s %s\nТренировка: %s",
		name, surname, workout.Name))
	b.api.Send(msg)

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	b.handleAdminStart(message)
}

// saveProgramToDB сохраняет программу в базу данных
func (b *Bot) saveProgramToDB(program *models.Program) error {
	_, err := b.db.Exec(`
		INSERT INTO public.training_programs
		(client_id, name, goal, description, total_weeks, days_per_week, start_date, status, file_path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
		program.ClientID, program.Name, program.Goal, program.Description,
		program.TotalWeeks, program.DaysPerWeek, program.StartDate,
		string(program.Status), program.FilePath)
	return err
}

// updateClientJournal обновляет запись клиента в журнале
func (b *Bot) updateClientJournal(clientID int, program *models.Program) {
	var name, surname, phone string
	var telegramID int64
	var goal sql.NullString
	var createdAt time.Time

	b.db.QueryRow(`SELECT name, surname, phone, telegram_id, goal, created_at
		FROM public.clients WHERE id = $1`, clientID).
		Scan(&name, &surname, &phone, &telegramID, &goal, &createdAt)

	entry := &models.JournalEntry{
		ClientID:        clientID,
		Name:            name,
		Surname:         surname,
		Phone:           phone,
		TelegramID:      telegramID,
		Goal:            goal.String,
		StartDate:       createdAt,
		TotalWorkouts:   len(program.Workouts),
		CurrentProgram:  program.Name,
		Status:          "active",
	}

	pm := excel.NewProgramManager(b.config.ClientsDir, b.config.JournalPath)
	if err := pm.UpdateJournal(entry); err != nil {
		log.Printf("Ошибка обновления журнала: %v", err)
	}
}

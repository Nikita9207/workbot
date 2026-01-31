package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"workbot/internal/generator"
	"workbot/internal/gsheets"
	"workbot/internal/models"
)

// fitnessStates хранит состояния для фитнес генераторов
var fitnessStates = struct {
	sync.RWMutex
	selector     *generator.ExerciseSelector
	programType  map[int64]string // hypertrophy, strength, fatloss, hyrox
	clientID     map[int64]int
	clientWeight map[int64]float64 // вес клиента
	weeks        map[int64]int
	daysPerWeek  map[int64]int
	split        map[int64]string
	includeHIIT  map[int64]bool
	lastProgram  map[int64]*models.GeneratedProgram
}{
	programType:  make(map[int64]string),
	clientID:     make(map[int64]int),
	clientWeight: make(map[int64]float64),
	weeks:        make(map[int64]int),
	daysPerWeek:  make(map[int64]int),
	split:        make(map[int64]string),
	includeHIIT:  make(map[int64]bool),
	lastProgram:  make(map[int64]*models.GeneratedProgram),
}

// getFitnessSelector возвращает или создаёт селектор упражнений
func getFitnessSelector() *generator.ExerciseSelector {
	fitnessStates.Lock()
	defer fitnessStates.Unlock()

	if fitnessStates.selector == nil {
		// Пробуем разные пути для data директории
		dataPaths := []string{"data", "/app/data", "./data"}
		var sel *generator.ExerciseSelector
		var err error

		for _, path := range dataPaths {
			sel, err = generator.NewExerciseSelector(path)
			if err == nil && sel != nil && len(sel.GetAllExercises()) > 0 {
				log.Printf("Загружено %d упражнений из %s", len(sel.GetAllExercises()), path)
				break
			}
		}

		if sel == nil || len(sel.GetAllExercises()) == 0 {
			log.Printf("Ошибка создания селектора или нет упражнений: %v", err)
			return nil
		}
		fitnessStates.selector = sel
	}
	return fitnessStates.selector
}

// handleFitnessMenu показывает меню фитнес программ
func (b *Bot) handleFitnessMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	msg := tgbotapi.NewMessage(chatID, "🏃 Генератор фитнес программ\n\n"+
		"Доступные программы:\n"+
		"• 💪 Гипертрофия (набор мышечной массы)\n"+
		"• 🏋️ Сила (максимальная сила)\n"+
		"• 🔥 Жиросжигание (похудение + тонус)\n"+
		"• 🏃 Hyrox (функциональный фитнес)\n\n"+
		"Выберите тип программы:")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("FIT: Гипертрофия"),
			tgbotapi.NewKeyboardButton("FIT: Сила"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("FIT: Жиросжигание"),
			tgbotapi.NewKeyboardButton("FIT: Hyrox"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleFitnessProgramType обрабатывает выбор типа программы
func (b *Bot) handleFitnessProgramType(message *tgbotapi.Message, programType string) {
	chatID := message.Chat.ID

	fitnessStates.Lock()
	fitnessStates.programType[chatID] = programType
	fitnessStates.Unlock()

	// Показываем клиентов для выбора
	b.showClientsForFitness(chatID, programType)
}

// showClientsForFitness показывает список клиентов
func (b *Bot) showClientsForFitness(chatID int64, programType string) {
	rows, err := b.db.Query(`
		SELECT c.id, c.name, c.surname
		FROM public.clients c
		WHERE c.deleted_at IS NULL
		ORDER BY c.name, c.surname`)
	if err != nil {
		log.Printf("Ошибка получения клиентов: %v", err)
		b.sendMessage(chatID, "Ошибка загрузки клиентов")
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
		buttonText := fmt.Sprintf("FIT>> %s %s [%d]", name, surname, id)
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		))
	}

	if len(buttons) == 0 {
		b.sendMessage(chatID, "Нет клиентов. Сначала добавьте клиента.")
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	typeName := getFitnessProgramTypeName(programType)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📋 Программа: %s\n\nВыберите клиента:", typeName))
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "fit_select_client"
	userStates.Unlock()
}

// handleFitnessClientSelect обрабатывает выбор клиента
func (b *Bot) handleFitnessClientSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearFitnessState(chatID)
		b.handleFitnessMenu(message)
		return
	}

	clientID := parseIDFromBrackets(text)
	if clientID == 0 {
		b.sendMessage(chatID, "Ошибка выбора клиента")
		return
	}

	fitnessStates.Lock()
	fitnessStates.clientID[chatID] = clientID
	fitnessStates.Unlock()

	// Спрашиваем вес клиента
	b.showFitnessWeightInput(chatID)
}

// showFitnessWeightInput запрашивает вес клиента
func (b *Bot) showFitnessWeightInput(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = "fit_enter_weight"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Введите вес клиента (кг):\n\nНапример: 70 или 65.5")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("60"),
			tgbotapi.NewKeyboardButton("70"),
			tgbotapi.NewKeyboardButton("80"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("90"),
			tgbotapi.NewKeyboardButton("100"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleFitnessWeightInput обрабатывает ввод веса
func (b *Bot) handleFitnessWeightInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearFitnessState(chatID)
		b.handleFitnessMenu(message)
		return
	}

	// Парсим вес
	var weight float64
	_, err := fmt.Sscanf(strings.Replace(text, ",", ".", 1), "%f", &weight)
	if err != nil || weight < 30 || weight > 300 {
		b.sendMessage(chatID, "❌ Введите корректный вес (от 30 до 300 кг)")
		return
	}

	fitnessStates.Lock()
	fitnessStates.clientWeight[chatID] = weight
	fitnessStates.Unlock()

	b.showFitnessWeeksSelection(chatID)
}

// showFitnessWeeksSelection показывает выбор количества недель
func (b *Bot) showFitnessWeeksSelection(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = "fit_select_weeks"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "На сколько недель составить программу?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4 недели"),
			tgbotapi.NewKeyboardButton("8 недель"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("12 недель"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleFitnessWeeksSelect обрабатывает выбор недель
func (b *Bot) handleFitnessWeeksSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearFitnessState(chatID)
		b.handleFitnessMenu(message)
		return
	}

	var weeks int
	switch text {
	case "4 недели":
		weeks = 4
	case "8 недель":
		weeks = 8
	case "12 недель":
		weeks = 12
	default:
		b.sendMessage(chatID, "Выберите из предложенных вариантов")
		return
	}

	fitnessStates.Lock()
	fitnessStates.weeks[chatID] = weeks
	fitnessStates.Unlock()

	b.showFitnessDaysSelection(chatID)
}

// showFitnessDaysSelection показывает выбор дней в неделю
func (b *Bot) showFitnessDaysSelection(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = "fit_select_days"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Сколько тренировок в неделю?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("2 дня"),
			tgbotapi.NewKeyboardButton("3 дня"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4 дня"),
			tgbotapi.NewKeyboardButton("5 дней"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleFitnessDaysSelect обрабатывает выбор дней
func (b *Bot) handleFitnessDaysSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearFitnessState(chatID)
		b.handleFitnessMenu(message)
		return
	}

	var days int
	switch text {
	case "2 дня":
		days = 2
	case "3 дня":
		days = 3
	case "4 дня":
		days = 4
	case "5 дней":
		days = 5
	default:
		b.sendMessage(chatID, "Выберите из предложенных вариантов")
		return
	}

	fitnessStates.Lock()
	fitnessStates.daysPerWeek[chatID] = days
	programType := fitnessStates.programType[chatID]
	fitnessStates.Unlock()

	// Для гипертрофии спрашиваем сплит
	if programType == "hypertrophy" {
		b.showFitnessSplitSelection(chatID)
		return
	}

	// Для жиросжигания спрашиваем про HIIT
	if programType == "fatloss" {
		b.showFitnessHIITSelection(chatID)
		return
	}

	// Для остальных сразу генерируем
	b.generateFitnessProgram(message)
}

// showFitnessSplitSelection показывает выбор сплита
func (b *Bot) showFitnessSplitSelection(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = "fit_select_split"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Выберите тип сплита:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Full Body"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Upper/Lower"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Push/Pull/Legs"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleFitnessSplitSelect обрабатывает выбор сплита
func (b *Bot) handleFitnessSplitSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearFitnessState(chatID)
		b.handleFitnessMenu(message)
		return
	}

	var split string
	switch text {
	case "Full Body":
		split = "fullbody"
	case "Upper/Lower":
		split = "upper_lower"
	case "Push/Pull/Legs":
		split = "push_pull_legs"
	default:
		b.sendMessage(chatID, "Выберите из предложенных вариантов")
		return
	}

	fitnessStates.Lock()
	fitnessStates.split[chatID] = split
	fitnessStates.Unlock()

	b.generateFitnessProgram(message)
}

// showFitnessHIITSelection показывает выбор HIIT
func (b *Bot) showFitnessHIITSelection(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = "fit_select_hiit"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Включить HIIT кардио в программу?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Да, включить HIIT"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Нет, только силовые"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleFitnessHIITSelect обрабатывает выбор HIIT
func (b *Bot) handleFitnessHIITSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearFitnessState(chatID)
		b.handleFitnessMenu(message)
		return
	}

	includeHIIT := text == "Да, включить HIIT"

	fitnessStates.Lock()
	fitnessStates.includeHIIT[chatID] = includeHIIT
	fitnessStates.Unlock()

	b.generateFitnessProgram(message)
}

// generateFitnessProgram генерирует фитнес программу
func (b *Bot) generateFitnessProgram(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	fitnessStates.RLock()
	programType := fitnessStates.programType[chatID]
	clientID := fitnessStates.clientID[chatID]
	clientWeight := fitnessStates.clientWeight[chatID]
	weeks := fitnessStates.weeks[chatID]
	days := fitnessStates.daysPerWeek[chatID]
	split := fitnessStates.split[chatID]
	includeHIIT := fitnessStates.includeHIIT[chatID]
	fitnessStates.RUnlock()

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Генерирую программу...")
	b.api.Send(waitMsg)

	// Загружаем профиль клиента
	client, err := b.loadClientProfile(clientID)
	if err != nil {
		log.Printf("Ошибка загрузки клиента: %v", err)
		b.sendMessage(chatID, "Ошибка загрузки данных клиента")
		return
	}

	// Используем введённый вес
	if clientWeight > 0 {
		client.Weight = clientWeight
	}

	selector := getFitnessSelector()
	if selector == nil {
		b.sendMessage(chatID, "Ошибка инициализации генератора")
		return
	}

	var program *models.GeneratedProgram

	switch programType {
	case "hypertrophy":
		gen := generator.NewHypertrophyGenerator(selector, client)
		config := generator.HypertrophyConfig{
			TotalWeeks:  weeks,
			DaysPerWeek: days,
			Split:       split,
		}
		program, err = gen.Generate(config)

	case "strength":
		gen := generator.NewStrengthGenerator(selector, client)
		config := generator.StrengthConfig{
			TotalWeeks:  weeks,
			DaysPerWeek: days,
			Focus:       "all",
		}
		program, err = gen.Generate(config)

	case "fatloss":
		gen := generator.NewFatLossGenerator(selector, client)
		config := generator.FatLossConfig{
			TotalWeeks:  weeks,
			DaysPerWeek: days,
			IncludeHIIT: includeHIIT,
		}
		program, err = gen.Generate(config)

	case "hyrox":
		gen := generator.NewHyroxGenerator(selector, client)
		config := generator.HyroxConfig{
			TotalWeeks:  weeks,
			DaysPerWeek: days,
		}
		program, err = gen.Generate(config)

	default:
		b.sendMessage(chatID, "Неизвестный тип программы")
		return
	}

	if err != nil {
		log.Printf("Ошибка генерации: %v", err)
		b.sendMessage(chatID, fmt.Sprintf("Ошибка генерации: %v", err))
		return
	}

	// Сохраняем программу
	fitnessStates.Lock()
	fitnessStates.lastProgram[chatID] = program
	fitnessStates.Unlock()

	// Показываем статистику
	statsMsg := fmt.Sprintf("✅ Программа сгенерирована!\n\n"+
		"📋 %s\n"+
		"👤 %s\n"+
		"📅 %d недель, %d тренировок/неделю\n\n"+
		"📊 Статистика:\n"+
		"• Всего тренировок: %d\n"+
		"• Всего подходов: %d\n"+
		"• Общий объём: %.0f кг\n\n"+
		"Фазы программы:\n",
		getFitnessProgramTypeName(programType),
		client.Name,
		program.TotalWeeks,
		program.DaysPerWeek,
		program.Statistics.TotalWorkouts,
		program.Statistics.TotalSets,
		program.Statistics.TotalVolume)

	for _, phase := range program.Phases {
		statsMsg += fmt.Sprintf("• Нед. %d-%d: %s\n", phase.WeekStart, phase.WeekEnd, phase.Name)
	}

	b.sendMessage(chatID, statsMsg)
	b.showFitnessProgramOptions(chatID)
}

// showFitnessProgramOptions показывает опции программы
func (b *Bot) showFitnessProgramOptions(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Что сделать с программой?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Показать 1-ю неделю"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Показать всю программу"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("FIT: Отправить клиенту"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("FIT: Экспорт в Google"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Новая программа"),
			tgbotapi.NewKeyboardButton("В меню"),
		),
	)
	msg.ReplyMarkup = keyboard

	userStates.Lock()
	userStates.states[chatID] = "fit_review"
	userStates.Unlock()

	b.api.Send(msg)
}

// handleFitnessReview обрабатывает действия с программой
func (b *Bot) handleFitnessReview(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	fitnessStates.RLock()
	program := fitnessStates.lastProgram[chatID]
	fitnessStates.RUnlock()

	if program == nil {
		b.sendMessage(chatID, "Программа не найдена")
		b.handleFitnessMenu(message)
		return
	}

	switch text {
	case "Показать 1-ю неделю":
		if len(program.Weeks) > 0 {
			formatted := formatFitnessWeek(program.Weeks[0])
			sendLongMessage(b, chatID, "📅 Неделя 1:\n\n"+formatted)
		}

	case "Показать всю программу":
		formatted := formatFitnessProgram(program)
		sendLongMessage(b, chatID, formatted)

	case "FIT: Отправить клиенту":
		b.handleFitnessSendToClient(message)
		return

	case "FIT: Экспорт в Google":
		b.handleFITExportToGoogle(message)
		return

	case "Новая программа":
		b.clearFitnessState(chatID)
		b.handleFitnessMenu(message)
		return

	case "В меню":
		b.clearFitnessState(chatID)
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()
		b.handleAdminStart(message)
		return
	}
}

// handleFitnessSendToClient отправляет программу клиенту
func (b *Bot) handleFitnessSendToClient(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	fitnessStates.RLock()
	program := fitnessStates.lastProgram[chatID]
	clientID := fitnessStates.clientID[chatID]
	fitnessStates.RUnlock()

	if program == nil {
		b.sendMessage(chatID, "Программа не найдена")
		return
	}

	// Получаем telegram_id клиента
	var telegramID int64
	var name, surname string
	err := b.db.QueryRow("SELECT telegram_id, name, surname FROM public.clients WHERE id = $1", clientID).
		Scan(&telegramID, &name, &surname)
	if err != nil {
		b.sendMessage(chatID, "Клиент не найден")
		return
	}

	if telegramID == 0 {
		b.sendMessage(chatID, fmt.Sprintf("❌ У клиента %s %s нет Telegram ID", name, surname))
		b.showFitnessProgramOptions(chatID)
		return
	}

	// Форматируем программу
	formatted := formatFitnessProgram(program)

	// Отправляем клиенту
	introMsg := fmt.Sprintf("🏋️ Твоя программа тренировок!\n\n"+
		"📋 Цель: %s\n"+
		"📅 %d недель, %d тренировок/неделю\n\n"+
		"Полная программа в файле ниже.",
		getFitnessProgramTypeName(string(program.Goal)),
		program.TotalWeeks,
		program.DaysPerWeek)

	clientMsg := tgbotapi.NewMessage(telegramID, introMsg)
	b.api.Send(clientMsg)

	// Отправляем файл
	doc := tgbotapi.NewDocument(telegramID, tgbotapi.FileBytes{
		Name:  fmt.Sprintf("Программа_%s.txt", sanitizeFilename(name)),
		Bytes: []byte(formatted),
	})
	b.api.Send(doc)

	b.sendMessage(chatID, fmt.Sprintf("✅ Программа отправлена клиенту %s %s", name, surname))
	b.showFitnessProgramOptions(chatID)
}

// loadClientProfile загружает профиль клиента из БД
func (b *Bot) loadClientProfile(clientID int) (*models.ClientProfile, error) {
	var name, surname string

	err := b.db.QueryRow(`
		SELECT name, surname
		FROM public.clients WHERE id = $1`, clientID).
		Scan(&name, &surname)
	if err != nil {
		return nil, err
	}

	profile := &models.ClientProfile{
		ID:         clientID,
		Name:       name + " " + surname,
		Gender:     "male",                       // По умолчанию
		Experience: models.ExpIntermediate,       // По умолчанию
		Age:        30,                           // По умолчанию
		Weight:     70,                           // По умолчанию
	}

	// Загружаем данные из анкеты клиента (если есть)
	var gender, experience sql.NullString
	var weight sql.NullFloat64
	var height sql.NullInt64
	err = b.db.QueryRow(`
		SELECT gender, weight, height, experience
		FROM public.client_forms
		WHERE client_id = $1
		ORDER BY created_at DESC LIMIT 1`, clientID).
		Scan(&gender, &weight, &height, &experience)
	if err == nil {
		if gender.Valid && gender.String != "" {
			profile.Gender = gender.String
		}
		if weight.Valid && weight.Float64 > 0 {
			profile.Weight = weight.Float64
		}
		if height.Valid && height.Int64 > 0 {
			profile.Height = float64(height.Int64)
		}
		if experience.Valid && experience.String != "" {
			switch experience.String {
			case "beginner", "новичок":
				profile.Experience = models.ExpBeginner
			case "intermediate", "средний":
				profile.Experience = models.ExpIntermediate
			case "advanced", "продвинутый":
				profile.Experience = models.ExpAdvanced
			}
		}
	}

	// Загружаем 1ПМ
	profile.OnePM = make(map[string]float64)
	rows, err := b.db.Query(`
		SELECT e.name, ep.one_pm_kg
		FROM public.exercise_1pm ep
		JOIN public.exercises e ON ep.exercise_id = e.id
		WHERE ep.client_id = $1
		ORDER BY ep.test_date DESC`, clientID)
	if err == nil {
		defer rows.Close()
		seen := make(map[string]bool)
		for rows.Next() {
			var exName string
			var onePM float64
			if err := rows.Scan(&exName, &onePM); err == nil {
				movement := mapExerciseToMovement(exName)
				if movement != "" && !seen[movement] {
					profile.OnePM[movement] = onePM
					seen[movement] = true
				}
			}
		}
	}

	// По умолчанию всё оборудование доступно
	profile.AvailableEquip = []models.EquipmentType{
		models.EquipmentBarbell,
		models.EquipmentDumbbell,
		models.EquipmentKettlebell,
		models.EquipmentCable,
		models.EquipmentMachine,
		models.EquipmentTRX,
	}

	return profile, nil
}

// handleFitnessState роутер состояний фитнеса
func (b *Bot) handleFitnessState(message *tgbotapi.Message, state string) {
	switch state {
	case "fit_select_client":
		b.handleFitnessClientSelect(message)
	case "fit_select_type_for_client":
		b.handleFitSelectTypeForClient(message)
	case "fit_enter_weight":
		b.handleFitnessWeightInput(message)
	case "fit_select_weeks":
		b.handleFitnessWeeksSelect(message)
	case "fit_select_days":
		b.handleFitnessDaysSelect(message)
	case "fit_select_split":
		b.handleFitnessSplitSelect(message)
	case "fit_select_hiit":
		b.handleFitnessHIITSelect(message)
	case "fit_review":
		b.handleFitnessReview(message)
	}
}

// handleFitSelectTypeForClient обрабатывает выбор типа программы когда клиент уже выбран
func (b *Bot) handleFitSelectTypeForClient(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Назад" {
		b.clearFitnessState(chatID)
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

	var programType string
	switch text {
	case "FIT: Гипертрофия":
		programType = "hypertrophy"
	case "FIT: Сила":
		programType = "strength"
	case "FIT: Жиросжигание":
		programType = "fatloss"
	case "FIT: Hyrox":
		programType = "hyrox"
	default:
		b.sendMessage(chatID, "Выберите тип программы из меню")
		return
	}

	b.handleFitnessProgramTypeForClient(message, programType)
}

// clearFitnessState очищает состояние
func (b *Bot) clearFitnessState(chatID int64) {
	fitnessStates.Lock()
	delete(fitnessStates.programType, chatID)
	delete(fitnessStates.clientID, chatID)
	delete(fitnessStates.clientWeight, chatID)
	delete(fitnessStates.weeks, chatID)
	delete(fitnessStates.daysPerWeek, chatID)
	delete(fitnessStates.split, chatID)
	delete(fitnessStates.includeHIIT, chatID)
	delete(fitnessStates.lastProgram, chatID)
	fitnessStates.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()
}

// === Вспомогательные функции ===

func getFitnessProgramTypeName(programType string) string {
	names := map[string]string{
		"hypertrophy": "💪 Гипертрофия",
		"strength":    "🏋️ Сила",
		"fatloss":     "🔥 Жиросжигание",
		"hyrox":       "🏃 Hyrox",
	}
	if name, ok := names[programType]; ok {
		return name
	}
	return programType
}

func mapExerciseToMovement(exName string) string {
	exLower := strings.ToLower(exName)
	if strings.Contains(exLower, "присед") {
		return "squat"
	}
	if strings.Contains(exLower, "жим") && strings.Contains(exLower, "лёж") {
		return "bench"
	}
	if strings.Contains(exLower, "станов") {
		return "deadlift"
	}
	if strings.Contains(exLower, "жим") && strings.Contains(exLower, "стоя") {
		return "ohp"
	}
	return ""
}

func formatFitnessWeek(week models.GeneratedWeek) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📆 Неделя %d — %s\n", week.WeekNum, week.PhaseName))
	if week.IsDeload {
		sb.WriteString("⚡ Разгрузочная неделя\n")
	}
	sb.WriteString(fmt.Sprintf("Интенсивность: %.0f%% | RPE: %.1f\n\n", week.IntensityPercent, week.RPETarget))

	for _, day := range week.Days {
		sb.WriteString(fmt.Sprintf("━━━ %s ━━━\n", day.Name))
		if day.EstimatedDuration > 0 {
			sb.WriteString(fmt.Sprintf("⏱ ~%d мин\n", day.EstimatedDuration))
		}
		sb.WriteString("\n")

		for _, ex := range day.Exercises {
			line := fmt.Sprintf("%d. %s", ex.OrderNum, ex.ExerciseName)
			if ex.Sets > 0 {
				line += fmt.Sprintf(" — %dx%s", ex.Sets, ex.Reps)
			}
			if ex.Weight > 0 {
				line += fmt.Sprintf(" @ %.0f кг", ex.Weight)
			} else if ex.WeightPercent > 0 {
				line += fmt.Sprintf(" @ %.0f%%", ex.WeightPercent)
			}
			if ex.RestSeconds > 0 {
				line += fmt.Sprintf(" (отдых %d сек)", ex.RestSeconds)
			}
			sb.WriteString(line + "\n")

			if ex.Notes != "" {
				sb.WriteString(fmt.Sprintf("   💡 %s\n", ex.Notes))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatFitnessProgram(program *models.GeneratedProgram) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ПРОГРАММА: %s\n\n", getFitnessProgramTypeName(string(program.Goal))))
	sb.WriteString(fmt.Sprintf("Клиент: %s\n", program.ClientName))
	sb.WriteString(fmt.Sprintf("Длительность: %d недель\n", program.TotalWeeks))
	sb.WriteString(fmt.Sprintf("Тренировок в неделю: %d\n\n", program.DaysPerWeek))

	sb.WriteString("══════════════════════════════════\n")
	sb.WriteString("ФАЗЫ ПРОГРАММЫ\n")
	sb.WriteString("══════════════════════════════════\n\n")
	for _, phase := range program.Phases {
		sb.WriteString(fmt.Sprintf("▸ %s (нед. %d-%d): %s\n",
			phase.Name, phase.WeekStart, phase.WeekEnd, phase.Focus))
	}
	sb.WriteString("\n")

	for _, week := range program.Weeks {
		sb.WriteString(formatFitnessWeek(week))
		sb.WriteString("\n")
	}

	return sb.String()
}

// parseIDFromBrackets определена в helpers.go

// handleFITProgramForClient переходит к FIT программам с уже выбранным клиентом
func (b *Bot) handleFITProgramForClient(message *tgbotapi.Message, clientID int) {
	chatID := message.Chat.ID

	// Сохраняем clientID
	fitnessStates.Lock()
	fitnessStates.clientID[chatID] = clientID
	fitnessStates.Unlock()

	// Показываем выбор типа программы
	msg := tgbotapi.NewMessage(chatID, "🏃 Выберите тип программы:")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("FIT: Гипертрофия"),
			tgbotapi.NewKeyboardButton("FIT: Сила"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("FIT: Жиросжигание"),
			tgbotapi.NewKeyboardButton("FIT: Hyrox"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "fit_select_type_for_client"
	userStates.Unlock()
}

// handleFITExportToGoogle экспортирует FIT программу в Google Sheets
func (b *Bot) handleFITExportToGoogle(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	fitnessStates.RLock()
	program := fitnessStates.lastProgram[chatID]
	fitnessStates.RUnlock()

	if program == nil {
		b.sendMessage(chatID, "Программа не найдена")
		b.handleFitnessMenu(message)
		return
	}

	if b.sheetsClient == nil {
		b.sendMessage(chatID, "❌ Google Sheets не настроен")
		b.showFitnessProgramOptions(chatID)
		return
	}

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Создаю таблицу в Google Sheets...")
	b.api.Send(waitMsg)

	// Конвертируем в формат gsheets
	programData := convertToFITProgramData(program)

	// Создаём таблицу
	spreadsheetID, err := b.sheetsClient.CreateProgramSpreadsheet(programData)
	if err != nil {
		b.sendError(chatID, "Ошибка создания таблицы", err)
		b.showFitnessProgramOptions(chatID)
		return
	}

	url := gsheets.GetSpreadsheetURL(spreadsheetID)

	b.sendMessage(chatID, fmt.Sprintf("✅ Таблица создана!\n\n📋 %s — %s\n\n🔗 %s",
		program.ClientName, getFitnessProgramTypeName(string(program.Goal)), url))
	b.showFitnessProgramOptions(chatID)
}

// convertToFITProgramData конвертирует models.GeneratedProgram в gsheets.ProgramData
func convertToFITProgramData(program *models.GeneratedProgram) gsheets.ProgramData {
	data := gsheets.ProgramData{
		ClientName:  program.ClientName,
		ProgramName: getFitnessProgramTypeName(string(program.Goal)),
		Goal:        string(program.Goal),
		TotalWeeks:  program.TotalWeeks,
		DaysPerWeek: program.DaysPerWeek,
		Methodology: string(program.Periodization),
		CreatedAt:   time.Now().Format("02.01.2006"),
		OnePMData:   make(map[string]float64),
	}

	// Период программы
	if len(program.Phases) > 0 {
		phases := make([]string, 0, len(program.Phases))
		for _, p := range program.Phases {
			phases = append(phases, p.Name)
		}
		data.Period = strings.Join(phases, " → ")
	}

	// Конвертируем недели
	for _, week := range program.Weeks {
		weekData := gsheets.WeekData{
			WeekNum:          week.WeekNum,
			Phase:            week.PhaseName,
			IntensityPercent: week.IntensityPercent,
			VolumePercent:    week.VolumePercent,
			RPETarget:        week.RPETarget,
			IsDeload:         week.IsDeload,
		}

		for _, day := range week.Days {
			workoutData := gsheets.WorkoutData{
				DayNum: day.DayNum,
				Name:   day.Name,
				Type:   day.Type,
			}

			// Собираем группы мышц
			for _, mg := range day.MuscleGroups {
				workoutData.MuscleGroups = append(workoutData.MuscleGroups, string(mg))
			}

			for _, ex := range day.Exercises {
				exData := gsheets.ExerciseData{
					OrderNum:      ex.OrderNum,
					Name:          ex.ExerciseName,
					MuscleGroup:   string(ex.MuscleGroup),
					MovementType:  string(ex.MovementType),
					Sets:          ex.Sets,
					Reps:          ex.Reps,
					WeightPercent: ex.WeightPercent,
					WeightKg:      ex.Weight,
					RestSeconds:   ex.RestSeconds,
					Tempo:         ex.Tempo,
					RPE:           ex.RPE,
					Notes:         ex.Notes,
				}
				workoutData.Exercises = append(workoutData.Exercises, exData)

				// Собираем 1ПМ данные из весов
				if ex.WeightPercent > 0 && ex.Weight > 0 {
					onepm := ex.Weight / (ex.WeightPercent / 100)
					if _, exists := data.OnePMData[ex.ExerciseName]; !exists {
						data.OnePMData[ex.ExerciseName] = onepm
					}
				}
			}

			weekData.Workouts = append(weekData.Workouts, workoutData)
		}

		data.Weeks = append(data.Weeks, weekData)
	}

	return data
}

// handleFitnessProgramTypeForClient обрабатывает выбор типа когда клиент уже выбран
func (b *Bot) handleFitnessProgramTypeForClient(message *tgbotapi.Message, programType string) {
	chatID := message.Chat.ID

	fitnessStates.Lock()
	fitnessStates.programType[chatID] = programType
	fitnessStates.Unlock()

	// Клиент уже выбран - переходим к вводу веса
	b.showFitnessWeightInput(chatID)
}

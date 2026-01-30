package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"workbot/clients/ai"
	"workbot/internal/gsheets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// plStates хранит состояния для генератора пауэрлифтинга
var plStates = struct {
	sync.RWMutex
	generator     *ai.ProgramGenerator
	selectedLift  map[int64]ai.LiftType
	selectedTempl map[int64]string
	squat         map[int64]float64
	bench         map[int64]float64
	deadlift      map[int64]float64
	hipThrust     map[int64]float64
	daysPerWeek   map[int64]int
	lastProgram   map[int64]*ai.PLGeneratedProgram
}{
	selectedLift:  make(map[int64]ai.LiftType),
	selectedTempl: make(map[int64]string),
	squat:         make(map[int64]float64),
	bench:         make(map[int64]float64),
	deadlift:      make(map[int64]float64),
	hipThrust:     make(map[int64]float64),
	daysPerWeek:   make(map[int64]int),
	lastProgram:   make(map[int64]*ai.PLGeneratedProgram),
}

// getPLGenerator возвращает или создаёт генератор программ
func getPLGenerator() *ai.ProgramGenerator {
	plStates.Lock()
	defer plStates.Unlock()

	if plStates.generator == nil {
		gen, err := ai.NewProgramGenerator()
		if err != nil {
			log.Printf("Ошибка создания генератора: %v", err)
			return nil
		}
		plStates.generator = gen
	}
	return plStates.generator
}

// handlePowerliftingMenu показывает меню пауэрлифтинга
func (b *Bot) handlePowerliftingMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	msg := tgbotapi.NewMessage(chatID, "🏋️ Генератор программ\n\n"+
		"Силовые программы:\n"+
		"• Шейко (КМС/МС уровень)\n"+
		"• Головинский (циклы 2/7/11 недель)\n"+
		"• Верхошанский (специализация)\n"+
		"• Муравьёв (классическая периодизация)\n"+
		"• Русский цикл (6 недель)\n\n"+
		"Фитнес/силовой спорт:\n"+
		"• Ягодичный мост 12 недель (Hip Thrust)\n\n"+
		"Выберите действие:")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Троеборье"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Жим лёжа"),
			tgbotapi.NewKeyboardButton("PL: Присед"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Становая тяга"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Ягодичный мост"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Авто-подбор"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Список шаблонов"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePLLiftType обрабатывает выбор типа дисциплины
func (b *Bot) handlePLLiftType(message *tgbotapi.Message, liftType ai.LiftType) {
	chatID := message.Chat.ID

	plStates.Lock()
	plStates.selectedLift[chatID] = liftType
	plStates.Unlock()

	gen := getPLGenerator()
	if gen == nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка инициализации генератора")
		b.api.Send(msg)
		return
	}

	// Показываем шаблоны, подходящие для выбранной дисциплины
	templates := gen.ListTemplates()
	var buttons [][]tgbotapi.KeyboardButton

	// Фильтруем шаблоны по типу дисциплины
	var relevantTemplates []string
	for _, t := range templates {
		tLower := strings.ToLower(t)
		switch liftType {
		case ai.LiftTypeBench:
			if strings.Contains(tLower, "жим") || strings.Contains(tLower, "bench") ||
				strings.Contains(tLower, "шейко") || strings.Contains(tLower, "головинский") {
				relevantTemplates = append(relevantTemplates, t)
			}
		case ai.LiftTypeSquat:
			if strings.Contains(tLower, "присед") || strings.Contains(tLower, "squat") ||
				strings.Contains(tLower, "шейко") || strings.Contains(tLower, "головинский") ||
				strings.Contains(tLower, "русский") {
				relevantTemplates = append(relevantTemplates, t)
			}
		case ai.LiftTypeDeadlift:
			if strings.Contains(tLower, "тяга") || strings.Contains(tLower, "deadlift") ||
				strings.Contains(tLower, "головинский") {
				relevantTemplates = append(relevantTemplates, t)
			}
		case ai.LiftTypeHipThrust:
			if strings.Contains(tLower, "ягодич") || strings.Contains(tLower, "мост") ||
				strings.Contains(tLower, "hip") || strings.Contains(tLower, "thrust") {
				relevantTemplates = append(relevantTemplates, t)
			}
		default: // Full powerlifting
			relevantTemplates = append(relevantTemplates, t)
		}
	}

	for _, t := range relevantTemplates {
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("TPL: %s", t)),
		))
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	liftName := getLiftTypeName(liftType)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Выбрана дисциплина: %s\n\nВыберите шаблон программы:", liftName))
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)

	userStates.Lock()
	userStates.states[chatID] = "pl_select_template"
	userStates.Unlock()
}

// handlePLTemplateSelection обрабатывает выбор шаблона
func (b *Bot) handlePLTemplateSelection(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	// Убираем префикс "TPL: "
	templateName := strings.TrimPrefix(text, "TPL: ")

	plStates.Lock()
	plStates.selectedTempl[chatID] = templateName
	liftType := plStates.selectedLift[chatID]
	plStates.Unlock()

	userStates.Lock()
	userStates.states[chatID] = "pl_enter_maxes"
	userStates.Unlock()

	// Запрашиваем максимумы в зависимости от дисциплины
	var prompt string
	switch liftType {
	case ai.LiftTypeBench:
		prompt = "Введите ваш 1ПМ в жиме лёжа (кг):\n\nПример: 100"
	case ai.LiftTypeSquat:
		prompt = "Введите ваш 1ПМ в приседе (кг):\n\nПример: 150"
	case ai.LiftTypeDeadlift:
		prompt = "Введите ваш 1ПМ в становой тяге (кг):\n\nПример: 180"
	case ai.LiftTypeHipThrust:
		prompt = "Введите ваш 1ПМ в ягодичном мосте (кг):\n\n" +
			"Пример: 200\n\n" +
			"(Опционально: добавьте жим через запятую для верха тела)\n" +
			"Пример: 200, 80"
	default:
		prompt = "Введите ваши максимумы (1ПМ) через запятую:\n" +
			"Присед, Жим, Тяга\n\n" +
			"Пример: 150, 100, 180"
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Шаблон: %s\n\n%s", templateName, prompt))
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePLMaxesInput обрабатывает ввод максимумов
func (b *Bot) handlePLMaxesInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	plStates.RLock()
	liftType := plStates.selectedLift[chatID]
	plStates.RUnlock()

	var squat, bench, deadlift, hipThrust float64
	var err error

	switch liftType {
	case ai.LiftTypeBench:
		bench, err = strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || bench <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Введите число (например: 100)")
			b.api.Send(msg)
			return
		}
	case ai.LiftTypeSquat:
		squat, err = strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || squat <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Введите число (например: 150)")
			b.api.Send(msg)
			return
		}
	case ai.LiftTypeDeadlift:
		deadlift, err = strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || deadlift <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Введите число (например: 180)")
			b.api.Send(msg)
			return
		}
	case ai.LiftTypeHipThrust:
		// Hip Thrust + опционально жим
		parts := strings.Split(text, ",")
		hipThrust, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil || hipThrust <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Введите число (например: 200)")
			b.api.Send(msg)
			return
		}
		// Опционально жим для верха тела
		if len(parts) > 1 {
			bench, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		}
	default:
		// Троеборье - парсим три числа
		parts := strings.Split(text, ",")
		if len(parts) != 3 {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Введите три числа через запятую:\nПрисед, Жим, Тяга\n\nПример: 150, 100, 180")
			b.api.Send(msg)
			return
		}
		squat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil || squat <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Неверное значение приседа")
			b.api.Send(msg)
			return
		}
		bench, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || bench <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Неверное значение жима")
			b.api.Send(msg)
			return
		}
		deadlift, err = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil || deadlift <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Неверное значение тяги")
			b.api.Send(msg)
			return
		}
	}

	plStates.Lock()
	plStates.squat[chatID] = squat
	plStates.bench[chatID] = bench
	plStates.deadlift[chatID] = deadlift
	plStates.hipThrust[chatID] = hipThrust
	plStates.Unlock()

	// Спрашиваем количество тренировок в неделю
	userStates.Lock()
	userStates.states[chatID] = "pl_select_days"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Сколько тренировок в неделю?\n\n"+
		"0 = как в шаблоне (по умолчанию)")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Как в шаблоне"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("2 дня"),
			tgbotapi.NewKeyboardButton("3 дня"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4 дня"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePLDaysInput обрабатывает выбор количества дней
func (b *Bot) handlePLDaysInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	var days int
	switch text {
	case "Как в шаблоне":
		days = 0
	case "2 дня":
		days = 2
	case "3 дня":
		days = 3
	case "4 дня":
		days = 4
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите вариант из кнопок")
		b.api.Send(msg)
		return
	}

	plStates.Lock()
	plStates.daysPerWeek[chatID] = days
	plStates.Unlock()

	// Генерируем программу
	b.generatePLProgram(message)
}

// generatePLProgram генерирует программу пауэрлифтинга
func (b *Bot) generatePLProgram(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	plStates.RLock()
	liftType := plStates.selectedLift[chatID]
	templateName := plStates.selectedTempl[chatID]
	squat := plStates.squat[chatID]
	bench := plStates.bench[chatID]
	deadlift := plStates.deadlift[chatID]
	hipThrust := plStates.hipThrust[chatID]
	days := plStates.daysPerWeek[chatID]
	plStates.RUnlock()

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Генерирую программу...")
	b.api.Send(waitMsg)

	gen := getPLGenerator()
	if gen == nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка генератора")
		b.api.Send(msg)
		return
	}

	maxes := ai.AthleteMaxes{
		Squat:     squat,
		Bench:     bench,
		Deadlift:  deadlift,
		HipThrust: hipThrust,
	}

	opts := ai.GenerationOptions{
		LiftType:         liftType,
		DaysPerWeek:      days,
		IncludeAccessory: true,
	}

	program, err := gen.GenerateFromTemplateWithOptions(templateName, maxes, opts)
	if err != nil {
		log.Printf("Ошибка генерации: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка генерации: %v", err))
		b.api.Send(msg)
		b.handlePowerliftingMenu(message)
		return
	}

	// Сохраняем программу
	plStates.Lock()
	plStates.lastProgram[chatID] = program
	plStates.Unlock()

	// Валидация
	validation := gen.ValidateProgram(program)

	// Форматируем статистику
	statsMsg := fmt.Sprintf("✅ Программа сгенерирована!\n\n"+
		"📊 Статистика:\n"+
		"• Шаблон: %s\n"+
		"• Недель: %d\n"+
		"• Тренировок: %d\n"+
		"• Общий КПШ: %d\n"+
		"• Тоннаж: %.1f т\n"+
		"• Средний КПШ/нед: %.0f\n",
		program.Name,
		len(program.Weeks),
		validation.Stats.TotalWorkouts,
		program.TotalKPS,
		program.TotalTonnage,
		validation.Stats.AvgKPSPerWeek)

	if len(validation.Warnings) > 0 {
		statsMsg += "\n⚠️ Предупреждения:\n"
		for _, w := range validation.Warnings {
			statsMsg += fmt.Sprintf("• %s\n", w)
		}
	}

	msg := tgbotapi.NewMessage(chatID, statsMsg)
	b.api.Send(msg)

	// Показываем опции
	b.showPLProgramOptions(chatID)
}

// showPLProgramOptions показывает меню действий с программой
func (b *Bot) showPLProgramOptions(chatID int64) {
	optionsMsg := tgbotapi.NewMessage(chatID, "Что сделать с программой?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Показать 1-ю неделю"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Показать всю программу"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Экспорт в файл"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Отправить клиенту"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Экспорт в Google"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Новая программа"),
			tgbotapi.NewKeyboardButton("В меню PL"),
		),
	)
	optionsMsg.ReplyMarkup = keyboard

	userStates.Lock()
	userStates.states[chatID] = "pl_review"
	userStates.Unlock()

	b.api.Send(optionsMsg)
}

// handlePLReview обрабатывает действия с программой
func (b *Bot) handlePLReview(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	plStates.RLock()
	program := plStates.lastProgram[chatID]
	plStates.RUnlock()

	if program == nil {
		msg := tgbotapi.NewMessage(chatID, "Программа не найдена")
		b.api.Send(msg)
		b.handlePowerliftingMenu(message)
		return
	}

	switch text {
	case "Показать 1-ю неделю":
		if len(program.Weeks) > 0 {
			formatted := ai.FormatWeekCompact(program.Weeks[0])
			sendLongMessage(b, chatID, "📅 Неделя 1:\n\n"+formatted)
		}

	case "Показать всю программу":
		formatted := ai.FormatPLProgram(program)
		sendLongMessage(b, chatID, formatted)

	case "Экспорт в файл":
		formatted := ai.FormatPLProgram(program)
		// Отправляем как документ
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
			Name:  fmt.Sprintf("%s.txt", sanitizeFilename(program.Name)),
			Bytes: []byte(formatted),
		})
		doc.Caption = fmt.Sprintf("Программа: %s", program.Name)
		b.api.Send(doc)

	case "PL: Отправить клиенту":
		b.handlePLSendToClient(message)
		return

	case "PL: Экспорт в Google":
		b.handlePLExportToGoogle(message)
		return

	case "Новая программа":
		// Очищаем состояние
		clearPLState(chatID)
		b.handlePowerliftingMenu(message)
		return

	case "В меню PL":
		clearPLState(chatID)
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()
		b.handlePowerliftingMenu(message)
		return
	}
}

// handlePLAutoSelect автоматический подбор программы
func (b *Bot) handlePLAutoSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = "pl_auto_maxes"
	userStates.Unlock()

	plStates.Lock()
	plStates.selectedLift[chatID] = ai.LiftTypeFull
	plStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "🎯 Авто-подбор программы\n\n"+
		"Система подберёт оптимальный шаблон на основе вашего уровня.\n\n"+
		"Введите ваши максимумы (1ПМ) через запятую:\n"+
		"Присед, Жим, Тяга\n\n"+
		"Пример: 150, 100, 180")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePLAutoMaxesInput обрабатывает ввод для авто-подбора
func (b *Bot) handlePLAutoMaxesInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	parts := strings.Split(text, ",")
	if len(parts) != 3 {
		msg := tgbotapi.NewMessage(chatID, "Введите три числа через запятую:\nПрисед, Жим, Тяга")
		b.api.Send(msg)
		return
	}

	squat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	bench, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	deadlift, err3 := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)

	if err1 != nil || err2 != nil || err3 != nil || squat <= 0 || bench <= 0 || deadlift <= 0 {
		msg := tgbotapi.NewMessage(chatID, "Неверный формат. Пример: 150, 100, 180")
		b.api.Send(msg)
		return
	}

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Анализирую уровень и подбираю программу...")
	b.api.Send(waitMsg)

	gen := getPLGenerator()
	if gen == nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка генератора")
		b.api.Send(msg)
		return
	}

	maxes := ai.AthleteMaxes{
		Squat:    squat,
		Bench:    bench,
		Deadlift: deadlift,
	}

	opts := ai.GenerationOptions{
		LiftType:         ai.LiftTypeFull,
		IncludeAccessory: true,
	}

	program, err := gen.GenerateAutomatic(maxes, opts)
	if err != nil {
		log.Printf("Ошибка авто-генерации: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		b.handlePowerliftingMenu(message)
		return
	}

	// Сохраняем программу
	plStates.Lock()
	plStates.lastProgram[chatID] = program
	plStates.squat[chatID] = squat
	plStates.bench[chatID] = bench
	plStates.deadlift[chatID] = deadlift
	plStates.Unlock()

	// Определяем уровень
	total := squat + bench + deadlift
	var level string
	switch {
	case total < 350:
		level = "Новичок"
	case total < 500:
		level = "Средний"
	case total < 650:
		level = "КМС"
	default:
		level = "МС+"
	}

	validation := gen.ValidateProgram(program)

	statsMsg := fmt.Sprintf("🎯 Программа подобрана автоматически!\n\n"+
		"📊 Ваши данные:\n"+
		"• Присед: %.0f кг\n"+
		"• Жим: %.0f кг\n"+
		"• Тяга: %.0f кг\n"+
		"• Сумма: %.0f кг\n"+
		"• Уровень: %s\n\n"+
		"📋 Выбранная программа: %s\n"+
		"• Недель: %d\n"+
		"• Тренировок: %d\n"+
		"• Общий КПШ: %d\n"+
		"• Тоннаж: %.1f т\n",
		squat, bench, deadlift, total, level,
		program.Name,
		len(program.Weeks),
		validation.Stats.TotalWorkouts,
		program.TotalKPS,
		program.TotalTonnage)

	msg := tgbotapi.NewMessage(chatID, statsMsg)
	b.api.Send(msg)

	// Показываем опции
	b.showPLProgramOptions(chatID)
}

// handlePLListTemplates показывает список всех шаблонов
func (b *Bot) handlePLListTemplates(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	gen := getPLGenerator()
	if gen == nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка генератора")
		b.api.Send(msg)
		return
	}

	templates := gen.ListTemplates()

	var sb strings.Builder
	sb.WriteString("📋 Доступные шаблоны программ:\n\n")

	for i, t := range templates {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t))
	}

	sb.WriteString("\n📖 Описание методик:\n\n")
	sb.WriteString("**Шейко** — много подходов × мало повторов, 80-90% интенсивность\n")
	sb.WriteString("**Головинский** — волнообразная периодизация, чёткие проценты\n")
	sb.WriteString("**Верхошанский** — блочная специализация на одном движении\n")
	sb.WriteString("**Муравьёв** — классическая 4-недельная периодизация\n")
	sb.WriteString("**Русский цикл** — 6 недель агрессивной прогрессии\n")

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)

	b.handlePowerliftingMenu(message)
}

// Вспомогательные функции

func getLiftTypeName(lt ai.LiftType) string {
	switch lt {
	case ai.LiftTypeBench:
		return "Жим лёжа"
	case ai.LiftTypeSquat:
		return "Присед"
	case ai.LiftTypeDeadlift:
		return "Становая тяга"
	case ai.LiftTypeHipThrust:
		return "Ягодичный мост"
	default:
		return "Троеборье"
	}
}

func clearPLState(chatID int64) {
	plStates.Lock()
	delete(plStates.selectedLift, chatID)
	delete(plStates.selectedTempl, chatID)
	delete(plStates.squat, chatID)
	delete(plStates.bench, chatID)
	delete(plStates.deadlift, chatID)
	delete(plStates.hipThrust, chatID)
	delete(plStates.daysPerWeek, chatID)
	delete(plStates.lastProgram, chatID)
	plStates.Unlock()
}

func sendLongMessage(b *Bot, chatID int64, text string) {
	if len(text) <= 4000 {
		msg := tgbotapi.NewMessage(chatID, text)
		b.api.Send(msg)
		return
	}

	// Разбиваем на части
	parts := splitMessage(text, 4000)
	for i, part := range parts {
		partMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("(%d/%d)\n\n%s", i+1, len(parts), part))
		b.api.Send(partMsg)
	}
}

func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var parts []string
	for len(text) > maxLen {
		// Ищем последний перенос строки в пределах maxLen
		splitIdx := strings.LastIndex(text[:maxLen], "\n")
		if splitIdx == -1 {
			splitIdx = maxLen
		}
		parts = append(parts, text[:splitIdx])
		text = text[splitIdx:]
		if len(text) > 0 && text[0] == '\n' {
			text = text[1:]
		}
	}
	if len(text) > 0 {
		parts = append(parts, text)
	}
	return parts
}

func sanitizeFilename(name string) string {
	// Убираем недопустимые символы
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}

// handlePLSendToClient показывает список клиентов для отправки программы
func (b *Bot) handlePLSendToClient(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	plStates.RLock()
	program := plStates.lastProgram[chatID]
	plStates.RUnlock()

	if program == nil {
		b.sendMessage(chatID, "Программа не найдена")
		b.handlePowerliftingMenu(message)
		return
	}

	// Получаем список клиентов
	rows, err := b.db.Query(`
		SELECT c.id, c.name, c.surname, COALESCE(c.telegram_id, 0)
		FROM public.clients c
		LEFT JOIN public.admins a ON c.telegram_id = a.telegram_id
		WHERE a.telegram_id IS NULL AND c.deleted_at IS NULL
		ORDER BY c.name`)
	if err != nil {
		b.sendError(chatID, "Ошибка загрузки клиентов", err)
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

		status := "📵"
		if telegramID > 0 {
			status = "✅"
		}

		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("PLClient: %s %s [%d] %s", name, surname, id, status)),
		))
	}

	if len(buttons) == 0 {
		b.sendMessage(chatID, "Нет клиентов для отправки")
		b.showPLProgramOptions(chatID)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	keyboard := tgbotapi.NewReplyKeyboard(buttons...)
	b.sendMessageWithKeyboard(chatID, fmt.Sprintf(
		"📤 Отправка программы: %s\n\nВыберите клиента:", program.Name), keyboard)

	userStates.Lock()
	userStates.states[chatID] = "pl_select_client"
	userStates.Unlock()
}

// handlePLClientSelection обрабатывает выбор клиента для отправки программы
func (b *Bot) handlePLClientSelection(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.showPLProgramOptions(chatID)
		return
	}

	clientID := parseIDFromBrackets(text)
	if clientID == 0 {
		b.sendMessage(chatID, "Ошибка выбора клиента")
		return
	}

	plStates.RLock()
	program := plStates.lastProgram[chatID]
	plStates.RUnlock()

	if program == nil {
		b.sendMessage(chatID, "Программа не найдена")
		b.handlePowerliftingMenu(message)
		return
	}

	// Получаем данные клиента
	var telegramID int64
	var name, surname string
	err := b.db.QueryRow("SELECT telegram_id, name, surname FROM public.clients WHERE id = $1", clientID).
		Scan(&telegramID, &name, &surname)
	if err != nil {
		b.sendError(chatID, "Клиент не найден", err)
		return
	}

	if telegramID == 0 {
		b.sendMessage(chatID, fmt.Sprintf("❌ У клиента %s %s нет Telegram ID. Программа не может быть отправлена.", name, surname))
		b.showPLProgramOptions(chatID)
		return
	}

	// Форматируем и отправляем программу клиенту
	formatted := ai.FormatPLProgram(program)

	// Отправляем текст
	introMsg := fmt.Sprintf("🏋️ Программа тренировок: %s\n\n"+
		"📊 Параметры:\n"+
		"• Недель: %d\n"+
		"• КПШ: %d\n"+
		"• Тоннаж: %.1f т\n\n"+
		"Полная программа в прикреплённом файле.",
		program.Name, len(program.Weeks), program.TotalKPS, program.TotalTonnage)

	clientMsg := tgbotapi.NewMessage(telegramID, introMsg)
	b.api.Send(clientMsg)

	// Отправляем файл
	doc := tgbotapi.NewDocument(telegramID, tgbotapi.FileBytes{
		Name:  fmt.Sprintf("%s.txt", sanitizeFilename(program.Name)),
		Bytes: []byte(formatted),
	})
	doc.Caption = "Программа тренировок"
	b.api.Send(doc)

	// Подтверждение тренеру
	b.sendMessage(chatID, fmt.Sprintf("✅ Программа отправлена клиенту %s %s", name, surname))

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	b.showPLProgramOptions(chatID)
}

// handlePLExportToGoogle экспортирует программу в Google Sheets
func (b *Bot) handlePLExportToGoogle(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	plStates.RLock()
	program := plStates.lastProgram[chatID]
	plStates.RUnlock()

	if program == nil {
		b.sendMessage(chatID, "Программа не найдена")
		b.handlePowerliftingMenu(message)
		return
	}

	if b.sheetsClient == nil {
		b.sendMessage(chatID, "❌ Google Sheets не настроен")
		b.showPLProgramOptions(chatID)
		return
	}

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Создаю таблицу в Google Sheets...")
	b.api.Send(waitMsg)

	// Конвертируем в формат gsheets
	plData := convertToPLProgramData(program)

	// Создаём таблицу
	spreadsheetID, err := b.sheetsClient.CreatePLProgramSpreadsheet(plData)
	if err != nil {
		b.sendError(chatID, "Ошибка создания таблицы", err)
		b.showPLProgramOptions(chatID)
		return
	}

	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", spreadsheetID)

	b.sendMessage(chatID, fmt.Sprintf("✅ Таблица создана!\n\n📊 %s\n\n🔗 %s", program.Name, url))
	b.showPLProgramOptions(chatID)
}

// convertToPLProgramData конвертирует ai.PLGeneratedProgram в gsheets.PLProgramData
func convertToPLProgramData(program *ai.PLGeneratedProgram) *gsheets.PLProgramData {
	data := &gsheets.PLProgramData{
		Name:         program.Name,
		TotalKPS:     program.TotalKPS,
		TotalTonnage: program.TotalTonnage,
	}
	data.AthleteMaxes.Squat = program.AthleteMaxes.Squat
	data.AthleteMaxes.Bench = program.AthleteMaxes.Bench
	data.AthleteMaxes.Deadlift = program.AthleteMaxes.Deadlift
	data.AthleteMaxes.HipThrust = program.AthleteMaxes.HipThrust

	for _, week := range program.Weeks {
		weekData := gsheets.PLWeekData{
			WeekNum:  week.WeekNum,
			Phase:    week.Phase,
			TotalKPS: week.TotalKPS,
			Tonnage:  week.Tonnage,
		}

		for _, workout := range week.Workouts {
			workoutData := gsheets.PLWorkoutData{
				DayNum:   workout.DayNum,
				Name:     workout.Name,
				TotalKPS: workout.TotalKPS,
				Tonnage:  workout.Tonnage,
			}

			for _, ex := range workout.Exercises {
				exData := gsheets.PLExerciseData{
					Name:       ex.Name,
					Type:       ex.Type,
					TotalReps:  ex.TotalReps,
					Tonnage:    ex.Tonnage,
					AvgPercent: ex.AvgPercent,
				}

				for _, set := range ex.Sets {
					exData.Sets = append(exData.Sets, gsheets.PLSetData{
						Percent:  set.Percent,
						Reps:     set.Reps,
						Sets:     set.Sets,
						WeightKg: set.WeightKg,
					})
				}

				workoutData.Exercises = append(workoutData.Exercises, exData)
			}

			weekData.Workouts = append(weekData.Workouts, workoutData)
		}

		data.Weeks = append(data.Weeks, weekData)
	}

	return data
}

// handlePLProgramForClient переходит к PL программам с уже выбранным клиентом
func (b *Bot) handlePLProgramForClient(message *tgbotapi.Message, clientID int) {
	chatID := message.Chat.ID

	// Сохраняем clientID в adminStates (уже сохранён) и показываем меню PL
	b.handlePowerliftingMenu(message)

	// Устанавливаем состояние для возврата к клиенту
	userStates.Lock()
	userStates.states[chatID] = "pl_from_client"
	userStates.Unlock()
}

package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"workbot/clients/ai"
	"workbot/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// aiStates хранит состояния AI для админов
var aiStates = struct {
	sync.RWMutex
	trainer          map[int64]*ai.TrainerAI
	selectedClientAI map[int64]int
	clientProfile    map[int64]ai.ClientProfile
	lastGenerated    map[int64]string // последняя сгенерированная тренировка
}{
	trainer:          make(map[int64]*ai.TrainerAI),
	selectedClientAI: make(map[int64]int),
	clientProfile:    make(map[int64]ai.ClientProfile),
	lastGenerated:    make(map[int64]string),
}

// progressionParams хранит параметры плана с прогрессией
var progressionParams = struct {
	sync.RWMutex
	weeks       map[int64]int
	daysPerWeek map[int64]int
	goal        map[int64]string
}{
	weeks:       make(map[int64]int),
	daysPerWeek: make(map[int64]int),
	goal:        make(map[int64]string),
}

// competitionParams хранит параметры для подготовки к соревнованиям
var competitionParams = struct {
	sync.RWMutex
	sport         map[int64]string
	weeksLeft     map[int64]int
	currentWeight map[int64]float64
	targetWeight  map[int64]float64
}{
	sport:         make(map[int64]string),
	weeksLeft:     make(map[int64]int),
	currentWeight: make(map[int64]float64),
	targetWeight:  make(map[int64]float64),
}

// getTrainerAI возвращает или создаёт AI-ассистента для тренера
func getTrainerAI(chatID int64) *ai.TrainerAI {
	aiStates.RLock()
	trainer := aiStates.trainer[chatID]
	aiStates.RUnlock()

	if trainer != nil {
		return trainer
	}

	// Создаём нового тренера
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Ошибка загрузки конфига: %v", err)
	}

	ollamaURL := "http://localhost:11434"
	ollamaModel := "gemma2:9b-instruct-q4_K_M"
	if cfg != nil {
		ollamaURL = cfg.OllamaURL
		ollamaModel = cfg.OllamaModel
	}

	aiStates.Lock()
	if aiStates.trainer[chatID] == nil {
		aiStates.trainer[chatID] = ai.NewTrainerAI(ollamaURL, ollamaModel)
	}
	trainer = aiStates.trainer[chatID]
	aiStates.Unlock()

	return trainer
}

// handleAIMenu показывает меню AI
func (b *Bot) handleAIMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	msg := tgbotapi.NewMessage(chatID, "🤖 AI-Ассистент тренера\n\nВыберите действие:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("AI: Сгенерировать тренировку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("AI: План с прогрессией"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("AI: Методики"),
			tgbotapi.NewKeyboardButton("AI: К соревнованиям"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("AI: План на неделю"),
			tgbotapi.NewKeyboardButton("AI: Годовой план"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("AI: Задать вопрос"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAIGenerateTraining запускает генерацию тренировки
func (b *Bot) handleAIGenerateTraining(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Показываем список клиентов (исключая админов и удалённых)
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
			tgbotapi.NewKeyboardButton(fmt.Sprintf("AI>> %s %s [%d]", name, surname, id)),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет клиентов. Сначала добавьте клиента.")
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите клиента для генерации тренировки:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// handleAIClientSelection обрабатывает выбор клиента для AI
func (b *Bot) handleAIClientSelection(message *tgbotapi.Message, text string) {
	chatID := message.Chat.ID

	// Парсим ID клиента
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора клиента")
		b.api.Send(msg)
		return
	}

	clientID, err := strconv.Atoi(text[start+1 : end])
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора клиента")
		b.api.Send(msg)
		return
	}

	// Получаем данные клиента
	var name, surname string
	err = b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).Scan(&name, &surname)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Клиент не найден")
		b.api.Send(msg)
		return
	}

	// Сохраняем выбранного клиента
	aiStates.selectedClientAI[chatID] = clientID
	aiStates.clientProfile[chatID] = ai.ClientProfile{
		Name:    name,
		Surname: surname,
	}

	userStates.Lock()
	userStates.states[chatID] = "ai_awaiting_params"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Клиент: %s %s\n\n"+
			"Выберите тип тренировки:",
		name, surname))

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Силовая"),
			tgbotapi.NewKeyboardButton("Гипертрофия"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Функциональная"),
			tgbotapi.NewKeyboardButton("Кардио"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAITrainingType обрабатывает выбор типа тренировки
func (b *Bot) handleAITrainingType(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	trainingType := message.Text

	userStates.Lock()
	userStates.states[chatID] = "ai_awaiting_direction"
	userStates.Unlock()

	// Сохраняем тип в профиле (временно в Goal)
	profile := aiStates.clientProfile[chatID]
	profile.Goal = trainingType
	aiStates.clientProfile[chatID] = profile

	msg := tgbotapi.NewMessage(chatID, "Выберите направленность:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Верх (Push)"),
			tgbotapi.NewKeyboardButton("Верх (Pull)"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Низ"),
			tgbotapi.NewKeyboardButton("Фуллбоди"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Core"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAIDirection обрабатывает выбор направленности и генерирует тренировку
func (b *Bot) handleAIDirection(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	direction := message.Text

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	profile := aiStates.clientProfile[chatID]
	trainingType := profile.Goal // временно хранили тип тут

	// Восстанавливаем Goal
	profile.Goal = ""
	aiStates.clientProfile[chatID] = profile

	// Получаем имя клиента из БД если профиль пустой
	clientID := aiStates.selectedClientAI[chatID]
	if profile.Name == "" && clientID > 0 {
		b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).
			Scan(&profile.Name, &profile.Surname)
		aiStates.clientProfile[chatID] = profile
	}

	// Отправляем сообщение о генерации
	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Генерирую тренировку...")
	b.api.Send(waitMsg)

	// Генерируем тренировку
	trainer := getTrainerAI(chatID)
	params := ai.TrainingParams{
		Type:      trainingType,
		Direction: direction,
		Duration:  45,
		Equipment: "зал",
	}

	response, err := trainer.GenerateTraining(profile, params)
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка генерации: %v\n\nПроверьте API ключ Groq.", err))
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	// Сохраняем результат
	aiStates.lastGenerated[chatID] = response

	// Отправляем результат
	clientName := fmt.Sprintf("%s %s", profile.Name, profile.Surname)
	if clientName == " " {
		clientName = "клиента"
	}
	resultMsg := fmt.Sprintf("🏋️ Сгенерированная тренировка для %s:\n\n%s", clientName, response)

	msg := tgbotapi.NewMessage(chatID, resultMsg)
	b.api.Send(msg)

	// Показываем опции
	optionsMsg := tgbotapi.NewMessage(chatID, "Что сделать с тренировкой?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Утвердить и сохранить"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Изменить тренировку"),
			tgbotapi.NewKeyboardButton("Сгенерировать заново"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	optionsMsg.ReplyMarkup = keyboard

	userStates.Lock()
	userStates.states[chatID] = "ai_review"
	userStates.Unlock()

	b.api.Send(optionsMsg)
}

// handleAIReview обрабатывает решение по сгенерированной тренировке
func (b *Bot) handleAIReview(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	switch text {
	case "Утвердить и сохранить":
		b.handleAIApprove(message)
	case "Изменить тренировку":
		b.handleAIModify(message)
	case "Сгенерировать заново":
		// Возвращаемся к выбору типа
		userStates.Lock()
		userStates.states[chatID] = "ai_awaiting_params"
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Выберите тип тренировки:")
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Силовая"),
				tgbotapi.NewKeyboardButton("Гипертрофия"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Функциональная"),
				tgbotapi.NewKeyboardButton("Кардио"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	case "Отмена":
		b.handleAdminCancel(message)
	}
}

// handleAIApprove утверждает тренировку и предлагает отправить клиенту
func (b *Bot) handleAIApprove(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	profile := aiStates.clientProfile[chatID]
	clientID := aiStates.selectedClientAI[chatID]

	userStates.Lock()
	userStates.states[chatID] = "ai_send_to_client"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"✅ Тренировка готова для %s %s!\n\n"+
			"Отправить тренировку клиенту?",
		profile.Name, profile.Surname))

	// Проверяем, есть ли у клиента telegram_id
	var telegramID int64
	b.db.QueryRow("SELECT COALESCE(telegram_id, 0) FROM public.clients WHERE id = $1", clientID).Scan(&telegramID)

	if telegramID > 0 {
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отправить клиенту"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Только сохранить"),
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		msg.ReplyMarkup = keyboard
	} else {
		msg.Text += "\n\n(У клиента нет Telegram, тренировка будет только сохранена)"
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Сохранить"),
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		msg.ReplyMarkup = keyboard
	}
	b.api.Send(msg)
}

// handleAISendToClient обрабатывает решение об отправке тренировки клиенту
func (b *Bot) handleAISendToClient(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	profile := aiStates.clientProfile[chatID]
	generated := aiStates.lastGenerated[chatID]
	clientID := aiStates.selectedClientAI[chatID]

	// Очищаем состояние
	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()
	delete(aiStates.lastGenerated, chatID)
	delete(aiStates.selectedClientAI, chatID)
	delete(aiStates.clientProfile, chatID)

	shouldSend := text == "Отправить клиенту"

	if shouldSend {
		// Получаем telegram_id клиента
		var telegramID int64
		b.db.QueryRow("SELECT telegram_id FROM public.clients WHERE id = $1", clientID).Scan(&telegramID)

		if telegramID > 0 {
			// Форматируем сообщение для клиента
			clientMsg := fmt.Sprintf("🏋️ Тренировка от тренера:\n\n%s\n\nУдачной тренировки!", generated)
			notification := tgbotapi.NewMessage(telegramID, clientMsg)
			if _, err := b.api.Send(notification); err != nil {
				log.Printf("Ошибка отправки клиенту %d: %v", clientID, err)
				msg := tgbotapi.NewMessage(chatID, "Ошибка отправки клиенту, но тренировка сохранена.")
				b.api.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
					"✅ Тренировка отправлена клиенту %s %s!",
					profile.Name, profile.Surname))
				b.api.Send(msg)
			}
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"✅ Тренировка сохранена для %s %s.\n\n"+
				"Тренировка:\n%s",
			profile.Name, profile.Surname, generated))
		b.api.Send(msg)
	}

	b.handleAIMenu(message)
}

// handleAIModify запускает модификацию тренировки
func (b *Bot) handleAIModify(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = "ai_modify"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID,
		"Введите инструкции по изменению тренировки.\n\n"+
			"Примеры:\n"+
			"• Убери становую тягу, добавь румынскую\n"+
			"• Сделай больше упражнений на бицепс\n"+
			"• Уменьши количество подходов до 3\n"+
			"• Добавь разминку на 10 минут")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAIModifyInput обрабатывает ввод инструкций по модификации
func (b *Bot) handleAIModifyInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	instructions := message.Text

	if instructions == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Изменяю тренировку...")
	b.api.Send(waitMsg)

	original := aiStates.lastGenerated[chatID]
	trainer := getTrainerAI(chatID)

	modified, err := trainer.ModifyTraining(original, instructions)
	if err != nil {
		log.Printf("Ошибка модификации: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		return
	}

	// Сохраняем новый результат
	aiStates.lastGenerated[chatID] = modified

	profile := aiStates.clientProfile[chatID]
	resultMsg := fmt.Sprintf("🏋️ Изменённая тренировка для %s %s:\n\n%s",
		profile.Name, profile.Surname, modified)

	msg := tgbotapi.NewMessage(chatID, resultMsg)
	b.api.Send(msg)

	// Показываем опции снова
	optionsMsg := tgbotapi.NewMessage(chatID, "Что сделать с тренировкой?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Утвердить и сохранить"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Изменить тренировку"),
			tgbotapi.NewKeyboardButton("Сгенерировать заново"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	optionsMsg.ReplyMarkup = keyboard

	userStates.Lock()
	userStates.states[chatID] = "ai_review"
	userStates.Unlock()

	b.api.Send(optionsMsg)
}

// handleAIWeekPlan генерирует план на неделю
func (b *Bot) handleAIWeekPlan(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = "ai_week_plan_client"
	userStates.Unlock()

	// Показываем список клиентов
	b.showClientsForAI(chatID, "Выберите клиента для недельного плана:")
}

// handleAIYearPlan генерирует годовой план
func (b *Bot) handleAIYearPlan(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = "ai_year_plan_client"
	userStates.Unlock()

	b.showClientsForAI(chatID, "Выберите клиента для годового плана:")
}

// showClientsForAI показывает список клиентов для AI функций
func (b *Bot) showClientsForAI(chatID int64, title string) {
	// Исключаем админов и удалённых клиентов из списка
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
			tgbotapi.NewKeyboardButton(fmt.Sprintf("AI>> %s %s [%d]", name, surname, id)),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет клиентов")
		b.api.Send(msg)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	msg := tgbotapi.NewMessage(chatID, title)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// handleAIQuestion обрабатывает вопрос к AI
func (b *Bot) handleAIQuestion(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = "ai_question"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID,
		"Задайте вопрос AI-ассистенту.\n\n"+
			"Примеры:\n"+
			"• Как правильно выполнять становую тягу?\n"+
			"• Сколько белка нужно для набора массы?\n"+
			"• Как восстановиться после травмы колена?\n"+
			"• Какие упражнения лучше для осанки?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAIQuestionInput обрабатывает ввод вопроса
func (b *Bot) handleAIQuestionInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	question := message.Text

	if question == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Думаю над ответом...")
	b.api.Send(waitMsg)

	trainer := getTrainerAI(chatID)
	answer, err := trainer.AskQuestion(question)
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "💡 Ответ:\n\n"+answer)
	b.api.Send(msg)

	b.handleAIMenu(message)
}

// handleAIWeekPlanGenerate генерирует недельный план после выбора клиента
func (b *Bot) handleAIWeekPlanGenerate(message *tgbotapi.Message, clientID int) {
	chatID := message.Chat.ID

	var name, surname string
	err := b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).Scan(&name, &surname)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Клиент не найден")
		b.api.Send(msg)
		return
	}

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Генерирую план на неделю...")
	b.api.Send(waitMsg)

	profile := ai.ClientProfile{Name: name, Surname: surname}
	trainer := getTrainerAI(chatID)

	plan, err := trainer.GenerateWeekPlan(profile, 3, "общая физическая подготовка")
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📅 Недельный план для %s %s:\n\n%s", name, surname, plan))
	b.api.Send(msg)

	b.handleAIMenu(message)
}

// handleAIYearPlanGenerate генерирует годовой план
func (b *Bot) handleAIYearPlanGenerate(message *tgbotapi.Message, clientID int) {
	chatID := message.Chat.ID

	var name, surname string
	err := b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).Scan(&name, &surname)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Клиент не найден")
		b.api.Send(msg)
		return
	}

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Генерирую годовой план... (это может занять минуту)")
	b.api.Send(waitMsg)

	profile := ai.ClientProfile{Name: name, Surname: surname}
	trainer := getTrainerAI(chatID)

	plan, err := trainer.GenerateYearPlan(profile, "общая физическая подготовка и здоровье")
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	// Годовой план может быть длинным, разбиваем на части
	if len(plan) > 4000 {
		parts := splitMessage(plan, 4000)
		for i, part := range parts {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📆 Годовой план для %s %s (часть %d/%d):\n\n%s",
				name, surname, i+1, len(parts), part))
			b.api.Send(msg)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📆 Годовой план для %s %s:\n\n%s", name, surname, plan))
		b.api.Send(msg)
	}

	b.handleAIMenu(message)
}

// splitMessage разбивает длинное сообщение на части
func splitMessage(text string, maxLen int) []string {
	var parts []string
	for len(text) > maxLen {
		// Ищем последний перенос строки в пределах лимита
		cutIndex := maxLen
		for i := maxLen - 1; i > maxLen/2; i-- {
			if text[i] == '\n' {
				cutIndex = i
				break
			}
		}
		parts = append(parts, text[:cutIndex])
		text = text[cutIndex:]
	}
	if len(text) > 0 {
		parts = append(parts, text)
	}
	return parts
}

// handleAIProgressionPlan запускает создание плана с прогрессией
func (b *Bot) handleAIProgressionPlan(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = "ai_progression_client"
	userStates.Unlock()

	b.showClientsForAI(chatID, "Выберите клиента для плана с прогрессией:")
}

// handleAIProgressionWeeks обрабатывает выбор клиента и спрашивает количество недель
func (b *Bot) handleAIProgressionWeeks(message *tgbotapi.Message, clientID int) {
	chatID := message.Chat.ID

	var name, surname string
	err := b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).Scan(&name, &surname)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Клиент не найден")
		b.api.Send(msg)
		return
	}

	aiStates.selectedClientAI[chatID] = clientID
	aiStates.clientProfile[chatID] = ai.ClientProfile{Name: name, Surname: surname}

	userStates.Lock()
	userStates.states[chatID] = "ai_progression_weeks"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Клиент: %s %s\n\n"+
			"На сколько недель составить план с прогрессией?",
		name, surname))

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4 недели"),
			tgbotapi.NewKeyboardButton("6 недель"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("8 недель"),
			tgbotapi.NewKeyboardButton("12 недель"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAIProgressionWeeksInput обрабатывает выбор количества недель
func (b *Bot) handleAIProgressionWeeksInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	var weeks int
	switch text {
	case "4 недели":
		weeks = 4
	case "6 недель":
		weeks = 6
	case "8 недель":
		weeks = 8
	case "12 недель":
		weeks = 12
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите вариант из кнопок")
		b.api.Send(msg)
		return
	}

	progressionParams.weeks[chatID] = weeks

	userStates.Lock()
	userStates.states[chatID] = "ai_progression_days"
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

// handleAIProgressionDaysInput обрабатывает выбор количества дней
func (b *Bot) handleAIProgressionDaysInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
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
		msg := tgbotapi.NewMessage(chatID, "Выберите вариант из кнопок")
		b.api.Send(msg)
		return
	}

	progressionParams.daysPerWeek[chatID] = days

	userStates.Lock()
	userStates.states[chatID] = "ai_progression_goal"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Какая цель тренировок?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Набор массы"),
			tgbotapi.NewKeyboardButton("Сила"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Похудение"),
			tgbotapi.NewKeyboardButton("Рельеф"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Калистеника"),
			tgbotapi.NewKeyboardButton("Общая подготовка"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAIProgressionGoalInput обрабатывает выбор цели и генерирует план
func (b *Bot) handleAIProgressionGoalInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	var goal string
	switch text {
	case "Набор массы":
		goal = "набор мышечной массы (гипертрофия)"
	case "Сила":
		goal = "развитие силы"
	case "Похудение":
		goal = "похудение и жиросжигание"
	case "Рельеф":
		goal = "рельеф и сушка"
	case "Калистеника":
		goal = "калистеника (работа с собственным весом)"
	case "Общая подготовка":
		goal = "общая физическая подготовка"
	default:
		goal = text
	}

	progressionParams.goal[chatID] = goal

	// Генерируем план
	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	profile := aiStates.clientProfile[chatID]
	weeks := progressionParams.weeks[chatID]
	days := progressionParams.daysPerWeek[chatID]

	// Очищаем временные данные
	delete(progressionParams.weeks, chatID)
	delete(progressionParams.daysPerWeek, chatID)
	delete(progressionParams.goal, chatID)

	waitMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"⏳ Генерирую план на %d недель с прогрессией...\n"+
			"(это может занять 1-2 минуты)", weeks))
	b.api.Send(waitMsg)

	trainer := getTrainerAI(chatID)
	params := ai.ProgressionParams{
		Weeks:       weeks,
		DaysPerWeek: days,
		Goal:        goal,
	}

	plan, err := trainer.GenerateProgressionPlan(profile, params)
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	// Сохраняем результат
	aiStates.lastGenerated[chatID] = plan

	// План может быть длинным, разбиваем на части
	header := fmt.Sprintf("📈 План с прогрессией для %s %s\n"+
		"Длительность: %d недель\n"+
		"Тренировок в неделю: %d\n"+
		"Цель: %s\n\n",
		profile.Name, profile.Surname, weeks, days, goal)

	if len(header+plan) > 4000 {
		msg := tgbotapi.NewMessage(chatID, header)
		b.api.Send(msg)

		parts := splitMessage(plan, 4000)
		for i, part := range parts {
			partMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Часть %d/%d:\n\n%s", i+1, len(parts), part))
			b.api.Send(partMsg)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, header+plan)
		b.api.Send(msg)
	}

	// Показываем опции
	optionsMsg := tgbotapi.NewMessage(chatID, "Что сделать с планом?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отправить клиенту"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сгенерировать заново"),
			tgbotapi.NewKeyboardButton("В меню AI"),
		),
	)
	optionsMsg.ReplyMarkup = keyboard

	userStates.Lock()
	userStates.states[chatID] = "ai_progression_review"
	userStates.Unlock()

	b.api.Send(optionsMsg)
}

// handleAIProgressionReview обрабатывает решение по плану с прогрессией
func (b *Bot) handleAIProgressionReview(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	switch text {
	case "Отправить клиенту":
		clientID := aiStates.selectedClientAI[chatID]
		profile := aiStates.clientProfile[chatID]
		plan := aiStates.lastGenerated[chatID]

		var telegramID int64
		b.db.QueryRow("SELECT COALESCE(telegram_id, 0) FROM public.clients WHERE id = $1", clientID).Scan(&telegramID)

		if telegramID > 0 {
			clientMsg := fmt.Sprintf("📈 Ваш персональный план тренировок с прогрессией:\n\n%s\n\nУдачных тренировок!", plan)

			if len(clientMsg) > 4000 {
				headerMsg := tgbotapi.NewMessage(telegramID, "📈 Ваш персональный план тренировок с прогрессией:")
				b.api.Send(headerMsg)

				parts := splitMessage(plan, 4000)
				for _, part := range parts {
					partMsg := tgbotapi.NewMessage(telegramID, part)
					b.api.Send(partMsg)
				}

				footerMsg := tgbotapi.NewMessage(telegramID, "Удачных тренировок!")
				b.api.Send(footerMsg)
			} else {
				notification := tgbotapi.NewMessage(telegramID, clientMsg)
				b.api.Send(notification)
			}

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ План отправлен клиенту %s %s!", profile.Name, profile.Surname))
			b.api.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(chatID, "У клиента нет Telegram, план не отправлен.")
			b.api.Send(msg)
		}

		// Очищаем состояние
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()
		delete(aiStates.lastGenerated, chatID)
		delete(aiStates.selectedClientAI, chatID)
		delete(aiStates.clientProfile, chatID)

		b.handleAIMenu(message)

	case "Сгенерировать заново":
		// Возвращаемся к выбору недель
		userStates.Lock()
		userStates.states[chatID] = "ai_progression_weeks"
		userStates.Unlock()

		profile := aiStates.clientProfile[chatID]
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Клиент: %s %s\n\n"+
				"На сколько недель составить план с прогрессией?",
			profile.Name, profile.Surname))

		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("4 недели"),
				tgbotapi.NewKeyboardButton("6 недель"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("8 недель"),
				tgbotapi.NewKeyboardButton("12 недель"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)

	case "В меню AI":
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()
		delete(aiStates.lastGenerated, chatID)
		delete(aiStates.selectedClientAI, chatID)
		delete(aiStates.clientProfile, chatID)

		b.handleAIMenu(message)
	}
}

// handleAIMethodologies показывает меню методик тренировок
func (b *Bot) handleAIMethodologies(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = "ai_methodology_select"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "📚 Методики тренировок\n\nВыберите методику или введите название:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("5x5 StrongLifts"),
			tgbotapi.NewKeyboardButton("Wendler 5/3/1"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Push/Pull/Legs"),
			tgbotapi.NewKeyboardButton("Upper/Lower"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Немецкий объём 10x10"),
			tgbotapi.NewKeyboardButton("DUP периодизация"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Conjugate (Westside)"),
			tgbotapi.NewKeyboardButton("RPE тренинг"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сравнить методики"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAIMethodologySelect обрабатывает выбор методики
func (b *Bot) handleAIMethodologySelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	if text == "Сравнить методики" {
		userStates.Lock()
		userStates.states[chatID] = "ai_methodology_compare"
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите названия методик для сравнения через запятую:\n\n"+
			"Например: 5x5, Wendler 5/3/1, Push/Pull/Legs")
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
		return
	}

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	waitMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⏳ Получаю информацию о методике: %s...", text))
	b.api.Send(waitMsg)

	trainer := getTrainerAI(chatID)
	info, err := trainer.GetMethodology(text)
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	// Разбиваем длинный ответ на части
	if len(info) > 4000 {
		parts := splitMessage(info, 4000)
		for i, part := range parts {
			partMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📚 %s (часть %d/%d):\n\n%s", text, i+1, len(parts), part))
			b.api.Send(partMsg)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📚 Методика: %s\n\n%s", text, info))
		b.api.Send(msg)
	}

	b.handleAIMethodologies(message)
}

// handleAIMethodologyCompare сравнивает методики
func (b *Bot) handleAIMethodologyCompare(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	userStates.Lock()
	userStates.states[chatID] = "ai_methodology_compare_goal"
	userStates.Unlock()

	// Сохраняем методики для сравнения
	aiStates.lastGenerated[chatID] = text

	msg := tgbotapi.NewMessage(chatID, "Для какой цели сравниваем?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Набор массы"),
			tgbotapi.NewKeyboardButton("Сила"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Похудение"),
			tgbotapi.NewKeyboardButton("Общая подготовка"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAIMethodologyCompareGoal выполняет сравнение методик
func (b *Bot) handleAIMethodologyCompareGoal(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	goal := message.Text

	if goal == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	methodsStr := aiStates.lastGenerated[chatID]
	methods := strings.Split(methodsStr, ",")
	for i := range methods {
		methods[i] = strings.TrimSpace(methods[i])
	}

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()
	delete(aiStates.lastGenerated, chatID)

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Сравниваю методики...")
	b.api.Send(waitMsg)

	trainer := getTrainerAI(chatID)
	comparison, err := trainer.CompareMethodologies(methods, goal)
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	if len(comparison) > 4000 {
		parts := splitMessage(comparison, 4000)
		for i, part := range parts {
			partMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📊 Сравнение (часть %d/%d):\n\n%s", i+1, len(parts), part))
			b.api.Send(partMsg)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, "📊 Сравнение методик:\n\n"+comparison)
		b.api.Send(msg)
	}

	b.handleAIMenu(message)
}

// handleAICompetition запускает подготовку плана к соревнованиям
func (b *Bot) handleAICompetition(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = "ai_competition_client"
	userStates.Unlock()

	b.showClientsForAI(chatID, "Выберите клиента для плана к соревнованиям:")
}

// handleAICompetitionSport выбор вида спорта
func (b *Bot) handleAICompetitionSport(message *tgbotapi.Message, clientID int) {
	chatID := message.Chat.ID

	var name, surname string
	err := b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).Scan(&name, &surname)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Клиент не найден")
		b.api.Send(msg)
		return
	}

	aiStates.selectedClientAI[chatID] = clientID
	aiStates.clientProfile[chatID] = ai.ClientProfile{Name: name, Surname: surname}

	userStates.Lock()
	userStates.states[chatID] = "ai_competition_sport"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Клиент: %s %s\n\n"+
			"Выберите вид соревнований:",
		name, surname))

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Пауэрлифтинг"),
			tgbotapi.NewKeyboardButton("Hyrox"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Гонки с препятствиями"),
			tgbotapi.NewKeyboardButton("Фитнес-бикини"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Жим лёжа"),
			tgbotapi.NewKeyboardButton("Становая тяга"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Ягодичный мост"),
			tgbotapi.NewKeyboardButton("Подъём на бицепс"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAICompetitionSportInput обрабатывает выбор вида спорта
func (b *Bot) handleAICompetitionSportInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	competitionParams.sport[chatID] = text

	userStates.Lock()
	userStates.states[chatID] = "ai_competition_weeks"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Сколько недель до соревнований?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4 недели"),
			tgbotapi.NewKeyboardButton("8 недель"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("12 недель"),
			tgbotapi.NewKeyboardButton("16 недель"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("20 недель"),
			tgbotapi.NewKeyboardButton("24 недели"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAICompetitionWeeksInput обрабатывает выбор недель
func (b *Bot) handleAICompetitionWeeksInput(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
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
	case "16 недель":
		weeks = 16
	case "20 недель":
		weeks = 20
	case "24 недели":
		weeks = 24
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите вариант из кнопок")
		b.api.Send(msg)
		return
	}

	competitionParams.weeksLeft[chatID] = weeks

	// Генерируем план
	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	profile := aiStates.clientProfile[chatID]
	sport := competitionParams.sport[chatID]

	// Очищаем временные данные
	delete(competitionParams.sport, chatID)
	delete(competitionParams.weeksLeft, chatID)

	waitMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"⏳ Генерирую план подготовки к соревнованиям по %s на %d недель...",
		sport, weeks))
	b.api.Send(waitMsg)

	trainer := getTrainerAI(chatID)
	params := ai.CompetitionParams{
		Sport:        sport,
		WeeksLeft:    weeks,
		CurrentLevel: "средний",
	}

	plan, err := trainer.GenerateCompetitionPlan(profile, params)
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		b.api.Send(msg)
		b.handleAIMenu(message)
		return
	}

	// Сохраняем результат
	aiStates.lastGenerated[chatID] = plan

	header := fmt.Sprintf("🏆 План подготовки к соревнованиям\n"+
		"Клиент: %s %s\n"+
		"Вид спорта: %s\n"+
		"До старта: %d недель\n\n",
		profile.Name, profile.Surname, sport, weeks)

	if len(header+plan) > 4000 {
		msg := tgbotapi.NewMessage(chatID, header)
		b.api.Send(msg)

		parts := splitMessage(plan, 4000)
		for i, part := range parts {
			partMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Часть %d/%d:\n\n%s", i+1, len(parts), part))
			b.api.Send(partMsg)
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, header+plan)
		b.api.Send(msg)
	}

	// Показываем опции
	optionsMsg := tgbotapi.NewMessage(chatID, "Что сделать с планом?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отправить клиенту"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("В меню AI"),
		),
	)
	optionsMsg.ReplyMarkup = keyboard

	userStates.Lock()
	userStates.states[chatID] = "ai_competition_review"
	userStates.Unlock()

	b.api.Send(optionsMsg)
}

// handleAICompetitionReview обрабатывает решение по плану к соревнованиям
func (b *Bot) handleAICompetitionReview(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	switch text {
	case "Отправить клиенту":
		clientID := aiStates.selectedClientAI[chatID]
		profile := aiStates.clientProfile[chatID]
		plan := aiStates.lastGenerated[chatID]

		var telegramID int64
		b.db.QueryRow("SELECT COALESCE(telegram_id, 0) FROM public.clients WHERE id = $1", clientID).Scan(&telegramID)

		if telegramID > 0 {
			clientMsg := fmt.Sprintf("🏆 Ваш план подготовки к соревнованиям:\n\n%s\n\nУспешной подготовки!", plan)

			if len(clientMsg) > 4000 {
				headerMsg := tgbotapi.NewMessage(telegramID, "🏆 Ваш план подготовки к соревнованиям:")
				b.api.Send(headerMsg)

				parts := splitMessage(plan, 4000)
				for _, part := range parts {
					partMsg := tgbotapi.NewMessage(telegramID, part)
					b.api.Send(partMsg)
				}

				footerMsg := tgbotapi.NewMessage(telegramID, "Успешной подготовки!")
				b.api.Send(footerMsg)
			} else {
				notification := tgbotapi.NewMessage(telegramID, clientMsg)
				b.api.Send(notification)
			}

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ План отправлен клиенту %s %s!", profile.Name, profile.Surname))
			b.api.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(chatID, "У клиента нет Telegram, план не отправлен.")
			b.api.Send(msg)
		}

		fallthrough

	case "В меню AI":
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()
		delete(aiStates.lastGenerated, chatID)
		delete(aiStates.selectedClientAI, chatID)
		delete(aiStates.clientProfile, chatID)

		b.handleAIMenu(message)
	}
}

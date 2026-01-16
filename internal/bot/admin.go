package bot

import (
	"log"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// aiClientStore хранит выбранного клиента для AI операций
var aiClientStore = struct {
	sync.RWMutex
	data map[int64]int
}{data: make(map[int64]int)}

// adminStates хранит состояния админов
var adminStates = struct {
	sync.RWMutex
	selectedClient map[int64]int // выбранный клиент для админа
}{
	selectedClient: make(map[int64]int),
}

// adminCache кэширует проверку админов для оптимизации
var adminCache = struct {
	sync.RWMutex
	cache map[int64]bool
}{
	cache: make(map[int64]bool),
}

// isAdmin проверяет, является ли пользователь админом (с кэшированием)
func (b *Bot) isAdmin(telegramID int64) bool {
	// Проверяем кэш
	adminCache.RLock()
	if cached, ok := adminCache.cache[telegramID]; ok {
		adminCache.RUnlock()
		return cached
	}
	adminCache.RUnlock()

	// Запрос к БД
	var exists bool
	err := b.db.QueryRow("SELECT EXISTS(SELECT 1 FROM public.admins WHERE telegram_id = $1)", telegramID).Scan(&exists)
	if err != nil {
		log.Printf("Ошибка проверки админа: %v", err)
		return false
	}

	// Кэшируем результат
	adminCache.Lock()
	adminCache.cache[telegramID] = exists
	adminCache.Unlock()

	return exists
}

// handleAdminCommand обрабатывает команды от админа
func (b *Bot) handleAdminCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.handleAdminStart(message)
	case "info":
		b.handleInfoCommand(message)
	default:
		b.sendMessage(message.Chat.ID, "Неизвестная команда")
	}
}

// handleAdminStart показывает админское меню
func (b *Bot) handleAdminStart(message *tgbotapi.Message) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Клиенты"),
			tgbotapi.NewKeyboardButton("Добавить клиента"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Создать программу"),
			tgbotapi.NewKeyboardButton("Отправить тренировку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Расписание"),
			tgbotapi.NewKeyboardButton("1ПМ Тестирование"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("AI Ассистент"),
			tgbotapi.NewKeyboardButton("Тренеры"),
		),
	)
	b.sendMessageWithKeyboard(message.Chat.ID, "Панель тренера", keyboard)
}

// handleAdminMessage обрабатывает сообщения от админа
func (b *Bot) handleAdminMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	userStates.RLock()
	state := userStates.states[chatID]
	userStates.RUnlock()

	// Обработка состояний добавления клиента
	if strings.HasPrefix(state, "add_client_") {
		b.processAddClient(message, state)
		return
	}

	// Обработка состояний добавления тренера
	if strings.HasPrefix(state, "add_trainer_") {
		b.processAddTrainer(message, state)
		return
	}

	// Обработка удаления тренера
	if state == "remove_trainer_select" {
		b.handleRemoveTrainerSelection(message)
		return
	}

	// Обработка состояний расписания
	if strings.HasPrefix(state, "schedule_") {
		if state == stateScheduleDeleteSelect {
			b.handleDeleteSlotSelection(message)
		} else {
			b.processSchedule(message, state)
		}
		return
	}

	// Обработка состояний управления записями
	if state == stateAppointmentManage {
		b.handleAppointmentSelection(message)
		return
	}
	if state == stateAppointmentSelectAction {
		b.handleAppointmentAction(message)
		return
	}

	// Обработка просмотра клиента
	if state == "viewing_client" {
		b.handleClientAction(message)
		return
	}

	// Обработка установки цели
	if state == "setting_goal" {
		b.handleSetGoal(message)
		return
	}

	// Обработка создания плана
	if state == "creating_plan" {
		b.handleCreatePlan(message)
		return
	}

	// Обработка ввода тренировки для выбранного клиента
	if state == "admin_awaiting_training" {
		b.handleAdminTrainingInput(message)
		return
	}

	// Обработка AI состояний
	if strings.HasPrefix(state, "ai_") {
		if state == "ai_send_to_client" {
			b.handleAISendToClient(message)
		} else {
			b.handleAIState(message, state)
		}
		return
	}

	// Обработка состояний 1ПМ
	if strings.HasPrefix(state, "1pm_") {
		b.handle1PMState(message, state)
		return
	}

	// Обработка состояний планов
	if strings.HasPrefix(state, "plan_") {
		b.handlePlanState(message, state)
		return
	}

	// Обработка состояний программ
	if strings.HasPrefix(state, "program_") {
		b.handleProgramState(message, state)
		return
	}

	// Обработка выбора клиента для программы
	if state == "program_select_client" && strings.HasPrefix(text, "Программа>> ") {
		b.handleProgramClientSelection(message, text)
		return
	}

	// Обработка выбора клиента для отправки тренировки из программы
	if state == "send_workout_select" && strings.HasPrefix(text, "Трен>> ") {
		b.handleSendWorkoutSelection(message, text)
		return
	}

	switch text {
	case "Клиенты":
		b.showClientsList(message)
	case "Добавить клиента":
		b.startAddClient(message)
	case "Отправить тренировку":
		b.showClientsForSending(message)
	case "Расписание":
		b.handleScheduleMenu(message)
	case "Добавить слот":
		b.handleAddScheduleSlot(message)
	case "Мое расписание":
		b.handleShowSchedule(message)
	case "Записи клиентов":
		b.handleTrainerAppointments(message)
	case "Удалить слот":
		b.handleDeleteScheduleSlot(message)
	case "Управление записями":
		b.handleManageAppointments(message)
	case "AI Ассистент":
		b.handleAIMenu(message)
	case "Тренеры":
		b.handleTrainersMenu(message)
	case "Добавить тренера":
		b.handleAddTrainer(message)
	case "Удалить тренера":
		b.handleRemoveTrainer(message)
	case "1ПМ Тестирование":
		b.handle1PMMenu(message)
	case "Планы тренировок":
		b.handlePlansMenu(message)
	case "Создать программу":
		b.handleCreateProgram(message)
	case "Активные программы":
		b.handleProgramsMenu(message)
	case "Отправить тренировку из программы":
		b.handleSendNextWorkout(message)
	case "Отправить первую тренировку":
		b.handleSendNextWorkout(message)
	case "AI: Сгенерировать тренировку":
		b.handleAIGenerateTraining(message)
	case "AI: План на неделю":
		b.handleAIWeekPlan(message)
	case "AI: Годовой план":
		b.handleAIYearPlan(message)
	case "AI: Задать вопрос":
		b.handleAIQuestion(message)
	case "AI: План с прогрессией":
		b.handleAIProgressionPlan(message)
	case "AI: Методики":
		b.handleAIMethodologies(message)
	case "AI: К соревнованиям":
		b.handleAICompetition(message)
	case "Отмена":
		b.handleAdminCancel(message)
	case "Назад":
		b.handleAdminStart(message)
	default:
		// Проверяем, не выбран ли клиент из списка для записи тренировки
		if strings.HasPrefix(text, ">> ") {
			b.handleClientSelection(message, text)
			return
		}
		// Проверяем, не выбран ли клиент для отправки тренировки
		if strings.HasPrefix(text, "Отправить: ") {
			b.handleSendTrainingSelection(message, text)
			return
		}
		// Проверяем выбор клиента для AI
		if strings.HasPrefix(text, "AI>> ") {
			b.handleAIClientSelection(message, text)
			return
		}
		b.sendMessage(chatID, "Неизвестная команда")
	}
}

// handleAIState обрабатывает различные AI состояния
func (b *Bot) handleAIState(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	switch state {
	case "ai_awaiting_params":
		// Выбор типа тренировки
		b.handleAITrainingType(message)
	case "ai_awaiting_direction":
		// Выбор направленности
		b.handleAIDirection(message)
	case "ai_review":
		// Решение по сгенерированной тренировке
		b.handleAIReview(message)
	case "ai_modify":
		// Ввод инструкций по модификации
		b.handleAIModifyInput(message)
	case "ai_question":
		// Ввод вопроса
		b.handleAIQuestionInput(message)
	case "ai_week_plan_client":
		// Выбор клиента для недельного плана
		if strings.HasPrefix(text, "AI>> ") {
			clientID := parseClientID(text)
			if clientID > 0 {
				userStates.Lock()
				delete(userStates.states, chatID)
				userStates.Unlock()
				b.handleAIWeekPlanGenerate(message, clientID)
			}
		}
	case "ai_year_plan_client":
		// Выбор клиента для годового плана
		if strings.HasPrefix(text, "AI>> ") {
			clientID := parseClientID(text)
			if clientID > 0 {
				userStates.Lock()
				delete(userStates.states, chatID)
				userStates.Unlock()
				b.handleAIYearPlanGenerate(message, clientID)
			}
		}
	case "ai_progression_client":
		// Выбор клиента для плана с прогрессией
		if strings.HasPrefix(text, "AI>> ") {
			clientID := parseClientID(text)
			if clientID > 0 {
				b.handleAIProgressionWeeks(message, clientID)
			}
		}
	case "ai_progression_weeks":
		b.handleAIProgressionWeeksInput(message)
	case "ai_progression_days":
		b.handleAIProgressionDaysInput(message)
	case "ai_progression_goal":
		b.handleAIProgressionGoalInput(message)
	case "ai_progression_review":
		b.handleAIProgressionReview(message)
	case "ai_methodology_select":
		b.handleAIMethodologySelect(message)
	case "ai_methodology_compare":
		b.handleAIMethodologyCompare(message)
	case "ai_methodology_compare_goal":
		b.handleAIMethodologyCompareGoal(message)
	case "ai_competition_client":
		if strings.HasPrefix(text, "AI>> ") {
			clientID := parseClientID(text)
			if clientID > 0 {
				b.handleAICompetitionSport(message, clientID)
			}
		}
	case "ai_competition_sport":
		b.handleAICompetitionSportInput(message)
	case "ai_competition_weeks":
		b.handleAICompetitionWeeksInput(message)
	case "ai_competition_review":
		b.handleAICompetitionReview(message)
	}
}

// parseClientID извлекает ID клиента из строки "AI>> Имя Фамилия [ID]"
func parseClientID(text string) int {
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		return 0
	}
	id, err := strconv.Atoi(text[start+1 : end])
	if err != nil {
		return 0
	}
	return id
}

// handleAdminCancel отменяет текущую операцию админа
func (b *Bot) handleAdminCancel(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	adminStates.Lock()
	delete(adminStates.selectedClient, chatID)
	adminStates.Unlock()

	clearState(chatID)

	b.sendMessage(chatID, "Отменено")
	b.handleAdminStart(message)
}

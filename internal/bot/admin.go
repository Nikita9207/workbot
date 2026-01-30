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
			tgbotapi.NewKeyboardButton("PL: Программы"),
			tgbotapi.NewKeyboardButton("FIT: Программы"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Расписание"),
			tgbotapi.NewKeyboardButton("1ПМ Тестирование"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Планы тренировок"),
			tgbotapi.NewKeyboardButton("Отправить тренировку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Статистика"),
			tgbotapi.NewKeyboardButton("Дни рождения"),
		),
		tgbotapi.NewKeyboardButtonRow(
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

	// Обработка состояний пауэрлифтинга
	if strings.HasPrefix(state, "pl_") {
		b.handlePLState(message, state)
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

	// Обработка состояний фитнес генератора
	if strings.HasPrefix(state, "fit_") {
		b.handleFitnessState(message, state)
		return
	}

	// Обработка состояний AI плана
	if strings.HasPrefix(state, "ai_") {
		b.handleAIState(message, state)
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
	case "Тренеры":
		b.handleTrainersMenu(message)
	case "Дни рождения":
		b.handleBirthdaysCommand(message.Chat.ID)
	case "Статистика":
		b.handleStatisticsMenu(message)
	case "📊 Общая статистика":
		b.handleGeneralStatistics(message.Chat.ID)
	case "👥 Топ активных":
		b.handleTopActiveClients(message.Chat.ID)
	case "📉 Неактивные клиенты":
		b.handleInactiveClients(message.Chat.ID)
	case "📅 За период":
		b.handlePeriodStatistics(message.Chat.ID)
	case "Добавить тренера":
		b.handleAddTrainer(message)
	case "Удалить тренера":
		b.handleRemoveTrainer(message)
	case "1ПМ Тестирование":
		b.handle1PMMenu(message)
	case "Планы тренировок":
		b.handlePlansMenu(message)
	case "PL: Программы":
		b.handlePowerliftingMenu(message)
	case "PL: Троеборье":
		b.handlePLLiftType(message, "powerlifting")
	case "PL: Жим лёжа":
		b.handlePLLiftType(message, "bench")
	case "PL: Присед":
		b.handlePLLiftType(message, "squat")
	case "PL: Становая тяга":
		b.handlePLLiftType(message, "deadlift")
	case "PL: Ягодичный мост":
		b.handlePLLiftType(message, "hip_thrust")
	case "PL: Авто-подбор":
		b.handlePLAutoSelect(message)
	case "PL: Список шаблонов":
		b.handlePLListTemplates(message)
	case "FIT: Программы":
		b.handleFitnessMenu(message)
	case "FIT: Гипертрофия":
		b.handleFitnessProgramType(message, "hypertrophy")
	case "FIT: Сила":
		b.handleFitnessProgramType(message, "strength")
	case "FIT: Жиросжигание":
		b.handleFitnessProgramType(message, "fatloss")
	case "FIT: Hyrox":
		b.handleFitnessProgramType(message, "hyrox")
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
		b.sendMessage(chatID, "Неизвестная команда")
	}
}

// handlePLState обрабатывает состояния пауэрлифтинга
func (b *Bot) handlePLState(message *tgbotapi.Message, state string) {
	text := message.Text

	if text == "Отмена" {
		b.handleAdminCancel(message)
		return
	}

	switch state {
	case "pl_select_template":
		if strings.HasPrefix(text, "TPL: ") {
			b.handlePLTemplateSelection(message)
		}
	case "pl_enter_maxes":
		b.handlePLMaxesInput(message)
	case "pl_select_days":
		b.handlePLDaysInput(message)
	case "pl_review":
		b.handlePLReview(message)
	case "pl_auto_maxes":
		b.handlePLAutoMaxesInput(message)
	case "pl_select_client":
		b.handlePLClientSelection(message)
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

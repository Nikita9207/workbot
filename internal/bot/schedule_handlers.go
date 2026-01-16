package bot

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"workbot/internal/calendar"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ScheduleData хранит данные настройки расписания
type ScheduleData struct {
	DayOfWeek    int
	StartTime    string
	EndTime      string
	SlotDuration int
	Step         int
}

var scheduleStore = struct {
	sync.RWMutex
	data map[int64]*ScheduleData
}{data: make(map[int64]*ScheduleData)}

const (
	stateScheduleDay      = "schedule_day"
	stateScheduleStart    = "schedule_start"
	stateScheduleEnd      = "schedule_end"
	stateScheduleDuration = "schedule_duration"
)

const (
	stateScheduleDeleteSelect    = "schedule_delete_select"
	stateAppointmentManage       = "appointment_manage"
	stateAppointmentSelectAction = "appointment_select_action"
)

// handleScheduleMenu показывает меню расписания для тренера
func (b *Bot) handleScheduleMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	msg := tgbotapi.NewMessage(chatID, "Управление расписанием:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Добавить слот"),
			tgbotapi.NewKeyboardButton("Мое расписание"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Удалить слот"),
			tgbotapi.NewKeyboardButton("Записи клиентов"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Управление записями"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAddScheduleSlot начинает добавление слота в расписание
func (b *Bot) handleAddScheduleSlot(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	scheduleStore.Lock()
	scheduleStore.data[chatID] = &ScheduleData{Step: 0}
	scheduleStore.Unlock()

	userStates.Lock()
	userStates.states[chatID] = stateScheduleDay
	userStates.Unlock()

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Понедельник"),
			tgbotapi.NewKeyboardButton("Вторник"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Среда"),
			tgbotapi.NewKeyboardButton("Четверг"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Пятница"),
			tgbotapi.NewKeyboardButton("Суббота"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Воскресенье"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Выберите день недели:")
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// processSchedule обрабатывает настройку расписания
func (b *Bot) processSchedule(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.cancelSchedule(chatID)
		return
	}

	scheduleStore.Lock()
	schedData := scheduleStore.data[chatID]
	if schedData == nil {
		scheduleStore.Unlock()
		b.cancelSchedule(chatID)
		return
	}

	switch state {
	case stateScheduleDay:
		dayNum := parseDayOfWeek(text)
		if dayNum < 0 {
			scheduleStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Выберите день из списка")
			b.api.Send(msg)
			return
		}
		schedData.DayOfWeek = dayNum
		schedData.Step = 1
		scheduleStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateScheduleStart
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите время начала работы (например, 09:00):")
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("09:00"),
				tgbotapi.NewKeyboardButton("10:00"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		b.api.Send(msg)

	case stateScheduleStart:
		_, _, err := calendar.ParseTime(text)
		if err != nil {
			scheduleStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Введите время как ЧЧ:ММ")
			b.api.Send(msg)
			return
		}
		schedData.StartTime = text
		schedData.Step = 2
		scheduleStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateScheduleEnd
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите время окончания работы (например, 21:00):")
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("20:00"),
				tgbotapi.NewKeyboardButton("21:00"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		b.api.Send(msg)

	case stateScheduleEnd:
		_, _, err := calendar.ParseTime(text)
		if err != nil {
			scheduleStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Введите время как ЧЧ:ММ")
			b.api.Send(msg)
			return
		}
		schedData.EndTime = text
		schedData.Step = 3
		scheduleStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateScheduleDuration
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Выберите длительность тренировки (в минутах):")
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("60"),
				tgbotapi.NewKeyboardButton("90"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		b.api.Send(msg)

	case stateScheduleDuration:
		var duration int
		_, err := fmt.Sscanf(text, "%d", &duration)
		if err != nil || duration < 30 || duration > 180 {
			scheduleStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Введите число от 30 до 180")
			b.api.Send(msg)
			return
		}

		dayOfWeek := schedData.DayOfWeek
		startTime := schedData.StartTime
		endTime := schedData.EndTime
		delete(scheduleStore.data, chatID)
		scheduleStore.Unlock()

		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()

		// Сохраняем расписание
		b.saveScheduleSlot(chatID, dayOfWeek, startTime, endTime, duration)

	default:
		scheduleStore.Unlock()
	}
}

// saveScheduleSlot сохраняет слот расписания
func (b *Bot) saveScheduleSlot(chatID int64, dayOfWeek int, startTime, endTime string, duration int) {
	_, err := b.db.Exec(`
		INSERT INTO public.trainer_schedule (trainer_id, day_of_week, start_time, end_time, slot_duration)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (trainer_id, day_of_week, start_time)
		DO UPDATE SET end_time = $4, slot_duration = $5, is_active = true`,
		chatID, dayOfWeek, startTime+":00", endTime+":00", duration)

	if err != nil {
		log.Printf("Ошибка сохранения расписания: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения расписания.")
		b.api.Send(msg)
	} else {
		dayName := getDayName(dayOfWeek)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Расписание сохранено!\n\n%s: %s - %s\nДлительность тренировки: %d мин",
			dayName, startTime, endTime, duration))
		b.api.Send(msg)
	}

	b.handleScheduleMenu(&tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}})
}

// handleShowSchedule показывает текущее расписание тренера
func (b *Bot) handleShowSchedule(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	rows, err := b.db.Query(`
		SELECT day_of_week, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'), slot_duration
		FROM public.trainer_schedule
		WHERE trainer_id = $1 AND is_active = true
		ORDER BY day_of_week, start_time`, chatID)
	if err != nil {
		log.Printf("Ошибка получения расписания: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки расписания.")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var schedule []string
	for rows.Next() {
		var dayOfWeek, duration int
		var startTime, endTime string
		if err := rows.Scan(&dayOfWeek, &startTime, &endTime, &duration); err != nil {
			continue
		}
		dayName := getDayName(dayOfWeek)
		schedule = append(schedule, fmt.Sprintf("%s: %s - %s (%d мин)",
			dayName, startTime, endTime, duration))
	}

	if len(schedule) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Расписание не настроено. Нажмите 'Добавить слот'.")
		b.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Ваше расписание:\n\n"+strings.Join(schedule, "\n"))
	b.api.Send(msg)
}

// handleTrainerAppointments показывает записи клиентов к тренеру
func (b *Bot) handleTrainerAppointments(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	rows, err := b.db.Query(`
		SELECT a.id, c.name, c.surname, a.appointment_date, TO_CHAR(a.start_time, 'HH24:MI'), a.status
		FROM public.appointments a
		JOIN public.clients c ON a.client_id = c.id
		WHERE a.trainer_id = $1 AND a.appointment_date >= CURRENT_DATE
		ORDER BY a.appointment_date, a.start_time
		LIMIT 20`, chatID)
	if err != nil {
		log.Printf("Ошибка получения записей: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки записей.")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var appointments []string
	for rows.Next() {
		var id int
		var name, surname, dateStr, timeStr, status string
		if err := rows.Scan(&id, &name, &surname, &dateStr, &timeStr, &status); err != nil {
			continue
		}

		statusText := getStatusText(status)
		appointments = append(appointments, fmt.Sprintf(
			"#%d: %s %s\n    %s в %s (%s)",
			id, name, surname, dateStr[:10], timeStr, statusText))
	}

	if len(appointments) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет предстоящих записей.")
		b.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Записи клиентов:\n\n"+strings.Join(appointments, "\n\n"))
	b.api.Send(msg)
}

// cancelSchedule отменяет настройку расписания
func (b *Bot) cancelSchedule(chatID int64) {
	scheduleStore.Lock()
	delete(scheduleStore.data, chatID)
	scheduleStore.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Настройка отменена.")
	b.api.Send(msg)
	b.handleScheduleMenu(&tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}})
}

// Вспомогательные функции

func parseDayOfWeek(text string) int {
	days := map[string]int{
		"Воскресенье": 0,
		"Понедельник": 1,
		"Вторник":     2,
		"Среда":       3,
		"Четверг":     4,
		"Пятница":     5,
		"Суббота":     6,
	}
	if day, ok := days[text]; ok {
		return day
	}
	return -1
}

func getDayName(day int) string {
	days := []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}
	if day >= 0 && day < len(days) {
		return days[day]
	}
	return ""
}

// handleDeleteScheduleSlot начинает удаление слота из расписания
func (b *Bot) handleDeleteScheduleSlot(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Получаем все активные слоты тренера
	rows, err := b.db.Query(`
		SELECT id, day_of_week, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'), slot_duration
		FROM public.trainer_schedule
		WHERE trainer_id = $1 AND is_active = true
		ORDER BY day_of_week, start_time`, chatID)
	if err != nil {
		log.Printf("Ошибка получения расписания: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки расписания.")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id, dayOfWeek, duration int
		var startTime, endTime string
		if err := rows.Scan(&id, &dayOfWeek, &startTime, &endTime, &duration); err != nil {
			continue
		}
		dayName := getDayName(dayOfWeek)
		buttonText := fmt.Sprintf("DEL>> %s %s-%s [%d]", dayName, startTime, endTime, id)
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет слотов для удаления.")
		b.api.Send(msg)
		b.handleScheduleMenu(message)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	userStates.Lock()
	userStates.states[chatID] = stateScheduleDeleteSelect
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Выберите слот для удаления:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// handleDeleteSlotSelection обрабатывает выбор слота для удаления
func (b *Bot) handleDeleteSlotSelection(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.cancelSchedule(chatID)
		return
	}

	// Парсим ID слота из текста "DEL>> День ЧЧ:ММ-ЧЧ:ММ [ID]"
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора слота")
		b.api.Send(msg)
		return
	}

	var slotID int
	_, err := fmt.Sscanf(text[start+1:end], "%d", &slotID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора слота")
		b.api.Send(msg)
		return
	}

	// Деактивируем слот (soft delete)
	_, err = b.db.Exec("UPDATE public.trainer_schedule SET is_active = false WHERE id = $1 AND trainer_id = $2",
		slotID, chatID)
	if err != nil {
		log.Printf("Ошибка удаления слота: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка удаления слота.")
		b.api.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(chatID, "Слот удалён из расписания.")
		b.api.Send(msg)
	}

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	b.handleScheduleMenu(message)
}

// handleManageAppointments показывает записи для управления
func (b *Bot) handleManageAppointments(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	rows, err := b.db.Query(`
		SELECT a.id, c.name, c.surname, a.appointment_date, TO_CHAR(a.start_time, 'HH24:MI'), a.status
		FROM public.appointments a
		JOIN public.clients c ON a.client_id = c.id
		WHERE a.trainer_id = $1 AND a.appointment_date >= CURRENT_DATE AND a.status != 'cancelled'
		ORDER BY a.appointment_date, a.start_time
		LIMIT 15`, chatID)
	if err != nil {
		log.Printf("Ошибка получения записей: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки записей.")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id int
		var name, surname, dateStr, timeStr, status string
		if err := rows.Scan(&id, &name, &surname, &dateStr, &timeStr, &status); err != nil {
			continue
		}

		statusEmoji := getStatusEmoji(status)
		buttonText := fmt.Sprintf("APT>> %s%s %s %s %s [%d]", statusEmoji, name, surname[:1]+".", dateStr[:10], timeStr, id)
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет активных записей для управления.")
		b.api.Send(msg)
		b.handleScheduleMenu(message)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	userStates.Lock()
	userStates.states[chatID] = stateAppointmentManage
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Выберите запись для управления:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// appointmentStore хранит выбранную запись для управления
var appointmentStore = struct {
	sync.RWMutex
	selected map[int64]int
}{selected: make(map[int64]int)}

// handleAppointmentSelection обрабатывает выбор записи для управления
func (b *Bot) handleAppointmentSelection(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()
		b.handleScheduleMenu(message)
		return
	}

	// Парсим ID записи из текста "APT>> ... [ID]"
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора записи")
		b.api.Send(msg)
		return
	}

	var appointmentID int
	_, err := fmt.Sscanf(text[start+1:end], "%d", &appointmentID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора записи")
		b.api.Send(msg)
		return
	}

	// Сохраняем выбранную запись
	appointmentStore.Lock()
	appointmentStore.selected[chatID] = appointmentID
	appointmentStore.Unlock()

	// Получаем детали записи
	var clientName, clientSurname, dateStr, timeStr, status string
	err = b.db.QueryRow(`
		SELECT c.name, c.surname, a.appointment_date, TO_CHAR(a.start_time, 'HH24:MI'), a.status
		FROM public.appointments a
		JOIN public.clients c ON a.client_id = c.id
		WHERE a.id = $1`, appointmentID).Scan(&clientName, &clientSurname, &dateStr, &timeStr, &status)
	if err != nil {
		log.Printf("Ошибка получения записи: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Запись не найдена.")
		b.api.Send(msg)
		b.handleScheduleMenu(message)
		return
	}

	statusText := getStatusText(status)
	detailsMsg := fmt.Sprintf(
		"Запись #%d\n\n"+
			"Клиент: %s %s\n"+
			"Дата: %s\n"+
			"Время: %s\n"+
			"Статус: %s\n\n"+
			"Выберите действие:",
		appointmentID, clientName, clientSurname, dateStr[:10], timeStr, statusText)

	userStates.Lock()
	userStates.states[chatID] = stateAppointmentSelectAction
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, detailsMsg)
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Подтвердить"),
			tgbotapi.NewKeyboardButton("Завершить"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отменить запись"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAppointmentAction обрабатывает действие с записью
func (b *Bot) handleAppointmentAction(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	action := message.Text

	if action == "Назад" {
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()
		appointmentStore.Lock()
		delete(appointmentStore.selected, chatID)
		appointmentStore.Unlock()
		b.handleManageAppointments(message)
		return
	}

	appointmentStore.RLock()
	appointmentID := appointmentStore.selected[chatID]
	appointmentStore.RUnlock()

	if appointmentID == 0 {
		msg := tgbotapi.NewMessage(chatID, "Запись не выбрана")
		b.api.Send(msg)
		b.handleScheduleMenu(message)
		return
	}

	var newStatus string
	var statusMsg string

	switch action {
	case "Подтвердить":
		newStatus = "confirmed"
		statusMsg = "Запись подтверждена"
	case "Завершить":
		newStatus = "completed"
		statusMsg = "Тренировка завершена"
	case "Отменить запись":
		newStatus = "cancelled"
		statusMsg = "Запись отменена"
	default:
		msg := tgbotapi.NewMessage(chatID, "Неизвестное действие")
		b.api.Send(msg)
		return
	}

	// Обновляем статус
	_, err := b.db.Exec("UPDATE public.appointments SET status = $1, updated_at = NOW() WHERE id = $2",
		newStatus, appointmentID)
	if err != nil {
		log.Printf("Ошибка обновления статуса: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка обновления статуса.")
		b.api.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(chatID, statusMsg)
		b.api.Send(msg)

		// Уведомляем клиента об изменении статуса
		b.notifyClientAboutStatusChange(appointmentID, newStatus)
	}

	// Очищаем состояние
	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()
	appointmentStore.Lock()
	delete(appointmentStore.selected, chatID)
	appointmentStore.Unlock()

	b.handleScheduleMenu(message)
}

// notifyClientAboutStatusChange уведомляет клиента об изменении статуса записи
func (b *Bot) notifyClientAboutStatusChange(appointmentID int, newStatus string) {
	// Получаем данные записи и клиента
	var clientTelegramID int64
	var clientName, dateStr, timeStr string
	err := b.db.QueryRow(`
		SELECT c.telegram_id, c.name, a.appointment_date, TO_CHAR(a.start_time, 'HH24:MI')
		FROM public.appointments a
		JOIN public.clients c ON a.client_id = c.id
		WHERE a.id = $1 AND c.telegram_id IS NOT NULL`, appointmentID).
		Scan(&clientTelegramID, &clientName, &dateStr, &timeStr)
	if err != nil || clientTelegramID == 0 {
		return // Клиент без telegram или ошибка
	}

	var statusMsg string
	switch newStatus {
	case "confirmed":
		statusMsg = fmt.Sprintf("Ваша запись на тренировку подтверждена!\n\nДата: %s\nВремя: %s", dateStr[:10], timeStr)
	case "completed":
		statusMsg = fmt.Sprintf("Тренировка %s в %s завершена. Отличная работа!", dateStr[:10], timeStr)
	case "cancelled":
		statusMsg = fmt.Sprintf("Ваша запись на %s в %s была отменена.\n\nДля записи на другое время нажмите 'Записаться на тренировку'", dateStr[:10], timeStr)
	default:
		return
	}

	msg := tgbotapi.NewMessage(clientTelegramID, statusMsg)
	b.api.Send(msg)
}

// getStatusEmoji возвращает эмодзи для статуса
func getStatusEmoji(status string) string {
	switch status {
	case "scheduled":
		return "📅"
	case "confirmed":
		return "✅"
	case "completed":
		return "🏆"
	case "cancelled":
		return "❌"
	default:
		return "📋"
	}
}

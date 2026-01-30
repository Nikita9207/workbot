package bot

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"workbot/internal/calendar"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BookingData хранит данные бронирования
type BookingData struct {
	Date      time.Time
	TimeSlot  string
	Step      int // 0=date, 1=time, 2=confirm
	MessageID int // ID сообщения с календарём для редактирования
	Calendar  *CalendarWidget
}

var bookingStore = struct {
	sync.RWMutex
	data map[int64]*BookingData
}{data: make(map[int64]*BookingData)}

// Константы состояний бронирования
const (
	stateBookingDate    = "booking_date"
	stateBookingTime    = "booking_time"
	stateBookingConfirm = "booking_confirm"
)

// handleBookTraining начинает процесс записи на тренировку
func (b *Bot) handleBookTraining(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Проверяем регистрацию
	var clientID int
	err := b.db.QueryRow("SELECT id FROM public.clients WHERE telegram_id = $1", chatID).Scan(&clientID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, b.t("booking_need_register", chatID))
		b.api.Send(msg)
		return
	}

	// Показываем доступные даты (следующие 14 дней)
	b.showAvailableDates(chatID)
}

// showAvailableDates показывает визуальный календарь для записи
func (b *Bot) showAvailableDates(chatID int64) {
	// Создаём виджет календаря
	cal := NewCalendarWidget()

	// Получаем полностью занятые даты (опционально)
	bookedDates := b.getFullyBookedDates()
	cal.SetBookedDates(bookedDates)

	// Инициализируем данные бронирования
	bookingStore.Lock()
	bookingStore.data[chatID] = &BookingData{
		Step:     0,
		Calendar: cal,
	}
	bookingStore.Unlock()

	userStates.Lock()
	userStates.states[chatID] = stateBookingDate
	userStates.Unlock()

	// Убираем Reply клавиатуру
	hideKeyboard := tgbotapi.NewRemoveKeyboard(true)
	hideMsg := tgbotapi.NewMessage(chatID, b.t("booking_select_date", chatID))
	hideMsg.ReplyMarkup = hideKeyboard
	b.api.Send(hideMsg)

	// Отправляем календарь
	msg := tgbotapi.NewMessage(chatID, b.t("booking_select_date_short", chatID))
	msg.ReplyMarkup = cal.GenerateCalendar()
	sentMsg, err := b.api.Send(msg)
	if err == nil {
		// Сохраняем ID сообщения для последующего редактирования
		bookingStore.Lock()
		if data, ok := bookingStore.data[chatID]; ok {
			data.MessageID = sentMsg.MessageID
		}
		bookingStore.Unlock()
	}
}

// getFullyBookedDates возвращает даты где все слоты заняты
func (b *Bot) getFullyBookedDates() []string {
	// Упрощённая реализация - можно расширить
	return nil
}

// handleCallbackQuery обрабатывает нажатия на inline-кнопки
func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	// Отвечаем на callback чтобы убрать "часики"
	callbackResponse := tgbotapi.NewCallback(callback.ID, "")
	b.api.Request(callbackResponse)

	// Обрабатываем разные типы callback
	switch {
	case data == "cal_ignore" || data == "time_ignore":
		// Игнорируем
		return

	case data == "cal_cancel":
		b.cancelBookingWithInline(chatID, callback.Message.MessageID)
		return

	case strings.HasPrefix(data, "cal_prev_"):
		b.handleCalendarNavigation(chatID, callback.Message.MessageID, data, -1)
		return

	case strings.HasPrefix(data, "cal_next_"):
		b.handleCalendarNavigation(chatID, callback.Message.MessageID, data, 1)
		return

	case strings.HasPrefix(data, "cal_day_"):
		dateStr := strings.TrimPrefix(data, "cal_day_")
		b.handleDateSelection(chatID, callback.Message.MessageID, dateStr)
		return

	case strings.HasPrefix(data, "time_slot_"):
		parts := strings.TrimPrefix(data, "time_slot_")
		b.handleTimeSlotSelection(chatID, callback.Message.MessageID, parts)
		return

	case data == "time_back":
		b.handleBackToCalendar(chatID, callback.Message.MessageID)
		return

	case strings.HasPrefix(data, "change_time_"):
		dateStr := strings.TrimPrefix(data, "change_time_")
		b.handleDateSelection(chatID, callback.Message.MessageID, dateStr)
		return

	case data == "change_date":
		b.handleBackToCalendar(chatID, callback.Message.MessageID)
		return

	case strings.HasPrefix(data, "confirm_"):
		parts := strings.TrimPrefix(data, "confirm_")
		b.handleBookingConfirmation(chatID, callback.Message.MessageID, parts)
		return

	case strings.HasPrefix(data, "stats_"):
		b.handleStatsCallback(chatID, callback.Message.MessageID, data)
		return

	case strings.HasPrefix(data, "settings_"), strings.HasPrefix(data, "lang_"):
		b.handleSettingsCallback(callback)
		return

	case strings.HasPrefix(data, "workout_"):
		b.handleWorkoutCallback(callback)
		return
	}
}

// handleCalendarNavigation обрабатывает навигацию по месяцам
func (b *Bot) handleCalendarNavigation(chatID int64, messageID int, data string, direction int) {
	bookingStore.Lock()
	bookData := bookingStore.data[chatID]
	if bookData == nil || bookData.Calendar == nil {
		bookingStore.Unlock()
		return
	}

	cal := bookData.Calendar
	// Переходим на месяц вперёд/назад
	newMonth := time.Date(cal.Year, cal.Month, 1, 0, 0, 0, 0, time.Local).AddDate(0, direction, 0)
	cal.Year = newMonth.Year()
	cal.Month = newMonth.Month()
	bookingStore.Unlock()

	// Обновляем сообщение с новым календарём
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, cal.GenerateCalendar())
	b.api.Send(edit)
}

// handleDateSelection обрабатывает выбор даты
func (b *Bot) handleDateSelection(chatID int64, messageID int, dateStr string) {
	date, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		return
	}

	bookingStore.Lock()
	bookData := bookingStore.data[chatID]
	if bookData == nil {
		bookingStore.Unlock()
		return
	}
	bookData.Date = date
	bookData.Step = 1
	bookingStore.Unlock()

	// Получаем доступные слоты
	availableSlots := b.getAvailableTimeSlotsForDate(date)
	if len(availableSlots) == 0 {
		// Нет свободных слотов
		text := fmt.Sprintf("❌ На %s нет свободных слотов.\nВыберите другую дату:", dateStr)
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		bookingStore.RLock()
		if bookData.Calendar != nil {
			edit.ReplyMarkup = &[]tgbotapi.InlineKeyboardMarkup{bookData.Calendar.GenerateCalendar()}[0]
		}
		bookingStore.RUnlock()
		b.api.Send(edit)
		return
	}

	// Показываем слоты времени
	text := fmt.Sprintf("🕐 Выберите время на %s:", dateStr)
	keyboard := GenerateTimeSlots(date, availableSlots)
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	b.api.Send(edit)
}

// handleTimeSlotSelection обрабатывает выбор времени
func (b *Bot) handleTimeSlotSelection(chatID int64, messageID int, data string) {
	// data формат: "02.01.2006_15:00"
	parts := strings.Split(data, "_")
	if len(parts) != 2 {
		return
	}
	dateStr, timeSlot := parts[0], parts[1]

	date, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		return
	}

	bookingStore.Lock()
	bookData := bookingStore.data[chatID]
	if bookData == nil {
		bookingStore.Unlock()
		return
	}
	bookData.Date = date
	bookData.TimeSlot = timeSlot
	bookData.Step = 2
	bookingStore.Unlock()

	// Показываем подтверждение
	dayName := russianWeekdayFull(date.Weekday())
	text := fmt.Sprintf(
		"📋 Подтвердите запись:\n\n"+
			"📅 Дата: %s (%s)\n"+
			"🕐 Время: %s\n\n"+
			"Подтвердить запись?",
		dateStr, dayName, timeSlot)

	keyboard := GenerateConfirmation(date, timeSlot)
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	b.api.Send(edit)
}

// handleBookingConfirmation обрабатывает подтверждение записи
func (b *Bot) handleBookingConfirmation(chatID int64, messageID int, data string) {
	// data формат: "02.01.2006_15:00"
	parts := strings.Split(data, "_")
	if len(parts) != 2 {
		return
	}
	dateStr, timeSlot := parts[0], parts[1]

	date, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		return
	}

	hour, minute, err := calendar.ParseTime(timeSlot)
	if err != nil {
		return
	}

	// Удаляем inline-клавиатуру
	edit := tgbotapi.NewEditMessageText(chatID, messageID, "⏳ Создаю запись...")
	b.api.Send(edit)

	// Очищаем состояние
	bookingStore.Lock()
	delete(bookingStore.data, chatID)
	bookingStore.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	// Создаём запись
	b.createAppointment(chatID, date, hour, minute)
}

// handleBackToCalendar возвращает к календарю
func (b *Bot) handleBackToCalendar(chatID int64, messageID int) {
	bookingStore.Lock()
	bookData := bookingStore.data[chatID]
	if bookData == nil {
		bookingStore.Unlock()
		return
	}

	if bookData.Calendar == nil {
		bookData.Calendar = NewCalendarWidget()
	}
	bookData.Step = 0
	bookingStore.Unlock()

	text := "📅 Выберите дату тренировки:"
	bookingStore.RLock()
	keyboard := bookData.Calendar.GenerateCalendar()
	bookingStore.RUnlock()

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ReplyMarkup = &keyboard
	b.api.Send(edit)
}

// cancelBookingWithInline отменяет бронирование и удаляет inline-сообщение
func (b *Bot) cancelBookingWithInline(chatID int64, messageID int) {
	bookingStore.Lock()
	delete(bookingStore.data, chatID)
	bookingStore.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	// Удаляем сообщение с календарём
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	b.api.Request(deleteMsg)

	msg := tgbotapi.NewMessage(chatID, "❌ "+b.t("booking_cancelled", chatID))
	b.api.Send(msg)
	b.restoreMainMenu(chatID)
}

// getAvailableTimeSlotsForDate возвращает доступные слоты для даты
func (b *Bot) getAvailableTimeSlotsForDate(date time.Time) []string {
	dayOfWeek := int(date.Weekday())

	// Получаем расписание тренера
	rows, err := b.db.Query(`
		SELECT start_time, end_time, slot_duration
		FROM public.trainer_schedule
		WHERE day_of_week = $1 AND is_active = true
		ORDER BY start_time`, dayOfWeek)

	var timeSlots []string

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var startTime, endTime string
			var slotDuration int
			if err := rows.Scan(&startTime, &endTime, &slotDuration); err != nil {
				continue
			}
			slots := generateTimeSlots(startTime, endTime, slotDuration)
			timeSlots = append(timeSlots, slots...)
		}
	}

	// Стандартные слоты если расписания нет
	if len(timeSlots) == 0 {
		timeSlots = []string{
			"09:00", "10:00", "11:00", "12:00",
			"14:00", "15:00", "16:00", "17:00",
			"18:00", "19:00", "20:00",
		}
	}

	// Фильтруем занятые
	bookedSlots, _ := b.getBookedSlots(date)
	return filterBookedSlots(timeSlots, bookedSlots)
}

// processBooking обрабатывает шаги бронирования
func (b *Bot) processBooking(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.cancelBooking(chatID)
		return
	}

	bookingStore.Lock()
	bookData := bookingStore.data[chatID]
	if bookData == nil {
		bookingStore.Unlock()
		b.cancelBooking(chatID)
		return
	}

	switch state {
	case stateBookingDate:
		// Парсим дату из текста "02.01.2006 (Пн)"
		parts := strings.Split(text, " ")
		if len(parts) < 1 {
			bookingStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите дату из списка")
			b.api.Send(msg)
			return
		}

		date, err := time.Parse("02.01.2006", parts[0])
		if err != nil {
			bookingStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Неверный формат даты. Выберите из списка.")
			b.api.Send(msg)
			return
		}

		bookData.Date = date
		bookData.Step = 1
		bookingStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateBookingTime
		userStates.Unlock()

		// Показываем доступные слоты времени
		b.showAvailableTimeSlots(chatID, date)

	case stateBookingTime:
		// Парсим время
		hour, minute, err := calendar.ParseTime(text)
		if err != nil {
			bookingStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите время из списка")
			b.api.Send(msg)
			return
		}

		bookData.TimeSlot = text
		date := bookData.Date
		delete(bookingStore.data, chatID)
		bookingStore.Unlock()

		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()

		// Создаём запись
		b.createAppointment(chatID, date, hour, minute)

	default:
		bookingStore.Unlock()
	}
}

// showAvailableTimeSlots показывает доступные слоты времени
func (b *Bot) showAvailableTimeSlots(chatID int64, date time.Time) {
	dayOfWeek := int(date.Weekday())

	// Получаем расписание тренера на этот день
	rows, err := b.db.Query(`
		SELECT start_time, end_time, slot_duration
		FROM public.trainer_schedule
		WHERE day_of_week = $1 AND is_active = true
		ORDER BY start_time`, dayOfWeek)

	var timeSlots []string

	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var startTime, endTime string
			var slotDuration int
			if err := rows.Scan(&startTime, &endTime, &slotDuration); err != nil {
				continue
			}

			// Генерируем слоты
			slots := generateTimeSlots(startTime, endTime, slotDuration)
			timeSlots = append(timeSlots, slots...)
		}
	}

	// Если расписания нет — показываем стандартные слоты
	if len(timeSlots) == 0 {
		timeSlots = []string{
			"09:00", "10:00", "11:00", "12:00",
			"14:00", "15:00", "16:00", "17:00",
			"18:00", "19:00", "20:00",
		}
	}

	// Фильтруем занятые слоты
	bookedSlots, _ := b.getBookedSlots(date)
	availableSlots := filterBookedSlots(timeSlots, bookedSlots)

	if len(availableSlots) == 0 {
		msg := tgbotapi.NewMessage(chatID, "К сожалению, на эту дату нет свободных слотов. Выберите другую дату.")
		b.api.Send(msg)
		b.showAvailableDates(chatID)
		return
	}

	// Создаём кнопки
	var buttonRows [][]tgbotapi.KeyboardButton
	for i := 0; i < len(availableSlots); i += 3 {
		var row []tgbotapi.KeyboardButton
		for j := i; j < i+3 && j < len(availableSlots); j++ {
			row = append(row, tgbotapi.NewKeyboardButton(availableSlots[j]))
		}
		buttonRows = append(buttonRows, row)
	}
	buttonRows = append(buttonRows, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Назад"),
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Выберите время на %s:",
		date.Format("02.01.2006")))
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttonRows...)
	b.api.Send(msg)
}

// getBookedSlots возвращает занятые слоты на дату
func (b *Bot) getBookedSlots(date time.Time) ([]string, error) {
	rows, err := b.db.Query(`
		SELECT TO_CHAR(start_time, 'HH24:MI')
		FROM public.appointments
		WHERE appointment_date = $1 AND status != 'cancelled'`,
		date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []string
	for rows.Next() {
		var slot string
		if err := rows.Scan(&slot); err != nil {
			continue
		}
		slots = append(slots, slot)
	}
	return slots, nil
}

// createAppointment создаёт запись на тренировку
func (b *Bot) createAppointment(chatID int64, date time.Time, hour, minute int) {
	// Получаем ID клиента
	var clientID int
	var clientName, clientSurname string
	err := b.db.QueryRow("SELECT id, name, surname FROM public.clients WHERE telegram_id = $1", chatID).
		Scan(&clientID, &clientName, &clientSurname)
	if err != nil {
		log.Printf("Ошибка получения клиента: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка создания записи.")
		b.api.Send(msg)
		b.restoreMainMenu(chatID)
		return
	}

	// Получаем тренера (первого админа)
	var trainerID int64
	err = b.db.QueryRow("SELECT telegram_id FROM public.admins LIMIT 1").Scan(&trainerID)
	if err != nil {
		log.Printf("Ошибка получения тренера: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка: тренер не найден.")
		b.api.Send(msg)
		b.restoreMainMenu(chatID)
		return
	}

	startTimeStr := fmt.Sprintf("%02d:%02d:00", hour, minute)
	endTimeStr := fmt.Sprintf("%02d:%02d:00", hour+1, minute) // +1 час

	// Проверяем, не занят ли слот (double-check перед INSERT)
	var existingCount int
	err = b.db.QueryRow(`
		SELECT COUNT(*) FROM public.appointments
		WHERE trainer_id = $1 AND appointment_date = $2 AND start_time = $3 AND status != 'cancelled'`,
		trainerID, date.Format("2006-01-02"), startTimeStr).Scan(&existingCount)
	if err == nil && existingCount > 0 {
		msg := tgbotapi.NewMessage(chatID, "❌ К сожалению, это время уже занято другим клиентом.\nПожалуйста, выберите другое время.")
		b.api.Send(msg)
		b.showAvailableDates(chatID)
		return
	}

	// Создаём запись в БД
	var appointmentID int
	err = b.db.QueryRow(`
		INSERT INTO public.appointments (client_id, trainer_id, appointment_date, start_time, end_time, status)
		VALUES ($1, $2, $3, $4, $5, 'scheduled')
		RETURNING id`,
		clientID, trainerID, date.Format("2006-01-02"), startTimeStr, endTimeStr).Scan(&appointmentID)

	if err != nil {
		log.Printf("Ошибка создания записи: %v", err)
		// Проверяем, является ли ошибка нарушением уникальности (duplicate key)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			msg := tgbotapi.NewMessage(chatID, "❌ Это время только что было занято другим клиентом.\nПожалуйста, выберите другое время.")
			b.api.Send(msg)
			b.showAvailableDates(chatID)
			return
		}
		msg := tgbotapi.NewMessage(chatID, "Ошибка создания записи. Попробуйте позже.")
		b.api.Send(msg)
		b.restoreMainMenu(chatID)
		return
	}

	// Формируем событие для ICS файла
	eventStart := calendar.CombineDateTime(date, hour, minute)
	eventEnd := eventStart.Add(time.Hour)
	event := calendar.Event{
		UID:         fmt.Sprintf("training-%d@workbot", appointmentID),
		Summary:     "Тренировка",
		Description: fmt.Sprintf("Персональная тренировка\\nКлиент: %s %s", clientName, clientSurname),
		StartTime:   eventStart,
		EndTime:     eventEnd,
		Reminder:    60, // напоминание за 1 час
	}
	icsContent := calendar.GenerateICS(event)

	// Формируем сообщение подтверждения
	confirmMsg := fmt.Sprintf(
		"✅ Вы записаны на тренировку!\n\n"+
			"📅 Дата: %s\n"+
			"🕐 Время: %02d:%02d",
		date.Format("02.01.2006"), hour, minute)

	msg := tgbotapi.NewMessage(chatID, confirmMsg)
	b.api.Send(msg)

	// Отправляем .ics файл для добавления в календарь
	fileName := fmt.Sprintf("training_%s_%02d%02d.ics", date.Format("02-01-2006"), hour, minute)
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
		Name:  fileName,
		Bytes: []byte(icsContent),
	})
	doc.Caption = "Откройте файл для добавления в календарь"
	b.api.Send(doc)

	// Уведомляем тренера
	trainerMsg := tgbotapi.NewMessage(trainerID, fmt.Sprintf(
		"📝 Новая запись на тренировку!\n\n"+
			"👤 Клиент: %s %s\n"+
			"📅 Дата: %s\n"+
			"🕐 Время: %02d:%02d",
		clientName, clientSurname, date.Format("02.01.2006"), hour, minute))
	b.api.Send(trainerMsg)

	b.restoreMainMenu(chatID)
}

// cancelBooking отменяет бронирование
func (b *Bot) cancelBooking(chatID int64) {
	bookingStore.Lock()
	delete(bookingStore.data, chatID)
	bookingStore.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, b.t("booking_cancelled", chatID))
	b.api.Send(msg)
	b.restoreMainMenu(chatID)
}

// handleMyAppointments показывает записи клиента
func (b *Bot) handleMyAppointments(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	var clientID int
	err := b.db.QueryRow("SELECT id FROM public.clients WHERE telegram_id = $1", chatID).Scan(&clientID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, b.t("reg_not_registered", chatID))
		b.api.Send(msg)
		return
	}

	rows, err := b.db.Query(`
		SELECT id, appointment_date, start_time, status
		FROM public.appointments
		WHERE client_id = $1 AND appointment_date >= CURRENT_DATE AND status != 'cancelled'
		ORDER BY appointment_date, start_time
		LIMIT 10`, clientID)
	if err != nil {
		log.Printf("Ошибка получения записей: %v", err)
		msg := tgbotapi.NewMessage(chatID, b.t("error", chatID))
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var appointments []string
	for rows.Next() {
		var id int
		var date, startTime, status string
		if err := rows.Scan(&id, &date, &startTime, &status); err != nil {
			continue
		}

		parsedDate, _ := time.Parse("2006-01-02T15:04:05Z", date)
		statusText := b.getStatusTextLocalized(status, chatID)
		appointments = append(appointments, fmt.Sprintf(
			"#%d: %s в %s (%s)",
			id, parsedDate.Format("02.01.2006"), startTime[:5], statusText))
	}

	if len(appointments) == 0 {
		msg := tgbotapi.NewMessage(chatID, b.t("appointments_empty", chatID))
		b.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, b.t("appointments_title", chatID)+"\n\n"+strings.Join(appointments, "\n"))
	b.api.Send(msg)
}

// handleExportCalendar экспортирует все записи в .ics
func (b *Bot) handleExportCalendar(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	var clientID int
	var clientName, clientSurname string
	err := b.db.QueryRow("SELECT id, name, surname FROM public.clients WHERE telegram_id = $1", chatID).
		Scan(&clientID, &clientName, &clientSurname)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, b.t("reg_not_registered", chatID))
		b.api.Send(msg)
		return
	}

	rows, err := b.db.Query(`
		SELECT id, appointment_date, start_time, end_time
		FROM public.appointments
		WHERE client_id = $1 AND appointment_date >= CURRENT_DATE AND status != 'cancelled'
		ORDER BY appointment_date, start_time`, clientID)
	if err != nil {
		log.Printf("Ошибка получения записей: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки записей.")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var events []calendar.Event
	for rows.Next() {
		var id int
		var dateStr, startTimeStr, endTimeStr string
		if err := rows.Scan(&id, &dateStr, &startTimeStr, &endTimeStr); err != nil {
			continue
		}

		date, _ := time.Parse("2006-01-02T15:04:05Z", dateStr)
		startHour, startMin, _ := calendar.ParseTime(startTimeStr[:5])
		endHour, endMin, _ := calendar.ParseTime(endTimeStr[:5])

		events = append(events, calendar.Event{
			UID:         fmt.Sprintf("training-%d@workbot", id),
			Summary:     "Тренировка",
			Description: fmt.Sprintf("Персональная тренировка\\nКлиент: %s %s", clientName, clientSurname),
			StartTime:   calendar.CombineDateTime(date, startHour, startMin),
			EndTime:     calendar.CombineDateTime(date, endHour, endMin),
			Reminder:    60,
		})
	}

	if len(events) == 0 {
		msg := tgbotapi.NewMessage(chatID, b.t("calendar_no_appointments", chatID))
		b.api.Send(msg)
		return
	}

	icsContent := calendar.GenerateMultipleICS(events)

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
		Name:  "trainings.ics",
		Bytes: []byte(icsContent),
	})
	doc.Caption = fmt.Sprintf("Ваши тренировки (%d записей)\nОткройте файл для добавления в календарь", len(events))
	b.api.Send(doc)
}

// Вспомогательные функции

func russianWeekday(w time.Weekday) string {
	days := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	return days[w]
}

func generateTimeSlots(startTime, endTime string, slotDuration int) []string {
	start, _ := time.Parse("15:04:05", startTime)
	end, _ := time.Parse("15:04:05", endTime)

	var slots []string
	for t := start; t.Before(end); t = t.Add(time.Duration(slotDuration) * time.Minute) {
		slots = append(slots, t.Format("15:04"))
	}
	return slots
}

func filterBookedSlots(all []string, booked []string) []string {
	bookedMap := make(map[string]bool)
	for _, s := range booked {
		bookedMap[s] = true
	}

	var available []string
	for _, s := range all {
		if !bookedMap[s] {
			available = append(available, s)
		}
	}
	return available
}

func getStatusText(status string) string {
	switch status {
	case "scheduled":
		return "запланирована"
	case "confirmed":
		return "подтверждена"
	case "completed":
		return "завершена"
	case "cancelled":
		return "отменена"
	default:
		return status
	}
}

// getStatusTextLocalized возвращает локализованный статус
func (b *Bot) getStatusTextLocalized(status string, chatID int64) string {
	lang := b.getLanguage(chatID)
	if lang == "en" {
		switch status {
		case "scheduled":
			return "scheduled"
		case "confirmed":
			return "confirmed"
		case "completed":
			return "completed"
		case "cancelled":
			return "cancelled"
		default:
			return status
		}
	}
	return getStatusText(status)
}

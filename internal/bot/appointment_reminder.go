package bot

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AppointmentReminder содержит информацию о записи для напоминания
type AppointmentReminder struct {
	AppointmentID   int
	ClientID        int
	ClientTelegramID int64
	ClientName      string
	ClientSurname   string
	TrainerID       int64
	AppointmentDate time.Time
	StartTime       string
	ReminderType    string // "1day", "1hour"
}

// StartAppointmentReminder запускает фоновую задачу напоминаний о тренировках
func (b *Bot) StartAppointmentReminder() {
	go func() {
		// Ждём 15 секунд после старта
		time.Sleep(15 * time.Second)
		log.Println("Запущен сервис напоминаний о тренировках")

		// Проверяем каждые 30 минут
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		// Первая проверка сразу
		b.checkAndSendAppointmentReminders()

		for range ticker.C {
			b.checkAndSendAppointmentReminders()
		}
	}()
}

// checkAndSendAppointmentReminders проверяет и отправляет напоминания
func (b *Bot) checkAndSendAppointmentReminders() {
	now := time.Now()
	log.Printf("Проверка напоминаний о тренировках: %s", now.Format("02.01.2006 15:04"))

	// Напоминания за 1 день (проверяем тренировки на завтра)
	tomorrow := now.AddDate(0, 0, 1)
	oneDayReminders := b.getAppointmentsForReminder(tomorrow, "1day")
	for _, reminder := range oneDayReminders {
		b.sendAppointmentReminder(reminder)
	}

	// Напоминания за 1 час
	oneHourLater := now.Add(time.Hour)
	oneHourReminders := b.getAppointmentsForReminder(oneHourLater, "1hour")
	for _, reminder := range oneHourReminders {
		b.sendAppointmentReminder(reminder)
	}
}

// getAppointmentsForReminder получает записи для напоминания
func (b *Bot) getAppointmentsForReminder(targetTime time.Time, reminderType string) []AppointmentReminder {
	var reminders []AppointmentReminder

	var query string
	var args []interface{}

	if reminderType == "1day" {
		// За 1 день: проверяем записи на целевую дату
		// Отправляем напоминания в 9:00-9:30 утра за день до тренировки
		currentHour := time.Now().Hour()
		if currentHour < 9 || currentHour >= 10 {
			return reminders // Напоминания за день отправляем только утром
		}

		query = `
			SELECT a.id, a.client_id, COALESCE(c.telegram_id, 0), c.name, c.surname,
			       a.trainer_id, a.appointment_date, TO_CHAR(a.start_time, 'HH24:MI')
			FROM public.appointments a
			JOIN public.clients c ON a.client_id = c.id
			WHERE a.appointment_date = $1
			  AND a.status IN ('scheduled', 'confirmed')
			  AND COALESCE(a.reminder_1day_sent, false) = false
			  AND c.telegram_id IS NOT NULL
			ORDER BY a.start_time`
		args = []interface{}{targetTime.Format("2006-01-02")}
	} else {
		// За 1 час: проверяем записи в пределах часа
		query = `
			SELECT a.id, a.client_id, COALESCE(c.telegram_id, 0), c.name, c.surname,
			       a.trainer_id, a.appointment_date, TO_CHAR(a.start_time, 'HH24:MI')
			FROM public.appointments a
			JOIN public.clients c ON a.client_id = c.id
			WHERE a.appointment_date = $1
			  AND a.start_time >= $2::time
			  AND a.start_time < ($2::time + interval '30 minutes')
			  AND a.status IN ('scheduled', 'confirmed')
			  AND COALESCE(a.reminder_1hour_sent, false) = false
			  AND c.telegram_id IS NOT NULL
			ORDER BY a.start_time`
		args = []interface{}{targetTime.Format("2006-01-02"), targetTime.Format("15:04:05")}
	}

	rows, err := b.db.Query(query, args...)
	if err != nil {
		log.Printf("Ошибка получения записей для напоминаний: %v", err)
		return reminders
	}
	defer rows.Close()

	for rows.Next() {
		var r AppointmentReminder
		var dateStr string
		if err := rows.Scan(&r.AppointmentID, &r.ClientID, &r.ClientTelegramID,
			&r.ClientName, &r.ClientSurname, &r.TrainerID, &dateStr, &r.StartTime); err != nil {
			continue
		}

		r.AppointmentDate, _ = time.Parse("2006-01-02T15:04:05Z", dateStr)
		r.ReminderType = reminderType
		reminders = append(reminders, r)
	}

	return reminders
}

// sendAppointmentReminder отправляет напоминание клиенту
func (b *Bot) sendAppointmentReminder(reminder AppointmentReminder) {
	if reminder.ClientTelegramID == 0 {
		return
	}

	chatID := reminder.ClientTelegramID
	dateStr := reminder.AppointmentDate.Format("02.01.2006")
	dayName := b.getWeekdayNameLocalized(reminder.AppointmentDate.Weekday(), chatID)

	var message string
	if reminder.ReminderType == "1day" {
		message = b.t("reminder_1day_title", chatID) + "\n\n" +
			b.tf("reminder_1day_text", chatID, dateStr, dayName, reminder.StartTime)
	} else {
		message = b.t("reminder_1hour_title", chatID) + "\n\n" +
			b.tf("reminder_1hour_text", chatID, dateStr, reminder.StartTime)
	}

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки напоминания клиенту %d: %v", chatID, err)
		return
	}

	// Отмечаем напоминание как отправленное
	b.markReminderSent(reminder.AppointmentID, reminder.ReminderType)
	log.Printf("Отправлено напоминание (%s) клиенту %s %s на %s %s",
		reminder.ReminderType, reminder.ClientName, reminder.ClientSurname,
		dateStr, reminder.StartTime)
}

// getWeekdayNameLocalized возвращает локализованное название дня недели
func (b *Bot) getWeekdayNameLocalized(w time.Weekday, chatID int64) string {
	keys := []string{
		"weekday_sunday", "weekday_monday", "weekday_tuesday",
		"weekday_wednesday", "weekday_thursday", "weekday_friday", "weekday_saturday",
	}
	return b.t(keys[w], chatID)
}

// markReminderSent отмечает напоминание как отправленное
func (b *Bot) markReminderSent(appointmentID int, reminderType string) {
	var column string
	if reminderType == "1day" {
		column = "reminder_1day_sent"
	} else {
		column = "reminder_1hour_sent"
	}

	_, err := b.db.Exec(
		fmt.Sprintf("UPDATE public.appointments SET %s = true WHERE id = $1", column),
		appointmentID)
	if err != nil {
		log.Printf("Ошибка обновления флага напоминания: %v", err)
	}
}

// Примечание: russianWeekdayFull определена в calendar_widget.go

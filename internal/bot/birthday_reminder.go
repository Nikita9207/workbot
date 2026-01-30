package bot

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BirthdayInfo хранит информацию о дне рождения клиента
type BirthdayInfo struct {
	ClientID  int
	Name      string
	Surname   string
	BirthDate time.Time
	Age       int
	DaysUntil int // 0 = сегодня, 1 = завтра, и т.д.
}

// StartBirthdayReminder запускает фоновую задачу проверки дней рождения
func (b *Bot) StartBirthdayReminder() {
	go func() {
		// Ждём 10 секунд после старта, чтобы бот полностью инициализировался
		time.Sleep(10 * time.Second)

		// Проверяем сразу при старте
		b.checkAndSendBirthdayReminders()

		// Запускаем ежедневную проверку в 9:00
		for {
			now := time.Now()
			// Вычисляем время до следующих 9:00
			next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			sleepDuration := next.Sub(now)

			log.Printf("Следующая проверка дней рождения через %v", sleepDuration.Round(time.Minute))
			time.Sleep(sleepDuration)

			b.checkAndSendBirthdayReminders()
		}
	}()
}

// checkAndSendBirthdayReminders проверяет и отправляет напоминания
func (b *Bot) checkAndSendBirthdayReminders() {
	log.Println("Проверка дней рождения клиентов...")

	// Получаем дни рождения на сегодня и завтра
	todayBirthdays := b.getUpcomingBirthdays(0)
	tomorrowBirthdays := b.getUpcomingBirthdays(1)
	weekBirthdays := b.getUpcomingBirthdays(7)

	// Получаем список админов (тренеров)
	admins := b.getAdminTelegramIDs()

	if len(admins) == 0 {
		log.Println("Нет админов для отправки напоминаний")
		return
	}

	// Отправляем напоминания каждому админу
	for _, adminID := range admins {
		b.sendBirthdayNotifications(adminID, todayBirthdays, tomorrowBirthdays, weekBirthdays)
	}
}

// getUpcomingBirthdays возвращает клиентов с днём рождения через daysAhead дней
func (b *Bot) getUpcomingBirthdays(daysAhead int) []BirthdayInfo {
	var birthdays []BirthdayInfo

	targetDate := time.Now().AddDate(0, 0, daysAhead)
	targetDay := targetDate.Day()
	targetMonth := int(targetDate.Month())

	// Ищем клиентов с днём рождения в указанный день
	rows, err := b.db.Query(`
		SELECT id, name, surname, birth_date
		FROM public.clients
		WHERE deleted_at IS NULL
		  AND birth_date IS NOT NULL
		  AND EXTRACT(DAY FROM birth_date::date) = $1
		  AND EXTRACT(MONTH FROM birth_date::date) = $2
		ORDER BY name, surname
	`, targetDay, targetMonth)
	if err != nil {
		log.Printf("Ошибка получения дней рождения: %v", err)
		return birthdays
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id        int
			name      string
			surname   string
			birthDate string
		)
		if err := rows.Scan(&id, &name, &surname, &birthDate); err != nil {
			continue
		}

		// Парсим дату
		parsedDate, err := time.Parse("02.01.2006", birthDate)
		if err != nil {
			// Пробуем другой формат (из БД может быть 2006-01-02)
			parsedDate, err = time.Parse("2006-01-02", birthDate)
			if err != nil {
				continue
			}
		}

		// Вычисляем возраст
		age := targetDate.Year() - parsedDate.Year()

		birthdays = append(birthdays, BirthdayInfo{
			ClientID:  id,
			Name:      name,
			Surname:   surname,
			BirthDate: parsedDate,
			Age:       age,
			DaysUntil: daysAhead,
		})
	}

	return birthdays
}

// getAdminTelegramIDs возвращает Telegram ID всех админов
func (b *Bot) getAdminTelegramIDs() []int64 {
	var admins []int64

	rows, err := b.db.Query("SELECT telegram_id FROM public.admins")
	if err != nil {
		log.Printf("Ошибка получения админов: %v", err)
		return admins
	}
	defer rows.Close()

	for rows.Next() {
		var telegramID int64
		if err := rows.Scan(&telegramID); err != nil {
			continue
		}
		admins = append(admins, telegramID)
	}

	return admins
}

// sendBirthdayNotifications отправляет уведомления о днях рождения админу
func (b *Bot) sendBirthdayNotifications(adminID int64, today, tomorrow, week []BirthdayInfo) {
	var message string

	// Сегодня
	if len(today) > 0 {
		message += "🎂 *СЕГОДНЯ ДЕНЬ РОЖДЕНИЯ!*\n\n"
		for _, bd := range today {
			message += fmt.Sprintf("🎉 *%s %s* — %d лет!\n", bd.Name, bd.Surname, bd.Age)
		}
		message += "\n"
	}

	// Завтра
	if len(tomorrow) > 0 {
		message += "🎈 *Завтра день рождения:*\n\n"
		for _, bd := range tomorrow {
			message += fmt.Sprintf("• %s %s — исполнится %d лет\n", bd.Name, bd.Surname, bd.Age)
		}
		message += "\n"
	}

	// На этой неделе (исключая сегодня и завтра)
	var weekFiltered []BirthdayInfo
	for _, bd := range week {
		if bd.DaysUntil > 1 {
			weekFiltered = append(weekFiltered, bd)
		}
	}
	if len(weekFiltered) > 0 {
		message += "📅 *На этой неделе:*\n\n"
		for _, bd := range weekFiltered {
			dayStr := getDayWord(bd.DaysUntil)
			message += fmt.Sprintf("• %s %s — через %d %s (%d лет)\n",
				bd.Name, bd.Surname, bd.DaysUntil, dayStr, bd.Age)
		}
	}

	// Отправляем только если есть что отправить
	if message != "" {
		msg := tgbotapi.NewMessage(adminID, message)
		msg.ParseMode = "Markdown"
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Ошибка отправки напоминания о ДР админу %d: %v", adminID, err)
		} else {
			log.Printf("Отправлено напоминание о днях рождения админу %d", adminID)
		}
	}
}

// getDayWord возвращает правильное склонение слова "день"
func getDayWord(days int) string {
	if days%10 == 1 && days%100 != 11 {
		return "день"
	}
	if days%10 >= 2 && days%10 <= 4 && (days%100 < 10 || days%100 >= 20) {
		return "дня"
	}
	return "дней"
}

// GetTodayBirthdays возвращает клиентов с днём рождения сегодня (для ручного вызова)
func (b *Bot) GetTodayBirthdays() []BirthdayInfo {
	return b.getUpcomingBirthdays(0)
}

// handleBirthdaysCommand обрабатывает команду просмотра дней рождения
func (b *Bot) handleBirthdaysCommand(chatID int64) {
	today := b.getUpcomingBirthdays(0)
	tomorrow := b.getUpcomingBirthdays(1)

	// Собираем ближайшие 7 дней
	var upcoming []BirthdayInfo
	for i := 2; i <= 7; i++ {
		upcoming = append(upcoming, b.getUpcomingBirthdays(i)...)
	}

	var message string

	if len(today) == 0 && len(tomorrow) == 0 && len(upcoming) == 0 {
		message = "📅 В ближайшую неделю дней рождения нет"
	} else {
		message = "📅 *Ближайшие дни рождения*\n\n"

		if len(today) > 0 {
			message += "🎂 *Сегодня:*\n"
			for _, bd := range today {
				message += fmt.Sprintf("  🎉 %s %s — %d лет!\n", bd.Name, bd.Surname, bd.Age)
			}
			message += "\n"
		}

		if len(tomorrow) > 0 {
			message += "🎈 *Завтра:*\n"
			for _, bd := range tomorrow {
				message += fmt.Sprintf("  • %s %s — %d лет\n", bd.Name, bd.Surname, bd.Age)
			}
			message += "\n"
		}

		if len(upcoming) > 0 {
			message += "📆 *Ближайшие 7 дней:*\n"
			for _, bd := range upcoming {
				dayStr := getDayWord(bd.DaysUntil)
				message += fmt.Sprintf("  • %s %s — через %d %s (%d лет)\n",
					bd.Name, bd.Surname, bd.DaysUntil, dayStr, bd.Age)
			}
		}
	}

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

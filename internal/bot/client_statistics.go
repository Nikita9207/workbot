package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ClientStats содержит статистику клиента
type ClientStats struct {
	ClientID           int
	Name               string
	Surname            string
	TotalAppointments  int
	CompletedTrainings int
	CancelledTrainings int
	LastTrainingDate   time.Time
	RegistrationDate   time.Time
	AttendanceRate     float64 // процент посещаемости
	AvgTrainingsPerMonth float64
}

// handleStatisticsMenu показывает меню статистики для тренера
func (b *Bot) handleStatisticsMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 Общая статистика"),
			tgbotapi.NewKeyboardButton("👥 Топ активных"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📉 Неактивные клиенты"),
			tgbotapi.NewKeyboardButton("📅 За период"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📊 *Статистика клиентов*\n\nВыберите тип отчёта:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleGeneralStatistics показывает общую статистику
func (b *Bot) handleGeneralStatistics(chatID int64) {
	var totalClients, activeClients, totalTrainings, completedTrainings, cancelledTrainings int
	var monthTrainings, weekTrainings int

	// Всего клиентов
	b.db.QueryRow("SELECT COUNT(*) FROM public.clients WHERE deleted_at IS NULL").Scan(&totalClients)

	// Активные клиенты (были на тренировке за последний месяц)
	b.db.QueryRow(`
		SELECT COUNT(DISTINCT client_id)
		FROM public.appointments
		WHERE appointment_date >= CURRENT_DATE - INTERVAL '30 days'
		  AND status IN ('completed', 'confirmed', 'scheduled')
	`).Scan(&activeClients)

	// Всего тренировок
	b.db.QueryRow("SELECT COUNT(*) FROM public.appointments").Scan(&totalTrainings)

	// Завершённые тренировки
	b.db.QueryRow("SELECT COUNT(*) FROM public.appointments WHERE status = 'completed'").Scan(&completedTrainings)

	// Отменённые
	b.db.QueryRow("SELECT COUNT(*) FROM public.appointments WHERE status = 'cancelled'").Scan(&cancelledTrainings)

	// За этот месяц
	b.db.QueryRow(`
		SELECT COUNT(*) FROM public.appointments
		WHERE appointment_date >= DATE_TRUNC('month', CURRENT_DATE)
		  AND status != 'cancelled'
	`).Scan(&monthTrainings)

	// За эту неделю
	b.db.QueryRow(`
		SELECT COUNT(*) FROM public.appointments
		WHERE appointment_date >= DATE_TRUNC('week', CURRENT_DATE)
		  AND status != 'cancelled'
	`).Scan(&weekTrainings)

	// Рассчитываем процент посещаемости
	attendanceRate := 0.0
	if totalTrainings > 0 {
		attendanceRate = float64(completedTrainings) / float64(totalTrainings) * 100
	}

	var message strings.Builder
	message.WriteString("📊 *Общая статистика*\n\n")

	message.WriteString("👥 *Клиенты:*\n")
	message.WriteString(fmt.Sprintf("  • Всего: %d\n", totalClients))
	message.WriteString(fmt.Sprintf("  • Активных (30 дней): %d\n\n", activeClients))

	message.WriteString("🏋️ *Тренировки:*\n")
	message.WriteString(fmt.Sprintf("  • Всего записей: %d\n", totalTrainings))
	message.WriteString(fmt.Sprintf("  • Завершено: %d\n", completedTrainings))
	message.WriteString(fmt.Sprintf("  • Отменено: %d\n", cancelledTrainings))
	message.WriteString(fmt.Sprintf("  • Посещаемость: %.1f%%\n\n", attendanceRate))

	message.WriteString("📅 *Период:*\n")
	message.WriteString(fmt.Sprintf("  • Эта неделя: %d тренировок\n", weekTrainings))
	message.WriteString(fmt.Sprintf("  • Этот месяц: %d тренировок\n", monthTrainings))

	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// handleTopActiveClients показывает топ активных клиентов
func (b *Bot) handleTopActiveClients(chatID int64) {
	rows, err := b.db.Query(`
		SELECT c.id, c.name, c.surname,
		       COUNT(a.id) as total_trainings,
		       COUNT(CASE WHEN a.status = 'completed' THEN 1 END) as completed,
		       MAX(a.appointment_date) as last_training
		FROM public.clients c
		LEFT JOIN public.appointments a ON c.id = a.client_id
		WHERE c.deleted_at IS NULL
		GROUP BY c.id, c.name, c.surname
		HAVING COUNT(CASE WHEN a.status = 'completed' THEN 1 END) > 0
		ORDER BY completed DESC, last_training DESC
		LIMIT 10
	`)
	if err != nil {
		log.Printf("Ошибка получения топа клиентов: %v", err)
		b.sendMessage(chatID, "Ошибка загрузки статистики")
		return
	}
	defer rows.Close()

	var message strings.Builder
	message.WriteString("🏆 *Топ-10 активных клиентов*\n\n")

	rank := 1
	for rows.Next() {
		var clientID int
		var name, surname string
		var totalTrainings, completed int
		var lastTraining *string

		if err := rows.Scan(&clientID, &name, &surname, &totalTrainings, &completed, &lastTraining); err != nil {
			continue
		}

		medal := ""
		switch rank {
		case 1:
			medal = "🥇"
		case 2:
			medal = "🥈"
		case 3:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("%d.", rank)
		}

		lastDateStr := "—"
		if lastTraining != nil {
			lastDate, _ := time.Parse("2006-01-02T15:04:05Z", *lastTraining)
			lastDateStr = lastDate.Format("02.01")
		}

		message.WriteString(fmt.Sprintf("%s *%s %s*\n", medal, name, surname))
		message.WriteString(fmt.Sprintf("   📈 %d тренировок | Последняя: %s\n\n", completed, lastDateStr))

		rank++
	}

	if rank == 1 {
		message.WriteString("Пока нет данных о тренировках клиентов")
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// handleInactiveClients показывает неактивных клиентов
func (b *Bot) handleInactiveClients(chatID int64) {
	rows, err := b.db.Query(`
		SELECT c.id, c.name, c.surname, c.phone,
		       MAX(a.appointment_date) as last_training,
		       CURRENT_DATE - MAX(a.appointment_date)::date as days_inactive
		FROM public.clients c
		LEFT JOIN public.appointments a ON c.id = a.client_id AND a.status = 'completed'
		WHERE c.deleted_at IS NULL
		GROUP BY c.id, c.name, c.surname, c.phone
		HAVING MAX(a.appointment_date) IS NULL
		    OR MAX(a.appointment_date) < CURRENT_DATE - INTERVAL '14 days'
		ORDER BY days_inactive DESC NULLS FIRST
		LIMIT 15
	`)
	if err != nil {
		log.Printf("Ошибка получения неактивных клиентов: %v", err)
		b.sendMessage(chatID, "Ошибка загрузки статистики")
		return
	}
	defer rows.Close()

	var message strings.Builder
	message.WriteString("📉 *Неактивные клиенты*\n")
	message.WriteString("_(более 14 дней без тренировок)_\n\n")

	count := 0
	for rows.Next() {
		var clientID int
		var name, surname, phone string
		var lastTraining *string
		var daysInactive *int

		if err := rows.Scan(&clientID, &name, &surname, &phone, &lastTraining, &daysInactive); err != nil {
			continue
		}

		count++

		inactiveStr := "никогда"
		if lastTraining != nil && daysInactive != nil {
			lastDate, _ := time.Parse("2006-01-02T15:04:05Z", *lastTraining)
			inactiveStr = fmt.Sprintf("%d дн. (с %s)", *daysInactive, lastDate.Format("02.01"))
		}

		message.WriteString(fmt.Sprintf("⚠️ *%s %s*\n", name, surname))
		message.WriteString(fmt.Sprintf("   📱 %s\n", phone))
		message.WriteString(fmt.Sprintf("   ⏰ Не был: %s\n\n", inactiveStr))
	}

	if count == 0 {
		message.WriteString("✅ Все клиенты активны!")
	} else {
		message.WriteString(fmt.Sprintf("\n_Всего неактивных: %d_", count))
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// handlePeriodStatistics показывает статистику за период
func (b *Bot) handlePeriodStatistics(chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 За неделю", "stats_week"),
			tgbotapi.NewInlineKeyboardButtonData("📅 За месяц", "stats_month"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 За 3 месяца", "stats_quarter"),
			tgbotapi.NewInlineKeyboardButtonData("📅 За год", "stats_year"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📅 Выберите период для статистики:")
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleStatsCallback обрабатывает callback для статистики
func (b *Bot) handleStatsCallback(chatID int64, messageID int, period string) {
	var interval string
	var periodName string

	switch period {
	case "stats_week":
		interval = "7 days"
		periodName = "за неделю"
	case "stats_month":
		interval = "30 days"
		periodName = "за месяц"
	case "stats_quarter":
		interval = "90 days"
		periodName = "за 3 месяца"
	case "stats_year":
		interval = "365 days"
		periodName = "за год"
	default:
		return
	}

	var totalTrainings, completedTrainings, cancelledTrainings, uniqueClients int
	var revenue float64 // если есть поле стоимости тренировки

	// Всего тренировок за период
	b.db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*) FROM public.appointments
		WHERE appointment_date >= CURRENT_DATE - INTERVAL '%s'
	`, interval)).Scan(&totalTrainings)

	// Завершённые
	b.db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*) FROM public.appointments
		WHERE appointment_date >= CURRENT_DATE - INTERVAL '%s'
		  AND status = 'completed'
	`, interval)).Scan(&completedTrainings)

	// Отменённые
	b.db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*) FROM public.appointments
		WHERE appointment_date >= CURRENT_DATE - INTERVAL '%s'
		  AND status = 'cancelled'
	`, interval)).Scan(&cancelledTrainings)

	// Уникальные клиенты
	b.db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(DISTINCT client_id) FROM public.appointments
		WHERE appointment_date >= CURRENT_DATE - INTERVAL '%s'
		  AND status != 'cancelled'
	`, interval)).Scan(&uniqueClients)

	// Статистика по дням недели
	dayStats := b.getTrainingsByDayOfWeek(interval)

	var message strings.Builder
	message.WriteString(fmt.Sprintf("📊 *Статистика %s*\n\n", periodName))

	message.WriteString("🏋️ *Тренировки:*\n")
	message.WriteString(fmt.Sprintf("  • Всего: %d\n", totalTrainings))
	message.WriteString(fmt.Sprintf("  • Завершено: %d\n", completedTrainings))
	message.WriteString(fmt.Sprintf("  • Отменено: %d\n", cancelledTrainings))

	if totalTrainings > 0 {
		rate := float64(completedTrainings) / float64(totalTrainings) * 100
		message.WriteString(fmt.Sprintf("  • Посещаемость: %.1f%%\n", rate))
	}

	message.WriteString(fmt.Sprintf("\n👥 Уникальных клиентов: %d\n", uniqueClients))

	if revenue > 0 {
		message.WriteString(fmt.Sprintf("💰 Доход: %.0f ₽\n", revenue))
	}

	// Популярные дни
	if len(dayStats) > 0 {
		message.WriteString("\n📅 *По дням недели:*\n")
		for _, ds := range dayStats {
			bar := strings.Repeat("█", ds.count/2)
			if len(bar) == 0 && ds.count > 0 {
				bar = "▌"
			}
			message.WriteString(fmt.Sprintf("  %s %s %d\n", ds.day, bar, ds.count))
		}
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, message.String())
	edit.ParseMode = "Markdown"
	b.api.Send(edit)
}

type dayStat struct {
	day   string
	count int
}

// getTrainingsByDayOfWeek возвращает статистику по дням недели
func (b *Bot) getTrainingsByDayOfWeek(interval string) []dayStat {
	rows, err := b.db.Query(fmt.Sprintf(`
		SELECT EXTRACT(DOW FROM appointment_date) as dow, COUNT(*) as cnt
		FROM public.appointments
		WHERE appointment_date >= CURRENT_DATE - INTERVAL '%s'
		  AND status = 'completed'
		GROUP BY dow
		ORDER BY dow
	`, interval))
	if err != nil {
		return nil
	}
	defer rows.Close()

	days := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	stats := make(map[int]int)

	for rows.Next() {
		var dow, cnt int
		if err := rows.Scan(&dow, &cnt); err != nil {
			continue
		}
		stats[dow] = cnt
	}

	var result []dayStat
	// Начинаем с понедельника
	for i := 1; i <= 7; i++ {
		dow := i % 7 // 1,2,3,4,5,6,0 -> Пн,Вт,Ср,Чт,Пт,Сб,Вс
		result = append(result, dayStat{
			day:   days[dow],
			count: stats[dow],
		})
	}

	return result
}

// getClientStatistics возвращает статистику по конкретному клиенту
func (b *Bot) getClientStatistics(clientID int) *ClientStats {
	stats := &ClientStats{ClientID: clientID}

	// Основные данные клиента
	var createdAt string
	err := b.db.QueryRow(`
		SELECT name, surname, created_at
		FROM public.clients WHERE id = $1`, clientID).
		Scan(&stats.Name, &stats.Surname, &createdAt)
	if err != nil {
		return nil
	}
	stats.RegistrationDate, _ = time.Parse("2006-01-02T15:04:05Z", createdAt)

	// Статистика тренировок
	b.db.QueryRow(`
		SELECT COUNT(*) FROM public.appointments WHERE client_id = $1`,
		clientID).Scan(&stats.TotalAppointments)

	b.db.QueryRow(`
		SELECT COUNT(*) FROM public.appointments
		WHERE client_id = $1 AND status = 'completed'`,
		clientID).Scan(&stats.CompletedTrainings)

	b.db.QueryRow(`
		SELECT COUNT(*) FROM public.appointments
		WHERE client_id = $1 AND status = 'cancelled'`,
		clientID).Scan(&stats.CancelledTrainings)

	// Последняя тренировка
	var lastDate *string
	b.db.QueryRow(`
		SELECT MAX(appointment_date) FROM public.appointments
		WHERE client_id = $1 AND status = 'completed'`,
		clientID).Scan(&lastDate)
	if lastDate != nil {
		stats.LastTrainingDate, _ = time.Parse("2006-01-02T15:04:05Z", *lastDate)
	}

	// Посещаемость (завершённые / (завершённые + отменённые))
	scheduled := stats.TotalAppointments - stats.CancelledTrainings
	if scheduled > 0 {
		stats.AttendanceRate = float64(stats.CompletedTrainings) / float64(scheduled) * 100
	}

	// Среднее в месяц
	if !stats.RegistrationDate.IsZero() {
		months := time.Since(stats.RegistrationDate).Hours() / 24 / 30
		if months >= 1 {
			stats.AvgTrainingsPerMonth = float64(stats.CompletedTrainings) / months
		}
	}

	return stats
}

// handleClientStatistics показывает статистику конкретного клиента для тренера
func (b *Bot) handleClientStatistics(chatID int64, clientID int) {
	stats := b.getClientStatistics(clientID)
	if stats == nil {
		b.sendMessage(chatID, "Клиент не найден")
		return
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("📊 *Статистика: %s %s*\n\n", stats.Name, stats.Surname))

	message.WriteString("🏋️ *Тренировки:*\n")
	message.WriteString(fmt.Sprintf("  • Всего записей: %d\n", stats.TotalAppointments))
	message.WriteString(fmt.Sprintf("  • Завершено: %d\n", stats.CompletedTrainings))
	message.WriteString(fmt.Sprintf("  • Отменено: %d\n", stats.CancelledTrainings))
	message.WriteString(fmt.Sprintf("  • Посещаемость: %.1f%%\n", stats.AttendanceRate))

	if stats.AvgTrainingsPerMonth > 0 {
		message.WriteString(fmt.Sprintf("  • В среднем: %.1f/мес\n", stats.AvgTrainingsPerMonth))
	}

	message.WriteString("\n📅 *Даты:*\n")
	message.WriteString(fmt.Sprintf("  • Регистрация: %s\n", stats.RegistrationDate.Format("02.01.2006")))
	if !stats.LastTrainingDate.IsZero() {
		daysAgo := int(time.Since(stats.LastTrainingDate).Hours() / 24)
		message.WriteString(fmt.Sprintf("  • Последняя тренировка: %s (%d дн. назад)\n",
			stats.LastTrainingDate.Format("02.01.2006"), daysAgo))
	} else {
		message.WriteString("  • Последняя тренировка: нет данных\n")
	}

	// Оценка активности
	var activityEmoji string
	if stats.AttendanceRate >= 80 {
		activityEmoji = "🌟"
	} else if stats.AttendanceRate >= 50 {
		activityEmoji = "👍"
	} else {
		activityEmoji = "⚠️"
	}
	message.WriteString(fmt.Sprintf("\n%s Оценка активности: ", activityEmoji))
	if stats.AttendanceRate >= 80 {
		message.WriteString("*Отличная*")
	} else if stats.AttendanceRate >= 50 {
		message.WriteString("*Хорошая*")
	} else {
		message.WriteString("*Требует внимания*")
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

package bot

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CalendarWidget визуальный календарь для Telegram
type CalendarWidget struct {
	Year        int
	Month       time.Month
	MinDate     time.Time // Минимальная дата для выбора
	MaxDate     time.Time // Максимальная дата для выбора
	BookedDates map[string]bool // Занятые даты
}

// NewCalendarWidget создаёт новый виджет календаря
func NewCalendarWidget() *CalendarWidget {
	now := time.Now()
	return &CalendarWidget{
		Year:        now.Year(),
		Month:       now.Month(),
		MinDate:     now.AddDate(0, 0, 1),   // Завтра
		MaxDate:     now.AddDate(0, 0, 30),  // +30 дней
		BookedDates: make(map[string]bool),
	}
}

// SetBookedDates устанавливает полностью занятые даты
func (c *CalendarWidget) SetBookedDates(dates []string) {
	c.BookedDates = make(map[string]bool)
	for _, d := range dates {
		c.BookedDates[d] = true
	}
}

// GenerateCalendar создаёт inline-клавиатуру с календарём
func (c *CalendarWidget) GenerateCalendar() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Заголовок: < Январь 2026 >
	monthName := russianMonth(c.Month)
	headerRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("◀", fmt.Sprintf("cal_prev_%d_%d", c.Year, c.Month)),
		tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %d", monthName, c.Year), "cal_ignore"),
		tgbotapi.NewInlineKeyboardButtonData("▶", fmt.Sprintf("cal_next_%d_%d", c.Year, c.Month)),
	}
	rows = append(rows, headerRow)

	// Дни недели
	weekdays := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
	var weekdayRow []tgbotapi.InlineKeyboardButton
	for _, wd := range weekdays {
		weekdayRow = append(weekdayRow, tgbotapi.NewInlineKeyboardButtonData(wd, "cal_ignore"))
	}
	rows = append(rows, weekdayRow)

	// Первый день месяца
	firstDay := time.Date(c.Year, c.Month, 1, 0, 0, 0, 0, time.Local)
	// Понедельник = 0, ... Воскресенье = 6
	startWeekday := int(firstDay.Weekday())
	if startWeekday == 0 {
		startWeekday = 7 // Воскресенье
	}
	startWeekday-- // Теперь 0 = Понедельник

	// Количество дней в месяце
	lastDay := firstDay.AddDate(0, 1, -1)
	daysInMonth := lastDay.Day()

	// Генерируем ряды дней
	day := 1
	for week := 0; week < 6; week++ {
		var dayRow []tgbotapi.InlineKeyboardButton
		for weekday := 0; weekday < 7; weekday++ {
			if (week == 0 && weekday < startWeekday) || day > daysInMonth {
				// Пустая ячейка
				dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData(" ", "cal_ignore"))
			} else {
				currentDate := time.Date(c.Year, c.Month, day, 0, 0, 0, 0, time.Local)
				dateStr := currentDate.Format("02.01.2006")

				// Проверяем доступность даты
				buttonText := fmt.Sprintf("%d", day)
				callbackData := fmt.Sprintf("cal_day_%s", dateStr)

				if currentDate.Before(c.MinDate) || currentDate.After(c.MaxDate) {
					// Дата вне диапазона
					buttonText = "·"
					callbackData = "cal_ignore"
				} else if c.BookedDates[dateStr] {
					// Полностью занята
					buttonText = fmt.Sprintf("✗%d", day)
					callbackData = "cal_ignore"
				} else {
					// Доступна - выделяем
					if currentDate.Equal(time.Now().Truncate(24*time.Hour).AddDate(0, 0, 1)) {
						buttonText = fmt.Sprintf("▸%d◂", day) // Завтра
					}
				}

				dayRow = append(dayRow, tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData))
				day++
			}
		}
		rows = append(rows, dayRow)
		if day > daysInMonth {
			break
		}
	}

	// Кнопка отмены
	cancelRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cal_cancel"),
	}
	rows = append(rows, cancelRow)

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// GenerateTimeSlots создаёт клавиатуру с временными слотами
func GenerateTimeSlots(date time.Time, availableSlots []string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Заголовок
	dateStr := date.Format("02.01.2006")
	dayName := russianWeekdayFull(date.Weekday())
	headerRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📅 %s (%s)", dateStr, dayName), "time_ignore"),
	}
	rows = append(rows, headerRow)

	// Группируем слоты по 3 в ряд
	for i := 0; i < len(availableSlots); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := i; j < i+3 && j < len(availableSlots); j++ {
			slot := availableSlots[j]
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🕐 %s", slot),
				fmt.Sprintf("time_slot_%s_%s", dateStr, slot),
			))
		}
		rows = append(rows, row)
	}

	// Кнопки навигации
	navRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("◀ Назад", "time_back"),
		tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cal_cancel"),
	}
	rows = append(rows, navRow)

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// GenerateConfirmation создаёт клавиатуру подтверждения
func GenerateConfirmation(date time.Time, timeSlot string) tgbotapi.InlineKeyboardMarkup {
	dateStr := date.Format("02.01.2006")
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", fmt.Sprintf("confirm_%s_%s", dateStr, timeSlot)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ Изменить время", fmt.Sprintf("change_time_%s", dateStr)),
			tgbotapi.NewInlineKeyboardButtonData("◀ Изменить дату", "change_date"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cal_cancel"),
		),
	)
}

// Вспомогательные функции

func russianMonth(m time.Month) string {
	months := []string{
		"", "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
		"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
	}
	return months[m]
}

func russianWeekdayFull(w time.Weekday) string {
	days := []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}
	return days[w]
}

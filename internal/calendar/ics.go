package calendar

import (
	"fmt"
	"strings"
	"time"
)

// Event представляет событие календаря
type Event struct {
	UID         string
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	Reminder    int // минут до события
}

// GenerateICS генерирует содержимое .ics файла для события
func GenerateICS(event Event) string {
	var sb strings.Builder

	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//WorkBot//Training Calendar//RU\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")

	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:%s\r\n", event.UID))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(time.Now())))
	sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(event.StartTime)))
	sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(event.EndTime)))
	sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(event.Summary)))

	if event.Description != "" {
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(event.Description)))
	}
	if event.Location != "" {
		sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICS(event.Location)))
	}

	// Напоминание
	if event.Reminder > 0 {
		sb.WriteString("BEGIN:VALARM\r\n")
		sb.WriteString("ACTION:DISPLAY\r\n")
		sb.WriteString(fmt.Sprintf("TRIGGER:-PT%dM\r\n", event.Reminder))
		sb.WriteString("DESCRIPTION:Напоминание о тренировке\r\n")
		sb.WriteString("END:VALARM\r\n")
	}

	sb.WriteString("END:VEVENT\r\n")
	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String()
}

// GenerateMultipleICS генерирует .ics файл с несколькими событиями
func GenerateMultipleICS(events []Event) string {
	var sb strings.Builder

	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//WorkBot//Training Calendar//RU\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString("X-WR-CALNAME:Тренировки\r\n")

	for _, event := range events {
		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s\r\n", event.UID))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(time.Now())))
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(event.StartTime)))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(event.EndTime)))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(event.Summary)))

		if event.Description != "" {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(event.Description)))
		}
		if event.Location != "" {
			sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICS(event.Location)))
		}

		if event.Reminder > 0 {
			sb.WriteString("BEGIN:VALARM\r\n")
			sb.WriteString("ACTION:DISPLAY\r\n")
			sb.WriteString(fmt.Sprintf("TRIGGER:-PT%dM\r\n", event.Reminder))
			sb.WriteString("DESCRIPTION:Напоминание о тренировке\r\n")
			sb.WriteString("END:VALARM\r\n")
		}

		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String()
}

// formatICSTime форматирует время в формат iCalendar
func formatICSTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// escapeICS экранирует специальные символы для iCalendar
func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// ParseDate парсит дату в формате ДД.ММ.ГГГГ
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("02.01.2006", dateStr)
}

// ParseTime парсит время в формате ЧЧ:ММ
func ParseTime(timeStr string) (hour, minute int, err error) {
	_, err = fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil {
		return 0, 0, fmt.Errorf("неверный формат времени, используйте ЧЧ:ММ")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("некорректное время")
	}
	return hour, minute, nil
}

// CombineDateTime объединяет дату и время
func CombineDateTime(date time.Time, hour, minute int) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
}

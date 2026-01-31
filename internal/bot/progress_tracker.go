package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ProgressEntry хранит запись прогресса клиента
type ProgressEntry struct {
	ID          int
	ClientID    int
	RecordDate  time.Time
	Weight      float64 // кг
	BodyFat     float64 // % жира (опционально)
	Chest       float64 // см
	Waist       float64 // см
	Hips        float64 // см
	Biceps      float64 // см
	Thigh       float64 // см
	PhotoFileID string  // Telegram file_id фотографии (опционально)
	Notes       string
	CreatedAt   time.Time
}

// ProgressState хранит состояние ввода прогресса
type ProgressState struct {
	ClientID    int
	Step        string // "weight", "measurements", "photo", "notes"
	Weight      float64
	BodyFat     float64
	Chest       float64
	Waist       float64
	Hips        float64
	Biceps      float64
	Thigh       float64
	PhotoFileID string
	Notes       string
}

var progressStore = struct {
	sync.RWMutex
	data map[int64]*ProgressState
}{data: make(map[int64]*ProgressState)}

// Состояния для трекера прогресса
const (
	stateProgressWeight       = "progress_weight"
	stateProgressMeasurements = "progress_measurements"
	stateProgressPhoto        = "progress_photo"
	stateProgressNotes        = "progress_notes"
	stateProgressViewHistory  = "progress_view_history"
)

// handleProgressMenu показывает меню прогресса для клиента
func (b *Bot) handleProgressMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏋️ Прогресс программы"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(b.t("progress_btn_record", chatID)),
			tgbotapi.NewKeyboardButton(b.t("progress_btn_view", chatID)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(b.t("progress_btn_weight", chatID)),
			tgbotapi.NewKeyboardButton(b.t("progress_btn_measurements", chatID)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(b.t("back", chatID)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, b.t("progress_menu_title", chatID))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleStartProgress начинает запись нового прогресса
func (b *Bot) handleStartProgress(chatID int64) {
	// Получаем ID клиента
	var clientID int
	err := b.db.QueryRow("SELECT id FROM public.clients WHERE telegram_id = $1", chatID).Scan(&clientID)
	if err != nil {
		b.sendMessage(chatID, b.t("reg_not_registered", chatID))
		return
	}

	// Инициализируем состояние
	progressStore.Lock()
	progressStore.data[chatID] = &ProgressState{
		ClientID: clientID,
		Step:     "weight",
	}
	progressStore.Unlock()

	setState(chatID, stateProgressWeight)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(b.t("skip", chatID)),
			tgbotapi.NewKeyboardButton(b.t("cancel", chatID)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "⚖️ *"+b.t("progress_enter_weight", chatID)+"*\n\nНапример: 75.5")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// processProgressState обрабатывает состояния ввода прогресса
func (b *Bot) processProgressState(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" || text == "Cancel" {
		b.cancelProgress(chatID)
		return
	}

	progressStore.Lock()
	pState := progressStore.data[chatID]
	if pState == nil {
		progressStore.Unlock()
		b.cancelProgress(chatID)
		return
	}

	switch state {
	case stateProgressWeight:
		if text != "Пропустить" && text != "Skip" {
			weight, err := strconv.ParseFloat(strings.Replace(text, ",", ".", 1), 64)
			if err != nil || weight <= 0 || weight > 500 {
				progressStore.Unlock()
				b.sendMessage(chatID, b.t("progress_invalid_number", chatID))
				return
			}
			pState.Weight = weight
		}
		pState.Step = "measurements"
		progressStore.Unlock()

		setState(chatID, stateProgressMeasurements)
		b.askMeasurements(chatID)

	case stateProgressMeasurements:
		if text != "Пропустить" && text != "Skip" {
			b.parseMeasurements(pState, text)
		}
		pState.Step = "photo"
		progressStore.Unlock()

		setState(chatID, stateProgressPhoto)
		b.askPhoto(chatID)

	case stateProgressPhoto:
		// Фото обрабатывается в handleProgressPhoto
		if text == "Пропустить" || text == "Skip" {
			pState.Step = "notes"
			progressStore.Unlock()
			setState(chatID, stateProgressNotes)
			b.askNotes(chatID)
		} else {
			progressStore.Unlock()
			b.sendMessage(chatID, b.t("progress_send_photo", chatID))
		}

	case stateProgressNotes:
		if text != "Пропустить" && text != "Skip" {
			pState.Notes = text
		}
		progressStore.Unlock()

		b.saveProgress(chatID)
	}
}

// handleProgressPhoto обрабатывает фото прогресса
func (b *Bot) handleProgressPhoto(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	if message.Photo == nil || len(message.Photo) == 0 {
		return
	}

	// Берём фото максимального размера
	photo := message.Photo[len(message.Photo)-1]

	progressStore.Lock()
	pState := progressStore.data[chatID]
	if pState == nil {
		progressStore.Unlock()
		return
	}
	pState.PhotoFileID = photo.FileID
	pState.Step = "notes"
	progressStore.Unlock()

	setState(chatID, stateProgressNotes)
	b.askNotes(chatID)
}

// askMeasurements спрашивает замеры
func (b *Bot) askMeasurements(chatID int64) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Пропустить"),
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)

	text := `📏 *Введите замеры тела (см)*

Формат: грудь/талия/бёдра/бицепс/бедро
Пример: 100/80/95/35/55

Можно указать не все значения:
• 100/80/95 — только грудь, талия, бёдра
• /80/ — только талия

Или нажмите "Пропустить"`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// parseMeasurements парсит строку с замерами
func (b *Bot) parseMeasurements(pState *ProgressState, text string) {
	parts := strings.Split(text, "/")

	parseMeasurement := func(s string) float64 {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0
		}
		v, _ := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
		return v
	}

	if len(parts) >= 1 {
		pState.Chest = parseMeasurement(parts[0])
	}
	if len(parts) >= 2 {
		pState.Waist = parseMeasurement(parts[1])
	}
	if len(parts) >= 3 {
		pState.Hips = parseMeasurement(parts[2])
	}
	if len(parts) >= 4 {
		pState.Biceps = parseMeasurement(parts[3])
	}
	if len(parts) >= 5 {
		pState.Thigh = parseMeasurement(parts[4])
	}
}

// askPhoto спрашивает фото
func (b *Bot) askPhoto(chatID int64) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Пропустить"),
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📷 *Отправьте фото прогресса*\n\nИли нажмите \"Пропустить\"")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// askNotes спрашивает заметки
func (b *Bot) askNotes(chatID int64) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Пропустить"),
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📝 *Добавьте заметки* (опционально)\n\nНапример: \"Начал новую диету\" или \"Пропустил тренировки на неделе\"")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// saveProgress сохраняет прогресс в БД
func (b *Bot) saveProgress(chatID int64) {
	progressStore.RLock()
	pState := progressStore.data[chatID]
	progressStore.RUnlock()

	if pState == nil {
		b.cancelProgress(chatID)
		return
	}

	// Сохраняем в БД
	_, err := b.db.Exec(`
		INSERT INTO public.client_progress
		(client_id, record_date, weight, body_fat, chest, waist, hips, biceps, thigh, photo_file_id, notes)
		VALUES ($1, CURRENT_DATE, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		pState.ClientID, pState.Weight, pState.BodyFat,
		pState.Chest, pState.Waist, pState.Hips, pState.Biceps, pState.Thigh,
		pState.PhotoFileID, pState.Notes)

	if err != nil {
		log.Printf("Ошибка сохранения прогресса: %v", err)
		b.sendMessage(chatID, "❌ Ошибка сохранения прогресса")
	} else {
		b.sendProgressSummary(chatID, pState)
	}

	// Очищаем состояние
	progressStore.Lock()
	delete(progressStore.data, chatID)
	progressStore.Unlock()
	clearState(chatID)

	b.restoreMainMenu(chatID)
}

// sendProgressSummary отправляет сводку записанного прогресса
func (b *Bot) sendProgressSummary(chatID int64, pState *ProgressState) {
	var summary strings.Builder
	summary.WriteString("✅ *Прогресс записан!*\n\n")
	summary.WriteString(fmt.Sprintf("📅 Дата: %s\n\n", time.Now().Format("02.01.2006")))

	if pState.Weight > 0 {
		summary.WriteString(fmt.Sprintf("⚖️ Вес: %.1f кг\n", pState.Weight))
	}

	hasMeasurements := pState.Chest > 0 || pState.Waist > 0 || pState.Hips > 0 || pState.Biceps > 0 || pState.Thigh > 0
	if hasMeasurements {
		summary.WriteString("\n📏 *Замеры:*\n")
		if pState.Chest > 0 {
			summary.WriteString(fmt.Sprintf("  • Грудь: %.1f см\n", pState.Chest))
		}
		if pState.Waist > 0 {
			summary.WriteString(fmt.Sprintf("  • Талия: %.1f см\n", pState.Waist))
		}
		if pState.Hips > 0 {
			summary.WriteString(fmt.Sprintf("  • Бёдра: %.1f см\n", pState.Hips))
		}
		if pState.Biceps > 0 {
			summary.WriteString(fmt.Sprintf("  • Бицепс: %.1f см\n", pState.Biceps))
		}
		if pState.Thigh > 0 {
			summary.WriteString(fmt.Sprintf("  • Бедро: %.1f см\n", pState.Thigh))
		}
	}

	if pState.PhotoFileID != "" {
		summary.WriteString("\n📷 Фото сохранено\n")
	}

	if pState.Notes != "" {
		summary.WriteString(fmt.Sprintf("\n📝 Заметки: %s\n", pState.Notes))
	}

	msg := tgbotapi.NewMessage(chatID, summary.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// cancelProgress отменяет ввод прогресса
func (b *Bot) cancelProgress(chatID int64) {
	progressStore.Lock()
	delete(progressStore.data, chatID)
	progressStore.Unlock()
	clearState(chatID)

	b.sendMessage(chatID, "❌ "+b.t("cancelled", chatID))
	b.restoreMainMenu(chatID)
}

// handleViewProgress показывает историю прогресса клиента
func (b *Bot) handleViewProgress(chatID int64) {
	var clientID int
	err := b.db.QueryRow("SELECT id FROM public.clients WHERE telegram_id = $1", chatID).Scan(&clientID)
	if err != nil {
		b.sendMessage(chatID, b.t("reg_not_registered", chatID))
		return
	}

	rows, err := b.db.Query(`
		SELECT record_date, weight, chest, waist, hips, biceps, thigh, notes
		FROM public.client_progress
		WHERE client_id = $1
		ORDER BY record_date DESC
		LIMIT 10`, clientID)
	if err != nil {
		log.Printf("Ошибка получения прогресса: %v", err)
		b.sendMessage(chatID, "Ошибка загрузки прогресса")
		return
	}
	defer rows.Close()

	var entries []string
	for rows.Next() {
		var dateStr string
		var weight, chest, waist, hips, biceps, thigh float64
		var notes string

		if err := rows.Scan(&dateStr, &weight, &chest, &waist, &hips, &biceps, &thigh, &notes); err != nil {
			continue
		}

		date, _ := time.Parse("2006-01-02T15:04:05Z", dateStr)
		entry := fmt.Sprintf("📅 *%s*\n", date.Format("02.01.2006"))

		if weight > 0 {
			entry += fmt.Sprintf("  ⚖️ Вес: %.1f кг\n", weight)
		}
		if chest > 0 || waist > 0 || hips > 0 {
			entry += fmt.Sprintf("  📏 %s\n", formatMeasurements(chest, waist, hips, biceps, thigh))
		}
		if notes != "" {
			entry += fmt.Sprintf("  📝 %s\n", notes)
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		b.sendMessage(chatID, b.t("progress_no_data", chatID))
		return
	}

	message := "📊 *Ваш прогресс (последние 10 записей):*\n\n" + strings.Join(entries, "\n")
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// handleWeightDynamics показывает динамику веса
func (b *Bot) handleWeightDynamics(chatID int64) {
	var clientID int
	err := b.db.QueryRow("SELECT id FROM public.clients WHERE telegram_id = $1", chatID).Scan(&clientID)
	if err != nil {
		b.sendMessage(chatID, b.t("reg_not_registered", chatID))
		return
	}

	rows, err := b.db.Query(`
		SELECT record_date, weight
		FROM public.client_progress
		WHERE client_id = $1 AND weight > 0
		ORDER BY record_date DESC
		LIMIT 12`, clientID)
	if err != nil {
		log.Printf("Ошибка получения динамики веса: %v", err)
		b.sendMessage(chatID, "Ошибка загрузки данных")
		return
	}
	defer rows.Close()

	type weightEntry struct {
		date   time.Time
		weight float64
	}
	var entries []weightEntry

	for rows.Next() {
		var dateStr string
		var weight float64
		if err := rows.Scan(&dateStr, &weight); err != nil {
			continue
		}
		date, _ := time.Parse("2006-01-02T15:04:05Z", dateStr)
		entries = append(entries, weightEntry{date, weight})
	}

	if len(entries) == 0 {
		b.sendMessage(chatID, "📈 Недостаточно данных для отображения динамики веса.\n\nЗаписывайте вес регулярно!")
		return
	}

	var message strings.Builder
	message.WriteString("📈 *Динамика веса*\n\n")

	// Показываем график (текстовый)
	maxWeight := entries[0].weight
	minWeight := entries[0].weight
	for _, e := range entries {
		if e.weight > maxWeight {
			maxWeight = e.weight
		}
		if e.weight < minWeight {
			minWeight = e.weight
		}
	}

	// Переворачиваем для хронологического порядка
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		bar := generateWeightBar(e.weight, minWeight, maxWeight)
		message.WriteString(fmt.Sprintf("%s │%s %.1f кг\n", e.date.Format("02.01"), bar, e.weight))
	}

	// Статистика
	if len(entries) >= 2 {
		first := entries[len(entries)-1].weight
		last := entries[0].weight
		diff := last - first

		message.WriteString("\n📊 *Статистика:*\n")
		message.WriteString(fmt.Sprintf("  Начало: %.1f кг\n", first))
		message.WriteString(fmt.Sprintf("  Сейчас: %.1f кг\n", last))

		if diff > 0 {
			message.WriteString(fmt.Sprintf("  Изменение: +%.1f кг ⬆️\n", diff))
		} else if diff < 0 {
			message.WriteString(fmt.Sprintf("  Изменение: %.1f кг ⬇️\n", diff))
		} else {
			message.WriteString("  Изменение: 0 кг ➡️\n")
		}
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// generateWeightBar генерирует текстовую полоску для графика
func generateWeightBar(value, min, max float64) string {
	if max == min {
		return "████████"
	}
	ratio := (value - min) / (max - min)
	length := int(ratio*10) + 1
	if length > 10 {
		length = 10
	}
	return strings.Repeat("█", length) + strings.Repeat("░", 10-length)
}

// handleMeasurementsDynamics показывает динамику замеров
func (b *Bot) handleMeasurementsDynamics(chatID int64) {
	var clientID int
	err := b.db.QueryRow("SELECT id FROM public.clients WHERE telegram_id = $1", chatID).Scan(&clientID)
	if err != nil {
		b.sendMessage(chatID, b.t("reg_not_registered", chatID))
		return
	}

	// Получаем первую и последнюю записи с замерами
	var firstDate, lastDate string
	var firstChest, firstWaist, firstHips, lastChest, lastWaist, lastHips float64

	// Первая запись
	err = b.db.QueryRow(`
		SELECT record_date, COALESCE(chest, 0), COALESCE(waist, 0), COALESCE(hips, 0)
		FROM public.client_progress
		WHERE client_id = $1 AND (chest > 0 OR waist > 0 OR hips > 0)
		ORDER BY record_date ASC
		LIMIT 1`, clientID).Scan(&firstDate, &firstChest, &firstWaist, &firstHips)
	if err != nil {
		b.sendMessage(chatID, "📏 Недостаточно данных для отображения динамики замеров.\n\nЗаписывайте замеры регулярно!")
		return
	}

	// Последняя запись
	err = b.db.QueryRow(`
		SELECT record_date, COALESCE(chest, 0), COALESCE(waist, 0), COALESCE(hips, 0)
		FROM public.client_progress
		WHERE client_id = $1 AND (chest > 0 OR waist > 0 OR hips > 0)
		ORDER BY record_date DESC
		LIMIT 1`, clientID).Scan(&lastDate, &lastChest, &lastWaist, &lastHips)
	if err != nil {
		b.sendMessage(chatID, "Ошибка загрузки данных")
		return
	}

	firstTime, _ := time.Parse("2006-01-02T15:04:05Z", firstDate)
	lastTime, _ := time.Parse("2006-01-02T15:04:05Z", lastDate)

	var message strings.Builder
	message.WriteString("📏 *Динамика замеров*\n\n")
	message.WriteString(fmt.Sprintf("📅 Период: %s — %s\n\n", firstTime.Format("02.01.2006"), lastTime.Format("02.01.2006")))

	message.WriteString("```\n")
	message.WriteString("           Было    Стало   Разница\n")
	message.WriteString("─────────────────────────────────\n")

	if firstChest > 0 || lastChest > 0 {
		diff := lastChest - firstChest
		diffStr := formatDiff(diff)
		message.WriteString(fmt.Sprintf("Грудь    %5.1f   %5.1f   %s\n", firstChest, lastChest, diffStr))
	}
	if firstWaist > 0 || lastWaist > 0 {
		diff := lastWaist - firstWaist
		diffStr := formatDiff(diff)
		message.WriteString(fmt.Sprintf("Талия    %5.1f   %5.1f   %s\n", firstWaist, lastWaist, diffStr))
	}
	if firstHips > 0 || lastHips > 0 {
		diff := lastHips - firstHips
		diffStr := formatDiff(diff)
		message.WriteString(fmt.Sprintf("Бёдра    %5.1f   %5.1f   %s\n", firstHips, lastHips, diffStr))
	}
	message.WriteString("```")

	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// formatDiff форматирует разницу
func formatDiff(diff float64) string {
	if diff > 0 {
		return fmt.Sprintf("+%.1f ↑", diff)
	} else if diff < 0 {
		return fmt.Sprintf("%.1f ↓", diff)
	}
	return "  0  →"
}

// formatMeasurements форматирует замеры в строку
func formatMeasurements(chest, waist, hips, biceps, thigh float64) string {
	var parts []string
	if chest > 0 {
		parts = append(parts, fmt.Sprintf("Гр:%.0f", chest))
	}
	if waist > 0 {
		parts = append(parts, fmt.Sprintf("Т:%.0f", waist))
	}
	if hips > 0 {
		parts = append(parts, fmt.Sprintf("Б:%.0f", hips))
	}
	if biceps > 0 {
		parts = append(parts, fmt.Sprintf("Би:%.0f", biceps))
	}
	if thigh > 0 {
		parts = append(parts, fmt.Sprintf("Бд:%.0f", thigh))
	}
	return strings.Join(parts, "/")
}

// === Прогресс по программе тренировок для клиентов ===

// handleClientProgramProgress показывает клиенту прогресс по его программе
func (b *Bot) handleClientProgramProgress(chatID int64) {
	// Получаем ID клиента
	clientID, err := b.repo.Program.GetClientByTelegramID(chatID)
	if err != nil || clientID == 0 {
		b.sendMessage(chatID, b.t("reg_not_registered", chatID))
		return
	}

	// Получаем прогресс программы
	progress, err := b.repo.Program.GetProgramProgress(clientID)
	if err != nil {
		b.sendMessage(chatID, "Ошибка загрузки прогресса программы")
		return
	}

	if progress == nil {
		b.sendMessage(chatID, "🏋️ У вас пока нет активной программы тренировок.\n\nОбратитесь к тренеру для получения программы!")
		return
	}

	// Формируем прогресс-бар
	progressBar := makeProgressBar(progress.ProgressPercent, 10)

	// Формируем текст
	var text strings.Builder
	text.WriteString("🏋️ *Прогресс программы*\n\n")
	text.WriteString(fmt.Sprintf("📋 *%s*\n", progress.ProgramName))
	if progress.Goal != "" {
		text.WriteString(fmt.Sprintf("🎯 Цель: %s\n", progress.Goal))
	}
	text.WriteString("\n")

	// Прогресс-бар и проценты
	text.WriteString(fmt.Sprintf("*Выполнено: %.0f%%*\n", progress.ProgressPercent))
	text.WriteString(progressBar)
	text.WriteString("\n\n")

	// Статистика по неделям
	text.WriteString(fmt.Sprintf("📅 *Неделя:* %d из %d\n", progress.CurrentWeek, progress.TotalWeeks))
	text.WriteString(fmt.Sprintf("🗓️ *Тренировок в неделю:* %d\n\n", progress.DaysPerWeek))

	// Статистика по тренировкам
	text.WriteString("*Статистика тренировок:*\n")
	text.WriteString(fmt.Sprintf("✅ Выполнено: %d\n", progress.CompletedCount))
	if progress.SentCount > 0 {
		text.WriteString(fmt.Sprintf("📤 Ожидает выполнения: %d\n", progress.SentCount))
	}
	text.WriteString(fmt.Sprintf("⏳ Впереди: %d\n", progress.PendingCount))
	if progress.SkippedCount > 0 {
		text.WriteString(fmt.Sprintf("⏭️ Пропущено: %d\n", progress.SkippedCount))
	}

	// Следующая тренировка
	if progress.NextWorkout != nil {
		text.WriteString(fmt.Sprintf("\n📌 *Следующая тренировка:*\n%s (Неделя %d, День %d)\n",
			progress.NextWorkout.Name, progress.NextWorkout.WeekNum, progress.NextWorkout.DayNum))

		if progress.NextWorkout.Status == "sent" {
			text.WriteString("\n💪 Тренировка уже отправлена — напиши /workouts чтобы начать!")
		}
	} else if progress.PendingCount == 0 && progress.SentCount == 0 {
		text.WriteString("\n\n🎉 *Поздравляем!*\nВы выполнили все тренировки программы! 🏆")
	}

	// Мотивация
	if progress.ProgressPercent > 0 && progress.ProgressPercent < 100 {
		text.WriteString("\n\n")
		if progress.ProgressPercent < 25 {
			text.WriteString("🚀 Отличное начало! Продолжайте в том же духе!")
		} else if progress.ProgressPercent < 50 {
			text.WriteString("💪 Вы на правильном пути! Уже почти половина!")
		} else if progress.ProgressPercent < 75 {
			text.WriteString("🔥 Больше половины позади! Не сдавайтесь!")
		} else {
			text.WriteString("🏆 Финишная прямая! Ещё немного до цели!")
		}
	}

	msg := tgbotapi.NewMessage(chatID, text.String())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// === Функции для тренера (просмотр прогресса клиента) ===

// handleAdminViewClientProgress показывает прогресс выбранного клиента для тренера
func (b *Bot) handleAdminViewClientProgress(chatID int64, clientID int) {
	var name, surname string
	err := b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).Scan(&name, &surname)
	if err != nil {
		b.sendMessage(chatID, "Клиент не найден")
		return
	}

	rows, err := b.db.Query(`
		SELECT record_date, weight, chest, waist, hips, biceps, thigh, photo_file_id, notes
		FROM public.client_progress
		WHERE client_id = $1
		ORDER BY record_date DESC
		LIMIT 10`, clientID)
	if err != nil {
		log.Printf("Ошибка получения прогресса клиента: %v", err)
		b.sendMessage(chatID, "Ошибка загрузки прогресса")
		return
	}
	defer rows.Close()

	var entries []string
	var photos []string

	for rows.Next() {
		var dateStr string
		var weight, chest, waist, hips, biceps, thigh float64
		var photoFileID, notes string

		if err := rows.Scan(&dateStr, &weight, &chest, &waist, &hips, &biceps, &thigh, &photoFileID, &notes); err != nil {
			continue
		}

		date, _ := time.Parse("2006-01-02T15:04:05Z", dateStr)
		entry := fmt.Sprintf("📅 *%s*\n", date.Format("02.01.2006"))

		if weight > 0 {
			entry += fmt.Sprintf("  ⚖️ Вес: %.1f кг\n", weight)
		}
		if chest > 0 || waist > 0 || hips > 0 {
			entry += fmt.Sprintf("  📏 %s\n", formatMeasurements(chest, waist, hips, biceps, thigh))
		}
		if notes != "" {
			entry += fmt.Sprintf("  📝 %s\n", notes)
		}
		if photoFileID != "" {
			entry += "  📷 Есть фото\n"
			photos = append(photos, photoFileID)
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		b.sendMessage(chatID, fmt.Sprintf("📊 У клиента %s %s пока нет записей прогресса", name, surname))
		return
	}

	message := fmt.Sprintf("📊 *Прогресс клиента %s %s*\n\n", name, surname) + strings.Join(entries, "\n")
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)

	// Отправляем фотографии (последние 3)
	if len(photos) > 0 {
		count := 3
		if len(photos) < count {
			count = len(photos)
		}
		for i := 0; i < count; i++ {
			photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(photos[i]))
			photo.Caption = fmt.Sprintf("Фото прогресса #%d", i+1)
			b.api.Send(photo)
		}
	}
}

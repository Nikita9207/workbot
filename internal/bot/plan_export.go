package bot

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"workbot/internal/excel"
	"workbot/internal/models"
)

// showPlansForExport shows plans available for export
func (b *Bot) showPlansForExport(chatID int64) {
	rows, err := b.db.Query(`
		SELECT tp.id, tp.name, c.name || ' ' || c.surname as client_name
		FROM public.training_plans tp
		JOIN public.clients c ON tp.client_id = c.id
		WHERE tp.status = 'active'
		ORDER BY tp.created_at DESC
		LIMIT 15`)
	if err != nil {
		log.Printf("Ошибка получения планов: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки планов")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id int
		var name, clientName string
		if err := rows.Scan(&id, &name, &clientName); err != nil {
			continue
		}
		buttonText := fmt.Sprintf("EXP>> %s [%d]", name, id)
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет активных планов для экспорта. Сначала создайте план.")
		b.api.Send(msg)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	userStates.Lock()
	userStates.states[chatID] = statePlanExportSelect
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Выберите план для экспорта в Excel:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// handlePlanExportSelect handles plan selection for export
func (b *Bot) handlePlanExportSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearPlanState(chatID)
		b.handlePlansMenu(message)
		return
	}

	// Parse plan ID from "EXP>> Name [ID]"
	planID := parsePlanExportID(text)
	if planID == 0 {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора плана")
		b.api.Send(msg)
		return
	}

	b.exportPlanToExcel(chatID, planID, message)
}

// exportPlanToExcel exports plan to Excel and sends file
func (b *Bot) exportPlanToExcel(chatID int64, planID int, originalMessage *tgbotapi.Message) {
	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Генерирую Excel файл...")
	b.api.Send(waitMsg)

	// Load plan from database
	plan, err := b.loadPlanForExport(planID)
	if err != nil {
		log.Printf("Ошибка загрузки плана: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки плана")
		b.api.Send(msg)
		return
	}

	// Load progression (for exercises with 1PM)
	progression, err := b.loadProgressionForExport(planID)
	if err != nil {
		log.Printf("Ошибка загрузки прогрессии: %v", err)
	}

	// Load 1PM history for client
	pm1History, err := b.load1PMHistoryForExport(plan.ClientID)
	if err != nil {
		log.Printf("Ошибка загрузки 1ПМ: %v", err)
	}

	// Load full workout program
	workouts, err := b.loadWorkoutsForExport(plan.ClientID, plan.Name)
	if err != nil {
		log.Printf("Ошибка загрузки тренировок: %v (продолжаем без них)", err)
		workouts = nil
	}

	// Generate Excel file with full workouts
	f, err := excel.ExportTrainingPlanWithWorkouts(plan, progression, pm1History, workouts)
	if err != nil {
		log.Printf("Ошибка генерации Excel: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка создания Excel файла")
		b.api.Send(msg)
		return
	}

	// Save to temp file
	filename := excel.GeneratePlanFilename(plan.Name, plan.ClientName)
	tempPath := "/tmp/" + filename

	if err := f.SaveAs(tempPath); err != nil {
		log.Printf("Ошибка сохранения файла: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения файла")
		b.api.Send(msg)
		return
	}
	defer os.Remove(tempPath)

	// Send file to Telegram
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(tempPath))
	caption := fmt.Sprintf("📊 План: %s\n📅 %d недель | %d тренировок/нед", plan.Name, plan.TotalWeeks, plan.DaysPerWeek)
	if workouts != nil && len(workouts.Weeks) > 0 {
		totalEx := 0
		for _, w := range workouts.Weeks {
			for _, d := range w.Days {
				totalEx += len(d.Exercises)
			}
		}
		caption += fmt.Sprintf("\n🏋️ %d упражнений", totalEx)
	}
	doc.Caption = caption
	if _, err := b.api.Send(doc); err != nil {
		log.Printf("Ошибка отправки файла: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка отправки файла")
		b.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "✅ Excel файл отправлен!")
	b.api.Send(msg)

	b.clearPlanState(chatID)
	b.handlePlansMenu(originalMessage)
}

// loadPlanForExport loads training plan with mesocycles
func (b *Bot) loadPlanForExport(planID int) (*models.TrainingPlan, error) {
	plan := &models.TrainingPlan{}

	err := b.db.QueryRow(`
		SELECT tp.id, tp.client_id, tp.name, tp.start_date, tp.end_date,
			tp.status, tp.goal, tp.days_per_week, tp.total_weeks,
			c.name || ' ' || c.surname as client_name
		FROM public.training_plans tp
		JOIN public.clients c ON tp.client_id = c.id
		WHERE tp.id = $1`, planID).Scan(
		&plan.ID, &plan.ClientID, &plan.Name, &plan.StartDate, &plan.EndDate,
		&plan.Status, &plan.Goal, &plan.DaysPerWeek, &plan.TotalWeeks,
		&plan.ClientName,
	)
	if err != nil {
		return nil, fmt.Errorf("план не найден: %w", err)
	}

	// Load mesocycles
	rows, err := b.db.Query(`
		SELECT id, name, phase, week_start, week_end, intensity_percent, volume_percent, rpe_target, order_num
		FROM public.mesocycles
		WHERE training_plan_id = $1
		ORDER BY order_num`, planID)
	if err != nil {
		return plan, nil // return plan without mesocycles
	}
	defer rows.Close()

	for rows.Next() {
		var meso models.Mesocycle
		var phase string
		if err := rows.Scan(&meso.ID, &meso.Name, &phase, &meso.WeekStart, &meso.WeekEnd,
			&meso.IntensityPercent, &meso.VolumePercent, &meso.RPETarget, &meso.OrderNum); err != nil {
			continue
		}
		meso.Phase = models.PlanPhase(phase)

		// Load microcycles for this mesocycle
		microRows, _ := b.db.Query(`
			SELECT week_number, name, is_deload, volume_modifier, intensity_modifier
			FROM public.microcycles
			WHERE mesocycle_id = $1
			ORDER BY week_number`, meso.ID)
		if microRows != nil {
			for microRows.Next() {
				var micro models.Microcycle
				microRows.Scan(&micro.WeekNumber, &micro.Name, &micro.IsDeload,
					&micro.VolumeModifier, &micro.IntensityModifier)
				meso.Microcycles = append(meso.Microcycles, micro)
			}
			microRows.Close()
		}

		plan.Mesocycles = append(plan.Mesocycles, meso)
	}

	return plan, nil
}

// loadProgressionForExport loads progression data
func (b *Bot) loadProgressionForExport(planID int) ([]models.Progression, error) {
	rows, err := b.db.Query(`
		SELECT pp.exercise_id, e.name, pp.week_number, pp.sets, pp.reps,
			pp.weight_kg, pp.intensity_percent, pp.is_deload
		FROM public.plan_progression pp
		JOIN public.exercises e ON pp.exercise_id = e.id
		WHERE pp.training_plan_id = $1
		ORDER BY e.name, pp.week_number`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progression []models.Progression
	for rows.Next() {
		var p models.Progression
		if err := rows.Scan(&p.ExerciseID, &p.ExerciseName, &p.WeekNumber, &p.Sets, &p.Reps,
			&p.WeightKg, &p.IntensityPercent, &p.IsDeload); err != nil {
			continue
		}
		progression = append(progression, p)
	}

	return progression, nil
}

// load1PMHistoryForExport loads 1PM history for client
func (b *Bot) load1PMHistoryForExport(clientID int) ([]models.Exercise1PMHistory, error) {
	// Get exercises with 1PM records
	rows, err := b.db.Query(`
		SELECT DISTINCT e.id, e.name
		FROM public.exercises e
		JOIN public.exercise_1pm pm ON pm.exercise_id = e.id
		WHERE pm.client_id = $1
		ORDER BY e.name`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.Exercise1PMHistory
	for rows.Next() {
		var exID int
		var exName string
		if err := rows.Scan(&exID, &exName); err != nil {
			continue
		}

		h := models.Exercise1PMHistory{
			ExerciseID:   exID,
			ExerciseName: exName,
		}

		// Get all records for this exercise
		recordRows, _ := b.db.Query(`
			SELECT one_pm_kg, test_date, calc_method, COALESCE(source_weight, 0), COALESCE(source_reps, 0)
			FROM public.exercise_1pm
			WHERE client_id = $1 AND exercise_id = $2
			ORDER BY test_date ASC`, clientID, exID)
		if recordRows != nil {
			var firstPM, lastPM float64
			for recordRows.Next() {
				var r models.Exercise1PM
				recordRows.Scan(&r.OnePMKg, &r.TestDate, &r.CalcMethod, &r.SourceWeight, &r.SourceReps)
				r.ExerciseID = exID
				r.ClientID = clientID
				h.Records = append(h.Records, r)

				if firstPM == 0 {
					firstPM = r.OnePMKg
				}
				lastPM = r.OnePMKg
			}
			recordRows.Close()

			h.InitialPM = firstPM
			h.CurrentPM = lastPM
			h.GainKg = lastPM - firstPM
			if firstPM > 0 {
				h.GainPercent = (lastPM - firstPM) / firstPM * 100
			}
		}

		history = append(history, h)
	}

	return history, nil
}

// loadWorkoutsForExport loads full workout program for a plan
func (b *Bot) loadWorkoutsForExport(clientID int, planName string) (*models.GeneratedProgram, error) {
	// Find the training program by client_id and name
	var programID int
	var goal string
	var totalWeeks, daysPerWeek int
	err := b.db.QueryRow(`
		SELECT id, COALESCE(goal, ''), total_weeks, days_per_week
		FROM public.training_programs
		WHERE client_id = $1 AND name = $2
		ORDER BY created_at DESC LIMIT 1`, clientID, planName).
		Scan(&programID, &goal, &totalWeeks, &daysPerWeek)

	if err != nil {
		return nil, fmt.Errorf("программа не найдена: %w", err)
	}

	program := &models.GeneratedProgram{
		ClientID:    clientID,
		Goal:        models.TrainingGoal(goal),
		TotalWeeks:  totalWeeks,
		DaysPerWeek: daysPerWeek,
	}

	// Load workouts
	workoutRows, err := b.db.Query(`
		SELECT id, week_num, day_num, name
		FROM public.program_workouts
		WHERE program_id = $1
		ORDER BY week_num, day_num`, programID)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки тренировок: %w", err)
	}
	defer workoutRows.Close()

	// Group workouts by week
	weekMap := make(map[int]*models.GeneratedWeek)

	for workoutRows.Next() {
		var workoutID, weekNum, dayNum int
		var name string
		if err := workoutRows.Scan(&workoutID, &weekNum, &dayNum, &name); err != nil {
			continue
		}

		// Get or create week
		week, exists := weekMap[weekNum]
		if !exists {
			week = &models.GeneratedWeek{
				WeekNum: weekNum,
			}
			weekMap[weekNum] = week
		}

		// Create day
		day := models.GeneratedDay{
			DayNum: dayNum,
			Name:   name,
		}

		// Load exercises for this workout
		exRows, err := b.db.Query(`
			SELECT order_num, exercise_name, sets, reps,
				COALESCE(weight, 0), COALESCE(weight_percent, 0),
				COALESCE(rest_seconds, 90), COALESCE(rpe, 0), COALESCE(notes, '')
			FROM public.workout_exercises
			WHERE workout_id = $1
			ORDER BY order_num`, workoutID)
		if err == nil {
			for exRows.Next() {
				var ex models.GeneratedExercise
				exRows.Scan(&ex.OrderNum, &ex.ExerciseName, &ex.Sets, &ex.Reps,
					&ex.Weight, &ex.WeightPercent, &ex.RestSeconds, &ex.RPE, &ex.Notes)
				day.Exercises = append(day.Exercises, ex)
			}
			exRows.Close()
		}

		week.Days = append(week.Days, day)
	}

	// Convert map to slice
	for i := 1; i <= totalWeeks; i++ {
		if week, exists := weekMap[i]; exists {
			program.Weeks = append(program.Weeks, *week)
		}
	}

	return program, nil
}

// parsePlanExportID extracts plan ID from button text
func parsePlanExportID(text string) int {
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		return 0
	}
	id, _ := strconv.Atoi(text[start+1 : end])
	return id
}

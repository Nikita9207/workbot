package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"workbot/internal/generator"
	"workbot/internal/models"
	"workbot/internal/training"
)

// States for training plan flow
const (
	statePlanMenu           = "plan_menu"
	statePlanSelectClient   = "plan_select_client"
	statePlanSelectGoal     = "plan_select_goal"
	statePlanSelectDuration = "plan_select_duration"
	statePlanSelectDays     = "plan_select_days"
	statePlanConfirm        = "plan_confirm"
	statePlanViewSelect     = "plan_view_select"
	statePlanExportSelect   = "plan_export_select"
)

// planStore stores temporary data for plan creation
var planStore = struct {
	sync.RWMutex
	clientID    map[int64]int
	goal        map[int64]string
	weeks       map[int64]int
	daysPerWeek map[int64]int
	planName    map[int64]string
}{
	clientID:    make(map[int64]int),
	goal:        make(map[int64]string),
	weeks:       make(map[int64]int),
	daysPerWeek: make(map[int64]int),
	planName:    make(map[int64]string),
}

// handlePlansMenu shows training plans menu
func (b *Bot) handlePlansMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	userStates.Lock()
	userStates.states[chatID] = statePlanMenu
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "📋 Тренировочные планы\n\nВыберите действие:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Создать план"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Просмотр планов"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Экспорт в Excel"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePlanMenuChoice handles plan menu selection
func (b *Bot) handlePlanMenuChoice(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	switch text {
	case "Создать план":
		userStates.Lock()
		userStates.states[chatID] = statePlanSelectClient
		userStates.Unlock()
		b.showClientsForPlan(chatID, "Выберите клиента для создания плана:")

	case "Просмотр планов":
		b.showPlansList(chatID)

	case "Экспорт в Excel":
		b.showPlansForExport(chatID)

	case "Назад":
		b.clearPlanState(chatID)
		b.handleAdminStart(message)

	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите действие из меню")
		b.api.Send(msg)
	}
}

// showClientsForPlan shows client list for plan creation
func (b *Bot) showClientsForPlan(chatID int64, text string) {
	rows, err := b.db.Query(`
		SELECT c.id, c.name, c.surname,
			(SELECT COUNT(*) FROM public.exercise_1pm WHERE client_id = c.id) as pm_count
		FROM public.clients c
		WHERE c.deleted_at IS NULL
		ORDER BY c.name, c.surname`)
	if err != nil {
		log.Printf("Ошибка получения клиентов: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки списка клиентов")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id, pmCount int
		var name, surname string
		if err := rows.Scan(&id, &name, &surname, &pmCount); err != nil {
			continue
		}
		buttonText := fmt.Sprintf("PLAN>> %s %s", name, surname)
		if pmCount > 0 {
			buttonText += fmt.Sprintf(" (%d 1ПМ)", pmCount)
		}
		buttonText += fmt.Sprintf(" [%d]", id)
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет клиентов. Сначала добавьте клиента через меню клиентов.")
		b.api.Send(msg)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(buttons...)
	b.api.Send(msg)
}

// handlePlanClientSelect handles client selection for plan
func (b *Bot) handlePlanClientSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearPlanState(chatID)
		b.handlePlansMenu(message)
		return
	}

	// Parse client ID
	clientID := parsePlanClientID(text)
	if clientID == 0 {
		msg := tgbotapi.NewMessage(chatID, "Ошибка выбора клиента")
		b.api.Send(msg)
		return
	}

	// Check if client has 1PM data
	var pmCount int
	b.db.QueryRow("SELECT COUNT(*) FROM public.exercise_1pm WHERE client_id = $1", clientID).Scan(&pmCount)

	if pmCount == 0 {
		msg := tgbotapi.NewMessage(chatID,
			"⚠️ У клиента нет записей 1ПМ.\n\n"+
				"Для создания плана с прогрессией рекомендуется сначала записать 1ПМ.\n\n"+
				"Продолжить без 1ПМ? (веса будут указаны в %)")
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Да, продолжить"),
				tgbotapi.NewKeyboardButton("Записать 1ПМ"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Отмена"),
			),
		)
		msg.ReplyMarkup = keyboard

		planStore.Lock()
		planStore.clientID[chatID] = clientID
		planStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = "plan_no_1pm_confirm"
		userStates.Unlock()

		b.api.Send(msg)
		return
	}

	planStore.Lock()
	planStore.clientID[chatID] = clientID
	planStore.Unlock()

	b.showPlanGoalSelection(chatID)
}

// handlePlanNo1PMConfirm handles confirmation when no 1PM data
func (b *Bot) handlePlanNo1PMConfirm(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	// Получаем clientID до очистки состояния
	planStore.RLock()
	clientID := planStore.clientID[chatID]
	planStore.RUnlock()

	switch text {
	case "Да, продолжить":
		b.showPlanGoalSelection(chatID)
	case "Записать 1ПМ":
		// Не очищаем planState — переходим к записи 1ПМ с флагом возврата
		b.handle1PMForClient(chatID, clientID, true)
		return
	case "Отмена":
		b.clearPlanState(chatID)
		b.handlePlansMenu(message)
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите вариант из меню")
		b.api.Send(msg)
	}
}

// showPlanGoalSelection shows goal selection
func (b *Bot) showPlanGoalSelection(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = statePlanSelectGoal
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Выберите цель программы:")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💪 Сила"),
			tgbotapi.NewKeyboardButton("🏋️ Масса"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔥 Похудение"),
			tgbotapi.NewKeyboardButton("🏆 Соревнования"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePlanGoalSelect handles goal selection
func (b *Bot) handlePlanGoalSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearPlanState(chatID)
		b.handlePlansMenu(message)
		return
	}

	var goal string
	switch text {
	case "💪 Сила":
		goal = "strength"
	case "🏋️ Масса":
		goal = "hypertrophy"
	case "🔥 Похудение":
		goal = "weight_loss"
	case "🏆 Соревнования":
		goal = "competition"
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите цель из меню")
		b.api.Send(msg)
		return
	}

	planStore.Lock()
	planStore.goal[chatID] = goal
	planStore.Unlock()

	b.showPlanDurationSelection(chatID)
}

// showPlanDurationSelection shows duration selection
func (b *Bot) showPlanDurationSelection(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = statePlanSelectDuration
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "На сколько недель составить план?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4 недели"),
			tgbotapi.NewKeyboardButton("8 недель"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("12 недель"),
			tgbotapi.NewKeyboardButton("16 недель"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePlanDurationSelect handles duration selection
func (b *Bot) handlePlanDurationSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearPlanState(chatID)
		b.handlePlansMenu(message)
		return
	}

	var weeks int
	switch text {
	case "4 недели":
		weeks = 4
	case "8 недель":
		weeks = 8
	case "12 недель":
		weeks = 12
	case "16 недель":
		weeks = 16
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите из предложенных вариантов")
		b.api.Send(msg)
		return
	}

	planStore.Lock()
	planStore.weeks[chatID] = weeks
	planStore.Unlock()

	b.showPlanDaysSelection(chatID)
}

// showPlanDaysSelection shows days per week selection
func (b *Bot) showPlanDaysSelection(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = statePlanSelectDays
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Сколько тренировок в неделю?")
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("2 дня"),
			tgbotapi.NewKeyboardButton("3 дня"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("4 дня"),
			tgbotapi.NewKeyboardButton("5 дней"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("6 дней"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePlanDaysSelect handles days selection
func (b *Bot) handlePlanDaysSelect(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.clearPlanState(chatID)
		b.handlePlansMenu(message)
		return
	}

	var days int
	switch text {
	case "2 дня":
		days = 2
	case "3 дня":
		days = 3
	case "4 дня":
		days = 4
	case "5 дней":
		days = 5
	case "6 дней":
		days = 6
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите из предложенных вариантов")
		b.api.Send(msg)
		return
	}

	planStore.Lock()
	planStore.daysPerWeek[chatID] = days
	planStore.Unlock()

	b.showPlanConfirmation(chatID)
}

// showPlanConfirmation shows plan parameters before creating
func (b *Bot) showPlanConfirmation(chatID int64) {
	userStates.Lock()
	userStates.states[chatID] = statePlanConfirm
	userStates.Unlock()

	planStore.RLock()
	clientID := planStore.clientID[chatID]
	goal := planStore.goal[chatID]
	weeks := planStore.weeks[chatID]
	days := planStore.daysPerWeek[chatID]
	planStore.RUnlock()

	// Get client name
	var clientName string
	b.db.QueryRow("SELECT name || ' ' || surname FROM public.clients WHERE id = $1", clientID).Scan(&clientName)

	// Get goal name
	goalName := training.PeriodizationTemplates[goal].Name

	text := fmt.Sprintf("📋 Подтверждение создания плана\n\n"+
		"👤 Клиент: %s\n"+
		"🎯 Цель: %s\n"+
		"📅 Длительность: %d недель\n"+
		"🏋️ Тренировок в неделю: %d\n\n"+
		"Создать план?",
		clientName, goalName, weeks, days)

	msg := tgbotapi.NewMessage(chatID, text)
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ Создать план"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Отмена"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handlePlanConfirm handles plan creation confirmation
func (b *Bot) handlePlanConfirm(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	switch text {
	case "✅ Создать план":
		b.createTrainingPlan(chatID, message)
	case "❌ Отмена":
		b.clearPlanState(chatID)
		b.handlePlansMenu(message)
	default:
		msg := tgbotapi.NewMessage(chatID, "Выберите действие из меню")
		b.api.Send(msg)
	}
}

// createTrainingPlan creates the training plan in database with full workout generation
func (b *Bot) createTrainingPlan(chatID int64, message *tgbotapi.Message) {
	planStore.RLock()
	clientID := planStore.clientID[chatID]
	goal := planStore.goal[chatID]
	weeks := planStore.weeks[chatID]
	days := planStore.daysPerWeek[chatID]
	planStore.RUnlock()

	// Get client name
	var clientName string
	b.db.QueryRow("SELECT name || ' ' || surname FROM public.clients WHERE id = $1", clientID).Scan(&clientName)

	goalName := training.PeriodizationTemplates[goal].Name
	planName := fmt.Sprintf("%s - %s", clientName, goalName)

	waitMsg := tgbotapi.NewMessage(chatID, "⏳ Создаю план тренировок с полной программой...")
	b.api.Send(waitMsg)

	// Load client profile for generator
	client, err := b.loadClientProfile(clientID)
	if err != nil {
		log.Printf("Ошибка загрузки профиля клиента: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки данных клиента")
		b.api.Send(msg)
		return
	}

	// Generate full workout program using the generator
	var program *models.GeneratedProgram
	selector, _ := generator.NewExerciseSelector("data")

	switch goal {
	case "strength", "competition":
		gen := generator.NewStrengthGenerator(selector, client)
		program, err = gen.Generate(generator.StrengthConfig{
			TotalWeeks:  weeks,
			DaysPerWeek: days,
			Focus:       "all",
		})
	case "hypertrophy":
		gen := generator.NewHypertrophyGenerator(selector, client)
		program, err = gen.Generate(generator.HypertrophyConfig{
			TotalWeeks:  weeks,
			DaysPerWeek: days,
			Split:       generator.GetDefaultSplit(days),
		})
	case "weight_loss":
		gen := generator.NewFatLossGenerator(selector, client)
		program, err = gen.Generate(generator.FatLossConfig{
			TotalWeeks:  weeks,
			DaysPerWeek: days,
			IncludeHIIT: true,
		})
	default:
		// По умолчанию - силовая
		gen := generator.NewStrengthGenerator(selector, client)
		program, err = gen.Generate(generator.StrengthConfig{
			TotalWeeks:  weeks,
			DaysPerWeek: days,
			Focus:       "all",
		})
	}

	if err != nil {
		log.Printf("Ошибка генерации программы: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка генерации программы")
		b.api.Send(msg)
		return
	}

	// Generate periodization structure
	plan := training.GenerateFullPeriodization(
		clientID,
		planName,
		time.Now(),
		weeks,
		days,
		goal,
		4, // deload every 4 weeks
	)

	// Save to database
	tx, err := b.db.Begin()
	if err != nil {
		log.Printf("Ошибка транзакции: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка создания плана")
		b.api.Send(msg)
		return
	}

	// Insert training plan
	var planID int
	err = tx.QueryRow(`
		INSERT INTO public.training_plans
			(client_id, name, start_date, end_date, status, goal, days_per_week, total_weeks, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		clientID, planName, plan.StartDate, plan.EndDate, "active", goal, days, weeks, chatID,
	).Scan(&planID)

	if err != nil {
		tx.Rollback()
		log.Printf("Ошибка создания плана: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения плана")
		b.api.Send(msg)
		return
	}

	// Insert mesocycles and microcycles
	for _, meso := range plan.Mesocycles {
		var mesoID int
		err = tx.QueryRow(`
			INSERT INTO public.mesocycles
				(training_plan_id, name, phase, week_start, week_end, intensity_percent, volume_percent, rpe_target, order_num)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id`,
			planID, meso.Name, meso.Phase, meso.WeekStart, meso.WeekEnd,
			meso.IntensityPercent, meso.VolumePercent, meso.RPETarget, meso.OrderNum,
		).Scan(&mesoID)

		if err != nil {
			tx.Rollback()
			log.Printf("Ошибка создания мезоцикла: %v", err)
			msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения периодизации")
			b.api.Send(msg)
			return
		}

		// Insert microcycles
		for _, micro := range meso.Microcycles {
			_, err = tx.Exec(`
				INSERT INTO public.microcycles
					(mesocycle_id, week_number, name, is_deload, volume_modifier, intensity_modifier)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				mesoID, micro.WeekNumber, micro.Name, micro.IsDeload,
				micro.VolumeModifier, micro.IntensityModifier,
			)
			if err != nil {
				tx.Rollback()
				log.Printf("Ошибка создания микроцикла: %v", err)
				return
			}
		}
	}

	// Save generated program to training_programs table
	var programID int
	startDate := time.Now()
	endDate := startDate.AddDate(0, 0, weeks*7)
	err = tx.QueryRow(`
		INSERT INTO public.training_programs
			(client_id, name, goal, total_weeks, days_per_week, start_date, end_date, status, ai_generated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		clientID, planName, string(program.Goal), weeks, days, startDate, endDate, "active", true,
	).Scan(&programID)

	if err != nil {
		log.Printf("Ошибка сохранения программы: %v", err)
		// Продолжаем без программы
	} else {
		// Save workouts and exercises
		for _, week := range program.Weeks {
			for _, day := range week.Days {
				var workoutID int
				workoutName := day.Name
				if workoutName == "" {
					workoutName = fmt.Sprintf("Неделя %d, День %d", week.WeekNum, day.DayNum)
				}

				err = tx.QueryRow(`
					INSERT INTO public.program_workouts
						(program_id, week_num, day_num, order_in_week, name, status)
					VALUES ($1, $2, $3, $4, $5, $6)
					RETURNING id`,
					programID, week.WeekNum, day.DayNum, day.DayNum, workoutName, "pending",
				).Scan(&workoutID)

				if err != nil {
					log.Printf("Ошибка сохранения тренировки: %v", err)
					continue
				}

				// Save exercises
				for _, ex := range day.Exercises {
					_, err = tx.Exec(`
						INSERT INTO public.workout_exercises
							(workout_id, order_num, exercise_name, sets, reps, weight, weight_percent, rest_seconds, rpe, notes)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
						workoutID, ex.OrderNum, ex.ExerciseName, ex.Sets, ex.Reps,
						sql.NullFloat64{Float64: ex.Weight, Valid: ex.Weight > 0},
						sql.NullFloat64{Float64: ex.WeightPercent, Valid: ex.WeightPercent > 0},
						ex.RestSeconds, ex.RPE, ex.Notes,
					)
					if err != nil {
						log.Printf("Ошибка сохранения упражнения: %v", err)
					}
				}
			}
		}
	}

	// Generate and save progression for exercises with 1PM
	exercises, client1PMs := b.getClientExercisesAnd1PMs(clientID)
	if len(exercises) > 0 && len(client1PMs) > 0 {
		config := training.DefaultProgressionConfig()
		config.TotalWeeks = weeks

		progression := training.GenerateProgression(exercises, client1PMs, plan.Mesocycles, config)

		for _, p := range progression {
			_, err = tx.Exec(`
				INSERT INTO public.plan_progression
					(training_plan_id, exercise_id, week_number, sets, reps, weight_kg, intensity_percent, is_deload)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (training_plan_id, exercise_id, week_number) DO UPDATE
				SET sets = $4, reps = $5, weight_kg = $6, intensity_percent = $7, is_deload = $8`,
				planID, p.ExerciseID, p.WeekNumber, p.Sets, p.Reps, p.WeightKg, p.IntensityPercent, p.IsDeload,
			)
			if err != nil {
				log.Printf("Ошибка сохранения прогрессии: %v", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Ошибка коммита: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения плана")
		b.api.Send(msg)
		return
	}

	// Count generated workouts
	totalWorkouts := 0
	totalExercises := 0
	for _, week := range program.Weeks {
		totalWorkouts += len(week.Days)
		for _, day := range week.Days {
			totalExercises += len(day.Exercises)
		}
	}

	// Show success message
	responseText := fmt.Sprintf("✅ План создан!\n\n"+
		"📋 %s\n"+
		"📅 %d недель, %d тренировок/неделю\n\n"+
		"Периодизация:\n", planName, weeks, days)

	for _, meso := range plan.Mesocycles {
		responseText += fmt.Sprintf("• Нед. %d-%d: %s (%s)\n",
			meso.WeekStart, meso.WeekEnd, meso.Name, meso.Phase.NameRu())
	}

	responseText += fmt.Sprintf("\n🏋️ Сгенерировано:\n"+
		"• %d тренировок\n"+
		"• %d упражнений всего\n", totalWorkouts, totalExercises)

	if len(client1PMs) > 0 {
		responseText += fmt.Sprintf("\n📈 Прогрессия весов рассчитана для %d упражнений с 1ПМ\n", len(client1PMs))
	} else {
		responseText += "\n💡 Добавьте 1ПМ для расчёта конкретных весов\n"
	}

	msg := tgbotapi.NewMessage(chatID, responseText)
	b.api.Send(msg)

	b.clearPlanState(chatID)
	b.handlePlansMenu(message)
}

// getClientExercisesAnd1PMs returns exercises and their 1PM values for a client
func (b *Bot) getClientExercisesAnd1PMs(clientID int) ([]models.Exercise, map[int]float64) {
	rows, err := b.db.Query(`
		SELECT DISTINCT e.id, e.name, e.muscle_group, e.movement_type, e.equipment,
			(SELECT one_pm_kg FROM public.exercise_1pm
				WHERE client_id = $1 AND exercise_id = e.id
				ORDER BY test_date DESC LIMIT 1) as current_1pm
		FROM public.exercises e
		INNER JOIN public.exercise_1pm pm ON pm.exercise_id = e.id
		WHERE pm.client_id = $1
		ORDER BY e.muscle_group, e.name`, clientID)
	if err != nil {
		log.Printf("Ошибка получения упражнений: %v", err)
		return nil, nil
	}
	defer rows.Close()

	var exercises []models.Exercise
	client1PMs := make(map[int]float64)

	for rows.Next() {
		var ex models.Exercise
		var current1PM sql.NullFloat64
		if err := rows.Scan(&ex.ID, &ex.Name, &ex.MuscleGroup, &ex.MovementType, &ex.Equipment, &current1PM); err != nil {
			continue
		}
		exercises = append(exercises, ex)
		if current1PM.Valid {
			client1PMs[ex.ID] = current1PM.Float64
		}
	}

	return exercises, client1PMs
}

// showPlansList shows list of existing plans
func (b *Bot) showPlansList(chatID int64) {
	rows, err := b.db.Query(`
		SELECT tp.id, tp.name, c.name || ' ' || c.surname as client_name,
			tp.status, tp.total_weeks, tp.start_date
		FROM public.training_plans tp
		JOIN public.clients c ON tp.client_id = c.id
		WHERE tp.status != 'archived'
		ORDER BY tp.created_at DESC
		LIMIT 20`)
	if err != nil {
		log.Printf("Ошибка получения планов: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки планов")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var text strings.Builder
	text.WriteString("📋 Тренировочные планы:\n\n")

	count := 0
	for rows.Next() {
		var id, weeks int
		var name, clientName, status string
		var startDate time.Time
		if err := rows.Scan(&id, &name, &clientName, &status, &weeks, &startDate); err != nil {
			continue
		}

		statusEmoji := "📋"
		switch status {
		case "active":
			statusEmoji = "🟢"
		case "completed":
			statusEmoji = "✅"
		case "draft":
			statusEmoji = "📝"
		}

		text.WriteString(fmt.Sprintf("%s #%d %s\n   👤 %s | %d нед. | с %s\n\n",
			statusEmoji, id, name, clientName, weeks, startDate.Format("02.01.2006")))
		count++
	}

	if count == 0 {
		text.WriteString("Нет планов. Создайте первый план!")
	}

	text.WriteString("\nИспользуйте кнопки меню для действий.")
	msg := tgbotapi.NewMessage(chatID, text.String())
	b.api.Send(msg)
}

// clearPlanState clears all temporary plan data
func (b *Bot) clearPlanState(chatID int64) {
	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	planStore.Lock()
	delete(planStore.clientID, chatID)
	delete(planStore.goal, chatID)
	delete(planStore.weeks, chatID)
	delete(planStore.daysPerWeek, chatID)
	delete(planStore.planName, chatID)
	planStore.Unlock()
}

// parsePlanClientID extracts client ID from button text
func parsePlanClientID(text string) int {
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		return 0
	}
	id, _ := strconv.Atoi(text[start+1 : end])
	return id
}

// handlePlanState routes plan states
func (b *Bot) handlePlanState(msg *tgbotapi.Message, state string) {
	switch state {
	case statePlanMenu:
		b.handlePlanMenuChoice(msg)
	case statePlanSelectClient:
		b.handlePlanClientSelect(msg)
	case "plan_no_1pm_confirm":
		b.handlePlanNo1PMConfirm(msg)
	case statePlanSelectGoal:
		b.handlePlanGoalSelect(msg)
	case statePlanSelectDuration:
		b.handlePlanDurationSelect(msg)
	case statePlanSelectDays:
		b.handlePlanDaysSelect(msg)
	case statePlanConfirm:
		b.handlePlanConfirm(msg)
	case statePlanExportSelect:
		b.handlePlanExportSelect(msg)
	}
}

package repository

import (
	"database/sql"
	"time"

	"workbot/internal/models"
)

// PlanRepository работает с тренировочными планами
type PlanRepository struct {
	db *sql.DB
}

// NewPlanRepository создаёт репозиторий планов
func NewPlanRepository(db *sql.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

// Create создаёт новый план
func (r *PlanRepository) Create(plan *models.TrainingPlan) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.training_plans
		(client_id, name, start_date, end_date, status, goal, days_per_week, total_weeks, ai_generated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		plan.ClientID, plan.Name, plan.StartDate, plan.EndDate,
		plan.Status, plan.Goal, plan.DaysPerWeek, plan.TotalWeeks, plan.AIGenerated,
	).Scan(&id)
	return id, err
}

// GetByID возвращает план по ID
func (r *PlanRepository) GetByID(id int) (*models.TrainingPlan, error) {
	plan := &models.TrainingPlan{}
	var endDate sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, client_id, name, start_date, end_date, status,
		       COALESCE(goal, ''), days_per_week, total_weeks,
		       COALESCE(ai_generated, false), created_at, updated_at
		FROM public.training_plans
		WHERE id = $1`, id).Scan(
		&plan.ID, &plan.ClientID, &plan.Name, &plan.StartDate, &endDate,
		&plan.Status, &plan.Goal, &plan.DaysPerWeek, &plan.TotalWeeks,
		&plan.AIGenerated, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if endDate.Valid {
		plan.EndDate = &endDate.Time
	}
	return plan, nil
}

// GetByClientID возвращает планы клиента
func (r *PlanRepository) GetByClientID(clientID int) ([]models.TrainingPlan, error) {
	rows, err := r.db.Query(`
		SELECT id, client_id, name, start_date, end_date, status,
		       COALESCE(goal, ''), days_per_week, total_weeks,
		       COALESCE(ai_generated, false), created_at, updated_at
		FROM public.training_plans
		WHERE client_id = $1
		ORDER BY created_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []models.TrainingPlan
	for rows.Next() {
		var plan models.TrainingPlan
		var endDate sql.NullTime
		if err := rows.Scan(
			&plan.ID, &plan.ClientID, &plan.Name, &plan.StartDate, &endDate,
			&plan.Status, &plan.Goal, &plan.DaysPerWeek, &plan.TotalWeeks,
			&plan.AIGenerated, &plan.CreatedAt, &plan.UpdatedAt,
		); err != nil {
			continue
		}
		if endDate.Valid {
			plan.EndDate = &endDate.Time
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// GetActivePlan возвращает активный план клиента
func (r *PlanRepository) GetActivePlan(clientID int) (*models.TrainingPlan, error) {
	plan := &models.TrainingPlan{}
	var endDate sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, client_id, name, start_date, end_date, status,
		       COALESCE(goal, ''), days_per_week, total_weeks,
		       COALESCE(ai_generated, false), created_at, updated_at
		FROM public.training_plans
		WHERE client_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1`, clientID).Scan(
		&plan.ID, &plan.ClientID, &plan.Name, &plan.StartDate, &endDate,
		&plan.Status, &plan.Goal, &plan.DaysPerWeek, &plan.TotalWeeks,
		&plan.AIGenerated, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if endDate.Valid {
		plan.EndDate = &endDate.Time
	}
	return plan, nil
}

// UpdateStatus обновляет статус плана
func (r *PlanRepository) UpdateStatus(planID int, status models.PlanStatus) error {
	_, err := r.db.Exec(
		"UPDATE public.training_plans SET status = $1, updated_at = NOW() WHERE id = $2",
		status, planID,
	)
	return err
}

// SaveMesocycle сохраняет мезоцикл
func (r *PlanRepository) SaveMesocycle(planID int, meso *models.Mesocycle) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.mesocycles
		(training_plan_id, name, phase, week_start, week_end, intensity_percent, volume_percent, rpe_target, order_num)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		planID, meso.Name, meso.Phase, meso.WeekStart, meso.WeekEnd,
		meso.IntensityPercent, meso.VolumePercent, meso.RPETarget, meso.OrderNum,
	).Scan(&id)
	return id, err
}

// GetMesocycles возвращает мезоциклы плана
func (r *PlanRepository) GetMesocycles(planID int) ([]models.Mesocycle, error) {
	rows, err := r.db.Query(`
		SELECT id, training_plan_id, COALESCE(name, ''), phase,
		       week_start, week_end, intensity_percent, volume_percent,
		       COALESCE(rpe_target, 7), order_num, created_at
		FROM public.mesocycles
		WHERE training_plan_id = $1
		ORDER BY order_num`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mesocycles []models.Mesocycle
	for rows.Next() {
		var m models.Mesocycle
		if err := rows.Scan(
			&m.ID, &m.TrainingPlanID, &m.Name, &m.Phase,
			&m.WeekStart, &m.WeekEnd, &m.IntensityPercent, &m.VolumePercent,
			&m.RPETarget, &m.OrderNum, &m.CreatedAt,
		); err != nil {
			continue
		}
		mesocycles = append(mesocycles, m)
	}
	return mesocycles, nil
}

// SaveMicrocycle сохраняет микроцикл
func (r *PlanRepository) SaveMicrocycle(mesoID int, micro *models.Microcycle) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.microcycles
		(mesocycle_id, week_number, name, is_deload, volume_modifier, intensity_modifier)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		mesoID, micro.WeekNumber, micro.Name, micro.IsDeload,
		micro.VolumeModifier, micro.IntensityModifier,
	).Scan(&id)
	return id, err
}

// SaveProgression сохраняет строку прогрессии
func (r *PlanRepository) SaveProgression(planID, exerciseID, weekNumber, sets, reps int, weightKg, intensityPercent float64, isDeload bool) error {
	_, err := r.db.Exec(`
		INSERT INTO public.plan_progression
		(training_plan_id, exercise_id, week_number, sets, reps, weight_kg, intensity_percent, is_deload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (training_plan_id, exercise_id, week_number)
		DO UPDATE SET sets = $4, reps = $5, weight_kg = $6, intensity_percent = $7, is_deload = $8`,
		planID, exerciseID, weekNumber, sets, reps, weightKg, intensityPercent, isDeload,
	)
	return err
}

// GetProgression возвращает прогрессию плана
func (r *PlanRepository) GetProgression(planID int) ([]models.Progression, error) {
	rows, err := r.db.Query(`
		SELECT pp.id, pp.training_plan_id, pp.exercise_id, e.name,
		       pp.week_number, pp.sets, pp.reps, COALESCE(pp.weight_kg, 0),
		       COALESCE(pp.intensity_percent, 0), COALESCE(pp.is_deload, false)
		FROM public.plan_progression pp
		JOIN public.exercises e ON pp.exercise_id = e.id
		WHERE pp.training_plan_id = $1
		ORDER BY pp.exercise_id, pp.week_number`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progressions []models.Progression
	for rows.Next() {
		var p models.Progression
		if err := rows.Scan(
			&p.ID, &p.TrainingPlanID, &p.ExerciseID, &p.ExerciseName,
			&p.WeekNumber, &p.Sets, &p.Reps, &p.WeightKg,
			&p.IntensityPercent, &p.IsDeload,
		); err != nil {
			continue
		}
		progressions = append(progressions, p)
	}
	return progressions, nil
}

// SaveTrainingLog сохраняет лог тренировки
func (r *PlanRepository) SaveTrainingLog(log *models.TrainingLog) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.training_logs
		(client_id, training_plan_id, exercise_id, training_date, sets_completed, reps_completed, weight_kg, rpe_actual, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		log.ClientID, log.TrainingPlanID, log.ExerciseID, log.TrainingDate,
		log.SetsCompleted, log.RepsCompleted, log.WeightKg, log.RPEActual, log.Status,
	).Scan(&id)
	return id, err
}

// GetTrainingLogs возвращает логи тренировок
func (r *PlanRepository) GetTrainingLogs(clientID int, from, to time.Time) ([]models.TrainingLog, error) {
	rows, err := r.db.Query(`
		SELECT tl.id, tl.client_id, COALESCE(tl.training_plan_id, 0), tl.exercise_id, e.name,
		       COALESCE(e.muscle_group, ''), tl.training_date, tl.sets_completed, tl.reps_completed,
		       COALESCE(tl.weight_kg, 0), tl.tonnage_kg, COALESCE(tl.rpe_actual, 0), tl.status, tl.created_at
		FROM public.training_logs tl
		JOIN public.exercises e ON e.id = tl.exercise_id
		WHERE tl.client_id = $1 AND tl.training_date BETWEEN $2 AND $3
		ORDER BY tl.training_date DESC, tl.id DESC`, clientID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.TrainingLog
	for rows.Next() {
		var l models.TrainingLog
		var planID sql.NullInt64
		if err := rows.Scan(
			&l.ID, &l.ClientID, &planID, &l.ExerciseID, &l.ExerciseName,
			&l.MuscleGroup, &l.TrainingDate, &l.SetsCompleted, &l.RepsCompleted, &l.WeightKg,
			&l.TonnageKg, &l.RPEActual, &l.Status, &l.CreatedAt,
		); err != nil {
			continue
		}
		if planID.Valid {
			pid := int(planID.Int64)
			l.TrainingPlanID = &pid
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// GetAllPlans возвращает все планы с базовой информацией
func (r *PlanRepository) GetAllPlans(limit int) ([]models.TrainingPlanListItem, error) {
	rows, err := r.db.Query(`
		SELECT tp.id, tp.name, c.name, tp.status, COALESCE(tp.goal, ''),
		       tp.total_weeks, tp.start_date
		FROM public.training_plans tp
		JOIN public.clients c ON c.id = tp.client_id
		ORDER BY tp.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []models.TrainingPlanListItem
	for rows.Next() {
		var p models.TrainingPlanListItem
		if err := rows.Scan(&p.ID, &p.Name, &p.ClientName, &p.Status, &p.Goal,
			&p.TotalWeeks, &p.StartDate); err != nil {
			continue
		}
		// Вычисляем текущую неделю
		daysSinceStart := int(time.Since(p.StartDate).Hours() / 24)
		p.CurrentWeek = (daysSinceStart / 7) + 1
		if p.CurrentWeek < 1 {
			p.CurrentWeek = 1
		}
		if p.CurrentWeek > p.TotalWeeks {
			p.CurrentWeek = p.TotalWeeks
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

// GetPlanWithDetails возвращает план со всеми связанными данными
func (r *PlanRepository) GetPlanWithDetails(planID int) (*models.TrainingPlan, error) {
	plan, err := r.GetByID(planID)
	if err != nil {
		return nil, err
	}

	// Загружаем мезоциклы
	mesocycles, err := r.GetMesocycles(planID)
	if err != nil {
		return nil, err
	}

	// Для каждого мезоцикла загружаем микроциклы
	for i := range mesocycles {
		microcycles, err := r.GetMicrocycles(mesocycles[i].ID)
		if err != nil {
			continue
		}

		// Для каждого микроцикла загружаем упражнения
		for j := range microcycles {
			exercises, err := r.GetPlanExercises(microcycles[j].ID)
			if err != nil {
				continue
			}
			microcycles[j].Exercises = exercises
		}
		mesocycles[i].Microcycles = microcycles
	}
	plan.Mesocycles = mesocycles

	// Загружаем прогрессию
	progression, err := r.GetProgression(planID)
	if err == nil {
		plan.Progression = progression
	}

	return plan, nil
}

// GetMicrocycles возвращает микроциклы мезоцикла
func (r *PlanRepository) GetMicrocycles(mesoID int) ([]models.Microcycle, error) {
	rows, err := r.db.Query(`
		SELECT id, mesocycle_id, week_number, COALESCE(name, ''),
		       is_deload, volume_modifier, intensity_modifier,
		       COALESCE(notes, ''), created_at
		FROM public.microcycles
		WHERE mesocycle_id = $1
		ORDER BY week_number`, mesoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var microcycles []models.Microcycle
	for rows.Next() {
		var m models.Microcycle
		if err := rows.Scan(&m.ID, &m.MesocycleID, &m.WeekNumber, &m.Name,
			&m.IsDeload, &m.VolumeModifier, &m.IntensityModifier,
			&m.Notes, &m.CreatedAt); err != nil {
			continue
		}
		microcycles = append(microcycles, m)
	}
	return microcycles, rows.Err()
}

// GetPlanExercises возвращает упражнения микроцикла
func (r *PlanRepository) GetPlanExercises(microcycleID int) ([]models.PlanExercise, error) {
	rows, err := r.db.Query(`
		SELECT pe.id, pe.microcycle_id, pe.exercise_id, e.name, COALESCE(e.muscle_group, ''),
		       pe.day_of_week, pe.order_num, pe.sets, pe.reps_min, pe.reps_max,
		       COALESCE(pe.intensity_percent, 0), COALESCE(pe.rpe_target, 0),
		       COALESCE(pe.rest_seconds, 90), COALESCE(pe.tempo, ''), COALESCE(pe.notes, '')
		FROM public.plan_exercises pe
		JOIN public.exercises e ON e.id = pe.exercise_id
		WHERE pe.microcycle_id = $1
		ORDER BY pe.day_of_week, pe.order_num`, microcycleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []models.PlanExercise
	for rows.Next() {
		var ex models.PlanExercise
		if err := rows.Scan(&ex.ID, &ex.MicrocycleID, &ex.ExerciseID, &ex.ExerciseName,
			&ex.MuscleGroup, &ex.DayOfWeek, &ex.OrderNum, &ex.Sets, &ex.RepsMin, &ex.RepsMax,
			&ex.IntensityPercent, &ex.RPETarget, &ex.RestSeconds, &ex.Tempo, &ex.Notes); err != nil {
			continue
		}
		exercises = append(exercises, ex)
	}
	return exercises, rows.Err()
}

// SavePlanExercise сохраняет упражнение в микроцикле
func (r *PlanRepository) SavePlanExercise(ex *models.PlanExercise) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.plan_exercises
		(microcycle_id, exercise_id, day_of_week, order_num, sets, reps_min, reps_max,
		 intensity_percent, rpe_target, rest_seconds, tempo, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		ex.MicrocycleID, ex.ExerciseID, ex.DayOfWeek, ex.OrderNum,
		ex.Sets, ex.RepsMin, ex.RepsMax, ex.IntensityPercent, ex.RPETarget,
		ex.RestSeconds, ex.Tempo, ex.Notes,
	).Scan(&id)
	return id, err
}

// DeletePlan удаляет план и все связанные данные (каскадно через FK)
func (r *PlanRepository) DeletePlan(planID int) error {
	_, err := r.db.Exec("DELETE FROM public.training_plans WHERE id = $1", planID)
	return err
}

// Update обновляет план
func (r *PlanRepository) Update(plan *models.TrainingPlan) error {
	_, err := r.db.Exec(`
		UPDATE public.training_plans
		SET name = $1, description = $2, start_date = $3, end_date = $4,
		    status = $5, goal = $6, days_per_week = $7, total_weeks = $8,
		    ai_prompt = $9, updated_at = NOW()
		WHERE id = $10`,
		plan.Name, plan.Description, plan.StartDate, plan.EndDate,
		plan.Status, plan.Goal, plan.DaysPerWeek, plan.TotalWeeks,
		plan.AIPrompt, plan.ID,
	)
	return err
}

// GetProgressionTable возвращает полную таблицу прогрессии для экспорта
func (r *PlanRepository) GetProgressionTable(planID int) (*models.ProgressionTable, error) {
	plan, err := r.GetByID(planID)
	if err != nil {
		return nil, err
	}

	// Получаем имя клиента
	var clientName string
	_ = r.db.QueryRow("SELECT name FROM public.clients WHERE id = $1", plan.ClientID).Scan(&clientName)

	// Получаем прогрессию
	progressions, err := r.GetProgression(planID)
	if err != nil {
		return nil, err
	}

	// Группируем по неделям и упражнениям
	table := &models.ProgressionTable{
		PlanName:      plan.Name,
		ClientName:    clientName,
		TotalWeeks:    plan.TotalWeeks,
		ExerciseNames: make([]string, 0),
		Weeks:         make([]models.WeekProgression, 0),
		ByExercise:    make(map[string][]models.Progression),
	}

	exerciseSet := make(map[string]bool)
	weekMap := make(map[int]*models.WeekProgression)

	for _, p := range progressions {
		// Добавляем упражнение в список
		if !exerciseSet[p.ExerciseName] {
			exerciseSet[p.ExerciseName] = true
			table.ExerciseNames = append(table.ExerciseNames, p.ExerciseName)
		}

		// Добавляем в карту по упражнениям
		table.ByExercise[p.ExerciseName] = append(table.ByExercise[p.ExerciseName], p)

		// Группируем по неделям
		if _, ok := weekMap[p.WeekNumber]; !ok {
			weekMap[p.WeekNumber] = &models.WeekProgression{
				WeekNumber: p.WeekNumber,
				IsDeload:   p.IsDeload,
				Exercises:  make([]models.Progression, 0),
			}
		}
		weekMap[p.WeekNumber].Exercises = append(weekMap[p.WeekNumber].Exercises, p)
	}

	// Собираем недели по порядку
	for w := 1; w <= plan.TotalWeeks; w++ {
		if week, ok := weekMap[w]; ok {
			table.Weeks = append(table.Weeks, *week)
		}
	}

	return table, nil
}

// GetWeeklyVolume возвращает объём тренировок по неделям
func (r *PlanRepository) GetWeeklyVolume(clientID int, planID *int) ([]models.VolumeAnalytics, error) {
	var rows *sql.Rows
	var err error

	if planID != nil {
		rows, err = r.db.Query(`
			SELECT id, client_id, training_plan_id, week_number, muscle_group,
			       total_sets, total_reps, total_tonnage, COALESCE(avg_intensity, 0), computed_at
			FROM public.volume_analytics
			WHERE client_id = $1 AND training_plan_id = $2
			ORDER BY week_number, muscle_group`, clientID, *planID)
	} else {
		rows, err = r.db.Query(`
			SELECT id, client_id, training_plan_id, week_number, muscle_group,
			       total_sets, total_reps, total_tonnage, COALESCE(avg_intensity, 0), computed_at
			FROM public.volume_analytics
			WHERE client_id = $1
			ORDER BY week_number, muscle_group`, clientID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analytics []models.VolumeAnalytics
	for rows.Next() {
		var a models.VolumeAnalytics
		var planIDNull sql.NullInt64
		if err := rows.Scan(&a.ID, &a.ClientID, &planIDNull, &a.WeekNumber, &a.MuscleGroup,
			&a.TotalSets, &a.TotalReps, &a.TotalTonnage, &a.AvgIntensity, &a.ComputedAt); err != nil {
			continue
		}
		if planIDNull.Valid {
			pid := int(planIDNull.Int64)
			a.TrainingPlanID = &pid
		}
		analytics = append(analytics, a)
	}
	return analytics, rows.Err()
}

// ComputeWeeklyVolume вычисляет и сохраняет объём за неделю
func (r *PlanRepository) ComputeWeeklyVolume(clientID int, planID *int, weekNumber int) error {
	_, err := r.db.Exec(`
		INSERT INTO public.volume_analytics
		(client_id, training_plan_id, week_number, muscle_group, total_sets, total_reps, total_tonnage, avg_intensity)
		SELECT
			tl.client_id,
			tl.training_plan_id,
			$3 as week_number,
			COALESCE(e.muscle_group, 'другое'),
			SUM(tl.sets_completed),
			SUM(tl.reps_completed),
			SUM(tl.tonnage_kg),
			AVG(CASE WHEN tl.weight_kg > 0 THEN tl.weight_kg END)
		FROM public.training_logs tl
		JOIN public.exercises e ON e.id = tl.exercise_id
		WHERE tl.client_id = $1
		  AND ($2::int IS NULL OR tl.training_plan_id = $2)
		  AND tl.week_number = $3
		GROUP BY tl.client_id, tl.training_plan_id, e.muscle_group
		ON CONFLICT (client_id, training_plan_id, week_number, muscle_group)
		DO UPDATE SET
			total_sets = EXCLUDED.total_sets,
			total_reps = EXCLUDED.total_reps,
			total_tonnage = EXCLUDED.total_tonnage,
			avg_intensity = EXCLUDED.avg_intensity,
			computed_at = NOW()`,
		clientID, planID, weekNumber)
	return err
}

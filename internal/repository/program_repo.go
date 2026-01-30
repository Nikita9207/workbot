package repository

import (
	"database/sql"
	"time"

	"workbot/internal/models"
)

// ProgramRepository репозиторий для работы с программами тренировок
type ProgramRepository struct {
	db *sql.DB
}

// NewProgramRepository создаёт новый репозиторий программ
func NewProgramRepository(db *sql.DB) *ProgramRepository {
	return &ProgramRepository{db: db}
}

// GetActiveProgram возвращает активную программу клиента
func (r *ProgramRepository) GetActiveProgram(clientID int) (*models.Program, error) {
	query := `
		SELECT id, client_id, name, COALESCE(goal, ''), total_weeks, days_per_week,
		       start_date, end_date, status, COALESCE(current_week, 1), created_at
		FROM public.training_programs
		WHERE client_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1`

	var program models.Program
	var endDate sql.NullTime
	err := r.db.QueryRow(query, clientID).Scan(
		&program.ID, &program.ClientID, &program.Name, &program.Goal,
		&program.TotalWeeks, &program.DaysPerWeek, &program.StartDate,
		&endDate, &program.Status, &program.CurrentWeek, &program.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if endDate.Valid {
		program.EndDate = &endDate.Time
	}

	// Загружаем тренировки
	workouts, err := r.GetWorkoutsByProgram(program.ID)
	if err != nil {
		return nil, err
	}
	program.Workouts = workouts

	return &program, nil
}

// GetProgramByID возвращает программу по ID
func (r *ProgramRepository) GetProgramByID(programID int) (*models.Program, error) {
	query := `
		SELECT id, client_id, name, COALESCE(goal, ''), total_weeks, days_per_week,
		       start_date, end_date, status, COALESCE(current_week, 1), created_at
		FROM public.training_programs
		WHERE id = $1`

	var program models.Program
	var endDate sql.NullTime
	err := r.db.QueryRow(query, programID).Scan(
		&program.ID, &program.ClientID, &program.Name, &program.Goal,
		&program.TotalWeeks, &program.DaysPerWeek, &program.StartDate,
		&endDate, &program.Status, &program.CurrentWeek, &program.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if endDate.Valid {
		program.EndDate = &endDate.Time
	}

	// Загружаем тренировки
	workouts, err := r.GetWorkoutsByProgram(program.ID)
	if err != nil {
		return nil, err
	}
	program.Workouts = workouts

	return &program, nil
}

// GetWorkoutsByProgram возвращает все тренировки программы
func (r *ProgramRepository) GetWorkoutsByProgram(programID int) ([]models.Workout, error) {
	query := `
		SELECT id, program_id, week_num, day_num, order_in_week, name,
		       planned_date, status, COALESCE(notes, ''), COALESCE(feedback, ''),
		       completed_at, sent_at
		FROM public.program_workouts
		WHERE program_id = $1
		ORDER BY week_num, order_in_week`

	rows, err := r.db.Query(query, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workouts []models.Workout
	for rows.Next() {
		var w models.Workout
		var plannedDate, completedAt, sentAt sql.NullTime
		err := rows.Scan(
			&w.ID, &w.ProgramID, &w.WeekNum, &w.DayNum, &w.OrderInWeek, &w.Name,
			&plannedDate, &w.Status, &w.Notes, &w.Feedback,
			&completedAt, &sentAt,
		)
		if err != nil {
			return nil, err
		}

		if plannedDate.Valid {
			w.Date = &plannedDate.Time
		}
		if completedAt.Valid {
			w.CompletedAt = &completedAt.Time
		}
		if sentAt.Valid {
			w.SentAt = &sentAt.Time
		}

		// Загружаем упражнения
		exercises, err := r.GetExercisesByWorkout(w.ID)
		if err != nil {
			return nil, err
		}
		w.Exercises = exercises

		workouts = append(workouts, w)
	}

	return workouts, nil
}

// GetWorkoutByID возвращает тренировку по ID
func (r *ProgramRepository) GetWorkoutByID(workoutID int) (*models.Workout, error) {
	query := `
		SELECT id, program_id, week_num, day_num, order_in_week, name,
		       planned_date, status, COALESCE(notes, ''), COALESCE(feedback, ''),
		       completed_at, sent_at
		FROM public.program_workouts
		WHERE id = $1`

	var w models.Workout
	var plannedDate, completedAt, sentAt sql.NullTime
	err := r.db.QueryRow(query, workoutID).Scan(
		&w.ID, &w.ProgramID, &w.WeekNum, &w.DayNum, &w.OrderInWeek, &w.Name,
		&plannedDate, &w.Status, &w.Notes, &w.Feedback,
		&completedAt, &sentAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if plannedDate.Valid {
		w.Date = &plannedDate.Time
	}
	if completedAt.Valid {
		w.CompletedAt = &completedAt.Time
	}
	if sentAt.Valid {
		w.SentAt = &sentAt.Time
	}

	// Загружаем упражнения
	exercises, err := r.GetExercisesByWorkout(w.ID)
	if err != nil {
		return nil, err
	}
	w.Exercises = exercises

	return &w, nil
}

// GetNextPendingWorkout возвращает следующую невыполненную тренировку клиента
func (r *ProgramRepository) GetNextPendingWorkout(clientID int) (*models.Workout, error) {
	query := `
		SELECT pw.id, pw.program_id, pw.week_num, pw.day_num, pw.order_in_week, pw.name,
		       pw.planned_date, pw.status, COALESCE(pw.notes, ''), COALESCE(pw.feedback, ''),
		       pw.completed_at, pw.sent_at
		FROM public.program_workouts pw
		JOIN public.training_programs tp ON pw.program_id = tp.id
		WHERE tp.client_id = $1 AND tp.status = 'active' AND pw.status IN ('pending', 'sent')
		ORDER BY pw.week_num, pw.order_in_week
		LIMIT 1`

	var w models.Workout
	var plannedDate, completedAt, sentAt sql.NullTime
	err := r.db.QueryRow(query, clientID).Scan(
		&w.ID, &w.ProgramID, &w.WeekNum, &w.DayNum, &w.OrderInWeek, &w.Name,
		&plannedDate, &w.Status, &w.Notes, &w.Feedback,
		&completedAt, &sentAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if plannedDate.Valid {
		w.Date = &plannedDate.Time
	}
	if completedAt.Valid {
		w.CompletedAt = &completedAt.Time
	}
	if sentAt.Valid {
		w.SentAt = &sentAt.Time
	}

	// Загружаем упражнения
	exercises, err := r.GetExercisesByWorkout(w.ID)
	if err != nil {
		return nil, err
	}
	w.Exercises = exercises

	return &w, nil
}

// GetExercisesByWorkout возвращает упражнения тренировки
func (r *ProgramRepository) GetExercisesByWorkout(workoutID int) ([]models.WorkoutExercise, error) {
	query := `
		SELECT id, workout_id, order_num, exercise_name, sets, COALESCE(reps, ''),
		       COALESCE(weight, 0), COALESCE(weight_percent, 0), COALESCE(rest_seconds, 0),
		       COALESCE(tempo, ''), COALESCE(rpe, 0), COALESCE(notes, ''),
		       COALESCE(actual_sets, 0), COALESCE(actual_reps, 0), COALESCE(actual_weight, 0),
		       COALESCE(actual_rpe, 0), COALESCE(completed, false)
		FROM public.workout_exercises
		WHERE workout_id = $1
		ORDER BY order_num`

	rows, err := r.db.Query(query, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []models.WorkoutExercise
	for rows.Next() {
		var e models.WorkoutExercise
		err := rows.Scan(
			&e.ID, &e.WorkoutID, &e.OrderNum, &e.ExerciseName, &e.Sets, &e.Reps,
			&e.Weight, &e.WeightPercent, &e.RestSeconds, &e.Tempo, &e.RPE, &e.Notes,
			&e.ActualSets, &e.ActualReps, &e.ActualWeight, &e.ActualRPE, &e.Completed,
		)
		if err != nil {
			return nil, err
		}
		exercises = append(exercises, e)
	}

	return exercises, nil
}

// GetExerciseByID возвращает упражнение по ID
func (r *ProgramRepository) GetExerciseByID(exerciseID int) (*models.WorkoutExercise, error) {
	query := `
		SELECT id, workout_id, order_num, exercise_name, sets, COALESCE(reps, ''),
		       COALESCE(weight, 0), COALESCE(weight_percent, 0), COALESCE(rest_seconds, 0),
		       COALESCE(tempo, ''), COALESCE(rpe, 0), COALESCE(notes, ''),
		       COALESCE(actual_sets, 0), COALESCE(actual_reps, 0), COALESCE(actual_weight, 0),
		       COALESCE(actual_rpe, 0), COALESCE(completed, false)
		FROM public.workout_exercises
		WHERE id = $1`

	var e models.WorkoutExercise
	err := r.db.QueryRow(query, exerciseID).Scan(
		&e.ID, &e.WorkoutID, &e.OrderNum, &e.ExerciseName, &e.Sets, &e.Reps,
		&e.Weight, &e.WeightPercent, &e.RestSeconds, &e.Tempo, &e.RPE, &e.Notes,
		&e.ActualSets, &e.ActualReps, &e.ActualWeight, &e.ActualRPE, &e.Completed,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// MarkWorkoutSent отмечает тренировку как отправленную
func (r *ProgramRepository) MarkWorkoutSent(workoutID int) error {
	query := `
		UPDATE public.program_workouts
		SET status = 'sent', sent_at = $1
		WHERE id = $2`
	_, err := r.db.Exec(query, time.Now(), workoutID)
	return err
}

// MarkWorkoutCompleted отмечает тренировку как выполненную
func (r *ProgramRepository) MarkWorkoutCompleted(workoutID int, feedback string) error {
	query := `
		UPDATE public.program_workouts
		SET status = 'completed', completed_at = $1, feedback = $2
		WHERE id = $3`
	_, err := r.db.Exec(query, time.Now(), feedback, workoutID)
	return err
}

// MarkWorkoutSkipped отмечает тренировку как пропущенную
func (r *ProgramRepository) MarkWorkoutSkipped(workoutID int) error {
	query := `
		UPDATE public.program_workouts
		SET status = 'skipped'
		WHERE id = $1`
	_, err := r.db.Exec(query, workoutID)
	return err
}

// UpdateExerciseResult обновляет фактические результаты упражнения
func (r *ProgramRepository) UpdateExerciseResult(exerciseID int, actualSets, actualReps int, actualWeight, actualRPE float64) error {
	query := `
		UPDATE public.workout_exercises
		SET actual_sets = $1, actual_reps = $2, actual_weight = $3, actual_rpe = $4, completed = true
		WHERE id = $5`
	_, err := r.db.Exec(query, actualSets, actualReps, actualWeight, actualRPE, exerciseID)
	return err
}

// MarkExerciseCompleted отмечает упражнение как выполненное с плановыми значениями
func (r *ProgramRepository) MarkExerciseCompleted(exerciseID int) error {
	query := `
		UPDATE public.workout_exercises
		SET actual_sets = sets, actual_reps = COALESCE(
			NULLIF(REGEXP_REPLACE(reps, '-.*', ''), ''),
			reps
		)::int, actual_weight = weight, completed = true
		WHERE id = $1`
	_, err := r.db.Exec(query, exerciseID)
	return err
}

// MarkExerciseSkipped отмечает упражнение как пропущенное
func (r *ProgramRepository) MarkExerciseSkipped(exerciseID int) error {
	query := `
		UPDATE public.workout_exercises
		SET actual_sets = 0, actual_reps = 0, actual_weight = 0, completed = false
		WHERE id = $1`
	_, err := r.db.Exec(query, exerciseID)
	return err
}

// GetClientPrograms возвращает все программы клиента
func (r *ProgramRepository) GetClientPrograms(clientID int) ([]models.Program, error) {
	query := `
		SELECT id, client_id, name, COALESCE(goal, ''), total_weeks, days_per_week,
		       start_date, end_date, status, COALESCE(current_week, 1), created_at
		FROM public.training_programs
		WHERE client_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programs []models.Program
	for rows.Next() {
		var p models.Program
		var endDate sql.NullTime
		err := rows.Scan(
			&p.ID, &p.ClientID, &p.Name, &p.Goal,
			&p.TotalWeeks, &p.DaysPerWeek, &p.StartDate,
			&endDate, &p.Status, &p.CurrentWeek, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if endDate.Valid {
			p.EndDate = &endDate.Time
		}
		programs = append(programs, p)
	}

	return programs, nil
}

// GetPendingWorkoutsForClient возвращает ожидающие тренировки клиента
func (r *ProgramRepository) GetPendingWorkoutsForClient(clientID int) ([]models.Workout, error) {
	query := `
		SELECT pw.id, pw.program_id, pw.week_num, pw.day_num, pw.order_in_week, pw.name,
		       pw.planned_date, pw.status, COALESCE(pw.notes, ''), COALESCE(pw.feedback, ''),
		       pw.completed_at, pw.sent_at
		FROM public.program_workouts pw
		JOIN public.training_programs tp ON pw.program_id = tp.id
		WHERE tp.client_id = $1 AND tp.status = 'active' AND pw.status = 'pending'
		ORDER BY pw.week_num, pw.order_in_week
		LIMIT 10`

	rows, err := r.db.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workouts []models.Workout
	for rows.Next() {
		var w models.Workout
		var plannedDate, completedAt, sentAt sql.NullTime
		err := rows.Scan(
			&w.ID, &w.ProgramID, &w.WeekNum, &w.DayNum, &w.OrderInWeek, &w.Name,
			&plannedDate, &w.Status, &w.Notes, &w.Feedback,
			&completedAt, &sentAt,
		)
		if err != nil {
			return nil, err
		}

		if plannedDate.Valid {
			w.Date = &plannedDate.Time
		}
		if completedAt.Valid {
			w.CompletedAt = &completedAt.Time
		}
		if sentAt.Valid {
			w.SentAt = &sentAt.Time
		}

		workouts = append(workouts, w)
	}

	return workouts, nil
}

// UpdateWorkoutFeedback обновляет обратную связь по тренировке
func (r *ProgramRepository) UpdateWorkoutFeedback(workoutID int, feedback string) error {
	query := `UPDATE public.program_workouts SET feedback = $1 WHERE id = $2`
	_, err := r.db.Exec(query, feedback, workoutID)
	return err
}

// GetClientByTelegramID возвращает client_id по telegram_id
func (r *ProgramRepository) GetClientByTelegramID(telegramID int64) (int, error) {
	var clientID int
	err := r.db.QueryRow(`SELECT id FROM public.clients WHERE telegram_id = $1`, telegramID).Scan(&clientID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return clientID, err
}

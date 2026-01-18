package repository

import (
	"database/sql"
	"strings"
	"time"

	"workbot/internal/models"
)

// ExerciseRepository работает с упражнениями и 1ПМ
type ExerciseRepository struct {
	db *sql.DB
}

// NewExerciseRepository создаёт репозиторий упражнений
func NewExerciseRepository(db *sql.DB) *ExerciseRepository {
	return &ExerciseRepository{db: db}
}

// GetAll возвращает все упражнения
func (r *ExerciseRepository) GetAll() ([]models.Exercise, error) {
	rows, err := r.db.Query(`
		SELECT id, name, name_normalized, COALESCE(muscle_group, ''),
		       COALESCE(movement_type, ''), COALESCE(equipment, ''),
		       COALESCE(is_trackable_1pm, false), created_at
		FROM public.exercises
		ORDER BY muscle_group, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []models.Exercise
	for rows.Next() {
		var e models.Exercise
		if err := rows.Scan(&e.ID, &e.Name, &e.NameNormalized, &e.MuscleGroup,
			&e.MovementType, &e.Equipment, &e.IsTrackable1PM, &e.CreatedAt); err != nil {
			continue
		}
		exercises = append(exercises, e)
	}
	return exercises, rows.Err()
}

// GetByID возвращает упражнение по ID
func (r *ExerciseRepository) GetByID(id int) (*models.Exercise, error) {
	e := &models.Exercise{}
	err := r.db.QueryRow(`
		SELECT id, name, name_normalized, COALESCE(muscle_group, ''),
		       COALESCE(movement_type, ''), COALESCE(equipment, ''),
		       COALESCE(is_trackable_1pm, false), created_at
		FROM public.exercises WHERE id = $1`, id).Scan(
		&e.ID, &e.Name, &e.NameNormalized, &e.MuscleGroup,
		&e.MovementType, &e.Equipment, &e.IsTrackable1PM, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// GetByMuscleGroup возвращает упражнения по группе мышц
func (r *ExerciseRepository) GetByMuscleGroup(muscleGroup string) ([]models.Exercise, error) {
	rows, err := r.db.Query(`
		SELECT id, name, name_normalized, muscle_group,
		       COALESCE(movement_type, ''), COALESCE(equipment, ''),
		       COALESCE(is_trackable_1pm, false), created_at
		FROM public.exercises
		WHERE muscle_group = $1
		ORDER BY name`, muscleGroup)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []models.Exercise
	for rows.Next() {
		var e models.Exercise
		if err := rows.Scan(&e.ID, &e.Name, &e.NameNormalized, &e.MuscleGroup,
			&e.MovementType, &e.Equipment, &e.IsTrackable1PM, &e.CreatedAt); err != nil {
			continue
		}
		exercises = append(exercises, e)
	}
	return exercises, rows.Err()
}

// GetMuscleGroups возвращает список всех групп мышц
func (r *ExerciseRepository) GetMuscleGroups() ([]string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT muscle_group
		FROM public.exercises
		WHERE muscle_group IS NOT NULL AND muscle_group != ''
		ORDER BY muscle_group`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			continue
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// GetTrackable1PM возвращает упражнения для отслеживания 1ПМ
func (r *ExerciseRepository) GetTrackable1PM() ([]models.Exercise, error) {
	rows, err := r.db.Query(`
		SELECT id, name, name_normalized, COALESCE(muscle_group, ''),
		       COALESCE(movement_type, ''), COALESCE(equipment, ''),
		       COALESCE(is_trackable_1pm, false), created_at
		FROM public.exercises
		WHERE is_trackable_1pm = true
		ORDER BY muscle_group, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []models.Exercise
	for rows.Next() {
		var e models.Exercise
		if err := rows.Scan(&e.ID, &e.Name, &e.NameNormalized, &e.MuscleGroup,
			&e.MovementType, &e.Equipment, &e.IsTrackable1PM, &e.CreatedAt); err != nil {
			continue
		}
		exercises = append(exercises, e)
	}
	return exercises, rows.Err()
}

// FindByName ищет упражнение по имени (нечёткий поиск)
func (r *ExerciseRepository) FindByName(name string) (*models.Exercise, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	e := &models.Exercise{}
	err := r.db.QueryRow(`
		SELECT id, name, name_normalized, COALESCE(muscle_group, ''),
		       COALESCE(movement_type, ''), COALESCE(equipment, ''),
		       COALESCE(is_trackable_1pm, false), created_at
		FROM public.exercises
		WHERE name_normalized = $1 OR name ILIKE $2
		LIMIT 1`, normalized, "%"+name+"%").Scan(
		&e.ID, &e.Name, &e.NameNormalized, &e.MuscleGroup,
		&e.MovementType, &e.Equipment, &e.IsTrackable1PM, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// SearchByName ищет все упражнения по имени
func (r *ExerciseRepository) SearchByName(query string) ([]models.Exercise, error) {
	normalized := strings.ToLower(strings.TrimSpace(query))
	rows, err := r.db.Query(`
		SELECT id, name, name_normalized, COALESCE(muscle_group, ''),
		       COALESCE(movement_type, ''), COALESCE(equipment, ''),
		       COALESCE(is_trackable_1pm, false), created_at
		FROM public.exercises
		WHERE name_normalized LIKE $1 OR name ILIKE $2
		ORDER BY
			CASE WHEN name_normalized = $3 THEN 0
			     WHEN name_normalized LIKE $4 THEN 1
			     ELSE 2
			END,
			name
		LIMIT 20`, "%"+normalized+"%", "%"+query+"%", normalized, normalized+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []models.Exercise
	for rows.Next() {
		var e models.Exercise
		if err := rows.Scan(&e.ID, &e.Name, &e.NameNormalized, &e.MuscleGroup,
			&e.MovementType, &e.Equipment, &e.IsTrackable1PM, &e.CreatedAt); err != nil {
			continue
		}
		exercises = append(exercises, e)
	}
	return exercises, rows.Err()
}

// Create создаёт новое упражнение
func (r *ExerciseRepository) Create(name, muscleGroup, movementType, equipment string, isTrackable1PM bool) (int, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.exercises (name, name_normalized, muscle_group, movement_type, equipment, is_trackable_1pm)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name_normalized) DO UPDATE SET name = $1, is_trackable_1pm = $6
		RETURNING id`,
		name, normalized, muscleGroup, movementType, equipment, isTrackable1PM,
	).Scan(&id)
	return id, err
}

// Update обновляет упражнение
func (r *ExerciseRepository) Update(id int, name, muscleGroup, movementType, equipment string, isTrackable1PM bool) error {
	normalized := strings.ToLower(strings.TrimSpace(name))
	_, err := r.db.Exec(`
		UPDATE public.exercises
		SET name = $1, name_normalized = $2, muscle_group = $3,
		    movement_type = $4, equipment = $5, is_trackable_1pm = $6
		WHERE id = $7`,
		name, normalized, muscleGroup, movementType, equipment, isTrackable1PM, id)
	return err
}

// Delete удаляет упражнение
func (r *ExerciseRepository) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM public.exercises WHERE id = $1`, id)
	return err
}

// Save1PM сохраняет 1ПМ
func (r *ExerciseRepository) Save1PM(clientID, exerciseID int, onePM float64, testDate time.Time, method string, sourceWeight float64, sourceReps int, notes string, createdBy int64) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.exercise_1pm (client_id, exercise_id, one_pm_kg, test_date, calc_method, source_weight, source_reps, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		clientID, exerciseID, onePM, testDate, method,
		sql.NullFloat64{Float64: sourceWeight, Valid: sourceWeight > 0},
		sql.NullInt32{Int32: int32(sourceReps), Valid: sourceReps > 0},
		notes, createdBy,
	).Scan(&id)
	return id, err
}

// Update1PM обновляет запись 1ПМ
func (r *ExerciseRepository) Update1PM(id int, onePM float64, testDate time.Time, method string, sourceWeight float64, sourceReps int, notes string) error {
	_, err := r.db.Exec(`
		UPDATE public.exercise_1pm
		SET one_pm_kg = $1, test_date = $2, calc_method = $3,
		    source_weight = $4, source_reps = $5, notes = $6
		WHERE id = $7`,
		onePM, testDate, method,
		sql.NullFloat64{Float64: sourceWeight, Valid: sourceWeight > 0},
		sql.NullInt32{Int32: int32(sourceReps), Valid: sourceReps > 0},
		notes, id)
	return err
}

// Delete1PM удаляет запись 1ПМ
func (r *ExerciseRepository) Delete1PM(id int) error {
	_, err := r.db.Exec(`DELETE FROM public.exercise_1pm WHERE id = $1`, id)
	return err
}

// GetLatest1PM возвращает последний 1ПМ для клиента и упражнения
func (r *ExerciseRepository) GetLatest1PM(clientID, exerciseID int) (*models.Exercise1PM, error) {
	pm := &models.Exercise1PM{}
	var sourceWeight sql.NullFloat64
	var sourceReps sql.NullInt32
	var notes sql.NullString
	var createdBy sql.NullInt64

	err := r.db.QueryRow(`
		SELECT p.id, p.client_id, p.exercise_id, e.name, p.one_pm_kg, p.test_date,
		       COALESCE(p.calc_method, 'manual'), p.source_weight, p.source_reps,
		       p.notes, p.created_at, p.created_by
		FROM public.exercise_1pm p
		JOIN public.exercises e ON e.id = p.exercise_id
		WHERE p.client_id = $1 AND p.exercise_id = $2
		ORDER BY p.test_date DESC
		LIMIT 1`, clientID, exerciseID).Scan(
		&pm.ID, &pm.ClientID, &pm.ExerciseID, &pm.ExerciseName, &pm.OnePMKg, &pm.TestDate,
		&pm.CalcMethod, &sourceWeight, &sourceReps, &notes, &pm.CreatedAt, &createdBy,
	)
	if err != nil {
		return nil, err
	}

	if sourceWeight.Valid {
		pm.SourceWeight = sourceWeight.Float64
	}
	if sourceReps.Valid {
		pm.SourceReps = int(sourceReps.Int32)
	}
	if notes.Valid {
		pm.Notes = notes.String
	}
	if createdBy.Valid {
		pm.CreatedBy = createdBy.Int64
	}

	return pm, nil
}

// Get1PMHistory возвращает историю 1ПМ для клиента и упражнения
func (r *ExerciseRepository) Get1PMHistory(clientID, exerciseID int, limit int) ([]models.Exercise1PM, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.client_id, p.exercise_id, e.name, p.one_pm_kg, p.test_date,
		       COALESCE(p.calc_method, 'manual'), p.source_weight, p.source_reps,
		       p.notes, p.created_at, p.created_by
		FROM public.exercise_1pm p
		JOIN public.exercises e ON e.id = p.exercise_id
		WHERE p.client_id = $1 AND p.exercise_id = $2
		ORDER BY p.test_date DESC
		LIMIT $3`, clientID, exerciseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.Exercise1PM
	for rows.Next() {
		var pm models.Exercise1PM
		var sourceWeight sql.NullFloat64
		var sourceReps sql.NullInt32
		var notes sql.NullString
		var createdBy sql.NullInt64

		if err := rows.Scan(&pm.ID, &pm.ClientID, &pm.ExerciseID, &pm.ExerciseName,
			&pm.OnePMKg, &pm.TestDate, &pm.CalcMethod, &sourceWeight, &sourceReps,
			&notes, &pm.CreatedAt, &createdBy); err != nil {
			continue
		}

		if sourceWeight.Valid {
			pm.SourceWeight = sourceWeight.Float64
		}
		if sourceReps.Valid {
			pm.SourceReps = int(sourceReps.Int32)
		}
		if notes.Valid {
			pm.Notes = notes.String
		}
		if createdBy.Valid {
			pm.CreatedBy = createdBy.Int64
		}

		history = append(history, pm)
	}
	return history, rows.Err()
}

// GetClient1PMMap возвращает карту последних 1ПМ для всех упражнений клиента
func (r *ExerciseRepository) GetClient1PMMap(clientID int) (map[int]float64, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT ON (exercise_id) exercise_id, one_pm_kg
		FROM public.exercise_1pm
		WHERE client_id = $1
		ORDER BY exercise_id, test_date DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]float64)
	for rows.Next() {
		var exerciseID int
		var onePM float64
		if err := rows.Scan(&exerciseID, &onePM); err != nil {
			continue
		}
		result[exerciseID] = onePM
	}
	return result, rows.Err()
}

// GetClientAll1PM возвращает все последние 1ПМ клиента с информацией об упражнениях
func (r *ExerciseRepository) GetClientAll1PM(clientID int) ([]models.Exercise1PM, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT ON (p.exercise_id)
		       p.id, p.client_id, p.exercise_id, e.name, p.one_pm_kg, p.test_date,
		       COALESCE(p.calc_method, 'manual'), p.source_weight, p.source_reps,
		       p.notes, p.created_at, p.created_by
		FROM public.exercise_1pm p
		JOIN public.exercises e ON e.id = p.exercise_id
		WHERE p.client_id = $1
		ORDER BY p.exercise_id, p.test_date DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.Exercise1PM
	for rows.Next() {
		var pm models.Exercise1PM
		var sourceWeight sql.NullFloat64
		var sourceReps sql.NullInt32
		var notes sql.NullString
		var createdBy sql.NullInt64

		if err := rows.Scan(&pm.ID, &pm.ClientID, &pm.ExerciseID, &pm.ExerciseName,
			&pm.OnePMKg, &pm.TestDate, &pm.CalcMethod, &sourceWeight, &sourceReps,
			&notes, &pm.CreatedAt, &createdBy); err != nil {
			continue
		}

		if sourceWeight.Valid {
			pm.SourceWeight = sourceWeight.Float64
		}
		if sourceReps.Valid {
			pm.SourceReps = int(sourceReps.Int32)
		}
		if notes.Valid {
			pm.Notes = notes.String
		}
		if createdBy.Valid {
			pm.CreatedBy = createdBy.Int64
		}

		records = append(records, pm)
	}
	return records, rows.Err()
}

// GetClient1PMSummary возвращает полную сводку 1ПМ клиента с историей
func (r *ExerciseRepository) GetClient1PMSummary(clientID int, clientName string) (*models.Client1PMSummary, error) {
	// Получаем все упражнения с 1ПМ для клиента
	rows, err := r.db.Query(`
		SELECT DISTINCT exercise_id
		FROM public.exercise_1pm
		WHERE client_id = $1`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exerciseIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		exerciseIDs = append(exerciseIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	summary := &models.Client1PMSummary{
		ClientID:   clientID,
		ClientName: clientName,
		Exercises:  make([]models.Exercise1PMHistory, 0, len(exerciseIDs)),
	}

	for _, exID := range exerciseIDs {
		history, err := r.Get1PMHistory(clientID, exID, 100)
		if err != nil || len(history) == 0 {
			continue
		}

		exercise, err := r.GetByID(exID)
		if err != nil {
			continue
		}

		exHistory := models.Exercise1PMHistory{
			ExerciseID:   exID,
			ExerciseName: exercise.Name,
			MuscleGroup:  exercise.MuscleGroup,
			Records:      history,
			CurrentPM:    history[0].OnePMKg,
			InitialPM:    history[len(history)-1].OnePMKg,
		}

		if exHistory.InitialPM > 0 {
			exHistory.GainKg = exHistory.CurrentPM - exHistory.InitialPM
			exHistory.GainPercent = (exHistory.GainKg / exHistory.InitialPM) * 100
		}

		if history[0].TestDate.After(summary.LastTestDate) {
			summary.LastTestDate = history[0].TestDate
		}

		summary.Exercises = append(summary.Exercises, exHistory)
	}

	return summary, nil
}

// GetClient1PMByName возвращает карту 1ПМ по названию упражнения для AI генератора
func (r *ExerciseRepository) GetClient1PMByName(clientID int) (map[string]float64, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT ON (p.exercise_id)
		       e.name, p.one_pm_kg
		FROM public.exercise_1pm p
		JOIN public.exercises e ON e.id = p.exercise_id
		WHERE p.client_id = $1
		ORDER BY p.exercise_id, p.test_date DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var name string
		var onePM float64
		if err := rows.Scan(&name, &onePM); err != nil {
			continue
		}
		result[name] = onePM
	}
	return result, rows.Err()
}

// GetExercisesWithClientPM возвращает список упражнений с текущими 1ПМ клиента
func (r *ExerciseRepository) GetExercisesWithClientPM(clientID int) ([]models.ExerciseListItem, error) {
	rows, err := r.db.Query(`
		SELECT e.id, e.name, COALESCE(e.muscle_group, ''),
		       CASE WHEN p.one_pm_kg IS NOT NULL THEN true ELSE false END as has_1pm,
		       COALESCE(p.one_pm_kg, 0) as current_1pm
		FROM public.exercises e
		LEFT JOIN LATERAL (
			SELECT one_pm_kg
			FROM public.exercise_1pm
			WHERE client_id = $1 AND exercise_id = e.id
			ORDER BY test_date DESC
			LIMIT 1
		) p ON true
		WHERE e.is_trackable_1pm = true
		ORDER BY e.muscle_group, e.name`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ExerciseListItem
	for rows.Next() {
		var item models.ExerciseListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.MuscleGroup, &item.Has1PM, &item.Current1PM); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

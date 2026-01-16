package repository

import (
	"database/sql"
	"time"
)

// Schedule представляет слот расписания тренера
type Schedule struct {
	ID           int
	TrainerID    int64
	DayOfWeek    int
	StartTime    string
	EndTime      string
	SlotDuration int
	IsActive     bool
	CreatedAt    time.Time
}

// ScheduleRepository работает с расписанием тренера
type ScheduleRepository struct {
	db *sql.DB
}

// NewScheduleRepository создаёт репозиторий расписания
func NewScheduleRepository(db *sql.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// Create создаёт или обновляет слот расписания
func (r *ScheduleRepository) Create(trainerID int64, dayOfWeek int, startTime, endTime string, slotDuration int) error {
	_, err := r.db.Exec(`
		INSERT INTO public.trainer_schedule (trainer_id, day_of_week, start_time, end_time, slot_duration)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (trainer_id, day_of_week, start_time)
		DO UPDATE SET end_time = $4, slot_duration = $5, is_active = true`,
		trainerID, dayOfWeek, startTime+":00", endTime+":00", slotDuration,
	)
	return err
}

// GetByTrainer возвращает активное расписание тренера
func (r *ScheduleRepository) GetByTrainer(trainerID int64) ([]Schedule, error) {
	rows, err := r.db.Query(`
		SELECT id, trainer_id, day_of_week,
		       TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'),
		       slot_duration, is_active, created_at
		FROM public.trainer_schedule
		WHERE trainer_id = $1 AND is_active = true
		ORDER BY day_of_week, start_time`, trainerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.TrainerID, &s.DayOfWeek,
			&s.StartTime, &s.EndTime, &s.SlotDuration, &s.IsActive, &s.CreatedAt); err != nil {
			continue
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

// GetByDayOfWeek возвращает расписание на конкретный день недели
func (r *ScheduleRepository) GetByDayOfWeek(dayOfWeek int) ([]Schedule, error) {
	rows, err := r.db.Query(`
		SELECT id, trainer_id, day_of_week,
		       TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'),
		       slot_duration, is_active, created_at
		FROM public.trainer_schedule
		WHERE day_of_week = $1 AND is_active = true
		ORDER BY start_time`, dayOfWeek)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.TrainerID, &s.DayOfWeek,
			&s.StartTime, &s.EndTime, &s.SlotDuration, &s.IsActive, &s.CreatedAt); err != nil {
			continue
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

// Deactivate деактивирует слот расписания (soft delete)
func (r *ScheduleRepository) Deactivate(id int, trainerID int64) error {
	_, err := r.db.Exec(
		"UPDATE public.trainer_schedule SET is_active = false WHERE id = $1 AND trainer_id = $2",
		id, trainerID,
	)
	return err
}

// GetByID возвращает слот по ID
func (r *ScheduleRepository) GetByID(id int) (*Schedule, error) {
	s := &Schedule{}
	err := r.db.QueryRow(`
		SELECT id, trainer_id, day_of_week,
		       TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'),
		       slot_duration, is_active, created_at
		FROM public.trainer_schedule WHERE id = $1`, id).Scan(
		&s.ID, &s.TrainerID, &s.DayOfWeek,
		&s.StartTime, &s.EndTime, &s.SlotDuration, &s.IsActive, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

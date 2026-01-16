package repository

import (
	"database/sql"
	"time"
)

// Appointment представляет запись на тренировку
type Appointment struct {
	ID              int
	ClientID        int
	TrainerID       int64
	AppointmentDate time.Time
	StartTime       string
	EndTime         string
	Status          string
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AppointmentWithClient содержит запись с данными клиента
type AppointmentWithClient struct {
	Appointment
	ClientName    string
	ClientSurname string
}

// AppointmentRepository работает с записями на тренировки
type AppointmentRepository struct {
	db *sql.DB
}

// NewAppointmentRepository создаёт репозиторий записей
func NewAppointmentRepository(db *sql.DB) *AppointmentRepository {
	return &AppointmentRepository{db: db}
}

// Create создаёт новую запись
func (r *AppointmentRepository) Create(clientID int, trainerID int64, date time.Time, startTime, endTime string) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.appointments (client_id, trainer_id, appointment_date, start_time, end_time, status)
		VALUES ($1, $2, $3, $4, $5, 'scheduled')
		RETURNING id`,
		clientID, trainerID, date.Format("2006-01-02"), startTime, endTime,
	).Scan(&id)
	return id, err
}

// GetByID возвращает запись по ID
func (r *AppointmentRepository) GetByID(id int) (*Appointment, error) {
	a := &Appointment{}
	err := r.db.QueryRow(`
		SELECT id, client_id, trainer_id, appointment_date,
		       TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'),
		       status, COALESCE(notes, ''), created_at, updated_at
		FROM public.appointments WHERE id = $1`, id).Scan(
		&a.ID, &a.ClientID, &a.TrainerID, &a.AppointmentDate,
		&a.StartTime, &a.EndTime, &a.Status, &a.Notes,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// GetByIDWithClient возвращает запись с данными клиента
func (r *AppointmentRepository) GetByIDWithClient(id int) (*AppointmentWithClient, error) {
	a := &AppointmentWithClient{}
	err := r.db.QueryRow(`
		SELECT a.id, a.client_id, a.trainer_id, a.appointment_date,
		       TO_CHAR(a.start_time, 'HH24:MI'), TO_CHAR(a.end_time, 'HH24:MI'),
		       a.status, COALESCE(a.notes, ''), a.created_at, a.updated_at,
		       c.name, c.surname
		FROM public.appointments a
		JOIN public.clients c ON a.client_id = c.id
		WHERE a.id = $1`, id).Scan(
		&a.ID, &a.ClientID, &a.TrainerID, &a.AppointmentDate,
		&a.StartTime, &a.EndTime, &a.Status, &a.Notes,
		&a.CreatedAt, &a.UpdatedAt,
		&a.ClientName, &a.ClientSurname,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// GetClientAppointments возвращает записи клиента
func (r *AppointmentRepository) GetClientAppointments(clientID int, limit int) ([]Appointment, error) {
	rows, err := r.db.Query(`
		SELECT id, client_id, trainer_id, appointment_date,
		       TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'),
		       status, COALESCE(notes, ''), created_at, updated_at
		FROM public.appointments
		WHERE client_id = $1 AND appointment_date >= CURRENT_DATE AND status != 'cancelled'
		ORDER BY appointment_date, start_time
		LIMIT $2`, clientID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []Appointment
	for rows.Next() {
		var a Appointment
		if err := rows.Scan(&a.ID, &a.ClientID, &a.TrainerID, &a.AppointmentDate,
			&a.StartTime, &a.EndTime, &a.Status, &a.Notes,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		appointments = append(appointments, a)
	}
	return appointments, nil
}

// GetTrainerAppointments возвращает записи тренера
func (r *AppointmentRepository) GetTrainerAppointments(trainerID int64, limit int) ([]AppointmentWithClient, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.client_id, a.trainer_id, a.appointment_date,
		       TO_CHAR(a.start_time, 'HH24:MI'), TO_CHAR(a.end_time, 'HH24:MI'),
		       a.status, COALESCE(a.notes, ''), a.created_at, a.updated_at,
		       c.name, c.surname
		FROM public.appointments a
		JOIN public.clients c ON a.client_id = c.id
		WHERE a.trainer_id = $1 AND a.appointment_date >= CURRENT_DATE
		ORDER BY a.appointment_date, a.start_time
		LIMIT $2`, trainerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []AppointmentWithClient
	for rows.Next() {
		var a AppointmentWithClient
		if err := rows.Scan(&a.ID, &a.ClientID, &a.TrainerID, &a.AppointmentDate,
			&a.StartTime, &a.EndTime, &a.Status, &a.Notes,
			&a.CreatedAt, &a.UpdatedAt,
			&a.ClientName, &a.ClientSurname); err != nil {
			continue
		}
		appointments = append(appointments, a)
	}
	return appointments, nil
}

// GetTrainerActiveAppointments возвращает активные записи тренера
func (r *AppointmentRepository) GetTrainerActiveAppointments(trainerID int64, limit int) ([]AppointmentWithClient, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.client_id, a.trainer_id, a.appointment_date,
		       TO_CHAR(a.start_time, 'HH24:MI'), TO_CHAR(a.end_time, 'HH24:MI'),
		       a.status, COALESCE(a.notes, ''), a.created_at, a.updated_at,
		       c.name, c.surname
		FROM public.appointments a
		JOIN public.clients c ON a.client_id = c.id
		WHERE a.trainer_id = $1 AND a.appointment_date >= CURRENT_DATE AND a.status != 'cancelled'
		ORDER BY a.appointment_date, a.start_time
		LIMIT $2`, trainerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []AppointmentWithClient
	for rows.Next() {
		var a AppointmentWithClient
		if err := rows.Scan(&a.ID, &a.ClientID, &a.TrainerID, &a.AppointmentDate,
			&a.StartTime, &a.EndTime, &a.Status, &a.Notes,
			&a.CreatedAt, &a.UpdatedAt,
			&a.ClientName, &a.ClientSurname); err != nil {
			continue
		}
		appointments = append(appointments, a)
	}
	return appointments, nil
}

// GetBookedSlots возвращает занятые слоты на дату
func (r *AppointmentRepository) GetBookedSlots(date time.Time) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT TO_CHAR(start_time, 'HH24:MI')
		FROM public.appointments
		WHERE appointment_date = $1 AND status != 'cancelled'`,
		date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []string
	for rows.Next() {
		var slot string
		if err := rows.Scan(&slot); err != nil {
			continue
		}
		slots = append(slots, slot)
	}
	return slots, nil
}

// UpdateStatus обновляет статус записи
func (r *AppointmentRepository) UpdateStatus(id int, status string) error {
	_, err := r.db.Exec(
		"UPDATE public.appointments SET status = $1, updated_at = NOW() WHERE id = $2",
		status, id,
	)
	return err
}

// GetClientTelegramID возвращает telegram_id клиента по записи
func (r *AppointmentRepository) GetClientTelegramID(appointmentID int) (int64, error) {
	var telegramID int64
	err := r.db.QueryRow(`
		SELECT COALESCE(c.telegram_id, 0)
		FROM public.appointments a
		JOIN public.clients c ON a.client_id = c.id
		WHERE a.id = $1`, appointmentID).Scan(&telegramID)
	return telegramID, err
}

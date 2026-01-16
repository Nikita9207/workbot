package repository

import (
	"database/sql"
	"time"
)

// Client представляет клиента из БД
type Client struct {
	ID           int
	TelegramID   int64
	Name         string
	Surname      string
	Phone        string
	BirthDate    string
	Goal         sql.NullString
	TrainingPlan sql.NullString
	Notes        sql.NullString
	CreatedAt    time.Time
	DeletedAt    sql.NullTime
}

// ClientRepository работает с таблицей clients
type ClientRepository struct {
	db *sql.DB
}

// NewClientRepository создаёт репозиторий клиентов
func NewClientRepository(db *sql.DB) *ClientRepository {
	return &ClientRepository{db: db}
}

// GetByID возвращает клиента по ID
func (r *ClientRepository) GetByID(id int) (*Client, error) {
	client := &Client{}
	err := r.db.QueryRow(`
		SELECT id, COALESCE(telegram_id, 0), name, surname,
		       COALESCE(phone, ''), COALESCE(birth_date, ''),
		       goal, training_plan, notes, created_at, deleted_at
		FROM public.clients
		WHERE id = $1`, id).Scan(
		&client.ID, &client.TelegramID, &client.Name, &client.Surname,
		&client.Phone, &client.BirthDate,
		&client.Goal, &client.TrainingPlan, &client.Notes,
		&client.CreatedAt, &client.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// GetByTelegramID возвращает клиента по Telegram ID
func (r *ClientRepository) GetByTelegramID(telegramID int64) (*Client, error) {
	client := &Client{}
	err := r.db.QueryRow(`
		SELECT id, telegram_id, name, surname,
		       COALESCE(phone, ''), COALESCE(birth_date, ''),
		       goal, training_plan, notes, created_at, deleted_at
		FROM public.clients
		WHERE telegram_id = $1`, telegramID).Scan(
		&client.ID, &client.TelegramID, &client.Name, &client.Surname,
		&client.Phone, &client.BirthDate,
		&client.Goal, &client.TrainingPlan, &client.Notes,
		&client.CreatedAt, &client.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// ExistsByTelegramID проверяет существование клиента
func (r *ClientRepository) ExistsByTelegramID(telegramID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM public.clients WHERE telegram_id = $1)",
		telegramID,
	).Scan(&exists)
	return exists, err
}

// GetAllActive возвращает всех активных клиентов (не админов, не удалённых)
func (r *ClientRepository) GetAllActive() ([]Client, error) {
	rows, err := r.db.Query(`
		SELECT c.id, COALESCE(c.telegram_id, 0), c.name, c.surname,
		       COALESCE(c.phone, ''), COALESCE(c.birth_date, ''),
		       c.goal, c.training_plan, c.notes, c.created_at, c.deleted_at
		FROM public.clients c
		LEFT JOIN public.admins a ON c.telegram_id = a.telegram_id
		WHERE a.telegram_id IS NULL AND c.deleted_at IS NULL
		ORDER BY c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		err := rows.Scan(
			&c.ID, &c.TelegramID, &c.Name, &c.Surname,
			&c.Phone, &c.BirthDate,
			&c.Goal, &c.TrainingPlan, &c.Notes,
			&c.CreatedAt, &c.DeletedAt,
		)
		if err != nil {
			continue
		}
		clients = append(clients, c)
	}
	return clients, nil
}

// Create создаёт нового клиента
func (r *ClientRepository) Create(telegramID int64, name, surname, phone, birthDate string) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO public.clients (telegram_id, name, surname, phone, birth_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		telegramID, name, surname, phone, birthDate,
	).Scan(&id)
	return id, err
}

// UpdateGoal обновляет цель клиента
func (r *ClientRepository) UpdateGoal(clientID int, goal string) error {
	_, err := r.db.Exec(
		"UPDATE public.clients SET goal = $1 WHERE id = $2",
		goal, clientID,
	)
	return err
}

// UpdateTrainingPlan обновляет план тренировок
func (r *ClientRepository) UpdateTrainingPlan(clientID int, plan string) error {
	_, err := r.db.Exec(
		"UPDATE public.clients SET training_plan = $1 WHERE id = $2",
		plan, clientID,
	)
	return err
}

// SoftDelete мягко удаляет клиента
func (r *ClientRepository) SoftDelete(clientID int) error {
	_, err := r.db.Exec(
		"UPDATE public.clients SET deleted_at = NOW() WHERE id = $1",
		clientID,
	)
	return err
}

// AddGoalHistory добавляет цель в историю
func (r *ClientRepository) AddGoalHistory(clientID int, goal string) error {
	_, err := r.db.Exec(
		"INSERT INTO public.client_goals (client_id, goal) VALUES ($1, $2)",
		clientID, goal,
	)
	return err
}

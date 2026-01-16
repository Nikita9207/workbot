package repository

import (
	"database/sql"
	"sync"
)

// AdminRepository работает с таблицей admins
type AdminRepository struct {
	db    *sql.DB
	cache sync.Map // кэш для IsAdmin
}

// NewAdminRepository создаёт репозиторий админов
func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// IsAdmin проверяет, является ли пользователь админом (с кэшированием)
func (r *AdminRepository) IsAdmin(telegramID int64) bool {
	// Проверяем кэш
	if cached, ok := r.cache.Load(telegramID); ok {
		return cached.(bool)
	}

	// Запрос к БД
	var exists bool
	err := r.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM public.admins WHERE telegram_id = $1)",
		telegramID,
	).Scan(&exists)
	if err != nil {
		return false
	}

	// Кэшируем
	r.cache.Store(telegramID, exists)
	return exists
}

// ClearCache очищает кэш (например, при добавлении нового админа)
func (r *AdminRepository) ClearCache() {
	r.cache = sync.Map{}
}

// GetFirst возвращает первого админа (тренера)
func (r *AdminRepository) GetFirst() (int64, error) {
	var telegramID int64
	err := r.db.QueryRow("SELECT telegram_id FROM public.admins LIMIT 1").Scan(&telegramID)
	return telegramID, err
}

// GetAll возвращает всех админов
func (r *AdminRepository) GetAll() ([]int64, error) {
	rows, err := r.db.Query("SELECT telegram_id FROM public.admins")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		admins = append(admins, id)
	}
	return admins, nil
}

package models

import "time"

// ProgramStatus статус программы тренировок
type ProgramStatus string

const (
	ProgramStatusActive    ProgramStatus = "active"
	ProgramStatusCompleted ProgramStatus = "completed"
	ProgramStatusPaused    ProgramStatus = "paused"
)

// WorkoutStatus статус отдельной тренировки
type WorkoutStatus string

const (
	WorkoutStatusPending   WorkoutStatus = "pending"   // Ожидает выполнения
	WorkoutStatusSent      WorkoutStatus = "sent"      // Отправлена клиенту
	WorkoutStatusCompleted WorkoutStatus = "completed" // Выполнена
	WorkoutStatusSkipped   WorkoutStatus = "skipped"   // Пропущена
)

// Program представляет полную программу тренировок клиента
type Program struct {
	ID          int           `json:"id"`
	ClientID    int           `json:"client_id"`
	ClientName  string        `json:"client_name"`
	Name        string        `json:"name"`        // Название программы
	Goal        string        `json:"goal"`        // Цель тренировок
	Description string        `json:"description"` // Описание программы
	TotalWeeks  int           `json:"total_weeks"` // Всего недель
	DaysPerWeek int           `json:"days_per_week"`
	StartDate   time.Time     `json:"start_date"`
	EndDate     *time.Time    `json:"end_date"`
	Status      ProgramStatus `json:"status"`
	CurrentWeek int           `json:"current_week"` // Текущая неделя
	Workouts    []Workout     `json:"workouts"`     // Все тренировки программы
	FilePath    string        `json:"file_path"`    // Путь к Excel файлу
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Workout представляет одну тренировку в программе
type Workout struct {
	ID          int             `json:"id"`
	ProgramID   int             `json:"program_id"`
	WeekNum     int             `json:"week_num"`     // Номер недели (1, 2, 3...)
	DayNum      int             `json:"day_num"`      // День недели (1-7)
	OrderInWeek int             `json:"order_in_week"` // Порядок в неделе (1, 2, 3...)
	Name        string          `json:"name"`         // "День 1 - Грудь/Трицепс"
	Date        *time.Time      `json:"date"`         // Планируемая дата
	Status      WorkoutStatus   `json:"status"`
	Exercises   []WorkoutExercise `json:"exercises"`
	Notes       string          `json:"notes"`        // Заметки тренера
	Feedback    string          `json:"feedback"`     // Обратная связь клиента
	CompletedAt *time.Time      `json:"completed_at"`
	SentAt      *time.Time      `json:"sent_at"`
}

// WorkoutExercise представляет упражнение в тренировке
type WorkoutExercise struct {
	ID            int     `json:"id"`
	WorkoutID     int     `json:"workout_id"`
	OrderNum      int     `json:"order_num"`      // Порядок в тренировке
	ExerciseName  string  `json:"exercise_name"`  // Название упражнения
	Sets          int     `json:"sets"`           // Количество подходов
	Reps          string  `json:"reps"`           // Повторения (может быть "8-10")
	Weight        float64 `json:"weight"`         // Рабочий вес
	WeightPercent float64 `json:"weight_percent"` // % от 1ПМ (если известен)
	RestSeconds   int     `json:"rest_seconds"`   // Отдых между подходами
	Tempo         string  `json:"tempo"`          // Темп выполнения (3-1-2-0)
	RPE           float64 `json:"rpe"`            // Целевой RPE
	Notes         string  `json:"notes"`          // Заметки к упражнению

	// Результаты выполнения (заполняются после)
	ActualSets   int     `json:"actual_sets"`
	ActualReps   int     `json:"actual_reps"`
	ActualWeight float64 `json:"actual_weight"`
	ActualRPE    float64 `json:"actual_rpe"`
	Completed    bool    `json:"completed"`
}

// ClientForm представляет анкету клиента
type ClientForm struct {
	ID              int        `json:"id"`
	ClientID        int        `json:"client_id"`
	TelegramID      int64      `json:"telegram_id"`
	Name            string     `json:"name"`
	Surname         string     `json:"surname"`
	Phone           string     `json:"phone"`
	BirthDate       string     `json:"birth_date"`
	Gender          string     `json:"gender"`
	Height          int        `json:"height"`
	Weight          float64    `json:"weight"`
	Goal            string     `json:"goal"`           // Основная цель
	GoalDetails     string     `json:"goal_details"`   // Детали цели
	Experience      string     `json:"experience"`     // Опыт тренировок
	ExperienceYears float64    `json:"experience_years"`
	TrainingDays    int        `json:"training_days"`  // Сколько дней в неделю может
	Injuries        string     `json:"injuries"`       // Травмы/ограничения
	Equipment       string     `json:"equipment"`      // Доступное оборудование
	Preferences     string     `json:"preferences"`    // Предпочтения
	Notes           string     `json:"notes"`          // Дополнительные заметки
	FilePath        string     `json:"file_path"`      // Путь к файлу анкеты
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// JournalEntry представляет запись в общем журнале
type JournalEntry struct {
	ID              int        `json:"id"`
	ClientID        int        `json:"client_id"`
	Name            string     `json:"name"`
	Surname         string     `json:"surname"`
	Phone           string     `json:"phone"`
	TelegramID      int64      `json:"telegram_id"`
	Goal            string     `json:"goal"`
	StartDate       time.Time  `json:"start_date"`      // Дата начала занятий
	TotalWorkouts   int        `json:"total_workouts"`  // Всего тренировок
	CompletedWorkouts int      `json:"completed_workouts"` // Выполнено
	CurrentProgram  string     `json:"current_program"` // Текущая программа
	Status          string     `json:"status"`          // active/paused/completed
	Notes           string     `json:"notes"`
	LastWorkout     *time.Time `json:"last_workout"`
}

// GetNextWorkout возвращает следующую невыполненную тренировку
func (p *Program) GetNextWorkout() *Workout {
	for i := range p.Workouts {
		if p.Workouts[i].Status == WorkoutStatusPending {
			return &p.Workouts[i]
		}
	}
	return nil
}

// GetWorkoutsByWeek возвращает тренировки определённой недели
func (p *Program) GetWorkoutsByWeek(weekNum int) []Workout {
	var result []Workout
	for _, w := range p.Workouts {
		if w.WeekNum == weekNum {
			result = append(result, w)
		}
	}
	return result
}

// GetProgress возвращает прогресс программы (0-100%)
func (p *Program) GetProgress() float64 {
	if len(p.Workouts) == 0 {
		return 0
	}
	completed := 0
	for _, w := range p.Workouts {
		if w.Status == WorkoutStatusCompleted {
			completed++
		}
	}
	return float64(completed) / float64(len(p.Workouts)) * 100
}

// GetCompletedCount возвращает количество выполненных тренировок
func (p *Program) GetCompletedCount() int {
	count := 0
	for _, w := range p.Workouts {
		if w.Status == WorkoutStatusCompleted {
			count++
		}
	}
	return count
}

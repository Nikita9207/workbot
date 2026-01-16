package models

import "time"

// Macrocycle represents a year-level training period
type Macrocycle struct {
	ID             int         `json:"id"`
	TrainingPlanID int         `json:"training_plan_id"`
	Name           string      `json:"name"`
	StartDate      time.Time   `json:"start_date"`
	EndDate        time.Time   `json:"end_date"`
	MainGoal       string      `json:"main_goal"`
	Notes          string      `json:"notes"`
	Mesocycles     []Mesocycle `json:"mesocycles"`
	CreatedAt      time.Time   `json:"created_at"`
}

// Mesocycle represents a 4-6 week training block
type Mesocycle struct {
	ID               int          `json:"id"`
	MacrocycleID     *int         `json:"macrocycle_id"`
	TrainingPlanID   int          `json:"training_plan_id"`
	Name             string       `json:"name"` // "Втягивающий", "Базовый", "Пиковый"
	Phase            PlanPhase    `json:"phase"`
	WeekStart        int          `json:"week_start"`
	WeekEnd          int          `json:"week_end"`
	VolumePercent    int          `json:"volume_percent"`
	IntensityPercent int          `json:"intensity_percent"` // целевая интенсивность % от 1ПМ
	RPETarget        float64      `json:"rpe_target"`
	Notes            string       `json:"notes"`
	OrderNum         int          `json:"order_num"`
	Microcycles      []Microcycle `json:"microcycles"`
	CreatedAt        time.Time    `json:"created_at"`
}

// GetWeekCount returns number of weeks in mesocycle
func (m *Mesocycle) GetWeekCount() int {
	return m.WeekEnd - m.WeekStart + 1
}

// Microcycle represents a single training week
type Microcycle struct {
	ID                int            `json:"id"`
	MesocycleID       int            `json:"mesocycle_id"`
	WeekNumber        int            `json:"week_number"` // абсолютный номер в плане
	Name              string         `json:"name"`
	IsDeload          bool           `json:"is_deload"`
	VolumeModifier    float64        `json:"volume_modifier"`    // 0.5 для deload
	IntensityModifier float64        `json:"intensity_modifier"` // 0.7 для deload
	Notes             string         `json:"notes"`
	Exercises         []PlanExercise `json:"exercises"`
	CreatedAt         time.Time      `json:"created_at"`
}

// PlanExercise represents an exercise within a training day
type PlanExercise struct {
	ID               int     `json:"id"`
	MicrocycleID     int     `json:"microcycle_id"`
	ExerciseID       int     `json:"exercise_id"`
	ExerciseName     string  `json:"exercise_name"` // joined
	MuscleGroup      string  `json:"muscle_group"`  // joined
	DayOfWeek        int     `json:"day_of_week"`   // 1-7 (1=Пн)
	OrderNum         int     `json:"order_num"`
	Sets             int     `json:"sets"`
	RepsMin          int     `json:"reps_min"`
	RepsMax          int     `json:"reps_max"`
	IntensityPercent float64 `json:"intensity_percent"` // % от 1ПМ
	RPETarget        float64 `json:"rpe_target"`
	RestSeconds      int     `json:"rest_seconds"`
	Tempo            string  `json:"tempo"` // "3-1-2-0" формат
	Notes            string  `json:"notes"`
	CalculatedWeight float64 `json:"calculated_weight"` // вычислено из 1ПМ
}

// GetRepsString returns formatted reps string like "8-12" or "8"
func (p *PlanExercise) GetRepsString() string {
	if p.RepsMin == p.RepsMax {
		return string(rune('0' + p.RepsMin))
	}
	return string(rune('0'+p.RepsMin)) + "-" + string(rune('0'+p.RepsMax))
}

// Progression represents week-by-week weight/volume progression
type Progression struct {
	ID               int       `json:"id"`
	TrainingPlanID   int       `json:"training_plan_id"`
	ExerciseID       int       `json:"exercise_id"`
	ExerciseName     string    `json:"exercise_name"`
	MuscleGroup      string    `json:"muscle_group"`
	WeekNumber       int       `json:"week_number"`
	DayOfWeek        int       `json:"day_of_week"`
	Sets             int       `json:"sets"`
	Reps             int       `json:"reps"`
	WeightKg         float64   `json:"weight_kg"`
	IntensityPercent float64   `json:"intensity_percent"`
	IsDeload         bool      `json:"is_deload"`
	Notes            string    `json:"notes"`
	CreatedAt        time.Time `json:"created_at"`
}

// WeekProgression groups all exercises for a single week
type WeekProgression struct {
	WeekNumber int           `json:"week_number"`
	IsDeload   bool          `json:"is_deload"`
	PhaseName  string        `json:"phase_name"`
	Exercises  []Progression `json:"exercises"`
}

// DayProgression groups exercises for a single day
type DayProgression struct {
	DayOfWeek int           `json:"day_of_week"`
	DayName   string        `json:"day_name"` // "Понедельник"
	Exercises []Progression `json:"exercises"`
}

// ProgressionTable full table for Excel export
type ProgressionTable struct {
	PlanName      string             `json:"plan_name"`
	ClientName    string             `json:"client_name"`
	TotalWeeks    int                `json:"total_weeks"`
	ExerciseNames []string           `json:"exercise_names"`
	Weeks         []WeekProgression  `json:"weeks"`
	ByExercise    map[string][]Progression `json:"by_exercise"` // exercise name -> all weeks
}

// GetDayName returns Russian day name
func GetDayName(day int) string {
	days := []string{"", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}
	if day >= 1 && day <= 7 {
		return days[day]
	}
	return ""
}

// GetDayShortName returns short day name
func GetDayShortName(day int) string {
	days := []string{"", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
	if day >= 1 && day <= 7 {
		return days[day]
	}
	return ""
}

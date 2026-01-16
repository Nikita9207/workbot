package models

import "time"

// PlanStatus represents the status of a training plan
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusActive    PlanStatus = "active"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusArchived  PlanStatus = "archived"
)

// PlanPhase represents phase types for periodization
type PlanPhase string

const (
	PhaseHypertrophy PlanPhase = "hypertrophy"
	PhaseStrength    PlanPhase = "strength"
	PhasePower       PlanPhase = "power"
	PhasePeaking     PlanPhase = "peaking"
	PhaseDeload      PlanPhase = "deload"
)

// PhaseNameRu returns Russian name for phase
func (p PlanPhase) NameRu() string {
	switch p {
	case PhaseHypertrophy:
		return "Гипертрофия"
	case PhaseStrength:
		return "Сила"
	case PhasePower:
		return "Мощность"
	case PhasePeaking:
		return "Пик"
	case PhaseDeload:
		return "Разгрузка"
	default:
		return string(p)
	}
}

// PhaseConfig contains default parameters for each phase
type PhaseConfig struct {
	Phase            PlanPhase
	NameRu           string
	SetsRange        [2]int     // min, max
	RepsRange        [2]int     // min, max
	IntensityRange   [2]float64 // % of 1PM
	RestSeconds      [2]int     // min, max
	RPETarget        float64
	VolumeMultiplier float64
}

// DefaultPhaseConfigs returns standard phase configurations
var DefaultPhaseConfigs = map[PlanPhase]PhaseConfig{
	PhaseHypertrophy: {
		Phase:            PhaseHypertrophy,
		NameRu:           "Гипертрофия",
		SetsRange:        [2]int{3, 4},
		RepsRange:        [2]int{8, 12},
		IntensityRange:   [2]float64{65, 80},
		RestSeconds:      [2]int{60, 90},
		RPETarget:        7.5,
		VolumeMultiplier: 1.0,
	},
	PhaseStrength: {
		Phase:            PhaseStrength,
		NameRu:           "Сила",
		SetsRange:        [2]int{4, 5},
		RepsRange:        [2]int{4, 6},
		IntensityRange:   [2]float64{80, 90},
		RestSeconds:      [2]int{180, 300},
		RPETarget:        8.5,
		VolumeMultiplier: 0.8,
	},
	PhasePower: {
		Phase:            PhasePower,
		NameRu:           "Мощность",
		SetsRange:        [2]int{5, 6},
		RepsRange:        [2]int{2, 4},
		IntensityRange:   [2]float64{85, 95},
		RestSeconds:      [2]int{180, 300},
		RPETarget:        8.0,
		VolumeMultiplier: 0.6,
	},
	PhasePeaking: {
		Phase:            PhasePeaking,
		NameRu:           "Пик",
		SetsRange:        [2]int{3, 5},
		RepsRange:        [2]int{1, 3},
		IntensityRange:   [2]float64{90, 100},
		RestSeconds:      [2]int{240, 360},
		RPETarget:        9.5,
		VolumeMultiplier: 0.5,
	},
	PhaseDeload: {
		Phase:            PhaseDeload,
		NameRu:           "Разгрузка",
		SetsRange:        [2]int{2, 3},
		RepsRange:        [2]int{6, 8},
		IntensityRange:   [2]float64{50, 60},
		RestSeconds:      [2]int{60, 90},
		RPETarget:        5.0,
		VolumeMultiplier: 0.4,
	},
}

// TrainingPlan represents a complete training program
type TrainingPlan struct {
	ID          int          `json:"id"`
	ClientID    int          `json:"client_id"`
	ClientName  string       `json:"client_name"` // joined
	Name        string       `json:"name"`
	Description string       `json:"description"`
	StartDate   time.Time    `json:"start_date"`
	EndDate     *time.Time   `json:"end_date"`
	Status      PlanStatus   `json:"status"`
	Goal        string       `json:"goal"`
	DaysPerWeek int          `json:"days_per_week"`
	TotalWeeks  int          `json:"total_weeks"`
	AIGenerated bool         `json:"ai_generated"`
	AIPrompt    string       `json:"ai_prompt"`
	Mesocycles  []Mesocycle  `json:"mesocycles"`
	Progression []Progression `json:"progression"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	CreatedBy   int64        `json:"created_by"`
}

// TrainingPlanListItem for displaying plan list
type TrainingPlanListItem struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	ClientName  string     `json:"client_name"`
	Status      PlanStatus `json:"status"`
	Goal        string     `json:"goal"`
	TotalWeeks  int        `json:"total_weeks"`
	CurrentWeek int        `json:"current_week"`
	StartDate   time.Time  `json:"start_date"`
}

// TrainingDayTemplate for reusable day structures
type TrainingDayTemplate struct {
	ID             int                `json:"id"`
	TrainingPlanID int                `json:"training_plan_id"`
	Name           string             `json:"name"` // "День А - Верх Push"
	DayType        string             `json:"day_type"` // push, pull, legs, upper, lower, fullbody
	Description    string             `json:"description"`
	Exercises      []TemplateExercise `json:"exercises"`
	CreatedAt      time.Time          `json:"created_at"`
}

// TemplateExercise represents exercise in a day template
type TemplateExercise struct {
	ID               int    `json:"id"`
	TemplateID       int    `json:"template_id"`
	ExerciseID       int    `json:"exercise_id"`
	ExerciseName     string `json:"exercise_name"`
	OrderNum         int    `json:"order_num"`
	SetsDefault      int    `json:"sets_default"`
	RepsMinDefault   int    `json:"reps_min_default"`
	RepsMaxDefault   int    `json:"reps_max_default"`
	RestSecondsDefault int  `json:"rest_seconds_default"`
	Notes            string `json:"notes"`
}

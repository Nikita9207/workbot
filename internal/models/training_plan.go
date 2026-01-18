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
	PhaseAccumulation PlanPhase = "accumulation" // Накопление объёма
	PhaseTransmutation PlanPhase = "transmutation" // Трансформация в силу
	PhaseRealization  PlanPhase = "realization"   // Реализация/пик
)

// TrainingPeriod представляет период годичного цикла
type TrainingPeriod string

const (
	PeriodPreparatory   TrainingPeriod = "preparatory"   // Подготовительный
	PeriodCompetitive   TrainingPeriod = "competitive"   // Соревновательный
	PeriodTransitional  TrainingPeriod = "transitional"  // Переходный
)

// PeriodNameRu returns Russian name for period
func (p TrainingPeriod) NameRu() string {
	switch p {
	case PeriodPreparatory:
		return "Подготовительный"
	case PeriodCompetitive:
		return "Соревновательный"
	case PeriodTransitional:
		return "Переходный"
	default:
		return string(p)
	}
}

// MesocycleType тип мезоцикла
type MesocycleType string

const (
	MesoIntroductory     MesocycleType = "introductory"      // Втягивающий
	MesoBasic            MesocycleType = "basic"             // Базовый
	MesoControlPrep      MesocycleType = "control_prep"      // Контрольно-подготовительный
	MesoPreCompetitive   MesocycleType = "pre_competitive"   // Предсоревновательный
	MesoCompetitive      MesocycleType = "competitive"       // Соревновательный
	MesoRecovery         MesocycleType = "recovery"          // Восстановительный
)

// MesoTypeNameRu returns Russian name for mesocycle type
func (m MesocycleType) NameRu() string {
	switch m {
	case MesoIntroductory:
		return "Втягивающий"
	case MesoBasic:
		return "Базовый"
	case MesoControlPrep:
		return "Контрольно-подготовительный"
	case MesoPreCompetitive:
		return "Предсоревновательный"
	case MesoCompetitive:
		return "Соревновательный"
	case MesoRecovery:
		return "Восстановительный"
	default:
		return string(m)
	}
}

// Methodology тип периодизации
type Methodology string

const (
	MethodLinear    Methodology = "linear"    // Линейная
	MethodDUP       Methodology = "dup"       // Волнообразная (Daily Undulating)
	MethodBlock     Methodology = "block"     // Блочная
	MethodConjugate Methodology = "conjugate" // Сопряжённая
	MethodHybrid    Methodology = "hybrid"    // Гибридная
)

// MethodologyNameRu returns Russian name
func (m Methodology) NameRu() string {
	switch m {
	case MethodLinear:
		return "Линейная"
	case MethodDUP:
		return "Волнообразная (DUP)"
	case MethodBlock:
		return "Блочная"
	case MethodConjugate:
		return "Сопряжённая"
	case MethodHybrid:
		return "Гибридная"
	default:
		return string(m)
	}
}

// WeekAccent акцент тренировочной недели
type WeekAccent string

const (
	AccentVolume      WeekAccent = "volume"       // Объём
	AccentIntensity   WeekAccent = "intensity"    // Интенсивность
	AccentTechnique   WeekAccent = "technique"    // Техника
	AccentSpeed       WeekAccent = "speed"        // Скорость/взрывная сила
	AccentEndurance   WeekAccent = "endurance"    // Выносливость
	AccentMaxStrength WeekAccent = "max_strength" // Максимальная сила
	AccentRecovery    WeekAccent = "recovery"     // Восстановление
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

	// Расширенные поля периодизации
	Methodology     Methodology       `json:"methodology"`      // Тип периодизации
	Period          TrainingPeriod    `json:"period"`           // Текущий период
	Weeks           []TrainingWeek    `json:"weeks"`            // Полная структура недель
	ProgressionRules *ProgressionRules `json:"progression_rules"` // Правила прогрессии
	OnePMData       map[string]float64 `json:"one_pm_data"`      // 1ПМ данные клиента
}

// TrainingWeek представляет полную неделю тренировок
type TrainingWeek struct {
	WeekNum          int            `json:"week_num"`
	Period           TrainingPeriod `json:"period"`            // Период (Подготовительный/Соревновательный/Переходный)
	MesocycleType    MesocycleType  `json:"mesocycle_type"`    // Тип мезоцикла
	Phase            PlanPhase      `json:"phase"`             // Фаза (hypertrophy/strength/power/deload)
	Focus            string         `json:"focus"`             // Фокус недели (текстовое описание)
	Accents          []WeekAccent   `json:"accents"`           // Акценты недели
	IntensityPercent float64        `json:"intensity_percent"` // Целевая интенсивность %
	VolumePercent    float64        `json:"volume_percent"`    // Объём относительно базового
	RPETarget        float64        `json:"rpe_target"`        // Целевой RPE
	IsDeload         bool           `json:"is_deload"`         // Разгрузочная неделя
	Notes            string         `json:"notes"`             // Заметки
	Workouts         []DayWorkout   `json:"workouts"`          // Тренировки недели
}

// DayWorkout представляет одну тренировку дня
type DayWorkout struct {
	DayNum            int               `json:"day_num"`             // Номер дня в неделе (1-7)
	Name              string            `json:"name"`                // "День 1 - Верх (Push)"
	Type              string            `json:"type"`                // push/pull/legs/upper/lower/fullbody
	MuscleGroups      []string          `json:"muscle_groups"`       // ["грудь", "плечи", "трицепс"]
	EstimatedDuration int               `json:"estimated_duration"`  // Примерная длительность (мин)
	Exercises         []WorkoutExerciseV2 `json:"exercises"`
}

// WorkoutExerciseV2 расширенная структура упражнения
type WorkoutExerciseV2 struct {
	OrderNum         int      `json:"order_num"`
	ExerciseName     string   `json:"exercise_name"`
	MuscleGroup      string   `json:"muscle_group"`
	MovementType     string   `json:"movement_type"` // compound/isolation
	Sets             int      `json:"sets"`
	Reps             string   `json:"reps"`          // "8-10" или "5"
	WeightPercent    float64  `json:"weight_percent"` // % от 1ПМ (если известен)
	WeightKg         float64  `json:"weight_kg"`      // Абсолютный вес в кг
	RestSeconds      int      `json:"rest_seconds"`
	Tempo            string   `json:"tempo"`          // "3-1-1-0"
	RPE              float64  `json:"rpe"`
	Notes            string   `json:"notes"`
	Alternatives     []string `json:"alternatives"`   // Альтернативные упражнения
	SupersetWith     string   `json:"superset_with"`  // Упражнение для суперсета
}

// ProgressionRules правила прогрессии
type ProgressionRules struct {
	CompoundIncrement        float64 `json:"compound_increment"`         // Шаг прогрессии для базовых (кг)
	IsolationIncrement       float64 `json:"isolation_increment"`        // Шаг прогрессии для изоляции (кг)
	DeloadFrequency          int     `json:"deload_frequency"`           // Частота разгрузок (недель)
	DeloadVolumeReduction    float64 `json:"deload_volume_reduction"`    // Снижение объёма при разгрузке
	DeloadIntensityReduction float64 `json:"deload_intensity_reduction"` // Снижение интенсивности при разгрузке
	WeeklyIntensityIncrease  float64 `json:"weekly_intensity_increase"`  // Еженедельное увеличение интенсивности
	WeeklyVolumeIncrease     float64 `json:"weekly_volume_increase"`     // Еженедельное увеличение объёма
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

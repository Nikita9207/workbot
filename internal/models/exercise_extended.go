package models

// ===============================================
// РАСШИРЕННЫЕ МОДЕЛИ УПРАЖНЕНИЙ ДЛЯ ГЕНЕРАТОРА
// ===============================================

// MovementType - тип движения
type MovementType string

const (
	MovementPush     MovementType = "push"     // Толкающие (жим, отжимания)
	MovementPull     MovementType = "pull"     // Тянущие (тяга, подтягивания)
	MovementHinge    MovementType = "hinge"    // Наклон/разгибание (RDL, тяга)
	MovementSquat    MovementType = "squat"    // Приседания
	MovementLunge    MovementType = "lunge"    // Выпады
	MovementCarry    MovementType = "carry"    // Переноски
	MovementRotation MovementType = "rotation" // Ротация (скручивания)
	MovementCardio   MovementType = "cardio"   // Кардио
	MovementPlyo     MovementType = "plyo"     // Плиометрика
	MovementCore     MovementType = "core"     // Кор/стабилизация
)

// MovementPlane - плоскость движения
type MovementPlane string

const (
	PlaneSagittal   MovementPlane = "sagittal"   // Сагиттальная (вперёд-назад)
	PlaneFrontal    MovementPlane = "frontal"    // Фронтальная (в стороны)
	PlaneTransverse MovementPlane = "transverse" // Поперечная (вращение)
)

// MuscleGroupExt - мышечная группа (расширенная)
type MuscleGroupExt string

const (
	MuscleChest        MuscleGroupExt = "chest"        // Грудь
	MuscleBack         MuscleGroupExt = "back"         // Спина (широчайшие)
	MuscleUpperBack    MuscleGroupExt = "upper_back"   // Верх спины
	MuscleShoulders    MuscleGroupExt = "shoulders"    // Плечи (дельты)
	MuscleRearDelts    MuscleGroupExt = "rear_delts"   // Задние дельты
	MuscleBiceps       MuscleGroupExt = "biceps"       // Бицепс
	MuscleTriceps      MuscleGroupExt = "triceps"      // Трицепс
	MuscleForearms     MuscleGroupExt = "forearms"     // Предплечья
	MuscleQuads        MuscleGroupExt = "quads"        // Квадрицепс
	MuscleHamstrings   MuscleGroupExt = "hamstrings"   // Бицепс бедра
	MuscleGlutes       MuscleGroupExt = "glutes"       // Ягодицы
	MuscleCalves       MuscleGroupExt = "calves"       // Икры
	MuscleCore         MuscleGroupExt = "core"         // Кор (пресс, косые)
	MuscleLowerBack    MuscleGroupExt = "lower_back"   // Поясница
	MuscleHipFlexors   MuscleGroupExt = "hip_flexors"  // Сгибатели бедра
	MuscleTraps        MuscleGroupExt = "traps"        // Трапеции
	MuscleAdductors    MuscleGroupExt = "adductors"    // Приводящие
	MuscleAbductors    MuscleGroupExt = "abductors"    // Отводящие
	MuscleFullBody     MuscleGroupExt = "full_body"    // Всё тело
	MuscleCardioSystem MuscleGroupExt = "cardio"       // Сердечно-сосудистая
)

// EquipmentType - оборудование
type EquipmentType string

const (
	EquipmentBarbell    EquipmentType = "barbell"    // Штанга
	EquipmentDumbbell   EquipmentType = "dumbbell"   // Гантели
	EquipmentKettlebell EquipmentType = "kettlebell" // Гири
	EquipmentCable      EquipmentType = "cable"      // Блоки/кроссовер
	EquipmentMachine    EquipmentType = "machine"    // Тренажёры
	EquipmentBodyweight EquipmentType = "bodyweight" // Собственный вес
	EquipmentTRX        EquipmentType = "trx"        // TRX петли
	EquipmentBands      EquipmentType = "bands"      // Резиновые ленты
	EquipmentSkiErg     EquipmentType = "ski_erg"    // Лыжный тренажёр
	EquipmentRowErg     EquipmentType = "row_erg"    // Гребной тренажёр
	EquipmentAssaultBike EquipmentType = "assault_bike" // Assault bike
	EquipmentSled       EquipmentType = "sled"       // Сани
	EquipmentBox        EquipmentType = "box"        // Тумба
	EquipmentPullupBar  EquipmentType = "pullup_bar" // Турник
	EquipmentBench      EquipmentType = "bench"      // Скамья
	EquipmentRack       EquipmentType = "rack"       // Силовая рама
	EquipmentMedball    EquipmentType = "medball"    // Медбол
	EquipmentWallBall   EquipmentType = "wall_ball"  // Wall ball
	EquipmentSandbag    EquipmentType = "sandbag"    // Сэндбэг
	EquipmentRope       EquipmentType = "rope"       // Канат
	EquipmentAbWheel    EquipmentType = "ab_wheel"   // Ролик для пресса
)

// LoadType - тип нагрузки
type LoadType string

const (
	LoadWeight   LoadType = "weight"   // Вес (кг)
	LoadReps     LoadType = "reps"     // Только повторения
	LoadTime     LoadType = "time"     // Время (сек)
	LoadTUT      LoadType = "tut"      // Time Under Tension
	LoadDistance LoadType = "distance" // Дистанция (м)
	LoadCalories LoadType = "calories" // Калории
	LoadLevel    LoadType = "level"    // Уровень сложности (TRX)
)

// MovementPattern - паттерн движения
type MovementPattern string

const (
	PatternBilateral   MovementPattern = "bilateral"   // Двусторонний
	PatternUnilateral  MovementPattern = "unilateral"  // Односторонний
	PatternAlternating MovementPattern = "alternating" // Попеременный
)

// KettlebellType - тип гиревого упражнения
type KettlebellType string

const (
	KBTypeBallistic KettlebellType = "ballistic" // Баллистика (свинги, рывки)
	KBTypeGrind     KettlebellType = "grind"     // Грайнд (жимы, тяги)
	KBTypeComplex   KettlebellType = "complex"   // Комплексы
)

// DifficultyLevel - уровень сложности
type DifficultyLevel int

const (
	DifficultyBeginner     DifficultyLevel = 1 // Новичок
	DifficultyIntermediate DifficultyLevel = 2 // Средний
	DifficultyAdvanced     DifficultyLevel = 3 // Продвинутый
)

// ExerciseExt - полная модель упражнения
type ExerciseExt struct {
	ID             string           `json:"id"`
	NameRu         string           `json:"name_ru"`          // Название (рус)
	NameEn         string           `json:"name_en"`          // Название (eng)
	MovementType   MovementType     `json:"movement_type"`    // Тип движения
	MovementPlane  MovementPlane    `json:"movement_plane"`   // Плоскость движения
	PrimaryMuscles []MuscleGroupExt `json:"primary_muscles"`  // Основные мышцы
	SecondaryMuscles []MuscleGroupExt `json:"secondary_muscles"` // Вспомогательные
	Equipment      []EquipmentType  `json:"equipment"`        // Требуемое оборудование
	LoadType       LoadType         `json:"load_type"`        // Тип нагрузки
	Pattern        MovementPattern  `json:"pattern"`          // Паттерн движения
	Difficulty     DifficultyLevel  `json:"difficulty"`       // Уровень сложности
	RequiresSpotter bool            `json:"requires_spotter"` // Требует страховки
	IsCompound     bool             `json:"is_compound"`      // Многосуставное

	// Для гирь
	KettlebellType KettlebellType `json:"kettlebell_type,omitempty"`

	// Для TRX
	TRXMinLevel    int    `json:"trx_min_level,omitempty"`    // Мин уровень (1-10)
	TRXMaxLevel    int    `json:"trx_max_level,omitempty"`    // Макс уровень (1-10)
	TRXBasePosition string `json:"trx_base_position,omitempty"` // Базовая позиция

	// Рекомендуемые диапазоны
	RecommendedRepsMin int `json:"recommended_reps_min"`
	RecommendedRepsMax int `json:"recommended_reps_max"`
	RecommendedSetsMin int `json:"recommended_sets_min"`
	RecommendedSetsMax int `json:"recommended_sets_max"`

	// Мета
	VideoURL     string   `json:"video_url,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

// ExerciseDB - база упражнений
type ExerciseDB struct {
	Exercises []ExerciseExt `json:"exercises"`
}

// ===============================================
// ПРОТИВОПОКАЗАНИЯ
// ===============================================

// ContraindicationType - тип противопоказания
type ContraindicationType string

const (
	ContraInjury      ContraindicationType = "injury"      // Травма
	ContraCondition   ContraindicationType = "condition"   // Состояние здоровья
	ContraPregnancy   ContraindicationType = "pregnancy"   // Беременность
	ContraHypertension ContraindicationType = "hypertension" // Гипертензия
)

// BodyZone - зона тела для противопоказаний
type BodyZone string

const (
	ZoneLowerBack BodyZone = "lower_back" // Поясница
	ZoneKnee      BodyZone = "knee"       // Колени
	ZoneShoulder  BodyZone = "shoulder"   // Плечи
	ZoneWrist     BodyZone = "wrist"      // Запястья
	ZoneCervical  BodyZone = "cervical"   // Шейный отдел
	ZoneHip       BodyZone = "hip"        // Тазобедренный
	ZoneAnkle     BodyZone = "ankle"      // Голеностоп
	ZoneElbow     BodyZone = "elbow"      // Локти
)

// ContraindicationSeverity - строгость противопоказания
type ContraindicationSeverity string

const (
	SeverityAbsolute ContraindicationSeverity = "absolute" // Абсолютное (нельзя)
	SeverityRelative ContraindicationSeverity = "relative" // Относительное (с осторожностью)
)

// Contraindication - противопоказание
type Contraindication struct {
	ExerciseID string                   `json:"exercise_id"`
	Type       ContraindicationType     `json:"type"`
	BodyZone   BodyZone                 `json:"body_zone"`
	Severity   ContraindicationSeverity `json:"severity"`
	Notes      string                   `json:"notes,omitempty"`
}

// ===============================================
// АЛЬТЕРНАТИВЫ УПРАЖНЕНИЙ
// ===============================================

// AlternativeReason - причина замены
type AlternativeReason string

const (
	ReasonMachineBusy       AlternativeReason = "machine_busy"       // Тренажёр занят
	ReasonEquipmentUnavail  AlternativeReason = "equipment_unavail"  // Нет оборудования
	ReasonRegression        AlternativeReason = "regression"         // Упрощение
	ReasonProgression       AlternativeReason = "progression"        // Усложнение
	ReasonContraindication  AlternativeReason = "contraindication"   // Противопоказание
)

// ExerciseAlternative - альтернатива упражнения
type ExerciseAlternative struct {
	ExerciseID    string            `json:"exercise_id"`
	AlternativeID string            `json:"alternative_id"`
	Priority      int               `json:"priority"` // 1 = лучшая замена
	Reason        AlternativeReason `json:"reason"`
}

// ===============================================
// КЛИЕНТ (РАСШИРЕННЫЙ)
// ===============================================

// ClientConstraint - ограничение клиента
type ClientConstraint struct {
	BodyZone BodyZone                 `json:"body_zone"`
	Severity ContraindicationSeverity `json:"severity"`
	Notes    string                   `json:"notes,omitempty"`
}

// TrainingLocation - место тренировок
type TrainingLocation string

const (
	LocationGym  TrainingLocation = "gym"  // Зал
	LocationHome TrainingLocation = "home" // Дома
)

// ExperienceLevel - уровень подготовки
type ExperienceLevel string

const (
	ExpBeginner     ExperienceLevel = "beginner"     // Новичок (< 1 год)
	ExpIntermediate ExperienceLevel = "intermediate" // Средний (1-3 года)
	ExpAdvanced     ExperienceLevel = "advanced"     // Продвинутый (3+ лет)
)

// ClientProfile - профиль клиента для генератора
type ClientProfile struct {
	ID               int                 `json:"id"`
	Name             string              `json:"name"`
	Gender           string              `json:"gender"` // male/female
	Age              int                 `json:"age"`
	Weight           float64             `json:"weight"` // кг
	Height           float64             `json:"height"` // см
	Experience       ExperienceLevel     `json:"experience"`
	Constraints      []ClientConstraint  `json:"constraints"`       // Ограничения здоровья
	AvailableEquip   []EquipmentType     `json:"available_equip"`   // Доступное оборудование
	AvailableKBWeights []float64         `json:"available_kb_weights"` // Доступные гири (кг)
	Location         TrainingLocation    `json:"location"`          // gym/home
	OnePM            map[string]float64  `json:"one_pm"`            // 1ПМ по движениям
}

// ===============================================
// ПРОГРАММА (РАСШИРЕННАЯ)
// ===============================================

// TrainingGoal - цель тренировок
type TrainingGoal string

const (
	GoalStrength    TrainingGoal = "strength"    // Сила
	GoalHypertrophy TrainingGoal = "hypertrophy" // Гипертрофия
	GoalFatLoss     TrainingGoal = "fat_loss"    // Жиросжигание
	GoalHyrox       TrainingGoal = "hyrox"       // Hyrox
	GoalEndurance   TrainingGoal = "endurance"   // Выносливость
	GoalGeneral     TrainingGoal = "general"     // ОФП
)

// PeriodizationType - тип периодизации
type PeriodizationType string

const (
	PeriodLinear      PeriodizationType = "linear"      // Линейная
	PeriodUndulating  PeriodizationType = "undulating"  // Волнообразная
	PeriodBlock       PeriodizationType = "block"       // Блочная
	PeriodReverse     PeriodizationType = "reverse"     // Обратная
)

// GeneratedProgram - сгенерированная программа
type GeneratedProgram struct {
	ClientID       int               `json:"client_id"`
	ClientName     string            `json:"client_name"`
	Goal           TrainingGoal      `json:"goal"`
	Periodization  PeriodizationType `json:"periodization"`
	TotalWeeks     int               `json:"total_weeks"`
	DaysPerWeek    int               `json:"days_per_week"`
	Phases         []ProgramPhase    `json:"phases"`
	Weeks          []GeneratedWeek   `json:"weeks"`
	Statistics     ProgramStats      `json:"statistics"`
	Substitutions  []Substitution    `json:"substitutions"` // Замены из-за ограничений
}

// ProgramPhase - фаза программы
type ProgramPhase struct {
	Name         string  `json:"name"`
	WeekStart    int     `json:"week_start"`
	WeekEnd      int     `json:"week_end"`
	Focus        string  `json:"focus"`
	IntensityMin float64 `json:"intensity_min"` // % от 1ПМ
	IntensityMax float64 `json:"intensity_max"`
	VolumeLevel  string  `json:"volume_level"` // high/medium/low
}

// GeneratedWeek - неделя программы
type GeneratedWeek struct {
	WeekNum          int             `json:"week_num"`
	PhaseName        string          `json:"phase_name"`
	IsDeload         bool            `json:"is_deload"`
	IntensityPercent float64         `json:"intensity_percent"`
	VolumePercent    float64         `json:"volume_percent"`
	RPETarget        float64         `json:"rpe_target"`
	WaveMultiplier   float64         `json:"wave_multiplier"`  // Множитель волновой периодизации (0.9/1.0/1.05/1.1)
	Days             []GeneratedDay  `json:"days"`
}

// GeneratedDay - тренировочный день
type GeneratedDay struct {
	DayNum            int                    `json:"day_num"`
	Name              string                 `json:"name"`    // "День 1 — Full Body A"
	Type              string                 `json:"type"`    // push/pull/legs/upper/lower/fullbody
	MuscleGroups      []MuscleGroupExt       `json:"muscle_groups"`
	EstimatedDuration int                    `json:"estimated_duration"` // минуты
	Exercises         []GeneratedExercise    `json:"exercises"`
}

// GeneratedExercise - упражнение в программе
type GeneratedExercise struct {
	OrderNum      int             `json:"order_num"`
	ExerciseID    string          `json:"exercise_id"`
	ExerciseName  string          `json:"exercise_name"`
	MuscleGroup   MuscleGroupExt  `json:"muscle_group"`
	MovementType  MovementType    `json:"movement_type"`
	Sets          int             `json:"sets"`
	Reps          string          `json:"reps"`           // "8-10" или "5"
	Weight        float64         `json:"weight"`         // кг (если рассчитан)
	WeightPercent float64         `json:"weight_percent"` // % от 1ПМ
	TRXLevel      int             `json:"trx_level,omitempty"` // Уровень TRX (1-10)
	Tempo         string          `json:"tempo"`          // "3-1-2-0"
	RestSeconds   int             `json:"rest_seconds"`
	RPE           float64         `json:"rpe"`
	Notes         string          `json:"notes,omitempty"`
	Alternative   *GeneratedExercise `json:"alternative,omitempty"` // Альтернатива
}

// ProgramStats - статистика программы
type ProgramStats struct {
	TotalWorkouts   int     `json:"total_workouts"`
	TotalSets       int     `json:"total_sets"`
	TotalVolume     float64 `json:"total_volume"` // тоннаж
	AvgWorkoutDur   int     `json:"avg_workout_duration"` // минут
	SetsPerMuscle   map[MuscleGroupExt]int `json:"sets_per_muscle"`
	MovementBalance *MovementBalance        `json:"movement_balance,omitempty"` // Баланс паттернов
}

// Substitution - замена упражнения
type Substitution struct {
	OriginalID   string `json:"original_id"`
	OriginalName string `json:"original_name"`
	ReplacedID   string `json:"replaced_id"`
	ReplacedName string `json:"replaced_name"`
	Reason       string `json:"reason"`
}

package progression

import "math"

// ProgressionModel тип модели прогрессии
type ProgressionModel string

const (
	ProgressionLinear        ProgressionModel = "linear"         // Линейная: +2.5кг/неделя
	ProgressionPercentage    ProgressionModel = "percentage"     // Процентная: +2-5%/неделя
	ProgressionWave          ProgressionModel = "wave"           // Волновая: 70→75→80→72→77→82
	ProgressionDouble        ProgressionModel = "double"         // Double: сначала повторения, потом вес
	ProgressionRPEBased      ProgressionModel = "rpe_based"      // RPE-based: по ощущениям
	ProgressionAMRAP         ProgressionModel = "amrap"          // AMRAP: по результатам последнего подхода
	ProgressionStepLoading   ProgressionModel = "step_loading"   // Ступенчатая: 3 недели рост, 1 deload
)

// DayIntensity тип нагрузки дня
type DayIntensity string

const (
	DayHeavy  DayIntensity = "heavy"  // Тяжёлый день
	DayMedium DayIntensity = "medium" // Средний день
	DayLight  DayIntensity = "light"  // Лёгкий день
)

// WavePattern паттерн волновой периодизации между неделями
type WavePattern string

const (
	// WaveNone без волновой периодизации (линейный рост)
	WaveNone WavePattern = "none"
	// WaveThreePlusOne 3+1: 3 недели роста + 1 разгрузка
	// Пример: 70→75→80→(deload 60)→75→80→85→(deload 65)...
	WaveThreePlusOne WavePattern = "three_plus_one"
	// WaveStepped ступенчатая волна: 70→75→80→72→77→82→74→79→84...
	// Каждая волна начинается выше предыдущей
	WaveStepped WavePattern = "stepped"
)

// VolumeLevel уровень объёма
type VolumeLevel string

const (
	VolumeMEV VolumeLevel = "mev" // Minimum Effective Volume - минимальный эффективный
	VolumeMAV VolumeLevel = "mav" // Maximum Adaptive Volume - оптимальный
	VolumeMRV VolumeLevel = "mrv" // Maximum Recoverable Volume - максимально переносимый
)

// VolumeLandmarks ориентиры объёма по мышечным группам (подходов в неделю)
var VolumeLandmarks = map[string]map[VolumeLevel]int{
	"chest": {
		VolumeMEV: 10,
		VolumeMAV: 14,
		VolumeMRV: 20,
	},
	"back": {
		VolumeMEV: 10,
		VolumeMAV: 16,
		VolumeMRV: 22,
	},
	"shoulders": {
		VolumeMEV: 8,
		VolumeMAV: 14,
		VolumeMRV: 20,
	},
	"quads": {
		VolumeMEV: 8,
		VolumeMAV: 14,
		VolumeMRV: 18,
	},
	"hamstrings": {
		VolumeMEV: 6,
		VolumeMAV: 10,
		VolumeMRV: 16,
	},
	"glutes": {
		VolumeMEV: 4,
		VolumeMAV: 10,
		VolumeMRV: 16,
	},
	"biceps": {
		VolumeMEV: 6,
		VolumeMAV: 12,
		VolumeMRV: 18,
	},
	"triceps": {
		VolumeMEV: 6,
		VolumeMAV: 10,
		VolumeMRV: 14,
	},
	"calves": {
		VolumeMEV: 8,
		VolumeMAV: 12,
		VolumeMRV: 16,
	},
	"core": {
		VolumeMEV: 0, // Часто достаточно косвенной нагрузки
		VolumeMAV: 8,
		VolumeMRV: 16,
	},
}

// AdvancedProgressionConfig расширенная конфигурация прогрессии
type AdvancedProgressionConfig struct {
	Model           ProgressionModel
	OnePM           float64
	StartIntensity  float64 // Стартовая интенсивность %
	EndIntensity    float64 // Финальная интенсивность %
	WeeklyIncrement float64 // Еженедельная прибавка (кг или %)
	StepSize        float64 // Шаг округления веса
	DeloadFrequency int     // Каждые N недель deload
	DeloadReduction float64 // На сколько снижать при deload (0.6 = 60%)
}

// DefaultProgressionConfigs конфигурации по умолчанию для разных целей
var DefaultProgressionConfigs = map[string]AdvancedProgressionConfig{
	"beginner_strength": {
		Model:           ProgressionLinear,
		StartIntensity:  70,
		EndIntensity:    85,
		WeeklyIncrement: 2.5, // +2.5кг в неделю
		StepSize:        2.5,
		DeloadFrequency: 4,
		DeloadReduction: 0.6,
	},
	"intermediate_strength": {
		Model:           ProgressionWave,
		StartIntensity:  75,
		EndIntensity:    92,
		WeeklyIncrement: 1.5, // Медленнее
		StepSize:        2.5,
		DeloadFrequency: 4,
		DeloadReduction: 0.65,
	},
	"advanced_strength": {
		Model:           ProgressionRPEBased,
		StartIntensity:  80,
		EndIntensity:    95,
		WeeklyIncrement: 0,   // По RPE
		StepSize:        2.5,
		DeloadFrequency: 3,
		DeloadReduction: 0.65,
	},
	"hypertrophy": {
		Model:           ProgressionDouble,
		StartIntensity:  65,
		EndIntensity:    77,
		WeeklyIncrement: 0, // Сначала повторения
		StepSize:        2.5,
		DeloadFrequency: 5,
		DeloadReduction: 0.5,
	},
	"fat_loss": {
		Model:           ProgressionPercentage,
		StartIntensity:  70,
		EndIntensity:    80,
		WeeklyIncrement: 1.5, // +1.5%
		StepSize:        2.5,
		DeloadFrequency: 0, // Без deload
		DeloadReduction: 0,
	},
}

// AdvancedWeightProgression расширенная прогрессия
type AdvancedWeightProgression struct {
	config AdvancedProgressionConfig
	// Состояние для double progression
	currentReps   int
	targetRepsMin int
	targetRepsMax int
}

// NewAdvancedWeightProgression создаёт прогрессию с конфигурацией
func NewAdvancedWeightProgression(config AdvancedProgressionConfig) *AdvancedWeightProgression {
	return &AdvancedWeightProgression{
		config:        config,
		targetRepsMin: 8,
		targetRepsMax: 12,
	}
}

// GetParams возвращает параметры для недели с учётом модели прогрессии
func (p *AdvancedWeightProgression) GetParams(weekNum, totalWeeks int, dayIntensity DayIntensity) ProgressionParams {
	// Проверяем deload
	isDeload := p.config.DeloadFrequency > 0 && weekNum%p.config.DeloadFrequency == 0

	var intensity float64
	var reps, sets int
	var rpe float64

	switch p.config.Model {
	case ProgressionLinear:
		intensity, reps, sets, rpe = p.calculateLinear(weekNum, isDeload)

	case ProgressionWave:
		intensity, reps, sets, rpe = p.calculateWave(weekNum, totalWeeks, isDeload)

	case ProgressionDouble:
		intensity, reps, sets, rpe = p.calculateDouble(weekNum, isDeload)

	case ProgressionPercentage:
		intensity, reps, sets, rpe = p.calculatePercentage(weekNum, totalWeeks, isDeload)

	case ProgressionStepLoading:
		intensity, reps, sets, rpe = p.calculateStepLoading(weekNum, isDeload)

	default:
		intensity = p.config.StartIntensity
		reps = 8
		sets = 4
		rpe = 7.5
	}

	// Корректируем по типу дня (H/M/L)
	intensity = p.adjustForDayIntensity(intensity, dayIntensity)

	return ProgressionParams{
		Intensity:   intensity,
		Weight:      p.calculateWeight(intensity),
		Reps:        reps,
		Sets:        sets,
		RestSeconds: p.getRestForIntensity(intensity),
		RPE:         rpe,
	}
}

// calculateLinear линейная прогрессия (+X кг/неделю)
func (p *AdvancedWeightProgression) calculateLinear(weekNum int, isDeload bool) (intensity float64, reps, sets int, rpe float64) {
	if isDeload {
		return p.config.StartIntensity * p.config.DeloadReduction, 6, 3, 5.0
	}

	// Прибавляем вес каждую неделю
	addedWeight := float64(weekNum-1) * p.config.WeeklyIncrement
	baseWeight := p.config.OnePM * p.config.StartIntensity / 100

	// Пересчитываем в интенсивность
	newWeight := baseWeight + addedWeight
	intensity = (newWeight / p.config.OnePM) * 100
	intensity = math.Min(intensity, p.config.EndIntensity)

	reps = getRepsForIntensity(intensity)
	sets = 5
	rpe = getRPEForIntensity(intensity)

	return
}

// calculateWave волновая прогрессия (70→75→80→72→77→82...)
func (p *AdvancedWeightProgression) calculateWave(weekNum, totalWeeks int, isDeload bool) (intensity float64, reps, sets int, rpe float64) {
	if isDeload {
		return p.config.StartIntensity * p.config.DeloadReduction, 6, 3, 5.0
	}

	// 3-недельные волны
	wavePosition := (weekNum - 1) % 3 // 0, 1, 2
	waveNumber := (weekNum - 1) / 3   // Номер волны

	// Базовая интенсивность растёт с каждой волной
	waveBaseIncrease := float64(waveNumber) * 2.5

	// Внутри волны: +5% каждую неделю
	switch wavePosition {
	case 0: // Лёгкая неделя волны
		intensity = p.config.StartIntensity + waveBaseIncrease
	case 1: // Средняя
		intensity = p.config.StartIntensity + 5 + waveBaseIncrease
	case 2: // Тяжёлая
		intensity = p.config.StartIntensity + 10 + waveBaseIncrease
	}

	intensity = math.Min(intensity, p.config.EndIntensity)
	reps = getRepsForIntensity(intensity)
	sets = getSetsForStrength(intensity)
	rpe = getRPEForIntensity(intensity)

	return
}

// calculateDouble double progression (сначала повторения, потом вес)
func (p *AdvancedWeightProgression) calculateDouble(weekNum int, isDeload bool) (intensity float64, reps, sets int, rpe float64) {
	if isDeload {
		return p.config.StartIntensity * p.config.DeloadReduction, 8, 3, 5.0
	}

	// Цикл: 4 недели на увеличение повторений, потом прибавка веса
	cyclePosition := (weekNum - 1) % 4 // 0, 1, 2, 3
	cycleNumber := (weekNum - 1) / 4

	// Интенсивность увеличивается только после цикла
	intensity = p.config.StartIntensity + float64(cycleNumber)*2.5
	intensity = math.Min(intensity, p.config.EndIntensity)

	// Повторения увеличиваются внутри цикла: 8→10→12→8 (с новым весом)
	switch cyclePosition {
	case 0:
		reps = 8
	case 1:
		reps = 10
	case 2:
		reps = 12
	case 3:
		reps = 8 // Сброс с новым весом
	}

	sets = 4
	rpe = 7.5 + float64(cyclePosition)*0.5 // RPE растёт: 7.5→8→8.5→7.5

	return
}

// calculatePercentage процентная прогрессия (+X% в неделю)
func (p *AdvancedWeightProgression) calculatePercentage(weekNum, totalWeeks int, isDeload bool) (intensity float64, reps, sets int, rpe float64) {
	if isDeload {
		return p.config.StartIntensity * p.config.DeloadReduction, 8, 3, 5.0
	}

	// Линейное увеличение интенсивности
	progressPercent := float64(weekNum-1) / float64(totalWeeks-1)
	intensity = p.config.StartIntensity + (p.config.EndIntensity-p.config.StartIntensity)*progressPercent

	reps = getRepsForIntensity(intensity)
	sets = 4
	rpe = 7.0 + progressPercent*2.0 // 7→9

	return
}

// calculateStepLoading ступенчатая прогрессия (3 недели рост, 1 deload)
func (p *AdvancedWeightProgression) calculateStepLoading(weekNum int, isDeload bool) (intensity float64, reps, sets int, rpe float64) {
	if isDeload {
		return p.config.StartIntensity * p.config.DeloadReduction, 6, 3, 5.0
	}

	// 4-недельный блок: 3 недели рост + 1 deload
	blockPosition := (weekNum - 1) % 4 // 0, 1, 2, 3
	blockNumber := (weekNum - 1) / 4

	if blockPosition == 3 {
		// Мини-deload каждый 4-й неделе блока
		intensity = p.config.StartIntensity + float64(blockNumber)*5
		reps = 6
		sets = 3
		rpe = 6.0
		return
	}

	// Рост внутри блока
	intensity = p.config.StartIntensity + float64(blockNumber)*5 + float64(blockPosition)*2.5
	intensity = math.Min(intensity, p.config.EndIntensity)

	reps = getRepsForIntensity(intensity)
	sets = getSetsForStrength(intensity)
	rpe = 7.5 + float64(blockPosition)*0.5

	return
}

// adjustForDayIntensity корректирует интенсивность по типу дня
func (p *AdvancedWeightProgression) adjustForDayIntensity(intensity float64, dayType DayIntensity) float64 {
	switch dayType {
	case DayHeavy:
		return intensity * 1.05 // +5%
	case DayMedium:
		return intensity // 100%
	case DayLight:
		return intensity * 0.85 // -15%
	default:
		return intensity
	}
}

// calculateWeight рассчитывает вес с округлением
func (p *AdvancedWeightProgression) calculateWeight(intensity float64) float64 {
	weight := p.config.OnePM * intensity / 100
	return roundToStep(weight, p.config.StepSize)
}

// getRestForIntensity возвращает время отдыха
func (p *AdvancedWeightProgression) getRestForIntensity(intensity float64) int {
	switch {
	case intensity >= 90:
		return 300 // 5 мин
	case intensity >= 85:
		return 240 // 4 мин
	case intensity >= 80:
		return 180 // 3 мин
	case intensity >= 70:
		return 120 // 2 мин
	default:
		return 90 // 1.5 мин
	}
}

// WeeklyDayPattern паттерн дней недели (H/M/L)
type WeeklyDayPattern struct {
	Days []DayIntensity
}

// GetDayPatterns возвращает рекомендуемые паттерны дней
var DayPatterns = map[int]WeeklyDayPattern{
	2: {Days: []DayIntensity{DayHeavy, DayMedium}},
	3: {Days: []DayIntensity{DayHeavy, DayLight, DayMedium}},
	4: {Days: []DayIntensity{DayHeavy, DayMedium, DayLight, DayMedium}},
	5: {Days: []DayIntensity{DayHeavy, DayMedium, DayLight, DayMedium, DayHeavy}},
	6: {Days: []DayIntensity{DayHeavy, DayMedium, DayLight, DayHeavy, DayMedium, DayLight}},
}

// GetDayIntensity возвращает тип нагрузки для дня
func GetDayIntensity(dayNum, daysPerWeek int) DayIntensity {
	pattern, ok := DayPatterns[daysPerWeek]
	if !ok || dayNum < 1 || dayNum > len(pattern.Days) {
		return DayMedium
	}
	return pattern.Days[dayNum-1]
}

// CalculateINOL рассчитывает INOL для подхода
// INOL = reps / (100 - intensity%)
// <0.4 = лёгкая нагрузка, 0.4-1.0 = оптимум, 1.0-2.0 = тяжёлая, >2.0 = чрезмерная
func CalculateINOL(reps int, intensityPercent float64) float64 {
	if intensityPercent >= 100 {
		return float64(reps) // Избегаем деления на 0
	}
	return float64(reps) / (100 - intensityPercent)
}

// CalculateWorkoutINOL рассчитывает суммарный INOL для тренировки
func CalculateWorkoutINOL(exercises []ExerciseINOLData) float64 {
	var total float64
	for _, ex := range exercises {
		for i := 0; i < ex.Sets; i++ {
			total += CalculateINOL(ex.Reps, ex.IntensityPercent)
		}
	}
	return total
}

// ExerciseINOLData данные упражнения для расчёта INOL
type ExerciseINOLData struct {
	Sets             int
	Reps             int
	IntensityPercent float64
}

// INOLRating оценивает INOL
type INOLRating string

const (
	INOLTooLow   INOLRating = "too_low"   // <0.4: слишком мало
	INOLOptimal  INOLRating = "optimal"   // 0.4-1.0: оптимально
	INOLHigh     INOLRating = "high"      // 1.0-2.0: высокая нагрузка
	INOLTooHigh  INOLRating = "too_high"  // >2.0: чрезмерная
)

// RateINOL оценивает значение INOL
func RateINOL(inol float64) INOLRating {
	switch {
	case inol < 0.4:
		return INOLTooLow
	case inol <= 1.0:
		return INOLOptimal
	case inol <= 2.0:
		return INOLHigh
	default:
		return INOLTooHigh
	}
}

// GetVolumeForWeek рассчитывает целевой объём по мышечной группе для недели блока
// weekInBlock - неделя внутри блока (1-4 обычно)
// blockWeeks - всего недель в блоке
func GetVolumeForWeek(muscleGroup string, weekInBlock, blockWeeks int) int {
	landmarks, ok := VolumeLandmarks[muscleGroup]
	if !ok {
		// Дефолтные значения
		landmarks = map[VolumeLevel]int{
			VolumeMEV: 8,
			VolumeMAV: 12,
			VolumeMRV: 16,
		}
	}

	// Прогрессия от MEV к MRV в течение блока
	progress := float64(weekInBlock-1) / float64(blockWeeks-1)

	// Рассчитываем объём
	volumeRange := float64(landmarks[VolumeMRV] - landmarks[VolumeMEV])
	volume := float64(landmarks[VolumeMEV]) + volumeRange*progress

	return int(math.Round(volume))
}

// FatigueAccumulation модель накопления усталости
type FatigueAccumulation struct {
	BaselineFatigue   float64 // Базовый уровень усталости (0-1)
	WeeklyAccumRate   float64 // Скорость накопления в неделю
	DeloadRecovery    float64 // Восстановление при deload
	PerformanceImpact float64 // Влияние на производительность
}

// DefaultFatigueModel дефолтная модель усталости
var DefaultFatigueModel = FatigueAccumulation{
	BaselineFatigue:   0.1,
	WeeklyAccumRate:   0.15,
	DeloadRecovery:    0.4,
	PerformanceImpact: 0.1,
}

// CalculateFatigue рассчитывает накопленную усталость
func (f *FatigueAccumulation) CalculateFatigue(weeksWithoutDeload int) float64 {
	fatigue := f.BaselineFatigue + f.WeeklyAccumRate*float64(weeksWithoutDeload)
	return math.Min(fatigue, 1.0)
}

// ShouldDeload определяет, нужна ли разгрузка
func (f *FatigueAccumulation) ShouldDeload(currentFatigue float64) bool {
	return currentFatigue > 0.7 // Порог 70%
}

// GetPerformanceMultiplier возвращает множитель производительности
func (f *FatigueAccumulation) GetPerformanceMultiplier(fatigue float64) float64 {
	return 1.0 - fatigue*f.PerformanceImpact
}

// ProgressionParams результат расчёта прогрессии
type ProgressionParams struct {
	Intensity   float64 // % от 1ПМ
	Weight      float64 // кг
	Reps        int
	Sets        int
	RestSeconds int
	RPE         float64
}

// === Хелперы для расчётов ===

// getRepsForIntensity возвращает рекомендуемые повторения по интенсивности
func getRepsForIntensity(intensity float64) int {
	switch {
	case intensity >= 95:
		return 1
	case intensity >= 90:
		return 2
	case intensity >= 87:
		return 3
	case intensity >= 85:
		return 4
	case intensity >= 82:
		return 5
	case intensity >= 80:
		return 6
	case intensity >= 77:
		return 7
	case intensity >= 75:
		return 8
	case intensity >= 72:
		return 10
	case intensity >= 70:
		return 12
	default:
		return 15
	}
}

// getRPEForIntensity возвращает RPE по интенсивности
func getRPEForIntensity(intensity float64) float64 {
	switch {
	case intensity >= 95:
		return 9.5
	case intensity >= 90:
		return 9.0
	case intensity >= 85:
		return 8.5
	case intensity >= 80:
		return 8.0
	case intensity >= 75:
		return 7.5
	case intensity >= 70:
		return 7.0
	default:
		return 6.5
	}
}

// getSetsForStrength возвращает кол-во подходов для силовой работы
func getSetsForStrength(intensity float64) int {
	switch {
	case intensity >= 90:
		return 3
	case intensity >= 85:
		return 4
	case intensity >= 80:
		return 5
	default:
		return 4
	}
}

// getRestForIntensity возвращает отдых между подходами
func getRestForIntensity(intensity float64) int {
	switch {
	case intensity >= 90:
		return 300 // 5 минут
	case intensity >= 85:
		return 240 // 4 минуты
	case intensity >= 80:
		return 180 // 3 минуты
	case intensity >= 70:
		return 120 // 2 минуты
	default:
		return 90 // 1.5 минуты
	}
}

// roundToStep округляет вес до шага
func roundToStep(weight, step float64) float64 {
	return math.Round(weight/step) * step
}

// === Автоматический расчёт длины блоков по цели ===

// BlockType тип тренировочного блока
type BlockType string

const (
	BlockAccumulation  BlockType = "accumulation"  // Накопительный (объёмный)
	BlockTransmutation BlockType = "transmutation" // Трансформационный (интенсификация)
	BlockRealization   BlockType = "realization"   // Реализационный (пиковый)
)

// BlockConfig конфигурация блока
type BlockConfig struct {
	Type            BlockType
	NameRu          string
	MinWeeks        int
	MaxWeeks        int
	OptimalWeeks    int
	IntensityStart  float64 // Начальная интенсивность %
	IntensityEnd    float64 // Конечная интенсивность %
	VolumeStart     float64 // Начальный объём (множитель)
	VolumeEnd       float64 // Конечный объём
	DeloadFrequency int     // Частота deload внутри блока (0 = в конце)
}

// GoalBlockConfig конфигурация блоков для цели
type GoalBlockConfig struct {
	Goal              string
	RecommendedWeeks  int           // Рекомендуемая длина программы
	MinWeeks          int           // Минимальная длина
	MaxWeeks          int           // Максимальная длина
	Blocks            []BlockConfig // Конфигурация блоков
	DeloadStrategy    string        // "periodic", "end_of_block", "fatigue_based"
	DeloadFrequency   int           // Стандартная частота deload
	WavePattern       WavePattern   // Паттерн волновой периодизации между неделями
}

// GoalBlockConfigs конфигурации блоков по целям
var GoalBlockConfigs = map[string]GoalBlockConfig{
	"strength": {
		Goal:             "сила",
		RecommendedWeeks: 12,
		MinWeeks:         8,
		MaxWeeks:         16,
		DeloadStrategy:   "end_of_block",
		DeloadFrequency:  4,
		WavePattern:      WaveThreePlusOne, // 3+1: 3 недели роста + 1 разгрузка
		Blocks: []BlockConfig{
			{
				Type:            BlockAccumulation,
				NameRu:          "Подготовительный (GPP)",
				MinWeeks:        2,
				MaxWeeks:        4,
				OptimalWeeks:    3,
				IntensityStart:  65,
				IntensityEnd:    72,
				VolumeStart:     1.0,
				VolumeEnd:       1.1,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockTransmutation,
				NameRu:          "Базовый силовой",
				MinWeeks:        3,
				MaxWeeks:        5,
				OptimalWeeks:    4,
				IntensityStart:  75,
				IntensityEnd:    85,
				VolumeStart:     0.9,
				VolumeEnd:       0.75,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockRealization,
				NameRu:          "Интенсивный",
				MinWeeks:        2,
				MaxWeeks:        4,
				OptimalWeeks:    3,
				IntensityStart:  85,
				IntensityEnd:    92,
				VolumeStart:     0.7,
				VolumeEnd:       0.5,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockRealization,
				NameRu:          "Пиковый",
				MinWeeks:        1,
				MaxWeeks:        3,
				OptimalWeeks:    2,
				IntensityStart:  92,
				IntensityEnd:    100,
				VolumeStart:     0.4,
				VolumeEnd:       0.25,
				DeloadFrequency: 0,
			},
		},
	},
	"hypertrophy": {
		Goal:             "масса",
		RecommendedWeeks: 12,
		MinWeeks:         8,
		MaxWeeks:         20,
		DeloadStrategy:   "periodic",
		DeloadFrequency:  5,
		WavePattern:      WaveStepped, // Ступенчатая волна: 70→75→80→72→77→82...
		Blocks: []BlockConfig{
			{
				Type:            BlockAccumulation,
				NameRu:          "Втягивающий",
				MinWeeks:        1,
				MaxWeeks:        2,
				OptimalWeeks:    2,
				IntensityStart:  60,
				IntensityEnd:    65,
				VolumeStart:     0.7,
				VolumeEnd:       0.85,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockAccumulation,
				NameRu:          "Накопительный 1",
				MinWeeks:        3,
				MaxWeeks:        6,
				OptimalWeeks:    4,
				IntensityStart:  65,
				IntensityEnd:    70,
				VolumeStart:     0.9,
				VolumeEnd:       1.0,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockAccumulation,
				NameRu:          "Накопительный 2",
				MinWeeks:        3,
				MaxWeeks:        6,
				OptimalWeeks:    4,
				IntensityStart:  68,
				IntensityEnd:    75,
				VolumeStart:     1.0,
				VolumeEnd:       1.15,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockTransmutation,
				NameRu:          "Интенсификация",
				MinWeeks:        2,
				MaxWeeks:        4,
				OptimalWeeks:    2,
				IntensityStart:  75,
				IntensityEnd:    82,
				VolumeStart:     0.85,
				VolumeEnd:       0.7,
				DeloadFrequency: 0,
			},
		},
	},
	"weight_loss": {
		Goal:             "похудение",
		RecommendedWeeks: 12,
		MinWeeks:         8,
		MaxWeeks:         16,
		DeloadStrategy:   "periodic",
		DeloadFrequency:  4,
		WavePattern:      WaveStepped, // Ступенчатая волна для постепенной прогрессии
		Blocks: []BlockConfig{
			{
				Type:            BlockAccumulation,
				NameRu:          "Адаптационный",
				MinWeeks:        2,
				MaxWeeks:        3,
				OptimalWeeks:    2,
				IntensityStart:  55,
				IntensityEnd:    62,
				VolumeStart:     0.8,
				VolumeEnd:       1.0,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockAccumulation,
				NameRu:          "Жиросжигающий 1",
				MinWeeks:        3,
				MaxWeeks:        5,
				OptimalWeeks:    4,
				IntensityStart:  60,
				IntensityEnd:    68,
				VolumeStart:     1.0,
				VolumeEnd:       1.15,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockAccumulation,
				NameRu:          "Жиросжигающий 2",
				MinWeeks:        3,
				MaxWeeks:        5,
				OptimalWeeks:    4,
				IntensityStart:  65,
				IntensityEnd:    72,
				VolumeStart:     1.1,
				VolumeEnd:       1.2,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockTransmutation,
				NameRu:          "Поддерживающий",
				MinWeeks:        2,
				MaxWeeks:        3,
				OptimalWeeks:    2,
				IntensityStart:  70,
				IntensityEnd:    78,
				VolumeStart:     0.85,
				VolumeEnd:       0.75,
				DeloadFrequency: 0,
			},
		},
	},
	"competition": {
		Goal:             "соревнования",
		RecommendedWeeks: 12,
		MinWeeks:         10,
		MaxWeeks:         16,
		DeloadStrategy:   "end_of_block",
		DeloadFrequency:  4,
		WavePattern:      WaveThreePlusOne, // 3+1: классическая силовая периодизация
		Blocks: []BlockConfig{
			{
				Type:            BlockAccumulation,
				NameRu:          "Общеподготовительный (GPP)",
				MinWeeks:        2,
				MaxWeeks:        4,
				OptimalWeeks:    3,
				IntensityStart:  60,
				IntensityEnd:    70,
				VolumeStart:     1.0,
				VolumeEnd:       1.1,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockTransmutation,
				NameRu:          "Специально-подготовительный",
				MinWeeks:        3,
				MaxWeeks:        5,
				OptimalWeeks:    4,
				IntensityStart:  75,
				IntensityEnd:    85,
				VolumeStart:     0.9,
				VolumeEnd:       0.75,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockRealization,
				NameRu:          "Предсоревновательный",
				MinWeeks:        2,
				MaxWeeks:        4,
				OptimalWeeks:    3,
				IntensityStart:  85,
				IntensityEnd:    95,
				VolumeStart:     0.65,
				VolumeEnd:       0.45,
				DeloadFrequency: 0,
			},
			{
				Type:            BlockRealization,
				NameRu:          "Пиковый / Тейпер",
				MinWeeks:        1,
				MaxWeeks:        2,
				OptimalWeeks:    2,
				IntensityStart:  90,
				IntensityEnd:    100,
				VolumeStart:     0.35,
				VolumeEnd:       0.2,
				DeloadFrequency: 0,
			},
		},
	},
}

// CalculatedBlock рассчитанный блок
type CalculatedBlock struct {
	Config         BlockConfig
	WeekStart      int
	WeekEnd        int
	Weeks          int
	HasDeload      bool   // Есть ли deload в конце блока
	WeeklyParams   []WeekParams
}

// WeekParams параметры недели
type WeekParams struct {
	WeekNumber       int
	IntensityPercent float64
	VolumeMultiplier float64
	IsDeload         bool
	DayIntensities   []DayIntensity // H/M/L паттерн
}

// CalculateBlockLengths автоматически рассчитывает длины блоков по цели и общей длине
func CalculateBlockLengths(goal string, totalWeeks int, daysPerWeek int) []CalculatedBlock {
	config, ok := GoalBlockConfigs[goal]
	if !ok {
		config = GoalBlockConfigs["strength"] // default
	}

	// Проверяем границы
	if totalWeeks < config.MinWeeks {
		totalWeeks = config.MinWeeks
	}
	if totalWeeks > config.MaxWeeks {
		totalWeeks = config.MaxWeeks
	}

	// Сумма оптимальных недель
	totalOptimal := 0
	for _, block := range config.Blocks {
		totalOptimal += block.OptimalWeeks
	}

	// Коэффициент масштабирования
	scale := float64(totalWeeks) / float64(totalOptimal)

	blocks := make([]CalculatedBlock, 0, len(config.Blocks))
	currentWeek := 1
	weeksAssigned := 0

	for i, blockCfg := range config.Blocks {
		// Расчёт недель для блока
		weeks := int(math.Round(float64(blockCfg.OptimalWeeks) * scale))

		// Применяем ограничения
		if weeks < blockCfg.MinWeeks {
			weeks = blockCfg.MinWeeks
		}
		if weeks > blockCfg.MaxWeeks {
			weeks = blockCfg.MaxWeeks
		}

		// Последний блок получает остаток
		if i == len(config.Blocks)-1 {
			weeks = totalWeeks - weeksAssigned
			if weeks < blockCfg.MinWeeks {
				weeks = blockCfg.MinWeeks
			}
		}

		// Определяем, будет ли deload в конце блока
		hasDeload := config.DeloadStrategy == "end_of_block" && i < len(config.Blocks)-1

		// Создаём рассчитанный блок
		block := CalculatedBlock{
			Config:    blockCfg,
			WeekStart: currentWeek,
			WeekEnd:   currentWeek + weeks - 1,
			Weeks:     weeks,
			HasDeload: hasDeload,
		}

		// Генерируем параметры для каждой недели блока
		block.WeeklyParams = generateWeeklyParams(block, config, daysPerWeek)

		blocks = append(blocks, block)
		currentWeek += weeks
		weeksAssigned += weeks
	}

	return blocks
}

// CalculateBlockLengthsWithWave рассчитывает блоки с кастомным волновым паттерном
// Позволяет переопределить волновой паттерн, заданный по умолчанию для цели
func CalculateBlockLengthsWithWave(goal string, totalWeeks int, daysPerWeek int, wavePattern WavePattern) []CalculatedBlock {
	config, ok := GoalBlockConfigs[goal]
	if !ok {
		config = GoalBlockConfigs["strength"] // default
	}

	// Переопределяем волновой паттерн если указан
	if wavePattern != "" && wavePattern != WaveNone {
		config.WavePattern = wavePattern
	}

	// Проверяем границы
	if totalWeeks < config.MinWeeks {
		totalWeeks = config.MinWeeks
	}
	if totalWeeks > config.MaxWeeks {
		totalWeeks = config.MaxWeeks
	}

	// Сумма оптимальных недель
	totalOptimal := 0
	for _, block := range config.Blocks {
		totalOptimal += block.OptimalWeeks
	}

	// Коэффициент масштабирования
	scale := float64(totalWeeks) / float64(totalOptimal)

	blocks := make([]CalculatedBlock, 0, len(config.Blocks))
	currentWeek := 1
	weeksAssigned := 0

	for i, blockCfg := range config.Blocks {
		// Расчёт недель для блока
		weeks := int(math.Round(float64(blockCfg.OptimalWeeks) * scale))

		// Применяем ограничения
		if weeks < blockCfg.MinWeeks {
			weeks = blockCfg.MinWeeks
		}
		if weeks > blockCfg.MaxWeeks {
			weeks = blockCfg.MaxWeeks
		}

		// Последний блок получает остаток
		if i == len(config.Blocks)-1 {
			weeks = totalWeeks - weeksAssigned
			if weeks < blockCfg.MinWeeks {
				weeks = blockCfg.MinWeeks
			}
		}

		// Определяем, будет ли deload в конце блока
		hasDeload := config.DeloadStrategy == "end_of_block" && i < len(config.Blocks)-1

		// Создаём рассчитанный блок
		block := CalculatedBlock{
			Config:    blockCfg,
			WeekStart: currentWeek,
			WeekEnd:   currentWeek + weeks - 1,
			Weeks:     weeks,
			HasDeload: hasDeload,
		}

		// Генерируем параметры для каждой недели блока с кастомным паттерном
		block.WeeklyParams = generateWeeklyParams(block, config, daysPerWeek)

		blocks = append(blocks, block)
		currentWeek += weeks
		weeksAssigned += weeks
	}

	return blocks
}

// generateWeeklyParams генерирует параметры для каждой недели блока
// Теперь поддерживает волновую периодизацию (3+1 и stepped)
func generateWeeklyParams(block CalculatedBlock, goalConfig GoalBlockConfig, daysPerWeek int) []WeekParams {
	params := make([]WeekParams, 0, block.Weeks)

	// Рассчитываем интенсивности для каждой недели с учётом волнового паттерна
	intensities := calculateWaveIntensities(block, goalConfig.WavePattern)

	for i := 0; i < block.Weeks; i++ {
		weekNum := block.WeekStart + i

		// Получаем интенсивность из волнового расчёта
		intensity := intensities[i].intensity
		isWaveDeload := intensities[i].isDeload

		// Линейная интерполяция объёма (объём всегда растёт линейно внутри блока)
		progress := float64(i) / float64(block.Weeks-1)
		if block.Weeks == 1 {
			progress = 0.5
		}
		volume := block.Config.VolumeStart + (block.Config.VolumeEnd-block.Config.VolumeStart)*progress

		// Определяем deload: либо волновой, либо стратегический
		isDeload := isWaveDeload
		if !isDeload && goalConfig.DeloadStrategy == "periodic" {
			isDeload = goalConfig.DeloadFrequency > 0 && weekNum%goalConfig.DeloadFrequency == 0
		} else if !isDeload && goalConfig.DeloadStrategy == "end_of_block" {
			isDeload = block.HasDeload && i == block.Weeks-1
		}

		// Корректируем для стратегического deload (волновой уже учтён в intensities)
		if isDeload && !isWaveDeload {
			intensity *= 0.65
			volume *= 0.5
		} else if isWaveDeload {
			// Для волнового deload снижаем только объём
			volume *= 0.6
		}

		// Получаем H/M/L паттерн
		dayPattern := getDayPattern(daysPerWeek)

		params = append(params, WeekParams{
			WeekNumber:       weekNum,
			IntensityPercent: intensity,
			VolumeMultiplier: volume,
			IsDeload:         isDeload,
			DayIntensities:   dayPattern,
		})
	}

	return params
}

// waveWeekResult результат расчёта недели волнового паттерна
type waveWeekResult struct {
	intensity float64
	isDeload  bool
}

// calculateWaveIntensities рассчитывает интенсивности для волнового паттерна
func calculateWaveIntensities(block CalculatedBlock, pattern WavePattern) []waveWeekResult {
	results := make([]waveWeekResult, block.Weeks)

	switch pattern {
	case WaveThreePlusOne:
		results = calculateThreePlusOneWave(block)
	case WaveStepped:
		results = calculateSteppedWave(block)
	default: // WaveNone или пустой
		results = calculateLinearProgression(block)
	}

	return results
}

// calculateThreePlusOneWave рассчитывает 3+1 волну
// Паттерн: 3 недели роста + 1 разгрузочная неделя
// Пример для блока 70-85%: 70→75→80→(deload ~65)→75→80→85→(deload ~70)
func calculateThreePlusOneWave(block CalculatedBlock) []waveWeekResult {
	results := make([]waveWeekResult, block.Weeks)

	// Параметры волны
	waveLength := 4 // 3+1
	stepPerWeek := 5.0 // +5% каждую неделю роста
	baseIncreasePerWave := 5.0 // База поднимается на 5% после каждой волны

	// Расчёт для ограничения интенсивности
	intensityRange := block.Config.IntensityEnd - block.Config.IntensityStart

	// Корректируем шаги если блок короткий
	if intensityRange > 0 && block.Weeks > 1 {
		// Подстраиваем шаги под диапазон блока
		effectiveGrowthWeeks := block.Weeks - (block.Weeks / waveLength) // вычитаем deload недели
		if effectiveGrowthWeeks > 0 {
			stepPerWeek = intensityRange / float64(effectiveGrowthWeeks)
			baseIncreasePerWave = stepPerWeek * 1.5 // База растёт медленнее
		}
	}

	for i := 0; i < block.Weeks; i++ {
		waveNum := i / waveLength // Номер текущей волны (0, 1, 2...)
		positionInWave := i % waveLength // Позиция внутри волны (0, 1, 2, 3)

		// Базовая интенсивность для этой волны
		waveBase := block.Config.IntensityStart + float64(waveNum)*baseIncreasePerWave

		var intensity float64
		var isDeload bool

		if positionInWave == 3 {
			// 4-я неделя = разгрузка
			// Deload на уровне начала следующей волны минус 5%
			intensity = waveBase + baseIncreasePerWave - 5.0
			isDeload = true
		} else {
			// Рост: +stepPerWeek за неделю
			intensity = waveBase + float64(positionInWave)*stepPerWeek
			isDeload = false
		}

		// Ограничиваем диапазоном блока
		intensity = math.Max(intensity, block.Config.IntensityStart)
		intensity = math.Min(intensity, block.Config.IntensityEnd)

		// Deload не выше начальной интенсивности следующей волны
		if isDeload {
			nextWaveBase := block.Config.IntensityStart + float64(waveNum+1)*baseIncreasePerWave
			intensity = math.Min(intensity, nextWaveBase-2.5)
		}

		results[i] = waveWeekResult{
			intensity: intensity,
			isDeload:  isDeload,
		}
	}

	return results
}

// calculateSteppedWave рассчитывает ступенчатую волну
// Паттерн: 70→75→80→72→77→82→74→79→84...
// Каждая волна начинается на 2% выше предыдущей
func calculateSteppedWave(block CalculatedBlock) []waveWeekResult {
	results := make([]waveWeekResult, block.Weeks)

	// Параметры ступенчатой волны
	waveLength := 3 // 3 недели в волне
	stepPerWeek := 5.0 // +5% внутри волны
	waveIncrement := 2.0 // Каждая новая волна начинается на 2% выше

	// Корректируем параметры под диапазон блока
	intensityRange := block.Config.IntensityEnd - block.Config.IntensityStart
	if intensityRange > 0 && block.Weeks > 1 {
		// Сколько полных волн поместится
		numWaves := (block.Weeks + waveLength - 1) / waveLength

		// Рассчитываем шаги для достижения конечной интенсивности
		// Последняя волна должна достичь IntensityEnd
		if numWaves > 0 {
			// Конечная точка последней волны = IntensityStart + (numWaves-1)*waveIncrement + (waveLength-1)*stepPerWeek
			// Должна равняться IntensityEnd
			// Решаем: waveIncrement*numWaves + stepPerWeek*(waveLength-1) ≈ intensityRange
			totalSteps := float64(numWaves-1)*waveIncrement + float64(waveLength-1)*stepPerWeek
			if totalSteps > 0 {
				scale := intensityRange / totalSteps
				stepPerWeek *= scale
				waveIncrement *= scale
			}
		}
	}

	for i := 0; i < block.Weeks; i++ {
		waveNum := i / waveLength // Номер волны
		positionInWave := i % waveLength // Позиция в волне

		// База волны растёт с каждой волной
		waveBase := block.Config.IntensityStart + float64(waveNum)*waveIncrement

		// Интенсивность внутри волны растёт
		intensity := waveBase + float64(positionInWave)*stepPerWeek

		// Ограничиваем диапазоном блока
		intensity = math.Max(intensity, block.Config.IntensityStart)
		intensity = math.Min(intensity, block.Config.IntensityEnd)

		results[i] = waveWeekResult{
			intensity: intensity,
			isDeload:  false, // В stepped волне нет явных deload недель
		}
	}

	return results
}

// calculateLinearProgression рассчитывает линейную прогрессию (fallback)
func calculateLinearProgression(block CalculatedBlock) []waveWeekResult {
	results := make([]waveWeekResult, block.Weeks)

	for i := 0; i < block.Weeks; i++ {
		progress := float64(i) / float64(block.Weeks-1)
		if block.Weeks == 1 {
			progress = 0.5
		}

		intensity := block.Config.IntensityStart + (block.Config.IntensityEnd-block.Config.IntensityStart)*progress

		results[i] = waveWeekResult{
			intensity: intensity,
			isDeload:  false,
		}
	}

	return results
}

// getDayPattern возвращает паттерн H/M/L для количества дней
func getDayPattern(daysPerWeek int) []DayIntensity {
	pattern, ok := DayPatterns[daysPerWeek]
	if !ok {
		// Генерируем дефолтный паттерн
		days := make([]DayIntensity, daysPerWeek)
		for i := range days {
			switch i % 3 {
			case 0:
				days[i] = DayHeavy
			case 1:
				days[i] = DayMedium
			case 2:
				days[i] = DayLight
			}
		}
		return days
	}
	return pattern.Days
}

// GetRecommendedProgramLength возвращает рекомендуемую длину программы по цели
func GetRecommendedProgramLength(goal string) (recommended, min, max int) {
	config, ok := GoalBlockConfigs[goal]
	if !ok {
		return 12, 8, 16 // defaults
	}
	return config.RecommendedWeeks, config.MinWeeks, config.MaxWeeks
}

// GetDeloadStrategy возвращает стратегию deload для цели
func GetDeloadStrategy(goal string) (strategy string, frequency int) {
	config, ok := GoalBlockConfigs[goal]
	if !ok {
		return "periodic", 4
	}
	return config.DeloadStrategy, config.DeloadFrequency
}

// GetWavePattern возвращает паттерн волновой периодизации для цели
func GetWavePattern(goal string) WavePattern {
	config, ok := GoalBlockConfigs[goal]
	if !ok {
		return WaveNone
	}
	if config.WavePattern == "" {
		return WaveNone
	}
	return config.WavePattern
}

// WavePatternDescription возвращает описание волнового паттерна
func WavePatternDescription(pattern WavePattern) string {
	switch pattern {
	case WaveThreePlusOne:
		return "3+1: три недели роста + одна разгрузочная"
	case WaveStepped:
		return "Ступенчатая: постепенный рост с откатами (70→75→80→72→77→82...)"
	case WaveNone:
		return "Линейная: постоянный рост без волн"
	default:
		return "Неизвестный паттерн"
	}
}

// === Расширенные модели для микроциклов ===

// MicrocycleType тип микроцикла
type MicrocycleType string

const (
	MicroLoading    MicrocycleType = "loading"    // Нагрузочный
	MicroIntensify  MicrocycleType = "intensify"  // Интенсификационный
	MicroDeload     MicrocycleType = "deload"     // Разгрузочный
	MicroTaper      MicrocycleType = "taper"      // Тейпер (перед соревнованиями)
	MicroCompetition MicrocycleType = "competition" // Соревновательный
)

// MicrocycleConfig конфигурация микроцикла
type MicrocycleConfig struct {
	Type             MicrocycleType
	NameRu           string
	VolumeMultiplier float64
	IntensityMod     float64
	RPETarget        float64
}

// MicrocycleConfigs дефолтные конфигурации микроциклов
var MicrocycleConfigs = map[MicrocycleType]MicrocycleConfig{
	MicroLoading: {
		Type:             MicroLoading,
		NameRu:           "Нагрузочная",
		VolumeMultiplier: 1.0,
		IntensityMod:     1.0,
		RPETarget:        7.5,
	},
	MicroIntensify: {
		Type:             MicroIntensify,
		NameRu:           "Интенсификационная",
		VolumeMultiplier: 0.85,
		IntensityMod:     1.05,
		RPETarget:        8.5,
	},
	MicroDeload: {
		Type:             MicroDeload,
		NameRu:           "Разгрузочная",
		VolumeMultiplier: 0.5,
		IntensityMod:     0.65,
		RPETarget:        5.0,
	},
	MicroTaper: {
		Type:             MicroTaper,
		NameRu:           "Тейпер",
		VolumeMultiplier: 0.4,
		IntensityMod:     0.9,
		RPETarget:        7.0,
	},
	MicroCompetition: {
		Type:             MicroCompetition,
		NameRu:           "Соревновательная",
		VolumeMultiplier: 0.3,
		IntensityMod:     1.0,
		RPETarget:        10.0,
	},
}

// DetermineMicrocycleType определяет тип микроцикла
func DetermineMicrocycleType(weekInBlock, blockWeeks int, blockType BlockType, isLastBlock bool) MicrocycleType {
	// Последняя неделя последнего блока для соревнований
	if isLastBlock && blockType == BlockRealization && weekInBlock == blockWeeks {
		return MicroCompetition
	}

	// Предпоследняя неделя последнего блока - тейпер
	if isLastBlock && blockType == BlockRealization && weekInBlock == blockWeeks-1 {
		return MicroTaper
	}

	// Последняя неделя блока (не последнего) - часто deload
	if weekInBlock == blockWeeks && !isLastBlock {
		return MicroDeload
	}

	// Блок реализации - интенсификационные недели
	if blockType == BlockRealization {
		return MicroIntensify
	}

	// По умолчанию - нагрузочная
	return MicroLoading
}

// === Утилиты для work capacity и объёма ===

// WorkCapacity ёмкость работы для мышечной группы
type WorkCapacity struct {
	MuscleGroup      string
	CurrentMEV       int     // Текущий MEV (может адаптироваться)
	CurrentMAV       int     // Текущий MAV
	CurrentMRV       int     // Текущий MRV
	AdaptationFactor float64 // Множитель адаптации (0.8 - 1.2)
}

// GetAdaptedVolume возвращает адаптированный объём
func (wc *WorkCapacity) GetAdaptedVolume(level VolumeLevel) int {
	base := VolumeLandmarks[wc.MuscleGroup]
	if base == nil {
		return 12 // default
	}

	adapted := float64(base[level]) * wc.AdaptationFactor
	return int(math.Round(adapted))
}

// NewWorkCapacity создаёт work capacity для мышечной группы
func NewWorkCapacity(muscleGroup string) *WorkCapacity {
	landmarks := VolumeLandmarks[muscleGroup]
	if landmarks == nil {
		return &WorkCapacity{
			MuscleGroup:      muscleGroup,
			CurrentMEV:       8,
			CurrentMAV:       12,
			CurrentMRV:       16,
			AdaptationFactor: 1.0,
		}
	}
	return &WorkCapacity{
		MuscleGroup:      muscleGroup,
		CurrentMEV:       landmarks[VolumeMEV],
		CurrentMAV:       landmarks[VolumeMAV],
		CurrentMRV:       landmarks[VolumeMRV],
		AdaptationFactor: 1.0,
	}
}

// IncreaseCapacity увеличивает адаптационный фактор после успешного блока
func (wc *WorkCapacity) IncreaseCapacity() {
	wc.AdaptationFactor = math.Min(wc.AdaptationFactor+0.05, 1.3) // макс +30%
}

// DecreaseCapacity уменьшает после неудачного блока или высокой усталости
func (wc *WorkCapacity) DecreaseCapacity() {
	wc.AdaptationFactor = math.Max(wc.AdaptationFactor-0.05, 0.7) // мин -30%
}

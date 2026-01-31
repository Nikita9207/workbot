package progression

import "math"

// WeightProgression - прогрессия для штанги/гантелей
type WeightProgression struct {
	OnePM             float64              // 1ПМ клиента
	Model             ProgressionModel     // Модель прогрессии
	Config            *AdvancedProgressionConfig
	advancedProg      *AdvancedWeightProgression
}

// NewWeightProgression создаёт новый калькулятор прогрессии (базовая версия)
func NewWeightProgression(onePM float64) *WeightProgression {
	return &WeightProgression{
		OnePM: onePM,
		Model: ProgressionLinear,
	}
}

// NewAdvancedProgression создаёт расширенный калькулятор прогрессии
func NewAdvancedProgression(onePM float64, model ProgressionModel, goal string) *WeightProgression {
	config := DefaultProgressionConfigs[goal]
	if config.Model == "" {
		config = DefaultProgressionConfigs["beginner_strength"]
	}
	config.Model = model
	config.OnePM = onePM

	return &WeightProgression{
		OnePM:        onePM,
		Model:        model,
		Config:       &config,
		advancedProg: NewAdvancedWeightProgression(config),
	}
}

// GetAdvancedParams возвращает параметры с учётом расширенной модели прогрессии
func (wp *WeightProgression) GetAdvancedParams(weekNum, totalWeeks int, dayIntensity DayIntensity) ProgressionParams {
	if wp.advancedProg != nil {
		return wp.advancedProg.GetParams(weekNum, totalWeeks, dayIntensity)
	}
	// Fallback к базовой версии
	return wp.GetStrengthParams(weekNum, "transmutation")
}

// GetParamsWithDayType возвращает параметры с учётом типа дня (H/M/L)
func (wp *WeightProgression) GetParamsWithDayType(weekNum int, phase string, dayIntensity DayIntensity) ProgressionParams {
	// Получаем базовые параметры
	params := wp.GetStrengthParams(weekNum, phase)

	// Корректируем по типу дня
	switch dayIntensity {
	case DayHeavy:
		params.Intensity *= 1.05
		params.RPE += 0.5
		params.Reps = maxInt(params.Reps-1, 1)
	case DayLight:
		params.Intensity *= 0.85
		params.RPE -= 1.5
		params.Sets = maxInt(params.Sets-1, 2)
		params.Reps += 2
	// DayMedium - без изменений
	}

	params.Weight = wp.CalculateWeight(params.Intensity)
	return params
}

// CalculateWeight рассчитывает рабочий вес по проценту от 1ПМ
func (wp *WeightProgression) CalculateWeight(percent float64) float64 {
	return roundToStep(wp.OnePM*percent/100, 2.5)
}

// GetStrengthParams возвращает параметры для силовой работы
// Интенсивность 80-95%, 1-5 повторений
func (wp *WeightProgression) GetStrengthParams(weekNum int, phase string) ProgressionParams {
	baseIntensity := 80.0

	// Прогрессия по неделям
	weekIntensityBonus := float64(weekNum-1) * 2.5
	intensity := math.Min(baseIntensity+weekIntensityBonus, 95.0)

	// Фазовая коррекция
	switch phase {
	case "accumulation":
		intensity = math.Min(intensity, 82.0)
	case "transmutation":
		intensity = math.Min(intensity, 88.0)
	case "realization":
		intensity = math.Min(intensity+5, 100.0)
	case "deload":
		intensity = intensity * 0.8
	}

	reps := getRepsForIntensity(intensity)

	return ProgressionParams{
		Intensity:   intensity,
		Weight:      wp.CalculateWeight(intensity),
		Reps:        reps,
		Sets:        getSetsForStrength(intensity),
		RestSeconds: getRestForIntensity(intensity),
		RPE:         getRPEForIntensity(intensity),
	}
}

// GetHypertrophyParams возвращает параметры для гипертрофии
// Интенсивность 65-80%, 6-12 повторений
func (wp *WeightProgression) GetHypertrophyParams(weekNum int, phase string) ProgressionParams {
	baseIntensity := 65.0

	// Прогрессия по неделям (double progression)
	weekIntensityBonus := float64(weekNum-1) * 2.0
	intensity := math.Min(baseIntensity+weekIntensityBonus, 80.0)

	// Фазовая коррекция
	switch phase {
	case "accumulation":
		// Больше объёма, меньше интенсивности
		intensity = math.Min(intensity, 72.0)
	case "intensification":
		// Выше интенсивность
		intensity = math.Min(intensity+5, 82.0)
	case "deload":
		intensity = intensity * 0.85
	}

	return ProgressionParams{
		Intensity:   intensity,
		Weight:      wp.CalculateWeight(intensity),
		Reps:        getRepsForHypertrophy(intensity),
		Sets:        getSetsForHypertrophy(phase),
		RestSeconds: 90,
		RPE:         7.5,
	}
}

// GetFatLossParams возвращает параметры для жиросжигания
// Сохраняем интенсивность, снижаем объём
func (wp *WeightProgression) GetFatLossParams(weekNum int) ProgressionParams {
	// Интенсивность сохраняется высокой для сохранения мышц
	intensity := 75.0 + float64(weekNum-1)*1.5
	intensity = math.Min(intensity, 85.0)

	return ProgressionParams{
		Intensity:   intensity,
		Weight:      wp.CalculateWeight(intensity),
		Reps:        8,  // Фиксированные повторения
		Sets:        3,  // Сниженный объём
		RestSeconds: 60, // Короткий отдых (плотность)
		RPE:         8.0,
	}
}

// ProgressionParams уже определён в progression_models.go

// === Вспомогательные функции для гипертрофии ===

// getRepsForHypertrophy возвращает повторения для гипертрофии
func getRepsForHypertrophy(intensity float64) int {
	switch {
	case intensity >= 80:
		return 6
	case intensity >= 75:
		return 8
	case intensity >= 70:
		return 10
	default:
		return 12
	}
}

// getSetsForHypertrophy возвращает подходы для гипертрофии
func getSetsForHypertrophy(phase string) int {
	switch phase {
	case "accumulation":
		return 4
	case "intensification":
		return 3
	case "deload":
		return 2
	default:
		return 3
	}
}

// === Формулы расчёта 1ПМ ===

// Estimate1PM рассчитывает 1ПМ по весу и повторениям
func Estimate1PM(weight float64, reps int, method string) float64 {
	if reps == 1 {
		return weight
	}

	r := float64(reps)

	switch method {
	case "brzycki":
		// Brzycki formula: 1RM = w × (36 / (37 - r))
		return weight * (36 / (37 - r))
	case "epley":
		// Epley formula: 1RM = w × (1 + r/30)
		return weight * (1 + r/30)
	case "lander":
		// Lander formula: 1RM = 100 × w / (101.3 - 2.67123 × r)
		return 100 * weight / (101.3 - 2.67123*r)
	case "lombardi":
		// Lombardi formula: 1RM = w × r^0.10
		return weight * math.Pow(r, 0.10)
	default:
		// Среднее из нескольких формул
		brzycki := weight * (36 / (37 - r))
		epley := weight * (1 + r/30)
		return (brzycki + epley) / 2
	}
}

// WeightIncrements - рекомендуемые шаги прибавки веса
var WeightIncrements = map[string]float64{
	"barbell_compound":  2.5, // Штанга, базовые
	"barbell_isolation": 1.25, // Штанга, изоляция
	"dumbbell_compound": 2.0,  // Гантели, базовые
	"dumbbell_isolation": 1.0,  // Гантели, изоляция
}

// === INOL-based валидация и оптимизация ===

// ValidateWithINOL проверяет параметры по INOL и корректирует при необходимости
func (wp *WeightProgression) ValidateWithINOL(params ProgressionParams, maxINOL float64) ProgressionParams {
	inol := CalculateINOL(params.Reps, params.Intensity) * float64(params.Sets)

	if inol > maxINOL {
		// Снижаем нагрузку - уменьшаем подходы
		for inol > maxINOL && params.Sets > 1 {
			params.Sets--
			inol = CalculateINOL(params.Reps, params.Intensity) * float64(params.Sets)
		}
		// Если всё ещё много - снижаем интенсивность
		for inol > maxINOL && params.Intensity > 60 {
			params.Intensity -= 2.5
			params.Weight = wp.CalculateWeight(params.Intensity)
			inol = CalculateINOL(params.Reps, params.Intensity) * float64(params.Sets)
		}
	}

	return params
}

// GetOptimalParams возвращает оптимальные параметры с учётом INOL, RPE и цели
func (wp *WeightProgression) GetOptimalParams(weekNum, totalWeeks int, goal string, dayIntensity DayIntensity) ProgressionParams {
	var params ProgressionParams
	var maxINOL float64

	switch goal {
	case "strength", "сила":
		params = wp.GetStrengthParams(weekNum, getPhaseForGoal(weekNum, totalWeeks, "strength"))
		maxINOL = 2.0 // Высокая нагрузка допустима
	case "hypertrophy", "масса":
		params = wp.GetHypertrophyParams(weekNum, getPhaseForGoal(weekNum, totalWeeks, "hypertrophy"))
		maxINOL = 1.5 // Умеренная нагрузка
	case "fat_loss", "похудение":
		params = wp.GetFatLossParams(weekNum)
		maxINOL = 1.0 // Низкая нагрузка (сохраняем мышцы, не перегружаем)
	default:
		params = wp.GetStrengthParams(weekNum, "transmutation")
		maxINOL = 1.5
	}

	// Корректировка по типу дня
	switch dayIntensity {
	case DayHeavy:
		params.Intensity *= 1.05
		params.Reps = maxInt(params.Reps-1, 1)
		maxINOL *= 1.2 // Допускаем больше на тяжёлый день
	case DayLight:
		params.Intensity *= 0.85
		params.Sets = maxInt(params.Sets-1, 2)
		params.Reps += 2
		maxINOL *= 0.7
	}

	params.Weight = wp.CalculateWeight(params.Intensity)

	// Валидация по INOL
	return wp.ValidateWithINOL(params, maxINOL)
}

// getPhaseForGoal определяет фазу по неделе и цели
func getPhaseForGoal(weekNum, totalWeeks int, goal string) string {
	progress := float64(weekNum) / float64(totalWeeks)

	switch goal {
	case "strength":
		if progress <= 0.33 {
			return "accumulation"
		} else if progress <= 0.66 {
			return "transmutation"
		}
		return "realization"
	case "hypertrophy":
		if progress <= 0.6 {
			return "accumulation"
		} else if progress < 1.0 {
			return "intensification"
		}
		return "deload"
	default:
		return "transmutation"
	}
}

// === Расширенные методы для блочной периодизации ===

// GetBlockParams возвращает параметры для конкретного блока
func (wp *WeightProgression) GetBlockParams(block CalculatedBlock, weekInBlock int, dayIntensity DayIntensity) ProgressionParams {
	if weekInBlock < 1 || weekInBlock > block.Weeks {
		weekInBlock = 1
	}

	// Получаем параметры недели из блока
	weekParams := block.WeeklyParams[weekInBlock-1]

	var reps int
	var sets int
	var rpe float64
	var rest int

	intensity := weekParams.IntensityPercent

	// Корректируем по типу дня
	switch dayIntensity {
	case DayHeavy:
		intensity *= 1.05
		reps = getRepsForIntensity(intensity)
		sets = getSetsForStrength(intensity)
		rpe = getRPEForIntensity(intensity) + 0.5
		rest = getRestForIntensity(intensity)
	case DayLight:
		intensity *= 0.85
		reps = getRepsForIntensity(intensity) + 2
		sets = maxInt(getSetsForStrength(intensity)-1, 2)
		rpe = getRPEForIntensity(intensity) - 1.5
		rest = int(float64(getRestForIntensity(intensity)) * 0.75)
	default: // DayMedium
		reps = getRepsForIntensity(intensity)
		sets = getSetsForStrength(intensity)
		rpe = getRPEForIntensity(intensity)
		rest = getRestForIntensity(intensity)
	}

	// Deload корректировка
	if weekParams.IsDeload {
		intensity *= 0.65
		sets = maxInt(sets-2, 2)
		reps = 6
		rpe = 5.0
		rest = 120
	}

	return ProgressionParams{
		Intensity:   intensity,
		Weight:      wp.CalculateWeight(intensity),
		Reps:        reps,
		Sets:        sets,
		RestSeconds: rest,
		RPE:         rpe,
	}
}

// === Утилиты ===

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

package progression

import "math"

// WeightProgression - прогрессия для штанги/гантелей
type WeightProgression struct {
	OnePM float64 // 1ПМ клиента
}

// NewWeightProgression создаёт новый калькулятор прогрессии
func NewWeightProgression(onePM float64) *WeightProgression {
	return &WeightProgression{OnePM: onePM}
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

// ProgressionParams - параметры прогрессии
type ProgressionParams struct {
	Intensity   float64 // % от 1ПМ
	Weight      float64 // Вес в кг
	Reps        int     // Повторения
	Sets        int     // Подходы
	RestSeconds int     // Отдых (сек)
	RPE         float64 // Целевой RPE
}

// === Вспомогательные функции ===

// roundToStep округляет вес до ближайшего шага
func roundToStep(weight, step float64) float64 {
	return math.Round(weight/step) * step
}

// getRepsForIntensity возвращает повторения для силовой работы
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
	case intensity >= 80:
		return 5
	default:
		return 6
	}
}

// getSetsForStrength возвращает подходы для силовой работы
func getSetsForStrength(intensity float64) int {
	switch {
	case intensity >= 95:
		return 3
	case intensity >= 90:
		return 4
	case intensity >= 85:
		return 5
	default:
		return 5
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
	default:
		return 120 // 2 минуты
	}
}

// getRPEForIntensity возвращает целевой RPE
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
	default:
		return 7.5
	}
}

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

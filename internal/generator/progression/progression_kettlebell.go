package progression

import "math"

// KettlebellProgression - прогрессия для гирь
// Дискретные веса: 8, 12, 16, 20, 24, 28, 32 кг
type KettlebellProgression struct {
	AvailableWeights []float64 // Доступные веса гирь
	ClientGender     string    // male/female
	Experience       string    // beginner/intermediate/advanced
}

// StandardKBWeights - стандартные веса гирь
var StandardKBWeights = []float64{8, 12, 16, 20, 24, 28, 32}

// NewKettlebellProgression создаёт новый калькулятор прогрессии для гирь
func NewKettlebellProgression(availableWeights []float64, gender, experience string) *KettlebellProgression {
	weights := availableWeights
	if len(weights) == 0 {
		weights = StandardKBWeights
	}

	return &KettlebellProgression{
		AvailableWeights: weights,
		ClientGender:     gender,
		Experience:       experience,
	}
}

// KBExerciseType - тип гиревого упражнения
type KBExerciseType string

const (
	KBTypeBallistic KBExerciseType = "ballistic" // Свинги, рывки, толчки
	KBTypeGrind     KBExerciseType = "grind"     // Жимы, приседания, тяги
	KBTypeComplex   KBExerciseType = "complex"   // Комплексы
)

// WeightRecommendations - рекомендации по стартовому весу
type WeightRecommendations struct {
	BallisticMin float64 // Минимум для баллистики
	BallisticMax float64 // Максимум для баллистики
	GrindMin     float64 // Минимум для грайнда
	GrindMax     float64 // Максимум для грайнда
}

// GetWeightRecommendations возвращает рекомендации по весу
func (kp *KettlebellProgression) GetWeightRecommendations() WeightRecommendations {
	var rec WeightRecommendations

	switch kp.Experience {
	case "beginner":
		if kp.ClientGender == "female" {
			rec = WeightRecommendations{
				BallisticMin: 8, BallisticMax: 12,
				GrindMin: 8, GrindMax: 12,
			}
		} else {
			rec = WeightRecommendations{
				BallisticMin: 12, BallisticMax: 16,
				GrindMin: 12, GrindMax: 16,
			}
		}
	case "intermediate":
		if kp.ClientGender == "female" {
			rec = WeightRecommendations{
				BallisticMin: 12, BallisticMax: 16,
				GrindMin: 12, GrindMax: 16,
			}
		} else {
			rec = WeightRecommendations{
				BallisticMin: 16, BallisticMax: 24,
				GrindMin: 16, GrindMax: 24,
			}
		}
	case "advanced":
		if kp.ClientGender == "female" {
			rec = WeightRecommendations{
				BallisticMin: 16, BallisticMax: 24,
				GrindMin: 16, GrindMax: 20,
			}
		} else {
			rec = WeightRecommendations{
				BallisticMin: 24, BallisticMax: 32,
				GrindMin: 24, GrindMax: 32,
			}
		}
	}

	return rec
}

// GetStartWeight возвращает стартовый вес для упражнения
func (kp *KettlebellProgression) GetStartWeight(exerciseType KBExerciseType) float64 {
	rec := kp.GetWeightRecommendations()

	var targetWeight float64
	switch exerciseType {
	case KBTypeBallistic:
		targetWeight = rec.BallisticMin
	case KBTypeGrind:
		targetWeight = rec.GrindMin
	case KBTypeComplex:
		// Для комплексов берём минимум от грайнда (слабое звено)
		targetWeight = rec.GrindMin
	}

	return kp.findClosestWeight(targetWeight)
}

// GetMaxWeight возвращает максимальный рекомендуемый вес
func (kp *KettlebellProgression) GetMaxWeight(exerciseType KBExerciseType) float64 {
	rec := kp.GetWeightRecommendations()

	var targetWeight float64
	switch exerciseType {
	case KBTypeBallistic:
		targetWeight = rec.BallisticMax
	case KBTypeGrind:
		targetWeight = rec.GrindMax
	case KBTypeComplex:
		targetWeight = rec.GrindMax
	}

	return kp.findClosestWeight(targetWeight)
}

// findClosestWeight находит ближайший доступный вес
func (kp *KettlebellProgression) findClosestWeight(target float64) float64 {
	if len(kp.AvailableWeights) == 0 {
		return target
	}

	closest := kp.AvailableWeights[0]
	minDiff := math.Abs(target - closest)

	for _, w := range kp.AvailableWeights {
		diff := math.Abs(target - w)
		if diff < minDiff {
			minDiff = diff
			closest = w
		}
	}

	return closest
}

// GetNextWeight возвращает следующий вес в прогрессии
func (kp *KettlebellProgression) GetNextWeight(currentWeight float64) (float64, bool) {
	for i, w := range kp.AvailableWeights {
		if w == currentWeight && i < len(kp.AvailableWeights)-1 {
			return kp.AvailableWeights[i+1], true
		}
	}
	return currentWeight, false // Уже максимальный вес
}

// KBParams - параметры гиревого упражнения
type KBParams struct {
	Weight        float64
	Reps          int
	Sets          int
	RestSeconds   int
	Pattern       string // bilateral/unilateral/alternating
	Notes         string
}

// GetBallisticParams возвращает параметры для баллистических упражнений
func (kp *KettlebellProgression) GetBallisticParams(weekNum int, totalWeeks int) KBParams {
	startWeight := kp.GetStartWeight(KBTypeBallistic)
	maxWeight := kp.GetMaxWeight(KBTypeBallistic)

	// Прогрессия веса: каждые 3-4 недели пробуем перейти на следующий вес
	currentWeight := startWeight
	progressionWeek := 3
	for w := weekNum; w > progressionWeek; w -= progressionWeek {
		nextWeight, hasNext := kp.GetNextWeight(currentWeight)
		if hasNext && nextWeight <= maxWeight {
			currentWeight = nextWeight
		}
	}

	// Баллистика: 10-20 повторений, мощность, короткий отдых
	reps := 15
	if currentWeight >= maxWeight*0.9 {
		reps = 10 // Тяжёлый вес — меньше повторений
	}

	return KBParams{
		Weight:      currentWeight,
		Reps:        reps,
		Sets:        4,
		RestSeconds: 60,
		Pattern:     "bilateral",
	}
}

// GetGrindParams возвращает параметры для грайнд упражнений
func (kp *KettlebellProgression) GetGrindParams(weekNum int, totalWeeks int, phase string) KBParams {
	startWeight := kp.GetStartWeight(KBTypeGrind)
	maxWeight := kp.GetMaxWeight(KBTypeGrind)

	// Прогрессия для грайнда медленнее
	currentWeight := startWeight
	progressionWeek := 4
	for w := weekNum; w > progressionWeek; w -= progressionWeek {
		nextWeight, hasNext := kp.GetNextWeight(currentWeight)
		if hasNext && nextWeight <= maxWeight {
			currentWeight = nextWeight
		}
	}

	// Грайнд: 5-10 повторений, контроль, длинный отдых
	reps := 8
	sets := 4
	rest := 120

	switch phase {
	case "accumulation":
		reps = 10
		sets = 3
	case "intensification":
		reps = 6
		sets = 5
		rest = 150
	case "deload":
		// При разгрузке снижаем вес, если возможно
		for i := len(kp.AvailableWeights) - 1; i >= 0; i-- {
			if kp.AvailableWeights[i] < currentWeight {
				currentWeight = kp.AvailableWeights[i]
				break
			}
		}
		reps = 8
		sets = 2
	}

	return KBParams{
		Weight:      currentWeight,
		Reps:        reps,
		Sets:        sets,
		RestSeconds: rest,
		Pattern:     "unilateral",
	}
}

// GetComplexParams возвращает параметры для комплексов
func (kp *KettlebellProgression) GetComplexParams(weekNum int) KBParams {
	// Для комплексов используем более лёгкий вес
	startWeight := kp.GetStartWeight(KBTypeComplex)

	// Прогрессия через повторения, не через вес
	reps := 3 + (weekNum-1)/2 // Начинаем с 3, добавляем по 1 каждые 2 недели
	if reps > 6 {
		reps = 6
	}

	return KBParams{
		Weight:      startWeight,
		Reps:        reps,
		Sets:        4,
		RestSeconds: 90,
		Pattern:     "unilateral",
		Notes:       "Выполнять без отдыха между элементами комплекса",
	}
}

// ProgressionAlternatives - альтернативные способы прогрессии
// когда следующий вес недоступен
type ProgressionAlternatives struct {
	AddReps      int    // Добавить повторения
	AddSets      int    // Добавить подход
	ChangePattern string // Сменить паттерн
	AddPause     bool   // Добавить паузу
	Notes        string
}

// GetProgressionAlternative предлагает альтернативу, если следующий вес недоступен
func (kp *KettlebellProgression) GetProgressionAlternative(currentWeight float64, currentReps, currentSets int) ProgressionAlternatives {
	_, hasNext := kp.GetNextWeight(currentWeight)

	if hasNext {
		return ProgressionAlternatives{} // Прогрессия через вес доступна
	}

	// Нет следующего веса — альтернативы
	alt := ProgressionAlternatives{}

	if currentReps < 12 {
		alt.AddReps = 2
		alt.Notes = "Добавьте 2 повторения к каждому подходу"
	} else if currentSets < 5 {
		alt.AddSets = 1
		alt.Notes = "Добавьте 1 подход"
	} else {
		alt.ChangePattern = "unilateral"
		alt.Notes = "Перейдите на одностороннее выполнение"
	}

	return alt
}

// GetPatternProgression возвращает прогрессию паттерна
func GetPatternProgression(weekNum int) string {
	switch {
	case weekNum <= 3:
		return "bilateral" // Две руки
	case weekNum <= 6:
		return "alternating" // Попеременно
	default:
		return "unilateral" // Одна рука
	}
}

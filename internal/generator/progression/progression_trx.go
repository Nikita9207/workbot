package progression

import "math"

// TRXProgression - прогрессия для TRX
// Уровни сложности 1-10 вместо градусов (понятнее клиенту)
type TRXProgression struct {
	ClientWeight float64 // Вес клиента (кг)
	Experience   string  // beginner/intermediate/advanced
}

// NewTRXProgression создаёт новый калькулятор TRX прогрессии
func NewTRXProgression(clientWeight float64, experience string) *TRXProgression {
	return &TRXProgression{
		ClientWeight: clientWeight,
		Experience:   experience,
	}
}

// TRXLevel представляет уровень сложности TRX
type TRXLevel struct {
	Level          int     // 1-10
	AngleApprox    int     // Примерный угол (градусы от вертикали)
	BodyWeightPct  float64 // % веса тела как нагрузка
	LoadKg         float64 // Нагрузка в кг (для сравнения)
	Description    string  // Описание позиции
}

// LevelDescriptions - описания уровней TRX
var LevelDescriptions = map[int]TRXLevel{
	1:  {Level: 1, AngleApprox: 10, BodyWeightPct: 0.15, Description: "Почти вертикально, минимальная нагрузка"},
	2:  {Level: 2, AngleApprox: 20, BodyWeightPct: 0.25, Description: "Небольшой наклон назад"},
	3:  {Level: 3, AngleApprox: 30, BodyWeightPct: 0.35, Description: "Умеренный наклон"},
	4:  {Level: 4, AngleApprox: 40, BodyWeightPct: 0.42, Description: "Заметный наклон"},
	5:  {Level: 5, AngleApprox: 45, BodyWeightPct: 0.50, Description: "Наклон 45°, половина веса тела"},
	6:  {Level: 6, AngleApprox: 50, BodyWeightPct: 0.58, Description: "Сильный наклон"},
	7:  {Level: 7, AngleApprox: 55, BodyWeightPct: 0.65, Description: "Значительный наклон"},
	8:  {Level: 8, AngleApprox: 60, BodyWeightPct: 0.72, Description: "Сильный наклон, высокая нагрузка"},
	9:  {Level: 9, AngleApprox: 70, BodyWeightPct: 0.80, Description: "Почти горизонтально"},
	10: {Level: 10, AngleApprox: 80, BodyWeightPct: 0.85, Description: "Максимальный наклон"},
}

// GetStartLevel возвращает стартовый уровень для упражнения
func (tp *TRXProgression) GetStartLevel(exerciseMinLevel, exerciseMaxLevel int) int {
	var baseLevel int

	switch tp.Experience {
	case "beginner":
		baseLevel = exerciseMinLevel
	case "intermediate":
		baseLevel = exerciseMinLevel + (exerciseMaxLevel-exerciseMinLevel)/3
	case "advanced":
		baseLevel = exerciseMinLevel + (exerciseMaxLevel-exerciseMinLevel)*2/3
	default:
		baseLevel = exerciseMinLevel
	}

	// Не выходим за границы упражнения
	if baseLevel < exerciseMinLevel {
		baseLevel = exerciseMinLevel
	}
	if baseLevel > exerciseMaxLevel {
		baseLevel = exerciseMaxLevel
	}

	return baseLevel
}

// GetProgressedLevel возвращает уровень с учётом прогрессии по неделям
func (tp *TRXProgression) GetProgressedLevel(startLevel, weekNum, totalWeeks, exerciseMaxLevel int) int {
	// Прогрессия: +1 уровень каждые 2-3 недели
	progressionRate := 0.4 // Примерно +1 уровень за 2.5 недели
	levelBonus := int(float64(weekNum-1) * progressionRate)

	newLevel := startLevel + levelBonus

	// Не превышаем максимум упражнения
	if newLevel > exerciseMaxLevel {
		newLevel = exerciseMaxLevel
	}

	return newLevel
}

// GetLoadKg возвращает эквивалентную нагрузку в кг для уровня
func (tp *TRXProgression) GetLoadKg(level int) float64 {
	if level < 1 {
		level = 1
	}
	if level > 10 {
		level = 10
	}

	levelInfo := LevelDescriptions[level]
	return math.Round(tp.ClientWeight * levelInfo.BodyWeightPct)
}

// TRXParams - параметры TRX упражнения
type TRXParams struct {
	Level       int    // Уровень 1-10
	LoadKg      float64 // Эквивалент в кг
	Reps        int
	Sets        int
	Tempo       string // "3-1-2-0" формат
	RestSeconds int
}

// GetHypertrophyParams возвращает параметры для гипертрофии на TRX
func (tp *TRXProgression) GetHypertrophyParams(exerciseMinLevel, exerciseMaxLevel, weekNum, totalWeeks int) TRXParams {
	startLevel := tp.GetStartLevel(exerciseMinLevel, exerciseMaxLevel)
	currentLevel := tp.GetProgressedLevel(startLevel, weekNum, totalWeeks, exerciseMaxLevel)

	// Tempo прогрессия
	tempo := getTRXTempo(weekNum)

	// Reps снижаются по мере роста уровня
	reps := getRepsForTRXLevel(currentLevel, exerciseMaxLevel)

	return TRXParams{
		Level:       currentLevel,
		LoadKg:      tp.GetLoadKg(currentLevel),
		Reps:        reps,
		Sets:        3,
		Tempo:       tempo,
		RestSeconds: 90,
	}
}

// GetStrengthParams возвращает параметры для силовой работы на TRX
func (tp *TRXProgression) GetStrengthParams(exerciseMinLevel, exerciseMaxLevel, weekNum, totalWeeks int) TRXParams {
	startLevel := tp.GetStartLevel(exerciseMinLevel, exerciseMaxLevel)
	// Для силы стартуем выше
	startLevel = min(startLevel+1, exerciseMaxLevel)
	currentLevel := tp.GetProgressedLevel(startLevel, weekNum, totalWeeks, exerciseMaxLevel)

	return TRXParams{
		Level:       currentLevel,
		LoadKg:      tp.GetLoadKg(currentLevel),
		Reps:        6, // Меньше повторений
		Sets:        4, // Больше подходов
		Tempo:       "4-2-2-0", // Медленный темп
		RestSeconds: 120,
	}
}

// GetEnduranceParams возвращает параметры для выносливости на TRX
func (tp *TRXProgression) GetEnduranceParams(exerciseMinLevel, exerciseMaxLevel, weekNum int) TRXParams {
	// Для выносливости используем более низкий уровень
	startLevel := tp.GetStartLevel(exerciseMinLevel, exerciseMaxLevel)
	if startLevel > exerciseMinLevel {
		startLevel--
	}

	return TRXParams{
		Level:       startLevel,
		LoadKg:      tp.GetLoadKg(startLevel),
		Reps:        15, // Много повторений
		Sets:        3,
		Tempo:       "2-0-2-0", // Быстрый темп
		RestSeconds: 60,
	}
}

// === Вспомогательные функции ===

// getTRXTempo возвращает темп в зависимости от недели
func getTRXTempo(weekNum int) string {
	switch {
	case weekNum <= 3:
		return "2-0-2-0" // Базовый темп
	case weekNum <= 6:
		return "3-1-2-0" // Увеличиваем эксцентрик
	case weekNum <= 9:
		return "4-2-2-0" // Максимальный TUT
	default:
		return "3-1-2-0"
	}
}

// getRepsForTRXLevel возвращает повторения в зависимости от уровня
func getRepsForTRXLevel(level, maxLevel int) int {
	// Чем выше уровень относительно максимума, тем меньше повторений
	ratio := float64(level) / float64(maxLevel)

	switch {
	case ratio >= 0.9:
		return 8
	case ratio >= 0.7:
		return 10
	case ratio >= 0.5:
		return 12
	default:
		return 15
	}
}

// TRXProgressionVariants - варианты прогрессии TRX
type TRXProgressionVariants struct {
	// Прогрессия позиции ног
	FeetCloser   bool // Ноги ближе к точке крепления = сложнее
	SingleLeg    bool // На одной ноге
	ElevatedFeet bool // Ноги на возвышении

	// Прогрессия паттерна
	Bilateral   bool // Двусторонний
	Unilateral  bool // Односторонний
	Alternating bool // Попеременный
}

// GetNextVariant предлагает следующий вариант прогрессии
func GetNextVariant(current TRXProgressionVariants, currentLevel, maxLevel int) TRXProgressionVariants {
	next := current

	// Если достигли максимального уровня — переходим к вариантам
	if currentLevel >= maxLevel {
		if !current.FeetCloser {
			next.FeetCloser = true
		} else if current.Bilateral && !current.Unilateral {
			next.Bilateral = false
			next.Unilateral = true
		} else if !current.ElevatedFeet {
			next.ElevatedFeet = true
		}
	}

	return next
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

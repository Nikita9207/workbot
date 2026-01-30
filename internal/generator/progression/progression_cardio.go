package progression

import "math"

// CardioProgression - прогрессия для кардио/метаболических упражнений
type CardioProgression struct {
	Goal       string  // fat_loss/hyrox/endurance
	Experience string  // beginner/intermediate/advanced
	MaxHR      int     // Максимальный пульс (220 - возраст)
}

// NewCardioProgression создаёт новый калькулятор кардио прогрессии
func NewCardioProgression(goal, experience string, age int) *CardioProgression {
	maxHR := 220 - age
	if maxHR < 160 {
		maxHR = 160
	}

	return &CardioProgression{
		Goal:       goal,
		Experience: experience,
		MaxHR:      maxHR,
	}
}

// CardioParams - параметры кардио упражнения
type CardioParams struct {
	Mode         string  // distance/time/calories/intervals
	Value        float64 // Значение (метры, секунды, калории)
	Sets         int     // Количество интервалов/раундов
	WorkSeconds  int     // Время работы (для интервалов)
	RestSeconds  int     // Время отдыха (для интервалов)
	TargetHRPct  float64 // Целевой % от максимального пульса
	Pace         string  // easy/moderate/hard/all_out
	Notes        string
}

// DistanceProgression - прогрессия по дистанции
type DistanceProgression struct {
	StartMeters    float64
	WeeklyIncrease float64 // % увеличения в неделю
	MaxMeters      float64
}

// TimeProgression - прогрессия по времени
type TimeProgression struct {
	StartSeconds   int
	WeeklyIncrease int // Секунд в неделю
	MaxSeconds     int
}

// === LISS (Low Intensity Steady State) ===

// GetLISSParams возвращает параметры для LISS кардио
func (cp *CardioProgression) GetLISSParams(weekNum int) CardioParams {
	// Базовая продолжительность по уровню
	var baseDuration int
	switch cp.Experience {
	case "beginner":
		baseDuration = 20 * 60 // 20 минут
	case "intermediate":
		baseDuration = 30 * 60 // 30 минут
	case "advanced":
		baseDuration = 40 * 60 // 40 минут
	}

	// Прогрессия: +5 минут каждые 2 недели
	durationBonus := ((weekNum - 1) / 2) * 5 * 60
	duration := baseDuration + durationBonus

	// Максимум 60 минут
	if duration > 60*60 {
		duration = 60 * 60
	}

	return CardioParams{
		Mode:        "time",
		Value:       float64(duration),
		Sets:        1,
		TargetHRPct: 0.65, // 65% от max HR
		Pace:        "easy",
		Notes:       "Зона жиросжигания, можно поддерживать разговор",
	}
}

// === HIIT (High Intensity Interval Training) ===

// GetHIITParams возвращает параметры для HIIT
func (cp *CardioProgression) GetHIITParams(weekNum int) CardioParams {
	var workTime, restTime, rounds int

	switch cp.Experience {
	case "beginner":
		workTime = 20
		restTime = 40
		rounds = 6
	case "intermediate":
		workTime = 30
		restTime = 30
		rounds = 8
	case "advanced":
		workTime = 40
		restTime = 20
		rounds = 10
	}

	// Прогрессия: добавляем раунды
	roundsBonus := (weekNum - 1) / 3
	rounds += roundsBonus
	if rounds > 15 {
		rounds = 15
	}

	return CardioParams{
		Mode:        "intervals",
		Sets:        rounds,
		WorkSeconds: workTime,
		RestSeconds: restTime,
		TargetHRPct: 0.85, // 85% от max HR в работе
		Pace:        "hard",
		Notes:       "Максимальное усилие в рабочих интервалах",
	}
}

// === Hyrox Specific ===

// HyroxStation - станция Hyrox
type HyroxStation struct {
	Name          string
	StandardValue float64 // Стандартное значение (метры, калории, повторения)
	Unit          string  // meters/calories/reps
}

// HyroxStations - стандартные станции Hyrox
var HyroxStations = map[string]HyroxStation{
	"ski_erg":     {Name: "Ski Erg", StandardValue: 1000, Unit: "meters"},
	"sled_push":   {Name: "Sled Push", StandardValue: 50, Unit: "meters"},
	"sled_pull":   {Name: "Sled Pull", StandardValue: 50, Unit: "meters"},
	"burpee_bj":   {Name: "Burpee Broad Jump", StandardValue: 80, Unit: "meters"},
	"rowing":      {Name: "Rowing", StandardValue: 1000, Unit: "meters"},
	"farmer_walk": {Name: "Farmer's Carry", StandardValue: 200, Unit: "meters"},
	"sandbag":     {Name: "Sandbag Lunges", StandardValue: 100, Unit: "meters"},
	"wall_ball":   {Name: "Wall Balls", StandardValue: 100, Unit: "reps"},
}

// GetHyroxStationParams возвращает параметры для станции Hyrox
func (cp *CardioProgression) GetHyroxStationParams(stationID string, weekNum, totalWeeks int, phase string) CardioParams {
	station, ok := HyroxStations[stationID]
	if !ok {
		return CardioParams{}
	}

	// Прогрессия по фазам
	var valuePct float64
	var notes string

	switch phase {
	case "strength":
		// Фаза силы: короткие отрезки с максимальным усилием
		valuePct = 0.3
		notes = "Фокус на силу и технику"
	case "power_endurance":
		// Переход: средние отрезки
		valuePct = 0.5
		notes = "Работа на мощностную выносливость"
	case "specific":
		// Специфика: близко к соревновательным значениям
		valuePct = 0.75
		notes = "Соревновательный темп"
	case "simulation":
		// Симуляция: полные значения
		valuePct = 1.0
		notes = "Полная симуляция соревнований"
	default:
		valuePct = 0.5
	}

	// Недельная прогрессия внутри фазы
	weekInPhase := ((weekNum - 1) % 4) + 1
	valuePct += float64(weekInPhase-1) * 0.05

	value := math.Round(station.StandardValue * valuePct)

	return CardioParams{
		Mode:  station.Unit,
		Value: value,
		Sets:  1,
		Pace:  getPaceForPhase(phase),
		Notes: notes,
	}
}

// GetHyroxRunParams возвращает параметры для беговых отрезков Hyrox
func (cp *CardioProgression) GetHyroxRunParams(weekNum int, phase string) CardioParams {
	// Hyrox: 8 x 1km бега
	standardDistance := 1000.0 // метров

	var sets int
	var distancePct float64
	var notes string

	switch phase {
	case "base":
		// Базовая аэробная работа
		sets = 4
		distancePct = 0.5
		notes = "Темповый бег, сохраняем дыхание"
	case "tempo":
		// Темповые интервалы
		sets = 6
		distancePct = 0.75
		notes = "Соревновательный темп"
	case "specific":
		// Специфическая подготовка
		sets = 8
		distancePct = 1.0
		notes = "Полные соревновательные отрезки"
	case "taper":
		// Подводка
		sets = 4
		distancePct = 0.5
		notes = "Лёгкий бег, сохранение свежести"
	default:
		sets = 4
		distancePct = 0.5
	}

	return CardioParams{
		Mode:        "distance",
		Value:       standardDistance * distancePct,
		Sets:        sets,
		RestSeconds: getRunRestForPhase(phase),
		Pace:        getPaceForPhase(phase),
		Notes:       notes,
	}
}

// === Fat Loss Specific ===

// GetFatLossCardioParams возвращает кардио для жиросжигания
func (cp *CardioProgression) GetFatLossCardioParams(weekNum int, dayType string) CardioParams {
	switch dayType {
	case "liss":
		// LISS: 2-3 раза в неделю
		return cp.GetLISSParams(weekNum)

	case "hiit":
		// HIIT: 1-2 раза в неделю
		return cp.GetHIITParams(weekNum)

	case "density":
		// Плотная работа: круговая тренировка
		return CardioParams{
			Mode:        "intervals",
			Sets:        3,
			WorkSeconds: 45,
			RestSeconds: 15,
			Pace:        "moderate",
			Notes:       "Круговая тренировка, минимальный отдых между упражнениями",
		}
	}

	return CardioParams{}
}

// === Вспомогательные функции ===

func getPaceForPhase(phase string) string {
	switch phase {
	case "strength", "base", "taper":
		return "moderate"
	case "power_endurance", "tempo":
		return "hard"
	case "specific", "simulation":
		return "all_out"
	default:
		return "moderate"
	}
}

func getRunRestForPhase(phase string) int {
	switch phase {
	case "base":
		return 120 // 2 мин
	case "tempo":
		return 90 // 1.5 мин
	case "specific":
		return 60 // 1 мин (как на соревнованиях - переход между станциями)
	case "taper":
		return 180 // 3 мин
	default:
		return 90
	}
}

// GetHRZone возвращает зону пульса
func (cp *CardioProgression) GetHRZone(zoneName string) (int, int) {
	zones := map[string][2]float64{
		"recovery":  {0.50, 0.60},
		"fat_burn":  {0.60, 0.70},
		"aerobic":   {0.70, 0.80},
		"threshold": {0.80, 0.90},
		"anaerobic": {0.90, 1.00},
	}

	zone, ok := zones[zoneName]
	if !ok {
		zone = zones["aerobic"]
	}

	return int(float64(cp.MaxHR) * zone[0]), int(float64(cp.MaxHR) * zone[1])
}

// FormatDuration форматирует время в читаемый формат
func FormatDuration(seconds int) string {
	if seconds < 60 {
		return string(rune('0'+seconds/10)) + string(rune('0'+seconds%10)) + " сек"
	}

	mins := seconds / 60
	secs := seconds % 60

	if secs == 0 {
		return string(rune('0'+mins)) + " мин"
	}

	return string(rune('0'+mins)) + ":" + string(rune('0'+secs/10)) + string(rune('0'+secs%10))
}

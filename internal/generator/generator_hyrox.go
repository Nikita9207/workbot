package generator

import (
	"fmt"

	"workbot/internal/generator/progression"
	"workbot/internal/models"
)

// HyroxGenerator - генератор программ для подготовки к Hyrox
type HyroxGenerator struct {
	selector *ExerciseSelector
	client   *models.ClientProfile
}

// NewHyroxGenerator создаёт новый генератор
func NewHyroxGenerator(selector *ExerciseSelector, client *models.ClientProfile) *HyroxGenerator {
	return &HyroxGenerator{
		selector: selector,
		client:   client,
	}
}

// HyroxConfig - конфигурация программы Hyrox
type HyroxConfig struct {
	TotalWeeks      int  // Всего недель (12-16)
	DaysPerWeek     int  // Дней в неделю (4-5)
	CompetitionDate bool // Есть конкретная дата соревнований
}

// Generate генерирует программу Hyrox
func (g *HyroxGenerator) Generate(config HyroxConfig) (*models.GeneratedProgram, error) {
	program := &models.GeneratedProgram{
		ClientID:      g.client.ID,
		ClientName:    g.client.Name,
		Goal:          models.GoalHyrox,
		Periodization: models.PeriodReverse, // Обратная периодизация
		TotalWeeks:    config.TotalWeeks,
		DaysPerWeek:   config.DaysPerWeek,
	}

	// Определяем фазы (обратная периодизация: сила → выносливость → специфика)
	program.Phases = g.definePhasesHyrox(config.TotalWeeks)

	// Генерируем недели
	for weekNum := 1; weekNum <= config.TotalWeeks; weekNum++ {
		week := g.generateWeekHyrox(weekNum, config)
		program.Weeks = append(program.Weeks, week)
	}

	// Считаем статистику
	program.Statistics = g.calculateStats(program)

	return program, nil
}

// definePhasesHyrox определяет фазы для Hyrox
func (g *HyroxGenerator) definePhasesHyrox(totalWeeks int) []models.ProgramPhase {
	// Обратная периодизация для Hyrox:
	// 1. Сила (силовые качества на станциях)
	// 2. Силовая выносливость (мощностная выносливость)
	// 3. Специфика (симуляция соревнований)
	// 4. Подводка (taper)

	strengthWeeks := totalWeeks / 4
	powerEndWeeks := totalWeeks / 3
	_ = totalWeeks - strengthWeeks - powerEndWeeks - 1 // specificWeeks используется в WeekEnd
	taperWeeks := 1

	return []models.ProgramPhase{
		{
			Name:         "Сила",
			WeekStart:    1,
			WeekEnd:      strengthWeeks,
			Focus:        "Развитие силы на станциях Hyrox",
			IntensityMin: 75,
			IntensityMax: 85,
			VolumeLevel:  "medium",
		},
		{
			Name:         "Силовая выносливость",
			WeekStart:    strengthWeeks + 1,
			WeekEnd:      strengthWeeks + powerEndWeeks,
			Focus:        "Мощностная выносливость, связки бег+станция",
			IntensityMin: 70,
			IntensityMax: 80,
			VolumeLevel:  "high",
		},
		{
			Name:         "Специфика",
			WeekStart:    strengthWeeks + powerEndWeeks + 1,
			WeekEnd:      totalWeeks - taperWeeks,
			Focus:        "Симуляции соревнований, соревновательный темп",
			IntensityMin: 85,
			IntensityMax: 95,
			VolumeLevel:  "medium",
		},
		{
			Name:         "Подводка",
			WeekStart:    totalWeeks,
			WeekEnd:      totalWeeks,
			Focus:        "Восстановление, сохранение формы",
			IntensityMin: 60,
			IntensityMax: 70,
			VolumeLevel:  "low",
		},
	}
}

// generateWeekHyrox генерирует неделю Hyrox
func (g *HyroxGenerator) generateWeekHyrox(weekNum int, config HyroxConfig) models.GeneratedWeek {
	phase := g.getPhaseForWeek(weekNum, config.TotalWeeks)
	isTaper := weekNum == config.TotalWeeks

	week := models.GeneratedWeek{
		WeekNum:   weekNum,
		PhaseName: phase,
		IsDeload:  isTaper,
	}

	if isTaper {
		week.IntensityPercent = 65
		week.VolumePercent = 50
		week.RPETarget = 6
	} else {
		week.IntensityPercent = g.getIntensityForPhase(phase)
		week.VolumePercent = g.getVolumeForPhase(phase)
		week.RPETarget = g.getRPEForPhase(phase)
	}

	// Генерируем дни
	dayTypes := g.getHyroxDayTypes(config.DaysPerWeek, phase)
	for dayNum, dayType := range dayTypes {
		day := g.generateDayHyrox(dayNum+1, dayType, weekNum, config, phase)
		week.Days = append(week.Days, day)
	}

	return week
}

// getHyroxDayTypes возвращает типы дней для Hyrox
// Hyrox = бег + станции, поэтому даже в силовой фазе нужен бег и работа на станциях
func (g *HyroxGenerator) getHyroxDayTypes(daysPerWeek int, phase string) []string {
	// Базовые шаблоны для каждого количества дней
	// Принцип: всегда бег + станции + сила, меняется только пропорция

	switch daysPerWeek {
	case 2:
		// Минимум: комбинированные тренировки
		switch phase {
		case "strength":
			return []string{"strength_stations", "run_intervals"}
		case "power_endurance":
			return []string{"station_work", "run_tempo"}
		case "specific":
			return []string{"simulation_half", "run_race_pace"}
		case "taper":
			return []string{"run_easy", "station_light"}
		}

	case 3:
		// Оптимальный минимум для Hyrox
		switch phase {
		case "strength":
			return []string{"strength_stations", "run_intervals", "station_work"}
		case "power_endurance":
			return []string{"station_work", "run_tempo", "strength_circuit"}
		case "specific":
			return []string{"simulation_half", "run_race_pace", "station_work"}
		case "taper":
			return []string{"run_easy", "station_light", "run_easy"}
		}

	case 4:
		switch phase {
		case "strength":
			return []string{"strength_stations", "run_intervals", "station_work", "run_tempo"}
		case "power_endurance":
			return []string{"station_work", "run_intervals", "strength_circuit", "run_tempo"}
		case "specific":
			return []string{"simulation_half", "run_race_pace", "station_work", "simulation_mini"}
		case "taper":
			return []string{"run_easy", "station_light", "run_easy", "rest"}
		}

	default: // 5+ дней
		switch phase {
		case "strength":
			return []string{"strength_stations", "run_intervals", "station_work", "run_tempo", "strength_full"}
		case "power_endurance":
			return []string{"station_work", "run_intervals", "strength_circuit", "run_tempo", "simulation_mini"}
		case "specific":
			return []string{"simulation_half", "run_race_pace", "station_work", "run_intervals", "simulation_full"}
		case "taper":
			return []string{"run_easy", "station_light", "run_easy", "rest", "rest"}
		}
	}

	// Fallback
	return []string{"station_work", "run_intervals", "strength_stations"}[:daysPerWeek]
}

// generateDayHyrox генерирует день Hyrox
func (g *HyroxGenerator) generateDayHyrox(dayNum int, dayType string, weekNum int, config HyroxConfig, phase string) models.GeneratedDay {
	day := models.GeneratedDay{
		DayNum: dayNum,
		Name:   fmt.Sprintf("День %d — %s", dayNum, g.getHyroxDayName(dayType)),
		Type:   dayType,
	}

	age := 30
	if g.client.Age > 0 {
		age = g.client.Age
	}
	cardioProg := progression.NewCardioProgression("hyrox", string(g.client.Experience), age)

	switch dayType {
	case "strength_upper", "strength_lower", "strength_full":
		day.Exercises = g.generateStrengthDay(dayType, weekNum, config.TotalWeeks)

	case "strength_stations":
		day.Exercises = g.generateStrengthStationsDay(weekNum, config.TotalWeeks)

	case "run_easy":
		day.Exercises = g.generateRunDay("base", weekNum, cardioProg)

	case "run_tempo":
		day.Exercises = g.generateRunDay("tempo", weekNum, cardioProg)

	case "run_intervals":
		day.Exercises = g.generateRunDay("intervals", weekNum, cardioProg)

	case "run_race_pace":
		day.Exercises = g.generateRunDay("specific", weekNum, cardioProg)

	case "run_recovery":
		day.Exercises = g.generateRunDay("recovery", weekNum, cardioProg)

	case "station_work":
		day.Exercises = g.generateStationWork(weekNum, phase, cardioProg)

	case "station_light":
		day.Exercises = g.generateStationWorkLight(weekNum, cardioProg)

	case "strength_circuit":
		day.Exercises = g.generateStrengthCircuit(weekNum)

	case "simulation_mini":
		day.Exercises = g.generateSimulation("mini", weekNum, cardioProg)

	case "simulation_half":
		day.Exercises = g.generateSimulation("half", weekNum, cardioProg)

	case "simulation_full":
		day.Exercises = g.generateSimulation("full", weekNum, cardioProg)

	case "rest":
		day.Exercises = []models.GeneratedExercise{
			{OrderNum: 1, ExerciseName: "Отдых", Notes: "Активное восстановление или полный отдых"},
		}
	}

	// Оценка длительности
	totalTime := 0
	for _, ex := range day.Exercises {
		if ex.RestSeconds > 0 {
			totalTime += ex.Sets * ex.RestSeconds / 60
		} else {
			totalTime += ex.Sets * 3
		}
	}
	day.EstimatedDuration = maxInt(totalTime, 30)

	return day
}

// generateStrengthDay генерирует силовой день для Hyrox
func (g *HyroxGenerator) generateStrengthDay(dayType string, weekNum, totalWeeks int) []models.GeneratedExercise {
	var exercises []models.GeneratedExercise

	weightProgs := make(map[string]*progression.WeightProgression)
	for movement, onePM := range g.client.OnePM {
		weightProgs[movement] = progression.NewWeightProgression(onePM)
	}

	// Упражнения специфичные для Hyrox
	switch dayType {
	case "strength_upper":
		exercises = []models.GeneratedExercise{
			{OrderNum: 1, ExerciseName: "Жим штанги лёжа", Sets: 4, Reps: "6-8"},
			{OrderNum: 2, ExerciseName: "Тяга штанги в наклоне", Sets: 4, Reps: "6-8"},
			{OrderNum: 3, ExerciseName: "Жим штанги стоя", Sets: 3, Reps: "8-10"},
			{OrderNum: 4, ExerciseName: "Подтягивания", Sets: 3, Reps: "8-10"},
			{OrderNum: 5, ExerciseName: "Farmer's Walk", Sets: 4, Reps: "40м", Notes: "Специфика Hyrox"},
		}

	case "strength_lower":
		exercises = []models.GeneratedExercise{
			{OrderNum: 1, ExerciseName: "Приседания со штангой", Sets: 4, Reps: "6-8"},
			{OrderNum: 2, ExerciseName: "Румынская тяга", Sets: 4, Reps: "8-10"},
			{OrderNum: 3, ExerciseName: "Выпады с гантелями", Sets: 3, Reps: "10+10", Notes: "Специфика для Sandbag Lunges"},
			{OrderNum: 4, ExerciseName: "Hip Thrust со штангой", Sets: 3, Reps: "10-12"},
			{OrderNum: 5, ExerciseName: "Wall Balls", Sets: 4, Reps: "20", Notes: "Специфика Hyrox"},
		}

	case "strength_full":
		exercises = []models.GeneratedExercise{
			{OrderNum: 1, ExerciseName: "Становая тяга", Sets: 4, Reps: "5"},
			{OrderNum: 2, ExerciseName: "Жим штанги лёжа", Sets: 3, Reps: "8"},
			{OrderNum: 3, ExerciseName: "Гоблет-присед", Sets: 3, Reps: "12"},
			{OrderNum: 4, ExerciseName: "Тяга верхнего блока", Sets: 3, Reps: "10"},
			{OrderNum: 5, ExerciseName: "Sled Push", Sets: 4, Reps: "25м", Notes: "Специфика Hyrox"},
		}
	}

	// Добавляем веса если есть 1ПМ
	for i := range exercises {
		for movement, wp := range weightProgs {
			if containsMovementName(exercises[i].ExerciseName, movement) {
				params := wp.GetStrengthParams(weekNum, "strength")
				exercises[i].Weight = params.Weight
				exercises[i].WeightPercent = params.Intensity
				exercises[i].RestSeconds = 120
				break
			}
		}
		if exercises[i].RestSeconds == 0 {
			exercises[i].RestSeconds = 90
		}
	}

	return exercises
}

// generateStrengthStationsDay генерирует силовой день с упражнениями специфичными для станций Hyrox
func (g *HyroxGenerator) generateStrengthStationsDay(weekNum, totalWeeks int) []models.GeneratedExercise {
	// Силовые упражнения, которые напрямую переносятся на станции Hyrox
	exercises := []models.GeneratedExercise{
		// Ноги (Sled Push/Pull, Lunges, Wall Balls)
		{OrderNum: 1, ExerciseName: "Гоблет-присед", Sets: 4, Reps: "12-15", RestSeconds: 60,
			Notes: "Перенос на Wall Balls"},
		{OrderNum: 2, ExerciseName: "Выпады с гантелями (ходьба)", Sets: 3, Reps: "20 шагов", RestSeconds: 60,
			Notes: "Перенос на Sandbag Lunges"},
		// Тяговые (Sled Pull, Rowing)
		{OrderNum: 3, ExerciseName: "Тяга горизонтального блока", Sets: 4, Reps: "12-15", RestSeconds: 60,
			Notes: "Перенос на Rowing и Sled Pull"},
		// Толкающие (Sled Push, SkiErg)
		{OrderNum: 4, ExerciseName: "Жим гантелей стоя", Sets: 3, Reps: "10-12", RestSeconds: 60,
			Notes: "Стабилизация для Sled Push"},
		// Хват (Farmer's Carry)
		{OrderNum: 5, ExerciseName: "Farmer's Walk", Sets: 4, Reps: "40м", RestSeconds: 90,
			Notes: "Станция Hyrox: 2x100м с 2x24кг"},
		// Метаболическое завершение
		{OrderNum: 6, ExerciseName: "Burpees", Sets: 3, Reps: "10", RestSeconds: 45,
			Notes: "Перенос на Burpee Broad Jumps"},
	}

	// Прогрессия по неделям: увеличиваем подходы
	if weekNum > totalWeeks/2 {
		for i := range exercises {
			if exercises[i].Sets < 5 {
				exercises[i].Sets++
			}
		}
	}

	return exercises
}

// generateRunDay генерирует беговой день
// Hyrox = 8x1км бега, поэтому тренировки строятся вокруг 1 км интервалов
func (g *HyroxGenerator) generateRunDay(runType string, weekNum int, cardioProg *progression.CardioProgression) []models.GeneratedExercise {
	var exercises []models.GeneratedExercise

	// Разминка
	exercises = append(exercises, models.GeneratedExercise{
		OrderNum:     1,
		ExerciseName: "Разминка",
		Sets:         1,
		Reps:         "10 мин",
		Notes:        "Лёгкий бег + динамическая растяжка + СБУ",
	})

	// Основная часть зависит от типа
	switch runType {
	case "intervals":
		// Hyrox специфика: 1км интервалы
		intervals := 4 + weekNum/2
		if intervals > 8 {
			intervals = 8
		}
		exercises = append(exercises, models.GeneratedExercise{
			OrderNum:     2,
			ExerciseName: "Интервалы 1 км",
			Sets:         intervals,
			Reps:         "1000м",
			RestSeconds:  120, // 2 мин отдых (имитация станции)
			Notes:        fmt.Sprintf("Темп: 4:30-5:00/км. Цель: %d км общего объёма", intervals),
		})

	case "tempo":
		// Темповый бег на соревновательном пейсе
		distance := 4000 + weekNum*500
		if distance > 8000 {
			distance = 8000
		}
		exercises = append(exercises, models.GeneratedExercise{
			OrderNum:     2,
			ExerciseName: "Темповый бег",
			Sets:         1,
			Reps:         fmt.Sprintf("%dм", distance),
			Notes:        "Равномерный темп 5:00-5:30/км. Зона ЧСС 80-85%",
		})

	case "specific", "race_pace":
		// Соревновательный темп с имитацией станций
		exercises = append(exercises, models.GeneratedExercise{
			OrderNum:     2,
			ExerciseName: "Бег + имитация станций",
			Sets:         4,
			Reps:         "1000м бег + 30 приседаний",
			RestSeconds:  60,
			Notes:        "Соревновательный темп 4:45-5:15/км. Приседания = имитация станции",
		})

	case "recovery":
		exercises = append(exercises, models.GeneratedExercise{
			OrderNum:     2,
			ExerciseName: "Восстановительный бег",
			Sets:         1,
			Reps:         "30-40 мин",
			Notes:        "Очень лёгкий темп, ЧСС < 140. Можно разбить на интервалы бег/ходьба",
		})

	case "base", "easy":
		fallthrough
	default:
		exercises = append(exercises, models.GeneratedExercise{
			OrderNum:     2,
			ExerciseName: "Базовый бег",
			Sets:         1,
			Reps:         "30-45 мин",
			Notes:        "Комфортный темп 5:30-6:00/км. Можно поддерживать разговор",
		})
	}

	// Заминка
	exercises = append(exercises, models.GeneratedExercise{
		OrderNum:     3,
		ExerciseName: "Заминка",
		Sets:         1,
		Reps:         "5 мин",
		Notes:        "Лёгкий бег + растяжка",
	})

	return exercises
}

// generateStationWork генерирует работу на станциях
func (g *HyroxGenerator) generateStationWork(weekNum int, phase string, cardioProg *progression.CardioProgression) []models.GeneratedExercise {
	var exercises []models.GeneratedExercise

	// Станции Hyrox в порядке соревнований
	// Реальные параметры: SkiErg 1000м, Sled Push 50м, Sled Pull 50м, Burpee BJ 80м,
	// Rowing 1000м, Farmer 200м, Lunges 100м, Wall Balls 100
	type stationDef struct {
		name     string
		reps     string
		fullReps string // Соревновательный объём
		notes    string
	}

	allStations := []stationDef{
		{"Ski Erg", "500м", "1000м", "Темп: 1:50-2:00/500м"},
		{"Sled Push", "4x25м", "2x25м", "Вес: 152/102кг (M/F)"},
		{"Sled Pull", "4x25м", "2x25м", "Вес: 103/78кг (M/F)"},
		{"Burpee Broad Jump", "40м", "80м", "Прыжок вперёд после каждого бёрпи"},
		{"Rowing", "500м", "1000м", "Темп: 1:45-1:55/500м"},
		{"Farmer's Carry", "2x50м", "2x100м", "Вес: 2x24/2x16кг (M/F)"},
		{"Sandbag Lunges", "50м", "100м", "Вес: 20/10кг"},
		{"Wall Balls", "50", "100", "Вес: 9/6кг, высота 3м/2.7м"},
	}

	// Ротация: каждую неделю смещаем начало на 2 станции
	startIdx := (weekNum * 2) % len(allStations)

	// Количество станций зависит от фазы
	numStations := 4
	if phase == "power_endurance" {
		numStations = 5
	} else if phase == "specific" {
		numStations = 6
	}

	// Добавляем разминку
	exercises = append(exercises, models.GeneratedExercise{
		OrderNum:     1,
		ExerciseName: "Разминка",
		Sets:         1,
		Reps:         "10 мин",
		Notes:        "Лёгкий бег 5 мин + мобильность",
	})

	// Добавляем станции
	for i := 0; i < numStations; i++ {
		station := allStations[(startIdx+i)%len(allStations)]

		reps := station.reps
		if phase == "specific" {
			reps = station.fullReps // В специфичной фазе — соревновательный объём
		}

		sets := 2
		if phase == "strength" {
			sets = 3 // Больше подходов в силовой фазе
		}

		ex := models.GeneratedExercise{
			OrderNum:     i + 2,
			ExerciseName: station.name,
			Sets:         sets,
			Reps:         reps,
			RestSeconds:  90,
			Notes:        station.notes,
		}
		exercises = append(exercises, ex)
	}

	return exercises
}

// generateStationWorkLight генерирует лёгкую работу на станциях (для подводки)
func (g *HyroxGenerator) generateStationWorkLight(weekNum int, cardioProg *progression.CardioProgression) []models.GeneratedExercise {
	return []models.GeneratedExercise{
		{OrderNum: 1, ExerciseName: "Ski Erg", Sets: 2, Reps: "500 м", RestSeconds: 120, Notes: "Лёгкий темп"},
		{OrderNum: 2, ExerciseName: "Wall Balls", Sets: 2, Reps: "20", RestSeconds: 90, Notes: "Техника"},
		{OrderNum: 3, ExerciseName: "Rowing", Sets: 2, Reps: "500 м", RestSeconds: 120, Notes: "Лёгкий темп"},
	}
}

// generateStrengthCircuit генерирует силовую круговую тренировку
func (g *HyroxGenerator) generateStrengthCircuit(weekNum int) []models.GeneratedExercise {
	rounds := 3 + weekNum/4
	if rounds > 5 {
		rounds = 5
	}

	return []models.GeneratedExercise{
		{
			OrderNum:     1,
			ExerciseName: "Силовая круговая",
			Sets:         rounds,
			Reps:         "5 упражнений по 45 сек",
			RestSeconds:  60,
			Notes:        "Гоблет-присед → Отжимания → Тяга гири → Выпады → Планка. Без отдыха между упражнениями, 60 сек между раундами",
		},
	}
}

// generateSimulation генерирует симуляцию Hyrox
func (g *HyroxGenerator) generateSimulation(simType string, weekNum int, cardioProg *progression.CardioProgression) []models.GeneratedExercise {
	var exercises []models.GeneratedExercise

	switch simType {
	case "mini":
		// 2 станции + 2 км бега
		exercises = []models.GeneratedExercise{
			{OrderNum: 1, ExerciseName: "Бег", Sets: 1, Reps: "1000 м", Notes: "Соревновательный темп"},
			{OrderNum: 2, ExerciseName: "Ski Erg", Sets: 1, Reps: "1000 м", Notes: "Без перерыва"},
			{OrderNum: 3, ExerciseName: "Бег", Sets: 1, Reps: "1000 м", Notes: "Соревновательный темп"},
			{OrderNum: 4, ExerciseName: "Wall Balls", Sets: 1, Reps: "100", Notes: "Без перерыва"},
		}

	case "half":
		// 4 станции + 4 км бега
		exercises = []models.GeneratedExercise{
			{OrderNum: 1, ExerciseName: "Бег", Sets: 1, Reps: "1000 м"},
			{OrderNum: 2, ExerciseName: "Ski Erg", Sets: 1, Reps: "1000 м"},
			{OrderNum: 3, ExerciseName: "Бег", Sets: 1, Reps: "1000 м"},
			{OrderNum: 4, ExerciseName: "Sled Push", Sets: 1, Reps: "50 м"},
			{OrderNum: 5, ExerciseName: "Бег", Sets: 1, Reps: "1000 м"},
			{OrderNum: 6, ExerciseName: "Sled Pull", Sets: 1, Reps: "50 м"},
			{OrderNum: 7, ExerciseName: "Бег", Sets: 1, Reps: "1000 м"},
			{OrderNum: 8, ExerciseName: "Wall Balls", Sets: 1, Reps: "100"},
		}
		for i := range exercises {
			exercises[i].Notes = "Без перерывов между станциями"
		}

	case "full":
		// Полная симуляция: 8 станций + 8 км бега
		stations := []string{"Ski Erg 1000м", "Sled Push 50м", "Sled Pull 50м", "Burpee Broad Jump 80м",
			"Rowing 1000м", "Farmer's Carry 200м", "Sandbag Lunges 100м", "Wall Balls 100"}

		for i, station := range stations {
			exercises = append(exercises, models.GeneratedExercise{
				OrderNum:     i*2 + 1,
				ExerciseName: "Бег",
				Sets:         1,
				Reps:         "1000 м",
			})
			exercises = append(exercises, models.GeneratedExercise{
				OrderNum:     i*2 + 2,
				ExerciseName: station,
				Sets:         1,
				Reps:         "-",
			})
		}
		exercises[0].Notes = "ПОЛНАЯ СИМУЛЯЦИЯ HYROX"
	}

	return exercises
}

// === Вспомогательные методы ===

func (g *HyroxGenerator) getPhaseForWeek(weekNum, totalWeeks int) string {
	strengthWeeks := totalWeeks / 4
	powerEndWeeks := totalWeeks / 3

	if weekNum <= strengthWeeks {
		return "strength"
	}
	if weekNum <= strengthWeeks+powerEndWeeks {
		return "power_endurance"
	}
	if weekNum < totalWeeks {
		return "specific"
	}
	return "taper"
}

func (g *HyroxGenerator) getIntensityForPhase(phase string) float64 {
	switch phase {
	case "strength":
		return 80
	case "power_endurance":
		return 75
	case "specific":
		return 90
	case "taper":
		return 65
	}
	return 75
}

func (g *HyroxGenerator) getVolumeForPhase(phase string) float64 {
	switch phase {
	case "strength":
		return 80
	case "power_endurance":
		return 100
	case "specific":
		return 90
	case "taper":
		return 50
	}
	return 80
}

func (g *HyroxGenerator) getRPEForPhase(phase string) float64 {
	switch phase {
	case "strength":
		return 8
	case "power_endurance":
		return 7.5
	case "specific":
		return 9
	case "taper":
		return 6
	}
	return 7.5
}

func (g *HyroxGenerator) getHyroxDayName(dayType string) string {
	names := map[string]string{
		"strength_upper":    "Сила (верх)",
		"strength_lower":    "Сила (низ)",
		"strength_full":     "Сила (всё тело)",
		"strength_stations": "Сила для станций",
		"run_easy":          "Бег (лёгкий)",
		"run_tempo":         "Бег (темповый)",
		"run_intervals":     "Бег (интервалы)",
		"run_race_pace":     "Бег (соревн. темп)",
		"run_recovery":      "Бег (восстановление)",
		"station_work":      "Работа на станциях",
		"station_light":     "Станции (лёгко)",
		"strength_circuit":  "Силовая круговая",
		"simulation_mini":   "Мини-симуляция",
		"simulation_half":   "Полу-симуляция",
		"simulation_full":   "Полная симуляция",
		"rest":              "Отдых",
	}
	if name, ok := names[dayType]; ok {
		return name
	}
	return dayType
}

func (g *HyroxGenerator) getRunTypeName(runType string) string {
	names := map[string]string{
		"base":     "базовый",
		"tempo":    "темповый",
		"intervals": "интервалы",
		"specific": "соревн. темп",
		"recovery": "восстановление",
	}
	if name, ok := names[runType]; ok {
		return name
	}
	return runType
}

func containsMovementName(exerciseName, movement string) bool {
	switch movement {
	case "squat":
		return exerciseName == "Приседания со штангой" || exerciseName == "Гоблет-присед"
	case "bench":
		return exerciseName == "Жим штанги лёжа"
	case "deadlift":
		return exerciseName == "Становая тяга" || exerciseName == "Румынская тяга"
	case "ohp":
		return exerciseName == "Жим штанги стоя"
	}
	return false
}

func (g *HyroxGenerator) calculateStats(program *models.GeneratedProgram) models.ProgramStats {
	stats := models.ProgramStats{
		SetsPerMuscle: make(map[models.MuscleGroupExt]int),
	}

	for _, week := range program.Weeks {
		for _, day := range week.Days {
			stats.TotalWorkouts++
			for _, ex := range day.Exercises {
				stats.TotalSets += ex.Sets
			}
		}
	}

	if stats.TotalWorkouts > 0 {
		stats.AvgWorkoutDur = 60 // Hyrox тренировки обычно ~60 мин
	}

	return stats
}

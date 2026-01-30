package generator

import (
	"fmt"

	"workbot/internal/generator/progression"
	"workbot/internal/models"
)

// FatLossGenerator - генератор программ для жиросжигания
type FatLossGenerator struct {
	selector *ExerciseSelector
	client   *models.ClientProfile
}

// NewFatLossGenerator создаёт новый генератор
func NewFatLossGenerator(selector *ExerciseSelector, client *models.ClientProfile) *FatLossGenerator {
	return &FatLossGenerator{
		selector: selector,
		client:   client,
	}
}

// FatLossConfig - конфигурация программы жиросжигания
type FatLossConfig struct {
	TotalWeeks  int // Всего недель (8-12)
	DaysPerWeek int // Дней в неделю (4-5)
	IncludeLISS bool // Включить LISS кардио
	IncludeHIIT bool // Включить HIIT
}

// Generate генерирует программу жиросжигания
func (g *FatLossGenerator) Generate(config FatLossConfig) (*models.GeneratedProgram, error) {
	program := &models.GeneratedProgram{
		ClientID:      g.client.ID,
		ClientName:    g.client.Name,
		Goal:          models.GoalFatLoss,
		Periodization: models.PeriodLinear,
		TotalWeeks:    config.TotalWeeks,
		DaysPerWeek:   config.DaysPerWeek,
	}

	// Определяем фазы (линейная периодизация с фокусом на плотность)
	program.Phases = g.definePhaseFatLoss(config.TotalWeeks)

	// Генерируем недели
	for weekNum := 1; weekNum <= config.TotalWeeks; weekNum++ {
		week := g.generateWeekFatLoss(weekNum, config)
		program.Weeks = append(program.Weeks, week)
	}

	// Считаем статистику
	program.Statistics = g.calculateStats(program)

	return program, nil
}

// definePhaseFatLoss определяет фазы для жиросжигания
func (g *FatLossGenerator) definePhaseFatLoss(totalWeeks int) []models.ProgramPhase {
	// Линейная периодизация:
	// - Сохраняем интенсивность (для сохранения мышц)
	// - Постепенно увеличиваем плотность тренировки
	// - Добавляем кардио

	midPoint := totalWeeks / 2

	return []models.ProgramPhase{
		{
			Name:         "Адаптация",
			WeekStart:    1,
			WeekEnd:      midPoint,
			Focus:        "Адаптация к дефициту, сохранение силы",
			IntensityMin: 75,
			IntensityMax: 82,
			VolumeLevel:  "medium",
		},
		{
			Name:         "Интенсификация",
			WeekStart:    midPoint + 1,
			WeekEnd:      totalWeeks - 1,
			Focus:        "Увеличение плотности, добавление кардио",
			IntensityMin: 75,
			IntensityMax: 85,
			VolumeLevel:  "medium-high",
		},
		{
			Name:         "Финиш",
			WeekStart:    totalWeeks,
			WeekEnd:      totalWeeks,
			Focus:        "Максимальная плотность, пик формы",
			IntensityMin: 80,
			IntensityMax: 85,
			VolumeLevel:  "high",
		},
	}
}

// generateWeekFatLoss генерирует неделю для жиросжигания
func (g *FatLossGenerator) generateWeekFatLoss(weekNum int, config FatLossConfig) models.GeneratedWeek {
	isDeload := weekNum%4 == 0 && weekNum < config.TotalWeeks

	week := models.GeneratedWeek{
		WeekNum:   weekNum,
		PhaseName: g.getPhaseNameForWeek(weekNum, config.TotalWeeks),
		IsDeload:  isDeload,
	}

	if isDeload {
		week.IntensityPercent = 65
		week.VolumePercent = 60
		week.RPETarget = 6
	} else {
		// Интенсивность сохраняется высокой
		week.IntensityPercent = 75 + float64(weekNum)*0.5
		if week.IntensityPercent > 85 {
			week.IntensityPercent = 85
		}
		week.VolumePercent = 80
		week.RPETarget = 8
	}

	// Генерируем дни
	dayTypes := g.getFatLossDayTypes(config)
	for dayNum, dayType := range dayTypes {
		day := g.generateDayFatLoss(dayNum+1, dayType, weekNum, config, isDeload)
		week.Days = append(week.Days, day)
	}

	return week
}

// getFatLossDayTypes возвращает типы дней
func (g *FatLossGenerator) getFatLossDayTypes(config FatLossConfig) []string {
	switch config.DaysPerWeek {
	case 5:
		if config.IncludeHIIT {
			return []string{"upper", "lower", "hiit", "upper", "lower"}
		}
		return []string{"upper", "lower", "fullbody", "upper", "lower"}
	case 4:
		if config.IncludeHIIT {
			return []string{"upper", "lower", "hiit", "fullbody"}
		}
		return []string{"upper", "lower", "upper", "lower"}
	default:
		return []string{"fullbody", "fullbody", "fullbody"}
	}
}

// generateDayFatLoss генерирует день для жиросжигания
func (g *FatLossGenerator) generateDayFatLoss(dayNum int, dayType string, weekNum int, config FatLossConfig, isDeload bool) models.GeneratedDay {
	day := models.GeneratedDay{
		DayNum: dayNum,
		Type:   dayType,
	}

	switch dayType {
	case "hiit":
		day.Name = fmt.Sprintf("День %d — HIIT", dayNum)
		day.Exercises = g.generateHIITExercises(weekNum, isDeload)
	case "liss":
		day.Name = fmt.Sprintf("День %d — LISS Кардио", dayNum)
		day.Exercises = g.generateLISSExercises(weekNum)
	default:
		day.Name = fmt.Sprintf("День %d — %s", dayNum, getDayTypeName(dayType))
		day.Exercises = g.generateStrengthDayFatLoss(dayType, weekNum, config.TotalWeeks, isDeload)
	}

	// Добавляем финишер (если не разгрузка и не кардио день)
	if !isDeload && dayType != "hiit" && dayType != "liss" {
		finisher := g.generateFinisher(weekNum)
		finisher.OrderNum = len(day.Exercises) + 1 // Корректный номер после всех упражнений
		day.Exercises = append(day.Exercises, finisher)
	}

	// Оценка длительности
	totalSets := 0
	for _, ex := range day.Exercises {
		totalSets += ex.Sets
	}
	day.EstimatedDuration = totalSets * 2 // Короткий отдых = ~2 мин на подход

	return day
}

// generateStrengthDayFatLoss генерирует силовой день для жиросжигания
func (g *FatLossGenerator) generateStrengthDayFatLoss(dayType string, weekNum, totalWeeks int, isDeload bool) []models.GeneratedExercise {
	var exercises []models.GeneratedExercise

	// Создаём прогрессии
	weightProgs := make(map[string]*progression.WeightProgression)
	for movement, onePM := range g.client.OnePM {
		weightProgs[movement] = progression.NewWeightProgression(onePM)
	}

	// Получаем упражнения через selector
	difficulty := g.getDifficultyLevel()
	selections := g.selector.SelectExercisesForDay(
		dayType,
		g.client.AvailableEquip,
		g.client.Constraints,
		difficulty,
	)

	// Сокращаем объём (fat loss = меньше подходов)
	for orderNum, result := range selections {
		ex := models.GeneratedExercise{
			OrderNum:     orderNum + 1,
			ExerciseID:   result.Exercise.ID,
			ExerciseName: result.Exercise.NameRu,
			MovementType: result.Exercise.MovementType,
		}

		if len(result.Exercise.PrimaryMuscles) > 0 {
			ex.MuscleGroup = result.Exercise.PrimaryMuscles[0]
		}

		// Параметры для fat loss:
		// - Сохраняем интенсивность (80-85%)
		// - Сокращаем объём (-30-40%)
		// - Короткий отдых (60-90 сек)

		if isDeload {
			ex.Sets = 2
			ex.Reps = "8"
			ex.RestSeconds = 90
			ex.RPE = 6
		} else {
			ex.Sets = 3 // Сокращённый объём
			ex.Reps = "8-10"
			ex.RestSeconds = 60 // Короткий отдых
			ex.RPE = 8

			// Пробуем найти 1ПМ
			for movement, wp := range weightProgs {
				if matchesMovement(result.Exercise, movement) {
					intensity := 80.0 + float64(weekNum)*0.5
					if intensity > 85 {
						intensity = 85
					}
					ex.Weight = wp.CalculateWeight(intensity)
					ex.WeightPercent = intensity
					break
				}
			}
		}

		// Альтернатива
		if result.Alternative != nil {
			alt := models.GeneratedExercise{
				ExerciseID:   result.Alternative.ID,
				ExerciseName: result.Alternative.NameRu,
			}
			ex.Alternative = &alt
		}

		exercises = append(exercises, ex)
	}

	return exercises
}

// generateHIITExercises генерирует HIIT тренировку
func (g *FatLossGenerator) generateHIITExercises(weekNum int, isDeload bool) []models.GeneratedExercise {
	age := 30 // default
	if g.client.Age > 0 {
		age = g.client.Age
	}

	cardioProg := progression.NewCardioProgression("fat_loss", string(g.client.Experience), age)
	params := cardioProg.GetHIITParams(weekNum)

	if isDeload {
		params.Sets = params.Sets / 2
	}

	return []models.GeneratedExercise{
		{
			OrderNum:     1,
			ExerciseName: "Разминка",
			Sets:         1,
			Reps:         "5 мин",
			Notes:        "Лёгкое кардио",
		},
		{
			OrderNum:     2,
			ExerciseName: "HIIT интервалы",
			Sets:         params.Sets,
			Reps:         fmt.Sprintf("%d сек работа / %d сек отдых", params.WorkSeconds, params.RestSeconds),
			Notes:        params.Notes,
		},
		{
			OrderNum:     3,
			ExerciseName: "Заминка",
			Sets:         1,
			Reps:         "5 мин",
			Notes:        "Растяжка",
		},
	}
}

// generateLISSExercises генерирует LISS тренировку
func (g *FatLossGenerator) generateLISSExercises(weekNum int) []models.GeneratedExercise {
	age := 30
	if g.client.Age > 0 {
		age = g.client.Age
	}

	cardioProg := progression.NewCardioProgression("fat_loss", string(g.client.Experience), age)
	params := cardioProg.GetLISSParams(weekNum)

	duration := int(params.Value / 60) // секунды в минуты

	return []models.GeneratedExercise{
		{
			OrderNum:     1,
			ExerciseName: "LISS Кардио",
			Sets:         1,
			Reps:         fmt.Sprintf("%d мин", duration),
			Notes:        fmt.Sprintf("%s. Пульс: %.0f%% от максимума", params.Notes, params.TargetHRPct*100),
		},
	}
}

// generateFinisher генерирует финишер (метаболическая работа в конце)
// Адаптировано под пол: женщинам больше ягодичной работы
func (g *FatLossGenerator) generateFinisher(weekNum int) models.GeneratedExercise {
	isFemale := g.client.Gender == "female" || g.client.Gender == "женский" || g.client.Gender == "ж"

	// Прогрессия финишера по неделям
	rounds := 2 + weekNum/3
	if rounds > 5 {
		rounds = 5
	}

	var notes string
	if isFemale {
		// Женский вариант: больше ягодичной работы
		notes = "Ягодичный мостик, выпады, скалолаз, планка — по кругу"
	} else {
		// Мужской вариант
		notes = "Бёрпи, скалолаз, джампинг джек, планка — по кругу"
	}

	return models.GeneratedExercise{
		OrderNum:     0, // Будет установлен при добавлении
		ExerciseName: "Финишер (круговая)",
		Sets:         rounds,
		Reps:         "45 сек работа / 15 сек отдых",
		RestSeconds:  60,
		Notes:        notes,
	}
}

// === Вспомогательные методы ===

func (g *FatLossGenerator) getPhaseNameForWeek(weekNum, totalWeeks int) string {
	midPoint := totalWeeks / 2

	if weekNum <= midPoint {
		return "Адаптация"
	}
	if weekNum < totalWeeks {
		return "Интенсификация"
	}
	return "Финиш"
}

func (g *FatLossGenerator) getDifficultyLevel() models.DifficultyLevel {
	switch g.client.Experience {
	case models.ExpAdvanced:
		return models.DifficultyAdvanced
	case models.ExpIntermediate:
		return models.DifficultyIntermediate
	default:
		return models.DifficultyBeginner
	}
}

func (g *FatLossGenerator) calculateStats(program *models.GeneratedProgram) models.ProgramStats {
	stats := models.ProgramStats{
		SetsPerMuscle: make(map[models.MuscleGroupExt]int),
	}

	for _, week := range program.Weeks {
		for _, day := range week.Days {
			stats.TotalWorkouts++
			for _, ex := range day.Exercises {
				stats.TotalSets += ex.Sets
				if ex.Weight > 0 {
					reps := 8
					fmt.Sscanf(ex.Reps, "%d", &reps)
					stats.TotalVolume += ex.Weight * float64(ex.Sets*reps)
				}
				stats.SetsPerMuscle[ex.MuscleGroup] += ex.Sets
			}
		}
	}

	if stats.TotalWorkouts > 0 {
		stats.AvgWorkoutDur = (stats.TotalSets * 2) / stats.TotalWorkouts
	}

	return stats
}

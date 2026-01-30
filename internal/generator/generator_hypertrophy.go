package generator

import (
	"fmt"

	"workbot/internal/generator/progression"
	"workbot/internal/models"
)

// HypertrophyGenerator - генератор программ для гипертрофии
type HypertrophyGenerator struct {
	selector *ExerciseSelector
	client   *models.ClientProfile
}

// NewHypertrophyGenerator создаёт новый генератор гипертрофии
func NewHypertrophyGenerator(selector *ExerciseSelector, client *models.ClientProfile) *HypertrophyGenerator {
	return &HypertrophyGenerator{
		selector: selector,
		client:   client,
	}
}

// HypertrophyConfig - конфигурация программы гипертрофии
type HypertrophyConfig struct {
	TotalWeeks   int // Всего недель (8-12)
	DaysPerWeek  int // Дней в неделю (3-5)
	Split        string // fullbody/upper_lower/push_pull_legs
}

// GetDefaultSplit возвращает оптимальный сплит для количества дней
func GetDefaultSplit(daysPerWeek int) string {
	switch daysPerWeek {
	case 2, 3:
		return "fullbody"
	case 4:
		return "upper_lower"
	case 5, 6:
		return "push_pull_legs"
	default:
		return "fullbody"
	}
}

// Generate генерирует программу гипертрофии
func (g *HypertrophyGenerator) Generate(config HypertrophyConfig) (*models.GeneratedProgram, error) {
	program := &models.GeneratedProgram{
		ClientID:      g.client.ID,
		ClientName:    g.client.Name,
		Goal:          models.GoalHypertrophy,
		Periodization: models.PeriodBlock,
		TotalWeeks:    config.TotalWeeks,
		DaysPerWeek:   config.DaysPerWeek,
	}

	// Определяем фазы
	program.Phases = g.definePhasesHypertrophy(config.TotalWeeks)

	// Определяем сплит
	dayTypes := g.getDayTypes(config.Split, config.DaysPerWeek)

	// Генерируем недели
	for weekNum := 1; weekNum <= config.TotalWeeks; weekNum++ {
		week := g.generateWeekHypertrophy(weekNum, config.TotalWeeks, dayTypes)
		program.Weeks = append(program.Weeks, week)
	}

	// Считаем статистику
	program.Statistics = g.calculateStats(program)

	// Фиксируем замены
	program.Substitutions = g.getSubstitutions()

	return program, nil
}

// definePhasesHypertrophy определяет фазы для гипертрофии
func (g *HypertrophyGenerator) definePhasesHypertrophy(totalWeeks int) []models.ProgramPhase {
	// Блочная периодизация для гипертрофии:
	// 1. Накопление (MEV → MRV): 60% времени
	// 2. Интенсификация: 30% времени
	// 3. Разгрузка: 1 неделя

	accumWeeks := int(float64(totalWeeks) * 0.6)
	intensWeeks := totalWeeks - accumWeeks - 1
	if intensWeeks < 1 {
		intensWeeks = 1
	}

	return []models.ProgramPhase{
		{
			Name:         "Накопление",
			WeekStart:    1,
			WeekEnd:      accumWeeks,
			Focus:        "Объём, рабочая гипертрофия, MEV→MRV",
			IntensityMin: 65,
			IntensityMax: 75,
			VolumeLevel:  "high",
		},
		{
			Name:         "Интенсификация",
			WeekStart:    accumWeeks + 1,
			WeekEnd:      accumWeeks + intensWeeks,
			Focus:        "Конверсия объёма в силу",
			IntensityMin: 75,
			IntensityMax: 82,
			VolumeLevel:  "medium",
		},
		{
			Name:         "Разгрузка",
			WeekStart:    totalWeeks,
			WeekEnd:      totalWeeks,
			Focus:        "Восстановление перед новым циклом",
			IntensityMin: 50,
			IntensityMax: 60,
			VolumeLevel:  "low",
		},
	}
}

// getDayTypes возвращает типы тренировочных дней
func (g *HypertrophyGenerator) getDayTypes(split string, daysPerWeek int) []string {
	switch split {
	case "upper_lower":
		if daysPerWeek == 4 {
			return []string{"upper", "lower", "upper", "lower"}
		}
		return []string{"upper", "lower", "upper"}

	case "push_pull_legs":
		if daysPerWeek >= 6 {
			return []string{"push", "pull", "legs", "push", "pull", "legs"}
		}
		return []string{"push", "pull", "legs"}

	default: // fullbody
		days := make([]string, daysPerWeek)
		for i := range days {
			days[i] = "fullbody"
		}
		return days
	}
}

// generateWeekHypertrophy генерирует неделю для гипертрофии
func (g *HypertrophyGenerator) generateWeekHypertrophy(weekNum, totalWeeks int, dayTypes []string) models.GeneratedWeek {
	phase := g.getPhaseForWeek(weekNum, totalWeeks)
	isDeload := weekNum == totalWeeks

	// Волновой множитель для интенсивности
	waveMultiplier := g.getWaveMultiplier(weekNum)
	waveWeekName := g.getWaveWeekName(weekNum)

	// Базовые параметры недели
	week := models.GeneratedWeek{
		WeekNum:   weekNum,
		PhaseName: fmt.Sprintf("%s (%s)", phase, waveWeekName),
		IsDeload:  isDeload,
	}

	// Расчёт интенсивности и объёма с учётом волновой периодизации
	if isDeload {
		week.IntensityPercent = 55
		week.VolumePercent = 50
		week.RPETarget = 5
	} else {
		baseIntensity := g.getIntensityForWeek(weekNum, phase)
		week.IntensityPercent = baseIntensity * waveMultiplier
		week.VolumePercent = g.getVolumeForWeek(weekNum, phase)
		week.RPETarget = g.getRPEForPhase(phase)

		// Корректируем RPE для лёгкой недели
		if waveMultiplier < 1.0 {
			week.RPETarget -= 1.5
		} else if waveMultiplier > 1.05 {
			week.RPETarget += 0.5
		}
	}

	// Сохраняем волновой множитель для использования при расчёте весов
	week.WaveMultiplier = waveMultiplier

	// Генерируем дни
	for dayNum, dayType := range dayTypes {
		day := g.generateDayHypertrophy(dayNum+1, dayType, weekNum, totalWeeks, isDeload)
		week.Days = append(week.Days, day)
	}

	return week
}

// generateDayHypertrophy генерирует тренировочный день
func (g *HypertrophyGenerator) generateDayHypertrophy(dayNum int, dayType string, weekNum, totalWeeks int, isDeload bool) models.GeneratedDay {
	day := models.GeneratedDay{
		DayNum: dayNum,
		Name:   fmt.Sprintf("День %d — %s", dayNum, getDayTypeName(dayType)),
		Type:   dayType,
	}

	// Получаем упражнения для дня
	difficulty := g.getDifficultyLevel()
	exercises := g.selector.SelectExercisesForDay(
		dayType,
		g.client.AvailableEquip,
		g.client.Constraints,
		difficulty,
	)

	// Создаём прогрессии
	weightProg := make(map[string]*progression.WeightProgression)
	for movement, onePM := range g.client.OnePM {
		weightProg[movement] = progression.NewWeightProgression(onePM)
	}

	trxProg := progression.NewTRXProgression(g.client.Weight, string(g.client.Experience))
	kbProg := progression.NewKettlebellProgression(g.client.AvailableKBWeights, g.client.Gender, string(g.client.Experience))

	// Конвертируем в GeneratedExercise
	phase := g.getPhaseForWeek(weekNum, totalWeeks)
	for orderNum, result := range exercises {
		ex := g.convertToGeneratedExercise(
			result,
			orderNum+1,
			weekNum,
			totalWeeks,
			phase,
			isDeload,
			weightProg,
			trxProg,
			kbProg,
		)
		day.Exercises = append(day.Exercises, ex)

		// Собираем мышечные группы
		for _, m := range result.Exercise.PrimaryMuscles {
			day.MuscleGroups = append(day.MuscleGroups, m)
		}
	}

	// Добавляем обязательные упражнения в конец: гиперэкстензия и пресс
	day.Exercises = append(day.Exercises, g.getFinishingExercises(len(day.Exercises)+1, isDeload)...)

	// Оценка длительности: ~3 мин на подход
	totalSets := 0
	for _, ex := range day.Exercises {
		totalSets += ex.Sets
	}
	day.EstimatedDuration = totalSets * 3

	return day
}

// getFinishingExercises возвращает обязательные упражнения в конце тренировки
// Адаптировано под пол: женщинам больше ягодичной работы
func (g *HypertrophyGenerator) getFinishingExercises(startOrder int, isDeload bool) []models.GeneratedExercise {
	exercises := []models.GeneratedExercise{}
	isFemale := g.client.Gender == "female" || g.client.Gender == "женский" || g.client.Gender == "ж"

	sets := 3
	hyperReps := "15"
	absReps := "15-20"
	gluteReps := "15-20"
	if isDeload {
		sets = 2
		hyperReps = "12"
		absReps = "12"
		gluteReps = "12"
	}

	if isFemale {
		// Женский вариант: ягодичный мост + пресс
		exercises = append(exercises, models.GeneratedExercise{
			OrderNum:     startOrder,
			ExerciseID:   "glute_bridge",
			ExerciseName: "Ягодичный мост",
			MuscleGroup:  models.MuscleGlutes,
			MovementType: models.MovementHinge,
			Sets:         sets,
			Reps:         gluteReps,
			RestSeconds:  60,
			Notes:        "Задержка в верхней точке 2 сек",
		})
	} else {
		// Мужской вариант: гиперэкстензия
		exercises = append(exercises, models.GeneratedExercise{
			OrderNum:     startOrder,
			ExerciseID:   "hyperextension",
			ExerciseName: "Гиперэкстензия",
			MuscleGroup:  models.MuscleBack,
			MovementType: models.MovementHinge,
			Sets:         sets,
			Reps:         hyperReps,
			RestSeconds:  60,
			Notes:        "Контролируемое движение, без рывков",
		})
	}

	// Пресс (скручивания) - для всех
	exercises = append(exercises, models.GeneratedExercise{
		OrderNum:     startOrder + 1,
		ExerciseID:   "crunches",
		ExerciseName: "Скручивания",
		MuscleGroup:  models.MuscleCore,
		MovementType: models.MovementCore,
		Sets:         sets,
		Reps:         absReps,
		RestSeconds:  45,
		Notes:        "",
	})

	return exercises
}

// convertToGeneratedExercise конвертирует результат подбора в упражнение программы
func (g *HypertrophyGenerator) convertToGeneratedExercise(
	result SelectionResult,
	orderNum, weekNum, totalWeeks int,
	phase string,
	isDeload bool,
	weightProg map[string]*progression.WeightProgression,
	trxProg *progression.TRXProgression,
	kbProg *progression.KettlebellProgression,
) models.GeneratedExercise {

	ex := result.Exercise

	genEx := models.GeneratedExercise{
		OrderNum:     orderNum,
		ExerciseID:   ex.ID,
		ExerciseName: ex.NameRu,
		MovementType: ex.MovementType,
	}

	if len(ex.PrimaryMuscles) > 0 {
		genEx.MuscleGroup = ex.PrimaryMuscles[0]
	}

	// Определяем параметры в зависимости от типа оборудования
	switch {
	case containsEquipment(ex.Equipment, models.EquipmentTRX):
		// TRX упражнение
		params := trxProg.GetHypertrophyParams(ex.TRXMinLevel, ex.TRXMaxLevel, weekNum, totalWeeks)
		if isDeload {
			params.Level = maxInt(params.Level-2, ex.TRXMinLevel)
			params.Sets = 2
		}
		genEx.TRXLevel = params.Level
		genEx.Reps = fmt.Sprintf("%d", params.Reps)
		genEx.Sets = params.Sets
		genEx.Tempo = params.Tempo
		genEx.RestSeconds = params.RestSeconds

	case containsEquipment(ex.Equipment, models.EquipmentKettlebell):
		// Гиревое упражнение
		var params progression.KBParams
		if ex.KettlebellType == models.KBTypeBallistic {
			params = kbProg.GetBallisticParams(weekNum, totalWeeks)
		} else {
			params = kbProg.GetGrindParams(weekNum, totalWeeks, phase)
		}
		if isDeload {
			params.Sets = 2
			params.Reps = 8
		}
		genEx.Weight = params.Weight
		genEx.Reps = fmt.Sprintf("%d", params.Reps)
		genEx.Sets = params.Sets
		genEx.RestSeconds = params.RestSeconds

	default:
		// Штанга/гантели/тренажёр
		// Пытаемся найти 1ПМ для движения
		var wp *progression.WeightProgression
		for movement, prog := range weightProg {
			if matchesMovement(ex, movement) {
				wp = prog
				break
			}
		}

		if wp != nil {
			params := wp.GetHypertrophyParams(weekNum, phase)
			if isDeload {
				params.Sets = 2
				params.Reps = 8
				params.Intensity = params.Intensity * 0.8
				params.Weight = wp.CalculateWeight(params.Intensity)
			}
			genEx.Weight = params.Weight
			genEx.WeightPercent = params.Intensity
			genEx.Reps = fmt.Sprintf("%d", params.Reps)
			genEx.Sets = params.Sets
			genEx.RestSeconds = params.RestSeconds
			genEx.RPE = params.RPE
		} else {
			// Нет 1ПМ — даём диапазон повторений
			genEx.Reps = g.getDefaultRepsRange(phase, isDeload)
			genEx.Sets = g.getDefaultSets(phase, isDeload)
			genEx.RestSeconds = 90
			genEx.RPE = 7.5
		}
	}

	// Добавляем альтернативу
	if result.Alternative != nil {
		alt := models.GeneratedExercise{
			ExerciseID:   result.Alternative.ID,
			ExerciseName: result.Alternative.NameRu,
		}
		genEx.Alternative = &alt
	}

	return genEx
}

// === Вспомогательные методы ===

func (g *HypertrophyGenerator) getPhaseForWeek(weekNum, totalWeeks int) string {
	accumWeeks := int(float64(totalWeeks) * 0.6)

	if weekNum == totalWeeks {
		return "deload"
	}
	if weekNum <= accumWeeks {
		return "accumulation"
	}
	return "intensification"
}

// getWaveMultiplier возвращает коэффициент волновой периодизации для недели
// Волновая периодизация (4-недельный цикл):
// Неделя 1: Средняя (100%)
// Неделя 2: Тяжёлая (105%)
// Неделя 3: Лёгкая (90%) - разгрузка
// Неделя 4: Пиковая (107-110%)
func (g *HypertrophyGenerator) getWaveMultiplier(weekNum int) float64 {
	// Определяем позицию в 4-недельном цикле
	cyclePos := ((weekNum - 1) % 4) + 1

	switch cyclePos {
	case 1:
		return 1.0 // Средняя неделя - 100%
	case 2:
		return 1.05 // Тяжёлая неделя - 105%
	case 3:
		return 0.90 // Лёгкая неделя - 90%
	case 4:
		return 1.10 // Пиковая неделя - 110%
	default:
		return 1.0
	}
}

// getWaveWeekName возвращает название типа недели в волновой периодизации
func (g *HypertrophyGenerator) getWaveWeekName(weekNum int) string {
	cyclePos := ((weekNum - 1) % 4) + 1

	switch cyclePos {
	case 1:
		return "Средняя"
	case 2:
		return "Тяжёлая"
	case 3:
		return "Лёгкая"
	case 4:
		return "Пиковая"
	default:
		return ""
	}
}

func (g *HypertrophyGenerator) getIntensityForWeek(weekNum int, phase string) float64 {
	switch phase {
	case "accumulation":
		return 65 + float64(weekNum-1)*2 // 65 → 75
	case "intensification":
		return 75 + float64(weekNum-1)*1.5 // 75 → 82
	default:
		return 70
	}
}

func (g *HypertrophyGenerator) getVolumeForWeek(weekNum int, phase string) float64 {
	switch phase {
	case "accumulation":
		return 80 + float64(weekNum-1)*5 // MEV → MRV
	case "intensification":
		return 100 - float64(weekNum-1)*5 // Снижаем
	default:
		return 80
	}
}

func (g *HypertrophyGenerator) getRPEForPhase(phase string) float64 {
	switch phase {
	case "accumulation":
		return 7.5
	case "intensification":
		return 8.5
	default:
		return 5
	}
}

func (g *HypertrophyGenerator) getDifficultyLevel() models.DifficultyLevel {
	switch g.client.Experience {
	case models.ExpAdvanced:
		return models.DifficultyAdvanced
	case models.ExpIntermediate:
		return models.DifficultyIntermediate
	default:
		return models.DifficultyBeginner
	}
}

func (g *HypertrophyGenerator) getDefaultRepsRange(phase string, isDeload bool) string {
	if isDeload {
		return "8-10"
	}
	switch phase {
	case "accumulation":
		return "10-12"
	case "intensification":
		return "6-8"
	default:
		return "8-10"
	}
}

func (g *HypertrophyGenerator) getDefaultSets(phase string, isDeload bool) int {
	if isDeload {
		return 2
	}
	switch phase {
	case "accumulation":
		return 4
	case "intensification":
		return 3
	default:
		return 3
	}
}

func (g *HypertrophyGenerator) calculateStats(program *models.GeneratedProgram) models.ProgramStats {
	stats := models.ProgramStats{
		SetsPerMuscle: make(map[models.MuscleGroupExt]int),
	}

	for _, week := range program.Weeks {
		for _, day := range week.Days {
			stats.TotalWorkouts++
			for _, ex := range day.Exercises {
				stats.TotalSets += ex.Sets
				if ex.Weight > 0 {
					reps := 10 // default
					fmt.Sscanf(ex.Reps, "%d", &reps)
					stats.TotalVolume += ex.Weight * float64(ex.Sets*reps)
				}
				stats.SetsPerMuscle[ex.MuscleGroup] += ex.Sets
			}
		}
	}

	if stats.TotalWorkouts > 0 {
		stats.AvgWorkoutDur = (stats.TotalSets * 3) / stats.TotalWorkouts
	}

	return stats
}

func (g *HypertrophyGenerator) getSubstitutions() []models.Substitution {
	// TODO: Собирать замены при генерации
	return nil
}

// === Утилиты ===

func getDayTypeName(dayType string) string {
	names := map[string]string{
		"push":     "Push (Грудь/Плечи/Трицепс)",
		"pull":     "Pull (Спина/Бицепс)",
		"legs":     "Legs (Ноги)",
		"upper":    "Upper (Верх)",
		"lower":    "Lower (Низ)",
		"fullbody": "Full Body",
	}
	if name, ok := names[dayType]; ok {
		return name
	}
	return dayType
}

func containsEquipment(equipment []models.EquipmentType, target models.EquipmentType) bool {
	for _, e := range equipment {
		if e == target {
			return true
		}
	}
	return false
}

func matchesMovement(ex models.ExerciseExt, movement string) bool {
	// Простое сопоставление названия движения с упражнением
	switch movement {
	case "squat":
		return ex.MovementType == models.MovementSquat
	case "bench":
		return ex.MovementType == models.MovementPush && containsMuscle(ex.PrimaryMuscles, models.MuscleChest)
	case "deadlift":
		return ex.MovementType == models.MovementHinge && containsMuscle(ex.PrimaryMuscles, models.MuscleBack)
	case "ohp":
		return ex.MovementType == models.MovementPush && containsMuscle(ex.PrimaryMuscles, models.MuscleShoulders)
	}
	return false
}

func containsMuscle(muscles []models.MuscleGroupExt, target models.MuscleGroupExt) bool {
	for _, m := range muscles {
		if m == target {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

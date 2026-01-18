package ai

import (
	"fmt"

	"workbot/internal/models"
)

// ProgramValidatorV3 валидатор программ с полной периодизацией
type ProgramValidatorV3 struct{}

// NewProgramValidatorV3 создаёт новый валидатор
func NewProgramValidatorV3() *ProgramValidatorV3 {
	return &ProgramValidatorV3{}
}

// Validate проверяет программу на корректность
func (v *ProgramValidatorV3) Validate(plan *models.TrainingPlan, req ProgramRequestV3) *ValidationResultV3 {
	result := &ValidationResultV3{
		IsValid:     true,
		Score:       100,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// 1. Проверка количества недель
	if len(plan.Weeks) != req.TotalWeeks {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Неверное количество недель: ожидалось %d, получено %d", req.TotalWeeks, len(plan.Weeks)))
		result.IsValid = false
		result.Score -= 30
	}

	// 2. Проверка количества тренировок в каждой неделе
	for _, week := range plan.Weeks {
		if len(week.Workouts) != req.DaysPerWeek {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Неделя %d: ожидалось %d тренировок, получено %d",
					week.WeekNum, req.DaysPerWeek, len(week.Workouts)))
			result.IsValid = false
			result.Score -= 10
		}
	}

	// 3. Проверка наличия периодизации
	hasDeload := false
	hasPeriod := false
	hasMesocycle := false

	for _, week := range plan.Weeks {
		if week.IsDeload || week.Phase == models.PhaseDeload {
			hasDeload = true
		}
		if week.Period != "" {
			hasPeriod = true
		}
		if week.MesocycleType != "" {
			hasMesocycle = true
		}
	}

	if !hasDeload && req.TotalWeeks >= 4 {
		result.Errors = append(result.Errors,
			"Отсутствует разгрузочная неделя (deload). При программе 4+ недель обязательна минимум 1 разгрузка")
		result.IsValid = false
		result.Score -= 15
	}

	if !hasPeriod {
		result.Warnings = append(result.Warnings,
			"Не указаны периоды (preparatory/competitive/transitional)")
		result.Score -= 5
	}

	if !hasMesocycle {
		result.Warnings = append(result.Warnings,
			"Не указаны типы мезоциклов")
		result.Score -= 5
	}

	// 4. Проверка прогрессии интенсивности
	v.checkProgression(plan, result)

	// 5. Проверка объёма на группы мышц
	v.checkVolumeBalance(plan, req, result)

	// 6. Проверка наличия упражнений
	for wi, week := range plan.Weeks {
		for di, workout := range week.Workouts {
			if len(workout.Exercises) == 0 {
				result.Errors = append(result.Errors,
					fmt.Sprintf("Неделя %d, День %d: нет упражнений", wi+1, di+1))
				result.IsValid = false
				result.Score -= 10
			}

			// Проверка каждого упражнения
			for ei, ex := range workout.Exercises {
				if ex.Sets <= 0 {
					result.Errors = append(result.Errors,
						fmt.Sprintf("Неделя %d, День %d, Упр %d (%s): количество подходов = 0",
							wi+1, di+1, ei+1, ex.ExerciseName))
					result.IsValid = false
					result.Score -= 5
				}
				if ex.Reps == "" {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Неделя %d, День %d, Упр %d (%s): не указаны повторения",
							wi+1, di+1, ei+1, ex.ExerciseName))
					result.Score -= 2
				}
			}
		}
	}

	// 7. Проверка логичности фаз
	v.checkPhaseLogic(plan, result)

	// Ограничиваем минимальный балл
	if result.Score < 0 {
		result.Score = 0
	}

	// Добавляем рекомендации
	v.addSuggestions(plan, req, result)

	return result
}

// checkProgression проверяет прогрессию интенсивности
func (v *ProgramValidatorV3) checkProgression(plan *models.TrainingPlan, result *ValidationResultV3) {
	if len(plan.Weeks) < 2 {
		return
	}

	prevIntensity := plan.Weeks[0].IntensityPercent
	consecutiveDrops := 0

	for i := 1; i < len(plan.Weeks); i++ {
		week := plan.Weeks[i]
		prevWeek := plan.Weeks[i-1]

		// Разгрузка - снижение ожидаемо
		if week.IsDeload || week.Phase == models.PhaseDeload {
			consecutiveDrops = 0
			prevIntensity = week.IntensityPercent
			continue
		}

		// После разгрузки - может быть ниже
		if prevWeek.IsDeload || prevWeek.Phase == models.PhaseDeload {
			prevIntensity = week.IntensityPercent
			continue
		}

		// Обычная неделя - проверяем прогрессию
		if week.IntensityPercent < prevIntensity {
			consecutiveDrops++
			if consecutiveDrops >= 2 {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Неделя %d: интенсивность снижается без разгрузки (%.0f%% -> %.0f%%)",
						week.WeekNum, prevIntensity, week.IntensityPercent))
				result.Score -= 3
			}
		} else {
			consecutiveDrops = 0
		}

		prevIntensity = week.IntensityPercent
	}
}

// checkVolumeBalance проверяет баланс объёма на группы мышц
func (v *ProgramValidatorV3) checkVolumeBalance(plan *models.TrainingPlan, req ProgramRequestV3, result *ValidationResultV3) {
	// Считаем средний недельный объём на группу мышц
	muscleVolume := make(map[string]int)

	for _, week := range plan.Weeks {
		if week.IsDeload {
			continue
		}
		for _, workout := range week.Workouts {
			for _, ex := range workout.Exercises {
				group := ex.MuscleGroup
				if group == "" {
					// Пытаемся определить по названию
					group = guessMusscleGroup(ex.ExerciseName)
				}
				if group != "" {
					muscleVolume[group] += ex.Sets
				}
			}
		}
	}

	// Нормализуем на количество недель (без разгрузок)
	workingWeeks := 0
	for _, week := range plan.Weeks {
		if !week.IsDeload {
			workingWeeks++
		}
	}
	if workingWeeks == 0 {
		workingWeeks = 1
	}

	// Определяем границы по опыту
	minSets, maxSets := getVolumeRangeByExperience(req.Experience)

	for group, totalSets := range muscleVolume {
		avgWeekly := totalSets / workingWeeks

		if avgWeekly < minSets {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Группа '%s': низкий объём (%d подх./нед., минимум %d)",
					group, avgWeekly, minSets))
			result.Score -= 3
		}

		if avgWeekly > maxSets {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Группа '%s': возможно избыточный объём (%d подх./нед., максимум %d)",
					group, avgWeekly, maxSets))
			result.Score -= 2
		}
	}
}

// checkPhaseLogic проверяет логичность последовательности фаз
func (v *ProgramValidatorV3) checkPhaseLogic(plan *models.TrainingPlan, result *ValidationResultV3) {
	// Проверяем что фазы идут в логичном порядке
	phaseOrder := map[models.PlanPhase]int{
		models.PhaseHypertrophy:   1,
		models.PhaseAccumulation:  1,
		models.PhaseStrength:      2,
		models.PhaseTransmutation: 2,
		models.PhasePower:         3,
		models.PhaseRealization:   3,
		models.PhasePeaking:       4,
		models.PhaseDeload:        0, // может быть где угодно
	}

	lastOrder := 0
	for _, week := range plan.Weeks {
		order, exists := phaseOrder[week.Phase]
		if !exists {
			continue
		}

		// Разгрузка может быть где угодно
		if week.Phase == models.PhaseDeload {
			continue
		}

		// Резкий скачок назад (не постепенный)
		if order < lastOrder-1 && lastOrder > 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Неделя %d: резкий переход с более тяжёлой фазы назад (может быть умышленно)",
					week.WeekNum))
		}

		lastOrder = order
	}
}

// addSuggestions добавляет рекомендации по улучшению
func (v *ProgramValidatorV3) addSuggestions(plan *models.TrainingPlan, req ProgramRequestV3, result *ValidationResultV3) {
	// Проверка на использование 1ПМ
	if len(req.OnePMData) > 0 {
		weightPercentUsed := false
		for _, week := range plan.Weeks {
			for _, workout := range week.Workouts {
				for _, ex := range workout.Exercises {
					if ex.WeightPercent > 0 {
						weightPercentUsed = true
						break
					}
				}
			}
		}
		if !weightPercentUsed {
			result.Suggestions = append(result.Suggestions,
				"У клиента есть данные 1ПМ, но weight_percent не используется. Рекомендуется указать % от 1ПМ для базовых упражнений")
		}
	}

	// Проверка RPE
	rpeUsed := false
	for _, week := range plan.Weeks {
		for _, workout := range week.Workouts {
			for _, ex := range workout.Exercises {
				if ex.RPE > 0 {
					rpeUsed = true
					break
				}
			}
		}
	}
	if !rpeUsed {
		result.Suggestions = append(result.Suggestions,
			"Рекомендуется указать целевой RPE для упражнений для авторегуляции нагрузки")
	}

	// Проверка темпа
	tempoUsed := false
	for _, week := range plan.Weeks {
		for _, workout := range week.Workouts {
			for _, ex := range workout.Exercises {
				if ex.Tempo != "" {
					tempoUsed = true
					break
				}
			}
		}
	}
	if !tempoUsed && (req.Goal == "hypertrophy" || req.Experience == "beginner") {
		result.Suggestions = append(result.Suggestions,
			"Для гипертрофии/новичков рекомендуется указать темп выполнения (tempo)")
	}
}

// Вспомогательные функции

func guessMusscleGroup(exerciseName string) string {
	keywords := map[string]string{
		"жим лёжа":      "грудь",
		"жим лежа":      "грудь",
		"жим гантелей":  "грудь",
		"разводка":      "грудь",
		"отжимания":     "грудь",
		"брусья":        "грудь",
		"присед":        "ноги",
		"приседания":    "ноги",
		"выпады":        "ноги",
		"жим ногами":    "ноги",
		"разгибания ног":"ноги",
		"сгибания ног":  "ноги",
		"становая":      "спина",
		"тяга":          "спина",
		"подтягивания":  "спина",
		"жим стоя":      "плечи",
		"жим сидя":      "плечи",
		"махи":          "плечи",
		"бицепс":        "руки",
		"трицепс":       "руки",
		"французский":   "руки",
		"планка":        "кор",
		"скручивания":   "кор",
	}

	lowerName := exerciseName
	for keyword, group := range keywords {
		if contains(lowerName, keyword) {
			return group
		}
	}
	return ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getVolumeRangeByExperience(experience string) (min, max int) {
	switch experience {
	case "beginner":
		return 8, 16
	case "intermediate":
		return 10, 22
	case "advanced":
		return 12, 28
	default:
		return 10, 20
	}
}

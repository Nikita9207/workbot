package ai

import (
	"fmt"
	"strings"

	"workbot/internal/models"
)

// ValidationResult результат проверки программы
type ValidationResult struct {
	IsValid     bool     `json:"is_valid"`      // true = программа готова к выдаче клиенту
	Errors      []string `json:"errors"`        // критические ошибки (требуют перегенерации)
	Warnings    []string `json:"warnings"`      // предупреждения (можно исправить)
	Suggestions []string `json:"suggestions"`   // рекомендации для AI при перегенерации
}

// ProgramValidator валидатор программ тренировок
type ProgramValidator struct {
	// Ограничения по объёму (подходов в неделю на группу мышц)
	volumeLimits map[string]struct{ min, max int }
}

// NewProgramValidator создаёт новый валидатор
func NewProgramValidator() *ProgramValidator {
	return &ProgramValidator{
		volumeLimits: map[string]struct{ min, max int }{
			"beginner":     {6, 16},
			"intermediate": {10, 22},
			"advanced":     {12, 28},
		},
	}
}

// ValidateProgram проверяет программу на корректность
// Возвращает IsValid=true только если программа полностью готова к выдаче клиенту
func (v *ProgramValidator) ValidateProgram(program *models.Program, experience string) *ValidationResult {
	result := &ValidationResult{
		IsValid:     true,
		Errors:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// 1. Проверка наличия тренировок
	if len(program.Workouts) == 0 {
		result.Errors = append(result.Errors, "Программа не содержит тренировок")
		result.Suggestions = append(result.Suggestions, "Сгенерируй программу с тренировками для каждого дня")
	}

	// 2. Проверка объёма по группам мышц
	v.validateVolume(program, experience, result)

	// 3. Проверка прогрессии
	v.validateProgression(program, result)

	// 4. Проверка наличия deload
	v.validateDeload(program, result)

	// 5. Проверка параметров упражнений
	v.validateExerciseParams(program, result)

	// 6. Проверка баланса (push/pull)
	v.validateBalance(program, result)

	// 7. Проверка отдыха между тренировками
	v.validateRecovery(program, result)

	// Программа валидна ТОЛЬКО если нет ошибок
	// Предупреждения допустимы, но лучше их тоже исправить
	if len(result.Errors) > 0 {
		result.IsValid = false
	}

	return result
}

// validateVolume проверяет объём тренировок
func (v *ProgramValidator) validateVolume(program *models.Program, experience string, result *ValidationResult) {
	muscleVolume := make(map[string]int)

	// Подсчёт объёма по группам мышц за неделю
	weekWorkouts := make(map[int][]models.Workout)
	for _, w := range program.Workouts {
		weekWorkouts[w.WeekNum] = append(weekWorkouts[w.WeekNum], w)
	}

	for weekNum, workouts := range weekWorkouts {
		// Сброс счётчика для каждой недели
		for k := range muscleVolume {
			delete(muscleVolume, k)
		}

		for _, workout := range workouts {
			for _, ex := range workout.Exercises {
				muscle := guessMuscleGroup(ex.ExerciseName)
				muscleVolume[muscle] += ex.Sets
			}
		}

		limits, ok := v.volumeLimits[experience]
		if !ok {
			limits = v.volumeLimits["intermediate"]
		}

		for muscle, sets := range muscleVolume {
			if muscle == "unknown" {
				continue
			}
			if sets < limits.min {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Неделя %d: мало объёма на %s (%d подходов, рекомендуется %d+)",
						weekNum, muscle, sets, limits.min))
			}
			if sets > limits.max {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Неделя %d: много объёма на %s (%d подходов, максимум %d)",
						weekNum, muscle, sets, limits.max))
			}
		}
	}
}

// validateProgression проверяет прогрессию нагрузки
func (v *ProgramValidator) validateProgression(program *models.Program, result *ValidationResult) {
	if program.TotalWeeks < 2 {
		return
	}

	// Группируем упражнения по неделям
	weekExercises := make(map[int]map[string]float64) // неделя -> упражнение -> вес/интенсивность

	for _, w := range program.Workouts {
		if weekExercises[w.WeekNum] == nil {
			weekExercises[w.WeekNum] = make(map[string]float64)
		}
		for _, ex := range w.Exercises {
			intensity := ex.Weight
			if ex.WeightPercent > 0 {
				intensity = ex.WeightPercent
			}
			// Берём максимальный вес для упражнения за неделю
			if intensity > weekExercises[w.WeekNum][ex.ExerciseName] {
				weekExercises[w.WeekNum][ex.ExerciseName] = intensity
			}
		}
	}

	// Проверяем прогрессию между неделями (исключая deload)
	hasProgression := false
	for week := 2; week <= program.TotalWeeks; week++ {
		prevWeek := week - 1
		if weekExercises[week] == nil || weekExercises[prevWeek] == nil {
			continue
		}

		for ex, currentIntensity := range weekExercises[week] {
			prevIntensity, ok := weekExercises[prevWeek][ex]
			if !ok {
				continue
			}

			// Если текущая неделя не deload и интенсивность растёт
			if currentIntensity > prevIntensity {
				hasProgression = true
			}
		}
	}

	if !hasProgression && program.TotalWeeks >= 4 {
		result.Warnings = append(result.Warnings,
			"Не обнаружена прогрессия нагрузки между неделями")
		result.Suggestions = append(result.Suggestions,
			"Добавьте постепенное увеличение весов/интенсивности от недели к неделе")
	}
}

// validateDeload проверяет наличие разгрузочных недель
func (v *ProgramValidator) validateDeload(program *models.Program, result *ValidationResult) {
	if program.TotalWeeks < 4 {
		return
	}

	// Проверяем наличие deload каждые 4-6 недель
	hasDeload := false
	deloadWeeks := []int{}

	// Ищем недели с пониженным объёмом или интенсивностью
	weekVolume := make(map[int]int)
	for _, w := range program.Workouts {
		for _, ex := range w.Exercises {
			weekVolume[w.WeekNum] += ex.Sets
		}
	}

	// Находим среднюю нагрузку и ищем недели с -30% и более
	totalVolume := 0
	for _, vol := range weekVolume {
		totalVolume += vol
	}
	avgVolume := totalVolume / len(weekVolume)

	for week, vol := range weekVolume {
		if float64(vol) < float64(avgVolume)*0.7 {
			hasDeload = true
			deloadWeeks = append(deloadWeeks, week)
		}
	}

	if !hasDeload {
		result.Warnings = append(result.Warnings,
			"Не обнаружены разгрузочные недели (deload)")
		result.Suggestions = append(result.Suggestions,
			"Рекомендуется добавить deload каждые 4-6 недель (снижение объёма на 40-50%)")
	} else {
		// Проверяем интервал между deload
		for i := 1; i < len(deloadWeeks); i++ {
			gap := deloadWeeks[i] - deloadWeeks[i-1]
			if gap > 6 {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Большой интервал между разгрузками: %d недель (между неделями %d и %d)",
						gap, deloadWeeks[i-1], deloadWeeks[i]))
			}
		}
	}
}

// validateExerciseParams проверяет параметры упражнений
func (v *ProgramValidator) validateExerciseParams(program *models.Program, result *ValidationResult) {
	for _, w := range program.Workouts {
		for _, ex := range w.Exercises {
			// Проверка подходов
			if ex.Sets < 1 || ex.Sets > 10 {
				result.Errors = append(result.Errors,
					fmt.Sprintf("Некорректное количество подходов: %s - %d (должно быть 1-10)",
						ex.ExerciseName, ex.Sets))
			}

			// Проверка повторов (парсим диапазон)
			reps := parseReps(ex.Reps)
			if reps < 1 || reps > 30 {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Необычное количество повторов: %s - %s",
						ex.ExerciseName, ex.Reps))
			}

			// Проверка отдыха
			if ex.RestSeconds > 0 && (ex.RestSeconds < 15 || ex.RestSeconds > 600) {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Необычное время отдыха: %s - %d сек",
						ex.ExerciseName, ex.RestSeconds))
			}

			// Проверка RPE
			if ex.RPE > 0 && (ex.RPE < 5 || ex.RPE > 10) {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Некорректный RPE: %s - %.1f (должно быть 5-10)",
						ex.ExerciseName, ex.RPE))
			}

			// Проверка процента от 1ПМ
			if ex.WeightPercent > 0 && (ex.WeightPercent < 30 || ex.WeightPercent > 105) {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Необычный процент от 1ПМ: %s - %.0f%%",
						ex.ExerciseName, ex.WeightPercent))
			}
		}
	}
}

// validateBalance проверяет баланс push/pull
func (v *ProgramValidator) validateBalance(program *models.Program, result *ValidationResult) {
	pushVolume := 0
	pullVolume := 0

	for _, w := range program.Workouts {
		for _, ex := range w.Exercises {
			movement := guessMovementType(ex.ExerciseName)
			if movement == "push" {
				pushVolume += ex.Sets
			} else if movement == "pull" {
				pullVolume += ex.Sets
			}
		}
	}

	if pushVolume > 0 && pullVolume > 0 {
		ratio := float64(pushVolume) / float64(pullVolume)
		if ratio > 1.5 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Дисбаланс: слишком много жимовых движений (push:pull = %.1f:1)", ratio))
			result.Suggestions = append(result.Suggestions,
				"Добавьте больше тяговых упражнений для баланса (тяги, подтягивания)")
		} else if ratio < 0.7 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Дисбаланс: слишком много тяговых движений (push:pull = 1:%.1f)", 1/ratio))
		}
	}
}

// validateRecovery проверяет время восстановления
func (v *ProgramValidator) validateRecovery(program *models.Program, result *ValidationResult) {
	// Группируем тренировки по неделям и дням
	type workoutKey struct {
		week int
		day  int
	}
	workoutMuscles := make(map[workoutKey][]string)

	for _, w := range program.Workouts {
		key := workoutKey{w.WeekNum, w.DayNum}
		for _, ex := range w.Exercises {
			muscle := guessMuscleGroup(ex.ExerciseName)
			if muscle != "unknown" {
				workoutMuscles[key] = append(workoutMuscles[key], muscle)
			}
		}
	}

	// Проверяем, что одна группа мышц не тренируется 2 дня подряд
	for key, muscles := range workoutMuscles {
		nextDay := workoutKey{key.week, key.day + 1}
		if key.day == 7 {
			nextDay = workoutKey{key.week + 1, 1}
		}

		if nextMuscles, ok := workoutMuscles[nextDay]; ok {
			for _, m1 := range muscles {
				for _, m2 := range nextMuscles {
					if m1 == m2 && isLargeMuscle(m1) {
						result.Warnings = append(result.Warnings,
							fmt.Sprintf("Недостаточное восстановление: %s тренируется 2 дня подряд (неделя %d, дни %d и %d)",
								m1, key.week, key.day, nextDay.day))
					}
				}
			}
		}
	}
}

// Вспомогательные функции

func guessMuscleGroup(exerciseName string) string {
	name := strings.ToLower(exerciseName)

	if containsAny(name, "жим лёжа", "жим лежа", "жим гантелей лёжа", "разводка", "кроссовер", "отжимания", "грудь", "chest") {
		return "грудь"
	}
	if containsAny(name, "тяга", "подтягивания", "спина", "тяга штанги", "блок", "pulldown", "row", "back") {
		return "спина"
	}
	if containsAny(name, "жим стоя", "армейский", "махи", "дельт", "плечи", "shoulder", "press") && !containsAny(name, "лёжа", "лежа") {
		return "плечи"
	}
	if containsAny(name, "присед", "жим ногами", "выпады", "разгибание ног", "квадрицепс", "squat", "leg press", "lunge") {
		return "квадрицепс"
	}
	if containsAny(name, "румынская", "сгибание ног", "бицепс бедра", "мёртвая тяга", "deadlift", "hamstring") {
		return "бицепс_бедра"
	}
	if containsAny(name, "бицепс", "сгибание рук", "молоток", "bicep", "curl") && !containsAny(name, "бедра") {
		return "бицепс"
	}
	if containsAny(name, "трицепс", "разгибание рук", "французский", "tricep", "pushdown", "extension") && !containsAny(name, "ног") {
		return "трицепс"
	}
	if containsAny(name, "икры", "голень", "подъём на носки", "calf") {
		return "икры"
	}
	if containsAny(name, "пресс", "скручивания", "планка", "abs", "core") {
		return "пресс"
	}
	if containsAny(name, "ягодиц", "glute", "hip thrust") {
		return "ягодицы"
	}

	return "unknown"
}

func guessMovementType(exerciseName string) string {
	name := strings.ToLower(exerciseName)

	// Push движения
	if containsAny(name, "жим", "отжимания", "разгибание", "французский", "разводка", "press", "pushdown", "fly") {
		return "push"
	}

	// Pull движения
	if containsAny(name, "тяга", "подтягивания", "сгибание", "curl", "row", "pulldown", "pull") {
		return "pull"
	}

	// Ноги
	if containsAny(name, "присед", "выпады", "squat", "lunge", "leg") {
		return "legs"
	}

	return "other"
}

func isLargeMuscle(muscle string) bool {
	largeMuscles := []string{"грудь", "спина", "квадрицепс", "бицепс_бедра", "ягодицы"}
	for _, m := range largeMuscles {
		if muscle == m {
			return true
		}
	}
	return false
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func parseReps(reps string) int {
	// Парсим "8-10" -> 9, "12" -> 12
	reps = strings.TrimSpace(reps)
	if strings.Contains(reps, "-") {
		parts := strings.Split(reps, "-")
		if len(parts) == 2 {
			var min, max int
			fmt.Sscanf(parts[0], "%d", &min)
			fmt.Sscanf(parts[1], "%d", &max)
			return (min + max) / 2
		}
	}
	var r int
	fmt.Sscanf(reps, "%d", &r)
	return r
}

// FormatValidationResult форматирует результат валидации для вывода
func FormatValidationResult(result *ValidationResult) string {
	var sb strings.Builder

	if result.IsValid {
		sb.WriteString("✅ Программа готова\n\n")
	} else {
		sb.WriteString("❌ Программа требует доработки\n\n")
	}

	if len(result.Errors) > 0 {
		sb.WriteString("🚫 ОШИБКИ (нужно исправить):\n")
		for _, err := range result.Errors {
			sb.WriteString(fmt.Sprintf("  • %s\n", err))
		}
		sb.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		sb.WriteString("⚠️ ЗАМЕЧАНИЯ:\n")
		for _, warn := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  • %s\n", warn))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetSuggestionsForRetry возвращает подсказки для AI при перегенерации
func GetSuggestionsForRetry(result *ValidationResult) string {
	if len(result.Errors) == 0 && len(result.Suggestions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("ИСПРАВЬ СЛЕДУЮЩИЕ ПРОБЛЕМЫ:\n")

	for _, err := range result.Errors {
		sb.WriteString(fmt.Sprintf("- %s\n", err))
	}

	for _, sug := range result.Suggestions {
		sb.WriteString(fmt.Sprintf("- %s\n", sug))
	}

	return sb.String()
}

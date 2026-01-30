package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"workbot/internal/models"
)

// ExerciseSelector - подбор упражнений с учётом ограничений
type ExerciseSelector struct {
	exercises        []models.ExerciseExt
	contraindications map[string][]models.Contraindication
	alternatives     map[string][]models.ExerciseAlternative
}

// NewExerciseSelector создаёт новый селектор упражнений
func NewExerciseSelector(dataDir string) (*ExerciseSelector, error) {
	selector := &ExerciseSelector{
		contraindications: make(map[string][]models.Contraindication),
		alternatives:     make(map[string][]models.ExerciseAlternative),
	}

	// Загружаем все файлы упражнений
	files := []string{
		"barbell_dumbbell.json",
		"trx.json",
		"kettlebell.json",
		"cardio_metabolic.json",
		"core.json",
	}

	for _, file := range files {
		path := filepath.Join(dataDir, "exercises", file)
		if err := selector.loadExerciseFile(path); err != nil {
			// Файл может не существовать - пропускаем
			continue
		}
	}

	return selector, nil
}

// loadExerciseFile загружает файл с упражнениями
func (s *ExerciseSelector) loadExerciseFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var fileData struct {
		Exercises         []models.ExerciseExt          `json:"exercises"`
		Contraindications []models.Contraindication     `json:"contraindications"`
		Alternatives      []models.ExerciseAlternative  `json:"alternatives"`
	}

	if err := json.Unmarshal(data, &fileData); err != nil {
		return err
	}

	s.exercises = append(s.exercises, fileData.Exercises...)

	for _, c := range fileData.Contraindications {
		s.contraindications[c.ExerciseID] = append(s.contraindications[c.ExerciseID], c)
	}

	for _, a := range fileData.Alternatives {
		s.alternatives[a.ExerciseID] = append(s.alternatives[a.ExerciseID], a)
	}

	return nil
}

// SelectionCriteria - критерии подбора
type SelectionCriteria struct {
	MovementType     models.MovementType     // Тип движения (push/pull/squat/hinge)
	PrimaryMuscle    models.MuscleGroupExt   // Целевая мышечная группа
	Equipment        []models.EquipmentType  // Доступное оборудование
	Constraints      []models.ClientConstraint // Ограничения клиента
	MaxDifficulty    models.DifficultyLevel  // Максимальная сложность
	RequireCompound  bool                    // Требуется базовое упражнение
	ExcludeIDs       []string                // Исключить эти упражнения
}

// SelectionResult - результат подбора
type SelectionResult struct {
	Exercise    models.ExerciseExt
	Alternative *models.ExerciseExt // Альтернатива (если есть)
	Reason      string              // Причина выбора/замены
}

// SelectExercise подбирает упражнение по критериям
func (s *ExerciseSelector) SelectExercise(criteria SelectionCriteria) *SelectionResult {
	candidates := s.filterExercises(criteria)

	if len(candidates) == 0 {
		return nil
	}

	// Сортируем по приоритету
	selected := s.rankAndSelect(candidates, criteria)

	// Ищем альтернативу
	var alternative *models.ExerciseExt
	alts, ok := s.alternatives[selected.ID]
	if ok && len(alts) > 0 {
		// Берём альтернативу с наивысшим приоритетом
		for _, alt := range alts {
			for i := range s.exercises {
				if s.exercises[i].ID == alt.AlternativeID {
					// Проверяем что альтернатива проходит фильтры
					if s.exercisePassesFilters(&s.exercises[i], criteria) {
						alternative = &s.exercises[i]
						break
					}
				}
			}
			if alternative != nil {
				break
			}
		}
	}

	return &SelectionResult{
		Exercise:    selected,
		Alternative: alternative,
	}
}

// SelectExercisesForDay подбирает упражнения для тренировочного дня
func (s *ExerciseSelector) SelectExercisesForDay(dayType string, equipment []models.EquipmentType, constraints []models.ClientConstraint, difficulty models.DifficultyLevel) []SelectionResult {
	var results []SelectionResult
	usedIDs := make([]string, 0)

	// Паттерны движений для разных типов дней
	patterns := getDayPatterns(dayType)

	for _, pattern := range patterns {
		criteria := SelectionCriteria{
			MovementType:    pattern.MovementType,
			PrimaryMuscle:   pattern.PrimaryMuscle,
			Equipment:       equipment,
			Constraints:     constraints,
			MaxDifficulty:   difficulty,
			RequireCompound: pattern.RequireCompound,
			ExcludeIDs:      usedIDs,
		}

		result := s.SelectExercise(criteria)
		if result != nil {
			results = append(results, *result)
			usedIDs = append(usedIDs, result.Exercise.ID)
		}
	}

	return results
}

// filterExercises фильтрует упражнения по критериям
func (s *ExerciseSelector) filterExercises(criteria SelectionCriteria) []models.ExerciseExt {
	var filtered []models.ExerciseExt

	for i := range s.exercises {
		ex := &s.exercises[i]

		if !s.exercisePassesFilters(ex, criteria) {
			continue
		}

		filtered = append(filtered, *ex)
	}

	return filtered
}

// exercisePassesFilters проверяет упражнение по всем фильтрам
func (s *ExerciseSelector) exercisePassesFilters(ex *models.ExerciseExt, criteria SelectionCriteria) bool {
	// Проверка типа движения
	if criteria.MovementType != "" && ex.MovementType != criteria.MovementType {
		return false
	}

	// Проверка мышечной группы
	if criteria.PrimaryMuscle != "" {
		found := false
		for _, m := range ex.PrimaryMuscles {
			if m == criteria.PrimaryMuscle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Проверка оборудования
	if len(criteria.Equipment) > 0 {
		hasEquipment := false
		for _, reqEquip := range ex.Equipment {
			for _, availEquip := range criteria.Equipment {
				if reqEquip == availEquip {
					hasEquipment = true
					break
				}
			}
			if hasEquipment {
				break
			}
		}
		if !hasEquipment {
			return false
		}
	}

	// Проверка сложности
	if criteria.MaxDifficulty > 0 && ex.Difficulty > criteria.MaxDifficulty {
		return false
	}

	// Проверка compound
	if criteria.RequireCompound && !ex.IsCompound {
		return false
	}

	// Проверка исключений
	for _, excludeID := range criteria.ExcludeIDs {
		if ex.ID == excludeID {
			return false
		}
	}

	// Проверка противопоказаний
	if !s.checkContraindications(ex.ID, criteria.Constraints) {
		return false
	}

	return true
}

// checkContraindications проверяет противопоказания
func (s *ExerciseSelector) checkContraindications(exerciseID string, constraints []models.ClientConstraint) bool {
	contras, ok := s.contraindications[exerciseID]
	if !ok {
		return true // Нет противопоказаний
	}

	for _, contra := range contras {
		for _, constraint := range constraints {
			if contra.BodyZone == constraint.BodyZone {
				// Абсолютное противопоказание - исключаем
				if contra.Severity == models.SeverityAbsolute {
					return false
				}
				// Относительное + строгое ограничение клиента - исключаем
				if contra.Severity == models.SeverityRelative && constraint.Severity == models.SeverityAbsolute {
					return false
				}
			}
		}
	}

	return true
}

// rankAndSelect выбирает лучшее упражнение из кандидатов
func (s *ExerciseSelector) rankAndSelect(candidates []models.ExerciseExt, criteria SelectionCriteria) models.ExerciseExt {
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Простая логика ранжирования:
	// 1. Свободные веса > тренажёры (для силы/гипертрофии)
	// 2. Compound > Isolation (если требуется)
	// 3. Меньшая сложность (для новичков)

	best := candidates[0]
	bestScore := s.scoreExercise(&best, criteria)

	for i := 1; i < len(candidates); i++ {
		score := s.scoreExercise(&candidates[i], criteria)
		if score > bestScore {
			best = candidates[i]
			bestScore = score
		}
	}

	return best
}

// scoreExercise оценивает упражнение
func (s *ExerciseSelector) scoreExercise(ex *models.ExerciseExt, criteria SelectionCriteria) int {
	score := 0

	// Свободные веса +2
	for _, eq := range ex.Equipment {
		if eq == models.EquipmentBarbell || eq == models.EquipmentDumbbell || eq == models.EquipmentKettlebell {
			score += 2
			break
		}
	}

	// Compound +3 (если требуется)
	if criteria.RequireCompound && ex.IsCompound {
		score += 3
	}

	// Bilateral +1 (обычно эффективнее)
	if ex.Pattern == models.PatternBilateral {
		score += 1
	}

	return score
}

// DayPattern - паттерн движения для дня
type DayPattern struct {
	MovementType    models.MovementType
	PrimaryMuscle   models.MuscleGroupExt
	RequireCompound bool
	Order           int // Порядок в тренировке
}

// getDayPatterns возвращает паттерны движений для типа дня
func getDayPatterns(dayType string) []DayPattern {
	switch strings.ToLower(dayType) {
	case "push":
		return []DayPattern{
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleChest, RequireCompound: true, Order: 1},
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleShoulders, RequireCompound: true, Order: 2},
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleChest, RequireCompound: false, Order: 3},
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleTriceps, RequireCompound: false, Order: 4},
		}

	case "pull":
		return []DayPattern{
			{MovementType: models.MovementPull, PrimaryMuscle: models.MuscleBack, RequireCompound: true, Order: 1},
			{MovementType: models.MovementPull, PrimaryMuscle: models.MuscleBack, RequireCompound: true, Order: 2},
			{MovementType: models.MovementPull, PrimaryMuscle: models.MuscleRearDelts, RequireCompound: false, Order: 3},
			{MovementType: models.MovementPull, PrimaryMuscle: models.MuscleBiceps, RequireCompound: false, Order: 4},
		}

	case "legs", "lower":
		return []DayPattern{
			{MovementType: models.MovementSquat, PrimaryMuscle: models.MuscleQuads, RequireCompound: true, Order: 1},
			{MovementType: models.MovementHinge, PrimaryMuscle: models.MuscleHamstrings, RequireCompound: true, Order: 2},
			{MovementType: models.MovementLunge, PrimaryMuscle: models.MuscleGlutes, RequireCompound: true, Order: 3},
			{MovementType: models.MovementSquat, PrimaryMuscle: models.MuscleQuads, RequireCompound: false, Order: 4},
			{MovementType: models.MovementHinge, PrimaryMuscle: models.MuscleHamstrings, RequireCompound: false, Order: 5},
			{MovementType: models.MovementCore, PrimaryMuscle: models.MuscleCalves, RequireCompound: false, Order: 6},
		}

	case "upper":
		return []DayPattern{
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleChest, RequireCompound: true, Order: 1},
			{MovementType: models.MovementPull, PrimaryMuscle: models.MuscleBack, RequireCompound: true, Order: 2},
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleShoulders, RequireCompound: true, Order: 3},
			{MovementType: models.MovementPull, PrimaryMuscle: models.MuscleBack, RequireCompound: false, Order: 4},
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleTriceps, RequireCompound: false, Order: 5},
			{MovementType: models.MovementPull, PrimaryMuscle: models.MuscleBiceps, RequireCompound: false, Order: 6},
		}

	case "fullbody", "full_body":
		return []DayPattern{
			{MovementType: models.MovementSquat, PrimaryMuscle: models.MuscleQuads, RequireCompound: true, Order: 1},
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleChest, RequireCompound: true, Order: 2},
			{MovementType: models.MovementPull, PrimaryMuscle: models.MuscleBack, RequireCompound: true, Order: 3},
			{MovementType: models.MovementHinge, PrimaryMuscle: models.MuscleHamstrings, RequireCompound: true, Order: 4},
			{MovementType: models.MovementPush, PrimaryMuscle: models.MuscleShoulders, RequireCompound: false, Order: 5},
			{MovementType: models.MovementCore, PrimaryMuscle: models.MuscleCore, RequireCompound: false, Order: 6},
		}

	default:
		// По умолчанию - full body
		return getDayPatterns("fullbody")
	}
}

// GetExerciseByID возвращает упражнение по ID
func (s *ExerciseSelector) GetExerciseByID(id string) *models.ExerciseExt {
	for i := range s.exercises {
		if s.exercises[i].ID == id {
			return &s.exercises[i]
		}
	}
	return nil
}

// GetAllExercises возвращает все загруженные упражнения
func (s *ExerciseSelector) GetAllExercises() []models.ExerciseExt {
	return s.exercises
}

// GetExercisesByMuscle возвращает упражнения для мышечной группы
func (s *ExerciseSelector) GetExercisesByMuscle(muscle models.MuscleGroupExt) []models.ExerciseExt {
	var result []models.ExerciseExt
	for _, ex := range s.exercises {
		for _, m := range ex.PrimaryMuscles {
			if m == muscle {
				result = append(result, ex)
				break
			}
		}
	}
	return result
}

// GetExercisesByEquipment возвращает упражнения для оборудования
func (s *ExerciseSelector) GetExercisesByEquipment(equipment models.EquipmentType) []models.ExerciseExt {
	var result []models.ExerciseExt
	for _, ex := range s.exercises {
		for _, eq := range ex.Equipment {
			if eq == equipment {
				result = append(result, ex)
				break
			}
		}
	}
	return result
}

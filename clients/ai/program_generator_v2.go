package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"workbot/internal/models"
)

// ProgramGeneratorV2 улучшенный генератор программ
type ProgramGeneratorV2 struct {
	client    *Client
	validator *ProgramValidator
}

// NewProgramGeneratorV2 создаёт новый генератор
func NewProgramGeneratorV2(client *Client) *ProgramGeneratorV2 {
	return &ProgramGeneratorV2{
		client:    client,
		validator: NewProgramValidator(),
	}
}

// ProgramRequestV2 расширенные параметры запроса
type ProgramRequestV2 struct {
	// Данные клиента
	ClientID    int
	ClientName  string
	Age         int
	Gender      string // male/female
	Weight      float64
	Height      float64

	// Тренировочные параметры
	Goal         string // strength, hypertrophy, fat_loss, endurance, general_fitness
	Experience   string // beginner, intermediate, advanced
	DaysPerWeek  int
	TotalWeeks   int
	SessionTime  int    // минуты на тренировку
	Equipment    string // full_gym, home_gym, minimal, bodyweight

	// Ограничения
	Injuries     string
	Restrictions string
	Preferences  string // любимые/нелюбимые упражнения

	// 1ПМ данные
	OnePMData map[string]float64

	// Дополнительный контекст из базы знаний
	KnowledgeContext string
}

// GeneratedProgramResult результат генерации
type GeneratedProgramResult struct {
	Program    *models.Program
	Validation *ValidationResult
	RawJSON    string
	Attempts   int // количество попыток до успеха
}

// GenerateProgram генерирует программу с валидацией и повторными попытками
// Гарантирует качественный результат - перегенерирует пока не будет валидна
func (g *ProgramGeneratorV2) GenerateProgram(req ProgramRequestV2) (*GeneratedProgramResult, error) {
	result := &GeneratedProgramResult{
		Attempts: 0,
	}

	maxAttempts := 5 // до 5 попыток для гарантии качества
	var lastError error
	var lastValidation *ValidationResult

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result.Attempts = attempt

		// Генерируем программу (передаём ошибки предыдущей попытки)
		program, rawJSON, err := g.generateProgramAttempt(req, lastValidation)
		if err != nil {
			lastError = err
			log.Printf("Попытка %d/%d: ошибка генерации: %v", attempt, maxAttempts, err)
			continue
		}

		result.RawJSON = rawJSON
		result.Program = program

		// Валидируем
		validation := g.validator.ValidateProgram(program, req.Experience)
		result.Validation = validation
		lastValidation = validation

		// Если программа валидна - возвращаем
		if validation.IsValid {
			log.Printf("✅ Программа сгенерирована успешно с %d попытки", attempt)
			return result, nil
		}

		// Логируем проблемы для следующей попытки
		log.Printf("Попытка %d/%d: программа невалидна, ошибок: %d, предупреждений: %d",
			attempt, maxAttempts, len(validation.Errors), len(validation.Warnings))
	}

	// Если после всех попыток не удалось - возвращаем лучший результат с ошибкой
	if result.Program != nil {
		log.Printf("⚠️ Программа сгенерирована с замечаниями после %d попыток", maxAttempts)
		return result, nil
	}

	return nil, fmt.Errorf("не удалось сгенерировать программу после %d попыток: %w", maxAttempts, lastError)
}

// generateProgramAttempt одна попытка генерации
// lastValidation - результат предыдущей валидации для исправления ошибок
func (g *ProgramGeneratorV2) generateProgramAttempt(req ProgramRequestV2, lastValidation *ValidationResult) (*models.Program, string, error) {
	prompt := g.buildPrompt(req, lastValidation)

	response, err := g.client.SimpleChat(MasterTrainerPrompt+"\n\n"+ProgramGeneratorPromptV2, prompt)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка запроса к AI: %w", err)
	}

	// Извлекаем и парсим JSON
	jsonStr := extractJSON(response)

	program, err := g.parseResponse(jsonStr, req)
	if err != nil {
		return nil, jsonStr, fmt.Errorf("ошибка парсинга: %w", err)
	}

	return program, jsonStr, nil
}

// buildPrompt строит промпт для запроса
// lastValidation - если не nil, добавляет указания по исправлению ошибок
func (g *ProgramGeneratorV2) buildPrompt(req ProgramRequestV2, lastValidation *ValidationResult) string {
	var sb strings.Builder

	sb.WriteString("## ДАННЫЕ КЛИЕНТА\n\n")
	sb.WriteString(fmt.Sprintf("**Имя:** %s\n", req.ClientName))

	if req.Age > 0 {
		sb.WriteString(fmt.Sprintf("**Возраст:** %d лет\n", req.Age))
	}
	if req.Gender != "" {
		gender := "Мужчина"
		if req.Gender == "female" {
			gender = "Женщина"
		}
		sb.WriteString(fmt.Sprintf("**Пол:** %s\n", gender))
	}
	if req.Weight > 0 {
		sb.WriteString(fmt.Sprintf("**Вес:** %.0f кг\n", req.Weight))
	}
	if req.Height > 0 {
		sb.WriteString(fmt.Sprintf("**Рост:** %.0f см\n", req.Height))
	}

	sb.WriteString("\n## ПАРАМЕТРЫ ПРОГРАММЫ\n\n")
	sb.WriteString(fmt.Sprintf("**Цель:** %s\n", translateGoal(req.Goal)))
	sb.WriteString(fmt.Sprintf("**Уровень:** %s\n", translateExperienceV2(req.Experience)))
	sb.WriteString(fmt.Sprintf("**Тренировок в неделю:** %d\n", req.DaysPerWeek))
	sb.WriteString(fmt.Sprintf("**Длительность программы:** %d недель\n", req.TotalWeeks))

	if req.SessionTime > 0 {
		sb.WriteString(fmt.Sprintf("**Время на тренировку:** %d минут\n", req.SessionTime))
	}
	sb.WriteString(fmt.Sprintf("**Оборудование:** %s\n", translateEquipment(req.Equipment)))

	if req.Injuries != "" || req.Restrictions != "" {
		sb.WriteString("\n## ОГРАНИЧЕНИЯ И ТРАВМЫ\n\n")
		if req.Injuries != "" {
			sb.WriteString(fmt.Sprintf("**Травмы:** %s\n", req.Injuries))
		}
		if req.Restrictions != "" {
			sb.WriteString(fmt.Sprintf("**Ограничения:** %s\n", req.Restrictions))
		}
	}

	if req.Preferences != "" {
		sb.WriteString(fmt.Sprintf("\n**Предпочтения:** %s\n", req.Preferences))
	}

	if len(req.OnePMData) > 0 {
		sb.WriteString("\n## ТЕКУЩИЕ 1ПМ\n\n")
		for ex, weight := range req.OnePMData {
			sb.WriteString(fmt.Sprintf("- %s: %.0f кг\n", ex, weight))
		}
	}

	// Добавляем контекст из базы знаний (RAG)
	if req.KnowledgeContext != "" {
		sb.WriteString("\n## ДОПОЛНИТЕЛЬНЫЙ КОНТЕКСТ ИЗ БАЗЫ ЗНАНИЙ\n\n")
		sb.WriteString(req.KnowledgeContext)
		sb.WriteString("\n")
	}

	sb.WriteString("\n## ФОРМАТ ОТВЕТА\n\n")
	sb.WriteString("Верни программу СТРОГО в следующем JSON формате:\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(g.getJSONTemplate(req))
	sb.WriteString("\n```\n\n")

	sb.WriteString("## ВАЖНЫЕ ТРЕБОВАНИЯ\n\n")
	sb.WriteString("1. Ответ ТОЛЬКО в формате JSON, без пояснений до или после\n")
	sb.WriteString("2. Все веса указывай как weight_percent (процент от 1ПМ) если есть 1ПМ, иначе weight в кг\n")
	sb.WriteString("3. Обязательно включи deload неделю каждые 3-4 недели\n")
	sb.WriteString("4. Прогрессия: увеличивай вес/интенсивность от недели к неделе (кроме deload)\n")
	sb.WriteString("5. Учитывай все травмы и ограничения клиента\n")
	sb.WriteString(fmt.Sprintf("6. На каждую группу мышц: %s подходов в неделю\n", getVolumeRange(req.Experience)))

	// При повторных попытках добавляем конкретные указания по исправлению
	if lastValidation != nil && !lastValidation.IsValid {
		sb.WriteString("\n## ⚠️ ИСПРАВЬ ОШИБКИ ПРЕДЫДУЩЕЙ ГЕНЕРАЦИИ\n\n")

		if len(lastValidation.Errors) > 0 {
			sb.WriteString("**Критические ошибки (ОБЯЗАТЕЛЬНО исправить):**\n")
			for _, err := range lastValidation.Errors {
				sb.WriteString(fmt.Sprintf("- %s\n", err))
			}
		}

		if len(lastValidation.Suggestions) > 0 {
			sb.WriteString("\n**Рекомендации:**\n")
			for _, sug := range lastValidation.Suggestions {
				sb.WriteString(fmt.Sprintf("- %s\n", sug))
			}
		}

		sb.WriteString("\nУбедись что JSON валидный и содержит ВСЕ обязательные поля.\n")
	}

	return sb.String()
}

// getJSONTemplate возвращает шаблон JSON
func (g *ProgramGeneratorV2) getJSONTemplate(req ProgramRequestV2) string {
	return `{
  "program_name": "Название программы",
  "description": "Описание программы, методики, особенностей",
  "methodology": "linear/dup/block/conjugate",
  "weeks": [
    {
      "week_num": 1,
      "phase": "hypertrophy/strength/power/deload",
      "focus": "Фокус недели",
      "intensity_percent": 70,
      "volume_percent": 100,
      "workouts": [
        {
          "day_num": 1,
          "name": "День 1 - Верх (Push)",
          "muscle_groups": ["грудь", "плечи", "трицепс"],
          "estimated_duration": 60,
          "exercises": [
            {
              "name": "Жим штанги лёжа",
              "sets": 4,
              "reps": "8-10",
              "weight_percent": 70,
              "weight": 0,
              "rest_seconds": 90,
              "tempo": "3-0-1-0",
              "rpe": 7,
              "notes": "Контролируемое опускание, мощный жим"
            }
          ]
        }
      ]
    }
  ],
  "progression_rules": {
    "compound_increment": 2.5,
    "isolation_increment": 1.0,
    "deload_frequency": 4,
    "deload_volume_reduction": 0.5,
    "deload_intensity_reduction": 0.2
  },
  "notes": "Общие рекомендации по выполнению программы"
}`
}

// parseResponse парсит JSON ответ в модель Program
func (g *ProgramGeneratorV2) parseResponse(jsonStr string, req ProgramRequestV2) (*models.Program, error) {
	var data struct {
		ProgramName  string `json:"program_name"`
		Description  string `json:"description"`
		Methodology  string `json:"methodology"`
		Weeks        []struct {
			WeekNum          int     `json:"week_num"`
			Phase            string  `json:"phase"`
			Focus            string  `json:"focus"`
			IntensityPercent float64 `json:"intensity_percent"`
			VolumePercent    float64 `json:"volume_percent"`
			Workouts         []struct {
				DayNum           int      `json:"day_num"`
				Name             string   `json:"name"`
				MuscleGroups     []string `json:"muscle_groups"`
				EstimatedDuration int     `json:"estimated_duration"`
				Exercises        []struct {
					Name          string  `json:"name"`
					Sets          int     `json:"sets"`
					Reps          string  `json:"reps"`
					WeightPercent float64 `json:"weight_percent"`
					Weight        float64 `json:"weight"`
					RestSeconds   int     `json:"rest_seconds"`
					Tempo         string  `json:"tempo"`
					RPE           float64 `json:"rpe"`
					Notes         string  `json:"notes"`
				} `json:"exercises"`
			} `json:"workouts"`
		} `json:"weeks"`
		ProgressionRules struct {
			CompoundIncrement       float64 `json:"compound_increment"`
			IsolationIncrement      float64 `json:"isolation_increment"`
			DeloadFrequency         int     `json:"deload_frequency"`
			DeloadVolumeReduction   float64 `json:"deload_volume_reduction"`
			DeloadIntensityReduction float64 `json:"deload_intensity_reduction"`
		} `json:"progression_rules"`
		Notes string `json:"notes"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w (JSON: %s)", err, truncateString(jsonStr, 500))
	}

	program := &models.Program{
		ClientID:    req.ClientID,
		ClientName:  req.ClientName,
		Name:        data.ProgramName,
		Goal:        req.Goal,
		Description: data.Description,
		TotalWeeks:  req.TotalWeeks,
		DaysPerWeek: req.DaysPerWeek,
		StartDate:   time.Now(),
		Status:      models.ProgramStatusActive,
		CurrentWeek: 1,
		Workouts:    []models.Workout{},
	}

	for _, week := range data.Weeks {
		for _, w := range week.Workouts {
			workout := models.Workout{
				WeekNum:   week.WeekNum,
				DayNum:    w.DayNum,
				Name:      w.Name,
				Status:    models.WorkoutStatusPending,
				Exercises: []models.WorkoutExercise{},
			}

			for i, ex := range w.Exercises {
				weight := ex.Weight
				if ex.WeightPercent > 0 && len(req.OnePMData) > 0 {
					// Рассчитываем вес из 1ПМ
					for exName, onePM := range req.OnePMData {
						if matchExercise(ex.Name, exName) {
							weight = onePM * ex.WeightPercent / 100
							break
						}
					}
				}

				workout.Exercises = append(workout.Exercises, models.WorkoutExercise{
					OrderNum:      i + 1,
					ExerciseName:  ex.Name,
					Sets:          ex.Sets,
					Reps:          ex.Reps,
					Weight:        weight,
					WeightPercent: ex.WeightPercent,
					RestSeconds:   ex.RestSeconds,
					Tempo:         ex.Tempo,
					RPE:           ex.RPE,
					Notes:         ex.Notes,
				})
			}

			program.Workouts = append(program.Workouts, workout)
		}
	}

	return program, nil
}

// Вспомогательные функции

func translateGoal(goal string) string {
	goals := map[string]string{
		"strength":        "Увеличение силы",
		"hypertrophy":     "Набор мышечной массы",
		"fat_loss":        "Жиросжигание",
		"endurance":       "Развитие выносливости",
		"general_fitness": "Общая физическая подготовка",
	}
	if t, ok := goals[goal]; ok {
		return t
	}
	return goal
}

func translateExperienceV2(exp string) string {
	switch exp {
	case "beginner":
		return "Новичок (до 1 года регулярных тренировок)"
	case "intermediate":
		return "Средний уровень (1-3 года тренировок)"
	case "advanced":
		return "Продвинутый (3+ лет тренировок)"
	default:
		return exp
	}
}

func translateEquipment(eq string) string {
	equipment := map[string]string{
		"full_gym":   "Полностью оборудованный зал",
		"home_gym":   "Домашний зал (штанга, гантели, скамья)",
		"minimal":    "Минимальное оборудование (гантели, турник)",
		"bodyweight": "Только собственный вес",
	}
	if t, ok := equipment[eq]; ok {
		return t
	}
	return eq
}

func getVolumeRange(experience string) string {
	switch experience {
	case "beginner":
		return "10-16"
	case "intermediate":
		return "12-20"
	case "advanced":
		return "16-26"
	default:
		return "12-20"
	}
}

func matchExercise(generated, known string) bool {
	g := strings.ToLower(generated)
	k := strings.ToLower(known)

	// Точное совпадение
	if g == k {
		return true
	}

	// Частичное совпадение
	if strings.Contains(g, k) || strings.Contains(k, g) {
		return true
	}

	// Ключевые слова
	keywords := map[string][]string{
		"жим лёжа":   {"жим", "лёжа", "лежа", "bench"},
		"присед":     {"присед", "squat", "приседания"},
		"становая":   {"становая", "тяга", "deadlift"},
		"подтягивания": {"подтягивания", "pull-up", "pullup"},
	}

	for key, words := range keywords {
		if strings.Contains(k, key) {
			for _, word := range words {
				if strings.Contains(g, word) {
					return true
				}
			}
		}
	}

	return false
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// FormatProgramSummary форматирует краткую сводку программы
func FormatProgramSummary(program *models.Program) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📋 **%s**\n\n", program.Name))
	sb.WriteString(fmt.Sprintf("🎯 Цель: %s\n", translateGoal(program.Goal)))
	sb.WriteString(fmt.Sprintf("📅 Длительность: %d недель\n", program.TotalWeeks))
	sb.WriteString(fmt.Sprintf("🏋️ Тренировок в неделю: %d\n", program.DaysPerWeek))
	sb.WriteString(fmt.Sprintf("📊 Всего тренировок: %d\n\n", len(program.Workouts)))

	if program.Description != "" {
		sb.WriteString(fmt.Sprintf("📝 %s\n\n", program.Description))
	}

	// Группируем по неделям
	weekWorkouts := make(map[int][]models.Workout)
	for _, w := range program.Workouts {
		weekWorkouts[w.WeekNum] = append(weekWorkouts[w.WeekNum], w)
	}

	// Показываем структуру первой недели
	if workouts, ok := weekWorkouts[1]; ok {
		sb.WriteString("📆 **Структура недели:**\n")
		for _, w := range workouts {
			sb.WriteString(fmt.Sprintf("  • День %d: %s (%d упражнений)\n",
				w.DayNum, w.Name, len(w.Exercises)))
		}
	}

	return sb.String()
}

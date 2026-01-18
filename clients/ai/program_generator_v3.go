package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"workbot/internal/models"
	"workbot/internal/training"
)

// ProgramGeneratorV3 полноценный генератор с научной периодизацией
type ProgramGeneratorV3 struct {
	client    *Client
	validator *ProgramValidatorV3
}

// NewProgramGeneratorV3 создаёт новый генератор
func NewProgramGeneratorV3(client *Client) *ProgramGeneratorV3 {
	return &ProgramGeneratorV3{
		client:    client,
		validator: NewProgramValidatorV3(),
	}
}

// ProgramRequestV3 расширенные параметры запроса
type ProgramRequestV3 struct {
	// Данные клиента
	ClientID   int
	ClientName string
	Age        int
	Gender     string  // male/female
	Weight     float64 // кг
	Height     float64 // см

	// Тренировочные параметры
	Goal        string // strength, hypertrophy, fat_loss, endurance, powerlifting, general_fitness
	Experience  string // beginner, intermediate, advanced
	DaysPerWeek int
	TotalWeeks  int
	SessionTime int    // минуты на тренировку
	Equipment   string // full_gym, home_gym, minimal, bodyweight

	// Ограничения
	Injuries     string
	Restrictions string
	Preferences  string // любимые/нелюбимые упражнения

	// 1ПМ данные (ключ - название упражнения, значение - вес в кг)
	OnePMData map[string]float64

	// Методология (опционально, AI выберет оптимальную если не указано)
	Methodology models.Methodology

	// Соревновательная дата (если есть)
	CompetitionDate *time.Time

	// Дополнительный контекст из базы знаний
	KnowledgeContext string
}

// GeneratedProgramResultV3 результат генерации
type GeneratedProgramResultV3 struct {
	Plan       *models.TrainingPlan
	Validation *ValidationResultV3
	RawJSON    string
	Attempts   int
}

// ValidationResultV3 результат валидации
type ValidationResultV3 struct {
	IsValid     bool     `json:"is_valid"`
	Score       int      `json:"score"`
	Errors      []string `json:"errors"`
	Warnings    []string `json:"warnings"`
	Suggestions []string `json:"suggestions"`
}

// GenerateProgram генерирует полную программу с периодизацией
// Использует пошаговую генерацию: сначала структура, потом недели по 2-3 штуки
func (g *ProgramGeneratorV3) GenerateProgram(req ProgramRequestV3) (*GeneratedProgramResultV3, error) {
	result := &GeneratedProgramResultV3{
		Attempts: 0,
	}

	log.Printf("🚀 Генерация программы: %d недель, %d дней/нед", req.TotalWeeks, req.DaysPerWeek)

	// Шаг 1: Генерируем структуру программы (без упражнений)
	log.Println("📋 Шаг 1: Генерация структуры периодизации...")
	structure, err := g.generateStructure(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации структуры: %w", err)
	}

	// Создаём план
	plan := &models.TrainingPlan{
		Name:        structure.ProgramName,
		Goal:        req.Goal,
		Description: structure.Description,
		Methodology: models.Methodology(structure.Methodology),
		OnePMData:   req.OnePMData,
		Weeks:       make([]models.TrainingWeek, 0, req.TotalWeeks),
	}

	// Шаг 2: Генерируем недели блоками по 2-3
	batchSize := 2
	if req.TotalWeeks <= 4 {
		batchSize = req.TotalWeeks // Короткие программы - всё сразу
	}

	for startWeek := 1; startWeek <= req.TotalWeeks; startWeek += batchSize {
		endWeek := startWeek + batchSize - 1
		if endWeek > req.TotalWeeks {
			endWeek = req.TotalWeeks
		}

		log.Printf("📅 Шаг 2.%d: Генерация недель %d-%d...", (startWeek-1)/batchSize+1, startWeek, endWeek)

		weeks, err := g.generateWeeksBatch(req, structure, startWeek, endWeek)
		if err != nil {
			log.Printf("⚠️ Ошибка генерации недель %d-%d: %v, пробуем по одной", startWeek, endWeek, err)
			// Пробуем по одной неделе
			for weekNum := startWeek; weekNum <= endWeek; weekNum++ {
				week, err := g.generateSingleWeek(req, structure, weekNum)
				if err != nil {
					return nil, fmt.Errorf("ошибка генерации недели %d: %w", weekNum, err)
				}
				plan.Weeks = append(plan.Weeks, *week)
			}
		} else {
			plan.Weeks = append(plan.Weeks, weeks...)
		}
	}

	// Валидируем
	validation := g.validator.Validate(plan, req)
	result.Validation = validation
	result.Plan = plan
	result.Attempts = 1

	if validation.IsValid {
		log.Printf("✅ Программа V3 сгенерирована успешно!")
	} else {
		log.Printf("⚠️ Программа сгенерирована с замечаниями (score: %d)", validation.Score)
	}

	// Рассчитываем абсолютные веса из 1ПМ
	g.calculateAbsoluteWeights(plan, req.OnePMData)

	return result, nil
}

// ProgramStructure структура программы (без деталей упражнений)
type ProgramStructure struct {
	ProgramName string `json:"program_name"`
	Description string `json:"description"`
	Methodology string `json:"methodology"`
	WeeksPlan   []struct {
		WeekNum       int      `json:"week_num"`
		Period        string   `json:"period"`
		MesocycleType string   `json:"mesocycle_type"`
		Phase         string   `json:"phase"`
		Focus         string   `json:"focus"`
		Accents       []string `json:"accents"`
		Intensity     float64  `json:"intensity_percent"`
		Volume        float64  `json:"volume_percent"`
		RPE           float64  `json:"rpe_target"`
		IsDeload      bool     `json:"is_deload"`
	} `json:"weeks_plan"`
}

// generateStructure генерирует только структуру периодизации
func (g *ProgramGeneratorV3) generateStructure(req ProgramRequestV3) (*ProgramStructure, error) {
	prompt := g.buildStructurePrompt(req)

	response, err := g.client.SimpleChat(ScientificPeriodizationPrompt, prompt)
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSON(response)

	var structure ProgramStructure
	if err := json.Unmarshal([]byte(jsonStr), &structure); err != nil {
		return nil, fmt.Errorf("ошибка парсинга структуры: %w (JSON: %.500s...)", err, jsonStr)
	}

	return &structure, nil
}

// buildStructurePrompt строит промпт для генерации структуры
func (g *ProgramGeneratorV3) buildStructurePrompt(req ProgramRequestV3) string {
	var sb strings.Builder

	sb.WriteString("## ЗАДАЧА: Создать СТРУКТУРУ ПЕРИОДИЗАЦИИ (без упражнений)\n\n")

	sb.WriteString("## ДАННЫЕ КЛИЕНТА\n\n")
	sb.WriteString(fmt.Sprintf("- Цель: %s\n", translateGoalV3(req.Goal)))
	sb.WriteString(fmt.Sprintf("- Уровень: %s\n", translateExperienceV3(req.Experience)))
	sb.WriteString(fmt.Sprintf("- Недель: %d\n", req.TotalWeeks))
	sb.WriteString(fmt.Sprintf("- Тренировок/нед: %d\n", req.DaysPerWeek))

	if req.CompetitionDate != nil {
		sb.WriteString(fmt.Sprintf("- Дата соревнований: %s\n", req.CompetitionDate.Format("02.01.2006")))
	}

	sb.WriteString("\n## ТРЕБОВАНИЯ\n\n")
	sb.WriteString(fmt.Sprintf("1. Создай структуру РОВНО на %d недель\n", req.TotalWeeks))
	sb.WriteString("2. Включи deload каждые 3-4 недели\n")
	sb.WriteString("3. Логичная прогрессия периодов и фаз\n")
	sb.WriteString("4. Верни ТОЛЬКО JSON без текста\n\n")

	sb.WriteString("## ФОРМАТ ОТВЕТА (JSON)\n\n```json\n")
	sb.WriteString(`{
  "program_name": "Название",
  "description": "Краткое описание",
  "methodology": "linear|dup|block",
  "weeks_plan": [
    {"week_num": 1, "period": "preparatory", "mesocycle_type": "introductory", "phase": "hypertrophy", "focus": "Адаптация", "accents": ["volume", "technique"], "intensity_percent": 65, "volume_percent": 100, "rpe_target": 7, "is_deload": false},
    {"week_num": 2, ...}
  ]
}`)
	sb.WriteString("\n```\n")

	return sb.String()
}

// generateWeeksBatch генерирует несколько недель за раз
func (g *ProgramGeneratorV3) generateWeeksBatch(req ProgramRequestV3, structure *ProgramStructure, startWeek, endWeek int) ([]models.TrainingWeek, error) {
	prompt := g.buildWeeksBatchPrompt(req, structure, startWeek, endWeek)

	response, err := g.client.SimpleChat(ScientificPeriodizationPrompt, prompt)
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSON(response)

	var weeksData struct {
		Weeks []struct {
			WeekNum  int `json:"week_num"`
			Workouts []struct {
				DayNum       int      `json:"day_num"`
				Name         string   `json:"name"`
				Type         string   `json:"type"`
				MuscleGroups []string `json:"muscle_groups"`
				Exercises    []struct {
					OrderNum      int      `json:"order_num"`
					ExerciseName  string   `json:"exercise_name"`
					MuscleGroup   string   `json:"muscle_group"`
					Sets          int      `json:"sets"`
					Reps          string   `json:"reps"`
					WeightPercent float64  `json:"weight_percent"`
					RestSeconds   int      `json:"rest_seconds"`
					Tempo         string   `json:"tempo"`
					RPE           float64  `json:"rpe"`
					Notes         string   `json:"notes"`
				} `json:"exercises"`
			} `json:"workouts"`
		} `json:"weeks"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &weeksData); err != nil {
		return nil, fmt.Errorf("ошибка парсинга недель: %w (JSON: %.500s...)", err, jsonStr)
	}

	weeks := make([]models.TrainingWeek, 0, len(weeksData.Weeks))

	for _, wd := range weeksData.Weeks {
		// Находим план для этой недели
		var weekPlan *struct {
			WeekNum       int
			Period        string
			MesocycleType string
			Phase         string
			Focus         string
			Accents       []string
			Intensity     float64
			Volume        float64
			RPE           float64
			IsDeload      bool
		}
		for i := range structure.WeeksPlan {
			if structure.WeeksPlan[i].WeekNum == wd.WeekNum {
				weekPlan = &struct {
					WeekNum       int
					Period        string
					MesocycleType string
					Phase         string
					Focus         string
					Accents       []string
					Intensity     float64
					Volume        float64
					RPE           float64
					IsDeload      bool
				}{
					WeekNum:       structure.WeeksPlan[i].WeekNum,
					Period:        structure.WeeksPlan[i].Period,
					MesocycleType: structure.WeeksPlan[i].MesocycleType,
					Phase:         structure.WeeksPlan[i].Phase,
					Focus:         structure.WeeksPlan[i].Focus,
					Accents:       structure.WeeksPlan[i].Accents,
					Intensity:     structure.WeeksPlan[i].Intensity,
					Volume:        structure.WeeksPlan[i].Volume,
					RPE:           structure.WeeksPlan[i].RPE,
					IsDeload:      structure.WeeksPlan[i].IsDeload,
				}
				break
			}
		}

		week := models.TrainingWeek{
			WeekNum:  wd.WeekNum,
			Workouts: make([]models.DayWorkout, 0, len(wd.Workouts)),
		}

		if weekPlan != nil {
			week.Period = models.TrainingPeriod(weekPlan.Period)
			week.MesocycleType = models.MesocycleType(weekPlan.MesocycleType)
			week.Phase = models.PlanPhase(weekPlan.Phase)
			week.Focus = weekPlan.Focus
			week.IntensityPercent = weekPlan.Intensity
			week.VolumePercent = weekPlan.Volume
			week.RPETarget = weekPlan.RPE
			week.IsDeload = weekPlan.IsDeload
			for _, a := range weekPlan.Accents {
				week.Accents = append(week.Accents, models.WeekAccent(a))
			}
		}

		for _, wo := range wd.Workouts {
			workout := models.DayWorkout{
				DayNum:       wo.DayNum,
				Name:         wo.Name,
				Type:         wo.Type,
				MuscleGroups: wo.MuscleGroups,
				Exercises:    make([]models.WorkoutExerciseV2, 0, len(wo.Exercises)),
			}

			for _, ex := range wo.Exercises {
				exercise := models.WorkoutExerciseV2{
					OrderNum:      ex.OrderNum,
					ExerciseName:  ex.ExerciseName,
					MuscleGroup:   ex.MuscleGroup,
					Sets:          ex.Sets,
					Reps:          ex.Reps,
					WeightPercent: ex.WeightPercent,
					RestSeconds:   ex.RestSeconds,
					Tempo:         ex.Tempo,
					RPE:           ex.RPE,
					Notes:         ex.Notes,
				}
				workout.Exercises = append(workout.Exercises, exercise)
			}

			week.Workouts = append(week.Workouts, workout)
		}

		weeks = append(weeks, week)
	}

	return weeks, nil
}

// buildWeeksBatchPrompt строит промпт для генерации недель
func (g *ProgramGeneratorV3) buildWeeksBatchPrompt(req ProgramRequestV3, structure *ProgramStructure, startWeek, endWeek int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## ЗАДАЧА: Создать ДЕТАЛЬНЫЕ ТРЕНИРОВКИ для недель %d-%d\n\n", startWeek, endWeek))

	sb.WriteString("## ПАРАМЕТРЫ\n\n")
	sb.WriteString(fmt.Sprintf("- Тренировок в неделю: %d\n", req.DaysPerWeek))
	sb.WriteString(fmt.Sprintf("- Оборудование: %s\n", translateEquipmentV3(req.Equipment)))
	sb.WriteString(fmt.Sprintf("- Цель: %s\n", translateGoalV3(req.Goal)))

	if len(req.OnePMData) > 0 {
		sb.WriteString("\n## 1ПМ КЛИЕНТА\n")
		for ex, w := range req.OnePMData {
			sb.WriteString(fmt.Sprintf("- %s: %.0f кг\n", ex, w))
		}
	}

	sb.WriteString("\n## ПЛАН НЕДЕЛЬ (фазы уже определены)\n\n")
	for i := startWeek - 1; i < endWeek && i < len(structure.WeeksPlan); i++ {
		wp := structure.WeeksPlan[i]
		sb.WriteString(fmt.Sprintf("- Неделя %d: %s, %s, интенсивность %.0f%%, RPE %.1f%s\n",
			wp.WeekNum, wp.Phase, wp.Focus, wp.Intensity, wp.RPE,
			map[bool]string{true: " (DELOAD)", false: ""}[wp.IsDeload]))
	}

	sb.WriteString("\n## ТРЕБОВАНИЯ\n\n")
	sb.WriteString(fmt.Sprintf("1. Создай РОВНО %d тренировок на КАЖДУЮ неделю\n", req.DaysPerWeek))
	sb.WriteString("2. 5-7 упражнений на тренировку\n")
	sb.WriteString("3. Указывай weight_percent для базовых упражнений (если есть 1ПМ)\n")
	sb.WriteString("4. Верни ТОЛЬКО JSON\n\n")

	sb.WriteString("## ФОРМАТ ОТВЕТА\n\n```json\n")
	sb.WriteString(`{"weeks": [{"week_num": X, "workouts": [{"day_num": 1, "name": "День 1 - Верх A", "type": "push", "muscle_groups": ["грудь", "плечи", "трицепс"], "exercises": [{"order_num": 1, "exercise_name": "Жим лёжа", "muscle_group": "грудь", "sets": 4, "reps": "6-8", "weight_percent": 75, "rest_seconds": 180, "tempo": "2-1-2-0", "rpe": 8, "notes": ""}]}]}]}`)
	sb.WriteString("\n```\n")

	return sb.String()
}

// generateSingleWeek генерирует одну неделю
func (g *ProgramGeneratorV3) generateSingleWeek(req ProgramRequestV3, structure *ProgramStructure, weekNum int) (*models.TrainingWeek, error) {
	weeks, err := g.generateWeeksBatch(req, structure, weekNum, weekNum)
	if err != nil {
		return nil, err
	}
	if len(weeks) == 0 {
		return nil, fmt.Errorf("пустой результат для недели %d", weekNum)
	}
	return &weeks[0], nil
}

// generateProgramAttempt одна попытка генерации
func (g *ProgramGeneratorV3) generateProgramAttempt(req ProgramRequestV3, lastValidation *ValidationResultV3) (*models.TrainingPlan, string, error) {
	prompt := g.buildPrompt(req, lastValidation)

	response, err := g.client.SimpleChat(ScientificPeriodizationPrompt, prompt)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка запроса к AI: %w", err)
	}

	jsonStr := extractJSON(response)

	plan, err := g.parseResponse(jsonStr, req)
	if err != nil {
		return nil, jsonStr, fmt.Errorf("ошибка парсинга: %w", err)
	}

	return plan, jsonStr, nil
}

// buildPrompt строит промпт для запроса
func (g *ProgramGeneratorV3) buildPrompt(req ProgramRequestV3, lastValidation *ValidationResultV3) string {
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
	sb.WriteString(fmt.Sprintf("**Цель:** %s\n", translateGoalV3(req.Goal)))
	sb.WriteString(fmt.Sprintf("**Уровень:** %s\n", translateExperienceV3(req.Experience)))
	sb.WriteString(fmt.Sprintf("**Тренировок в неделю:** %d\n", req.DaysPerWeek))
	sb.WriteString(fmt.Sprintf("**Длительность программы:** %d недель\n", req.TotalWeeks))

	if req.SessionTime > 0 {
		sb.WriteString(fmt.Sprintf("**Время на тренировку:** %d минут\n", req.SessionTime))
	}
	sb.WriteString(fmt.Sprintf("**Оборудование:** %s\n", translateEquipmentV3(req.Equipment)))

	if req.Methodology != "" {
		sb.WriteString(fmt.Sprintf("**Методология периодизации:** %s\n", req.Methodology.NameRu()))
	}

	if req.CompetitionDate != nil {
		sb.WriteString(fmt.Sprintf("**Дата соревнований:** %s\n", req.CompetitionDate.Format("02.01.2006")))
	}

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
		sb.WriteString("\n## ТЕКУЩИЕ 1ПМ КЛИЕНТА\n\n")
		sb.WriteString("Используй эти данные для расчёта рабочих весов через weight_percent:\n")
		for ex, weight := range req.OnePMData {
			sb.WriteString(fmt.Sprintf("- %s: %.0f кг\n", ex, weight))
		}
	}

	if req.KnowledgeContext != "" {
		sb.WriteString("\n## ДОПОЛНИТЕЛЬНЫЙ КОНТЕКСТ\n\n")
		sb.WriteString(req.KnowledgeContext)
		sb.WriteString("\n")
	}

	// JSON шаблон
	sb.WriteString("\n## ФОРМАТ ОТВЕТА\n\n")
	sb.WriteString("Верни программу СТРОГО в следующем JSON формате.\n")
	sb.WriteString(fmt.Sprintf("КРИТИЧНО: Программа должна содержать РОВНО %d недель и %d тренировок в каждой неделе!\n\n", req.TotalWeeks, req.DaysPerWeek))
	sb.WriteString("```json\n")
	sb.WriteString(g.getJSONTemplate(req))
	sb.WriteString("\n```\n\n")

	sb.WriteString("## КРИТИЧЕСКИ ВАЖНЫЕ ТРЕБОВАНИЯ\n\n")
	sb.WriteString(fmt.Sprintf("1. Программа ОБЯЗАТЕЛЬНО должна содержать ВСЕ %d недель полностью заполненными\n", req.TotalWeeks))
	sb.WriteString(fmt.Sprintf("2. Каждая неделя ОБЯЗАТЕЛЬНО содержит РОВНО %d тренировок\n", req.DaysPerWeek))
	sb.WriteString("3. НЕ используй многоточие (...) или сокращения - пиши ВСЁ полностью\n")
	sb.WriteString("4. Каждая неделя имеет period, mesocycle_type, phase, accents\n")
	sb.WriteString("5. Прогрессия: веса/интенсивность растут от недели к неделе (кроме deload)\n")
	sb.WriteString("6. Deload неделя обязательна каждые 3-4 недели\n")
	sb.WriteString("7. weight_percent указывай для всех упражнений где есть 1ПМ\n")
	sb.WriteString("8. Ответ ТОЛЬКО JSON, без текста до или после\n")

	// Исправление ошибок предыдущей попытки
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
	}

	return sb.String()
}

// getJSONTemplate возвращает шаблон JSON с полной периодизацией
func (g *ProgramGeneratorV3) getJSONTemplate(req ProgramRequestV3) string {
	return fmt.Sprintf(`{
  "program_name": "Название программы",
  "description": "Подробное описание программы, методики, научного обоснования",
  "methodology": "linear|dup|block|conjugate|hybrid",
  "weeks": [
    {
      "week_num": 1,
      "period": "preparatory|competitive|transitional",
      "mesocycle_type": "introductory|basic|control_prep|pre_competitive|competitive|recovery",
      "phase": "hypertrophy|strength|power|deload|accumulation|transmutation|realization",
      "focus": "Фокус недели (например: Адаптация, техника базовых движений)",
      "accents": ["volume", "technique"],
      "intensity_percent": 65,
      "volume_percent": 100,
      "rpe_target": 7,
      "is_deload": false,
      "notes": "Заметки к неделе",
      "workouts": [
        {
          "day_num": 1,
          "name": "День 1 - Верх (Push)",
          "type": "push|pull|legs|upper|lower|fullbody",
          "muscle_groups": ["грудь", "плечи", "трицепс"],
          "estimated_duration": 60,
          "exercises": [
            {
              "order_num": 1,
              "exercise_name": "Жим штанги лёжа",
              "muscle_group": "грудь",
              "movement_type": "compound",
              "sets": 4,
              "reps": "8-10",
              "weight_percent": 70,
              "weight_kg": 0,
              "rest_seconds": 90,
              "tempo": "3-0-1-0",
              "rpe": 7,
              "notes": "Контролируемое опускание",
              "alternatives": ["Жим гантелей лёжа"],
              "superset_with": ""
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
    "deload_intensity_reduction": 0.2,
    "weekly_intensity_increase": 2.5,
    "weekly_volume_increase": 5
  },
  "notes": "Общие рекомендации по выполнению программы"
}

ВАЖНО: Массив weeks должен содержать РОВНО %d элементов (недель).
Каждый элемент workouts должен содержать РОВНО %d тренировок.`, req.TotalWeeks, req.DaysPerWeek)
}

// parseResponse парсит JSON ответ в TrainingPlan
func (g *ProgramGeneratorV3) parseResponse(jsonStr string, req ProgramRequestV3) (*models.TrainingPlan, error) {
	var data struct {
		ProgramName  string `json:"program_name"`
		Description  string `json:"description"`
		Methodology  string `json:"methodology"`
		Weeks        []struct {
			WeekNum          int      `json:"week_num"`
			Period           string   `json:"period"`
			MesocycleType    string   `json:"mesocycle_type"`
			Phase            string   `json:"phase"`
			Focus            string   `json:"focus"`
			Accents          []string `json:"accents"`
			IntensityPercent float64  `json:"intensity_percent"`
			VolumePercent    float64  `json:"volume_percent"`
			RPETarget        float64  `json:"rpe_target"`
			IsDeload         bool     `json:"is_deload"`
			Notes            string   `json:"notes"`
			Workouts         []struct {
				DayNum            int      `json:"day_num"`
				Name              string   `json:"name"`
				Type              string   `json:"type"`
				MuscleGroups      []string `json:"muscle_groups"`
				EstimatedDuration int      `json:"estimated_duration"`
				Exercises         []struct {
					OrderNum      int      `json:"order_num"`
					ExerciseName  string   `json:"exercise_name"`
					MuscleGroup   string   `json:"muscle_group"`
					MovementType  string   `json:"movement_type"`
					Sets          int      `json:"sets"`
					Reps          string   `json:"reps"`
					WeightPercent float64  `json:"weight_percent"`
					WeightKg      float64  `json:"weight_kg"`
					RestSeconds   int      `json:"rest_seconds"`
					Tempo         string   `json:"tempo"`
					RPE           float64  `json:"rpe"`
					Notes         string   `json:"notes"`
					Alternatives  []string `json:"alternatives"`
					SupersetWith  string   `json:"superset_with"`
				} `json:"exercises"`
			} `json:"workouts"`
		} `json:"weeks"`
		ProgressionRules struct {
			CompoundIncrement        float64 `json:"compound_increment"`
			IsolationIncrement       float64 `json:"isolation_increment"`
			DeloadFrequency          int     `json:"deload_frequency"`
			DeloadVolumeReduction    float64 `json:"deload_volume_reduction"`
			DeloadIntensityReduction float64 `json:"deload_intensity_reduction"`
			WeeklyIntensityIncrease  float64 `json:"weekly_intensity_increase"`
			WeeklyVolumeIncrease     float64 `json:"weekly_volume_increase"`
		} `json:"progression_rules"`
		Notes string `json:"notes"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w (JSON: %s)", err, truncateString(jsonStr, 500))
	}

	plan := &models.TrainingPlan{
		ClientID:    req.ClientID,
		ClientName:  req.ClientName,
		Name:        data.ProgramName,
		Description: data.Description,
		Goal:        req.Goal,
		DaysPerWeek: req.DaysPerWeek,
		TotalWeeks:  req.TotalWeeks,
		StartDate:   time.Now(),
		Status:      models.PlanStatusActive,
		AIGenerated: true,
		Methodology: models.Methodology(data.Methodology),
		OnePMData:   req.OnePMData,
		Weeks:       make([]models.TrainingWeek, 0, len(data.Weeks)),
	}

	// Правила прогрессии
	plan.ProgressionRules = &models.ProgressionRules{
		CompoundIncrement:        data.ProgressionRules.CompoundIncrement,
		IsolationIncrement:       data.ProgressionRules.IsolationIncrement,
		DeloadFrequency:          data.ProgressionRules.DeloadFrequency,
		DeloadVolumeReduction:    data.ProgressionRules.DeloadVolumeReduction,
		DeloadIntensityReduction: data.ProgressionRules.DeloadIntensityReduction,
		WeeklyIntensityIncrease:  data.ProgressionRules.WeeklyIntensityIncrease,
		WeeklyVolumeIncrease:     data.ProgressionRules.WeeklyVolumeIncrease,
	}

	// Парсим недели
	for _, weekData := range data.Weeks {
		week := models.TrainingWeek{
			WeekNum:          weekData.WeekNum,
			Period:           models.TrainingPeriod(weekData.Period),
			MesocycleType:    models.MesocycleType(weekData.MesocycleType),
			Phase:            models.PlanPhase(weekData.Phase),
			Focus:            weekData.Focus,
			IntensityPercent: weekData.IntensityPercent,
			VolumePercent:    weekData.VolumePercent,
			RPETarget:        weekData.RPETarget,
			IsDeload:         weekData.IsDeload,
			Notes:            weekData.Notes,
			Workouts:         make([]models.DayWorkout, 0, len(weekData.Workouts)),
		}

		// Парсим акценты
		for _, accent := range weekData.Accents {
			week.Accents = append(week.Accents, models.WeekAccent(accent))
		}

		// Парсим тренировки
		for _, workoutData := range weekData.Workouts {
			workout := models.DayWorkout{
				DayNum:            workoutData.DayNum,
				Name:              workoutData.Name,
				Type:              workoutData.Type,
				MuscleGroups:      workoutData.MuscleGroups,
				EstimatedDuration: workoutData.EstimatedDuration,
				Exercises:         make([]models.WorkoutExerciseV2, 0, len(workoutData.Exercises)),
			}

			// Парсим упражнения
			for _, exData := range workoutData.Exercises {
				ex := models.WorkoutExerciseV2{
					OrderNum:      exData.OrderNum,
					ExerciseName:  exData.ExerciseName,
					MuscleGroup:   exData.MuscleGroup,
					MovementType:  exData.MovementType,
					Sets:          exData.Sets,
					Reps:          exData.Reps,
					WeightPercent: exData.WeightPercent,
					WeightKg:      exData.WeightKg,
					RestSeconds:   exData.RestSeconds,
					Tempo:         exData.Tempo,
					RPE:           exData.RPE,
					Notes:         exData.Notes,
					Alternatives:  exData.Alternatives,
					SupersetWith:  exData.SupersetWith,
				}
				workout.Exercises = append(workout.Exercises, ex)
			}

			week.Workouts = append(week.Workouts, workout)
		}

		plan.Weeks = append(plan.Weeks, week)
	}

	return plan, nil
}

// calculateAbsoluteWeights рассчитывает абсолютные веса из 1ПМ
func (g *ProgramGeneratorV3) calculateAbsoluteWeights(plan *models.TrainingPlan, onePMData map[string]float64) {
	if len(onePMData) == 0 {
		return
	}

	for wi := range plan.Weeks {
		for di := range plan.Weeks[wi].Workouts {
			for ei := range plan.Weeks[wi].Workouts[di].Exercises {
				ex := &plan.Weeks[wi].Workouts[di].Exercises[ei]

				// Если есть процент и нет абсолютного веса
				if ex.WeightPercent > 0 && ex.WeightKg == 0 {
					// Ищем 1ПМ для этого упражнения
					for exName, onePM := range onePMData {
						if matchExerciseV3(ex.ExerciseName, exName) {
							ex.WeightKg = training.CalculateWorkingWeight(onePM, ex.WeightPercent)
							break
						}
					}
				}
			}
		}
	}
}

// matchExerciseV3 проверяет соответствие названий упражнений
func matchExerciseV3(generated, known string) bool {
	g := strings.ToLower(generated)
	k := strings.ToLower(known)

	if g == k {
		return true
	}

	if strings.Contains(g, k) || strings.Contains(k, g) {
		return true
	}

	// Ключевые слова для сопоставления
	keywords := map[string][]string{
		"жим лёжа":       {"жим", "лёжа", "лежа", "bench", "жим штанги лёжа"},
		"присед":         {"присед", "squat", "приседания", "приседания со штангой"},
		"становая":       {"становая", "тяга", "deadlift", "становая тяга"},
		"подтягивания":   {"подтягивания", "pull-up", "pullup"},
		"жим стоя":       {"жим стоя", "армейский", "overhead press"},
		"тяга в наклоне": {"тяга в наклоне", "тяга штанги", "barbell row"},
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

// Вспомогательные функции
func translateGoalV3(goal string) string {
	goals := map[string]string{
		"strength":        "Развитие максимальной силы",
		"hypertrophy":     "Набор мышечной массы (гипертрофия)",
		"fat_loss":        "Жиросжигание с сохранением мышц",
		"endurance":       "Развитие мышечной выносливости",
		"powerlifting":    "Подготовка к соревнованиям по пауэрлифтингу",
		"general_fitness": "Общая физическая подготовка",
	}
	if t, ok := goals[goal]; ok {
		return t
	}
	return goal
}

func translateExperienceV3(exp string) string {
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

func translateEquipmentV3(eq string) string {
	equipment := map[string]string{
		"full_gym":   "Полностью оборудованный зал (штанги, гантели, тренажёры)",
		"home_gym":   "Домашний зал (штанга, гантели, скамья, стойки)",
		"minimal":    "Минимальное оборудование (гантели, турник, брусья)",
		"bodyweight": "Только собственный вес",
	}
	if t, ok := equipment[eq]; ok {
		return t
	}
	return eq
}

// FormatProgramSummaryV3 форматирует сводку программы
func FormatProgramSummaryV3(plan *models.TrainingPlan) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📋 **%s**\n\n", plan.Name))
	sb.WriteString(fmt.Sprintf("🎯 Цель: %s\n", translateGoalV3(plan.Goal)))
	sb.WriteString(fmt.Sprintf("📅 Длительность: %d недель\n", plan.TotalWeeks))
	sb.WriteString(fmt.Sprintf("🏋️ Тренировок в неделю: %d\n", plan.DaysPerWeek))
	sb.WriteString(fmt.Sprintf("📊 Методология: %s\n", plan.Methodology.NameRu()))

	if plan.Description != "" {
		sb.WriteString(fmt.Sprintf("\n📝 %s\n", plan.Description))
	}

	// Структура мезоциклов
	sb.WriteString("\n📆 **Структура программы:**\n")

	currentMeso := ""
	for _, week := range plan.Weeks {
		mesoName := week.MesocycleType.NameRu()
		if mesoName != currentMeso {
			currentMeso = mesoName
			sb.WriteString(fmt.Sprintf("\n**%s:**\n", mesoName))
		}

		deloadMark := ""
		if week.IsDeload {
			deloadMark = " 🔄"
		}
		sb.WriteString(fmt.Sprintf("  • Неделя %d: %s (%.0f%% интенс.)%s\n",
			week.WeekNum, week.Phase.NameRu(), week.IntensityPercent, deloadMark))
	}

	// Структура первой недели
	if len(plan.Weeks) > 0 {
		week := plan.Weeks[0]
		sb.WriteString("\n📌 **Пример недели 1:**\n")
		for _, w := range week.Workouts {
			sb.WriteString(fmt.Sprintf("  • %s (%d упражнений, ~%d мин)\n",
				w.Name, len(w.Exercises), w.EstimatedDuration))
		}
	}

	return sb.String()
}

package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"workbot/internal/models"
)

// ProgramRequest параметры для генерации программы
type ProgramRequest struct {
	ClientName   string
	Goal         string
	Experience   string // beginner, intermediate, advanced
	DaysPerWeek  int
	TotalWeeks   int
	Equipment    string // gym, home, minimal
	Injuries     string
	Preferences  string
	OnePMData    map[string]float64 // упражнение -> 1ПМ
}

// GenerateFullProgram генерирует полную программу тренировок через AI
func (c *Client) GenerateFullProgram(req ProgramRequest) (*models.Program, error) {
	prompt := buildProgramPrompt(req)

	response, err := c.SimpleChat("Ты профессиональный фитнес-тренер.", prompt)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к AI: %w", err)
	}

	program, err := parseProgramResponse(response, req)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	return program, nil
}

func buildProgramPrompt(req ProgramRequest) string {
	var sb strings.Builder

	sb.WriteString("Ты - профессиональный тренер. Создай полную программу тренировок.\n\n")

	sb.WriteString("ДАННЫЕ КЛИЕНТА:\n")
	sb.WriteString(fmt.Sprintf("- Имя: %s\n", req.ClientName))
	sb.WriteString(fmt.Sprintf("- Цель: %s\n", req.Goal))
	sb.WriteString(fmt.Sprintf("- Опыт: %s\n", translateExperience(req.Experience)))
	sb.WriteString(fmt.Sprintf("- Тренировок в неделю: %d\n", req.DaysPerWeek))
	sb.WriteString(fmt.Sprintf("- Длительность программы: %d недель\n", req.TotalWeeks))
	sb.WriteString(fmt.Sprintf("- Оборудование: %s\n", req.Equipment))

	if req.Injuries != "" {
		sb.WriteString(fmt.Sprintf("- Травмы/ограничения: %s\n", req.Injuries))
	}
	if req.Preferences != "" {
		sb.WriteString(fmt.Sprintf("- Предпочтения: %s\n", req.Preferences))
	}

	if len(req.OnePMData) > 0 {
		sb.WriteString("\n1ПМ клиента:\n")
		for ex, weight := range req.OnePMData {
			sb.WriteString(fmt.Sprintf("- %s: %.0f кг\n", ex, weight))
		}
	}

	sb.WriteString("\nФОРМАТ ОТВЕТА (строго JSON):\n")
	sb.WriteString(`{
  "program_name": "Название программы",
  "description": "Описание программы и методики",
  "weeks": [
    {
      "week_num": 1,
      "phase": "hypertrophy",
      "workouts": [
        {
          "day_num": 1,
          "name": "День 1 - Грудь/Трицепс",
          "exercises": [
            {
              "name": "Жим штанги лёжа",
              "sets": 4,
              "reps": "8-10",
              "weight_percent": 70,
              "rest_seconds": 90,
              "tempo": "3-1-2-0",
              "rpe": 7,
              "notes": "Контролируемое опускание"
            }
          ]
        }
      ]
    }
  ]
}`)

	sb.WriteString("\n\nТРЕБОВАНИЯ:\n")
	sb.WriteString("1. Учитывай периодизацию: гипертрофия -> сила -> мощность (если цель - сила)\n")
	sb.WriteString("2. Включай разгрузочные недели каждые 3-4 недели\n")
	sb.WriteString("3. Прогрессия нагрузки от недели к неделе\n")
	sb.WriteString("4. Указывай отдых между подходами в секундах\n")
	sb.WriteString("5. Если есть 1ПМ - указывай weight_percent, иначе - примерный вес\n")
	sb.WriteString("6. Темп в формате: эксцентрика-пауза внизу-концентрика-пауза вверху\n")
	sb.WriteString("7. Учитывай травмы и ограничения клиента\n")
	sb.WriteString("8. ВАЖНО: Ответ ТОЛЬКО в формате JSON, без пояснений\n")

	return sb.String()
}

func translateExperience(exp string) string {
	switch exp {
	case "beginner":
		return "Новичок (до 1 года)"
	case "intermediate":
		return "Средний (1-3 года)"
	case "advanced":
		return "Продвинутый (3+ лет)"
	default:
		return exp
	}
}

type programJSON struct {
	ProgramName string `json:"program_name"`
	Description string `json:"description"`
	Weeks       []struct {
		WeekNum  int    `json:"week_num"`
		Phase    string `json:"phase"`
		Workouts []struct {
			DayNum    int    `json:"day_num"`
			Name      string `json:"name"`
			Exercises []struct {
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
}

func parseProgramResponse(response string, req ProgramRequest) (*models.Program, error) {
	// Извлекаем JSON из ответа
	response = extractJSON(response)

	var data programJSON
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	program := &models.Program{
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

	workoutOrder := 1
	for _, week := range data.Weeks {
		orderInWeek := 1
		for _, w := range week.Workouts {
			workout := models.Workout{
				WeekNum:     week.WeekNum,
				DayNum:      w.DayNum,
				OrderInWeek: orderInWeek,
				Name:        w.Name,
				Status:      models.WorkoutStatusPending,
				Exercises:   []models.WorkoutExercise{},
			}

			for i, ex := range w.Exercises {
				weight := ex.Weight
				if ex.WeightPercent > 0 && len(req.OnePMData) > 0 {
					// Если есть 1ПМ и указан процент - рассчитываем вес
					for exName, onePM := range req.OnePMData {
						if strings.Contains(strings.ToLower(ex.Name), strings.ToLower(exName)) {
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
			orderInWeek++
			workoutOrder++
		}
	}

	return program, nil
}

func extractJSON(s string) string {
	// Убираем markdown блоки ```json ... ```
	if idx := strings.Index(s, "```json"); idx != -1 {
		s = s[idx+7:]
	} else if idx := strings.Index(s, "```"); idx != -1 {
		s = s[idx+3:]
	}
	if idx := strings.LastIndex(s, "```"); idx != -1 {
		s = s[:idx]
	}

	// Ищем начало JSON
	start := strings.Index(s, "{")
	if start == -1 {
		return s
	}

	// Ищем конец JSON (последняя закрывающая скобка)
	end := strings.LastIndex(s, "}")
	if end == -1 || end <= start {
		return s
	}

	jsonStr := s[start : end+1]

	// Убираем JavaScript-style комментарии // ...
	lines := strings.Split(jsonStr, "\n")
	var cleanLines []string
	for _, line := range lines {
		// Убираем однострочные комментарии (но не внутри строк)
		cleanLine := removeLineComment(line)
		cleanLines = append(cleanLines, cleanLine)
	}

	return strings.Join(cleanLines, "\n")
}

// removeLineComment убирает комментарии из строки, не трогая содержимое в кавычках
func removeLineComment(line string) string {
	inString := false
	escaped := false
	for i, ch := range line {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		// Нашли // вне строки - обрезаем
		if !inString && ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			return strings.TrimRight(line[:i], " \t")
		}
	}
	return line
}

// FormatWorkoutMessage форматирует тренировку для отправки клиенту
func FormatWorkoutMessage(workout *models.Workout, weekNum int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Тренировка: %s\n", workout.Name))
	sb.WriteString(fmt.Sprintf("Неделя %d\n", weekNum))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

	for i, ex := range workout.Exercises {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ex.ExerciseName))
		sb.WriteString(fmt.Sprintf("   Подходы: %d\n", ex.Sets))
		sb.WriteString(fmt.Sprintf("   Повторы: %s\n", ex.Reps))

		if ex.Weight > 0 {
			sb.WriteString(fmt.Sprintf("   Вес: %.0f кг\n", ex.Weight))
		} else if ex.WeightPercent > 0 {
			sb.WriteString(fmt.Sprintf("   Вес: %.0f%% от 1ПМ\n", ex.WeightPercent))
		}

		if ex.RestSeconds > 0 {
			if ex.RestSeconds >= 60 {
				minutes := ex.RestSeconds / 60
				seconds := ex.RestSeconds % 60
				if seconds > 0 {
					sb.WriteString(fmt.Sprintf("   Отдых: %d мин %d сек\n", minutes, seconds))
				} else {
					sb.WriteString(fmt.Sprintf("   Отдых: %d мин\n", minutes))
				}
			} else {
				sb.WriteString(fmt.Sprintf("   Отдых: %d сек\n", ex.RestSeconds))
			}
		}

		if ex.Tempo != "" {
			sb.WriteString(fmt.Sprintf("   Темп: %s\n", ex.Tempo))
		}

		if ex.RPE > 0 {
			sb.WriteString(fmt.Sprintf("   RPE: %.0f\n", ex.RPE))
		}

		if ex.Notes != "" {
			sb.WriteString(fmt.Sprintf("   Заметки: %s\n", ex.Notes))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("Удачной тренировки!")

	return sb.String()
}

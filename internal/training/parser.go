package training

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"workbot/internal/models"
)

// Parse парсит текст тренировки из Telegram
func Parse(text string) ([]models.ExerciseInput, time.Time, error) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 0 {
		return nil, time.Time{}, fmt.Errorf("пустой текст")
	}

	trainingDate := time.Now()
	startLine := 0

	// Проверяем, является ли первая строка датой
	firstLine := strings.TrimSpace(lines[0])
	if parsedDate, ok := tryParseDate(firstLine); ok {
		trainingDate = parsedDate
		startLine = 1
	}

	var exercises []models.ExerciseInput

	// Регулярные выражения для парсинга упражнения
	// Формат 1: "Жим лежа 4x10x60", "Подтягивания 3x12" (через x)
	// Формат 2: "Подтягивания 4/10 20", "Жим лежа 4/10" (через /)
	patternX := regexp.MustCompile(`^(.+?)\s+(\d+)[xх](\d+)(?:[xх](\d+(?:[.,]\d+)?))?\s*(\S*)$`)
	patternSlash := regexp.MustCompile(`^(.+?)\s+(\d+)/(\d+)(?:\s+(\d+(?:[.,]\d+)?))?\s*$`)

	for i := startLine; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		var exercise *models.ExerciseInput

		// Пробуем формат с / (Подтягивания 4/10 20)
		if matches := patternSlash.FindStringSubmatch(line); matches != nil {
			exercise = &models.ExerciseInput{
				Name: strings.TrimSpace(matches[1]),
			}
			exercise.Sets, _ = strconv.Atoi(matches[2])
			exercise.Reps, _ = strconv.Atoi(matches[3])
			if matches[4] != "" {
				exercise.Weight, _ = strconv.ParseFloat(strings.Replace(matches[4], ",", ".", 1), 64)
			}
		}

		// Пробуем формат с x (Жим лежа 4x10x60)
		if exercise == nil {
			if matches := patternX.FindStringSubmatch(line); matches != nil {
				exercise = &models.ExerciseInput{
					Name: strings.TrimSpace(matches[1]),
				}
				exercise.Sets, _ = strconv.Atoi(matches[2])
				exercise.Reps, _ = strconv.Atoi(matches[3])
				if matches[4] != "" {
					exercise.Weight, _ = strconv.ParseFloat(strings.Replace(matches[4], ",", ".", 1), 64)
				}
				if matches[5] != "" {
					exercise.Comment = matches[5]
				}
			}
		}

		// Пробуем альтернативный формат (Жим лежа 4 10 60)
		if exercise == nil {
			exercise = parseAlternativeFormat(line)
		}

		if exercise != nil {
			exercises = append(exercises, *exercise)
		}
	}

	return exercises, trainingDate, nil
}

// parseAlternativeFormat парсит формат "Жим лежа 4 10 60"
func parseAlternativeFormat(line string) *models.ExerciseInput {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil
	}

	exercise := models.ExerciseInput{}
	numIdx := len(parts) - 1

	for numIdx >= 0 {
		if _, err := strconv.Atoi(parts[numIdx]); err != nil {
			if _, err := strconv.ParseFloat(strings.Replace(parts[numIdx], ",", ".", 1), 64); err != nil {
				break
			}
		}
		numIdx--
	}

	if numIdx < len(parts)-2 {
		exercise.Name = strings.Join(parts[:numIdx+1], " ")
		nums := parts[numIdx+1:]

		if len(nums) >= 2 {
			exercise.Sets, _ = strconv.Atoi(nums[0])
			exercise.Reps, _ = strconv.Atoi(nums[1])
			if len(nums) >= 3 {
				exercise.Weight, _ = strconv.ParseFloat(strings.Replace(nums[2], ",", ".", 1), 64)
			}
			return &exercise
		}
	}
	return nil
}

// tryParseDate пробует распарсить дату в разных форматах
func tryParseDate(s string) (time.Time, bool) {
	formats := []string{
		"02.01.2006",
		"2.01.2006",
		"02.1.2006",
		"2.1.2006",
		"02.01",
		"2.01",
		"02.1",
		"2.1",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			if t.Year() == 0 {
				t = t.AddDate(time.Now().Year(), 0, 0)
			}
			return t, true
		}
	}
	return time.Time{}, false
}

// FormatConfirmation форматирует подтверждение сохранения тренировки
func FormatConfirmation(exercises []models.ExerciseInput, trainingDate time.Time) string {
	var totalTonnage float64
	var exerciseList strings.Builder

	for i, ex := range exercises {
		tonnage := float64(ex.Sets) * float64(ex.Reps) * ex.Weight
		totalTonnage += tonnage
		exerciseList.WriteString(fmt.Sprintf("%d. %s %dx%d", i+1, ex.Name, ex.Sets, ex.Reps))
		if ex.Weight > 0 {
			exerciseList.WriteString(fmt.Sprintf("x%.0fкг", ex.Weight))
		}
		if ex.Comment != "" {
			exerciseList.WriteString(fmt.Sprintf(" (%s)", ex.Comment))
		}
		exerciseList.WriteString("\n")
	}

	return fmt.Sprintf(
		"Тренировка сохранена!\n\n"+
			"Дата: %s\n"+
			"Упражнений: %d\n"+
			"Общий тоннаж: %.0f кг\n\n"+
			"%s",
		trainingDate.Format("02.01.2006"),
		len(exercises),
		totalTonnage,
		exerciseList.String(),
	)
}

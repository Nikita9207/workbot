package gsheets

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/api/sheets/v4"
)

// WorkoutResult результат выполнения упражнения
type WorkoutResult struct {
	Date         time.Time
	WeekNum      int
	DayNum       int
	ExerciseName string
	PlannedSets  int
	ActualSets   int
	PlannedReps  string
	ActualReps   string
	Weight       float64
	RPE          float64
	Comment      string
}

// ReadProgramFromSheet читает программу из Google Sheets
func (c *Client) ReadProgramFromSheet(spreadsheetID string) (*ProgramData, error) {
	ctx := context.Background()
	config := DefaultProgramSheetConfig()

	// Читаем обзор
	overviewResp, err := c.sheets.Spreadsheets.Values.Get(spreadsheetID, config.OverviewSheet+"!A1:D50").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения обзора: %w", err)
	}

	program := &ProgramData{
		Weeks: []WeekData{},
	}

	// Парсим обзор
	for _, row := range overviewResp.Values {
		if len(row) < 2 {
			continue
		}
		key := fmt.Sprintf("%v", row[0])
		value := fmt.Sprintf("%v", row[1])

		switch key {
		case "Клиент:":
			program.ClientName = value
		case "Программа:":
			program.ProgramName = value
		case "Цель:":
			program.Goal = value
		case "Методология:":
			program.Methodology = value
		case "Период:":
			program.Period = value
		}
	}

	// Получаем список листов чтобы найти листы недель
	spreadsheet, err := c.sheets.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения структуры: %w", err)
	}

	// Читаем каждую неделю
	for _, sheet := range spreadsheet.Sheets {
		title := sheet.Properties.Title
		if len(title) > len(config.WeekSheetPrefix) && title[:len(config.WeekSheetPrefix)] == config.WeekSheetPrefix {
			weekData, err := c.readWeekSheet(spreadsheetID, title)
			if err != nil {
				log.Printf("Ошибка чтения недели %s: %v", title, err)
				continue
			}
			program.Weeks = append(program.Weeks, *weekData)
			program.TotalWeeks++
		}
	}

	return program, nil
}

// readWeekSheet читает данные одной недели
func (c *Client) readWeekSheet(spreadsheetID, sheetName string) (*WeekData, error) {
	ctx := context.Background()

	resp, err := c.sheets.Spreadsheets.Values.Get(spreadsheetID, sheetName+"!A1:L100").Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	week := &WeekData{
		Workouts: []WorkoutData{},
	}

	// Парсим номер недели из названия листа
	fmt.Sscanf(sheetName, "Неделя_%d", &week.WeekNum)

	var currentWorkout *WorkoutData
	inExercises := false

	for _, row := range resp.Values {
		if len(row) == 0 {
			continue
		}

		firstCell := fmt.Sprintf("%v", row[0])

		// Заголовок дня
		if len(firstCell) > 5 && firstCell[:5] == "ДЕНЬ " {
			if currentWorkout != nil {
				week.Workouts = append(week.Workouts, *currentWorkout)
			}
			currentWorkout = &WorkoutData{
				Exercises: []ExerciseData{},
			}
			fmt.Sscanf(firstCell, "ДЕНЬ %d:", &currentWorkout.DayNum)
			// Парсим название после ":"
			if idx := len("ДЕНЬ X: "); idx < len(firstCell) {
				currentWorkout.Name = firstCell[idx:]
			}
			inExercises = false
			continue
		}

		// Заголовок упражнений
		if firstCell == "№" {
			inExercises = true
			continue
		}

		// Упражнение
		// Колонки: №(0), Упражнение(1), Группа мышц(2), Тип движения(3), Подходы(4), Повторы(5), %1ПМ(6), Вес(7), Отдых(8), Темп(9), RPE(10), Заметки(11)
		if inExercises && currentWorkout != nil && len(row) >= 5 {
			ex := ExerciseData{}
			if v, ok := row[0].(float64); ok {
				ex.OrderNum = int(v)
			}
			if len(row) > 1 {
				ex.Name = fmt.Sprintf("%v", row[1])
			}
			if len(row) > 2 {
				ex.MuscleGroup = fmt.Sprintf("%v", row[2])
			}
			if len(row) > 3 {
				ex.MovementType = fmt.Sprintf("%v", row[3])
			}
			if len(row) > 4 {
				if v, ok := row[4].(float64); ok {
					ex.Sets = int(v)
				}
			}
			if len(row) > 5 {
				ex.Reps = fmt.Sprintf("%v", row[5])
			}
			if len(row) > 9 {
				ex.Tempo = fmt.Sprintf("%v", row[9])
			}
			if len(row) > 11 {
				ex.Notes = fmt.Sprintf("%v", row[11])
			}

			if ex.Name != "" {
				currentWorkout.Exercises = append(currentWorkout.Exercises, ex)
			}
		}
	}

	// Добавляем последнюю тренировку
	if currentWorkout != nil {
		week.Workouts = append(week.Workouts, *currentWorkout)
	}

	return week, nil
}

// WriteWorkoutResult записывает результат выполнения тренировки
func (c *Client) WriteWorkoutResult(spreadsheetID string, result WorkoutResult) error {
	ctx := context.Background()
	config := DefaultProgramSheetConfig()

	// Получаем текущие данные журнала
	resp, err := c.sheets.Spreadsheets.Values.Get(spreadsheetID, config.JournalSheet+"!A:K").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения журнала: %w", err)
	}

	// Ищем строку для данного упражнения
	for i, row := range resp.Values {
		if i < 3 { // Пропускаем заголовки
			continue
		}
		if len(row) < 4 {
			continue
		}

		// Проверяем неделю, день и упражнение
		weekNum := 0
		dayNum := 0
		if v, ok := row[1].(float64); ok {
			weekNum = int(v)
		}
		if v, ok := row[2].(float64); ok {
			dayNum = int(v)
		}
		exName := fmt.Sprintf("%v", row[3])

		if weekNum == result.WeekNum && dayNum == result.DayNum && exName == result.ExerciseName {
			// Нашли строку - обновляем
			updateRange := fmt.Sprintf("%s!A%d:K%d", config.JournalSheet, i+1, i+1)

			values := [][]interface{}{{
				result.Date.Format("02.01.2006"),
				result.WeekNum,
				result.DayNum,
				result.ExerciseName,
				result.PlannedSets,
				result.ActualSets,
				result.PlannedReps,
				result.ActualReps,
				result.Weight,
				result.RPE,
				result.Comment,
			}}

			valueRange := &sheets.ValueRange{Values: values}
			_, err = c.sheets.Spreadsheets.Values.Update(spreadsheetID, updateRange, valueRange).
				ValueInputOption("USER_ENTERED").
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("ошибка записи результата: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("не найдена строка для упражнения %s (неделя %d, день %d)",
		result.ExerciseName, result.WeekNum, result.DayNum)
}

// GetTodayWorkout возвращает тренировку на сегодня
func (c *Client) GetTodayWorkout(spreadsheetID string, weekNum, dayNum int) (*WorkoutData, error) {
	program, err := c.ReadProgramFromSheet(spreadsheetID)
	if err != nil {
		return nil, err
	}

	for _, week := range program.Weeks {
		if week.WeekNum == weekNum {
			for _, workout := range week.Workouts {
				if workout.DayNum == dayNum {
					return &workout, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("тренировка не найдена: неделя %d, день %d", weekNum, dayNum)
}

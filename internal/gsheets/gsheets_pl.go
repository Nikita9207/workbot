package gsheets

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/sheets/v4"
)

// ============================================
// Пауэрлифтинг программы
// ============================================

// PLProgramData данные пауэрлифтинг программы для экспорта
type PLProgramData struct {
	Name         string
	AthleteMaxes struct {
		Squat     float64
		Bench     float64
		Deadlift  float64
		HipThrust float64
	}
	Weeks        []PLWeekData
	TotalKPS     int
	TotalTonnage float64
}

// PLWeekData данные недели пауэрлифтинг программы
type PLWeekData struct {
	WeekNum  int
	Phase    string
	Workouts []PLWorkoutData
	TotalKPS int
	Tonnage  float64
}

// PLWorkoutData данные тренировки пауэрлифтинг программы
type PLWorkoutData struct {
	DayNum    int
	Name      string
	Exercises []PLExerciseData
	TotalKPS  int
	Tonnage   float64
}

// PLExerciseData данные упражнения пауэрлифтинг программы
type PLExerciseData struct {
	Name       string
	Type       string
	Sets       []PLSetData
	TotalReps  int
	Tonnage    float64
	AvgPercent float64
}

// PLSetData данные подхода
type PLSetData struct {
	Percent  float64
	Reps     int
	Sets     int
	WeightKg float64
}

// CreatePLProgramSpreadsheet создаёт Google таблицу для пауэрлифтинг программы
func (c *Client) CreatePLProgramSpreadsheet(program interface{}) (string, error) {
	ctx := context.Background()

	// Преобразуем в PLProgramData через JSON
	plProgram, ok := program.(*PLProgramData)
	if !ok {
		// Пробуем работать с ai.PLGeneratedProgram через рефлексию
		return c.createPLProgramFromGenerated(program)
	}

	return c.createPLProgramSheet(ctx, plProgram)
}

// createPLProgramFromGenerated создаёт базовую таблицу для неизвестного типа программы
func (c *Client) createPLProgramFromGenerated(program interface{}) (string, error) {
	ctx := context.Background()

	title := "Программа пауэрлифтинга"

	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{Title: title},
		Sheets: []*sheets.Sheet{
			{Properties: &sheets.SheetProperties{Title: "Программа", Index: 0}},
		},
	}

	created, err := c.sheets.Spreadsheets.Create(spreadsheet).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	spreadsheetID := created.SpreadsheetId

	if c.folderID != "" {
		if _, err = c.drive.Files.Update(spreadsheetID, nil).
			AddParents(c.folderID).
			Context(ctx).
			Do(); err != nil {
			log.Printf("Предупреждение: не удалось переместить таблицу: %v", err)
		}
	}

	headers := []interface{}{
		"Неделя", "Тренировка", "Упражнение", "Подходы×Повторы", "Вес (кг)", "%1ПМ", "КПШ", "Тоннаж",
	}
	c.writeRow(spreadsheetID, "Программа", 1, headers)
	c.formatHeaders(spreadsheetID, 0)

	log.Printf("Создана Google таблица для PL программы: %s", spreadsheetID)
	return spreadsheetID, nil
}

// createPLProgramSheet создаёт таблицу из PLProgramData
func (c *Client) createPLProgramSheet(ctx context.Context, program *PLProgramData) (string, error) {
	title := program.Name

	// Создаём листы
	sheetsList := []*sheets.Sheet{
		{Properties: &sheets.SheetProperties{Title: "Обзор", Index: 0}},
	}

	// Листы для каждой недели
	for i := range program.Weeks {
		sheetsList = append(sheetsList, &sheets.Sheet{
			Properties: &sheets.SheetProperties{
				Title: fmt.Sprintf("Неделя_%d", i+1),
				Index: int64(i + 1),
			},
		})
	}

	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{Title: title},
		Sheets:     sheetsList,
	}

	created, err := c.sheets.Spreadsheets.Create(spreadsheet).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	spreadsheetID := created.SpreadsheetId

	// Перемещаем в папку
	if c.folderID != "" {
		_, err = c.drive.Files.Update(spreadsheetID, nil).
			AddParents(c.folderID).
			Context(ctx).
			Do()
		if err != nil {
			log.Printf("Предупреждение: не удалось переместить таблицу: %v", err)
		}
	}

	// Заполняем обзорный лист
	overviewData := [][]interface{}{
		{"Программа", program.Name},
		{""},
		{"1ПМ данные:"},
	}
	if program.AthleteMaxes.Squat > 0 {
		overviewData = append(overviewData, []interface{}{"Присед", fmt.Sprintf("%.0f кг", program.AthleteMaxes.Squat)})
	}
	if program.AthleteMaxes.Bench > 0 {
		overviewData = append(overviewData, []interface{}{"Жим лёжа", fmt.Sprintf("%.0f кг", program.AthleteMaxes.Bench)})
	}
	if program.AthleteMaxes.Deadlift > 0 {
		overviewData = append(overviewData, []interface{}{"Тяга", fmt.Sprintf("%.0f кг", program.AthleteMaxes.Deadlift)})
	}
	if program.AthleteMaxes.HipThrust > 0 {
		overviewData = append(overviewData, []interface{}{"Ягодичный мост", fmt.Sprintf("%.0f кг", program.AthleteMaxes.HipThrust)})
	}
	overviewData = append(overviewData,
		[]interface{}{""},
		[]interface{}{"Статистика:"},
		[]interface{}{"Общий КПШ", program.TotalKPS},
		[]interface{}{"Общий тоннаж", fmt.Sprintf("%.1f т", program.TotalTonnage)},
		[]interface{}{"Недель", len(program.Weeks)},
	)
	c.writeRows(spreadsheetID, "Обзор", 1, overviewData)
	c.formatHeaders(spreadsheetID, 0)

	// Заполняем листы недель
	for i, week := range program.Weeks {
		sheetName := fmt.Sprintf("Неделя_%d", i+1)

		// Собираем все данные для записи одним batch
		data := [][]interface{}{
			{"Тренировка", "Упражнение", "Подходы×Повторы", "Вес (кг)", "%1ПМ", "КПШ", "Тоннаж"},
		}

		for _, workout := range week.Workouts {
			for j, ex := range workout.Exercises {
				workoutName := ""
				if j == 0 {
					workoutName = workout.Name
				}

				// Форматируем подходы
				setsStr := ""
				for k, set := range ex.Sets {
					if k > 0 {
						setsStr += ", "
					}
					if set.Sets > 1 {
						setsStr += fmt.Sprintf("%dx%d", set.Sets, set.Reps)
					} else {
						setsStr += fmt.Sprintf("%d", set.Reps)
					}
				}

				// Вес первого подхода
				weight := ""
				percent := ""
				if len(ex.Sets) > 0 && ex.Sets[0].WeightKg > 0 {
					weight = fmt.Sprintf("%.0f", ex.Sets[0].WeightKg)
					if ex.Sets[0].Percent > 0 {
						percent = fmt.Sprintf("%.0f%%", ex.Sets[0].Percent)
					}
				}

				data = append(data, []interface{}{
					workoutName, ex.Name, setsStr, weight, percent, ex.TotalReps, fmt.Sprintf("%.2f", ex.Tonnage),
				})
			}
			// Пустая строка между тренировками
			data = append(data, []interface{}{"", "", "", "", "", "", ""})
		}

		// Итоги недели
		data = append(data, []interface{}{
			"Итого неделя:", "", "", "", "", week.TotalKPS, fmt.Sprintf("%.2f т", week.Tonnage),
		})

		c.writeRows(spreadsheetID, sheetName, 1, data)
		c.formatHeaders(spreadsheetID, int64(i+1))
	}

	log.Printf("Создана Google таблица для PL программы: %s", spreadsheetID)
	return spreadsheetID, nil
}

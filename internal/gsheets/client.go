package gsheets

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Client клиент для работы с Google Sheets
type Client struct {
	sheets   *sheets.Service
	drive    *drive.Service
	folderID string
}

// NewClient создаёт новый клиент Google Sheets
func NewClient(credentialsPath, folderID string) (*Client, error) {
	ctx := context.Background()

	// Читаем credentials
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать credentials: %w", err)
	}

	// Создаём конфигурацию
	config, err := google.JWTConfigFromJSON(data,
		sheets.SpreadsheetsScope,
		drive.DriveScope,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка конфигурации: %w", err)
	}

	client := config.Client(ctx)

	// Создаём сервис Sheets
	sheetsSrv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания Sheets сервиса: %w", err)
	}

	// Создаём сервис Drive
	driveSrv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания Drive сервиса: %w", err)
	}

	return &Client{
		sheets:   sheetsSrv,
		drive:    driveSrv,
		folderID: folderID,
	}, nil
}

// CreateClientSpreadsheet создаёт таблицу для клиента
func (c *Client) CreateClientSpreadsheet(clientID int, name, surname string) (string, error) {
	ctx := context.Background()

	title := fmt.Sprintf("%s %s", name, surname)

	// Создаём новую таблицу
	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: title,
		},
		Sheets: []*sheets.Sheet{
			{
				Properties: &sheets.SheetProperties{
					Title: "Тренировки",
					Index: 0,
				},
			},
			{
				Properties: &sheets.SheetProperties{
					Title: "Анкета",
					Index: 1,
				},
			},
			{
				Properties: &sheets.SheetProperties{
					Title: "Статистика",
					Index: 2,
				},
			},
		},
	}

	created, err := c.sheets.Spreadsheets.Create(spreadsheet).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	spreadsheetID := created.SpreadsheetId

	// Перемещаем в папку WorkBot
	_, err = c.drive.Files.Update(spreadsheetID, nil).
		AddParents(c.folderID).
		Context(ctx).
		Do()
	if err != nil {
		log.Printf("Предупреждение: не удалось переместить таблицу в папку: %v", err)
	}

	// Добавляем заголовки на лист "Тренировки"
	headers := []interface{}{
		"Дата", "№ тренировки", "Упражнение", "Подходы", "Повторы", "Вес (кг)", "Тоннаж", "Заметки",
	}
	err = c.writeRow(spreadsheetID, "Тренировки", 1, headers)
	if err != nil {
		log.Printf("Ошибка записи заголовков: %v", err)
	}

	// Форматируем заголовки
	c.formatHeaders(spreadsheetID, 0)

	// Добавляем поля анкеты
	anketaFields := [][]interface{}{
		{"Поле", "Значение"},
		{"ФИО", fmt.Sprintf("%s %s", name, surname)},
		{"ID клиента", clientID},
		{"Дата регистрации", time.Now().Format("02.01.2006")},
		{"Телефон", ""},
		{"Дата рождения", ""},
		{"Цель тренировок", ""},
		{"Опыт", ""},
		{"Травмы/ограничения", ""},
		{"Примечания", ""},
	}
	c.writeRows(spreadsheetID, "Анкета", 1, anketaFields)
	c.formatHeaders(spreadsheetID, 1)

	log.Printf("Создана Google таблица для %s %s: %s", name, surname, spreadsheetID)
	return spreadsheetID, nil
}

// AddTraining добавляет тренировку в таблицу клиента
func (c *Client) AddTraining(spreadsheetID string, trainingDate time.Time, trainingNum int, exercises []TrainingExercise) error {
	ctx := context.Background()

	// Получаем текущие данные чтобы найти последнюю строку
	resp, err := c.sheets.Spreadsheets.Values.Get(spreadsheetID, "Тренировки!A:A").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("ошибка чтения таблицы: %w", err)
	}

	nextRow := len(resp.Values) + 1

	// Формируем данные для записи
	var values [][]interface{}
	for i, ex := range exercises {
		row := []interface{}{
			"", // Дата - только в первой строке тренировки
			"", // № тренировки - только в первой строке
			ex.Name,
			ex.Sets,
			ex.Reps,
			ex.Weight,
			ex.Sets * ex.Reps * int(ex.Weight), // Тоннаж
			ex.Notes,
		}
		if i == 0 {
			row[0] = trainingDate.Format("02.01.2006")
			row[1] = trainingNum
		}
		values = append(values, row)
	}

	// Записываем
	writeRange := fmt.Sprintf("Тренировки!A%d", nextRow)
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err = c.sheets.Spreadsheets.Values.Append(spreadsheetID, writeRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("ошибка записи тренировки: %w", err)
	}

	log.Printf("Тренировка добавлена в Google Sheets: %s", spreadsheetID)
	return nil
}

// TrainingExercise упражнение в тренировке
type TrainingExercise struct {
	Name   string
	Sets   int
	Reps   int
	Weight float64
	Notes  string
}

// writeRow записывает одну строку
func (c *Client) writeRow(spreadsheetID, sheetName string, row int, values []interface{}) error {
	ctx := context.Background()
	writeRange := fmt.Sprintf("%s!A%d", sheetName, row)
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{values},
	}
	_, err := c.sheets.Spreadsheets.Values.Update(spreadsheetID, writeRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	return err
}

// writeRows записывает несколько строк
func (c *Client) writeRows(spreadsheetID, sheetName string, startRow int, values [][]interface{}) error {
	ctx := context.Background()
	writeRange := fmt.Sprintf("%s!A%d", sheetName, startRow)
	valueRange := &sheets.ValueRange{
		Values: values,
	}
	_, err := c.sheets.Spreadsheets.Values.Update(spreadsheetID, writeRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	return err
}

// formatHeaders форматирует заголовки (жирный шрифт, цвет фона)
func (c *Client) formatHeaders(spreadsheetID string, sheetIndex int64) {
	ctx := context.Background()

	requests := []*sheets.Request{
		{
			RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:          sheetIndex,
					StartRowIndex:    0,
					EndRowIndex:      1,
					StartColumnIndex: 0,
					EndColumnIndex:   10,
				},
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						BackgroundColor: &sheets.Color{
							Red:   0.2,
							Green: 0.4,
							Blue:  0.8,
						},
						TextFormat: &sheets.TextFormat{
							Bold: true,
							ForegroundColor: &sheets.Color{
								Red:   1,
								Green: 1,
								Blue:  1,
							},
						},
					},
				},
				Fields: "userEnteredFormat(backgroundColor,textFormat)",
			},
		},
	}

	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	_, err := c.sheets.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
	if err != nil {
		log.Printf("Ошибка форматирования: %v", err)
	}
}

// GetSpreadsheetURL возвращает URL таблицы
func GetSpreadsheetURL(spreadsheetID string) string {
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", spreadsheetID)
}

// ============================================
// Методы для работы с программами тренировок
// ============================================

// ProgramSheetConfig конфигурация листов программы
type ProgramSheetConfig struct {
	OverviewSheet  string // Обзор программы
	WeekSheetPrefix string // Префикс для листов недель "Неделя_"
	JournalSheet   string // Журнал выполнения
	OnePMSheet     string // Данные 1ПМ
	ProgressSheet  string // Прогресс
}

// DefaultProgramSheetConfig возвращает стандартную конфигурацию
func DefaultProgramSheetConfig() ProgramSheetConfig {
	return ProgramSheetConfig{
		OverviewSheet:   "Обзор",
		WeekSheetPrefix: "Неделя_",
		JournalSheet:    "Журнал",
		OnePMSheet:      "1ПМ",
		ProgressSheet:   "Прогресс",
	}
}

// ProgramData данные программы для экспорта
type ProgramData struct {
	ClientName   string
	ProgramName  string
	Goal         string
	TotalWeeks   int
	DaysPerWeek  int
	Methodology  string
	Period       string
	CreatedAt    string
	OnePMData    map[string]float64
	Weeks        []WeekData
}

// WeekData данные недели
type WeekData struct {
	WeekNum          int
	Phase            string
	Focus            string
	IntensityPercent float64
	VolumePercent    float64
	RPETarget        float64
	IsDeload         bool
	Workouts         []WorkoutData
}

// WorkoutData данные тренировки
type WorkoutData struct {
	DayNum       int
	Name         string
	Type         string
	MuscleGroups []string
	Exercises    []ExerciseData
}

// ExerciseData данные упражнения
type ExerciseData struct {
	OrderNum      int
	Name          string
	MuscleGroup   string
	Sets          int
	Reps          string
	WeightPercent float64
	WeightKg      float64
	RestSeconds   int
	Tempo         string
	RPE           float64
	Notes         string
}

// CreateProgramSpreadsheet создаёт таблицу с программой тренировок
func (c *Client) CreateProgramSpreadsheet(program ProgramData) (string, error) {
	ctx := context.Background()
	config := DefaultProgramSheetConfig()

	title := fmt.Sprintf("%s - %s", program.ClientName, program.ProgramName)

	// Создаём листы для каждой недели + служебные листы
	sheetsList := []*sheets.Sheet{
		{Properties: &sheets.SheetProperties{Title: config.OverviewSheet, Index: 0}},
	}

	// Добавляем листы для каждой недели
	for i := 1; i <= program.TotalWeeks; i++ {
		sheetsList = append(sheetsList, &sheets.Sheet{
			Properties: &sheets.SheetProperties{
				Title: fmt.Sprintf("%s%d", config.WeekSheetPrefix, i),
				Index: int64(i),
			},
		})
	}

	// Добавляем журнал и прогресс
	sheetsList = append(sheetsList,
		&sheets.Sheet{Properties: &sheets.SheetProperties{Title: config.JournalSheet, Index: int64(program.TotalWeeks + 1)}},
		&sheets.Sheet{Properties: &sheets.SheetProperties{Title: config.OnePMSheet, Index: int64(program.TotalWeeks + 2)}},
		&sheets.Sheet{Properties: &sheets.SheetProperties{Title: config.ProgressSheet, Index: int64(program.TotalWeeks + 3)}},
	)

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
			log.Printf("Предупреждение: не удалось переместить в папку: %v", err)
		}
	}

	// Заполняем лист "Обзор"
	c.fillOverviewSheet(spreadsheetID, config.OverviewSheet, program)

	// Заполняем листы недель
	for _, week := range program.Weeks {
		sheetName := fmt.Sprintf("%s%d", config.WeekSheetPrefix, week.WeekNum)
		c.fillWeekSheet(spreadsheetID, sheetName, week, program.OnePMData)
	}

	// Заполняем лист 1ПМ
	c.fillOnePMSheet(spreadsheetID, config.OnePMSheet, program.OnePMData)

	// Подготавливаем журнал
	c.fillJournalSheet(spreadsheetID, config.JournalSheet, program)

	log.Printf("Создана программа в Google Sheets: %s", spreadsheetID)
	return spreadsheetID, nil
}

// fillOverviewSheet заполняет лист обзора
func (c *Client) fillOverviewSheet(spreadsheetID, sheetName string, program ProgramData) {
	data := [][]interface{}{
		{"ПРОГРАММА ТРЕНИРОВОК", "", "", ""},
		{"", "", "", ""},
		{"Клиент:", program.ClientName, "", ""},
		{"Программа:", program.ProgramName, "", ""},
		{"Цель:", program.Goal, "", ""},
		{"Длительность:", fmt.Sprintf("%d недель", program.TotalWeeks), "", ""},
		{"Тренировок/нед:", program.DaysPerWeek, "", ""},
		{"Методология:", program.Methodology, "", ""},
		{"Период:", program.Period, "", ""},
		{"Создана:", program.CreatedAt, "", ""},
		{"", "", "", ""},
		{"СТРУКТУРА ПРОГРАММЫ", "", "", ""},
		{"Неделя", "Фаза", "Фокус", "Интенсивность"},
	}

	// Добавляем информацию о каждой неделе
	for _, week := range program.Weeks {
		deloadMark := ""
		if week.IsDeload {
			deloadMark = " (Разгрузка)"
		}
		data = append(data, []interface{}{
			fmt.Sprintf("Неделя %d%s", week.WeekNum, deloadMark),
			week.Phase,
			week.Focus,
			fmt.Sprintf("%.0f%%", week.IntensityPercent),
		})
	}

	c.writeRows(spreadsheetID, sheetName, 1, data)
	c.formatHeaders(spreadsheetID, 0)
}

// fillWeekSheet заполняет лист недели
func (c *Client) fillWeekSheet(spreadsheetID, sheetName string, week WeekData, onePMData map[string]float64) {
	// Заголовок недели
	deloadMark := ""
	if week.IsDeload {
		deloadMark = " (РАЗГРУЗКА)"
	}

	data := [][]interface{}{
		{fmt.Sprintf("НЕДЕЛЯ %d%s", week.WeekNum, deloadMark), "", "", "", "", "", "", "", ""},
		{fmt.Sprintf("Фаза: %s | Фокус: %s | Интенсивность: %.0f%% | RPE: %.1f",
			week.Phase, week.Focus, week.IntensityPercent, week.RPETarget), "", "", "", "", "", "", "", ""},
		{"", "", "", "", "", "", "", "", ""},
	}

	// Для каждой тренировки
	for _, workout := range week.Workouts {
		// Заголовок тренировки
		data = append(data, []interface{}{
			fmt.Sprintf("ДЕНЬ %d: %s", workout.DayNum, workout.Name),
			"", "", "", "", "", "", "", "",
		})

		// Заголовки упражнений
		data = append(data, []interface{}{
			"№", "Упражнение", "Группа", "Подходы", "Повторы", "%1ПМ", "Вес(кг)", "Отдых", "Темп", "RPE", "Заметки",
		})

		// Упражнения
		for _, ex := range workout.Exercises {
			weightKg := ex.WeightKg
			// Если указан % от 1ПМ и есть данные 1ПМ - вычисляем вес
			if ex.WeightPercent > 0 && onePMData != nil {
				if onepm, ok := onePMData[ex.Name]; ok {
					weightKg = onepm * ex.WeightPercent / 100
				}
			}

			weightStr := ""
			if weightKg > 0 {
				weightStr = fmt.Sprintf("%.1f", weightKg)
			}

			percentStr := ""
			if ex.WeightPercent > 0 {
				percentStr = fmt.Sprintf("%.0f%%", ex.WeightPercent)
			}

			restStr := ""
			if ex.RestSeconds > 0 {
				restStr = fmt.Sprintf("%d сек", ex.RestSeconds)
			}

			rpeStr := ""
			if ex.RPE > 0 {
				rpeStr = fmt.Sprintf("%.1f", ex.RPE)
			}

			data = append(data, []interface{}{
				ex.OrderNum,
				ex.Name,
				ex.MuscleGroup,
				ex.Sets,
				ex.Reps,
				percentStr,
				weightStr,
				restStr,
				ex.Tempo,
				rpeStr,
				ex.Notes,
			})
		}

		// Пустая строка между тренировками
		data = append(data, []interface{}{"", "", "", "", "", "", "", "", "", "", ""})
	}

	c.writeRows(spreadsheetID, sheetName, 1, data)
}

// fillOnePMSheet заполняет лист 1ПМ
func (c *Client) fillOnePMSheet(spreadsheetID, sheetName string, onePMData map[string]float64) {
	data := [][]interface{}{
		{"ДАННЫЕ 1ПМ (Одноповторный максимум)", "", ""},
		{"", "", ""},
		{"Упражнение", "1ПМ (кг)", "Дата теста"},
	}

	for name, value := range onePMData {
		data = append(data, []interface{}{name, value, ""})
	}

	c.writeRows(spreadsheetID, sheetName, 1, data)
}

// fillJournalSheet заполняет журнал выполнения
func (c *Client) fillJournalSheet(spreadsheetID, sheetName string, program ProgramData) {
	data := [][]interface{}{
		{"ЖУРНАЛ ТРЕНИРОВОК", "", "", "", "", "", "", "", ""},
		{"", "", "", "", "", "", "", "", ""},
		{"Дата", "Неделя", "День", "Упражнение", "План подх.", "Факт подх.", "План повт.", "Факт повт.", "Вес(кг)", "RPE", "Комментарий"},
	}

	// Создаём строки для всех упражнений программы
	for _, week := range program.Weeks {
		for _, workout := range week.Workouts {
			for _, ex := range workout.Exercises {
				data = append(data, []interface{}{
					"", // Дата - клиент заполнит
					week.WeekNum,
					workout.DayNum,
					ex.Name,
					ex.Sets,
					"", // Факт подходы
					ex.Reps,
					"", // Факт повторы
					"", // Вес
					"", // RPE
					"", // Комментарий
				})
			}
		}
	}

	c.writeRows(spreadsheetID, sheetName, 1, data)
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

	resp, err := c.sheets.Spreadsheets.Values.Get(spreadsheetID, sheetName+"!A1:K100").Context(ctx).Do()
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
				if v, ok := row[3].(float64); ok {
					ex.Sets = int(v)
				}
			}
			if len(row) > 4 {
				ex.Reps = fmt.Sprintf("%v", row[4])
			}
			if len(row) > 8 {
				ex.Tempo = fmt.Sprintf("%v", row[8])
			}
			if len(row) > 10 {
				ex.Notes = fmt.Sprintf("%v", row[10])
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

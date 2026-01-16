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

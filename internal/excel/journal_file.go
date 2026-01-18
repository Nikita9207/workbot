package excel

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

// Названия листов в файле Журнал.xlsx
const (
	SheetJournalClients = "Клиенты"
	SheetHealth         = "Анкета здоровья" // добавим позже
)

// CreateJournalFile создаёт отдельный файл Журнал.xlsx для сбора данных о клиентах
func CreateJournalFile(filePath string, db *sql.DB) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Ошибка закрытия файла: %v", err)
		}
	}()

	// Создаём листы
	f.SetSheetName("Sheet1", SheetJournalClients)
	f.NewSheet(SheetHealth)

	// Заполняем лист клиентов
	if err := createClientsSheet(f); err != nil {
		return fmt.Errorf("ошибка создания листа клиентов: %w", err)
	}

	// Заполняем лист анкеты здоровья (пока заготовка)
	if err := createHealthSheet(f); err != nil {
		return fmt.Errorf("ошибка создания анкеты здоровья: %w", err)
	}

	// Если есть БД - подтягиваем существующих клиентов
	if db != nil {
		if err := loadClientsFromDB(f, db); err != nil {
			log.Printf("Предупреждение: не удалось загрузить клиентов из БД: %v", err)
		}
	}

	f.SetActiveSheet(0)

	if err := f.SaveAs(filePath); err != nil {
		return fmt.Errorf("ошибка сохранения файла: %w", err)
	}

	log.Printf("Создан файл журнала клиентов: %s", filePath)
	return nil
}

// createClientsSheet создаёт лист с данными клиентов
func createClientsSheet(f *excelize.File) error {
	sheet := SheetJournalClients

	// Стиль заголовка таблицы
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Стиль заголовков колонок
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#0D3B66", Style: 1},
			{Type: "right", Color: "#0D3B66", Style: 1},
			{Type: "top", Color: "#0D3B66", Style: 1},
			{Type: "bottom", Color: "#0D3B66", Style: 1},
		},
	})

	// Заголовок
	f.SetCellValue(sheet, "A1", "ЖУРНАЛ КЛИЕНТОВ")
	f.MergeCell(sheet, "A1", "L1")
	f.SetCellStyle(sheet, "A1", "L1", titleStyle)
	f.SetRowHeight(sheet, 1, 35)

	// Инструкция
	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Italic: true, Color: "#666666"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})
	f.SetCellValue(sheet, "A2", "Здесь собираются данные всех клиентов при регистрации через Telegram бота. Данные автоматически синхронизируются с базой данных.")
	f.MergeCell(sheet, "A2", "L2")
	f.SetCellStyle(sheet, "A2", "L2", instructionStyle)
	f.SetRowHeight(sheet, 2, 25)

	// Заголовки колонок
	headers := []struct {
		col   string
		title string
		width float64
	}{
		{"A", "ID", 6},
		{"B", "Telegram ID", 12},
		{"C", "Username", 15},
		{"D", "Имя", 15},
		{"E", "Фамилия", 15},
		{"F", "Телефон", 15},
		{"G", "Email", 20},
		{"H", "Формат", 10},          // онлайн/офлайн
		{"I", "Дата регистрации", 16},
		{"J", "Статус", 12},
		{"K", "Примечания", 25},
		{"L", "Файл таблицы", 20},    // путь к персональной таблице клиента
	}

	for _, h := range headers {
		f.SetCellValue(sheet, h.col+"3", h.title)
		f.SetColWidth(sheet, h.col, h.col, h.width)
	}
	f.SetCellStyle(sheet, "A3", "L3", headerStyle)
	f.SetRowHeight(sheet, 3, 30)

	// Фиксируем заголовки
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      3,
		TopLeftCell: "A4",
		ActivePane:  "bottomLeft",
	})

	// Выпадающие списки
	// Формат (H) - онлайн/офлайн
	formatVal := excelize.NewDataValidation(true)
	formatVal.Sqref = "H4:H1000"
	formatVal.SetDropList([]string{"онлайн", "офлайн"})
	f.AddDataValidation(sheet, formatVal)

	// Статус (J)
	statusVal := excelize.NewDataValidation(true)
	statusVal.Sqref = "J4:J1000"
	statusVal.SetDropList([]string{"активный", "приостановлен", "завершил", "новый"})
	f.AddDataValidation(sheet, statusVal)

	// Условное форматирование для формата
	onlineFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
	})
	offlineFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#1F4E79", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#BDD7EE"}, Pattern: 1},
	})

	f.SetConditionalFormat(sheet, "H4:H1000", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &onlineFmt, Value: "\"онлайн\""},
	})
	f.SetConditionalFormat(sheet, "H4:H1000", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &offlineFmt, Value: "\"офлайн\""},
	})

	// Условное форматирование для статуса
	activeFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
	})
	pausedFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#9C5700"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFEB9C"}, Pattern: 1},
	})
	finishedFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#666666"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#D9D9D9"}, Pattern: 1},
	})

	f.SetConditionalFormat(sheet, "J4:J1000", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &activeFmt, Value: "\"активный\""},
	})
	f.SetConditionalFormat(sheet, "J4:J1000", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &pausedFmt, Value: "\"приостановлен\""},
	})
	f.SetConditionalFormat(sheet, "J4:J1000", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &finishedFmt, Value: "\"завершил\""},
	})

	return nil
}

// createHealthSheet создаёт лист анкеты здоровья (заготовка для будущего)
func createHealthSheet(f *excelize.File) error {
	sheet := SheetHealth

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#548235"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#375623", Style: 1},
			{Type: "right", Color: "#375623", Style: 1},
			{Type: "top", Color: "#375623", Style: 1},
			{Type: "bottom", Color: "#375623", Style: 1},
		},
	})

	// Заголовок
	f.SetCellValue(sheet, "A1", "АНКЕТА ЗДОРОВЬЯ")
	f.MergeCell(sheet, "A1", "N1")
	f.SetCellStyle(sheet, "A1", "N1", titleStyle)
	f.SetRowHeight(sheet, 1, 35)

	// Инструкция
	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Italic: true, Color: "#666666"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})
	f.SetCellValue(sheet, "A2", "Анкета здоровья заполняется клиентами при регистрации через Telegram бота. Данные помогают составить безопасную программу тренировок.")
	f.MergeCell(sheet, "A2", "N2")
	f.SetCellStyle(sheet, "A2", "N2", instructionStyle)
	f.SetRowHeight(sheet, 2, 25)

	// Заголовки колонок анкеты
	headers := []struct {
		col   string
		title string
		width float64
	}{
		{"A", "ID клиента", 10},
		{"B", "Имя", 15},
		{"C", "Возраст", 10},
		{"D", "Рост (см)", 10},
		{"E", "Вес (кг)", 10},
		{"F", "Цель", 20},
		{"G", "Опыт тренировок", 15},
		{"H", "Травмы", 20},
		{"I", "Хронические заболевания", 22},
		{"J", "Противопоказания", 20},
		{"K", "Аллергии", 15},
		{"L", "Принимает лекарства", 18},
		{"M", "Дата заполнения", 15},
		{"N", "Примечания врача", 20},
	}

	for _, h := range headers {
		f.SetCellValue(sheet, h.col+"3", h.title)
		f.SetColWidth(sheet, h.col, h.col, h.width)
	}
	f.SetCellStyle(sheet, "A3", "N3", headerStyle)
	f.SetRowHeight(sheet, 3, 30)

	// Фиксируем заголовки
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      3,
		TopLeftCell: "A4",
		ActivePane:  "bottomLeft",
	})

	// Выпадающие списки
	// Цель (F)
	goalVal := excelize.NewDataValidation(true)
	goalVal.Sqref = "F4:F1000"
	goalVal.SetDropList([]string{"похудение", "набор массы", "поддержание формы", "сила", "выносливость", "здоровье", "реабилитация"})
	f.AddDataValidation(sheet, goalVal)

	// Опыт (G)
	expVal := excelize.NewDataValidation(true)
	expVal.Sqref = "G4:G1000"
	expVal.SetDropList([]string{"нет опыта", "до 6 месяцев", "6-12 месяцев", "1-2 года", "2-5 лет", "5+ лет"})
	f.AddDataValidation(sheet, expVal)

	return nil
}

// loadClientsFromDB загружает существующих клиентов из БД
func loadClientsFromDB(f *excelize.File, db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, telegram_id, name, surname, phone, birth_date, goal, created_at
		FROM public.clients
		WHERE deleted_at IS NULL
		ORDER BY id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	sheet := SheetJournalClients
	row := 4

	for rows.Next() {
		var id int
		var telegramID sql.NullInt64
		var name, surname, phone, birthDate, goal sql.NullString
		var createdAt sql.NullTime

		if err := rows.Scan(&id, &telegramID, &name, &surname, &phone, &birthDate, &goal, &createdAt); err != nil {
			continue
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
		if telegramID.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), telegramID.Int64)
		}

		if name.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), name.String)
		}
		if surname.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), surname.String)
		}
		if phone.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), phone.String)
		}
		if birthDate.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), birthDate.String)
		}
		if goal.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), goal.String)
		}
		if createdAt.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), createdAt.Time.Format("02.01.2006"))
		}

		// По умолчанию новый клиент
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), "новый")

		row++
	}

	return nil
}

package excel

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"workbot/internal/models"

	"github.com/xuri/excelize/v2"
)

// CreateClientSheet создаёт dashboard лист для клиента
func CreateClientSheet(f *excelize.File, sheetName string, clientID int, name, surname, phone, birthDate string) error {
	sheetIndex, err := f.GetSheetIndex(sheetName)
	if err != nil {
		return fmt.Errorf("ошибка проверки листа: %w", err)
	}
	if sheetIndex >= 0 {
		return nil
	}

	if _, err := f.NewSheet(sheetName); err != nil {
		return fmt.Errorf("ошибка создания листа: %w", err)
	}

	dataSheetName := sheetName + "_data"
	if len(dataSheetName) > 31 {
		sheetRunes := []rune(sheetName)
		if len(sheetRunes) > 25 {
			dataSheetName = string(sheetRunes[:25]) + "_data"
		}
	}
	if err := CreateExercisesDataSheet(f, dataSheetName, clientID); err != nil {
		return fmt.Errorf("ошибка создания листа данных: %w", err)
	}

	styles, err := createDashboardStyles(f)
	if err != nil {
		return err
	}

	if err := setupDashboardColumns(f, sheetName); err != nil {
		return err
	}

	if err := f.SetCellStyle(sheetName, "A1", "I40", styles.darkBg); err != nil {
		return fmt.Errorf("ошибка установки фона: %w", err)
	}

	if err := fillClientInfo(f, sheetName, styles, clientID, name, surname, phone, birthDate); err != nil {
		return err
	}

	if err := fillMonthTotals(f, sheetName, styles, dataSheetName); err != nil {
		return err
	}

	if err := fillCalendar(f, sheetName, styles); err != nil {
		return err
	}

	if err := fillGraphsPlaceholder(f, sheetName, styles); err != nil {
		return err
	}

	if err := fillFeedbackTable(f, sheetName, styles); err != nil {
		return err
	}

	log.Printf("Создан dashboard лист: %s", sheetName)
	return nil
}

type dashboardStyles struct {
	darkBg          int
	sectionTitle    int
	clientName      int
	subText         int
	bigNumber       int
	numberLabel     int
	calendarRest    int
	calendarPlanned int
	calendarDone    int
	calendarTonnage int
	tableHeader     int
	tableCell       int
}

func createDashboardStyles(f *excelize.File) (*dashboardStyles, error) {
	styles := &dashboardStyles{}
	var err error

	styles.darkBg, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#16213e"}, Pattern: 1},
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля darkBg: %w", err)
	}

	styles.sectionTitle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "#4ade80"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16213e"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля sectionTitle: %w", err)
	}

	styles.clientName, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 20, Color: "#ffffff"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16213e"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля clientName: %w", err)
	}

	styles.subText, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#9ca3af"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16213e"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля subText: %w", err)
	}

	styles.bigNumber, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 32, Color: "#4ade80"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16213e"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля bigNumber: %w", err)
	}

	styles.numberLabel, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9, Color: "#9ca3af"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16213e"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "top"},
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля numberLabel: %w", err)
	}

	calendarBorder := []excelize.Border{
		{Type: "left", Color: "#0f172a", Style: 1},
		{Type: "right", Color: "#0f172a", Style: 1},
		{Type: "top", Color: "#0f172a", Style: 1},
		{Type: "bottom", Color: "#0f172a", Style: 1},
	}

	styles.calendarRest, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "#e2e8f0"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1e3a5f"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    calendarBorder,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля calendarRest: %w", err)
	}

	styles.calendarPlanned, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "#ffffff"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ea580c"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    calendarBorder,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля calendarPlanned: %w", err)
	}

	styles.calendarDone, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "#ffffff"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16a34a"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    calendarBorder,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля calendarDone: %w", err)
	}

	styles.calendarTonnage, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#94a3b8"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1e3a5f"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:    calendarBorder,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля calendarTonnage: %w", err)
	}

	styles.tableHeader, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "#e2e8f0"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1e293b"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#334155", Style: 1},
			{Type: "right", Color: "#334155", Style: 1},
			{Type: "top", Color: "#334155", Style: 1},
			{Type: "bottom", Color: "#334155", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля tableHeader: %w", err)
	}

	styles.tableCell, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#cbd5e1"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16213e"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#1e293b", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стиля tableCell: %w", err)
	}

	return styles, nil
}

func setupDashboardColumns(f *excelize.File, sheetName string) error {
	colWidths := map[string]float64{
		"A": 14, "B": 10, "C": 2, "D": 14,
		"E": 10, "F": 2, "G": 12, "H": 12, "I": 25,
	}
	for col, width := range colWidths {
		if err := f.SetColWidth(sheetName, col, col, width); err != nil {
			return fmt.Errorf("ошибка установки ширины колонки %s: %w", col, err)
		}
	}
	return nil
}

func fillClientInfo(f *excelize.File, sheetName string, styles *dashboardStyles, clientID int, name, surname, phone, birthDate string) error {
	if err := f.SetCellValue(sheetName, "A1", "КЛИЕНТ"); err != nil {
		return fmt.Errorf("ошибка записи КЛИЕНТ: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "A1", "A1", styles.sectionTitle); err != nil {
		return fmt.Errorf("ошибка стиля A1: %w", err)
	}

	fullName := fmt.Sprintf("%s %s", name, surname)
	if err := f.SetCellValue(sheetName, "A2", fullName); err != nil {
		return fmt.Errorf("ошибка записи имени: %w", err)
	}
	if err := f.MergeCell(sheetName, "A2", "E2"); err != nil {
		return fmt.Errorf("ошибка объединения A2:E2: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "A2", "E2", styles.clientName); err != nil {
		return fmt.Errorf("ошибка стиля A2: %w", err)
	}

	info := fmt.Sprintf("ID: %d  |  %s  |  %s", clientID, phone, birthDate)
	if err := f.SetCellValue(sheetName, "A3", info); err != nil {
		return fmt.Errorf("ошибка записи инфо: %w", err)
	}
	if err := f.MergeCell(sheetName, "A3", "E3"); err != nil {
		return fmt.Errorf("ошибка объединения A3:E3: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "A3", "E3", styles.subText); err != nil {
		return fmt.Errorf("ошибка стиля A3: %w", err)
	}

	return nil
}

func fillMonthTotals(f *excelize.File, sheetName string, styles *dashboardStyles, dataSheetName string) error {
	if err := f.SetCellValue(sheetName, "G1", "ИТОГИ МЕСЯЦА"); err != nil {
		return fmt.Errorf("ошибка записи ИТОГИ: %w", err)
	}
	if err := f.MergeCell(sheetName, "G1", "I1"); err != nil {
		return fmt.Errorf("ошибка объединения G1:I1: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "G1", "I1", styles.sectionTitle); err != nil {
		return fmt.Errorf("ошибка стиля G1: %w", err)
	}

	if err := f.SetCellFormula(sheetName, "G2", fmt.Sprintf("'%s'!M2", dataSheetName)); err != nil {
		return fmt.Errorf("ошибка формулы тоннажа: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "G2", "G2", styles.bigNumber); err != nil {
		return fmt.Errorf("ошибка стиля G2: %w", err)
	}
	if err := f.SetCellValue(sheetName, "G3", "тоннаж (кг)"); err != nil {
		return fmt.Errorf("ошибка записи подписи тоннажа: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "G3", "G3", styles.numberLabel); err != nil {
		return fmt.Errorf("ошибка стиля G3: %w", err)
	}

	if err := f.SetCellFormula(sheetName, "H2", fmt.Sprintf("'%s'!N2", dataSheetName)); err != nil {
		return fmt.Errorf("ошибка формулы тренировок: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "H2", "H2", styles.bigNumber); err != nil {
		return fmt.Errorf("ошибка стиля H2: %w", err)
	}
	if err := f.SetCellValue(sheetName, "H3", "тренировок"); err != nil {
		return fmt.Errorf("ошибка записи подписи тренировок: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "H3", "H3", styles.numberLabel); err != nil {
		return fmt.Errorf("ошибка стиля H3: %w", err)
	}

	return nil
}

func fillCalendar(f *excelize.File, sheetName string, styles *dashboardStyles) error {
	now := time.Now()
	month := now.Month()
	daysInMonth := time.Date(now.Year(), month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	monthNames := map[time.Month]string{
		time.January: "янв", time.February: "фев", time.March: "мар",
		time.April: "апр", time.May: "май", time.June: "июн",
		time.July: "июл", time.August: "авг", time.September: "сен",
		time.October: "окт", time.November: "ноя", time.December: "дек",
	}

	for day := 1; day <= 16; day++ {
		row := 4 + day
		dateStr := fmt.Sprintf("%d %s", day, monthNames[month])
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), dateStr); err != nil {
			return fmt.Errorf("ошибка записи даты %d: %w", day, err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), "—"); err != nil {
			return fmt.Errorf("ошибка записи тоннажа %d: %w", day, err)
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styles.calendarRest); err != nil {
			return fmt.Errorf("ошибка стиля даты %d: %w", day, err)
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styles.calendarTonnage); err != nil {
			return fmt.Errorf("ошибка стиля тоннажа %d: %w", day, err)
		}
		if err := f.SetRowHeight(sheetName, row, 20); err != nil {
			return fmt.Errorf("ошибка высоты строки %d: %w", row, err)
		}
	}

	for day := 17; day <= 31; day++ {
		row := 4 + (day - 16)
		if day <= daysInMonth {
			dateStr := fmt.Sprintf("%d %s", day, monthNames[month])
			if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), dateStr); err != nil {
				return fmt.Errorf("ошибка записи даты %d: %w", day, err)
			}
			if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), "—"); err != nil {
				return fmt.Errorf("ошибка записи тоннажа %d: %w", day, err)
			}
			if err := f.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), styles.calendarRest); err != nil {
				return fmt.Errorf("ошибка стиля даты %d: %w", day, err)
			}
			if err := f.SetCellStyle(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), styles.calendarTonnage); err != nil {
				return fmt.Errorf("ошибка стиля тоннажа %d: %w", day, err)
			}
		}
	}

	return nil
}

func fillGraphsPlaceholder(f *excelize.File, sheetName string, styles *dashboardStyles) error {
	if err := f.SetCellValue(sheetName, "G5", "ТОННАЖ ПО ДНЯМ"); err != nil {
		return fmt.Errorf("ошибка записи заголовка графика: %w", err)
	}
	if err := f.MergeCell(sheetName, "G5", "I5"); err != nil {
		return fmt.Errorf("ошибка объединения G5:I5: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "G5", "I5", styles.sectionTitle); err != nil {
		return fmt.Errorf("ошибка стиля G5: %w", err)
	}

	if err := f.SetCellValue(sheetName, "G6", "(график)"); err != nil {
		return fmt.Errorf("ошибка записи графика: %w", err)
	}
	if err := f.MergeCell(sheetName, "G6", "I12"); err != nil {
		return fmt.Errorf("ошибка объединения G6:I12: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "G6", "I12", styles.subText); err != nil {
		return fmt.Errorf("ошибка стиля G6: %w", err)
	}

	if err := f.SetCellValue(sheetName, "G14", "ОЦЕНКИ ТРЕНИРОВОК"); err != nil {
		return fmt.Errorf("ошибка записи заголовка оценок: %w", err)
	}
	if err := f.MergeCell(sheetName, "G14", "I14"); err != nil {
		return fmt.Errorf("ошибка объединения G14:I14: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "G14", "I14", styles.sectionTitle); err != nil {
		return fmt.Errorf("ошибка стиля G14: %w", err)
	}

	if err := f.SetCellValue(sheetName, "G15", "(график)"); err != nil {
		return fmt.Errorf("ошибка записи графика оценок: %w", err)
	}
	if err := f.MergeCell(sheetName, "G15", "I20"); err != nil {
		return fmt.Errorf("ошибка объединения G15:I20: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "G15", "I20", styles.subText); err != nil {
		return fmt.Errorf("ошибка стиля G15: %w", err)
	}

	return nil
}

func fillFeedbackTable(f *excelize.File, sheetName string, styles *dashboardStyles) error {
	if err := f.SetCellValue(sheetName, "A22", "ОБРАТНАЯ СВЯЗЬ"); err != nil {
		return fmt.Errorf("ошибка записи ОБРАТНАЯ СВЯЗЬ: %w", err)
	}
	if err := f.MergeCell(sheetName, "A22", "I22"); err != nil {
		return fmt.Errorf("ошибка объединения A22:I22: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "A22", "I22", styles.sectionTitle); err != nil {
		return fmt.Errorf("ошибка стиля A22: %w", err)
	}

	headers := []struct {
		col   string
		title string
	}{
		{"A", "ДАТА"}, {"B", "ТРЕНИРОВКА"}, {"C", "ОБРАТНАЯ СВЯЗЬ"},
		{"D", "ОЦЕНКА"}, {"E", "ДАТА ВЫП."}, {"F", "ВРЕМЯ"},
	}

	for _, h := range headers {
		cell := fmt.Sprintf("%s23", h.col)
		if err := f.SetCellValue(sheetName, cell, h.title); err != nil {
			return fmt.Errorf("ошибка записи заголовка %s: %w", h.title, err)
		}
		if err := f.SetCellStyle(sheetName, cell, cell, styles.tableHeader); err != nil {
			return fmt.Errorf("ошибка стиля заголовка %s: %w", h.title, err)
		}
	}

	for row := 24; row <= 33; row++ {
		for _, col := range []string{"A", "B", "C", "D", "E", "F"} {
			cell := fmt.Sprintf("%s%d", col, row)
			if err := f.SetCellStyle(sheetName, cell, cell, styles.tableCell); err != nil {
				return fmt.Errorf("ошибка стиля ячейки %s: %w", cell, err)
			}
		}
	}

	return nil
}

// CreateExercisesDataSheet создаёт лист с данными упражнений
func CreateExercisesDataSheet(f *excelize.File, sheetName string, clientID int) error {
	sheetIndex, err := f.GetSheetIndex(sheetName)
	if err != nil {
		return fmt.Errorf("ошибка проверки листа данных: %w", err)
	}
	if sheetIndex >= 0 {
		return nil
	}

	if _, err := f.NewSheet(sheetName); err != nil {
		return fmt.Errorf("ошибка создания листа данных: %w", err)
	}

	headers := []string{
		"Дата", "№ трен.", "Упражнение", "Подходы", "Повторения",
		"Вес (кг)", "Тоннаж", "Статус", "Обр.связь", "Оценка",
		"Дата вып.", "Время вып.", "Общ.тоннаж", "Тренировок", "Отправлено",
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return fmt.Errorf("ошибка создания стиля заголовков: %w", err)
	}

	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("ошибка преобразования координат: %w", err)
		}
		if err := f.SetCellValue(sheetName, cell, h); err != nil {
			return fmt.Errorf("ошибка записи заголовка %s: %w", h, err)
		}
	}
	if err := f.SetCellStyle(sheetName, "A1", "O1", headerStyle); err != nil {
		return fmt.Errorf("ошибка стиля заголовков: %w", err)
	}

	if err := f.SetCellValue(sheetName, "P1", "client_id"); err != nil {
		return fmt.Errorf("ошибка записи client_id label: %w", err)
	}
	if err := f.SetCellValue(sheetName, "P2", clientID); err != nil {
		return fmt.Errorf("ошибка записи client_id: %w", err)
	}

	if err := f.SetCellFormula(sheetName, "M2", "SUM(G:G)"); err != nil {
		return fmt.Errorf("ошибка формулы общего тоннажа: %w", err)
	}
	if err := f.SetCellFormula(sheetName, "N2", "MAX(B:B)"); err != nil {
		return fmt.Errorf("ошибка формулы количества тренировок: %w", err)
	}

	for row := 2; row <= 100; row++ {
		formula := fmt.Sprintf("IF(OR(D%d=\"\",E%d=\"\"),\"\",D%d*E%d*IF(F%d=\"\",0,F%d))", row, row, row, row, row, row)
		if err := f.SetCellFormula(sheetName, fmt.Sprintf("G%d", row), formula); err != nil {
			return fmt.Errorf("ошибка формулы тоннажа строки %d: %w", row, err)
		}
	}

	colWidths := []struct {
		start, end string
		width      float64
	}{
		{"A", "A", 12}, {"B", "B", 8}, {"C", "C", 25},
		{"D", "E", 10}, {"F", "G", 10}, {"H", "H", 14},
		{"I", "I", 20}, {"J", "L", 10},
	}
	for _, cw := range colWidths {
		if err := f.SetColWidth(sheetName, cw.start, cw.end, cw.width); err != nil {
			return fmt.Errorf("ошибка ширины колонки %s: %w", cw.start, err)
		}
	}

	statusValidation := excelize.NewDataValidation(true)
	statusValidation.Sqref = "H2:H1000"
	if err := statusValidation.SetDropList([]string{"запланировано", "в процессе", "выполнено", "пропущено", "перенесено"}); err != nil {
		return fmt.Errorf("ошибка настройки валидации статуса: %w", err)
	}
	if err := f.AddDataValidation(sheetName, statusValidation); err != nil {
		return fmt.Errorf("ошибка добавления валидации статуса: %w", err)
	}

	ratingValidation := excelize.NewDataValidation(true)
	ratingValidation.Sqref = "J2:J1000"
	if err := ratingValidation.SetRange(1, 10, excelize.DataValidationTypeWhole, excelize.DataValidationOperatorBetween); err != nil {
		return fmt.Errorf("ошибка настройки валидации оценки: %w", err)
	}
	if err := f.AddDataValidation(sheetName, ratingValidation); err != nil {
		return fmt.Errorf("ошибка добавления валидации оценки: %w", err)
	}

	log.Printf("Создан лист данных: %s", sheetName)
	return nil
}

// UpdateAllDashboards обновляет календари на всех dashboard листах
func UpdateAllDashboards(filePath string, db *sql.DB) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer f.Close()

	trainings, err := ReadAllClientTrainings(filePath, db)
	if err != nil {
		return fmt.Errorf("ошибка чтения тренировок: %w", err)
	}

	currentMonth := time.Now().Month()
	currentYear := time.Now().Year()

	sheetDays := make(map[string]map[int]*models.DayData)

	for _, t := range trainings {
		day := parseDayFromDate(t.Date, currentMonth, currentYear)
		if day == 0 {
			continue
		}

		dashboardSheet := t.SheetName
		if idx := len(dashboardSheet) - 5; idx > 0 && dashboardSheet[idx:] == "_data" {
			dashboardSheet = dashboardSheet[:idx]
		}

		if _, ok := sheetDays[dashboardSheet]; !ok {
			sheetDays[dashboardSheet] = make(map[int]*models.DayData)
		}

		if _, ok := sheetDays[dashboardSheet][day]; !ok {
			sheetDays[dashboardSheet][day] = &models.DayData{}
		}

		sheetDays[dashboardSheet][day].Tonnage += t.Tonnage
		if t.Status == "выполнено" || t.Status == "Выполнено" {
			sheetDays[dashboardSheet][day].Status = "done"
		} else if sheetDays[dashboardSheet][day].Status != "done" {
			sheetDays[dashboardSheet][day].Status = "planned"
		}
	}

	for sheetName, days := range sheetDays {
		if err := updateDashboardCalendar(f, sheetName, days); err != nil {
			log.Printf("Ошибка обновления календаря %s: %v", sheetName, err)
		}
	}

	return f.Save()
}

func parseDayFromDate(dateStr string, currentMonth time.Month, currentYear int) int {
	dateStr = trimString(dateStr)

	if serial, err := strconv.ParseFloat(dateStr, 64); err == nil && serial > 40000 {
		excelEpoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		t := excelEpoch.Add(time.Duration(serial) * 24 * time.Hour)
		if t.Month() == currentMonth && t.Year() == currentYear {
			return t.Day()
		}
		return 0
	}

	if t, err := time.Parse("02.01.2006", dateStr); err == nil {
		if t.Month() == currentMonth && t.Year() == currentYear {
			return t.Day()
		}
		return 0
	}

	if t, err := time.Parse("02.01", dateStr); err == nil {
		if t.Month() == currentMonth {
			return t.Day()
		}
		return 0
	}

	if day, err := strconv.Atoi(dateStr); err == nil && day >= 1 && day <= 31 {
		return day
	}

	return 0
}

func trimString(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
			result = append(result, s[i])
		} else if len(result) > 0 && result[len(result)-1] != ' ' {
			result = append(result, ' ')
		}
	}
	if len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return string(result)
}

func updateDashboardCalendar(f *excelize.File, sheetName string, days map[int]*models.DayData) error {
	idx, err := f.GetSheetIndex(sheetName)
	if err != nil || idx < 0 {
		return nil
	}

	calendarPlanned, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "#ffffff"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ea580c"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#0f172a", Style: 1},
			{Type: "right", Color: "#0f172a", Style: 1},
			{Type: "top", Color: "#0f172a", Style: 1},
			{Type: "bottom", Color: "#0f172a", Style: 1},
		},
	})

	calendarDone, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "#ffffff"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16a34a"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#0f172a", Style: 1},
			{Type: "right", Color: "#0f172a", Style: 1},
			{Type: "top", Color: "#0f172a", Style: 1},
			{Type: "bottom", Color: "#0f172a", Style: 1},
		},
	})

	tonnageStylePlanned, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#ffffff"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ea580c"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#0f172a", Style: 1},
			{Type: "right", Color: "#0f172a", Style: 1},
			{Type: "top", Color: "#0f172a", Style: 1},
			{Type: "bottom", Color: "#0f172a", Style: 1},
		},
	})

	tonnageStyleDone, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#ffffff"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#16a34a"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#0f172a", Style: 1},
			{Type: "right", Color: "#0f172a", Style: 1},
			{Type: "top", Color: "#0f172a", Style: 1},
			{Type: "bottom", Color: "#0f172a", Style: 1},
		},
	})

	for day, data := range days {
		var dateCol, tonnageCol string
		var row int

		if day <= 16 {
			dateCol, tonnageCol = "A", "B"
			row = 4 + day
		} else {
			dateCol, tonnageCol = "D", "E"
			row = 4 + (day - 16)
		}

		dateCell := fmt.Sprintf("%s%d", dateCol, row)
		tonnageCell := fmt.Sprintf("%s%d", tonnageCol, row)

		var dateStyle, tonnageStyle int
		if data.Status == "done" {
			dateStyle, tonnageStyle = calendarDone, tonnageStyleDone
		} else {
			dateStyle, tonnageStyle = calendarPlanned, tonnageStylePlanned
		}

		f.SetCellStyle(sheetName, dateCell, dateCell, dateStyle)

		if data.Tonnage > 0 {
			f.SetCellValue(sheetName, tonnageCell, fmt.Sprintf("%.0f", data.Tonnage))
		}
		f.SetCellStyle(sheetName, tonnageCell, tonnageCell, tonnageStyle)
	}

	return nil
}

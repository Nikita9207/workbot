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

const (
	SheetData = "Данные"
)

// Колонки основной таблицы данных
const (
	ColDate        = "A" // Дата
	ColClient      = "B" // Клиент
	ColExercise    = "C" // Упражнение
	ColSets        = "D" // Подходы
	ColReps        = "E" // Повторения
	ColWeight      = "F" // Вес
	ColTonnage     = "G" // Тоннаж
	ColTrainingNum = "H" // № тренировки
	ColStatus      = "I" // Статус
	ColRating      = "J" // Оценка
	ColComment     = "K" // Комментарий
	ColSent        = "L" // Отправлено
	ColDoneDate    = "M" // Дата выполнения
	ColClientID    = "N" // ID клиента (скрытая)

	// Колонки статистики справа
	ColStatLabel   = "P" // Метка статистики
	ColStatValue   = "Q" // Значение статистики
)

// unifiedStyles содержит стили для единой таблицы
type unifiedStyles struct {
	header       int
	dateCell     int
	textCell     int
	numberCell   int
	statusPlan   int
	statusDone   int
	statusMissed int
	sentYes      int
	sentNo       int
	statHeader   int
	statLabel    int
	statValue    int
}

// CreateUnifiedSheet создаёт единый лист с данными тренировок
func CreateUnifiedSheet(filePath string, db *sql.DB) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		// Создаём новый файл если не существует
		f = excelize.NewFile()
	}
	defer func() {
		if err := f.Save(); err != nil {
			log.Printf("Ошибка сохранения файла: %v", err)
		}
		f.Close()
	}()

	// Удаляем старый лист если есть
	if idx, _ := f.GetSheetIndex(SheetData); idx >= 0 {
		f.DeleteSheet(SheetData)
	}

	// Создаём новый лист
	sheetIndex, err := f.NewSheet(SheetData)
	if err != nil {
		return fmt.Errorf("ошибка создания листа: %w", err)
	}
	f.SetActiveSheet(sheetIndex)

	// Удаляем дефолтный лист
	f.DeleteSheet("Sheet1")

	styles, err := createUnifiedStyles(f)
	if err != nil {
		return fmt.Errorf("ошибка создания стилей: %w", err)
	}

	// Настраиваем ширину колонок
	if err := setupUnifiedColumns(f); err != nil {
		return err
	}

	// Создаём заголовки
	if err := createUnifiedHeaders(f, styles); err != nil {
		return err
	}

	// Создаём блок статистики справа
	if err := createStatisticsBlock(f, styles); err != nil {
		return err
	}

	// Добавляем выпадающие списки
	if err := addUnifiedValidations(f); err != nil {
		return err
	}

	// Добавляем условное форматирование
	if err := addConditionalFormatting(f); err != nil {
		log.Printf("Предупреждение: не удалось добавить условное форматирование: %v", err)
	}

	// Загружаем список клиентов для выпадающего списка
	if db != nil {
		if err := addClientDropdown(f, db); err != nil {
			log.Printf("Предупреждение: не удалось добавить список клиентов: %v", err)
		}
	}

	log.Printf("Создан единый лист данных: %s", SheetData)
	return nil
}

func createUnifiedStyles(f *excelize.File) (*unifiedStyles, error) {
	styles := &unifiedStyles{}
	var err error

	// Заголовок таблицы
	styles.header, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#1F4E79", Style: 1},
			{Type: "right", Color: "#1F4E79", Style: 1},
			{Type: "top", Color: "#1F4E79", Style: 1},
			{Type: "bottom", Color: "#1F4E79", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Ячейка с датой
	styles.dateCell, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		NumFmt:    14, // DD.MM.YYYY
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Текстовая ячейка
	styles.textCell, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Числовая ячейка
	styles.numberCell, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Статус: запланировано
	styles.statusPlan, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#9C5700"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFEB9C"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Статус: выполнено
	styles.statusDone, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Статус: пропущено
	styles.statusMissed, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#9C0006"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFC7CE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Отправлено: да
	styles.sentYes, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Отправлено: нет
	styles.sentNo, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#9C5700"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFEB9C"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Заголовок статистики
	styles.statHeader, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#507E32", Style: 2},
			{Type: "right", Color: "#507E32", Style: 2},
			{Type: "top", Color: "#507E32", Style: 2},
			{Type: "bottom", Color: "#507E32", Style: 2},
		},
	})
	if err != nil {
		return nil, err
	}

	// Метка статистики
	styles.statLabel, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "#404040"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#C6E0B4", Style: 1},
			{Type: "right", Color: "#C6E0B4", Style: 1},
			{Type: "bottom", Color: "#C6E0B4", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Значение статистики
	styles.statValue, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "#375623"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#C6E0B4", Style: 1},
			{Type: "right", Color: "#C6E0B4", Style: 1},
			{Type: "bottom", Color: "#C6E0B4", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	return styles, nil
}

func setupUnifiedColumns(f *excelize.File) error {
	columns := []struct {
		col   string
		width float64
	}{
		{ColDate, 12},        // Дата
		{ColClient, 20},      // Клиент
		{ColExercise, 30},    // Упражнение
		{ColSets, 10},        // Подходы
		{ColReps, 12},        // Повторения
		{ColWeight, 10},      // Вес
		{ColTonnage, 12},     // Тоннаж
		{ColTrainingNum, 10}, // № тренировки
		{ColStatus, 14},      // Статус
		{ColRating, 10},      // Оценка
		{ColComment, 25},     // Комментарий
		{ColSent, 12},        // Отправлено
		{ColDoneDate, 14},    // Дата выполнения
		{ColClientID, 8},     // ID клиента
		{"O", 3},             // Разделитель
		{ColStatLabel, 20},   // Метка статистики
		{ColStatValue, 15},   // Значение статистики
	}

	for _, c := range columns {
		if err := f.SetColWidth(SheetData, c.col, c.col, c.width); err != nil {
			return fmt.Errorf("ошибка установки ширины колонки %s: %w", c.col, err)
		}
	}

	// Скрываем колонку с ID клиента
	if err := f.SetColVisible(SheetData, ColClientID, false); err != nil {
		log.Printf("Предупреждение: не удалось скрыть колонку ID: %v", err)
	}

	return nil
}

func createUnifiedHeaders(f *excelize.File, styles *unifiedStyles) error {
	headers := []struct {
		col   string
		title string
	}{
		{ColDate, "Дата"},
		{ColClient, "Клиент"},
		{ColExercise, "Упражнение"},
		{ColSets, "Подходы"},
		{ColReps, "Повтор."},
		{ColWeight, "Вес (кг)"},
		{ColTonnage, "Тоннаж"},
		{ColTrainingNum, "№ трен."},
		{ColStatus, "Статус"},
		{ColRating, "Оценка"},
		{ColComment, "Комментарий"},
		{ColSent, "Отправл."},
		{ColDoneDate, "Дата вып."},
		{ColClientID, "ID"},
	}

	for _, h := range headers {
		cell := h.col + "1"
		if err := f.SetCellValue(SheetData, cell, h.title); err != nil {
			return fmt.Errorf("ошибка записи заголовка %s: %w", h.title, err)
		}
	}

	// Применяем стиль к заголовкам
	if err := f.SetCellStyle(SheetData, "A1", "N1", styles.header); err != nil {
		return fmt.Errorf("ошибка установки стиля заголовков: %w", err)
	}

	// Фиксируем первую строку
	if err := f.SetPanes(SheetData, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		log.Printf("Предупреждение: не удалось зафиксировать строку: %v", err)
	}

	// Устанавливаем высоту строки заголовков
	if err := f.SetRowHeight(SheetData, 1, 30); err != nil {
		return fmt.Errorf("ошибка установки высоты строки: %w", err)
	}

	// Добавляем формулу для тоннажа на строки 2-1000
	for row := 2; row <= 500; row++ {
		formula := fmt.Sprintf("IF(OR(%s%d=\"\",%s%d=\"\"),\"\",D%d*E%d*IF(F%d=\"\",0,F%d))",
			ColSets, row, ColReps, row, row, row, row, row)
		if err := f.SetCellFormula(SheetData, fmt.Sprintf("%s%d", ColTonnage, row), formula); err != nil {
			return fmt.Errorf("ошибка формулы тоннажа строки %d: %w", row, err)
		}
	}

	return nil
}

func createStatisticsBlock(f *excelize.File, styles *unifiedStyles) error {
	// Заголовок блока статистики
	if err := f.SetCellValue(SheetData, ColStatLabel+"1", "СТАТИСТИКА"); err != nil {
		return err
	}
	if err := f.MergeCell(SheetData, ColStatLabel+"1", ColStatValue+"1"); err != nil {
		return err
	}
	if err := f.SetCellStyle(SheetData, ColStatLabel+"1", ColStatValue+"1", styles.statHeader); err != nil {
		return err
	}

	stats := []struct {
		label   string
		formula string
	}{
		{"Всего тренировок", "MAX(H:H)"},
		{"Общий тоннаж (кг)", "SUM(G:G)"},
		{"Упражнений всего", "COUNTA(C:C)-1"},
		{"Выполнено", "COUNTIF(I:I,\"выполнено\")"},
		{"Запланировано", "COUNTIF(I:I,\"запланировано\")"},
		{"Пропущено", "COUNTIF(I:I,\"пропущено\")"},
		{"Отправлено", "COUNTIF(L:L,\"да\")"},
		{"Средняя оценка", "IFERROR(AVERAGE(J:J),\"-\")"},
	}

	for i, stat := range stats {
		row := i + 2
		labelCell := fmt.Sprintf("%s%d", ColStatLabel, row)
		valueCell := fmt.Sprintf("%s%d", ColStatValue, row)

		if err := f.SetCellValue(SheetData, labelCell, stat.label); err != nil {
			return err
		}
		if err := f.SetCellFormula(SheetData, valueCell, stat.formula); err != nil {
			return err
		}
		if err := f.SetCellStyle(SheetData, labelCell, labelCell, styles.statLabel); err != nil {
			return err
		}
		if err := f.SetCellStyle(SheetData, valueCell, valueCell, styles.statValue); err != nil {
			return err
		}
	}

	return nil
}

func addUnifiedValidations(f *excelize.File) error {
	// Валидация статуса
	statusValidation := excelize.NewDataValidation(true)
	statusValidation.Sqref = ColStatus + "2:" + ColStatus + "1000"
	if err := statusValidation.SetDropList([]string{"запланировано", "в процессе", "выполнено", "пропущено", "перенесено"}); err != nil {
		return err
	}
	if err := f.AddDataValidation(SheetData, statusValidation); err != nil {
		return err
	}

	// Валидация оценки (1-10)
	ratingValidation := excelize.NewDataValidation(true)
	ratingValidation.Sqref = ColRating + "2:" + ColRating + "1000"
	if err := ratingValidation.SetRange(1, 10, excelize.DataValidationTypeWhole, excelize.DataValidationOperatorBetween); err != nil {
		return err
	}
	if err := f.AddDataValidation(SheetData, ratingValidation); err != nil {
		return err
	}

	// Валидация "Отправлено"
	sentValidation := excelize.NewDataValidation(true)
	sentValidation.Sqref = ColSent + "2:" + ColSent + "1000"
	if err := sentValidation.SetDropList([]string{"да", "нет"}); err != nil {
		return err
	}
	if err := f.AddDataValidation(SheetData, sentValidation); err != nil {
		return err
	}

	return nil
}

func addConditionalFormatting(f *excelize.File) error {
	// Условное форматирование для статуса "выполнено"
	greenFormat, err := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// Условное форматирование для статуса "запланировано"
	yellowFormat, err := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#9C5700"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFEB9C"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// Условное форматирование для статуса "пропущено"
	redFormat, err := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#9C0006"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFC7CE"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// Применяем условное форматирование к колонке статуса
	statusRange := ColStatus + "2:" + ColStatus + "500"

	if err := f.SetConditionalFormat(SheetData, statusRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &greenFormat, Value: "\"выполнено\""},
	}); err != nil {
		return err
	}

	if err := f.SetConditionalFormat(SheetData, statusRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &yellowFormat, Value: "\"запланировано\""},
	}); err != nil {
		return err
	}

	if err := f.SetConditionalFormat(SheetData, statusRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &redFormat, Value: "\"пропущено\""},
	}); err != nil {
		return err
	}

	// Условное форматирование для колонки "Отправлено"
	sentRange := ColSent + "2:" + ColSent + "500"

	if err := f.SetConditionalFormat(SheetData, sentRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &greenFormat, Value: "\"да\""},
	}); err != nil {
		return err
	}

	if err := f.SetConditionalFormat(SheetData, sentRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &yellowFormat, Value: "\"нет\""},
	}); err != nil {
		return err
	}

	return nil
}

func addClientDropdown(f *excelize.File, db *sql.DB) error {
	rows, err := db.Query("SELECT name, surname FROM public.clients ORDER BY name, surname")
	if err != nil {
		return err
	}
	defer rows.Close()

	var clients []string
	for rows.Next() {
		var name, surname string
		if err := rows.Scan(&name, &surname); err != nil {
			continue
		}
		clients = append(clients, fmt.Sprintf("%s %s", name, surname))
	}

	if len(clients) == 0 {
		return nil
	}

	clientValidation := excelize.NewDataValidation(true)
	clientValidation.Sqref = ColClient + "2:" + ColClient + "1000"
	if err := clientValidation.SetDropList(clients); err != nil {
		return err
	}
	if err := f.AddDataValidation(SheetData, clientValidation); err != nil {
		return err
	}

	return nil
}

// ReadUnifiedTrainings читает тренировки из единого листа (Журнал)
func ReadUnifiedTrainings(filePath string) ([]models.ClientTraining, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Используем лист "Журнал" из гибридной таблицы
	sheetName := SheetJournal
	rows, err := f.GetRows(sheetName)
	if err != nil {
		// Fallback на старый лист если Журнал не найден
		rows, err = f.GetRows(SheetData)
		if err != nil {
			return nil, err
		}
		sheetName = SheetData
	}

	var trainings []models.ClientTraining
	for i, row := range rows {
		if i == 0 || len(row) < 8 {
			continue
		}

		// Колонки в Журнале (hybrid.go):
		// A(0)-Дата, B(1)-Клиент, C(2)-Формат, D(3)-№трен, E(4)-Упражнение,
		// F(5)-Подходы, G(6)-Повторы, H(7)-Вес, I(8)-Объём, J(9)-%1RM, K(10)-RPE,
		// L(11)-Отдых, M(12)-Факт вес, N(13)-Факт повт, O(14)-Статус, P(15)-Оценка,
		// Q(16)-Заметки, R(17)-Отправлено, S(18)-Дата вып., T(19)-ID
		exercise := getCell(row, 4) // E - Упражнение
		if exercise == "" {
			continue
		}

		clientID, _ := strconv.Atoi(getCell(row, 19))         // T - ID клиента
		trainingNum, _ := strconv.Atoi(getCell(row, 3))       // D - № тренировки
		sets, _ := strconv.Atoi(getCell(row, 5))              // F - Подходы
		reps, _ := strconv.Atoi(getCell(row, 6))              // G - Повторения
		weight, _ := strconv.ParseFloat(getCell(row, 7), 64)  // H - Вес
		tonnage, _ := strconv.ParseFloat(getCell(row, 8), 64) // I - Объём
		rating, _ := strconv.Atoi(getCell(row, 15))           // P - Оценка

		sentStr := getCell(row, 17) // R - Отправлено
		sent := sentStr == "да" || sentStr == "Да" || sentStr == "true"

		trainings = append(trainings, models.ClientTraining{
			SheetName:     sheetName,
			RowNum:        i + 1,
			ClientID:      clientID,
			Date:          getCell(row, 0),  // A - Дата
			TrainingNum:   trainingNum,
			Exercise:      exercise,
			Sets:          sets,
			Reps:          reps,
			Weight:        weight,
			Tonnage:       tonnage,
			Status:        getCell(row, 14), // O - Статус
			Feedback:      getCell(row, 16), // Q - Заметки
			Rating:        rating,
			CompletedDate: getCell(row, 18), // S - Дата выполнения
			Sent:          sent,
		})
	}

	return trainings, nil
}

// SaveTrainingToUnified сохраняет тренировку в лист Журнал
func SaveTrainingToUnified(filePath string, clientID int, clientName string, trainingDate time.Time, exercises []models.ExerciseInput) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer func() {
		if err := f.Save(); err != nil {
			log.Printf("Ошибка сохранения файла: %v", err)
		}
		f.Close()
	}()

	// Используем лист Журнал из гибридной таблицы
	sheetName := SheetJournal
	if idx, _ := f.GetSheetIndex(sheetName); idx < 0 {
		// Fallback на старый лист
		sheetName = SheetData
		if idx, _ := f.GetSheetIndex(sheetName); idx < 0 {
			return fmt.Errorf("лист не найден")
		}
	}

	// Определяем следующий номер тренировки для этого клиента
	// Колонки в Журнале: D(3)-№трен, T(19)-ID клиента
	nextTrainingNum := 1
	rows, err := f.GetRows(sheetName)
	if err == nil {
		for i := 1; i < len(rows); i++ {
			if len(rows[i]) > 19 {
				rowClientID, _ := strconv.Atoi(getCell(rows[i], 19)) // T - ID клиента
				if rowClientID == clientID {
					if num, _ := strconv.Atoi(getCell(rows[i], 3)); num >= nextTrainingNum { // D - № тренировки
						nextTrainingNum = num + 1
					}
				}
			}
		}
	}

	// Находим следующую пустую строку
	nextRow := len(rows) + 1
	if nextRow < 2 {
		nextRow = 2
	}

	dateStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 14})

	// Колонки в Журнале (hybrid.go):
	// A-Дата, B-Клиент, C-Формат, D-№трен, E-Упражнение, F-Подходы, G-Повторы, H-Вес,
	// I-Объём (формула), O-Статус, Q-Заметки, R-Отправлено, T-ID
	for i, ex := range exercises {
		row := nextRow + i

		// A - Дата
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), trainingDate); err != nil {
			return err
		}
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dateStyle)

		// B - Клиент
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), clientName); err != nil {
			return err
		}

		// C - Формат (онлайн по умолчанию)
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), "онлайн"); err != nil {
			return err
		}

		// D - № тренировки
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), nextTrainingNum); err != nil {
			return err
		}

		// E - Упражнение
		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), ex.Name); err != nil {
			return err
		}

		// F - Подходы
		if err := f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), ex.Sets); err != nil {
			return err
		}

		// G - Повторы
		if err := f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), ex.Reps); err != nil {
			return err
		}

		// H - Вес
		if ex.Weight > 0 {
			if err := f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), ex.Weight); err != nil {
				return err
			}
		}

		// O - Статус
		if err := f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), "запланировано"); err != nil {
			return err
		}

		// Q - Заметки
		if ex.Comment != "" {
			if err := f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), ex.Comment); err != nil {
				return err
			}
		}

		// R - Отправлено
		if err := f.SetCellValue(sheetName, fmt.Sprintf("R%d", row), "нет"); err != nil {
			return err
		}

		// T - ID клиента (скрытая колонка)
		if err := f.SetCellValue(sheetName, fmt.Sprintf("T%d", row), clientID); err != nil {
			return err
		}
	}

	return nil
}

// MarkUnifiedTrainingAsSent помечает тренировку как отправленную
func MarkUnifiedTrainingAsSent(filePath string, rowNum int) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Используем лист Журнал
	sheetName := SheetJournal
	if idx, _ := f.GetSheetIndex(sheetName); idx < 0 {
		sheetName = SheetData
	}

	// R - Отправлено (колонка 17 в hybrid.go)
	cell := fmt.Sprintf("R%d", rowNum)
	if err := f.SetCellValue(sheetName, cell, "да"); err != nil {
		return err
	}

	return f.Save()
}

// GetUnsentTrainingsUnified возвращает неотправленные тренировки клиента из единого листа
func GetUnsentTrainingsUnified(filePath string, clientID int) ([]models.TrainingGroup, error) {
	trainings, err := ReadUnifiedTrainings(filePath)
	if err != nil {
		return nil, err
	}

	groups := make(map[int]*models.TrainingGroup)

	for _, t := range trainings {
		if t.ClientID != clientID || t.Sent || t.Exercise == "" {
			continue
		}

		if groups[t.TrainingNum] == nil {
			groups[t.TrainingNum] = &models.TrainingGroup{
				SheetName:   t.SheetName,
				ClientID:    t.ClientID,
				TrainingNum: t.TrainingNum,
				Exercises:   []models.ClientTraining{},
			}
		}
		groups[t.TrainingNum].Exercises = append(groups[t.TrainingNum].Exercises, t)
	}

	var result []models.TrainingGroup
	for _, g := range groups {
		result = append(result, *g)
	}

	return result, nil
}

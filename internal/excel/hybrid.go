package excel

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

// Названия листов гибридной таблицы
const (
	SheetJournal      = "Журнал"
	SheetDashboard    = "Dashboard"
	SheetPeriod       = "Периодизация"
	SheetProgress     = "Прогресс"
	SheetReference    = "Справочники"
	SheetWorkout      = "Тренировка"  // лист для ввода и отправки тренировки клиенту
)

// CreateHybridWorkbook создаёт гибридную таблицу со всеми листами
func CreateHybridWorkbook(filePath string, db *sql.DB) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.SaveAs(filePath); err != nil {
			log.Printf("Ошибка сохранения файла: %v", err)
		}
		f.Close()
	}()

	// Создаём все листы
	sheets := []string{SheetJournal, SheetDashboard, SheetPeriod, SheetProgress, SheetReference, SheetWorkout}
	for i, name := range sheets {
		if i == 0 {
			f.SetSheetName("Sheet1", name)
		} else {
			f.NewSheet(name)
		}
	}

	// Заполняем каждый лист
	if err := createJournalSheet(f); err != nil {
		return fmt.Errorf("ошибка создания журнала: %w", err)
	}

	if err := createDashboardSheetHybrid(f); err != nil {
		return fmt.Errorf("ошибка создания dashboard: %w", err)
	}

	if err := createPeriodizationSheet(f); err != nil {
		return fmt.Errorf("ошибка создания периодизации: %w", err)
	}

	if err := createProgressSheet(f); err != nil {
		return fmt.Errorf("ошибка создания прогресса: %w", err)
	}

	if err := createReferenceSheet(f); err != nil {
		return fmt.Errorf("ошибка создания справочников: %w", err)
	}

	if err := createWorkoutSheet(f); err != nil {
		return fmt.Errorf("ошибка создания листа тренировки: %w", err)
	}

	// Добавляем выпадающий список клиентов если есть БД
	if db != nil {
		addClientDropdownHybrid(f, db)
	}

	f.SetActiveSheet(0)
	log.Printf("Создана гибридная таблица: %s", filePath)
	return nil
}

// ============ ЛИСТ 1: ЖУРНАЛ ТРЕНИРОВОК ============

func createJournalSheet(f *excelize.File) error {
	sheet := SheetJournal

	// Стиль заголовка
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#1F4E79", Style: 1},
			{Type: "right", Color: "#1F4E79", Style: 1},
			{Type: "top", Color: "#1F4E79", Style: 1},
			{Type: "bottom", Color: "#1F4E79", Style: 1},
		},
	})

	// Заголовки колонок
	// Формат: онлайн = отправлять клиенту, офлайн = только для тренера
	headers := []struct {
		col   string
		title string
		width float64
	}{
		{"A", "Дата", 12},
		{"B", "Клиент", 18},
		{"C", "Формат", 9},        // онлайн/офлайн
		{"D", "№ трен.", 8},
		{"E", "Упражнение", 25},
		{"F", "Подходы", 9},
		{"G", "Повторы", 9},
		{"H", "Вес (кг)", 10},
		{"I", "Объём", 10},
		{"J", "% 1RM", 8},
		{"K", "RPE", 6},
		{"L", "Отдых (с)", 10},
		{"M", "Факт вес", 10},
		{"N", "Факт повт", 10},
		{"O", "Статус", 12},
		{"P", "Оценка", 8},
		{"Q", "Заметки", 25},
		{"R", "Отправлено", 11},
		{"S", "Дата вып.", 12},
		{"T", "ID", 6},
	}

	for _, h := range headers {
		f.SetCellValue(sheet, h.col+"1", h.title)
		f.SetColWidth(sheet, h.col, h.col, h.width)
	}
	f.SetCellStyle(sheet, "A1", "T1", headerStyle)
	f.SetRowHeight(sheet, 1, 35)

	// Скрываем колонку ID
	f.SetColVisible(sheet, "T", false)

	// Формулы для объёма (F*G*H) - подходы * повторы * вес
	for row := 2; row <= 500; row++ {
		formula := fmt.Sprintf("IF(OR(F%d=\"\",G%d=\"\"),\"\",F%d*G%d*IF(H%d=\"\",0,H%d))", row, row, row, row, row, row)
		f.SetCellFormula(sheet, fmt.Sprintf("I%d", row), formula)
	}

	// Фиксируем первую строку
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Выпадающие списки
	// Формат тренировки (C) - онлайн отправляется клиенту, офлайн только для тренера
	formatVal := excelize.NewDataValidation(true)
	formatVal.Sqref = "C2:C1000"
	formatVal.SetDropList([]string{"онлайн", "офлайн"})
	f.AddDataValidation(sheet, formatVal)

	// Статус (O)
	statusVal := excelize.NewDataValidation(true)
	statusVal.Sqref = "O2:O1000"
	statusVal.SetDropList([]string{"запланировано", "в процессе", "выполнено", "пропущено", "перенесено"})
	f.AddDataValidation(sheet, statusVal)

	// RPE (K)
	rpeVal := excelize.NewDataValidation(true)
	rpeVal.Sqref = "K2:K1000"
	rpeVal.SetDropList([]string{"6", "7", "7.5", "8", "8.5", "9", "9.5", "10"})
	f.AddDataValidation(sheet, rpeVal)

	// Оценка (P) - от 1 до 10
	ratingVal := excelize.NewDataValidation(true)
	ratingVal.Sqref = "P2:P1000"
	ratingVal.SetRange(1, 10, excelize.DataValidationTypeWhole, excelize.DataValidationOperatorBetween)
	f.AddDataValidation(sheet, ratingVal)

	// Отправлено (R)
	sentVal := excelize.NewDataValidation(true)
	sentVal.Sqref = "R2:R1000"
	sentVal.SetDropList([]string{"да", "нет"})
	f.AddDataValidation(sheet, sentVal)

	// Условное форматирование для статуса и отправлено
	addStatusConditionalFormatting(f, sheet, "O2:O500")
	addStatusConditionalFormatting(f, sheet, "R2:R500")

	// Условное форматирование для формата (онлайн = зелёный, офлайн = синий)
	addFormatConditionalFormatting(f, sheet, "C2:C500")

	return nil
}

func addStatusConditionalFormatting(f *excelize.File, sheet, cellRange string) {
	greenFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
	})
	yellowFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#9C5700"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFEB9C"}, Pattern: 1},
	})
	redFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#9C0006"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFC7CE"}, Pattern: 1},
	})

	f.SetConditionalFormat(sheet, cellRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &greenFmt, Value: "\"выполнено\""},
	})
	f.SetConditionalFormat(sheet, cellRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &greenFmt, Value: "\"да\""},
	})
	f.SetConditionalFormat(sheet, cellRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &yellowFmt, Value: "\"запланировано\""},
	})
	f.SetConditionalFormat(sheet, cellRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &yellowFmt, Value: "\"нет\""},
	})
	f.SetConditionalFormat(sheet, cellRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &redFmt, Value: "\"пропущено\""},
	})
}

func addFormatConditionalFormatting(f *excelize.File, sheet, cellRange string) {
	// Онлайн = зелёный (отправляется клиенту)
	onlineFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
	})
	// Офлайн = синий (только для тренера)
	offlineFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#1F4E79", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#BDD7EE"}, Pattern: 1},
	})

	f.SetConditionalFormat(sheet, cellRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &onlineFmt, Value: "\"онлайн\""},
	})
	f.SetConditionalFormat(sheet, cellRange, []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &offlineFmt, Value: "\"офлайн\""},
	})
}

// ============ ЛИСТ 2: DASHBOARD С ГРАФИКАМИ ============

func createDashboardSheetHybrid(f *excelize.File) error {
	sheet := SheetDashboard

	// Единая ширина для ВСЕХ колонок
	const colWidth = 4.5
	for i := 1; i <= 30; i++ {
		col, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheet, col, col, colWidth)
	}

	// Стили
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "#1F4E79"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#507E32", Style: 2},
			{Type: "right", Color: "#507E32", Style: 2},
			{Type: "top", Color: "#507E32", Style: 2},
			{Type: "bottom", Color: "#507E32", Style: 2},
		},
	})

	statLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "#404040"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    []excelize.Border{{Type: "bottom", Color: "#C6E0B4", Style: 1}},
	})

	statValueStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "#375623"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:    []excelize.Border{{Type: "bottom", Color: "#C6E0B4", Style: 1}},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "#1F4E79"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#BDD7EE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#9BC2E6", Style: 1},
			{Type: "right", Color: "#9BC2E6", Style: 1},
			{Type: "top", Color: "#9BC2E6", Style: 1},
			{Type: "bottom", Color: "#9BC2E6", Style: 1},
		},
	})

	dataCellStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})

	// Заголовок (A2:L2, объединяем 12 колонок)
	f.SetCellValue(sheet, "A2", "DASHBOARD ТРЕНЕРА")
	f.MergeCell(sheet, "A2", "L2")
	f.SetCellStyle(sheet, "A2", "L2", titleStyle)
	f.SetRowHeight(sheet, 2, 30)

	// Инструкция
	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9, Italic: true, Color: "#666666"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})
	f.SetCellValue(sheet, "A3", "Заполняйте лист «Журнал» — все данные здесь считаются автоматически. Колонки: Дата, Клиент, №трен., Упражнение, Подходы, Повторы, Вес, и т.д.")
	f.MergeCell(sheet, "A3", "O3")
	f.SetCellStyle(sheet, "A3", "O3", instructionStyle)

	// ===== БЛОК 1: ОБЩАЯ СТАТИСТИКА (A4:G12) =====
	f.SetCellValue(sheet, "A4", "ОБЩАЯ СТАТИСТИКА")
	f.MergeCell(sheet, "A4", "G4")
	f.SetCellStyle(sheet, "A4", "G4", sectionStyle)

	// Колонки Журнала: A=Дата, B=Клиент, C=Формат, D=№трен, E=Упражнение, F=Подходы, G=Повторы, H=Вес, I=Объём, J=%1RM, K=RPE, L=Отдых, M=Факт вес, N=Факт повт, O=Статус, P=Оценка, Q=Заметки, R=Отправлено
	stats := []struct {
		label   string
		formula string
	}{
		{"Всего тренировок", "MAX(Журнал!D:D)"},
		{"Общий объём (кг)", "SUM(Журнал!I:I)"},
		{"Упражнений записано", "COUNTA(Журнал!E:E)-1"},
		{"Выполнено", "COUNTIF(Журнал!O:O,\"выполнено\")"},
		{"Запланировано", "COUNTIF(Журнал!O:O,\"запланировано\")"},
		{"Отправлено клиентам", "COUNTIF(Журнал!R:R,\"да\")"},
		{"Средняя оценка", "IFERROR(AVERAGE(Журнал!P:P),\"-\")"},
		{"Средний RPE", "IFERROR(AVERAGE(Журнал!K:K),\"-\")"},
	}

	for i, s := range stats {
		row := 5 + i
		// Label: A-E (5 cols), Value: F-G (2 cols)
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), s.label)
		f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), statLabelStyle)

		f.SetCellFormula(sheet, fmt.Sprintf("F%d", row), s.formula)
		f.MergeCell(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("G%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("G%d", row), statValueStyle)
	}

	// ===== БЛОК 2: СТАТИСТИКА ПО КЛИЕНТАМ (I4:O12) =====
	f.SetCellValue(sheet, "I4", "ПО КЛИЕНТАМ")
	f.MergeCell(sheet, "I4", "O4")
	f.SetCellStyle(sheet, "I4", "O4", sectionStyle)

	f.SetCellValue(sheet, "I5", "Выберите клиента:")
	f.MergeCell(sheet, "I5", "L5")
	f.SetCellStyle(sheet, "I5", "L5", statLabelStyle)

	f.SetCellValue(sheet, "M5", "(все)")
	f.MergeCell(sheet, "M5", "O5")
	f.SetCellStyle(sheet, "M5", "O5", statValueStyle)

	clientStats := []struct {
		label   string
		formula string
	}{
		{"Тренировок клиента", "IF(M5=\"(все)\",MAX(Журнал!D:D),SUMPRODUCT((Журнал!B:B=M5)*(Журнал!D:D=MAX(IF(Журнал!B:B=M5,Журнал!D:D)))))"},
		{"Объём клиента (кг)", "IF(M5=\"(все)\",SUM(Журнал!I:I),SUMIF(Журнал!B:B,M5,Журнал!I:I))"},
		{"Выполнено клиентом", "IF(M5=\"(все)\",COUNTIF(Журнал!O:O,\"выполнено\"),COUNTIFS(Журнал!B:B,M5,Журнал!O:O,\"выполнено\"))"},
		{"Средняя оценка", "IF(M5=\"(все)\",IFERROR(AVERAGE(Журнал!P:P),\"-\"),IFERROR(AVERAGEIF(Журнал!B:B,M5,Журнал!P:P),\"-\"))"},
	}

	for i, s := range clientStats {
		row := 6 + i
		// Label: I-L (4 cols), Value: M-O (3 cols)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), s.label)
		f.MergeCell(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("L%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("L%d", row), statLabelStyle)

		f.SetCellFormula(sheet, fmt.Sprintf("M%d", row), s.formula)
		f.MergeCell(sheet, fmt.Sprintf("M%d", row), fmt.Sprintf("O%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("M%d", row), fmt.Sprintf("O%d", row), statValueStyle)
	}

	// ===== БЛОК 3: ДАННЫЕ ДЛЯ ГРАФИКОВ (Q4:Z17) =====
	// Данные считаются автоматически из Журнала по номеру недели в году
	f.SetCellValue(sheet, "Q4", "ДАННЫЕ ДЛЯ ГРАФИКОВ (авто)")
	f.MergeCell(sheet, "Q4", "Z4")
	f.SetCellStyle(sheet, "Q4", "Z4", sectionStyle)

	// Заголовки (каждый по 2 колонки)
	chartHeaders := []struct {
		title    string
		startCol int
		endCol   int
	}{
		{"Неделя", 17, 18},    // Q-R
		{"Объём", 19, 20},     // S-T
		{"Интенс.", 21, 22},   // U-V
		{"Тренир.", 23, 24},   // W-X
		{"Ср.RPE", 25, 26},    // Y-Z
	}

	for _, h := range chartHeaders {
		startCol, _ := excelize.ColumnNumberToName(h.startCol)
		endCol, _ := excelize.ColumnNumberToName(h.endCol)
		f.SetCellValue(sheet, startCol+"5", h.title)
		f.MergeCell(sheet, startCol+"5", endCol+"5")
		f.SetCellStyle(sheet, startCol+"5", endCol+"5", headerStyle)
	}

	// Формулы для автоматического подсчёта по неделям
	// Используем IFERROR и проверку на непустую дату чтобы избежать #ЗНАЧ!
	// Колонки Журнала: A=Дата, I=Объём, J=%1RM, K=RPE, E=Упражнение
	for week := 1; week <= 12; week++ {
		row := 5 + week

		// Номер недели
		f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), week)
		f.MergeCell(sheet, fmt.Sprintf("Q%d", row), fmt.Sprintf("R%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("Q%d", row), fmt.Sprintf("R%d", row), dataCellStyle)

		// Объём = сумма объёма за эту неделю (с проверкой на пустые даты)
		volumeFormula := fmt.Sprintf("IFERROR(SUMPRODUCT((Журнал!A2:A500<>\"\")*(WEEKNUM(Журнал!A2:A500)=%d)*(Журнал!I2:I500)),0)", week)
		f.SetCellFormula(sheet, fmt.Sprintf("S%d", row), volumeFormula)
		f.MergeCell(sheet, fmt.Sprintf("S%d", row), fmt.Sprintf("T%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("S%d", row), fmt.Sprintf("T%d", row), dataCellStyle)

		// Интенсивность = средний %1RM за неделю
		intensFormula := fmt.Sprintf("IFERROR(ROUND(SUMPRODUCT((Журнал!A2:A500<>\"\")*(WEEKNUM(Журнал!A2:A500)=%d)*(Журнал!J2:J500))/SUMPRODUCT((Журнал!A2:A500<>\"\")*(WEEKNUM(Журнал!A2:A500)=%d)*(Журнал!J2:J500<>\"\")),0),0)", week, week)
		f.SetCellFormula(sheet, fmt.Sprintf("U%d", row), intensFormula)
		f.MergeCell(sheet, fmt.Sprintf("U%d", row), fmt.Sprintf("V%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("U%d", row), fmt.Sprintf("V%d", row), dataCellStyle)

		// Количество упражнений за неделю
		trainFormula := fmt.Sprintf("IFERROR(SUMPRODUCT((Журнал!A2:A500<>\"\")*(WEEKNUM(Журнал!A2:A500)=%d)*(Журнал!E2:E500<>\"\")),0)", week)
		f.SetCellFormula(sheet, fmt.Sprintf("W%d", row), trainFormula)
		f.MergeCell(sheet, fmt.Sprintf("W%d", row), fmt.Sprintf("X%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("W%d", row), fmt.Sprintf("X%d", row), dataCellStyle)

		// Средний RPE за неделю
		rpeFormula := fmt.Sprintf("IFERROR(ROUND(SUMPRODUCT((Журнал!A2:A500<>\"\")*(WEEKNUM(Журнал!A2:A500)=%d)*(Журнал!K2:K500))/SUMPRODUCT((Журнал!A2:A500<>\"\")*(WEEKNUM(Журнал!A2:A500)=%d)*(Журнал!K2:K500<>\"\")),1),0)", week, week)
		f.SetCellFormula(sheet, fmt.Sprintf("Y%d", row), rpeFormula)
		f.MergeCell(sheet, fmt.Sprintf("Y%d", row), fmt.Sprintf("Z%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("Y%d", row), fmt.Sprintf("Z%d", row), dataCellStyle)
	}

	// ===== ГРАФИК 1: Объём по неделям =====
	volumeChart := &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{
			{
				Name:       "Объём",
				Categories: fmt.Sprintf("%s!$Q$6:$Q$17", sheet),
				Values:     fmt.Sprintf("%s!$S$6:$S$17", sheet),
				Fill:       excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
			},
		},
		Title:     []excelize.RichTextRun{{Text: "Объём по неделям (кг)"}},
		PlotArea:  excelize.ChartPlotArea{ShowCatName: false, ShowVal: true},
		Legend:    excelize.ChartLegend{Position: "bottom"},
		Dimension: excelize.ChartDimension{Width: 480, Height: 290},
	}
	f.AddChart(sheet, "A20", volumeChart)

	// ===== ГРАФИК 2: Интенсивность =====
	intensityChart := &excelize.Chart{
		Type: excelize.Line,
		Series: []excelize.ChartSeries{
			{
				Name:       "Интенсивность %",
				Categories: fmt.Sprintf("%s!$Q$6:$Q$17", sheet),
				Values:     fmt.Sprintf("%s!$U$6:$U$17", sheet),
				Line:       excelize.ChartLine{Smooth: true, Width: 2.5},
				Marker:     excelize.ChartMarker{Symbol: "circle", Size: 8},
			},
		},
		Title:     []excelize.RichTextRun{{Text: "Динамика интенсивности"}},
		PlotArea:  excelize.ChartPlotArea{ShowVal: false},
		Legend:    excelize.ChartLegend{Position: "bottom"},
		Dimension: excelize.ChartDimension{Width: 480, Height: 290},
	}
	f.AddChart(sheet, "L20", intensityChart)

	// ===== БЛОК 4: ТОП УПРАЖНЕНИЯ (A38:I49) =====
	f.SetCellValue(sheet, "A38", "ТОП-10 УПРАЖНЕНИЙ ПО ОБЪЁМУ")
	f.MergeCell(sheet, "A38", "I38")
	f.SetCellStyle(sheet, "A38", "I38", sectionStyle)

	topHeaders := []struct {
		title    string
		startCol int
		endCol   int
	}{
		{"№", 1, 1},           // A
		{"Упражнение", 2, 5},  // B-E
		{"Общий объём", 6, 7}, // F-G
		{"Раз выполн.", 8, 9}, // H-I
	}

	for _, h := range topHeaders {
		startCol, _ := excelize.ColumnNumberToName(h.startCol)
		endCol, _ := excelize.ColumnNumberToName(h.endCol)
		f.SetCellValue(sheet, startCol+"39", h.title)
		f.MergeCell(sheet, startCol+"39", endCol+"39")
		f.SetCellStyle(sheet, startCol+"39", endCol+"39", headerStyle)
	}

	for i := 1; i <= 10; i++ {
		row := 39 + i
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataCellStyle)
		for _, h := range topHeaders[1:] {
			startCol, _ := excelize.ColumnNumberToName(h.startCol)
			endCol, _ := excelize.ColumnNumberToName(h.endCol)
			f.MergeCell(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row))
			f.SetCellStyle(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row), dataCellStyle)
		}
	}

	// ===== БЛОК 5: ПРОГРЕСС 1RM (K38:S45) =====
	f.SetCellValue(sheet, "K38", "ПРОГРЕСС 1RM (ключевые упражнения)")
	f.MergeCell(sheet, "K38", "S38")
	f.SetCellStyle(sheet, "K38", "S38", sectionStyle)

	rmHeaders := []struct {
		title    string
		startCol int
		endCol   int
	}{
		{"Упражнение", 11, 14}, // K-N
		{"Начало", 15, 16},     // O-P
		{"Текущий", 17, 18},    // Q-R
		{"%", 19, 19},          // S
	}

	for _, h := range rmHeaders {
		startCol, _ := excelize.ColumnNumberToName(h.startCol)
		endCol, _ := excelize.ColumnNumberToName(h.endCol)
		f.SetCellValue(sheet, startCol+"39", h.title)
		f.MergeCell(sheet, startCol+"39", endCol+"39")
		f.SetCellStyle(sheet, startCol+"39", endCol+"39", headerStyle)
	}

	keyExercises := []string{"Приседания", "Жим лёжа", "Становая тяга", "Жим стоя", "Тяга в наклоне", "Ягодичный мост"}
	for i, ex := range keyExercises {
		row := 40 + i
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), ex)
		for _, h := range rmHeaders {
			startCol, _ := excelize.ColumnNumberToName(h.startCol)
			endCol, _ := excelize.ColumnNumberToName(h.endCol)
			f.MergeCell(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row))
			f.SetCellStyle(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row), dataCellStyle)
		}
	}

	return nil
}

// ============ ЛИСТ 3: ПЕРИОДИЗАЦИЯ ============

func createPeriodizationSheet(f *excelize.File) error {
	sheet := SheetPeriod

	// Единая ширина для ВСЕХ колонок
	const colWidth = 4.5
	const labelCols = 5 // Количество колонок для названий (A-E объединяются)

	// Устанавливаем одинаковую ширину для всех колонок (A до BF)
	for i := 1; i <= 57; i++ { // 5 для названий + 52 недели
		col, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheet, col, col, colWidth)
	}

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "#1F4E79"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#BDD7EE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#9BC2E6", Style: 1},
			{Type: "right", Color: "#9BC2E6", Style: 1},
			{Type: "top", Color: "#9BC2E6", Style: 1},
			{Type: "bottom", Color: "#9BC2E6", Style: 1},
		},
	})

	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})

	weekCellStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})

	// Первая колонка недель (F = 6)
	firstWeekCol := labelCols + 1
	lastWeekColNum := labelCols + 52
	lastCol, _ := excelize.ColumnNumberToName(lastWeekColNum)

	// Заголовок
	f.SetCellValue(sheet, "A1", "ГОДОВОЙ ПЛАН ПЕРИОДИЗАЦИИ")
	f.MergeCell(sheet, "A1", lastCol+"1")
	f.SetCellStyle(sheet, "A1", lastCol+"1", titleStyle)
	f.SetRowHeight(sheet, 1, 25)

	// Инструкция по заполнению
	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9, Italic: true, Color: "#666666"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})
	instruction := "Как заполнять: отметьте «X» или закрасьте ячейки в строках ПЕРИОД/МЕЗОЦИКЛ/АКЦЕНТЫ для каждой недели. В НАГРУЗКА укажите % объёма и интенсивности (например: 70, 80). Внизу — шаблоны мезо- и микроциклов."
	f.SetCellValue(sheet, "A2", instruction)
	f.MergeCell(sheet, "A2", lastCol+"2")
	f.SetCellStyle(sheet, "A2", lastCol+"2", instructionStyle)

	// Строка месяцев (строка 4, т.к. строка 2 - инструкция, строка 3 - пустая)
	f.SetCellValue(sheet, "A4", "Месяц")
	labelEndCol, _ := excelize.ColumnNumberToName(labelCols)
	f.MergeCell(sheet, "A4", labelEndCol+"4")
	f.SetCellStyle(sheet, "A4", labelEndCol+"4", headerStyle)

	// Распределение месяцев (52 недели)
	monthWeeks := []struct {
		name  string
		weeks int
	}{
		{"Янв", 5}, {"Фев", 4}, {"Мар", 4}, {"Апр", 5}, {"Май", 4}, {"Июн", 4},
		{"Июл", 5}, {"Авг", 4}, {"Сен", 4}, {"Окт", 5}, {"Ноя", 4}, {"Дек", 4},
	}

	col := firstWeekCol
	for _, m := range monthWeeks {
		startCol, _ := excelize.ColumnNumberToName(col)
		endCol, _ := excelize.ColumnNumberToName(col + m.weeks - 1)
		f.SetCellValue(sheet, startCol+"4", m.name)
		if m.weeks > 1 {
			f.MergeCell(sheet, startCol+"4", endCol+"4")
		}
		f.SetCellStyle(sheet, startCol+"4", endCol+"4", headerStyle)
		col += m.weeks
	}

	// Строка недель (строка 5)
	f.SetCellValue(sheet, "A5", "Неделя")
	f.MergeCell(sheet, "A5", labelEndCol+"5")
	f.SetCellStyle(sheet, "A5", labelEndCol+"5", headerStyle)

	for week := 1; week <= 52; week++ {
		colName, _ := excelize.ColumnNumberToName(firstWeekCol + week - 1)
		f.SetCellValue(sheet, colName+"5", week)
		f.SetCellStyle(sheet, colName+"5", colName+"5", headerStyle)
	}

	// Секции периодизации (сдвинуты на 1 строку вниз из-за инструкции)
	sections := []struct {
		row    int
		title  string
		items  []string
		colors []string
	}{
		{7, "ПЕРИОД", []string{"Подготовительный", "Соревновательный", "Переходный"}, []string{"#70AD47", "#ED7D31", "#5B9BD5"}},
		{11, "МЕЗОЦИКЛ", []string{"Втягивающий", "Базовый", "Контрольно-подгот.", "Предсоревноват.", "Соревновательный", "Восстановительный"}, []string{"#A9D08E", "#70AD47", "#548235", "#ED7D31", "#C55A11", "#5B9BD5"}},
		{19, "НАГРУЗКА", []string{"Объём (%)", "Интенсивность (%)", "ОФП (%)", "СФП (%)"}, []string{"#BDD7EE", "#BDD7EE", "#BDD7EE", "#BDD7EE"}},
		{25, "АКЦЕНТЫ", []string{"Сила", "Мощность", "Гипертрофия", "Выносливость"}, []string{"#FF6B6B", "#FFE66D", "#4ECDC4", "#95E1D3"}},
	}

	for _, sec := range sections {
		// Заголовок секции
		secStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Size: 9, Color: "#FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		})
		f.SetCellValue(sheet, fmt.Sprintf("A%d", sec.row), sec.title)
		f.MergeCell(sheet, fmt.Sprintf("A%d", sec.row), fmt.Sprintf("%s%d", labelEndCol, sec.row))
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", sec.row), fmt.Sprintf("%s%d", labelEndCol, sec.row), secStyle)

		for i, item := range sec.items {
			row := sec.row + 1 + i

			// Название элемента (объединённые A-E)
			itemStyle, _ := f.NewStyle(&excelize.Style{
				Font:      &excelize.Font{Size: 9},
				Fill:      excelize.Fill{Type: "pattern", Color: []string{sec.colors[i%len(sec.colors)]}, Pattern: 1},
				Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
				Border: []excelize.Border{
					{Type: "left", Color: "#D9D9D9", Style: 1},
					{Type: "right", Color: "#D9D9D9", Style: 1},
					{Type: "top", Color: "#D9D9D9", Style: 1},
					{Type: "bottom", Color: "#D9D9D9", Style: 1},
				},
			})
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item)
			f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", labelEndCol, row))
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", labelEndCol, row), itemStyle)

			// Ячейки недель
			for week := 1; week <= 52; week++ {
				colName, _ := excelize.ColumnNumberToName(firstWeekCol + week - 1)
				f.SetCellStyle(sheet, fmt.Sprintf("%s%d", colName, row), fmt.Sprintf("%s%d", colName, row), weekCellStyle)
			}
		}
	}

	// ===== МЕЗОЦИКЛ (4 НЕДЕЛИ) =====
	mesoStartRow := 31

	mesoHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#1F4E79", Style: 1},
			{Type: "right", Color: "#1F4E79", Style: 1},
			{Type: "top", Color: "#1F4E79", Style: 1},
			{Type: "bottom", Color: "#1F4E79", Style: 1},
		},
	})

	dataCellStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})

	// Заголовок таблицы мезоцикла (объединяем 22 колонки: A-V)
	mesoEndCol, _ := excelize.ColumnNumberToName(22)
	f.SetCellValue(sheet, fmt.Sprintf("A%d", mesoStartRow), "ДЕТАЛИЗАЦИЯ МЕЗОЦИКЛА (4 недели)")
	f.MergeCell(sheet, fmt.Sprintf("A%d", mesoStartRow), fmt.Sprintf("%s%d", mesoEndCol, mesoStartRow))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", mesoStartRow), fmt.Sprintf("%s%d", mesoEndCol, mesoStartRow), titleStyle)

	// Заголовки колонок (каждая занимает несколько объединённых ячеек)
	mesoHeaders := []struct {
		title    string
		startCol int
		endCol   int
	}{
		{"Микроцикл", 1, 6},
		{"Объём %", 7, 9},
		{"Интенс. %", 10, 12},
		{"Направленность", 13, 16},
		{"Тренировок", 17, 19},
		{"Тестирование", 20, 22},
	}

	mesoHeaderRow := mesoStartRow + 1
	for _, h := range mesoHeaders {
		startCol, _ := excelize.ColumnNumberToName(h.startCol)
		endCol, _ := excelize.ColumnNumberToName(h.endCol)
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", startCol, mesoHeaderRow), h.title)
		f.MergeCell(sheet, fmt.Sprintf("%s%d", startCol, mesoHeaderRow), fmt.Sprintf("%s%d", endCol, mesoHeaderRow))
		f.SetCellStyle(sheet, fmt.Sprintf("%s%d", startCol, mesoHeaderRow), fmt.Sprintf("%s%d", endCol, mesoHeaderRow), mesoHeaderStyle)
	}
	f.SetRowHeight(sheet, mesoHeaderRow, 20)

	// Данные мезоцикла
	mesoWeeks := []struct {
		name   string
		volume string
		intens string
	}{
		{"Неделя 1 (втягивающий)", "60-70", "60-65"},
		{"Неделя 2 (нагрузочный)", "75-85", "70-75"},
		{"Неделя 3 (ударный)", "85-95", "80-85"},
		{"Неделя 4 (разгрузочный)", "40-50", "50-60"},
	}

	for i, w := range mesoWeeks {
		row := mesoHeaderRow + 1 + i
		// Применяем такую же структуру объединения как в заголовках
		for _, h := range mesoHeaders {
			startCol, _ := excelize.ColumnNumberToName(h.startCol)
			endCol, _ := excelize.ColumnNumberToName(h.endCol)
			f.MergeCell(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row))
			f.SetCellStyle(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row), dataCellStyle)
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), w.name)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), w.volume)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), w.intens)
		// Выравнивание слева для названия
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("F%d", row), labelStyle)
	}

	// ===== НЕДЕЛЬНЫЙ МИКРОЦИКЛ =====
	microStartRow := mesoHeaderRow + 6

	// Заголовки микроцикла
	microHeaders := []struct {
		title    string
		startCol int
		endCol   int
	}{
		{"День", 1, 4},
		{"Тип", 5, 7},
		{"Направление", 8, 10},
		{"Объём %", 11, 12},
		{"Интенс. %", 13, 14},
		{"Основные упражнения", 15, 19},
		{"Заметки", 20, 22},
	}

	f.SetCellValue(sheet, fmt.Sprintf("A%d", microStartRow), "НЕДЕЛЬНЫЙ МИКРОЦИКЛ")
	f.MergeCell(sheet, fmt.Sprintf("A%d", microStartRow), fmt.Sprintf("%s%d", mesoEndCol, microStartRow))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", microStartRow), fmt.Sprintf("%s%d", mesoEndCol, microStartRow), titleStyle)

	microHeaderRow := microStartRow + 1
	for _, h := range microHeaders {
		startCol, _ := excelize.ColumnNumberToName(h.startCol)
		endCol, _ := excelize.ColumnNumberToName(h.endCol)
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", startCol, microHeaderRow), h.title)
		f.MergeCell(sheet, fmt.Sprintf("%s%d", startCol, microHeaderRow), fmt.Sprintf("%s%d", endCol, microHeaderRow))
		f.SetCellStyle(sheet, fmt.Sprintf("%s%d", startCol, microHeaderRow), fmt.Sprintf("%s%d", endCol, microHeaderRow), mesoHeaderStyle)
	}
	f.SetRowHeight(sheet, microHeaderRow, 20)

	days := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}
	for i, d := range days {
		row := microHeaderRow + 1 + i
		// Объединяем ячейки по структуре заголовков
		for _, h := range microHeaders {
			startCol, _ := excelize.ColumnNumberToName(h.startCol)
			endCol, _ := excelize.ColumnNumberToName(h.endCol)
			f.MergeCell(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row))
			f.SetCellStyle(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row), dataCellStyle)
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), d)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), labelStyle)
	}

	// Выпадающие списки для микроцикла
	trainTypeVal := excelize.NewDataValidation(true)
	trainTypeVal.Sqref = fmt.Sprintf("E%d:G%d", microHeaderRow+1, microHeaderRow+7)
	trainTypeVal.SetDropList([]string{"Силовая", "Кардио", "ВИИТ", "Восст.", "Отдых"})
	f.AddDataValidation(sheet, trainTypeVal)

	directionVal := excelize.NewDataValidation(true)
	directionVal.Sqref = fmt.Sprintf("H%d:J%d", microHeaderRow+1, microHeaderRow+7)
	directionVal.SetDropList([]string{"Верх", "Низ", "Фуллбоди", "Push", "Pull"})
	f.AddDataValidation(sheet, directionVal)

	return nil
}

// ============ ЛИСТ 4: ПРОГРЕСС КЛИЕНТОВ ============

func createProgressSheet(f *excelize.File) error {
	sheet := SheetProgress

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#548235"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#375623", Style: 1},
			{Type: "right", Color: "#375623", Style: 1},
			{Type: "top", Color: "#375623", Style: 1},
			{Type: "bottom", Color: "#375623", Style: 1},
		},
	})

	// ===== ПРОГРЕСС 1RM =====
	f.SetCellValue(sheet, "A1", "ОТСЛЕЖИВАНИЕ ПРОГРЕССА ПО УПРАЖНЕНИЯМ")
	f.MergeCell(sheet, "A1", "G1")
	f.SetCellStyle(sheet, "A1", "G1", titleStyle)
	f.SetRowHeight(sheet, 1, 30)

	rmHeaders := []string{"Клиент", "Упражнение", "Начальный 1RM", "Текущий 1RM", "Прирост (кг)", "Прирост %", "Цель 1RM"}
	for i, h := range rmHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(sheet, col+"3", h)
		f.SetColWidth(sheet, col, col, 15)
	}
	f.SetCellStyle(sheet, "A3", "G3", headerStyle)

	// Формулы для прироста
	for row := 4; row <= 50; row++ {
		// Прирост (кг) = Текущий - Начальный
		f.SetCellFormula(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("IF(OR(C%d=\"\",D%d=\"\"),\"\",D%d-C%d)", row, row, row, row))
		// Прирост % = (Текущий - Начальный) / Начальный * 100
		f.SetCellFormula(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("IF(OR(C%d=\"\",D%d=\"\"),\"\",ROUND((D%d-C%d)/C%d*100,1))", row, row, row, row, row))
	}

	// ===== АНТРОПОМЕТРИЯ =====
	f.SetCellValue(sheet, "I1", "АНТРОПОМЕТРИЧЕСКИЕ ЗАМЕРЫ")
	f.MergeCell(sheet, "I1", "Q1")
	f.SetCellStyle(sheet, "I1", "Q1", titleStyle)

	anthroHeaders := []string{"Клиент", "Дата", "Вес (кг)", "Грудь", "Талия", "Бёдра", "Бицепс", "Голень", "% жира"}
	for i, h := range anthroHeaders {
		col, _ := excelize.ColumnNumberToName(9 + i)
		f.SetCellValue(sheet, col+"3", h)
		f.SetColWidth(sheet, col, col, 10)
	}
	f.SetCellStyle(sheet, "I3", "Q3", headerStyle)

	// ===== ИСТОРИЯ ВЕСА ДЛЯ ГРАФИКА =====
	f.SetCellValue(sheet, "A55", "ИСТОРИЯ ВЕСА КЛИЕНТОВ (для графика)")
	f.MergeCell(sheet, "A55", "E55")

	weightHeaders := []string{"Клиент", "Дата", "Вес (кг)", "Цель", "До цели"}
	for i, h := range weightHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(sheet, col+"56", h)
	}

	return nil
}

// ============ ЛИСТ 5: СПРАВОЧНИКИ ============

func createReferenceSheet(f *excelize.File) error {
	sheet := SheetReference

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#5B9BD5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// ===== ТАБЛИЦА %1RM =====
	f.SetCellValue(sheet, "A1", "ТАБЛИЦА ИНТЕНСИВНОСТИ (% от 1RM)")
	f.MergeCell(sheet, "A1", "L1")
	f.SetCellStyle(sheet, "A1", "L1", titleStyle)
	f.SetRowHeight(sheet, 1, 25)

	// Заголовки процентов
	percentages := []int{50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100}
	f.SetCellValue(sheet, "A3", "1RM (кг)")
	for i, p := range percentages {
		col, _ := excelize.ColumnNumberToName(i + 2)
		f.SetCellValue(sheet, col+"3", fmt.Sprintf("%d%%", p))
	}
	f.SetCellStyle(sheet, "A3", "L3", headerStyle)

	// Значения 1RM от 40 до 250 с шагом 10
	row := 4
	for rm := 40; rm <= 250; rm += 10 {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), rm)
		for i, p := range percentages {
			col, _ := excelize.ColumnNumberToName(i + 2)
			value := float64(rm) * float64(p) / 100
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), value)
		}
		row++
	}

	f.SetColWidth(sheet, "A", "L", 8)

	// ===== ШКАЛА RPE =====
	f.SetCellValue(sheet, "N1", "ШКАЛА RPE (Воспринимаемая нагрузка)")
	f.MergeCell(sheet, "N1", "Q1")
	f.SetCellStyle(sheet, "N1", "Q1", titleStyle)

	rpeHeaders := []string{"RPE", "RIR", "% 1RM", "Описание"}
	for i, h := range rpeHeaders {
		col, _ := excelize.ColumnNumberToName(14 + i)
		f.SetCellValue(sheet, col+"3", h)
	}
	f.SetCellStyle(sheet, "N3", "Q3", headerStyle)

	rpeData := [][]interface{}{
		{10, 0, "100%", "Максимум, отказ"},
		{9.5, 0.5, "97%", "Возможно ещё 0.5 повт."},
		{9, 1, "95%", "Точно можно ещё 1 повт."},
		{8.5, 1.5, "92%", "1-2 повторения в запасе"},
		{8, 2, "90%", "Можно ещё 2 повторения"},
		{7.5, 2.5, "87%", "2-3 повторения в запасе"},
		{7, 3, "85%", "Можно ещё 3 повторения"},
		{6, 4, "80%", "Лёгкая нагрузка, 4+ в запасе"},
		{5, 5, "75%", "Разминочный вес"},
	}

	for i, data := range rpeData {
		row := 4 + i
		for j, val := range data {
			col, _ := excelize.ColumnNumberToName(14 + j)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), val)
		}
	}

	f.SetColWidth(sheet, "N", "N", 6)
	f.SetColWidth(sheet, "O", "O", 6)
	f.SetColWidth(sheet, "P", "P", 8)
	f.SetColWidth(sheet, "Q", "Q", 28)

	// ===== СПИСОК УПРАЖНЕНИЙ ПО ГРУППАМ =====
	f.SetCellValue(sheet, "A28", "УПРАЖНЕНИЯ ПО ГРУППАМ МЫШЦ")
	f.MergeCell(sheet, "A28", "F28")
	f.SetCellStyle(sheet, "A28", "F28", titleStyle)

	muscleGroups := []struct {
		name      string
		exercises []string
	}{
		{"Грудь", []string{"Жим лёжа", "Жим гантелей", "Разводка", "Отжимания", "Кроссовер"}},
		{"Спина", []string{"Становая тяга", "Тяга в наклоне", "Подтягивания", "Тяга верхнего блока", "Тяга горизонтального блока"}},
		{"Ноги", []string{"Приседания", "Жим ногами", "Выпады", "Ягодичный мост", "Разгибания ног", "Сгибания ног", "Подъёмы на носки"}},
		{"Плечи", []string{"Жим стоя", "Жим Арнольда", "Махи в стороны", "Махи вперёд", "Тяга к подбородку"}},
		{"Руки", []string{"Подъём на бицепс", "Французский жим", "Молотки", "Разгибания на блоке", "Сгибания Зоттмана"}},
		{"Кор", []string{"Планка", "Скручивания", "Подъём ног", "Русский твист", "Гиперэкстензия"}},
	}

	col := 1
	for _, group := range muscleGroups {
		colName, _ := excelize.ColumnNumberToName(col)

		groupStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 10, Color: "#FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center"},
		})
		f.SetCellValue(sheet, colName+"30", group.name)
		f.SetCellStyle(sheet, colName+"30", colName+"30", groupStyle)

		for i, ex := range group.exercises {
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName, 31+i), ex)
		}
		f.SetColWidth(sheet, colName, colName, 22)
		col++
	}

	return nil
}

// Добавление выпадающего списка клиентов
func addClientDropdownHybrid(f *excelize.File, db *sql.DB) {
	rows, err := db.Query("SELECT name, surname FROM public.clients ORDER BY name, surname")
	if err != nil {
		return
	}
	defer rows.Close()

	var clients []string
	clients = append(clients, "(все)")
	for rows.Next() {
		var name, surname string
		if err := rows.Scan(&name, &surname); err != nil {
			continue
		}
		clients = append(clients, fmt.Sprintf("%s %s", name, surname))
	}

	if len(clients) <= 1 {
		return
	}

	// Журнал - колонка Клиент
	clientVal := excelize.NewDataValidation(true)
	clientVal.Sqref = "B2:B1000"
	clientVal.SetDropList(clients[1:]) // без "(все)"
	f.AddDataValidation(SheetJournal, clientVal)

	// Dashboard - выбор клиента для фильтрации (ячейка M5)
	dashboardVal := excelize.NewDataValidation(true)
	dashboardVal.Sqref = "M5"
	dashboardVal.SetDropList(clients)
	f.AddDataValidation(SheetDashboard, dashboardVal)
}

// ============ ЛИСТ 6: ТРЕНИРОВКА (ввод и отправка) ============

func createWorkoutSheet(f *excelize.File) error {
	sheet := SheetWorkout

	// Единая ширина для колонок
	const colWidth = 4.5
	for i := 1; i <= 20; i++ {
		col, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheet, col, col, colWidth)
	}

	// Стили
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ED7D31"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#C55A11"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#843C0C", Style: 1},
			{Type: "right", Color: "#843C0C", Style: 1},
			{Type: "top", Color: "#843C0C", Style: 1},
			{Type: "bottom", Color: "#843C0C", Style: 1},
		},
	})

	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#404040"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FCE4D6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#F4B084", Style: 1},
			{Type: "right", Color: "#F4B084", Style: 1},
			{Type: "top", Color: "#F4B084", Style: 1},
			{Type: "bottom", Color: "#F4B084", Style: 1},
		},
	})

	inputStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFFFFF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#ED7D31", Style: 2},
			{Type: "right", Color: "#ED7D31", Style: 2},
			{Type: "top", Color: "#ED7D31", Style: 2},
			{Type: "bottom", Color: "#ED7D31", Style: 2},
		},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#C55A11"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#843C0C", Style: 1},
			{Type: "right", Color: "#843C0C", Style: 1},
			{Type: "top", Color: "#843C0C", Style: 1},
			{Type: "bottom", Color: "#843C0C", Style: 1},
		},
	})

	exerciseStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
		},
	})

	// Заголовок листа
	f.SetCellValue(sheet, "A1", "ТРЕНИРОВКА ДЛЯ ОТПРАВКИ КЛИЕНТУ")
	f.MergeCell(sheet, "A1", "T1")
	f.SetCellStyle(sheet, "A1", "T1", titleStyle)
	f.SetRowHeight(sheet, 1, 30)

	// Инструкция
	instructionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9, Italic: true, Color: "#666666"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})
	instruction := "Заполните тренировку здесь. Для онлайн-клиентов используйте команду «Отправить» в боте — тренировка будет отправлена клиенту. Для офлайн-клиентов вы видите тренировку только в этой таблице."
	f.SetCellValue(sheet, "A2", instruction)
	f.MergeCell(sheet, "A2", "T2")
	f.SetCellStyle(sheet, "A2", "T2", instructionStyle)
	f.SetRowHeight(sheet, 2, 30)

	// ===== БЛОК ИНФОРМАЦИИ О ТРЕНИРОВКЕ (строки 4-7) =====
	f.SetCellValue(sheet, "A4", "ИНФОРМАЦИЯ О ТРЕНИРОВКЕ")
	f.MergeCell(sheet, "A4", "J4")
	f.SetCellStyle(sheet, "A4", "J4", sectionStyle)

	// Дата тренировки
	f.SetCellValue(sheet, "A5", "Дата:")
	f.MergeCell(sheet, "A5", "C5")
	f.SetCellStyle(sheet, "A5", "C5", labelStyle)
	f.MergeCell(sheet, "D5", "F5")
	f.SetCellStyle(sheet, "D5", "F5", inputStyle)

	// Номер тренировки
	f.SetCellValue(sheet, "G5", "№ трен:")
	f.MergeCell(sheet, "G5", "H5")
	f.SetCellStyle(sheet, "G5", "H5", labelStyle)
	f.MergeCell(sheet, "I5", "J5")
	f.SetCellStyle(sheet, "I5", "J5", inputStyle)

	// Тип тренировки
	f.SetCellValue(sheet, "A6", "Тип:")
	f.MergeCell(sheet, "A6", "C6")
	f.SetCellStyle(sheet, "A6", "C6", labelStyle)
	f.MergeCell(sheet, "D6", "F6")
	f.SetCellStyle(sheet, "D6", "F6", inputStyle)

	// Направленность
	f.SetCellValue(sheet, "G6", "Направл:")
	f.MergeCell(sheet, "G6", "H6")
	f.SetCellStyle(sheet, "G6", "H6", labelStyle)
	f.MergeCell(sheet, "I6", "J6")
	f.SetCellStyle(sheet, "I6", "J6", inputStyle)

	// Примечание к тренировке
	f.SetCellValue(sheet, "A7", "Примечание:")
	f.MergeCell(sheet, "A7", "C7")
	f.SetCellStyle(sheet, "A7", "C7", labelStyle)
	f.MergeCell(sheet, "D7", "J7")
	f.SetCellStyle(sheet, "D7", "J7", inputStyle)

	// ===== СТАТУС ОТПРАВКИ (сразу после инфо, колонки K-O) =====
	// Расширяем колонку K для лейблов
	f.SetColWidth(sheet, "K", "K", 12)

	statusTitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	f.SetCellValue(sheet, "K4", "СТАТУС")
	f.MergeCell(sheet, "K4", "O4")
	f.SetCellStyle(sheet, "K4", "O4", statusTitleStyle)

	statusLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    []excelize.Border{{Type: "bottom", Color: "#C6E0B4", Style: 1}},
	})

	statusValueStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    []excelize.Border{{Type: "bottom", Color: "#C6E0B4", Style: 1}},
	})

	f.SetCellValue(sheet, "K5", "Отправлено:")
	f.MergeCell(sheet, "K5", "L5")
	f.SetCellStyle(sheet, "K5", "L5", statusLabelStyle)
	f.SetCellValue(sheet, "M5", "нет")
	f.MergeCell(sheet, "M5", "O5")
	f.SetCellStyle(sheet, "M5", "O5", statusValueStyle)

	f.SetCellValue(sheet, "K6", "Дата отпр:")
	f.MergeCell(sheet, "K6", "L6")
	f.SetCellStyle(sheet, "K6", "L6", statusLabelStyle)
	f.MergeCell(sheet, "M6", "O6")
	f.SetCellStyle(sheet, "M6", "O6", statusValueStyle)

	f.SetCellValue(sheet, "K7", "Выполнено:")
	f.MergeCell(sheet, "K7", "L7")
	f.SetCellStyle(sheet, "K7", "L7", statusLabelStyle)
	f.SetCellValue(sheet, "M7", "нет")
	f.MergeCell(sheet, "M7", "O7")
	f.SetCellStyle(sheet, "M7", "O7", statusValueStyle)

	// ===== ТАБЛИЦА УПРАЖНЕНИЙ (строки 9-18) =====
	f.SetCellValue(sheet, "A9", "УПРАЖНЕНИЯ")
	f.MergeCell(sheet, "A9", "T9")
	f.SetCellStyle(sheet, "A9", "T9", sectionStyle)

	// Заголовки таблицы упражнений
	exerciseHeaders := []struct {
		title    string
		startCol int
		endCol   int
	}{
		{"№", 1, 1},              // A
		{"Упражнение", 2, 6},     // B-F (5 колонок)
		{"Подходы", 7, 8},        // G-H
		{"Повторы", 9, 10},       // I-J
		{"Вес (кг)", 11, 12},     // K-L
		{"Отдых (с)", 13, 14},    // M-N
		{"% 1RM", 15, 16},        // O-P
		{"RPE", 17, 17},          // Q
		{"Заметки", 18, 20},      // R-T
	}

	for _, h := range exerciseHeaders {
		startCol, _ := excelize.ColumnNumberToName(h.startCol)
		endCol, _ := excelize.ColumnNumberToName(h.endCol)
		f.SetCellValue(sheet, startCol+"10", h.title)
		if h.startCol != h.endCol {
			f.MergeCell(sheet, startCol+"10", endCol+"10")
		}
		f.SetCellStyle(sheet, startCol+"10", endCol+"10", headerStyle)
	}
	f.SetRowHeight(sheet, 10, 22)

	// Строки для ввода упражнений (8 строк: 11-18)
	for i := 1; i <= 8; i++ {
		row := 10 + i
		// Номер упражнения
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), exerciseStyle)

		// Применяем структуру объединения как в заголовках
		for _, h := range exerciseHeaders[1:] {
			startCol, _ := excelize.ColumnNumberToName(h.startCol)
			endCol, _ := excelize.ColumnNumberToName(h.endCol)
			if h.startCol != h.endCol {
				f.MergeCell(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row))
			}
			f.SetCellStyle(sheet, fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("%s%d", endCol, row), exerciseStyle)
		}
	}

	// ===== РАЗМИНКА (строки 20-24) =====
	warmupStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#5B9BD5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	f.SetCellValue(sheet, "A20", "РАЗМИНКА")
	f.MergeCell(sheet, "A20", "J20")
	f.SetCellStyle(sheet, "A20", "J20", warmupStyle)

	warmupInputStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DEEBF7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#9BC2E6", Style: 1},
			{Type: "right", Color: "#9BC2E6", Style: 1},
			{Type: "top", Color: "#9BC2E6", Style: 1},
			{Type: "bottom", Color: "#9BC2E6", Style: 1},
		},
	})

	for row := 21; row <= 24; row++ {
		f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("J%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("J%d", row), warmupInputStyle)
	}

	// ===== ЗАМИНКА (строки 20-24, справа) =====
	cooldownStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	f.SetCellValue(sheet, "L20", "ЗАМИНКА / РАСТЯЖКА")
	f.MergeCell(sheet, "L20", "T20")
	f.SetCellStyle(sheet, "L20", "T20", cooldownStyle)

	cooldownInputStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#C6E0B4", Style: 1},
			{Type: "right", Color: "#C6E0B4", Style: 1},
			{Type: "top", Color: "#C6E0B4", Style: 1},
			{Type: "bottom", Color: "#C6E0B4", Style: 1},
		},
	})

	for row := 21; row <= 24; row++ {
		f.MergeCell(sheet, fmt.Sprintf("L%d", row), fmt.Sprintf("T%d", row))
		f.SetCellStyle(sheet, fmt.Sprintf("L%d", row), fmt.Sprintf("T%d", row), cooldownInputStyle)
	}

	// Выпадающие списки
	// Тип тренировки (D6)
	typeVal := excelize.NewDataValidation(true)
	typeVal.Sqref = "D6:F6"
	typeVal.SetDropList([]string{"Силовая", "Кардио", "ВИИТ", "Функциональная", "Восстановительная"})
	f.AddDataValidation(sheet, typeVal)

	// Направленность (I6)
	dirVal := excelize.NewDataValidation(true)
	dirVal.Sqref = "I6:J6"
	dirVal.SetDropList([]string{"Верх", "Низ", "Фуллбоди", "Push", "Pull", "Core"})
	f.AddDataValidation(sheet, dirVal)

	// Статус отправки (M5:O5)
	sentVal := excelize.NewDataValidation(true)
	sentVal.Sqref = "M5:O5"
	sentVal.SetDropList([]string{"да", "нет"})
	f.AddDataValidation(sheet, sentVal)

	// Статус выполнения (M7:O7)
	doneVal := excelize.NewDataValidation(true)
	doneVal.Sqref = "M7:O7"
	doneVal.SetDropList([]string{"да", "нет", "частично"})
	f.AddDataValidation(sheet, doneVal)

	// RPE для упражнений (Q11:Q18)
	rpeVal := excelize.NewDataValidation(true)
	rpeVal.Sqref = "Q11:Q18"
	rpeVal.SetDropList([]string{"6", "7", "7.5", "8", "8.5", "9", "9.5", "10"})
	f.AddDataValidation(sheet, rpeVal)

	// Условное форматирование для статуса отправки
	sentYesFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#006100", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
	})
	sentNoFmt, _ := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#9C5700", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFEB9C"}, Pattern: 1},
	})

	f.SetConditionalFormat(sheet, "M5:O5", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &sentYesFmt, Value: "\"да\""},
	})
	f.SetConditionalFormat(sheet, "M5:O5", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &sentNoFmt, Value: "\"нет\""},
	})

	f.SetConditionalFormat(sheet, "M7:O7", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &sentYesFmt, Value: "\"да\""},
	})
	f.SetConditionalFormat(sheet, "M7:O7", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Format: &sentNoFmt, Value: "\"нет\""},
	})

	return nil
}

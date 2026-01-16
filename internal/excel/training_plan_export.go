package excel

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"workbot/internal/models"
)

// Excel sheet names
const (
	SheetPlanOverview   = "Обзор плана"
	SheetPeriodization  = "Периодизация"
	SheetProgression    = "Прогрессия весов"
	Sheet1PMHistory     = "История 1ПМ"
	SheetVolumeAnalysis = "Анализ объёма"
)

// ExportTrainingPlan creates a comprehensive Excel workbook for a training plan
func ExportTrainingPlan(
	plan *models.TrainingPlan,
	progression []models.Progression,
	pm1History []models.Exercise1PMHistory,
) (*excelize.File, error) {
	f := excelize.NewFile()

	// Create sheets
	f.SetSheetName("Sheet1", SheetPlanOverview)
	f.NewSheet(SheetPeriodization)
	f.NewSheet(SheetProgression)
	f.NewSheet(Sheet1PMHistory)
	f.NewSheet(SheetVolumeAnalysis)

	// Fill sheets
	if err := createPlanOverviewSheetExport(f, plan); err != nil {
		return nil, fmt.Errorf("ошибка создания обзора: %w", err)
	}

	if err := createPeriodizationSheetExport(f, plan); err != nil {
		return nil, fmt.Errorf("ошибка создания периодизации: %w", err)
	}

	if err := createProgressionSheetExport(f, plan, progression); err != nil {
		return nil, fmt.Errorf("ошибка создания прогрессии: %w", err)
	}

	if err := create1PMHistorySheetExport(f, pm1History); err != nil {
		return nil, fmt.Errorf("ошибка создания истории 1ПМ: %w", err)
	}

	if err := createVolumeAnalysisSheetExport(f, plan, progression); err != nil {
		return nil, fmt.Errorf("ошибка создания анализа объёма: %w", err)
	}

	f.SetActiveSheet(0)
	return f, nil
}

// createPlanOverviewSheetExport creates the plan overview sheet
func createPlanOverviewSheetExport(f *excelize.File, plan *models.TrainingPlan) error {
	sheet := SheetPlanOverview

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Title
	f.SetCellValue(sheet, "A1", "ТРЕНИРОВОЧНЫЙ ПЛАН")
	f.MergeCell(sheet, "A1", "D1")
	f.SetCellStyle(sheet, "A1", "D1", headerStyle)
	f.SetRowHeight(sheet, 1, 30)

	// Info section style
	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E2EFDA"}, Pattern: 1},
	})

	// Plan info
	info := [][]string{
		{"Название:", plan.Name},
		{"Клиент:", plan.ClientName},
		{"Цель:", plan.Goal},
		{"Дата начала:", plan.StartDate.Format("02.01.2006")},
		{"Длительность:", fmt.Sprintf("%d недель", plan.TotalWeeks)},
		{"Тренировок/неделю:", fmt.Sprintf("%d", plan.DaysPerWeek)},
		{"Статус:", string(plan.Status)},
	}

	for i, row := range info {
		rowNum := i + 3
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), row[0])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), row[1])
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("A%d", rowNum), labelStyle)
	}

	// Column widths
	f.SetColWidth(sheet, "A", "A", 20)
	f.SetColWidth(sheet, "B", "B", 30)
	f.SetColWidth(sheet, "C", "D", 15)

	// Mesocycles summary
	f.SetCellValue(sheet, "A12", "ПЕРИОДИЗАЦИЯ:")
	f.SetCellStyle(sheet, "A12", "A12", labelStyle)

	row := 13
	for _, meso := range plan.Mesocycles {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("Нед. %d-%d:", meso.WeekStart, meso.WeekEnd))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), meso.Name)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), meso.Phase.NameRu())
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("%d%%", meso.IntensityPercent))
		row++
	}

	return nil
}

// createPeriodizationSheetExport creates detailed periodization view
func createPeriodizationSheetExport(f *excelize.File, plan *models.TrainingPlan) error {
	sheet := SheetPeriodization

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	// Phase colors
	phaseColors := map[models.PlanPhase]string{
		models.PhaseHypertrophy: "BDD7EE",
		models.PhaseStrength:    "F8CBAD",
		models.PhasePower:       "FFE699",
		models.PhasePeaking:     "C6EFCE",
		models.PhaseDeload:      "D9D9D9",
	}

	// Title
	f.SetCellValue(sheet, "A1", "ПЕРИОДИЗАЦИЯ ТРЕНИРОВОК")
	f.MergeCell(sheet, "A1", "F1")
	f.SetCellStyle(sheet, "A1", "F1", headerStyle)
	f.SetRowHeight(sheet, 1, 25)

	// Headers
	headers := []string{"Мезоцикл", "Фаза", "Недели", "Интенсивность", "Объём", "Описание"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s3", colName(i+1))
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Data rows
	row := 4
	for _, meso := range plan.Mesocycles {
		// Phase-specific style
		phaseStyle, _ := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{phaseColors[meso.Phase]}, Pattern: 1},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
			},
			Alignment: &excelize.Alignment{Vertical: "center"},
		})

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), meso.Name)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), meso.Phase.NameRu())
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%d - %d", meso.WeekStart, meso.WeekEnd))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("%d%%", meso.IntensityPercent))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%d%%", meso.VolumePercent))

		phaseConfig := models.DefaultPhaseConfigs[meso.Phase]
		description := fmt.Sprintf("%d-%d подх, %d-%d повт",
			phaseConfig.SetsRange[0], phaseConfig.SetsRange[1],
			phaseConfig.RepsRange[0], phaseConfig.RepsRange[1])
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), description)

		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("F%d", row), phaseStyle)
		row++
	}

	// Column widths
	f.SetColWidth(sheet, "A", "A", 25)
	f.SetColWidth(sheet, "B", "B", 15)
	f.SetColWidth(sheet, "C", "C", 12)
	f.SetColWidth(sheet, "D", "E", 14)
	f.SetColWidth(sheet, "F", "F", 25)

	// Legend
	row += 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "ЛЕГЕНДА ФАЗ:")
	row++
	for phase, color := range phaseColors {
		legendStyle, _ := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
		})
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), phase.NameRu())
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), legendStyle)
		row++
	}

	return nil
}

// createProgressionSheetExport creates the main progression table
func createProgressionSheetExport(f *excelize.File, plan *models.TrainingPlan, progression []models.Progression) error {
	sheet := SheetProgression

	// Styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	normalStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	deloadStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D9D9D9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	// Title
	f.SetCellValue(sheet, "A1", "ТАБЛИЦА ПРОГРЕССИИ ВЕСОВ")
	f.MergeCell(sheet, "A1", colName(plan.TotalWeeks+1)+"1")
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheet, "A1", colName(plan.TotalWeeks+1)+"1", titleStyle)
	f.SetRowHeight(sheet, 1, 25)

	// Header row - Exercise column
	f.SetCellValue(sheet, "A3", "Упражнение")
	f.SetCellStyle(sheet, "A3", "A3", headerStyle)

	// Week headers
	for week := 1; week <= plan.TotalWeeks; week++ {
		col := colName(week + 1)
		isDeload := week%4 == 0 // deload every 4th week

		headerText := fmt.Sprintf("Нед %d", week)
		if isDeload {
			headerText += "\n(D)"
		}

		f.SetCellValue(sheet, col+"3", headerText)
		f.SetCellStyle(sheet, col+"3", col+"3", headerStyle)
	}

	// Group progression by exercise
	exerciseMap := make(map[string]map[int]models.Progression)
	exerciseOrder := make([]string, 0)
	seen := make(map[string]bool)

	for _, p := range progression {
		if !seen[p.ExerciseName] {
			seen[p.ExerciseName] = true
			exerciseOrder = append(exerciseOrder, p.ExerciseName)
			exerciseMap[p.ExerciseName] = make(map[int]models.Progression)
		}
		exerciseMap[p.ExerciseName][p.WeekNumber] = p
	}

	// Fill data
	row := 4
	exerciseStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	for _, exName := range exerciseOrder {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), exName)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), exerciseStyle)

		for week := 1; week <= plan.TotalWeeks; week++ {
			col := colName(week + 1)
			cell := fmt.Sprintf("%s%d", col, row)

			if p, ok := exerciseMap[exName][week]; ok {
				// Format: "4x8 60кг" or "3x6 45кг (D)"
				cellValue := fmt.Sprintf("%dx%d %.1fкг", p.Sets, p.Reps, p.WeightKg)

				f.SetCellValue(sheet, cell, cellValue)

				if p.IsDeload {
					f.SetCellStyle(sheet, cell, cell, deloadStyle)
				} else {
					f.SetCellStyle(sheet, cell, cell, normalStyle)
				}
			}
		}
		row++
	}

	// Column widths
	f.SetColWidth(sheet, "A", "A", 25)
	for week := 1; week <= plan.TotalWeeks; week++ {
		f.SetColWidth(sheet, colName(week+1), colName(week+1), 12)
	}

	// Freeze header
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		XSplit:      1,
		YSplit:      3,
		TopLeftCell: "B4",
	})

	return nil
}

// create1PMHistorySheetExport creates the 1PM history sheet
func create1PMHistorySheetExport(f *excelize.File, history []models.Exercise1PMHistory) error {
	sheet := Sheet1PMHistory

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	// Title
	f.SetCellValue(sheet, "A1", "ИСТОРИЯ 1ПМ")
	f.MergeCell(sheet, "A1", "E1")
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"375623"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheet, "A1", "E1", titleStyle)

	// Summary table headers
	summaryHeaders := []string{"Упражнение", "Начальный 1ПМ", "Текущий 1ПМ", "Прирост (кг)", "Прирост (%)"}
	for i, h := range summaryHeaders {
		cell := fmt.Sprintf("%s3", colName(i+1))
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Summary data
	row := 4
	for _, h := range history {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), h.ExerciseName)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%.1f кг", h.InitialPM))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%.1f кг", h.CurrentPM))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("%+.1f кг", h.GainKg))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%+.1f%%", h.GainPercent))
		row++
	}

	// Detailed history section
	row += 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "ДЕТАЛЬНАЯ ИСТОРИЯ")
	f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), titleStyle)
	row++

	detailHeaders := []string{"Дата", "Упражнение", "1ПМ (кг)", "Метод", "Источник"}
	for i, h := range detailHeaders {
		cell := fmt.Sprintf("%s%d", colName(i+1), row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	row++

	for _, h := range history {
		for _, record := range h.Records {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), record.TestDate.Format("02.01.2006"))
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), h.ExerciseName)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%.1f", record.OnePMKg))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record.CalcMethod)
			if record.CalcMethod != "manual" && record.SourceWeight > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("E%d", row),
					fmt.Sprintf("%.1fкг × %d повт", record.SourceWeight, record.SourceReps))
			}
			row++
		}
	}

	// Column widths
	f.SetColWidth(sheet, "A", "A", 20)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 15)
	f.SetColWidth(sheet, "D", "D", 15)
	f.SetColWidth(sheet, "E", "E", 20)

	return nil
}

// createVolumeAnalysisSheetExport creates volume analysis with chart
func createVolumeAnalysisSheetExport(f *excelize.File, plan *models.TrainingPlan, progression []models.Progression) error {
	sheet := SheetVolumeAnalysis

	// Calculate weekly tonnage
	weeklyTonnage := make(map[int]float64)
	for _, p := range progression {
		tonnage := float64(p.Sets) * float64(p.Reps) * p.WeightKg
		weeklyTonnage[p.WeekNumber] += tonnage
	}

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"ED7D31"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Title
	f.SetCellValue(sheet, "A1", "АНАЛИЗ ТРЕНИРОВОЧНОГО ОБЪЁМА")
	f.MergeCell(sheet, "A1", "C1")
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"C65911"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheet, "A1", "C1", titleStyle)

	// Headers
	f.SetCellValue(sheet, "A3", "Неделя")
	f.SetCellValue(sheet, "B3", "Тоннаж (кг)")
	f.SetCellValue(sheet, "C3", "Статус")
	f.SetCellStyle(sheet, "A3", "C3", headerStyle)

	// Data
	row := 4
	for week := 1; week <= plan.TotalWeeks; week++ {
		tonnage := weeklyTonnage[week]
		isDeload := week%4 == 0

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("Неделя %d", week))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%.0f", tonnage))

		status := "Нагрузка"
		if isDeload {
			status = "Разгрузка"
		}
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), status)
		row++
	}

	// Add tonnage chart
	chartDataEnd := 3 + plan.TotalWeeks
	if err := addTonnageChart(f, sheet, chartDataEnd); err != nil {
		// Chart creation failed, but continue without it
	}

	// Column widths
	f.SetColWidth(sheet, "A", "A", 15)
	f.SetColWidth(sheet, "B", "B", 15)
	f.SetColWidth(sheet, "C", "C", 15)

	return nil
}

// addTonnageChart adds a bar chart for weekly tonnage
func addTonnageChart(f *excelize.File, sheet string, dataEndRow int) error {
	chart := &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{
			{
				Name:       fmt.Sprintf("'%s'!$B$3", sheet),
				Categories: fmt.Sprintf("'%s'!$A$4:$A$%d", sheet, dataEndRow),
				Values:     fmt.Sprintf("'%s'!$B$4:$B$%d", sheet, dataEndRow),
				Fill: excelize.Fill{
					Type:    "pattern",
					Color:   []string{"ED7D31"},
					Pattern: 1,
				},
			},
		},
		Title: []excelize.RichTextRun{
			{Text: "Тоннаж по неделям"},
		},
		PlotArea: excelize.ChartPlotArea{
			ShowVal: true,
		},
		Dimension: excelize.ChartDimension{
			Width:  480,
			Height: 300,
		},
	}

	return f.AddChart(sheet, "E3", chart)
}

// colName returns Excel column name for index (1 = A, 2 = B, etc.)
func colName(n int) string {
	result := ""
	for n > 0 {
		n--
		result = string(rune('A'+n%26)) + result
		n /= 26
	}
	return result
}

// GeneratePlanFilename generates a filename for the Excel export
func GeneratePlanFilename(planName string, clientName string) string {
	// Sanitize names for filename
	planName = sanitizeFilename(planName)
	clientName = sanitizeFilename(clientName)

	return fmt.Sprintf("План_%s_%s_%s.xlsx",
		clientName,
		planName,
		time.Now().Format("2006-01-02"))
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}


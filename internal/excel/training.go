package excel

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"workbot/internal/models"

	"github.com/xuri/excelize/v2"
)

// ReadAllClientTrainings читает упражнения со всех листов данных клиентов
func ReadAllClientTrainings(filePath string, db *sql.DB) ([]models.ClientTraining, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var trainings []models.ClientTraining

	sheets := f.GetSheetList()

	for _, sheetName := range sheets {
		if !strings.HasSuffix(sheetName, "_data") {
			continue
		}

		clientIDStr, _ := f.GetCellValue(sheetName, "P2")
		clientID, _ := strconv.Atoi(clientIDStr)

		if clientID == 0 {
			continue
		}

		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}

		for i, row := range rows {
			if i < 1 {
				continue
			}
			if len(row) < 3 {
				continue
			}

			trainingDate := getCell(row, 0)
			trainingNum, _ := strconv.Atoi(getCell(row, 1))
			exercise := getCell(row, 2)
			sets, _ := strconv.Atoi(getCell(row, 3))
			reps, _ := strconv.Atoi(getCell(row, 4))
			weight, _ := strconv.ParseFloat(getCell(row, 5), 64)
			tonnage, _ := strconv.ParseFloat(getCell(row, 6), 64)
			status := getCell(row, 7)
			feedback := getCell(row, 8)
			rating, _ := strconv.Atoi(getCell(row, 9))
			completedDate := getCell(row, 10)
			completedTime := getCell(row, 11)
			sentStr := getCell(row, 14)
			sent := sentStr == "true" || sentStr == "TRUE" || sentStr == "да" || sentStr == "Да"

			if exercise == "" {
				continue
			}

			trainings = append(trainings, models.ClientTraining{
				SheetName:     sheetName,
				RowNum:        i + 1,
				ClientID:      clientID,
				Date:          trainingDate,
				TrainingNum:   trainingNum,
				Exercise:      exercise,
				Sets:          sets,
				Reps:          reps,
				Weight:        weight,
				Tonnage:       tonnage,
				Status:        status,
				Feedback:      feedback,
				Rating:        rating,
				CompletedDate: completedDate,
				CompletedTime: completedTime,
				Sent:          sent,
			})
		}
	}

	return trainings, nil
}

func getCell(row []string, index int) string {
	if index < len(row) {
		return row[index]
	}
	return ""
}

// MarkClientTrainingAsSent помечает тренировку как отправленную
func MarkClientTrainingAsSent(filePath string, sheetName string, rowNum int) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	cell, _ := excelize.CoordinatesToCellName(15, rowNum)
	f.SetCellValue(sheetName, cell, "да")

	return f.Save()
}

// ReadTrainings читает тренировки из общего листа
func ReadTrainings(filePath string) ([]models.Training, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows, err := f.GetRows(SheetTrainings)
	if err != nil {
		return nil, err
	}

	var trainings []models.Training
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 5 {
			continue
		}

		clientID, _ := strconv.Atoi(row[0])
		sent := row[4] == "true" || row[4] == "TRUE" || row[4] == "1"

		trainings = append(trainings, models.Training{
			RowNum:      i + 1,
			ClientID:    clientID,
			Date:        row[1],
			Time:        row[2],
			Description: row[3],
			Sent:        sent,
		})
	}

	return trainings, nil
}

// MarkAsSent помечает тренировку как отправленную
func MarkAsSent(filePath string, rowNum int) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	cell, _ := excelize.CoordinatesToCellName(5, rowNum)
	f.SetCellValue(SheetTrainings, cell, "true")

	return f.Save()
}

// SaveTrainingToExcel сохраняет тренировку из Telegram в Excel (единый лист)
func SaveTrainingToExcel(filePath string, db *sql.DB, clientID int, name, surname string, trainingDate time.Time, exercises []models.ExerciseInput) error {
	clientName := fmt.Sprintf("%s %s", name, surname)
	return SaveTrainingToUnified(filePath, clientID, clientName, trainingDate, exercises)
}

// GetClientTrainings возвращает последние N тренировок клиента в виде отформатированных строк
func GetClientTrainings(filePath string, clientID int, limit int) ([]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Ищем лист данных клиента
	sheetName := ""
	for _, name := range f.GetSheetList() {
		if !strings.HasSuffix(name, "_data") {
			continue
		}
		idStr, _ := f.GetCellValue(name, "P2")
		id, _ := strconv.Atoi(idStr)
		if id == clientID {
			sheetName = name
			break
		}
	}

	if sheetName == "" {
		return nil, nil
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	// Группируем по номеру тренировки
	type trainingGroup struct {
		date      string
		num       int
		exercises []string
		tonnage   float64
	}

	groups := make(map[int]*trainingGroup)
	var groupOrder []int

	for i, row := range rows {
		if i < 1 || len(row) < 3 {
			continue
		}

		trainingDate := getCell(row, 0)
		trainingNum, _ := strconv.Atoi(getCell(row, 1))
		exercise := getCell(row, 2)
		sets, _ := strconv.Atoi(getCell(row, 3))
		reps, _ := strconv.Atoi(getCell(row, 4))
		weight, _ := strconv.ParseFloat(getCell(row, 5), 64)
		tonnage, _ := strconv.ParseFloat(getCell(row, 6), 64)

		if exercise == "" || trainingNum == 0 {
			continue
		}

		if _, exists := groups[trainingNum]; !exists {
			groups[trainingNum] = &trainingGroup{
				date: trainingDate,
				num:  trainingNum,
			}
			groupOrder = append(groupOrder, trainingNum)
		}

		exStr := fmt.Sprintf("  • %s %d/%d", exercise, sets, reps)
		if weight > 0 {
			exStr += fmt.Sprintf(" %.0fкг", weight)
		}
		groups[trainingNum].exercises = append(groups[trainingNum].exercises, exStr)
		groups[trainingNum].tonnage += tonnage
	}

	// Берём последние N тренировок
	var result []string
	start := 0
	if len(groupOrder) > limit {
		start = len(groupOrder) - limit
	}

	for i := len(groupOrder) - 1; i >= start; i-- {
		num := groupOrder[i]
		g := groups[num]
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("━━━ Тренировка #%d (%s) ━━━\n", g.num, g.date))
		for _, ex := range g.exercises {
			sb.WriteString(ex)
			sb.WriteString("\n")
		}
		if g.tonnage > 0 {
			sb.WriteString(fmt.Sprintf("Тоннаж: %.0f кг", g.tonnage))
		}
		result = append(result, sb.String())
	}

	return result, nil
}

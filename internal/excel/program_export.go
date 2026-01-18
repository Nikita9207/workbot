package excel

import (
	"fmt"
	"log"
	"time"

	"github.com/xuri/excelize/v2"
	"workbot/internal/models"
)

// AddProgramToClientWorkbook добавляет программу тренировок в существующую таблицу клиента
// Записывает данные в лист "Журнал" с правильной структурой колонок
func AddProgramToClientWorkbook(filePath string, program *models.Program, clientName string, clientID int) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer func() {
		if saveErr := f.Save(); saveErr != nil {
			log.Printf("Ошибка сохранения файла: %v", saveErr)
		}
		f.Close()
	}()

	// Проверяем наличие листа Журнал
	sheet := SheetJournal
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		return fmt.Errorf("лист '%s' не найден в файле", sheet)
	}

	// Находим следующую пустую строку
	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("ошибка чтения листа: %w", err)
	}

	nextRow := len(rows) + 1
	if nextRow < 2 {
		nextRow = 2
	}

	// Определяем следующий номер тренировки
	nextTrainingNum := getNextTrainingNum(rows, clientID)

	// Стиль для даты
	dateStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 14})

	// Записываем все тренировки из программы
	for _, workout := range program.Workouts {
		// Вычисляем дату тренировки на основе недели и дня
		workoutDate := calculateWorkoutDate(program.StartDate, workout.WeekNum, workout.DayNum)

		for _, ex := range workout.Exercises {
			// Записываем строку в Журнал
			// Структура колонок из hybrid.go:
			// A-Дата, B-Клиент, C-Формат, D-№трен, E-Упражнение, F-Подходы, G-Повторы, H-Вес,
			// I-Объём (формула), J-%1RM, K-RPE, L-Отдых, M-Факт вес, N-Факт повт,
			// O-Статус, P-Оценка, Q-Заметки, R-Отправлено, S-Дата вып., T-ID

			// A - Дата
			f.SetCellValue(sheet, fmt.Sprintf("A%d", nextRow), workoutDate)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", nextRow), fmt.Sprintf("A%d", nextRow), dateStyle)

			// B - Клиент
			f.SetCellValue(sheet, fmt.Sprintf("B%d", nextRow), clientName)

			// C - Формат (онлайн для программ из бота)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", nextRow), "онлайн")

			// D - № тренировки (уникальный для каждой тренировки в программе)
			trainingNum := nextTrainingNum + (workout.WeekNum-1)*program.DaysPerWeek + workout.DayNum - 1
			f.SetCellValue(sheet, fmt.Sprintf("D%d", nextRow), trainingNum)

			// E - Упражнение
			f.SetCellValue(sheet, fmt.Sprintf("E%d", nextRow), ex.ExerciseName)

			// F - Подходы
			f.SetCellValue(sheet, fmt.Sprintf("F%d", nextRow), ex.Sets)

			// G - Повторы
			f.SetCellValue(sheet, fmt.Sprintf("G%d", nextRow), ex.Reps)

			// H - Вес (кг)
			if ex.Weight > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("H%d", nextRow), ex.Weight)
			}

			// I - Объём (формула уже есть в шаблоне, но добавим если нет)
			// Формула: F*G*H

			// J - % 1RM
			if ex.WeightPercent > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("J%d", nextRow), ex.WeightPercent)
			}

			// K - RPE
			if ex.RPE > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("K%d", nextRow), ex.RPE)
			}

			// L - Отдых (секунды)
			if ex.RestSeconds > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("L%d", nextRow), ex.RestSeconds)
			}

			// O - Статус
			f.SetCellValue(sheet, fmt.Sprintf("O%d", nextRow), "запланировано")

			// Q - Заметки (темп + notes)
			notes := ""
			if ex.Tempo != "" {
				notes = fmt.Sprintf("Темп: %s", ex.Tempo)
			}
			if ex.Notes != "" {
				if notes != "" {
					notes += ". "
				}
				notes += ex.Notes
			}
			if notes != "" {
				f.SetCellValue(sheet, fmt.Sprintf("Q%d", nextRow), notes)
			}

			// R - Отправлено
			f.SetCellValue(sheet, fmt.Sprintf("R%d", nextRow), "нет")

			// T - ID клиента (скрытая колонка)
			f.SetCellValue(sheet, fmt.Sprintf("T%d", nextRow), clientID)

			nextRow++
		}
	}

	log.Printf("Добавлено %d упражнений из программы '%s' в таблицу клиента", nextRow-len(rows)-1, program.Name)
	return nil
}

// AddProgramWeekToWorkbook добавляет одну неделю программы в таблицу
func AddProgramWeekToWorkbook(filePath string, program *models.Program, weekNum int, clientName string, clientID int) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer func() {
		if saveErr := f.Save(); saveErr != nil {
			log.Printf("Ошибка сохранения файла: %v", saveErr)
		}
		f.Close()
	}()

	sheet := SheetJournal
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		return fmt.Errorf("лист '%s' не найден", sheet)
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("ошибка чтения листа: %w", err)
	}

	nextRow := len(rows) + 1
	if nextRow < 2 {
		nextRow = 2
	}

	nextTrainingNum := getNextTrainingNum(rows, clientID)
	dateStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 14})

	workouts := program.GetWorkoutsByWeek(weekNum)
	exerciseCount := 0

	for _, workout := range workouts {
		workoutDate := calculateWorkoutDate(program.StartDate, workout.WeekNum, workout.DayNum)
		trainingNum := nextTrainingNum + workout.DayNum - 1

		for _, ex := range workout.Exercises {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", nextRow), workoutDate)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", nextRow), fmt.Sprintf("A%d", nextRow), dateStyle)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", nextRow), clientName)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", nextRow), "онлайн")
			f.SetCellValue(sheet, fmt.Sprintf("D%d", nextRow), trainingNum)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", nextRow), ex.ExerciseName)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", nextRow), ex.Sets)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", nextRow), ex.Reps)

			if ex.Weight > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("H%d", nextRow), ex.Weight)
			}
			if ex.WeightPercent > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("J%d", nextRow), ex.WeightPercent)
			}
			if ex.RPE > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("K%d", nextRow), ex.RPE)
			}
			if ex.RestSeconds > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("L%d", nextRow), ex.RestSeconds)
			}

			f.SetCellValue(sheet, fmt.Sprintf("O%d", nextRow), "запланировано")

			notes := ""
			if ex.Tempo != "" {
				notes = fmt.Sprintf("Темп: %s", ex.Tempo)
			}
			if ex.Notes != "" {
				if notes != "" {
					notes += ". "
				}
				notes += ex.Notes
			}
			if notes != "" {
				f.SetCellValue(sheet, fmt.Sprintf("Q%d", nextRow), notes)
			}

			f.SetCellValue(sheet, fmt.Sprintf("R%d", nextRow), "нет")
			f.SetCellValue(sheet, fmt.Sprintf("T%d", nextRow), clientID)

			nextRow++
			exerciseCount++
		}
	}

	log.Printf("Добавлена неделя %d: %d упражнений", weekNum, exerciseCount)
	return nil
}

// FillWorkoutSheet заполняет лист "Тренировка" одной тренировкой для отправки
func FillWorkoutSheet(filePath string, workout *models.Workout, trainingNum int) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer func() {
		if saveErr := f.Save(); saveErr != nil {
			log.Printf("Ошибка сохранения файла: %v", saveErr)
		}
		f.Close()
	}()

	sheet := SheetWorkout
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		return fmt.Errorf("лист '%s' не найден", sheet)
	}

	// Информация о тренировке (ячейки из hybrid.go)
	// D5:F5 - Дата, I5:J5 - № тренировки
	if workout.Date != nil {
		f.SetCellValue(sheet, "D5", workout.Date.Format("02.01.2006"))
	} else {
		f.SetCellValue(sheet, "D5", time.Now().Format("02.01.2006"))
	}
	f.SetCellValue(sheet, "I5", trainingNum)

	// D6:F6 - Тип тренировки (определяем из названия)
	f.SetCellValue(sheet, "D6", "Силовая")

	// I6:J6 - Направленность (из названия тренировки)
	f.SetCellValue(sheet, "I6", extractDirection(workout.Name))

	// Упражнения (строки 11-18, колонки по структуре из hybrid.go)
	// B-F Упражнение, G-H Подходы, I-J Повторы, K-L Вес, M-N Отдых, O-P %1RM, Q RPE, R-T Заметки
	for i, ex := range workout.Exercises {
		if i >= 8 {
			break // Максимум 8 упражнений на листе
		}
		row := 11 + i

		// Упражнение (B-F объединены)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), ex.ExerciseName)

		// Подходы (G-H)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), ex.Sets)

		// Повторы (I-J)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), ex.Reps)

		// Вес (K-L)
		if ex.Weight > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("K%d", row), ex.Weight)
		}

		// Отдых (M-N)
		if ex.RestSeconds > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("M%d", row), ex.RestSeconds)
		}

		// %1RM (O-P)
		if ex.WeightPercent > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("O%d", row), ex.WeightPercent)
		}

		// RPE (Q)
		if ex.RPE > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), ex.RPE)
		}

		// Заметки (R-T)
		notes := ""
		if ex.Tempo != "" {
			notes = ex.Tempo
		}
		if ex.Notes != "" {
			if notes != "" {
				notes += " | "
			}
			notes += ex.Notes
		}
		if notes != "" {
			f.SetCellValue(sheet, fmt.Sprintf("R%d", row), notes)
		}
	}

	// Статус отправки
	f.SetCellValue(sheet, "M5", "нет")
	f.SetCellValue(sheet, "M7", "нет")

	log.Printf("Заполнен лист тренировки: %s (%d упражнений)", workout.Name, len(workout.Exercises))
	return nil
}

// ClearWorkoutSheet очищает лист тренировки для новой записи
func ClearWorkoutSheet(filePath string) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer func() {
		if saveErr := f.Save(); saveErr != nil {
			log.Printf("Ошибка сохранения файла: %v", saveErr)
		}
		f.Close()
	}()

	sheet := SheetWorkout
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		return nil // Лист не существует, ничего не делаем
	}

	// Очищаем поля информации
	f.SetCellValue(sheet, "D5", "")
	f.SetCellValue(sheet, "I5", "")
	f.SetCellValue(sheet, "D6", "")
	f.SetCellValue(sheet, "I6", "")
	f.SetCellValue(sheet, "D7", "")

	// Очищаем упражнения (строки 11-18)
	for row := 11; row <= 18; row++ {
		for _, col := range []string{"B", "G", "I", "K", "M", "O", "Q", "R"} {
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), "")
		}
	}

	// Очищаем разминку и заминку (строки 21-24)
	for row := 21; row <= 24; row++ {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "")
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), "")
	}

	// Сбрасываем статусы
	f.SetCellValue(sheet, "M5", "нет")
	f.SetCellValue(sheet, "M6", "")
	f.SetCellValue(sheet, "M7", "нет")

	return nil
}

// Вспомогательные функции

// getNextTrainingNum определяет следующий номер тренировки для клиента
func getNextTrainingNum(rows [][]string, clientID int) int {
	maxNum := 0
	for i := 1; i < len(rows); i++ {
		if len(rows[i]) < 20 {
			continue
		}
		// T (колонка 19) - ID клиента, D (колонка 3) - номер тренировки
		rowClientID := 0
		if len(rows[i]) > 19 {
			fmt.Sscanf(rows[i][19], "%d", &rowClientID)
		}
		if rowClientID == clientID || clientID == 0 {
			var num int
			if len(rows[i]) > 3 {
				fmt.Sscanf(rows[i][3], "%d", &num)
			}
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return maxNum + 1
}

// calculateWorkoutDate вычисляет дату тренировки
func calculateWorkoutDate(startDate time.Time, weekNum, dayNum int) time.Time {
	// Считаем дни от начала программы
	// weekNum начинается с 1, dayNum - день в неделе (1-7)
	daysOffset := (weekNum-1)*7 + (dayNum - 1)
	return startDate.AddDate(0, 0, daysOffset)
}

// extractDirection извлекает направленность из названия тренировки
func extractDirection(name string) string {
	directions := map[string]string{
		"верх":     "Верх",
		"низ":      "Низ",
		"push":     "Push",
		"pull":     "Pull",
		"ноги":     "Низ",
		"грудь":    "Push",
		"спина":    "Pull",
		"фуллбоди": "Фуллбоди",
		"full":     "Фуллбоди",
	}

	nameLower := name
	for key, val := range directions {
		if containsIgnoreCase(nameLower, key) {
			return val
		}
	}
	return "Фуллбоди"
}

func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			sLower[i] = s[i] + 32
		} else if s[i] >= 0xD0 && i+1 < len(s) { // Кириллица UTF-8
			sLower[i] = s[i]
		} else {
			sLower[i] = s[i]
		}
	}
	for i := 0; i < len(substr); i++ {
		if substr[i] >= 'A' && substr[i] <= 'Z' {
			substrLower[i] = substr[i] + 32
		} else {
			substrLower[i] = substr[i]
		}
	}

	return containsBytes(sLower, substrLower)
}

func containsBytes(s, substr []byte) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

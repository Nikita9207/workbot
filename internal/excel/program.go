package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"workbot/internal/models"

	"github.com/xuri/excelize/v2"
)

// ProgramManager управляет Excel файлами программ клиентов
type ProgramManager struct {
	clientsDir  string
	journalPath string
}

// NewProgramManager создаёт новый менеджер программ
// clientsDir - папка с папками клиентов (например: ~/Desktop/Работа/Клиенты)
func NewProgramManager(clientsDir, journalPath string) *ProgramManager {
	return &ProgramManager{
		clientsDir:  clientsDir,
		journalPath: journalPath,
	}
}

// CreateProgramFile создаёт Excel файл с программой тренировок для клиента
func (pm *ProgramManager) CreateProgramFile(program *models.Program) (string, error) {
	// Создаём директорию если не существует
	if err := os.MkdirAll(pm.clientsDir, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания директории: %w", err)
	}

	f := excelize.NewFile()
	defer f.Close()

	// Создаём лист "Программа"
	sheetName := "Программа"
	f.SetSheetName("Sheet1", sheetName)

	// Заголовок программы
	f.SetCellValue(sheetName, "A1", "Программа тренировок")
	f.SetCellValue(sheetName, "A2", fmt.Sprintf("Клиент: %s", program.ClientName))
	f.SetCellValue(sheetName, "A3", fmt.Sprintf("Цель: %s", program.Goal))
	f.SetCellValue(sheetName, "A4", fmt.Sprintf("Длительность: %d недель", program.TotalWeeks))
	f.SetCellValue(sheetName, "A5", fmt.Sprintf("Тренировок в неделю: %d", program.DaysPerWeek))
	f.SetCellValue(sheetName, "A6", fmt.Sprintf("Начало: %s", program.StartDate.Format("02.01.2006")))

	// Стили для заголовка
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})
	f.SetCellStyle(sheetName, "A1", "A1", headerStyle)

	// Создаём листы для каждой недели
	for weekNum := 1; weekNum <= program.TotalWeeks; weekNum++ {
		weekSheetName := fmt.Sprintf("Неделя %d", weekNum)
		f.NewSheet(weekSheetName)

		// Заголовки таблицы
		headers := []string{"Тренировка", "Статус", "Дата", "Упражнение", "Подходы", "Повторы", "Вес", "Отдых", "Темп", "RPE", "Заметки", "Выполнено"}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(weekSheetName, cell, h)
		}

		// Стиль заголовков
		headerRowStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
			},
		})
		f.SetCellStyle(weekSheetName, "A1", "L1", headerRowStyle)

		// Заполняем тренировки недели
		row := 2
		workouts := program.GetWorkoutsByWeek(weekNum)
		for _, workout := range workouts {
			for _, ex := range workout.Exercises {
				f.SetCellValue(weekSheetName, fmt.Sprintf("A%d", row), workout.Name)
				f.SetCellValue(weekSheetName, fmt.Sprintf("B%d", row), string(workout.Status))
				if workout.Date != nil {
					f.SetCellValue(weekSheetName, fmt.Sprintf("C%d", row), workout.Date.Format("02.01.2006"))
				}
				f.SetCellValue(weekSheetName, fmt.Sprintf("D%d", row), ex.ExerciseName)
				f.SetCellValue(weekSheetName, fmt.Sprintf("E%d", row), ex.Sets)
				f.SetCellValue(weekSheetName, fmt.Sprintf("F%d", row), ex.Reps)
				if ex.Weight > 0 {
					f.SetCellValue(weekSheetName, fmt.Sprintf("G%d", row), ex.Weight)
				} else if ex.WeightPercent > 0 {
					f.SetCellValue(weekSheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("%.0f%% 1ПМ", ex.WeightPercent))
				}
				f.SetCellValue(weekSheetName, fmt.Sprintf("H%d", row), fmt.Sprintf("%dс", ex.RestSeconds))
				f.SetCellValue(weekSheetName, fmt.Sprintf("I%d", row), ex.Tempo)
				if ex.RPE > 0 {
					f.SetCellValue(weekSheetName, fmt.Sprintf("J%d", row), ex.RPE)
				}
				f.SetCellValue(weekSheetName, fmt.Sprintf("K%d", row), ex.Notes)
				if ex.Completed {
					f.SetCellValue(weekSheetName, fmt.Sprintf("L%d", row), "Да")
				}
				row++
			}
			row++ // Пустая строка между тренировками
		}

		// Ширина столбцов
		f.SetColWidth(weekSheetName, "A", "A", 20)
		f.SetColWidth(weekSheetName, "B", "B", 12)
		f.SetColWidth(weekSheetName, "C", "C", 12)
		f.SetColWidth(weekSheetName, "D", "D", 25)
		f.SetColWidth(weekSheetName, "E", "F", 10)
		f.SetColWidth(weekSheetName, "G", "G", 12)
		f.SetColWidth(weekSheetName, "H", "J", 8)
		f.SetColWidth(weekSheetName, "K", "K", 30)
		f.SetColWidth(weekSheetName, "L", "L", 12)
	}

	// Сохраняем файл в папку клиента
	// Структура: Клиенты/Имя_Фамилия/Программа_Имя_Дата.xlsx
	clientFolderName := sanitizeFilename(program.ClientName)
	clientFolder := filepath.Join(pm.clientsDir, clientFolderName)

	// Создаём папку клиента если не существует
	if err := os.MkdirAll(clientFolder, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания папки клиента: %w", err)
	}

	filename := fmt.Sprintf("Программа_%s_%s.xlsx",
		sanitizeFilename(program.Name),
		time.Now().Format("2006-01-02"))
	filePath := filepath.Join(clientFolder, filename)

	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("ошибка сохранения файла: %w", err)
	}

	return filePath, nil
}

// UpdateWorkoutStatus обновляет статус тренировки в файле
func (pm *ProgramManager) UpdateWorkoutStatus(filePath string, weekNum int, workoutName string, status models.WorkoutStatus) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer f.Close()

	weekSheetName := fmt.Sprintf("Неделя %d", weekNum)
	rows, err := f.GetRows(weekSheetName)
	if err != nil {
		return fmt.Errorf("ошибка чтения листа: %w", err)
	}

	for i, row := range rows {
		if len(row) > 0 && row[0] == workoutName {
			f.SetCellValue(weekSheetName, fmt.Sprintf("B%d", i+1), string(status))
			if status == models.WorkoutStatusCompleted {
				f.SetCellValue(weekSheetName, fmt.Sprintf("L%d", i+1), "Да")
			}
		}
	}

	return f.Save()
}

// GetNextWorkoutFromFile читает следующую непройденную тренировку из файла
func (pm *ProgramManager) GetNextWorkoutFromFile(filePath string) (*models.Workout, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer f.Close()

	// Ищем первую тренировку со статусом pending
	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		if sheet == "Программа" {
			continue
		}

		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		var currentWorkout *models.Workout
		for i, row := range rows {
			if i == 0 || len(row) < 6 { // Пропускаем заголовок
				continue
			}

			workoutName := row[0]
			status := row[1]

			if status == string(models.WorkoutStatusPending) && workoutName != "" {
				if currentWorkout == nil || currentWorkout.Name != workoutName {
					currentWorkout = &models.Workout{
						Name:      workoutName,
						Status:    models.WorkoutStatusPending,
						Exercises: []models.WorkoutExercise{},
					}
				}

				// Парсим упражнение
				exercise := models.WorkoutExercise{
					ExerciseName: row[3],
				}
				if len(row) > 4 {
					fmt.Sscanf(row[4], "%d", &exercise.Sets)
				}
				if len(row) > 5 {
					exercise.Reps = row[5]
				}
				if len(row) > 6 {
					fmt.Sscanf(row[6], "%f", &exercise.Weight)
				}
				if len(row) > 7 {
					fmt.Sscanf(row[7], "%dс", &exercise.RestSeconds)
				}
				if len(row) > 8 {
					exercise.Tempo = row[8]
				}
				if len(row) > 9 {
					fmt.Sscanf(row[9], "%f", &exercise.RPE)
				}
				if len(row) > 10 {
					exercise.Notes = row[10]
				}

				if exercise.ExerciseName != "" {
					currentWorkout.Exercises = append(currentWorkout.Exercises, exercise)
				}
			}

			// Если нашли тренировку и перешли к следующей - возвращаем
			if currentWorkout != nil && workoutName != "" && workoutName != currentWorkout.Name {
				return currentWorkout, nil
			}
		}

		if currentWorkout != nil && len(currentWorkout.Exercises) > 0 {
			return currentWorkout, nil
		}
	}

	return nil, nil
}

// SaveClientForm сохраняет анкету клиента в Excel в его персональную папку
func (pm *ProgramManager) SaveClientForm(form *models.ClientForm) (string, error) {
	// Создаём папку клиента
	clientFolderName := fmt.Sprintf("%s_%s", sanitizeFilename(form.Name), sanitizeFilename(form.Surname))
	clientFolder := filepath.Join(pm.clientsDir, clientFolderName)

	if err := os.MkdirAll(clientFolder, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания директории: %w", err)
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Анкета"
	f.SetSheetName("Sheet1", sheetName)

	// Заполняем анкету
	data := [][]string{
		{"Анкета клиента", ""},
		{"", ""},
		{"ФИО", fmt.Sprintf("%s %s", form.Name, form.Surname)},
		{"Телефон", form.Phone},
		{"Дата рождения", form.BirthDate},
		{"Пол", form.Gender},
		{"Рост (см)", fmt.Sprintf("%d", form.Height)},
		{"Вес (кг)", fmt.Sprintf("%.1f", form.Weight)},
		{"", ""},
		{"Цель", form.Goal},
		{"Детали цели", form.GoalDetails},
		{"", ""},
		{"Опыт тренировок", form.Experience},
		{"Лет занимается", fmt.Sprintf("%.1f", form.ExperienceYears)},
		{"Дней в неделю может заниматься", fmt.Sprintf("%d", form.TrainingDays)},
		{"", ""},
		{"Травмы/ограничения", form.Injuries},
		{"Оборудование", form.Equipment},
		{"Предпочтения", form.Preferences},
		{"", ""},
		{"Дополнительные заметки", form.Notes},
		{"", ""},
		{"Дата заполнения", form.CreatedAt.Format("02.01.2006 15:04")},
	}

	for i, row := range data {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", i+1), row[0])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", i+1), row[1])
	}

	// Стили
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
	})
	f.SetCellStyle(sheetName, "A1", "A1", titleStyle)

	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	for _, row := range []int{3, 4, 5, 6, 7, 8, 10, 11, 13, 14, 15, 17, 18, 19, 21, 23} {
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), labelStyle)
	}

	f.SetColWidth(sheetName, "A", "A", 30)
	f.SetColWidth(sheetName, "B", "B", 50)

	// Сохраняем в папку клиента как Анкета.xlsx
	filePath := filepath.Join(clientFolder, "Анкета.xlsx")

	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("ошибка сохранения: %w", err)
	}

	return filePath, nil
}

// UpdateJournal обновляет или добавляет запись в журнал
func (pm *ProgramManager) UpdateJournal(entry *models.JournalEntry) error {
	var f *excelize.File

	// Открываем или создаём файл
	if _, statErr := os.Stat(pm.journalPath); os.IsNotExist(statErr) {
		f = excelize.NewFile()
		sheetName := "Журнал"
		f.SetSheetName("Sheet1", sheetName)

		// Заголовки
		headers := []string{"ID", "Имя", "Фамилия", "Телефон", "Telegram ID", "Цель", "Дата начала",
			"Всего тренировок", "Выполнено", "Текущая программа", "Статус", "Последняя тренировка", "Заметки"}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, h)
		}

		// Стиль заголовков
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center"},
		})
		f.SetCellStyle(sheetName, "A1", "M1", headerStyle)
	} else {
		var openErr error
		f, openErr = excelize.OpenFile(pm.journalPath)
		if openErr != nil {
			return fmt.Errorf("ошибка открытия журнала: %w", openErr)
		}
	}
	defer f.Close()

	sheetName := "Журнал"

	// Ищем существующую запись или добавляем новую
	rows, _ := f.GetRows(sheetName)
	targetRow := len(rows) + 1

	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) > 0 {
			var existingID int
			fmt.Sscanf(row[0], "%d", &existingID)
			if existingID == entry.ClientID {
				targetRow = i + 1
				break
			}
		}
	}

	// Записываем данные
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", targetRow), entry.ClientID)
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", targetRow), entry.Name)
	f.SetCellValue(sheetName, fmt.Sprintf("C%d", targetRow), entry.Surname)
	f.SetCellValue(sheetName, fmt.Sprintf("D%d", targetRow), entry.Phone)
	f.SetCellValue(sheetName, fmt.Sprintf("E%d", targetRow), entry.TelegramID)
	f.SetCellValue(sheetName, fmt.Sprintf("F%d", targetRow), entry.Goal)
	f.SetCellValue(sheetName, fmt.Sprintf("G%d", targetRow), entry.StartDate.Format("02.01.2006"))
	f.SetCellValue(sheetName, fmt.Sprintf("H%d", targetRow), entry.TotalWorkouts)
	f.SetCellValue(sheetName, fmt.Sprintf("I%d", targetRow), entry.CompletedWorkouts)
	f.SetCellValue(sheetName, fmt.Sprintf("J%d", targetRow), entry.CurrentProgram)
	f.SetCellValue(sheetName, fmt.Sprintf("K%d", targetRow), entry.Status)
	if entry.LastWorkout != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", targetRow), entry.LastWorkout.Format("02.01.2006"))
	}
	f.SetCellValue(sheetName, fmt.Sprintf("M%d", targetRow), entry.Notes)

	// Ширина столбцов
	f.SetColWidth(sheetName, "A", "A", 8)
	f.SetColWidth(sheetName, "B", "C", 15)
	f.SetColWidth(sheetName, "D", "D", 15)
	f.SetColWidth(sheetName, "E", "E", 12)
	f.SetColWidth(sheetName, "F", "F", 20)
	f.SetColWidth(sheetName, "G", "G", 12)
	f.SetColWidth(sheetName, "H", "I", 10)
	f.SetColWidth(sheetName, "J", "J", 20)
	f.SetColWidth(sheetName, "K", "K", 10)
	f.SetColWidth(sheetName, "L", "L", 15)
	f.SetColWidth(sheetName, "M", "M", 30)

	return f.SaveAs(pm.journalPath)
}

// sanitizeForFilename использует общую функцию sanitizeFilename из training_plan_export.go

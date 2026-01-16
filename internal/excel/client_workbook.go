package excel

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// ClientInfo информация о клиенте для создания персональной таблицы
type ClientInfo struct {
	ID        int
	TelegramID int64
	Name      string
	Surname   string
	Format    string // "онлайн" или "офлайн"
}

// CreateClientWorkbook создаёт персональную копию таблицы для клиента
// Копирует все листы из шаблона и добавляет информацию о клиенте
func CreateClientWorkbook(templatePath, outputDir string, client ClientInfo) (string, error) {
	// Проверяем существование шаблона
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return "", fmt.Errorf("шаблон не найден: %s", templatePath)
	}

	// Создаём директорию для файлов клиентов если не существует
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания директории: %w", err)
	}

	// Формируем имя файла клиента
	clientFileName := fmt.Sprintf("%s_%s.xlsx", client.Name, client.Surname)
	if client.Surname == "" {
		clientFileName = fmt.Sprintf("%s_%d.xlsx", client.Name, client.ID)
	}
	outputPath := filepath.Join(outputDir, clientFileName)

	// Открываем шаблон
	template, err := excelize.OpenFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия шаблона: %w", err)
	}
	defer template.Close()

	// Сохраняем копию как новый файл
	if err := template.SaveAs(outputPath); err != nil {
		return "", fmt.Errorf("ошибка сохранения копии: %w", err)
	}

	// Открываем созданную копию для модификации
	clientFile, err := excelize.OpenFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия файла клиента: %w", err)
	}
	defer func() {
		if err := clientFile.Save(); err != nil {
			log.Printf("Ошибка сохранения файла клиента: %v", err)
		}
		clientFile.Close()
	}()

	// Добавляем информацию о клиенте на лист Dashboard
	if err := addClientInfoToDashboard(clientFile, client); err != nil {
		log.Printf("Предупреждение: не удалось добавить информацию о клиенте: %v", err)
	}

	log.Printf("Создана таблица для клиента %s %s: %s", client.Name, client.Surname, outputPath)
	return outputPath, nil
}

// addClientInfoToDashboard добавляет информацию о клиенте на Dashboard
// Размещается в колонках AB-AF строки 2-5 (справа от всего, не пересекается с данными)
func addClientInfoToDashboard(f *excelize.File, client ClientInfo) error {
	sheet := SheetDashboard

	// Устанавливаем ширину колонок для блока клиента
	for _, col := range []string{"AB", "AC", "AD", "AE", "AF"} {
		f.SetColWidth(sheet, col, col, 4.5)
	}

	// Стиль для заголовка клиента
	clientTitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#1F4E79", Style: 2},
			{Type: "right", Color: "#1F4E79", Style: 2},
			{Type: "top", Color: "#1F4E79", Style: 2},
			{Type: "bottom", Color: "#1F4E79", Style: 2},
		},
	})

	clientInfoStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DEEBF7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#9BC2E6", Style: 1},
			{Type: "right", Color: "#9BC2E6", Style: 1},
			{Type: "bottom", Color: "#9BC2E6", Style: 1},
		},
	})

	// Блок клиента в колонках AB-AF, строки 2-5
	// Заголовок
	f.SetCellValue(sheet, "AB2", "КЛИЕНТ")
	f.MergeCell(sheet, "AB2", "AF2")
	f.SetCellStyle(sheet, "AB2", "AF2", clientTitleStyle)

	// Имя клиента
	clientName := fmt.Sprintf("%s %s", client.Name, client.Surname)
	f.SetCellValue(sheet, "AB3", clientName)
	f.MergeCell(sheet, "AB3", "AF3")
	f.SetCellStyle(sheet, "AB3", "AF3", clientInfoStyle)

	// Формат (онлайн/офлайн)
	format := client.Format
	if format == "" {
		format = "офлайн"
	}

	var formatStyle int
	if format == "онлайн" {
		formatStyle, _ = f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 10, Color: "#006100"},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			Border: []excelize.Border{
				{Type: "left", Color: "#9BC2E6", Style: 1},
				{Type: "right", Color: "#9BC2E6", Style: 1},
				{Type: "bottom", Color: "#9BC2E6", Style: 1},
			},
		})
	} else {
		formatStyle, _ = f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 10, Color: "#1F4E79"},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#BDD7EE"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			Border: []excelize.Border{
				{Type: "left", Color: "#9BC2E6", Style: 1},
				{Type: "right", Color: "#9BC2E6", Style: 1},
				{Type: "bottom", Color: "#9BC2E6", Style: 1},
			},
		})
	}
	f.SetCellValue(sheet, "AB4", format)
	f.MergeCell(sheet, "AB4", "AF4")
	f.SetCellStyle(sheet, "AB4", "AF4", formatStyle)

	// ID клиента
	f.SetCellValue(sheet, "AB5", fmt.Sprintf("ID: %d", client.ID))
	f.MergeCell(sheet, "AB5", "AF5")
	f.SetCellStyle(sheet, "AB5", "AF5", clientInfoStyle)

	return nil
}

// GetClientFolderName возвращает имя папки клиента
func GetClientFolderName(client ClientInfo) string {
	if client.Surname != "" {
		return fmt.Sprintf("%s_%s", client.Name, client.Surname)
	}
	return fmt.Sprintf("%s_%d", client.Name, client.ID)
}

// CreateClientWorkbookFromTemplate создаёт таблицу для клиента в его персональной папке
// Структура: Клиенты/Имя_Фамилия/Имя_Фамилия.xlsx
func CreateClientWorkbookFromTemplate(outputDir string, client ClientInfo) (string, error) {
	// Формируем путь к персональной папке клиента
	clientFolderName := GetClientFolderName(client)
	clientFolder := filepath.Join(outputDir, clientFolderName)

	// Создаём персональную папку клиента
	if err := os.MkdirAll(clientFolder, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания папки клиента: %w", err)
	}

	// Формируем имя файла клиента (в его папке)
	clientFileName := fmt.Sprintf("%s.xlsx", clientFolderName)
	outputPath := filepath.Join(clientFolder, clientFileName)

	// Создаём новую таблицу с помощью CreateHybridWorkbook
	if err := CreateHybridWorkbook(outputPath, nil); err != nil {
		return "", fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	// Открываем созданный файл для добавления информации о клиенте
	clientFile, err := excelize.OpenFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer func() {
		if err := clientFile.Save(); err != nil {
			log.Printf("Ошибка сохранения: %v", err)
		}
		clientFile.Close()
	}()

	// Добавляем информацию о клиенте
	if err := addClientInfoToDashboard(clientFile, client); err != nil {
		log.Printf("Предупреждение: %v", err)
	}

	log.Printf("Создана папка и таблица для клиента %s %s: %s", client.Name, client.Surname, outputPath)
	return outputPath, nil
}

// GetClientWorkbookPath возвращает путь к таблице клиента
// Структура: Клиенты/Имя_Фамилия/Имя_Фамилия.xlsx
func GetClientWorkbookPath(outputDir string, client ClientInfo) string {
	clientFolderName := GetClientFolderName(client)
	clientFileName := fmt.Sprintf("%s.xlsx", clientFolderName)
	return filepath.Join(outputDir, clientFolderName, clientFileName)
}

// GetClientFolderPath возвращает путь к папке клиента
func GetClientFolderPath(outputDir string, client ClientInfo) string {
	clientFolderName := GetClientFolderName(client)
	return filepath.Join(outputDir, clientFolderName)
}

// ClientWorkbookExists проверяет существование таблицы клиента
func ClientWorkbookExists(outputDir string, client ClientInfo) bool {
	path := GetClientWorkbookPath(outputDir, client)
	_, err := os.Stat(path)
	return err == nil
}

// CreateClientQuestionnaire создаёт пустую анкету клиента в его папке
func CreateClientQuestionnaire(outputDir string, client ClientInfo) (string, error) {
	clientFolder := GetClientFolderPath(outputDir, client)

	// Создаём папку клиента если не существует
	if err := os.MkdirAll(clientFolder, 0755); err != nil {
		return "", fmt.Errorf("ошибка создания папки клиента: %w", err)
	}

	questionnairePath := filepath.Join(clientFolder, "Анкета.xlsx")

	// Если анкета уже существует, не перезаписываем
	if _, err := os.Stat(questionnairePath); err == nil {
		return questionnairePath, nil
	}

	// Создаём новый файл анкеты
	f := excelize.NewFile()
	defer f.Close()

	// Основной лист - Анкета
	sheetName := "Анкета"
	f.SetSheetName("Sheet1", sheetName)

	// Заголовки анкеты
	headers := []struct {
		cell  string
		value string
		width float64
	}{
		{"A1", "Поле", 25},
		{"B1", "Значение", 40},
	}

	for _, h := range headers {
		f.SetCellValue(sheetName, h.cell, h.value)
		col := string(h.cell[0])
		f.SetColWidth(sheetName, col, col, h.width)
	}

	// Стиль заголовков
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheetName, "A1", "B1", headerStyle)

	// Поля анкеты
	fields := []string{
		"ФИО",
		"Дата рождения",
		"Телефон",
		"Email",
		"Цель тренировок",
		"Опыт тренировок",
		"Травмы/ограничения",
		"Хронические заболевания",
		"Текущий вес",
		"Целевой вес",
		"Рост",
		"Формат тренировок",
		"Предпочтения по времени",
		"Примечания",
	}

	for i, field := range fields {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), field)
	}

	// Заполняем известные данные
	f.SetCellValue(sheetName, "B2", fmt.Sprintf("%s %s", client.Name, client.Surname))

	// Стиль для полей
	fieldStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DEEBF7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})
	f.SetCellStyle(sheetName, "A2", fmt.Sprintf("A%d", len(fields)+1), fieldStyle)

	if err := f.SaveAs(questionnairePath); err != nil {
		return "", fmt.Errorf("ошибка сохранения анкеты: %w", err)
	}

	log.Printf("Создана анкета для клиента %s %s: %s", client.Name, client.Surname, questionnairePath)
	return questionnairePath, nil
}

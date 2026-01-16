package excel

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	SheetClients   = "Клиенты"
	SheetTrainings = "Тренировки"
)

// Пути к рабочим файлам - устанавливаются при инициализации через SetPaths
var (
	FilePath   string // Путь к журналу (например: ~/Desktop/Работа/Журнал.xlsx)
	ClientsDir string // Папка с таблицами клиентов (например: ~/Desktop/Работа/Клиенты)
)

// SetPaths устанавливает пути к рабочим файлам
func SetPaths(journalPath, clientsDir string) {
	FilePath = journalPath
	ClientsDir = clientsDir
}

// InitJournalFile инициализирует файл журнала клиентов
func InitJournalFile(db *sql.DB) error {
	journalPath := FilePath
	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		return CreateJournalFile(journalPath, db)
	}
	return nil
}

// CreateTemplateIfNotExists создаёт Excel шаблон если файл не существует
func CreateTemplateIfNotExists(filePath string) error {
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	f := excelize.NewFile()
	defer f.Close()

	clientsIndex, err := f.NewSheet(SheetClients)
	if err != nil {
		return err
	}

	_, err = f.NewSheet(SheetTrainings)
	if err != nil {
		return err
	}

	f.SetActiveSheet(clientsIndex)
	f.DeleteSheet("Sheet1")

	// Лист "Клиенты"
	clientHeaders := []string{"id", "Имя", "Фамилия", "Телефон", "Дата рождения", "Telegram ID"}
	for i, h := range clientHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(SheetClients, cell, h)
	}

	styleClients, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(SheetClients, "A1", "F1", styleClients)

	f.SetColWidth(SheetClients, "A", "A", 8)
	f.SetColWidth(SheetClients, "B", "F", 15)

	// Лист "Тренировки"
	trainingHeaders := []string{"client_id", "Дата", "Время", "Описание тренировки", "Отправлено"}
	for i, h := range trainingHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(SheetTrainings, cell, h)
	}

	styleTrainings, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(SheetTrainings, "A1", "E1", styleTrainings)

	f.SetColWidth(SheetTrainings, "A", "A", 12)
	f.SetColWidth(SheetTrainings, "B", "C", 12)
	f.SetColWidth(SheetTrainings, "D", "D", 50)
	f.SetColWidth(SheetTrainings, "E", "E", 12)

	return f.SaveAs(filePath)
}

// SyncClientsFromDB синхронизирует клиентов из БД в Журнал.xlsx
func SyncClientsFromDB(filePath string, db *sql.DB) error {
	// Создаём журнал клиентов если не существует
	if err := InitJournalFile(db); err != nil {
		log.Printf("Ошибка создания журнала: %v", err)
		return err
	}
	return nil
}

// AddClientToExcel добавляет нового клиента - создаёт персональную папку с таблицей и анкетой
// Структура: Клиенты/Имя_Фамилия/
//            ├── Имя_Фамилия.xlsx (таблица тренировок)
//            └── Анкета.xlsx (анкета клиента)
func AddClientToExcel(filePath string, clientID int, name, surname, phone, birthDate string, telegramID int64) error {
	// Проверяем, что пути установлены
	if ClientsDir == "" {
		return fmt.Errorf("ClientsDir не установлен, вызовите SetPaths()")
	}

	// Создаём информацию о клиенте
	client := ClientInfo{
		ID:         clientID,
		TelegramID: telegramID,
		Name:       name,
		Surname:    surname,
		Format:     "онлайн", // по умолчанию онлайн
	}

	// Создаём персональную папку и таблицу клиента
	_, err := CreateClientWorkbookFromTemplate(ClientsDir, client)
	if err != nil {
		log.Printf("Ошибка создания таблицы для клиента %s %s: %v", name, surname, err)
		return err
	}

	// Создаём анкету клиента
	_, err = CreateClientQuestionnaire(ClientsDir, client)
	if err != nil {
		log.Printf("Ошибка создания анкеты для клиента %s %s: %v", name, surname, err)
		// Не возвращаем ошибку - таблица уже создана
	}

	return nil
}

// GetShortMonthName возвращает сокращённое название месяца
func GetShortMonthName(month time.Month) string {
	months := map[time.Month]string{
		time.January: "Янв", time.February: "Фев", time.March: "Мар",
		time.April: "Апр", time.May: "Май", time.June: "Июн",
		time.July: "Июл", time.August: "Авг", time.September: "Сен",
		time.October: "Окт", time.November: "Ноя", time.December: "Дек",
	}
	return months[month]
}

// GetMonthlySheetName формирует имя листа с месяцем и годом
func GetMonthlySheetName(name, surname string, t time.Time) string {
	monthName := GetShortMonthName(t.Month())
	surnameShort := surname
	if len([]rune(surname)) > 1 {
		surnameShort = string([]rune(surname)[0]) + "."
	}
	sheetName := fmt.Sprintf("%s %s - %s %d", name, surnameShort, monthName, t.Year())
	if len(sheetName) > 31 {
		nameRunes := []rune(name)
		if len(nameRunes) > 6 {
			sheetName = fmt.Sprintf("%s. %s - %s %d", string(nameRunes[0]), surnameShort, monthName, t.Year())
		}
	}
	return sheetName
}

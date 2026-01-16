package excel

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"workbot/internal/models"

	"github.com/fsnotify/fsnotify"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// cleanExerciseName убирает номер из начала названия упражнения (если есть)
func cleanExerciseName(name string) string {
	// Убираем "1. ", "1.", "1) " и т.д. из начала
	re := regexp.MustCompile(`^\d+[\.\)]\s*`)
	return strings.TrimSpace(re.ReplaceAllString(name, ""))
}

// Watcher следит за изменениями в Excel файле
type Watcher struct {
	bot      *tgbotapi.BotAPI
	db       *sql.DB
	filePath string
}

// NewWatcher создаёт новый наблюдатель за Excel файлом
func NewWatcher(bot *tgbotapi.BotAPI, db *sql.DB, filePath string) *Watcher {
	return &Watcher{
		bot:      bot,
		db:       db,
		filePath: filePath,
	}
}

// StartWatching запускает наблюдение за файлами
func (w *Watcher) StartWatching() {
	// Проверяем, что пути установлены
	if FilePath == "" || ClientsDir == "" {
		log.Printf("ВНИМАНИЕ: Пути к Excel файлам не установлены! Вызовите SetPaths() перед StartWatching()")
		return
	}

	// Создаём журнал клиентов если не существует
	if err := InitJournalFile(w.db); err != nil {
		log.Printf("Ошибка инициализации журнала: %v", err)
	}

	// Создаём папку клиентов если не существует
	if err := os.MkdirAll(ClientsDir, 0755); err != nil {
		log.Printf("Ошибка создания папки клиентов: %v", err)
	}

	go w.startDBSync(30 * time.Second)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Ошибка создания наблюдателя: %v", err)
		return
	}

	go func() {
		defer watcher.Close()

		var lastEvent time.Time

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					if time.Since(lastEvent) < 2*time.Second {
						continue
					}
					lastEvent = time.Now()

					log.Printf("Excel файл изменён: %s", event.Name)
					// Автоматическая отправка отключена - тренер отправляет вручную через бота
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Ошибка наблюдателя: %v", err)
			}
		}
	}()

	// Следим за папкой с таблицами клиентов
	if err := watcher.Add(ClientsDir); err != nil {
		log.Printf("Ошибка добавления папки clients в наблюдатель: %v", err)
	}

	// Следим за журналом клиентов
	if err := watcher.Add(FilePath); err != nil {
		log.Printf("Ошибка добавления журнала в наблюдатель: %v", err)
	}

	log.Printf("Наблюдение за %s и %s запущено", ClientsDir, FilePath)
}

// GetUnsentTrainings возвращает неотправленные тренировки клиента
func GetUnsentTrainings(filePath string, db *sql.DB, clientID int) ([]models.TrainingGroup, error) {
	// Используем единый лист данных
	return GetUnsentTrainingsUnified(filePath, clientID)
}

// FormatTrainingMessage форматирует тренировку для отправки
func FormatTrainingMessage(group *models.TrainingGroup) string {
	header := "Тренировка"
	if group.TrainingNum > 0 {
		header = fmt.Sprintf("Тренировка #%d", group.TrainingNum)
	}

	var exercises string
	var totalTonnage float64
	var totalVolume int

	for i, ex := range group.Exercises {
		var weightStr string
		if ex.Weight > 0 {
			weightStr = fmt.Sprintf("%.0f кг", ex.Weight)
			tonnage := float64(ex.Sets) * float64(ex.Reps) * ex.Weight
			totalTonnage += tonnage
		} else {
			weightStr = "свой вес"
			totalVolume += ex.Sets * ex.Reps
		}

		exercises += fmt.Sprintf("%d. %s\n   %dx%d, %s\n",
			i+1, cleanExerciseName(ex.Exercise), ex.Sets, ex.Reps, weightStr)
	}

	var summary string
	if totalTonnage > 0 {
		summary = fmt.Sprintf("Общий тоннаж: %.0f кг", totalTonnage)
	}
	if totalVolume > 0 {
		if summary != "" {
			summary += "\n"
		}
		summary += fmt.Sprintf("Объём (свой вес): %d повторений", totalVolume)
	}

	return fmt.Sprintf("%s\n\n%s\n%s\n\nУдачной тренировки!", header, exercises, summary)
}

// MarkTrainingGroupAsSent помечает все упражнения группы как отправленные
func MarkTrainingGroupAsSent(filePath string, group *models.TrainingGroup) error {
	for _, ex := range group.Exercises {
		if err := MarkUnifiedTrainingAsSent(filePath, ex.RowNum); err != nil {
			return err
		}
	}
	return nil
}

func (w *Watcher) startDBSync(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Фоновая синхронизация БД -> Excel запущена (интервал: %v)", interval)

	for range ticker.C {
		if err := SyncClientsFromDB(w.filePath, w.db); err != nil {
			log.Printf("Ошибка фоновой синхронизации: %v", err)
		}
	}
}

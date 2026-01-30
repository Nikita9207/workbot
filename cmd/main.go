package main

import (
	"database/sql"
	"log"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"

	"workbot/internal/bot"
	"workbot/internal/config"
	"workbot/internal/excel"
	"workbot/internal/i18n"
)

func main() {
	// Загружаем конфигурацию из .env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализируем Telegram Bot API
	botAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram бота: %v", err)
	}
	botAPI.Debug = true
	log.Printf("Бот авторизован как %s", botAPI.Self.UserName)

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка проверки соединения с БД: %v", err)
	}
	log.Println("Подключение к базе данных установлено")

	// Загружаем локализацию
	localesDir := filepath.Join(cfg.WorkDir, "locales")
	if err := i18n.Load(localesDir); err != nil {
		log.Printf("Предупреждение: ошибка загрузки локализации: %v", err)
	}

	// Устанавливаем пути для Excel файлов
	excel.SetPaths(cfg.JournalPath, cfg.ClientsDir)
	log.Printf("Рабочая директория: %s", cfg.WorkDir)
	log.Printf("Журнал: %s", cfg.JournalPath)
	log.Printf("Клиенты: %s", cfg.ClientsDir)

	// Запускаем наблюдение за Excel файлами
	excelWatcher := excel.NewWatcher(botAPI, db, cfg.WorkDir)
	excelWatcher.StartWatching()

	// Запускаем бота
	telegramBot := bot.New(botAPI, db, cfg)
	log.Println("Бот запущен и готов к работе")
	if err := telegramBot.Start(); err != nil {
		log.Fatal(err)
	}
}

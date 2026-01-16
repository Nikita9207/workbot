package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config содержит конфигурацию приложения
type Config struct {
	BotToken   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	GroqAPIKey string

	// Пути к рабочим файлам
	WorkDir     string // ~/Desktop/Работа
	ClientsDir  string // ~/Desktop/Работа/Клиенты (папки клиентов с таблицами и анкетами)
	JournalPath string // ~/Desktop/Работа/Журнал.xlsx

	// Google Sheets
	GoogleCredentialsPath string // Путь к google-credentials.json
	GoogleDriveFolderID   string // ID папки в Google Drive
}

// Load загружает конфигурацию из переменных окружения или .env файла
func Load() (*Config, error) {
	// Сначала пробуем загрузить из .env файла (для локальной разработки)
	env, err := loadEnvFile(".env")
	if err != nil {
		// Если .env не найден, используем переменные окружения (Docker)
		env = make(map[string]string)
	}

	// Функция для получения значения: сначала из окружения, потом из .env
	getEnv := func(key, defaultValue string) string {
		// Приоритет: переменные окружения > .env файл > значение по умолчанию
		if value := os.Getenv(key); value != "" {
			return value
		}
		if value, ok := env[key]; ok && value != "" {
			return value
		}
		return defaultValue
	}

	// Определяем домашнюю директорию
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	workDir := getEnv("WORK_DIR", homeDir+"/Desktop/Работа")

	cfg := &Config{
		BotToken:   getEnv("BOT_TOKEN", ""),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "postgres"),
		GroqAPIKey: getEnv("GROQ_API_KEY", ""),

		WorkDir:     workDir,
		ClientsDir:  workDir + "/Клиенты",
		JournalPath: workDir + "/Журнал.xlsx",

		GoogleCredentialsPath: getEnv("GOOGLE_CREDENTIALS_PATH", "google-credentials.json"),
		GoogleDriveFolderID:   getEnv("GOOGLE_DRIVE_FOLDER_ID", ""),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN не задан")
	}

	return cfg, nil
}

// DSN возвращает строку подключения к базе данных
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

// loadEnvFile читает .env файл и возвращает map переменных
func loadEnvFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Разделяем по первому знаку равенства
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Удаляем кавычки, если есть
		value = strings.Trim(value, `"'`)

		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return env, nil
}

// getOrDefault возвращает значение из map или значение по умолчанию
func getOrDefault(env map[string]string, key, defaultValue string) string {
	if value, ok := env[key]; ok && value != "" {
		return value
	}
	return defaultValue
}

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

	// Пути к рабочим файлам
	WorkDir     string // ~/Desktop/Работа
	ClientsDir  string // ~/Desktop/Работа/Клиенты
	JournalPath string // ~/Desktop/Работа/Журнал.xlsx

	// Google Sheets
	GoogleCredentialsPath string
	GoogleDriveFolderID   string

	// Ollama
	OllamaURL   string // URL Ollama, например http://localhost:11434
	OllamaModel string // Модель, например gemma2:9b-instruct-q4_K_M

	// Groq (только для транскрипции голоса)
	GroqAPIKey string

	// RAG индекс (готовый, через Tailscale)
	RAGIndexPath string // Путь к готовому индексу knowledge.json
}

// Load загружает конфигурацию из переменных окружения или .env файла
func Load() (*Config, error) {
	env, err := loadEnvFile(".env")
	if err != nil {
		env = make(map[string]string)
	}

	getEnv := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		if value, ok := env[key]; ok && value != "" {
			return value
		}
		return defaultValue
	}

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

		WorkDir:     workDir,
		ClientsDir:  workDir + "/Клиенты",
		JournalPath: workDir + "/Журнал.xlsx",

		GoogleCredentialsPath: getEnv("GOOGLE_CREDENTIALS_PATH", "google-credentials.json"),
		GoogleDriveFolderID:   getEnv("GOOGLE_DRIVE_FOLDER_ID", ""),

		// Ollama
		OllamaURL:   getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel: getEnv("OLLAMA_MODEL", "gemma2:9b-instruct-q4_K_M"),

		// Groq (только для транскрипции)
		GroqAPIKey: getEnv("GROQ_API_KEY", ""),

		// Готовый RAG индекс
		RAGIndexPath: getEnv("RAG_INDEX_PATH", ""),
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

// loadEnvFile читает .env файл
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
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		env[key] = value
	}

	return env, scanner.Err()
}

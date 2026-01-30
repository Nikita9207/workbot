package i18n

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Language представляет поддерживаемый язык
type Language string

const (
	LangRussian Language = "ru"
	LangEnglish Language = "en"
	DefaultLang Language = LangRussian
)

// translations хранит все переводы
var translations = struct {
	sync.RWMutex
	data map[Language]map[string]string
}{data: make(map[Language]map[string]string)}

// Load загружает переводы из файлов локализации
func Load(localesDir string) error {
	translations.Lock()
	defer translations.Unlock()

	languages := []Language{LangRussian, LangEnglish}

	for _, lang := range languages {
		filePath := filepath.Join(localesDir, string(lang)+".json")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("ошибка чтения файла локализации %s: %w", filePath, err)
		}

		var langData map[string]string
		if err := json.Unmarshal(data, &langData); err != nil {
			return fmt.Errorf("ошибка парсинга файла локализации %s: %w", filePath, err)
		}

		translations.data[lang] = langData
		log.Printf("Загружена локализация: %s (%d ключей)", lang, len(langData))
	}

	return nil
}

// T возвращает перевод для указанного ключа и языка
func T(key string, lang Language) string {
	translations.RLock()
	defer translations.RUnlock()

	// Пробуем получить перевод для указанного языка
	if langData, ok := translations.data[lang]; ok {
		if text, ok := langData[key]; ok {
			return text
		}
	}

	// Fallback на русский
	if lang != DefaultLang {
		if langData, ok := translations.data[DefaultLang]; ok {
			if text, ok := langData[key]; ok {
				return text
			}
		}
	}

	// Если ключ не найден, возвращаем сам ключ
	log.Printf("Перевод не найден: key=%s, lang=%s", key, lang)
	return key
}

// Tf возвращает форматированный перевод
func Tf(key string, lang Language, args ...interface{}) string {
	template := T(key, lang)
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}

// IsValidLanguage проверяет, является ли язык поддерживаемым
func IsValidLanguage(lang string) bool {
	switch Language(strings.ToLower(lang)) {
	case LangRussian, LangEnglish:
		return true
	default:
		return false
	}
}

// ParseLanguage преобразует строку в Language
func ParseLanguage(lang string) Language {
	switch Language(strings.ToLower(lang)) {
	case LangEnglish:
		return LangEnglish
	default:
		return LangRussian
	}
}

// GetLanguageName возвращает название языка на этом языке
func GetLanguageName(lang Language) string {
	switch lang {
	case LangEnglish:
		return "English"
	default:
		return "Русский"
	}
}

// GetLanguageFlag возвращает флаг для языка
func GetLanguageFlag(lang Language) string {
	switch lang {
	case LangEnglish:
		return "🇬🇧"
	default:
		return "🇷🇺"
	}
}

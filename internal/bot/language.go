package bot

import (
	"log"
	"sync"

	"workbot/internal/i18n"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// langCache кэширует язык пользователей
var langCache = struct {
	sync.RWMutex
	cache map[int64]i18n.Language
}{cache: make(map[int64]i18n.Language)}

// getLanguage возвращает язык пользователя (с кэшированием)
func (b *Bot) getLanguage(telegramID int64) i18n.Language {
	// Проверяем кэш
	langCache.RLock()
	if lang, ok := langCache.cache[telegramID]; ok {
		langCache.RUnlock()
		return lang
	}
	langCache.RUnlock()

	// Запрос к БД
	var langStr string
	err := b.db.QueryRow("SELECT COALESCE(language, 'ru') FROM public.clients WHERE telegram_id = $1", telegramID).Scan(&langStr)
	if err != nil {
		// Если клиент не найден — используем русский
		return i18n.DefaultLang
	}

	lang := i18n.ParseLanguage(langStr)

	// Кэшируем результат
	langCache.Lock()
	langCache.cache[telegramID] = lang
	langCache.Unlock()

	return lang
}

// setLanguage устанавливает язык пользователя
func (b *Bot) setLanguage(telegramID int64, lang i18n.Language) error {
	_, err := b.db.Exec("UPDATE public.clients SET language = $1 WHERE telegram_id = $2", string(lang), telegramID)
	if err != nil {
		return err
	}

	// Обновляем кэш
	langCache.Lock()
	langCache.cache[telegramID] = lang
	langCache.Unlock()

	return nil
}

// clearLanguageCache очищает кэш языка для пользователя
func clearLanguageCache(telegramID int64) {
	langCache.Lock()
	delete(langCache.cache, telegramID)
	langCache.Unlock()
}

// t возвращает перевод для пользователя
func (b *Bot) t(key string, telegramID int64) string {
	lang := b.getLanguage(telegramID)
	return i18n.T(key, lang)
}

// tf возвращает форматированный перевод для пользователя
func (b *Bot) tf(key string, telegramID int64, args ...interface{}) string {
	lang := b.getLanguage(telegramID)
	return i18n.Tf(key, lang, args...)
}

// handleSettingsMenu показывает меню настроек
func (b *Bot) handleSettingsMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	lang := b.getLanguage(chatID)
	langName := i18n.GetLanguageName(lang)
	langFlag := i18n.GetLanguageFlag(lang)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.tf("settings_language", chatID, langFlag+" "+langName),
				"settings_language",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(b.t("back", chatID), "settings_back"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, b.t("settings_title", chatID))
	msg.ReplyMarkup = keyboard
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Ошибка отправки меню настроек: %v", err)
	}
}

// handleLanguageSelection показывает выбор языка
func (b *Bot) handleLanguageSelection(chatID int64, messageID int) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇷🇺 Русский", "lang_ru"),
			tgbotapi.NewInlineKeyboardButtonData("🇬🇧 English", "lang_en"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(b.t("back", chatID), "settings_back"),
		),
	)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, b.t("settings_select_language", chatID))
	edit.ReplyMarkup = &keyboard
	if _, err := b.api.Send(edit); err != nil {
		log.Printf("Ошибка редактирования сообщения: %v", err)
	}
}

// handleLanguageChange меняет язык пользователя
func (b *Bot) handleLanguageChange(chatID int64, messageID int, langCode string) {
	lang := i18n.ParseLanguage(langCode)

	if err := b.setLanguage(chatID, lang); err != nil {
		log.Printf("Ошибка установки языка: %v", err)
		return
	}

	langName := i18n.GetLanguageName(lang)
	text := b.tf("settings_language_changed", chatID, langName)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if _, err := b.api.Send(edit); err != nil {
		log.Printf("Ошибка редактирования сообщения: %v", err)
	}

	// Обновляем главное меню
	b.restoreMainMenu(chatID)
}

// handleSettingsCallback обрабатывает callback-запросы настроек
func (b *Bot) handleSettingsCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID
	data := callback.Data

	// Подтверждаем получение callback
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	switch data {
	case "settings_language":
		b.handleLanguageSelection(chatID, messageID)
	case "lang_ru":
		b.handleLanguageChange(chatID, messageID, "ru")
	case "lang_en":
		b.handleLanguageChange(chatID, messageID, "en")
	case "settings_back":
		b.restoreMainMenu(chatID)
		// Удаляем inline-сообщение
		b.api.Send(tgbotapi.NewDeleteMessage(chatID, messageID))
	}
}

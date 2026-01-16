package bot

import (
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// sendError sends error message to user and logs it
func (b *Bot) sendError(chatID int64, userMessage string, err error) {
	if err != nil {
		log.Printf("Error [chat=%d]: %v", chatID, err)
	}
	msg := tgbotapi.NewMessage(chatID, userMessage)
	if _, sendErr := b.api.Send(msg); sendErr != nil {
		log.Printf("Failed to send error message [chat=%d]: %v", chatID, sendErr)
	}
}

// sendMessage sends message to user with error logging
func (b *Bot) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Failed to send message [chat=%d]: %v", chatID, err)
	}
	return err
}

// sendMessageWithKeyboard sends message with keyboard
func (b *Bot) sendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Failed to send message with keyboard [chat=%d]: %v", chatID, err)
	}
	return err
}

// parseIDFromBrackets extracts ID from text like "Some text [123]"
func parseIDFromBrackets(text string) int {
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		return 0
	}
	id, err := strconv.Atoi(text[start+1 : end])
	if err != nil {
		return 0
	}
	return id
}

// createCancelKeyboard creates a simple keyboard with just Cancel button
func createCancelKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
}

// setState sets user state with proper locking
func setState(chatID int64, state string) {
	userStates.Lock()
	userStates.states[chatID] = state
	userStates.Unlock()
}

// getState gets user state with proper locking
func getState(chatID int64) string {
	userStates.RLock()
	defer userStates.RUnlock()
	return userStates.states[chatID]
}

// clearState clears user state
func clearState(chatID int64) {
	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()
}

// safeFloat64 safely converts string to float64, returns 0 on error
func safeFloat64(s string) float64 {
	s = strings.Replace(s, ",", ".", 1)
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// safeInt safely converts string to int, returns 0 on error
func safeInt(s string) int {
	s = strings.TrimSpace(s)
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// truncateString truncates string to maxLen with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

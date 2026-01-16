package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// trainerStore хранит данные добавления тренера
var trainerStore = struct {
	sync.RWMutex
	data map[int64]*AddTrainerData
}{data: make(map[int64]*AddTrainerData)}

// AddTrainerData хранит данные добавления тренера
type AddTrainerData struct {
	TelegramID int64
	Name       string
	Step       int // 0=telegram_id, 1=name
}

// Константы состояний добавления тренера
const (
	stateAddTrainerID   = "add_trainer_id"
	stateAddTrainerName = "add_trainer_name"
)

// handleTrainersMenu показывает меню управления тренерами
func (b *Bot) handleTrainersMenu(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Получаем список тренеров
	rows, err := b.db.Query("SELECT telegram_id, name FROM public.admins ORDER BY name")
	if err != nil {
		log.Printf("Ошибка получения тренеров: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки списка тренеров")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var trainers []string
	for rows.Next() {
		var telegramID int64
		var name string
		if err := rows.Scan(&telegramID, &name); err != nil {
			continue
		}
		trainers = append(trainers, fmt.Sprintf("• %s (ID: %d)", name, telegramID))
	}

	text := "Управление тренерами\n\n"
	if len(trainers) > 0 {
		text += "Текущие тренеры:\n" + strings.Join(trainers, "\n")
	} else {
		text += "Тренеров пока нет"
	}

	msg := tgbotapi.NewMessage(chatID, text)
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Добавить тренера"),
			tgbotapi.NewKeyboardButton("Удалить тренера"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleAddTrainer начинает процесс добавления тренера
func (b *Bot) handleAddTrainer(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	trainerStore.Lock()
	trainerStore.data[chatID] = &AddTrainerData{Step: 0}
	trainerStore.Unlock()

	userStates.Lock()
	userStates.states[chatID] = stateAddTrainerID
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Добавление тренера\n\nВведите Telegram ID нового тренера:\n\n(Тренер может узнать свой ID написав боту @userinfobot)")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	b.api.Send(msg)
}

// processAddTrainer обрабатывает шаги добавления тренера
func (b *Bot) processAddTrainer(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.cancelAddTrainer(chatID, message)
		return
	}

	trainerStore.Lock()
	data := trainerStore.data[chatID]
	if data == nil {
		trainerStore.Unlock()
		b.cancelAddTrainer(chatID, message)
		return
	}

	switch state {
	case stateAddTrainerID:
		// Парсим Telegram ID
		telegramID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			trainerStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Некорректный Telegram ID. Введите число:")
			b.api.Send(msg)
			return
		}

		// Проверяем, не добавлен ли уже
		var exists bool
		b.db.QueryRow("SELECT EXISTS(SELECT 1 FROM public.admins WHERE telegram_id = $1)", telegramID).Scan(&exists)
		if exists {
			trainerStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Этот пользователь уже является тренером. Введите другой ID:")
			b.api.Send(msg)
			return
		}

		data.TelegramID = telegramID
		data.Step = 1
		trainerStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateAddTrainerName
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите имя тренера (как будет отображаться в системе):")
		b.api.Send(msg)

	case stateAddTrainerName:
		name := strings.TrimSpace(text)
		if len(name) < 2 {
			trainerStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "Имя должно содержать минимум 2 символа. Введите имя:")
			b.api.Send(msg)
			return
		}

		telegramID := data.TelegramID
		delete(trainerStore.data, chatID)
		trainerStore.Unlock()

		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()

		// Сохраняем тренера в БД
		_, err := b.db.Exec(
			"INSERT INTO public.admins (telegram_id, name) VALUES ($1, $2) ON CONFLICT (telegram_id) DO UPDATE SET name = $2",
			telegramID, name,
		)
		if err != nil {
			log.Printf("Ошибка добавления тренера: %v", err)
			msg := tgbotapi.NewMessage(chatID, "Ошибка при добавлении тренера")
			b.api.Send(msg)
			b.handleTrainersMenu(message)
			return
		}

		// Очищаем кэш админов
		adminCache.Lock()
		delete(adminCache.cache, telegramID)
		adminCache.Unlock()

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Тренер добавлен!\n\nИмя: %s\nTelegram ID: %d\n\nТеперь этот пользователь может использовать панель тренера.",
			name, telegramID,
		))
		b.api.Send(msg)
		b.handleTrainersMenu(message)

	default:
		trainerStore.Unlock()
	}
}

// handleRemoveTrainer показывает список тренеров для удаления
func (b *Bot) handleRemoveTrainer(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	rows, err := b.db.Query("SELECT telegram_id, name FROM public.admins ORDER BY name")
	if err != nil {
		log.Printf("Ошибка получения тренеров: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки списка тренеров")
		b.api.Send(msg)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var telegramID int64
		var name string
		if err := rows.Scan(&telegramID, &name); err != nil {
			continue
		}
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("Удалить: %s [%d]", name, telegramID)),
		))
	}

	if len(buttons) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Нет тренеров для удаления")
		b.api.Send(msg)
		b.handleTrainersMenu(message)
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Отмена"),
	))

	userStates.Lock()
	userStates.states[chatID] = "remove_trainer_select"
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Выберите тренера для удаления:")
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard:       buttons,
		ResizeKeyboard: true,
	}
	b.api.Send(msg)
}

// handleRemoveTrainerSelection обрабатывает выбор тренера для удаления
func (b *Bot) handleRemoveTrainerSelection(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()
		b.handleTrainersMenu(message)
		return
	}

	if !strings.HasPrefix(text, "Удалить: ") {
		msg := tgbotapi.NewMessage(chatID, "Выберите тренера из списка")
		b.api.Send(msg)
		return
	}

	// Извлекаем Telegram ID из строки "Удалить: Имя [123456789]"
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || start >= end {
		msg := tgbotapi.NewMessage(chatID, "Ошибка: неверный формат")
		b.api.Send(msg)
		return
	}

	telegramID, err := strconv.ParseInt(text[start+1:end], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка: неверный ID")
		b.api.Send(msg)
		return
	}

	// Проверяем, не пытается ли админ удалить себя
	if telegramID == chatID {
		msg := tgbotapi.NewMessage(chatID, "Вы не можете удалить себя из списка тренеров")
		b.api.Send(msg)
		return
	}

	// Удаляем тренера
	_, err = b.db.Exec("DELETE FROM public.admins WHERE telegram_id = $1", telegramID)
	if err != nil {
		log.Printf("Ошибка удаления тренера: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при удалении тренера")
		b.api.Send(msg)
		b.handleTrainersMenu(message)
		return
	}

	// Очищаем кэш админов
	adminCache.Lock()
	delete(adminCache.cache, telegramID)
	adminCache.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Тренер удалён")
	b.api.Send(msg)
	b.handleTrainersMenu(message)
}

// cancelAddTrainer отменяет добавление тренера
func (b *Bot) cancelAddTrainer(chatID int64, message *tgbotapi.Message) {
	trainerStore.Lock()
	delete(trainerStore.data, chatID)
	trainerStore.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Отменено")
	b.api.Send(msg)
	b.handleTrainersMenu(message)
}

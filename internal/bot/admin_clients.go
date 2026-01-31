package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// showClientsList показывает список клиентов
func (b *Bot) showClientsList(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	rows, err := b.db.Query(`
		SELECT c.id, c.name, c.surname
		FROM public.clients c
		LEFT JOIN public.admins a ON c.telegram_id = a.telegram_id
		WHERE a.telegram_id IS NULL AND c.deleted_at IS NULL
		ORDER BY c.name`)
	if err != nil {
		b.sendError(chatID, "Ошибка загрузки клиентов", err)
		return
	}
	defer rows.Close()

	var buttons [][]tgbotapi.KeyboardButton
	for rows.Next() {
		var id int
		var name, surname string
		if err := rows.Scan(&id, &name, &surname); err != nil {
			log.Printf("Ошибка сканирования клиента: %v", err)
			continue
		}
		buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf(">> %s %s [%d]", name, surname, id)),
		))
	}
	if err := rows.Err(); err != nil {
		log.Printf("Ошибка итерации по клиентам: %v", err)
	}

	if len(buttons) == 0 {
		b.sendMessage(chatID, "Список клиентов пуст")
		return
	}

	buttons = append(buttons, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Назад"),
	))

	keyboard := tgbotapi.NewReplyKeyboard(buttons...)
	b.sendMessageWithKeyboard(chatID, "Выберите клиента:", keyboard)
}

// handleClientSelection обрабатывает выбор клиента — показывает профиль
func (b *Bot) handleClientSelection(message *tgbotapi.Message, text string) {
	chatID := message.Chat.ID

	clientID := parseIDFromBrackets(text)
	if clientID == 0 {
		b.sendMessage(chatID, "Ошибка выбора клиента")
		return
	}

	adminStates.Lock()
	adminStates.selectedClient[chatID] = clientID
	adminStates.Unlock()

	b.showClientProfile(chatID, clientID)
}

// showClientProfile показывает профиль клиента с меню действий
func (b *Bot) showClientProfile(chatID int64, clientID int) {
	var name, surname, phone, birthDate string
	var goal, trainingPlan, notes sql.NullString
	err := b.db.QueryRow(`
		SELECT name, surname, COALESCE(phone, ''), COALESCE(birth_date, ''),
		       goal, training_plan, notes
		FROM public.clients WHERE id = $1`, clientID).
		Scan(&name, &surname, &phone, &birthDate, &goal, &trainingPlan, &notes)
	if err != nil {
		b.sendError(chatID, "Клиент не найден", err)
		b.handleAdminStart(&tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}})
		return
	}

	var profile strings.Builder
	profile.WriteString(fmt.Sprintf("Клиент: %s %s\n", name, surname))
	profile.WriteString("-------------------\n")

	if phone != "" {
		profile.WriteString(fmt.Sprintf("Телефон: %s\n", phone))
	}
	if birthDate != "" {
		profile.WriteString(fmt.Sprintf("Дата рождения: %s\n", birthDate))
	}

	profile.WriteString("\n")

	if goal.Valid && goal.String != "" {
		profile.WriteString(fmt.Sprintf("Цель: %s\n", goal.String))
	} else {
		profile.WriteString("Цель: не задана\n")
	}

	if trainingPlan.Valid && trainingPlan.String != "" {
		planPreview := trainingPlan.String
		if len(planPreview) > 200 {
			planPreview = planPreview[:200] + "..."
		}
		profile.WriteString(fmt.Sprintf("\nПлан:\n%s\n", planPreview))
	} else {
		profile.WriteString("План: не составлен\n")
	}

	if notes.Valid && notes.String != "" {
		profile.WriteString(fmt.Sprintf("\nЗаметки: %s\n", notes.String))
	}

	profile.WriteString("\n-------------------\n")
	profile.WriteString("Выберите действие:")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 Прогресс программы"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Записать тренировку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("PL: Программа"),
			tgbotapi.NewKeyboardButton("FIT: Программа"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Задать цель"),
			tgbotapi.NewKeyboardButton("Составить план"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("История"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Удалить клиента"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	b.sendMessageWithKeyboard(chatID, profile.String(), keyboard)

	setState(chatID, "viewing_client")
}

// handleClientAction обрабатывает действия с клиентом
func (b *Bot) handleClientAction(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	adminStates.RLock()
	clientID := adminStates.selectedClient[chatID]
	adminStates.RUnlock()

	if clientID == 0 {
		b.handleAdminStart(message)
		return
	}

	switch text {
	case "📊 Прогресс программы":
		b.showProgramProgress(clientID, chatID)
	case "Записать тренировку":
		b.startTrainingInput(chatID, clientID)
	case "PL: Программа":
		// Переход к пауэрлифтинг программе с предвыбранным клиентом
		b.handlePLProgramForClient(message, clientID)
	case "FIT: Программа":
		// Переход к фитнес программе с предвыбранным клиентом
		b.handleFITProgramForClient(message, clientID)
	case "Задать цель":
		b.startSetGoal(chatID, clientID)
	case "Составить план":
		b.startCreatePlan(chatID, clientID)
	case "История":
		b.showClientHistory(chatID, clientID)
	case "Удалить клиента":
		b.confirmDeleteClient(chatID, clientID)
	case "Да, удалить":
		b.deleteClient(chatID, clientID)
	case "Нет, отмена":
		b.showClientProfile(chatID, clientID)
	case "Назад":
		adminStates.Lock()
		delete(adminStates.selectedClient, chatID)
		adminStates.Unlock()
		clearState(chatID)
		b.showClientsList(message)
	default:
		b.sendMessage(chatID, "Выберите действие из меню")
	}
}

// confirmDeleteClient запрашивает подтверждение удаления
func (b *Bot) confirmDeleteClient(chatID int64, clientID int) {
	var name, surname string
	if err := b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).
		Scan(&name, &surname); err != nil {
		log.Printf("Ошибка получения имени клиента для удаления: %v", err)
	}

	text := fmt.Sprintf(
		"Вы уверены, что хотите удалить клиента?\n\n"+
			"Клиент: %s %s\n\n"+
			"Данные клиента будут сохранены в истории, но он не будет отображаться в списках.",
		name, surname)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Да, удалить"),
			tgbotapi.NewKeyboardButton("Нет, отмена"),
		),
	)
	b.sendMessageWithKeyboard(chatID, text, keyboard)
}

// deleteClient выполняет мягкое удаление клиента
func (b *Bot) deleteClient(chatID int64, clientID int) {
	var name, surname string
	if err := b.db.QueryRow("SELECT name, surname FROM public.clients WHERE id = $1", clientID).
		Scan(&name, &surname); err != nil {
		log.Printf("Ошибка получения имени клиента: %v", err)
	}

	_, err := b.db.Exec("UPDATE public.clients SET deleted_at = NOW() WHERE id = $1", clientID)
	if err != nil {
		b.sendError(chatID, "Ошибка удаления клиента", err)
		b.showClientProfile(chatID, clientID)
		return
	}

	b.sendMessage(chatID, fmt.Sprintf("Клиент %s %s удалён", name, surname))

	adminStates.Lock()
	delete(adminStates.selectedClient, chatID)
	adminStates.Unlock()

	clearState(chatID)

	b.handleAdminStart(&tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}})
}

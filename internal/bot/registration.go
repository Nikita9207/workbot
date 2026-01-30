package bot

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"workbot/internal/excel"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Регулярные выражения для валидации
var (
	phoneRegex     = regexp.MustCompile(`^[\d\s\-\+\(\)]{7,20}$`)
	dateRegex      = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}$`)
	nameRegex      = regexp.MustCompile(`^[а-яА-ЯёЁa-zA-Z\-\s]{2,50}$`)
)

// validateName проверяет имя/фамилию
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 {
		return "", fmt.Errorf("имя должно содержать минимум 2 символа")
	}
	if len(name) > 50 {
		return "", fmt.Errorf("имя слишком длинное (максимум 50 символов)")
	}

	// Проверяем что имя содержит только буквы, дефис и пробелы
	for _, r := range name {
		if !unicode.IsLetter(r) && r != '-' && r != ' ' {
			return "", fmt.Errorf("имя должно содержать только буквы")
		}
	}

	// Проверяем что первая буква - заглавная
	runes := []rune(name)
	if len(runes) > 0 && unicode.IsLetter(runes[0]) {
		runes[0] = unicode.ToUpper(runes[0])
	}

	return string(runes), nil
}

// validatePhone проверяет номер телефона
func validatePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)

	// Удаляем все кроме цифр и +
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '+' {
			return r
		}
		return -1
	}, phone)

	if len(cleaned) < 7 {
		return "", fmt.Errorf("номер телефона слишком короткий")
	}
	if len(cleaned) > 15 {
		return "", fmt.Errorf("номер телефона слишком длинный")
	}

	// Проверяем что есть хотя бы 7 цифр
	digitCount := 0
	for _, r := range cleaned {
		if unicode.IsDigit(r) {
			digitCount++
		}
	}
	if digitCount < 7 {
		return "", fmt.Errorf("номер телефона должен содержать минимум 7 цифр")
	}

	return phone, nil
}

// validateBirthDate проверяет дату рождения
func validateBirthDate(date string) (string, error) {
	date = strings.TrimSpace(date)

	if !dateRegex.MatchString(date) {
		return "", fmt.Errorf("введите дату в формате ДД.ММ.ГГГГ (например: 15.03.1990)")
	}

	// Парсим дату
	parsedDate, err := time.Parse("02.01.2006", date)
	if err != nil {
		return "", fmt.Errorf("некорректная дата, используйте формат ДД.ММ.ГГГГ")
	}

	// Проверяем что дата не в будущем
	if parsedDate.After(time.Now()) {
		return "", fmt.Errorf("дата рождения не может быть в будущем")
	}

	// Проверяем что возраст разумный (от 5 до 120 лет)
	age := time.Now().Year() - parsedDate.Year()
	if age < 5 {
		return "", fmt.Errorf("возраст должен быть не менее 5 лет")
	}
	if age > 120 {
		return "", fmt.Errorf("проверьте год рождения")
	}

	return date, nil
}

// RegistrationData хранит данные регистрации пользователя
// Все пользователи регистрируются как спортсмены
type RegistrationData struct {
	Name      string
	Surname   string
	Phone     string
	BirthDate string
	Step      int // 1=name, 2=surname, 3=phone, 4=birthdate
}

// AddClientData хранит данные добавления клиента админом
type AddClientData struct {
	Name      string
	Surname   string
	Phone     string
	BirthDate string
	Step      int // 0=name, 1=surname, 2=phone, 3=birthdate
}

// registrationStore хранит данные регистрации для каждого пользователя
var registrationStore = struct {
	sync.RWMutex
	data map[int64]*RegistrationData
}{data: make(map[int64]*RegistrationData)}

// addClientStore хранит данные добавления клиента для админов
var addClientStore = struct {
	sync.RWMutex
	data map[int64]*AddClientData
}{data: make(map[int64]*AddClientData)}

// Константы состояний регистрации
const (
	stateRegName      = "reg_name"
	stateRegSurname   = "reg_surname"
	stateRegPhone     = "reg_phone"
	stateRegBirthDate = "reg_birthdate"
)

// Константы состояний добавления клиента
const (
	stateAddClientName      = "add_client_name"
	stateAddClientSurname   = "add_client_surname"
	stateAddClientPhone     = "add_client_phone"
	stateAddClientBirthDate = "add_client_birthdate"
)

// startRegistration начинает процесс регистрации
// Обычные пользователи регистрируются только как спортсмены
// Тренеры добавляются только администраторами вручную
func (b *Bot) startRegistration(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Проверяем, не зарегистрирован ли пользователь уже
	var existingID int
	var name, surname string
	err := b.db.QueryRow("SELECT id, name, surname FROM public.clients WHERE telegram_id = $1", chatID).
		Scan(&existingID, &name, &surname)
	if err == nil {
		// Пользователь уже зарегистрирован
		msg := tgbotapi.NewMessage(chatID, b.tf("reg_already_registered", chatID, existingID, name, surname))
		b.api.Send(msg)
		b.restoreMainMenu(chatID)
		return
	}

	// Инициализируем данные регистрации
	// Все пользователи регистрируются как спортсмены
	registrationStore.Lock()
	registrationStore.data[chatID] = &RegistrationData{Step: 1}
	registrationStore.Unlock()

	// Сразу переходим к вводу имени (пропускаем выбор роли)
	userStates.Lock()
	userStates.states[chatID] = stateRegName
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, b.t("reg_title", chatID)+"\n\n"+b.t("reg_enter_name", chatID))
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(b.t("cancel", chatID)),
		),
	)
	b.api.Send(msg)
}

// processRegistration обрабатывает шаги регистрации
func (b *Bot) processRegistration(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	// Проверяем отмену на обоих языках
	if text == "Отмена" || text == "Cancel" {
		b.cancelRegistration(chatID)
		return
	}

	registrationStore.Lock()
	regData := registrationStore.data[chatID]
	if regData == nil {
		registrationStore.Unlock()
		b.cancelRegistration(chatID)
		return
	}

	switch state {
	case stateRegName:
		validatedName, err := validateName(text)
		if err != nil {
			registrationStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "❌ "+b.t("validation_name_letters", chatID)+"\n\n"+b.t("reg_enter_name", chatID))
			b.api.Send(msg)
			return
		}
		regData.Name = validatedName
		regData.Step = 2
		registrationStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateRegSurname
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, b.t("reg_enter_surname", chatID))
		b.api.Send(msg)

	case stateRegSurname:
		validatedSurname, err := validateName(text)
		if err != nil {
			registrationStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "❌ "+b.t("validation_name_letters", chatID)+"\n\n"+b.t("reg_enter_surname", chatID))
			b.api.Send(msg)
			return
		}
		regData.Surname = validatedSurname
		regData.Step = 3
		registrationStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateRegPhone
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, b.t("reg_enter_phone", chatID))
		b.api.Send(msg)

	case stateRegPhone:
		validatedPhone, err := validatePhone(text)
		if err != nil {
			registrationStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "❌ "+b.t("validation_phone_digits", chatID)+"\n\n"+b.t("reg_enter_phone", chatID))
			b.api.Send(msg)
			return
		}
		regData.Phone = validatedPhone
		regData.Step = 4
		registrationStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateRegBirthDate
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, b.t("reg_enter_birthdate", chatID))
		b.api.Send(msg)

	case stateRegBirthDate:
		validatedDate, err := validateBirthDate(text)
		if err != nil {
			registrationStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, "❌ "+b.t("validation_date_format", chatID))
			b.api.Send(msg)
			return
		}
		regData.BirthDate = validatedDate
		// Копируем данные перед разблокировкой
		name := regData.Name
		surname := regData.Surname
		phone := regData.Phone
		birthDate := regData.BirthDate
		delete(registrationStore.data, chatID)
		registrationStore.Unlock()

		// Очищаем состояние
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()

		// Сохраняем в БД (все пользователи регистрируются как спортсмены)
		b.completeRegistration(chatID, name, surname, phone, birthDate)

	default:
		registrationStore.Unlock()
	}
}

// completeRegistration завершает регистрацию и сохраняет данные
// Все пользователи регистрируются как спортсмены
// Тренеры добавляются только администраторами
func (b *Bot) completeRegistration(chatID int64, name, surname, phone, birthDate string) {
	var clientID int
	err := b.db.QueryRow(
		"INSERT INTO public.clients (name, surname, phone, birth_date, telegram_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id",
		name, surname, phone, birthDate, chatID,
	).Scan(&clientID)
	if err != nil {
		log.Println("Ошибка сохранения клиента:", err)
		errMsg := tgbotapi.NewMessage(chatID, b.t("reg_error", chatID))
		b.api.Send(errMsg)
		b.restoreMainMenu(chatID)
		return
	}

	if err := excel.AddClientToExcel(excel.FilePath, clientID, name, surname, phone, birthDate, chatID); err != nil {
		log.Printf("Ошибка добавления клиента в Excel: %v", err)
	}

	// Создаём Google таблицу для клиента
	if b.sheetsClient != nil {
		sheetID, err := b.sheetsClient.CreateClientSpreadsheet(clientID, name, surname)
		if err != nil {
			log.Printf("Ошибка создания Google таблицы: %v", err)
		} else {
			// Сохраняем ID таблицы в БД
			_, err = b.db.Exec("UPDATE clients SET google_sheet_id = $1 WHERE id = $2", sheetID, clientID)
			if err != nil {
				log.Printf("Ошибка сохранения google_sheet_id: %v", err)
			}
		}
	}

	successMsg := tgbotapi.NewMessage(chatID, b.tf("reg_success", chatID, clientID, name, surname, phone, birthDate))
	b.api.Send(successMsg)
	b.restoreMainMenu(chatID)
}

// cancelRegistration отменяет регистрацию
func (b *Bot) cancelRegistration(chatID int64) {
	registrationStore.Lock()
	delete(registrationStore.data, chatID)
	registrationStore.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, b.t("reg_cancelled", chatID))
	b.api.Send(msg)
	b.restoreMainMenu(chatID)
}

// startAddClient начинает процесс добавления клиента админом
func (b *Bot) startAddClient(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Инициализируем данные
	addClientStore.Lock()
	addClientStore.data[chatID] = &AddClientData{Step: 0}
	addClientStore.Unlock()

	// Устанавливаем состояние
	userStates.Lock()
	userStates.states[chatID] = stateAddClientName
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Добавление нового клиента\n\nВведите имя:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	b.api.Send(msg)
}

// processAddClient обрабатывает шаги добавления клиента
func (b *Bot) processAddClient(message *tgbotapi.Message, state string) {
	chatID := message.Chat.ID
	text := message.Text

	if text == "Отмена" {
		b.cancelAddClient(chatID, message)
		return
	}

	addClientStore.Lock()
	clientData := addClientStore.data[chatID]
	if clientData == nil {
		addClientStore.Unlock()
		b.cancelAddClient(chatID, message)
		return
	}

	switch state {
	case stateAddClientName:
		validatedName, err := validateName(text)
		if err != nil {
			addClientStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ %s\n\nВведите имя:", err.Error()))
			b.api.Send(msg)
			return
		}
		clientData.Name = validatedName
		clientData.Step = 1
		addClientStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateAddClientSurname
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите фамилию:")
		b.api.Send(msg)

	case stateAddClientSurname:
		validatedSurname, err := validateName(text)
		if err != nil {
			addClientStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ %s\n\nВведите фамилию:", err.Error()))
			b.api.Send(msg)
			return
		}
		clientData.Surname = validatedSurname
		clientData.Step = 2
		addClientStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateAddClientPhone
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите номер телефона:")
		b.api.Send(msg)

	case stateAddClientPhone:
		validatedPhone, err := validatePhone(text)
		if err != nil {
			addClientStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ %s\n\nВведите номер телефона:", err.Error()))
			b.api.Send(msg)
			return
		}
		clientData.Phone = validatedPhone
		clientData.Step = 3
		addClientStore.Unlock()

		userStates.Lock()
		userStates.states[chatID] = stateAddClientBirthDate
		userStates.Unlock()

		msg := tgbotapi.NewMessage(chatID, "Введите дату рождения (ДД.ММ.ГГГГ):")
		b.api.Send(msg)

	case stateAddClientBirthDate:
		validatedDate, err := validateBirthDate(text)
		if err != nil {
			addClientStore.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ %s", err.Error()))
			b.api.Send(msg)
			return
		}
		clientData.BirthDate = validatedDate
		// Копируем данные перед разблокировкой
		name := clientData.Name
		surname := clientData.Surname
		phone := clientData.Phone
		birthDate := clientData.BirthDate
		delete(addClientStore.data, chatID)
		addClientStore.Unlock()

		// Очищаем состояние
		userStates.Lock()
		delete(userStates.states, chatID)
		userStates.Unlock()

		// Сохраняем клиента
		b.completeAddClient(chatID, name, surname, phone, birthDate, message)

	default:
		addClientStore.Unlock()
	}
}

// completeAddClient завершает добавление клиента
func (b *Bot) completeAddClient(chatID int64, name, surname, phone, birthDate string, message *tgbotapi.Message) {
	var clientID int
	err := b.db.QueryRow(
		"INSERT INTO public.clients (name, surname, phone, birth_date, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id",
		name, surname, phone, birthDate,
	).Scan(&clientID)
	if err != nil {
		log.Println("Ошибка сохранения клиента:", err)
		errMsg := tgbotapi.NewMessage(chatID, "Ошибка при добавлении клиента.")
		b.api.Send(errMsg)
		b.handleAdminStart(message)
		return
	}

	if err := excel.AddClientToExcel(excel.FilePath, clientID, name, surname, phone, birthDate, 0); err != nil {
		log.Printf("Ошибка добавления клиента в Excel: %v", err)
	}

	// Создаём Google таблицу для клиента
	var sheetURL string
	if b.sheetsClient != nil {
		sheetID, err := b.sheetsClient.CreateClientSpreadsheet(clientID, name, surname)
		if err != nil {
			log.Printf("Ошибка создания Google таблицы: %v", err)
		} else {
			// Сохраняем ID таблицы в БД
			_, err = b.db.Exec("UPDATE clients SET google_sheet_id = $1 WHERE id = $2", sheetID, clientID)
			if err != nil {
				log.Printf("Ошибка сохранения google_sheet_id: %v", err)
			}
			sheetURL = fmt.Sprintf("\n\nGoogle таблица: https://docs.google.com/spreadsheets/d/%s", sheetID)
		}
	}

	successMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Клиент добавлен!\n\nID: %d\nИмя: %s %s\nТелефон: %s\nДата рождения: %s%s",
		clientID, name, surname, phone, birthDate, sheetURL,
	))
	b.api.Send(successMsg)
	b.handleAdminStart(message)
}

// cancelAddClient отменяет добавление клиента
func (b *Bot) cancelAddClient(chatID int64, message *tgbotapi.Message) {
	addClientStore.Lock()
	delete(addClientStore.data, chatID)
	addClientStore.Unlock()

	userStates.Lock()
	delete(userStates.states, chatID)
	userStates.Unlock()

	msg := tgbotapi.NewMessage(chatID, "Отменено")
	b.api.Send(msg)
	b.handleAdminStart(message)
}

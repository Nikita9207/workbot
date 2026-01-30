package bot

import (
	"database/sql"
	"log"

	"workbot/internal/config"
	"workbot/internal/gsheets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot представляет Telegram бота
type Bot struct {
	api          *tgbotapi.BotAPI
	db           *sql.DB
	config       *config.Config
	sheetsClient *gsheets.Client
}

// New создаёт новый экземпляр бота
func New(api *tgbotapi.BotAPI, db *sql.DB, cfg *config.Config) *Bot {
	// Инициализируем Google Sheets клиент
	var sheetsClient *gsheets.Client

	// Сначала пробуем OAuth2 (личный аккаунт)
	if cfg.GoogleOAuthCredPath != "" && cfg.GoogleTokenPath != "" {
		var err error
		sheetsClient, err = gsheets.NewOAuthClient(cfg.GoogleOAuthCredPath, cfg.GoogleTokenPath, cfg.GoogleDriveFolderID)
		if err != nil {
			log.Printf("Google Sheets OAuth2 не инициализирован: %v", err)
		}
	}

	// Если OAuth не настроен — пробуем Service Account
	if sheetsClient == nil && cfg.GoogleCredentialsPath != "" && cfg.GoogleDriveFolderID != "" {
		var err error
		sheetsClient, err = gsheets.NewClient(cfg.GoogleCredentialsPath, cfg.GoogleDriveFolderID)
		if err != nil {
			log.Printf("Google Sheets Service Account не инициализирован: %v", err)
		} else {
			log.Println("Google Sheets клиент инициализирован (Service Account)")
		}
	}

	return &Bot{
		api:          api,
		db:           db,
		config:       cfg,
		sheetsClient: sheetsClient,
	}
}

// Start запускает бота
func (b *Bot) Start() error {
	updates, err := b.initUpdatesChannel()
	if err != nil {
		return err
	}

	b.handleUpdates(updates)
	return nil
}

func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		// Обработка callback-запросов (от inline-кнопок)
		if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		isAdmin := b.isAdmin(chatID)

		if update.Message.IsCommand() {
			if isAdmin {
				b.handleAdminCommand(update.Message)
			} else {
				b.handleCommand(update.Message)
			}
			continue
		}

		if isAdmin {
			b.handleAdminMessage(update.Message)
		} else {
			b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) initUpdatesChannel() (tgbotapi.UpdatesChannel, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	return b.api.GetUpdatesChan(u), nil
}

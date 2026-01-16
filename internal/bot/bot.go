package bot

import (
	"database/sql"
	"log"

	"workbot/clients/ai"
	"workbot/internal/config"
	"workbot/internal/gsheets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot представляет Telegram бота
type Bot struct {
	api           *tgbotapi.BotAPI
	db            *sql.DB
	aiClient      *ai.Client
	config        *config.Config
	sheetsClient  *gsheets.Client
}

// New создаёт новый экземпляр бота
func New(api *tgbotapi.BotAPI, db *sql.DB, cfg *config.Config) *Bot {
	var aiClient *ai.Client
	if cfg.GroqAPIKey != "" {
		aiClient = ai.NewClient(cfg.GroqAPIKey)
	}

	// Инициализируем Google Sheets клиент
	var sheetsClient *gsheets.Client
	if cfg.GoogleDriveFolderID != "" {
		var err error
		sheetsClient, err = gsheets.NewClient(cfg.GoogleCredentialsPath, cfg.GoogleDriveFolderID)
		if err != nil {
			log.Printf("Предупреждение: Google Sheets не инициализирован: %v", err)
		} else {
			log.Println("Google Sheets клиент инициализирован")
		}
	}

	return &Bot{
		api:          api,
		db:           db,
		aiClient:     aiClient,
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

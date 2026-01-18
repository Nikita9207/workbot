package bot

import (
	"database/sql"
	"log"

	"workbot/clients/ai"
	"workbot/clients/knowledge"
	"workbot/internal/config"
	"workbot/internal/gsheets"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot представляет Telegram бота
type Bot struct {
	api            *tgbotapi.BotAPI
	db             *sql.DB
	aiClient       *ai.Client
	whisperClient  *ai.WhisperClient
	knowledgeStore *knowledge.Store
	config         *config.Config
	sheetsClient   *gsheets.Client
}

// New создаёт новый экземпляр бота
func New(api *tgbotapi.BotAPI, db *sql.DB, cfg *config.Config) *Bot {
	// Инициализируем AI клиент (Ollama)
	aiClient := ai.NewClientWithURL(cfg.OllamaURL, cfg.OllamaModel)

	if aiClient.IsAvailable() {
		log.Printf("Ollama доступен: %s (модель: %s)", cfg.OllamaURL, cfg.OllamaModel)
	} else {
		log.Printf("Ollama недоступен: %s", cfg.OllamaURL)
	}

	// Инициализируем Whisper клиент (Groq) для транскрипции
	whisperClient := ai.NewWhisperClient(cfg.GroqAPIKey)
	if whisperClient.IsAvailable() {
		log.Println("Groq Whisper доступен для транскрипции голоса")
	} else {
		log.Println("Groq Whisper недоступен (GROQ_API_KEY не настроен)")
	}

	// Инициализируем хранилище знаний (RAG)
	knowledgeStore := knowledge.NewStore()
	if cfg.RAGIndexPath != "" {
		if err := knowledgeStore.Load(cfg.RAGIndexPath); err != nil {
			log.Printf("RAG индекс не загружен: %v", err)
		} else {
			log.Printf("RAG индекс загружен: %d документов", knowledgeStore.Count())
		}
	}

	// Инициализируем Google Sheets клиент
	var sheetsClient *gsheets.Client
	if cfg.GoogleDriveFolderID != "" {
		var err error
		sheetsClient, err = gsheets.NewClient(cfg.GoogleCredentialsPath, cfg.GoogleDriveFolderID)
		if err != nil {
			log.Printf("Google Sheets не инициализирован: %v", err)
		} else {
			log.Println("Google Sheets клиент инициализирован")
		}
	}

	return &Bot{
		api:            api,
		db:             db,
		aiClient:       aiClient,
		whisperClient:  whisperClient,
		knowledgeStore: knowledgeStore,
		config:         cfg,
		sheetsClient:   sheetsClient,
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

		// Обработка голосовых сообщений
		if update.Message.Voice != nil {
			if !isAdmin {
				b.handleFeedbackVoice(update.Message)
			}
			continue
		}

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

package internal

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot     *tgbotapi.BotAPI
	update  *tgbotapi.Update
	updates *tgbotapi.UpdatesChannel
	message *tgbotapi.Message
}

func NewBot(bot *tgbotapi.BotAPI) *Bot {
	return &Bot{bot: bot}
}

func (b *Bot) Start() error {
	updates, err := b.initUpdatesChannel()
	if err != nil {
		return err
	}

	b.handelUpdates(updates)
	return nil
}

func (b *Bot) handelUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			b.handelCommand(update.Message)
			continue
		}

		b.handelMessage(update.Message, update, b.bot, updates)
	}
}

func (b *Bot) initUpdatesChannel() (tgbotapi.UpdatesChannel, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	return b.bot.GetUpdatesChan(u), nil
}

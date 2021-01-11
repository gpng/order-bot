package telegram

import (
	tgbotapi "github.com/dilfish/telegram-bot-api-up"
)

// Bot with all methods
type Bot struct {
	BotAPI tgbotapi.BotAPI
}

// New db connection and trigger migrations
func New(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	// connection string
	return &Bot{*bot}, nil
}

// SendMessage util
func (bot *Bot) SendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.DisableWebPagePreview = true
	bot.BotAPI.Send(msg)
}

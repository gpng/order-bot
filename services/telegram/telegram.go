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

// SendMessage text
func (bot *Bot) SendMessage(chatID int64, formatHTML bool, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if formatHTML {
		msg.ParseMode = tgbotapi.ModeHTML
	}
	msg.DisableWebPagePreview = true
	bot.BotAPI.Send(msg)
}

// SendInlineKeyboardMessage with options
func (bot *Bot) SendInlineKeyboardMessage(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	bot.BotAPI.Send(msg)
}

// EditMessage text
func (bot *Bot) EditMessage(chatID int64, messageID int, text string) {
	msg := tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:    chatID,
			MessageID: messageID,
		},
		Text: text,
	}
	bot.BotAPI.Send(msg)
}

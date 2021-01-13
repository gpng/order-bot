package models

// Chat model
type Chat struct {
	ID int64 `json:"id"`
}

// User model
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
}

// Message model
type Message struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	From      User   `json:"from"`
}

// TelegramUpdate model
type TelegramUpdate struct {
	UpdateID int     `json:"update_id"`
	Message  *Message `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

// CallbackQuery model
type CallbackQuery struct {
	ID string `json:"id"`
	From User `json:"from"`
	Data string `json:"data"`
	Message *Message `json:"message"`
	InlineMessageID string `json:"inline_message_id"`
	ChatInstance string `json:"chat_instance"`
}
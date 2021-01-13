package handlers

import (
	"github.com/gocraft/work"
	"github.com/gpng/order-bot/services/telegram"
	"github.com/gpng/order-bot/sqlc/models"
	"go.uber.org/zap"
)

// Handlers struct
type Handlers struct {
	BotToken string
	Logger *zap.Logger
	DB     models.DBTX
	Repo   models.Querier
	Bot    *telegram.Bot
	Queue  *work.Enqueuer
}

// New service
func New(
	botToken string,
	logger *zap.Logger,
	db models.DBTX,
	repo models.Querier,
	bot *telegram.Bot,
	queue *work.Enqueuer,
) *Handlers {
	return &Handlers{botToken, logger, db, repo, bot, queue}
}

// JobName are job names
type JobName string

// Job names
const (
	JobNotifyExpiry JobName = "notify_expiry"
)

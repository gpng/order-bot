package handlers

import (
	"github.com/gpng/order-bot/services/telegram"
	"github.com/gpng/order-bot/sqlc/models"
	"go.uber.org/zap"
)

// Handlers struct
type Handlers struct {
	logger *zap.Logger
	db     models.DBTX
	repo   models.Querier
	bot    *telegram.Bot
}

// New service
func New(
	logger *zap.Logger,
	db models.DBTX,
	repo models.Querier,
	bot *telegram.Bot,
) *Handlers {
	return &Handlers{logger, db, repo, bot}
}

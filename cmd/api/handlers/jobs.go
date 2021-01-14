package handlers

import (
	"context"

	"github.com/gocraft/work"
	"go.uber.org/zap"
)

const (
	jobArgOrderID   = "order_id"
	jobArgPreExpiry = "pre_expiry"
)

// JobNotifyExpiry sends an alert when job is done
func (h *Handlers) JobNotifyExpiry(job *work.Job) error {
	orderID := int32(job.ArgInt64(jobArgOrderID))
	preExpiry := job.ArgBool(jobArgPreExpiry)

	l := h.Logger.With(zap.String("job", string(JobNotifyExpiry)), zap.Int32("order_id", orderID))

	order, err := h.Repo.GetOrderByID(context.Background(), orderID)
	if err != nil {
		l.Error("failed to retrieve order", zap.Error(err))
		return err
	}

	err = h.Repo.DeactivateOrder(context.Background(), orderID)
	if err != nil {
		l.Error("failed to deactivate order", zap.Error(err))
		return err
	}

	err = h.sendOverview(l, order, preExpiry)
	if err != nil {
		l.Error("failed to send notification", zap.Error(err))
		return err
	}
	if !preExpiry {
		h.Bot.SendMessage(int64(order.ChatID), false, MsgCancelTakeOrders)
	}

	return nil
}

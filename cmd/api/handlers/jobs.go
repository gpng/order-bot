package handlers

import (
	"context"

	"github.com/gocraft/work"
	"go.uber.org/zap"
)

// JobNotifyExpiry sends an alert when job is done
func (h *Handlers) JobNotifyExpiry(job *work.Job) error {
	orderID := int32(job.ArgInt64("order_id"))

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

	err = h.sendOverview(l, order)
	if err != nil {
		l.Error("failed to send notification", zap.Error(err))
		return err
	}

	return nil
}

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gpng/order-bot/sqlc/models"
	"go.uber.org/zap"
)

func (h *Handlers) handleUpdates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		update := &models.TelegramUpdate{}
		if err := json.NewDecoder(r.Body).Decode(update); err != nil {
			h.logger.Error("failed to decoding body", zap.Error(err))
			return
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text
		split := strings.Split(text, " ")

		var err error
		switch strings.ToLower(split[0]) {
		case "/neworder":
			err = h.handleNewOrder(chatID, text)
			break
		case "/cancelorder":
			err = h.handleCancelOrder(chatID)
			break
		case "/order":
			err = h.handlerOrder(chatID, text, update.Message.From)
			break
		}

		if err != nil {
			h.bot.SendMessage(chatID, "Oops something went wrong")
		}
	}
}

func (h *Handlers) handleCancelOrder(chatID int64) error {
	l := h.logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/cancelorder"))

	err := h.repo.CancelOrder(context.Background(), int32(chatID))
	if err != nil {
		l.Error("error cancelling active orders", zap.Error(err))
		return nil
	}

	h.bot.SendMessage(chatID, "Active order cancelled")

	return nil
}

func (h *Handlers) handleNewOrder(chatID int64, text string) error {
	l := h.logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/neworder"))

	split := strings.Split(text, " ")

	if len(split) < 3 {
		h.bot.SendMessage(chatID, "Invalid format\\! Create new order collection like /neworder 15:30 SoGood Bakery")
		return nil
	}

	expiry := split[1]
	title := strings.Join(split[2:], " ")
	r, err := regexp.Compile("^(2[0-3]|[01]?[0-9]):([0-5]?[0-9])$")
	if err != nil {
		l.Error("error compiling regex", zap.Error(err))
		return err
	}
	if !r.MatchString(expiry) {
		h.bot.SendMessage(chatID, "Invalid time\\! Create new order collection like /neworder 15:30 SoGood Bakery")
		return nil
	}

	expirySplit := strings.Split(expiry, ":")

	hour, err := strconv.Atoi(expirySplit[0])
	if err != nil {
		h.bot.SendMessage(chatID, "Invalid time\\! Create new order collection like /neworder 15:30 SoGood Bakery")
		return nil
	}

	min, err := strconv.Atoi(expirySplit[1])
	if err != nil {
		h.bot.SendMessage(chatID, "Invalid time\\! Create new order collection like /neworder 15:30 SoGood Bakery")
		return nil
	}

	location, err := getLocation()
	if err != nil {
		l.Error("error loading time location", zap.Error(err))
		return nil
	}

	now := time.Now().In(location)

	expiryTime := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, location)

	isTomorrow := expiryTime.Before(now)
	if isTomorrow {
		expiryTime = expiryTime.Add(time.Hour * 24)
	}

	activeOrder, err := h.repo.GetActiveOrder(context.Background(), models.GetActiveOrderParams{
		ChatID: int32(chatID),
		Expiry: now,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		l.Error("error fetching active orders", zap.Error(err))
		return err
	}
	if err == nil {
		h.bot.SendMessage(chatID, "There is already an existing order for "+activeOrder.Title+". Use /cancelorder to cancel the current active order")
		return nil
	}

	_, err = h.repo.CreateOrder(context.Background(), models.CreateOrderParams{
		ChatID: int32(chatID),
		Title:  title,
		Expiry: expiryTime,
	})
	if err != nil {
		l.Error("error creating order", zap.Error(err))
		return err
	}

	message := "Taking orders for " + title + ", ending at " + expiry
	if isTomorrow {
		message += " tomorrow"
	}

	h.bot.SendMessage(chatID, message)

	return nil
}

func (h *Handlers) handlerOrder(chatID int64, text string, user models.User) error {
	l := h.logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/order"))

	location, err := getLocation()
	if err != nil {
		l.Error("error loading time location", zap.Error(err))
		return err
	}

	now := time.Now().In(location)

	order, err := h.repo.GetActiveOrder(context.Background(), models.GetActiveOrderParams{
		ChatID: int32(chatID),
		Expiry: now,
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.bot.SendMessage(chatID, "No active orders\\! Create new order collection like /neworder 15:30 SoGood Bakery")
			return nil
		}

		l.Error("error fetching active orders", zap.Error(err))
		return err
	}

	split := strings.Split(text, " ")

	if len(split) < 3 {
		h.bot.SendMessage(chatID, "Invalid order\\! Add to the order like /order 2 chicken pie")
		return nil
	}

	quantity, err := strconv.Atoi(split[1])
	if err != nil || quantity <= 0 {
		h.bot.SendMessage(chatID, "Invalid quantity\\! Add to the order like /order 2 chicken pie")
		return nil
	}

	name := strings.Join(split[2:], " ")

	item, err := h.repo.GetItem(context.Background(), models.GetItemParams{
		OrderID: order.ID,
		UserID:  int32(user.ID),
		Lower:   name,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		l.Error("error checking item", zap.Error(err))
		return err
	}

	if err == nil {
		_, err = h.repo.UpdateItemQuantity(context.Background(), models.UpdateItemQuantityParams{
			OrderID:  order.ID,
			Quantity: int32(quantity + int(item.Quantity)),
			Lower:    name,
			UserID:   int32(user.ID),
		})
		if err != nil {
			l.Error("error creating item", zap.Error(err))
			return err
		}
	} else {
		_, err = h.repo.CreateItem(context.Background(), models.CreateItemParams{
			OrderID:  order.ID,
			Quantity: int32(quantity),
			Name:     name,
			UserID:   int32(user.ID),
			UserName: user.FirstName,
		})
		if err != nil {
			l.Error("error creating item", zap.Error(err))
			return err
		}
	}
	quantity = quantity + int(item.Quantity)

	return h.sendOverview(l, chatID, order)
}

func (h *Handlers) sendOverview(l *zap.Logger, chatID int64, order models.Order) error {
	items, err := h.repo.GetItemsByOrderID(context.Background(), order.ID)
	if err != nil {
		l.Error("error getting order items", zap.Error(err))
		return err
	}

	location, err := getLocation()
	if err != nil {
		l.Error("error loading time location", zap.Error(err))
		return err
	}

	now := time.Now().In(location)

	expiry := order.Expiry.Format("15:04")
	if order.Expiry.Day() > now.Day() {
		expiry += " tomorrow"
	}

	allItems := map[string]int{}
	itemsText := ""
	for _, item := range items {
		itemsText += fmt.Sprintf("[%s](tg://user?id=%d) %d %s\n", item.UserName, item.UserID, item.Quantity, item.Name)
		lowered := strings.ToLower(item.Name)
		allItems[lowered] += int(item.Quantity)
	}

	allItemsText := ""
	for name, quantity := range allItems {
		allItemsText += fmt.Sprintf("%d x %s\n", quantity, name)
	}

	h.bot.SendMessage(chatID, fmt.Sprintf(`
*%s*
%s

%s

*Consolidated*
%s
`, order.Title, expiry, itemsText, allItemsText))

	return nil
}

func getLocation() (*time.Location, error) {
	return time.LoadLocation("Asia/Singapore")
}

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/dilfish/telegram-bot-api-up"
	"github.com/gocraft/work"
	"github.com/gpng/order-bot/sqlc/models"
	"go.uber.org/zap"
)

func (h *Handlers) handleUpdates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		update := &models.TelegramUpdate{}

		if err := json.NewDecoder(r.Body).Decode(update); err != nil {
			h.Logger.Error("failed to decoding body", zap.Error(err))
			return
		}

		if update.Message.GroupChatCreated {
			h.handleStart(update.Message.Chat.ID)
		}
		if len(update.Message.NewChatMembers) > 0 {
			h.handleNewChatMembers(update.Message.Chat.ID, update.Message.NewChatMembers)
			return
		}

		if update.CallbackQuery != nil {
			var err error
			switch strings.ToLower(strings.Split(update.CallbackQuery.Data, " ")[0]) {
			case "/delete":
				err = h.handleDeleteItem(*update.CallbackQuery)
				break
			case "/cancel":
				err = h.handleCancelDeleteOrder(*update.CallbackQuery)
				break
			}
			h.Bot.BotAPI.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			if err != nil {

				return
			}
			return
		}

		if update.Message != nil {
			chatID := update.Message.Chat.ID
			text := update.Message.Text
			split := strings.Split(text, " ")

			var err error
			switch strings.ToLower(split[0]) {
			case "/start", "/help":
				h.handleStart(chatID)
				return
			case "/takeorders", "/takeorder", "/neworder", "/neworders":
				err = h.handleNewOrder(chatID, text)
				break
			case "/endorders", "/endorder", "/endtakeorders", "/endtakeorder":
				err = h.handleCancelTakeOrder(chatID)
				break
			case "/order":
				err = h.handlerOrder(chatID, text, update.Message.From)
				break
			case "/cancelorder", "/removeorder":
				err = h.handleCancelOrder(chatID, update.Message.From)
				break
			}

			if err != nil {
				h.Bot.SendMessage(chatID, false, MsgError)
			}
		}
	}
}

func (h *Handlers) handleStart(chatID int64) {
	h.Bot.SendMessage(chatID, false, fmt.Sprintf(`Use HelpMeBuyLehBot to collect group orders!

%s
%s
%s
`, MsgTakeOrders, MsgOrder, MsgEndTakeOrders))
}

func (h *Handlers) handleCancelTakeOrder(chatID int64) error {
	l := h.Logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/endorders"))

	order, err := h.Repo.CancelOrder(context.Background(), int32(chatID))
	log.Printf("order, err: %v, %v\n", order, err)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.Bot.SendMessage(chatID, false, MsgNoActiveOrders)
			return nil
		}
		l.Error("error cancelling active orders", zap.Error(err))
		return err
	}

	if order.ReminderRunAt.Valid && order.ReminderID.Valid {
		err = h.WorkClient.DeleteScheduledJob(order.ReminderRunAt.Int64, order.ReminderID.String)
		log.Printf("err: %v\n", err)
		if err != nil && !errors.Is(err, work.ErrNotDeleted) {
			l.Error("error deleteing reminder job", zap.Error(err))
			return err
		}
	}
	if order.ExpiryRunAt.Valid && order.ExpiryID.Valid {
		err = h.WorkClient.DeleteScheduledJob(order.ExpiryRunAt.Int64, order.ExpiryID.String)
		log.Printf("err: %v\n", err)
		if err != nil && !errors.Is(err, work.ErrNotDeleted) {
			l.Error("error deleteing expiry job", zap.Error(err))
			return err
		}
	}

	err = h.sendOverview(l, order, false)
	if err != nil {
		l.Error("error sennding overview", zap.Error(err))
		return err
	}
	h.Bot.SendMessage(chatID, false, MsgCancelTakeOrders)

	return nil
}

func (h *Handlers) handleNewOrder(chatID int64, text string) error {
	l := h.Logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/takeorders"))

	split := strings.Split(text, " ")

	if len(split) < 3 {
		h.Bot.SendMessage(chatID, false, MsgNewTakeOrderInvalidFormat)
		return nil
	}

	expiry := split[1]
	title := escapeString(strings.Join(split[2:], " "))
	r, err := regexp.Compile("^(2[0-3]|[01]?[0-9]):([0-5]?[0-9])$")
	if err != nil {
		l.Error("error compiling regex", zap.Error(err))
		return err
	}
	if !r.MatchString(expiry) {
		h.Bot.SendMessage(chatID, false, MsgNewTakeOrderInvalidTime)
		return nil
	}

	expirySplit := strings.Split(expiry, ":")

	hour, err := strconv.Atoi(expirySplit[0])
	if err != nil {
		h.Bot.SendMessage(chatID, false, MsgNewTakeOrderInvalidTime)
		return nil
	}

	min, err := strconv.Atoi(expirySplit[1])
	if err != nil {
		h.Bot.SendMessage(chatID, false, MsgNewTakeOrderInvalidTime)
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

	activeOrder, err := h.Repo.GetActiveOrder(context.Background(), models.GetActiveOrderParams{
		ChatID: int32(chatID),
		Expiry: now,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		l.Error("error fetching active orders", zap.Error(err))
		return err
	}
	if err == nil {
		h.Bot.SendMessage(chatID, false, MsgNewTakeOrderExistingOrder(activeOrder.Title))
		return nil
	}

	order, err := h.Repo.CreateOrder(context.Background(), models.CreateOrderParams{
		ChatID: int32(chatID),
		Title:  title,
		Expiry: expiryTime,
	})
	if err != nil {
		l.Error("error creating order", zap.Error(err))
		return err
	}

	now = time.Now().In(location)
	diff := expiryTime.Sub(now).Seconds()

	if diff > 600 { // only notify 5 minutes before if more than 10 minutes to go
		scheduledJob, err := h.Queue.EnqueueUniqueIn(string(JobNotifyExpiry), int64(diff-300), work.Q{
			jobArgOrderID:   int64(order.ID),
			jobArgPreExpiry: true,
		})
		if err != nil {
			l.Error("error scheduling job", zap.Error(err))
			return err
		}
		err = h.Repo.UpdateReminder(context.Background(), models.UpdateReminderParams{
			ID:            order.ID,
			ReminderRunAt: sql.NullInt64{Int64: scheduledJob.EnqueuedAt, Valid: true},
			ReminderID:    sql.NullString{String: scheduledJob.ID, Valid: true},
		})
		if err != nil {
			l.Error("error updating reminder details", zap.Error(err))
			return err
		}
	}

	scheduledJob, err := h.Queue.EnqueueUniqueIn(string(JobNotifyExpiry), int64(diff), work.Q{
		jobArgOrderID:   int64(order.ID),
		jobArgPreExpiry: false,
	})

	if err != nil {
		l.Error("error scheduling job", zap.Error(err))
		return err
	}
	err = h.Repo.UpdateExpiry(context.Background(), models.UpdateExpiryParams{
		ID:          order.ID,
		ExpiryRunAt: sql.NullInt64{Int64: scheduledJob.EnqueuedAt, Valid: true},
		ExpiryID:    sql.NullString{String: scheduledJob.ID, Valid: true},
	})
	if err != nil {
		l.Error("error updating reminder details", zap.Error(err))
		return err
	}

	message := "Taking orders for " + title + ", ending at " + expiry
	if isTomorrow {
		message += " tomorrow"
	}

	fullMessage := fmt.Sprintf(`%s
	
%s
%s
`, message, MsgEndTakeOrders, MsgOrder)

	h.Bot.SendMessage(chatID, false, fullMessage)

	return nil
}

func (h *Handlers) handlerOrder(chatID int64, text string, user models.User) error {
	l := h.Logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/order"))

	location, err := getLocation()
	if err != nil {
		l.Error("error loading time location", zap.Error(err))
		return err
	}

	now := time.Now().In(location)

	order, err := h.Repo.GetActiveOrder(context.Background(), models.GetActiveOrderParams{
		ChatID: int32(chatID),
		Expiry: now,
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.Bot.SendMessage(chatID, false, MsgNoActiveOrders)
			return nil
		}

		l.Error("error fetching active orders", zap.Error(err))
		return err
	}

	split := strings.Split(text, " ")

	if len(split) < 3 {
		h.Bot.SendMessage(chatID, false, MsgOrderInvalidFormat)
		return nil
	}

	quantity, err := strconv.Atoi(split[1])
	if err != nil || quantity <= 0 {
		h.Bot.SendMessage(chatID, false, MsgOrderInvalidQuantity)
		return nil
	}

	name := strings.Join(split[2:], " ")

	item, err := h.Repo.GetItem(context.Background(), models.GetItemParams{
		OrderID: order.ID,
		UserID:  int32(user.ID),
		Lower:   name,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		l.Error("error checking item", zap.Error(err))
		return err
	}

	if err == nil {
		_, err = h.Repo.UpdateItemQuantity(context.Background(), models.UpdateItemQuantityParams{
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
		_, err = h.Repo.CreateItem(context.Background(), models.CreateItemParams{
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

	return h.sendOverview(l, order, false)
}

func (h *Handlers) sendOverview(l *zap.Logger, order models.Order, isPreExpiry bool) error {
	items, err := h.Repo.GetItemsByOrderID(context.Background(), order.ID)
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

	title := order.Title

	expiry := order.Expiry.Format("15:04")
	if order.Expiry.Day() > now.Day() {
		expiry += " tomorrow"
	}

	if isPreExpiry {
		expiry += " in 5 minutes"
		title = "REMINDER\n" + title
	}

	allItems := map[string]int{}
	itemsText := ""
	for _, item := range items {
		name := html.EscapeString(strings.ToLower(item.Name))
		itemsText += fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a> %d x %s\n", item.UserID, item.UserName, item.Quantity, name)
		allItems[name] += int(item.Quantity)
	}

	allItemsText := ""
	for name, quantity := range allItems {
		allItemsText += fmt.Sprintf("%d x %s\n", quantity, name)
	}

	message := fmt.Sprintf(`
<b>%s</b>
%s

%s
<b>Consolidated</b>
%s
%s
%s
`, title, expiry, itemsText, allItemsText, MsgEndTakeOrders, MsgCancelOrder)

	h.Bot.SendMessage(int64(order.ChatID), true, message)

	return nil
}

func getLocation() (*time.Location, error) {
	return time.LoadLocation("Asia/Singapore")
}

func (h *Handlers) handleCancelOrder(chatID int64, user models.User) error {
	l := h.Logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/cancelorder"))

	location, err := getLocation()
	if err != nil {
		l.Error("error loading time location", zap.Error(err))
		return nil
	}
	now := time.Now().In(location)

	order, err := h.Repo.GetActiveOrder(context.Background(), models.GetActiveOrderParams{
		ChatID: int32(chatID),
		Expiry: now,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.Bot.SendMessage(chatID, false, MsgNoActiveOrders)
			return nil
		}
		l.Error("failed to retrieve active order", zap.Error(err))
		return err
	}

	items, err := h.Repo.GetUserItems(context.Background(), models.GetUserItemsParams{
		UserID:  int32(user.ID),
		OrderID: order.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.Bot.SendMessage(chatID, false, MsgNoOrders)
			return nil
		}
		l.Error("failed to retrieve user items", zap.Error(err))
		return err
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, len(items))
	for i, item := range items {
		rows[i] = tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d x %s", item.Quantity, item.Name), "/delete "+strconv.Itoa(int(item.ID))),
		)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Cancel", "/cancel"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	h.Bot.SendInlineKeyboardMessage(chatID, MsgSelectDeleteOrder, keyboard)

	return nil
}

func (h *Handlers) handleDeleteItem(cq models.CallbackQuery) error {
	if cq.Message == nil {
		return nil
	}
	l := h.Logger.With(zap.Int64("chat_id", cq.Message.Chat.ID), zap.String("command", "/deleteitem"))

	split := strings.Split(cq.Data, " ")
	if len(split) < 2 {
		l.Error("invalid delete item format", zap.String("data", cq.Data))
		h.Bot.EditMessage(cq.Message.Chat.ID, cq.Message.MessageID, MsgInvalidItem)
		return nil
	}

	location, err := getLocation()
	if err != nil {
		l.Error("error loading time location", zap.Error(err))
		return err
	}
	now := time.Now().In(location)

	order, err := h.Repo.GetActiveOrder(context.Background(), models.GetActiveOrderParams{
		ChatID: int32(cq.Message.Chat.ID),
		Expiry: now,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.Bot.EditMessage(cq.Message.Chat.ID, cq.Message.MessageID, MsgNoActiveOrders)
			return nil
		}
		l.Error("failed to retrieve active order", zap.Error(err))
		return err
	}

	itemID, err := strconv.Atoi(split[1])
	if err != nil {
		l.Error("invalid item id", zap.String("data", cq.Data), zap.Error(err))
		h.Bot.EditMessage(cq.Message.Chat.ID, cq.Message.MessageID, MsgInvalidItem)
		return nil
	}

	item, err := h.Repo.DeleteItemByUser(context.Background(), models.DeleteItemByUserParams{
		ID:     int32(itemID),
		UserID: int32(cq.From.ID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.Bot.EditMessage(cq.Message.Chat.ID, cq.Message.MessageID, MsgInvalidItem)
			return nil
		}
		l.Error("failed to delete item", zap.Error(err))
		return err
	}

	h.Bot.EditMessage(cq.Message.Chat.ID, cq.Message.MessageID, MsgDeletedOrder(int(item.Quantity), item.Name))

	return h.sendOverview(l, order, false)
}

func (h *Handlers) handleCancelDeleteOrder(cq models.CallbackQuery) error {
	if cq.Message != nil {
		msg := tgbotapi.EditMessageTextConfig{
			BaseEdit: tgbotapi.BaseEdit{
				ChatID:    cq.Message.Chat.ID,
				MessageID: cq.Message.MessageID,
			},
			Text: MsgCanceledDeleteOrderRequest,
		}
		h.Bot.BotAPI.Send(msg)
	}
	return nil
}

func (h *Handlers) handleNewChatMembers(chatID int64, newChatMembers []models.User) {
	l := h.Logger.With(zap.Int64("chat_id", chatID), zap.String("event", "new members"))

	botID, err := strconv.Atoi(strings.Split(h.BotToken, ":")[0])
	if err != nil {
		l.Error("failed to get bot id from token", zap.Error(err))
		return
	}
	for _, member := range newChatMembers {
		if member.IsBot && member.ID == int64(botID) {
			h.handleStart(chatID)
		}
	}
}

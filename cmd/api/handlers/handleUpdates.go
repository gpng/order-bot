package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
			log.Printf("chatID: %v\n", chatID)
			text := update.Message.Text
			split := strings.Split(text, " ")
	
			var err error
			switch strings.ToLower(split[0]) {
			case "/takeorders", "/takeorder", "/collectorders", "/collectorder", "/neworder", "/neworders":
				err = h.handleNewOrder(chatID, text)
				break
			case "/cancelorder":
				err = h.handleCancelOrder(chatID)
				break
			case "/order":
				err = h.handlerOrder(chatID, text, update.Message.From)
				break
			case "/deleteorder", "/removeorder":
				err = h.handleDeleteOrder(chatID, update.Message.From)
				break
			}
	
			if err != nil {
				h.Bot.SendMessage(chatID, false, "Oops something went wrong")
			}
		}
	}
}

func (h *Handlers) handleCancelOrder(chatID int64) error {
	l := h.Logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/cancelorder"))

	err := h.Repo.CancelOrder(context.Background(), int32(chatID))
	if err != nil {
		l.Error("error cancelling active orders", zap.Error(err))
		return nil
	}

	h.Bot.SendMessage(chatID, false, "Active order cancelled")

	return nil
}

func (h *Handlers) handleNewOrder(chatID int64, text string) error {
	l := h.Logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/neworder"))

	split := strings.Split(text, " ")

	if len(split) < 3 {
		h.Bot.SendMessage(chatID, false, "Invalid format! Create new order collection like /neworder 15:30 SoGood Bakery")
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
		h.Bot.SendMessage(chatID, false, "Invalid time! Create new order collection like /neworder 15:30 SoGood Bakery")
		return nil
	}

	expirySplit := strings.Split(expiry, ":")

	hour, err := strconv.Atoi(expirySplit[0])
	if err != nil {
		h.Bot.SendMessage(chatID, false, "Invalid time! Create new order collection like /neworder 15:30 SoGood Bakery")
		return nil
	}

	min, err := strconv.Atoi(expirySplit[1])
	if err != nil {
		h.Bot.SendMessage(chatID, false, "Invalid time! Create new order collection like /neworder 15:30 SoGood Bakery")
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
		h.Bot.SendMessage(chatID, false, "There is already an existing order for "+activeOrder.Title+". Use /cancelorder to cancel the current active order")
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
		_, err = h.Queue.EnqueueUniqueIn(string(JobNotifyExpiry), int64(diff-300), work.Q{
			jobArgOrderID:   int64(order.ID),
			jobArgPreExpiry: true,
		})
		if err != nil {
			l.Error("error scheduling job", zap.Error(err))
			return err
		}
	}

	_, err = h.Queue.EnqueueUniqueIn(string(JobNotifyExpiry), int64(diff), work.Q{
		jobArgOrderID:   int64(order.ID),
		jobArgPreExpiry: false,
	})
	if err != nil {
		l.Error("error scheduling job", zap.Error(err))
		return err
	}

	message := "Taking orders for " + title + ", ending at " + expiry
	if isTomorrow {
		message += " tomorrow"
	}

	h.Bot.SendMessage(chatID, false, message)

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
			h.Bot.SendMessage(chatID, false, "No active orders! Create new order collection like /neworder 15:30 SoGood Bakery")
			return nil
		}

		l.Error("error fetching active orders", zap.Error(err))
		return err
	}

	split := strings.Split(text, " ")

	if len(split) < 3 {
		h.Bot.SendMessage(chatID, false, "Invalid order! Add to the order like /order 2 chicken pie")
		return nil
	}

	quantity, err := strconv.Atoi(split[1])
	if err != nil || quantity <= 0 {
		h.Bot.SendMessage(chatID, false, "Invalid quantity! Add to the order like /order 2 chicken pie")
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
		itemsText += fmt.Sprintf("[%s](tg://user?id=%d) %d x %s\n", item.UserName, item.UserID, item.Quantity, item.Name)
		lowered := strings.ToLower(item.Name)
		allItems[lowered] += int(item.Quantity)
	}

	allItemsText := ""
	for name, quantity := range allItems {
		allItemsText += fmt.Sprintf("%d x %s\n", quantity, name)
	}

	message := fmt.Sprintf(`
*%s*
%s

%s

*Consolidated*
%s
`, title, expiry, itemsText, allItemsText)

	h.Bot.SendMessage(int64(order.ChatID), true, message)

	return nil
}

func getLocation() (*time.Location, error) {
	return time.LoadLocation("Asia/Singapore")
}

func (h *Handlers) handleDeleteOrder(chatID int64, user models.User) error {
	l := h.Logger.With(zap.Int64("chat_id", chatID), zap.String("command", "/deleteorder"))

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
		l.Error("failed to retrieve active order", zap.Error(err))
		return err
	}

	items, err := h.Repo.GetUserItems(context.Background(), models.GetUserItemsParams{
		UserID: int32(user.ID),
		OrderID: order.ID,
	})
	if err != nil {
		l.Error("failed to retrieve user items", zap.Error(err))
		return err
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, len(items))
	for i, item := range items {
		rows[i] = tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(item.Name, "/delete " + strconv.Itoa(int(item.ID))),
		)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Cancel", "/cancel"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	h.Bot.SendInlineKeyboardMessage(chatID, "Select order item to delete", keyboard)

	return nil
}

func (h *Handlers) handleDeleteItem(callbackQuery models.CallbackQuery) error {
	if callbackQuery.Message == nil {
		return nil
	}
	l := h.Logger.With(zap.Int64("chat_id", callbackQuery.Message.Chat.ID), zap.String("command", "/deleteitem"))

	split := strings.Split(callbackQuery.Data, " ")
	if len(split) < 2 {
		l.Error("invalid delete item format", zap.String("data", callbackQuery.Data))
		msg := tgbotapi.EditMessageTextConfig{
			BaseEdit: tgbotapi.BaseEdit{
				ChatID: callbackQuery.Message.Chat.ID,
				MessageID: callbackQuery.Message.MessageID,
			},
			Text: "Invalid Item",
		}
		h.Bot.BotAPI.Send(msg)
		return nil
	 }

	itemID, err := strconv.Atoi(split[1])
	if err != nil {
	l.Error("invalid item id", zap.String("data", callbackQuery.Data), zap.Error(err))
	msg := tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID: callbackQuery.Message.Chat.ID,
			MessageID: callbackQuery.Message.MessageID,
		},
		Text: "Invalid Item",
	}
	h.Bot.BotAPI.Send(msg)
	return nil
	}
	
	item, err := h.Repo.DeleteItemByUser(context.Background(), models.DeleteItemByUserParams{
		ID: int32(itemID),
		UserID: int32(callbackQuery.From.ID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			msg := tgbotapi.EditMessageTextConfig{
				BaseEdit: tgbotapi.BaseEdit{
					ChatID: callbackQuery.Message.Chat.ID,
					MessageID: callbackQuery.Message.MessageID,
				},
				Text: "Invalid Item",
			}
			h.Bot.BotAPI.Send(msg)
			return nil
		}
		return err
	}
	msg := tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID: callbackQuery.Message.Chat.ID,
			MessageID: callbackQuery.Message.MessageID,
		},
		Text: "Order item removed",
	}
	h.Bot.BotAPI.Send(msg)
	return nil
}

func (h *Handlers)  handleCancelDeleteOrder(callbackQuery models.CallbackQuery) error {
	 if callbackQuery.Message != nil {
		 msg := tgbotapi.EditMessageTextConfig{
			 BaseEdit: tgbotapi.BaseEdit{
				 ChatID: callbackQuery.Message.Chat.ID,
				 MessageID: callbackQuery.Message.MessageID,
			 },
			 Text: "Canceled delete order",
		 }
		h.Bot.BotAPI.Send(msg)
	 }
	return nil
}
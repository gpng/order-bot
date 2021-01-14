package handlers

import "fmt"

// messages
const (
	MsgError                      = "Oops, something went wrong"
	MsgTakeOrder                  = "Start taking orders using /takeorders 15:00 Coffeeshop Kopi"
	MsgEndTakeOrders              = "Use /endorders to stop taking orders"
	MsgOrder                      = "Add orders using /order 2 kopi o kosong"
	MsgNewTakeOrderInvalidFormat  = "Invalid format! " + MsgTakeOrder
	MsgNewTakeOrderInvalidTime    = "Invalid time! " + MsgTakeOrder
	MsgCancelTakeOrders           = "Stopped taking orders"
	MsgNoActiveOrders             = "No active orders! " + MsgTakeOrder
	MsgOrderInvalidFormat         = "Invalid order! " + MsgOrder
	MsgOrderInvalidQuantity       = "Invalid quantity! " + MsgOrder
	MsgNoOrders                   = "You have no current orders"
	MsgSelectDeleteOrder          = "Select order item to delete"
	MsgInvalidItem                = "Invalid Item"
	MsgCanceledDeleteOrderRequest = "Canceled cancel order request"
	MsgCancelOrder                = "Cancel your order using /cancelorder"
)

// MsgNewTakeOrderExistingOrder message
func MsgNewTakeOrderExistingOrder(title string) string {
	return "There is already an existing order for " + title + ". " + MsgEndTakeOrders
}

// MsgDeletedOrder message
func MsgDeletedOrder(quantity int, name string) string {
	return fmt.Sprintf("Deleted order: %d x %s", quantity, name)
}

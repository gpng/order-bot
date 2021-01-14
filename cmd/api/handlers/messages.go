package handlers

import "fmt"

// messages
const (
	MsgError                      = "Oops, something went wrong"
	MsgTakeOrder                  = "Create new order collection like /neworder 15:30 Coffeeshop Kopi"
	MsgCancelOrder                = "Use /canceltakeorders to cancel the current active take order"
	MsgOrder                      = "Add to the order like /order 2 kopi o kosong"
	MsgNewTakeOrderInvalidFormat  = "Invalid format! " + MsgTakeOrder
	MsgNewTakeOrderInvalidTime    = "Invalid time! " + MsgTakeOrder
	MsgCancelTakeOrders           = "Active take orders cancelled"
	MsgNoActiveOrders             = "No active orders! " + MsgTakeOrder
	MsgOrderInvalidFormat         = "Invalid order! " + MsgOrder
	MsgOrderInvalidQuantity       = "Invalid quantity! " + MsgOrder
	MsgNoOrders                   = "You have no current orders"
	MsgSelectDeleteOrder          = "Select order item to delete"
	MsgInvalidItem                = "Invalid Item"
	MsgCanceledDeleteOrderRequest = "Canceled cancel order request"
)

// MsgNewTakeOrderExistingOrder message
func MsgNewTakeOrderExistingOrder(title string) string {
	return "There is already an existing order for " + title + ". " + MsgCancelOrder
}

// MsgDeletedOrder message
func MsgDeletedOrder(quantity int, name string) string {
	return fmt.Sprintf("Deleted order: %d x %s", quantity, name)
}

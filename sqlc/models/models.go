// Code generated by sqlc. DO NOT EDIT.

package models

import (
	"database/sql"
)

type Item struct {
	ID       int32  `json:"id"`
	UserID   int32  `json:"user_id"`
	UserName string `json:"user_name"`
	OrderID  int32  `json:"order_id"`
	Quantity int32  `json:"quantity"`
	Name     string `json:"name"`
}

type Order struct {
	ID            int32          `json:"id"`
	ChatID        int32          `json:"chat_id"`
	Title         string         `json:"title"`
	Expiry        sql.NullTime   `json:"expiry"`
	Active        bool           `json:"active"`
	ReminderRunAt sql.NullInt64  `json:"reminder_run_at"`
	ReminderID    sql.NullString `json:"reminder_id"`
	ExpiryRunAt   sql.NullInt64  `json:"expiry_run_at"`
	ExpiryID      sql.NullString `json:"expiry_id"`
}

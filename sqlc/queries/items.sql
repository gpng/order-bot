-- name: CreateItem :one
INSERT INTO items (order_id, quantity, name, user_id, user_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetItemsByOrderID :many
SELECT * FROM items
WHERE order_id = $1;

-- name: GetItem :one
SELECT * FROM items
WHERE order_id = $1
AND user_id = $2
AND LOWER(name) = LOWER($3);

-- name: UpdateItemQuantity :one
UPDATE items
SET quantity = $4
WHERE order_id = $1
AND user_id = $2
AND LOWER(name) = LOWER($3)
RETURNING *;

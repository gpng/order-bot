-- name: CreateOrder :one
INSERT INTO orders (chat_id, title, expiry)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetActiveOrder :one
SELECT * FROM orders
WHERE chat_id = $1
AND expiry > $2
AND active = TRUE;

-- name: CancelOrder :one
UPDATE orders
SET active = FALSE
WHERE chat_id = $1
AND active = TRUE
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders
WHERE id = $1;

-- name: DeactivateOrder :exec
UPDATE orders
SET active = FALSE
WHERE id = $1;
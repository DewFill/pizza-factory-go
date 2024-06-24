-- name: GetOrder :one
SELECT id AS order_id, is_done as done
FROM orders
WHERE id = $1
LIMIT 1;


-- name: ListOrders :many
SELECT orders.id AS order_id, orders.is_done as done
FROM orders;


-- name: ListOrdersByDone :many
SELECT orders.id AS order_id, orders.is_done as done
FROM orders
WHERE is_done = $1;


-- name: GetOrderWithItemIds :one
SELECT orders.id, orders.is_done, ARRAY_AGG(oi.id)::int[] AS item_ids
FROM orders
         JOIN order_items oi ON orders.id = oi.order_id
WHERE orders.id = $1
GROUP BY orders.id
LIMIT 1;


-- name: GetOrderStatus :one
SELECT is_done AS done
FROM orders
WHERE id = $1;


-- name: SetOrderDone :exec
UPDATE orders
SET is_done = true
WHERE id = $1;


-- name: CreateOrder :one
INSERT INTO orders (is_done)
VALUES ($1)
RETURNING id, is_done;

-- name: CreateOrderItems :one
INSERT INTO order_items (order_id, item_id)
VALUES ($1, $2)
RETURNING id;

-- name: InsertOrder :one
INSERT INTO orders (
    id,
    idempotency_key,
    establishment_id,
    customer_id,
    delivery_id,
    status_id,
    total_price,
    rejection_reason,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING
    id,
    idempotency_key,
    establishment_id,
    customer_id,
    delivery_id,
    status_id,
    total_price,
    rejection_reason,
    created_at,
    updated_at;

-- name: InsertOrderItem :one
INSERT INTO order_item (
    id, order_id, product_id, external_product_id, product_name,
    unit_price, quantity, total_sum
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, order_id, product_id, external_product_id, product_name,
    unit_price, quantity, total_sum;

-- name: SelectOrderByID :one
SELECT
    id,
    idempotency_key,
    establishment_id,
    customer_id,
    delivery_id,
    status_id,
    total_price,
    rejection_reason,
    created_at,
    updated_at
FROM orders
WHERE id = $1;

-- name: SelectOrderByIDForUpdate :one
SELECT
    id,
    idempotency_key,
    establishment_id,
    customer_id,
    delivery_id,
    status_id,
    total_price,
    rejection_reason,
    created_at,
    updated_at
FROM orders
WHERE id = $1
FOR UPDATE;

-- name: SelectOrderItemsByOrderID :many
SELECT id, order_id, product_id, external_product_id, product_name,
    unit_price, quantity, total_sum
FROM order_item
WHERE order_id = $1
ORDER BY id;

-- name: SelectOrderByIdempotencyKey :one
SELECT
    id,
    idempotency_key,
    establishment_id,
    customer_id,
    delivery_id,
    status_id,
    total_price,
    rejection_reason,
    created_at,
    updated_at
FROM orders
WHERE idempotency_key = $1
FOR UPDATE;

-- name: SelectOrdersByEstablishmentID :many
SELECT
    id,
    idempotency_key,
    establishment_id,
    customer_id,
    delivery_id,
    status_id,
    total_price,
    rejection_reason,
    created_at,
    updated_at
FROM orders
WHERE establishment_id = $1
ORDER BY created_at, id;

-- name: SelectOrdersByEstablishmentIDAndStatus :many
SELECT
    id, idempotency_key, establishment_id, customer_id, delivery_id,
    status_id, total_price, rejection_reason, created_at, updated_at
FROM orders
WHERE establishment_id = $1
  AND status_id = $2
ORDER BY created_at, id;

-- name: UpdateOrderStatusByID :one
UPDATE orders
SET status_id = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING
    id,
    idempotency_key,
    establishment_id,
    customer_id,
    delivery_id,
    status_id,
    total_price,
    rejection_reason,
    created_at,
    updated_at;

-- name: RejectOrderByID :one
UPDATE orders
SET status_id = 6,
    rejection_reason = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING
    id, idempotency_key, establishment_id, customer_id, delivery_id,
    status_id, total_price, rejection_reason, created_at, updated_at;

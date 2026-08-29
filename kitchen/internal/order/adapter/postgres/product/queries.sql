-- name: SelectProductByID :one
SELECT
    id,
    external_id,
    establishment_id,
    category_id,
    name,
    description,
    available,
    inventory_type_id,
    quantity,
    price,
    created_at,
    updated_at,
    deleted_at
FROM product
WHERE id = $1;

-- name: SelectProductByIDForUpdate :one
SELECT
    id,
    external_id,
    establishment_id,
    category_id,
    name,
    description,
    available,
    inventory_type_id,
    quantity,
    price,
    created_at,
    updated_at,
    deleted_at
FROM product
WHERE id = $1
FOR UPDATE;

-- name: DecrementProductQuantity :execrows
UPDATE product
SET quantity = quantity - $2,
    updated_at = NOW()
WHERE id = $1
  AND inventory_type_id = 1
  AND quantity >= $2;

-- name: IncrementProductQuantity :execrows
UPDATE product
SET quantity = quantity + $2,
    updated_at = NOW()
WHERE id = $1
  AND inventory_type_id = 1;

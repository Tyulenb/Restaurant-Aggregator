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

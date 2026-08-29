-- name: SelectProductsByEstablishmentIdAndCategoryExternalId :many
SELECT
    p.id,
    p.external_id,
    p.establishment_id,
    p.category_id,
    p.name,
    p.description,
    p.available,
    p.inventory_type_id,
    p.quantity,
    p.price,
    p.created_at,
    p.updated_at,
    p.deleted_at
FROM product AS p
JOIN category AS c
  ON c.id = p.category_id
 AND c.establishment_id = p.establishment_id
WHERE p.establishment_id = $1
  AND c.external_id = $2
ORDER BY p.created_at, p.id;

-- name: SelectProductsByEstablishmentId :many
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
WHERE establishment_id = $1
ORDER BY created_at, id;

-- name: InsertProduct :one
INSERT INTO product (
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
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING
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
    deleted_at;

-- name: SelectProductByExternalId :one
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
WHERE establishment_id = $1
  AND external_id = $2;

-- name: SelectProductByExternalIdForUpdate :one
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
WHERE establishment_id = $1
AND external_id = $2
FOR UPDATE;

-- name: UpdateProductByExternalId :one
UPDATE product
SET category_id = CASE
        WHEN sqlc.narg(category_external_id)::text IS NULL THEN category_id
        ELSE (
            SELECT c.id
            FROM category AS c
            WHERE c.establishment_id = product.establishment_id
              AND c.external_id = sqlc.narg(category_external_id)::text
              AND c.deleted_at IS NULL
        )
    END,
    name = COALESCE(sqlc.narg(name)::text, name),
    description = CASE
        WHEN sqlc.arg(description_set)::boolean THEN sqlc.narg(description)::text
        ELSE description
    END,
    price = COALESCE(sqlc.narg(price)::numeric, price),
    available = COALESCE(sqlc.narg(available)::boolean, available),
    updated_at = NOW()
WHERE product.establishment_id = sqlc.arg(establishment_id)
  AND product.external_id = sqlc.arg(external_id)
  AND product.deleted_at IS NULL
RETURNING
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
    deleted_at;

-- name: DeleteProductByExternalId :one
UPDATE product
SET deleted_at = COALESCE(deleted_at, NOW()),
    updated_at = CASE WHEN deleted_at IS NULL THEN NOW() ELSE updated_at END
WHERE establishment_id = $1
  AND external_id = $2
RETURNING
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
    deleted_at;

-- name: UpdateProductQuantity :one
UPDATE product
SET quantity = $1,
    updated_at = NOW()
WHERE establishment_id = $2
  AND external_id = $3
  AND deleted_at IS NULL
  AND inventory_type_id = 1
RETURNING
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
    deleted_at;

-- name: DeleteProductByCategoryId :many
UPDATE product
SET deleted_at = COALESCE(deleted_at, NOW()),
    updated_at = CASE WHEN deleted_at IS NULL THEN NOW() ELSE updated_at END
WHERE category_id = $1
RETURNING
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
    deleted_at;

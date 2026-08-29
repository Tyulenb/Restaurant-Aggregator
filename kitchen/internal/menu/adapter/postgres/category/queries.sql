-- name: SelectCategories :many
SELECT
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at
FROM category
WHERE establishment_id = $1
  AND deleted_at IS NULL
ORDER BY created_at, id;

-- name: SelectCategoriesWithDeleted :many
SELECT
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at
FROM category
WHERE establishment_id = $1
ORDER BY created_at, id;

-- name: SelectCategoryByExternalID :one
SELECT
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at
FROM category
WHERE establishment_id = $1
  AND external_id = $2;

-- name: InsertCategory :one
INSERT INTO category (
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at;

-- name: UpdateCategoryNameByExternalID :one
UPDATE category
SET name = $1,
    updated_at = NOW()
WHERE establishment_id = $2
  AND external_id = $3
  AND deleted_at IS NULL
RETURNING
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at;

-- name: UpdateCategoryDescriptionByExternalID :one
UPDATE category
SET description = $1,
    updated_at = NOW()
WHERE establishment_id = $2
  AND external_id = $3
  AND deleted_at IS NULL
RETURNING
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at;

-- name: UpdateCategoryNameAndDescriptionByExternalID :one
UPDATE category
SET name = $1,
    description = $2,
    updated_at = NOW()
WHERE establishment_id = $3
  AND external_id = $4
  AND deleted_at IS NULL
RETURNING
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at;

-- name: DeleteCategoryByExternalID :one
UPDATE category
SET deleted_at = COALESCE(deleted_at, NOW()),
    updated_at = CASE WHEN deleted_at IS NULL THEN NOW() ELSE updated_at END
WHERE establishment_id = $1
  AND external_id = $2
RETURNING
    id,
    establishment_id,
    external_id,
    name,
    description,
    created_at,
    updated_at,
    deleted_at;

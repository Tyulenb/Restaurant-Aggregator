-- name: GetEstablishments :many
SELECT * FROM establishment
ORDER BY created_at, id;

-- name: GetEstablishmentsByAvailability :many
SELECT * FROM establishment
WHERE available = $1
ORDER BY created_at, id;

-- name: GetEstablishmentByID :one
SELECT * FROM establishment
WHERE id = $1;

-- name: UpdateEstablishmentAvailability :one
UPDATE establishment
SET available = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING id, name, description, available, created_at, updated_at;

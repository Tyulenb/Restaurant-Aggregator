-- name: GetAvailableEstablishments :many
SELECT * FROM establishment
WHERE available = True;

-- name: UpdateEstablishmentAvailability :one
UPDATE establishment
SET available = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING id, name, description, available, created_at, updated_at;

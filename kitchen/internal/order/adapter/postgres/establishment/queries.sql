-- name: SelectEstablishmentByID :one
SELECT id, name, description, available, created_at, updated_at
FROM establishment
WHERE id = $1;

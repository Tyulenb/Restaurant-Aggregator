-- +goose Up
-- Development establishment used by the partner simulator in docker-compose.yaml.
INSERT INTO establishment (id, name, description, available, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Demo Pizzeria',
    'Establishment managed by the local demo simulator',
    FALSE,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;

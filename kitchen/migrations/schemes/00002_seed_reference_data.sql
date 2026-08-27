-- +goose Up
INSERT INTO inventory_type (id, code)
VALUES
    (1, 'finite'),
    (2, 'unlimited');

INSERT INTO order_status (id, code)
VALUES
    (1, 'created'),
    (2, 'accepted'),
    (3, 'cooking'),
    (4, 'ready'),
    (5, 'completed'),
    (6, 'cancelled');

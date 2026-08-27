-- +goose Up
INSERT INTO inventory_type (id, inventory_type)
VALUES
    (1, 'finite'),
    (2, 'unlimited');

INSERT INTO order_status (id, status)
VALUES
    (1, 'created'),
    (2, 'accepted'),
    (3, 'preparing'),
    (4, 'ready'),
    (5, 'completed'),
    (6, 'rejected'),
    (7, 'cancelled');

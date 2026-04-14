-- +goose Up
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
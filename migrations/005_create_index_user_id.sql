-- +goose Up
CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);
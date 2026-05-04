-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT GENERATED ALWAYS AS IDENTITY,
    login VARCHAR(32) NOT NULL UNIQUE DEFAULT '',
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    PRIMARY KEY(user_id)
);

-- +goose Down
DROP TABLE IF EXISTS users;
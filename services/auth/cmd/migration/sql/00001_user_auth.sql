-- +goose Up
CREATE TABLE users(
 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
 deleted_at  TIMESTAMP DEFAULT NULL,
 name varchar(64) NOT NULL,
 email varchar(64) UNIQUE NOT NULL,
 password char(60) NOT NULL,
 user_uuid uuid PRIMARY KEY
);
ALTER TABLE users ADD CONSTRAINT min_len_name CHECK (char_length(name) >= 2);
ALTER TABLE users ADD CONSTRAINT check_valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');
CREATE INDEX idx_user_uuid ON users(user_uuid);
CREATE INDEX idx_user_deleted_at ON users(deleted_at);
-- +goose Down
DROP TABLE IF EXISTS users
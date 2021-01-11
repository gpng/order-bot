-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE orders (
  id SERIAL PRIMARY KEY,
  chat_id INT NOT NULL,
  title TEXT NOT NULL,
  expiry TIMESTAMP NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS orders;

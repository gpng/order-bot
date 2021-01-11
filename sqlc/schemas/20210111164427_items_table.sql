-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE items (
  id SERIAL PRIMARY KEY,
  user_id INT NOT NULL,
  user_name TEXT NOT NULL,
  order_id INT NOT NULL REFERENCES orders(id),
  quantity INT NOT NULL,
  name TEXT NOT NULL
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS items;

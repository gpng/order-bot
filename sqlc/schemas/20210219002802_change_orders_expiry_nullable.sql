-- +goose Up
-- SQL in this section is executed when the migration is applied.
ALTER TABLE orders ALTER COLUMN expiry DROP NOT NULL;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE orders ALTER COLUMN expiry SET NOT NULL;
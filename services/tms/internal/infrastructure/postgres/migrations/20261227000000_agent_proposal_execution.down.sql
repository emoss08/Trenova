-- Postgres cannot remove a value from an enum type, so the added members stay.
-- Nothing else in this migration is reversible; the columns belong to the
-- migration that follows it.
SELECT 1;

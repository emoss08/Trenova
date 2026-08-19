-- PostgreSQL cannot remove a value from an enum. The value is harmless when
-- unused, so the down migration leaves it in place.
SELECT 1;

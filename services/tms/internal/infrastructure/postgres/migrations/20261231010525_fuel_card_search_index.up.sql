CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_fuel_cards_search" ON "fuel_cards" USING GIN("search_vector");

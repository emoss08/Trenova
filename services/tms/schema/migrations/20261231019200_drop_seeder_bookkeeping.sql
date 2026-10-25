DROP TABLE IF EXISTS "seed_history", "seed_created_entities";

DELETE FROM trenova_rls.global_tables
WHERE "table_name" IN ('seed_history', 'seed_created_entities');

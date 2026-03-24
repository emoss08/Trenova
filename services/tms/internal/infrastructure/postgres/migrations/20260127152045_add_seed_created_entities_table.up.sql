CREATE TABLE IF NOT EXISTS seed_created_entities (
    id SERIAL PRIMARY KEY,
    seed_name VARCHAR(255) NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    created_at BIGINT NOT NULL,
    UNIQUE(seed_name, table_name, entity_id)
);

CREATE INDEX idx_seed_created_entities_seed_name ON seed_created_entities(seed_name);
CREATE INDEX idx_seed_created_entities_table_name ON seed_created_entities(table_name);
CREATE INDEX idx_seed_created_entities_entity_id ON seed_created_entities(entity_id);

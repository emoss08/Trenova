CREATE TABLE IF NOT EXISTS "page_favorites"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Relationship identifiers (Non-Primary-Keys)
    "user_id" varchar(100) NOT NULL,
    -- Core fields
    "page_url" varchar(500) NOT NULL,
    "page_title" varchar(255) NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_page_favorites_business_unit_id" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id"),
    CONSTRAINT "fk_page_favorites_organization_id" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id"),
    CONSTRAINT "fk_page_favorites_user_id" FOREIGN KEY ("user_id") REFERENCES "users"("id")
);


-- Which page tours a person has already been shown. A tour opens by itself the first time
-- someone lands on its page; this is what keeps it from opening again, on any device. A tour
-- whose steps change raises its version, and a view of an older version no longer counts.
CREATE TABLE IF NOT EXISTS "page_tour_views"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "tour" varchar(64) NOT NULL,
    "tour_version" integer NOT NULL,
    "outcome" varchar(20) NOT NULL,
    "seen_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_page_tour_views" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_page_tour_views_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_page_tour_views_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_page_tour_views_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_page_tour_views_outcome" CHECK ("outcome" IN ('Finished', 'Skipped')),
    CONSTRAINT "ck_page_tour_views_tour_version" CHECK ("tour_version" BETWEEN 1 AND 10000)
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_page_tour_views_user_tour" ON "page_tour_views"("organization_id", "business_unit_id", "user_id", "tour");

CREATE INDEX IF NOT EXISTS "idx_page_tour_views_business_unit" ON "page_tour_views"("business_unit_id");

COMMENT ON TABLE "page_tour_views" IS 'The page tours a person has been shown: which tour, which version of it, whether they finished or skipped it, and when';

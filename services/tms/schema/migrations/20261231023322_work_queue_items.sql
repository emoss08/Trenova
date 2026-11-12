CREATE TABLE IF NOT EXISTS "work_queue_items" (
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "kind" varchar(60) NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "due_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_work_queue_items" PRIMARY KEY ("organization_id", "business_unit_id", "kind", "subject_id"),
    CONSTRAINT "fk_work_queue_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_work_queue_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- The drain claims a tenant's due work of one kind, oldest first.
CREATE INDEX IF NOT EXISTS "idx_work_queue_items_tenant_due" ON "work_queue_items"("organization_id", "business_unit_id", "kind", "due_at");

-- The backstop sweep lists the tenants with due work of a kind.
CREATE INDEX IF NOT EXISTS "idx_work_queue_items_kind_due" ON "work_queue_items"("kind", "due_at");

COMMENT ON TABLE "work_queue_items" IS 'Subjects waiting for background work of a kind, drained per tenant by a debounced workflow';

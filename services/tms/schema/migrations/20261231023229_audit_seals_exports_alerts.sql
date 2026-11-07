CREATE TABLE IF NOT EXISTS "audit_chain_heads"(
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "last_seq" BIGINT NOT NULL DEFAULT 0,
    "last_hash" VARCHAR(64) NOT NULL DEFAULT '',
    "sealed_through" BIGINT NOT NULL DEFAULT 0,
    "hash_key_id" VARCHAR(40),
    "last_verified_seq" BIGINT NOT NULL DEFAULT 0,
    "last_verified_at" BIGINT,
    "last_verification_status" VARCHAR(20),
    "last_verification_failed_seq" BIGINT,
    "last_verification_detail" TEXT,
    "verify_cursor_seq" BIGINT NOT NULL DEFAULT 0,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_audit_chain_heads" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "fk_audit_chain_heads_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_chain_heads_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_audit_chain_heads_verification" CHECK ("last_verification_status" IS NULL OR "last_verification_status" IN ('Verified', 'Broken', 'KeyMissing'))
);

COMMENT ON TABLE "audit_chain_heads" IS 'Where each tenant''s chain of audit seals ends, how far its entries are sealed and what the last verification found';

CREATE TABLE IF NOT EXISTS "audit_seals"(
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "seq" BIGINT NOT NULL,
    "from_ts" BIGINT NOT NULL,
    "to_ts" BIGINT NOT NULL,
    "row_count" BIGINT NOT NULL,
    "class_digests" JSONB NOT NULL,
    "prev_hash" VARCHAR(64) NOT NULL,
    "hash" VARCHAR(64) NOT NULL,
    "hash_key_id" VARCHAR(40),
    "hash_version" SMALLINT NOT NULL,
    "sealed_at" BIGINT NOT NULL,
    CONSTRAINT "pk_audit_seals" PRIMARY KEY ("organization_id", "business_unit_id", "seq"),
    CONSTRAINT "fk_audit_seals_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_seals_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_audit_seals_range" CHECK ("to_ts" > "from_ts"),
    CONSTRAINT "ck_audit_seals_seq" CHECK ("seq" > 0),
    CONSTRAINT "ck_audit_seals_digests" CHECK (jsonb_typeof("class_digests") = 'object'),
    CONSTRAINT "ck_audit_seals_hash_version" CHECK ("hash_version" IN (1, 2))
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_audit_seals_tenant_range" ON "audit_seals"("organization_id", "business_unit_id", "from_ts");

COMMENT ON TABLE "audit_seals" IS 'One signed seal per tenant per sealed hour of audit entries: the count and SHA-256 digest of each retention class''s rows, chained to the seal before it';

INSERT INTO trenova_rls.append_only_tables("table_name", "reason")
VALUES ('audit_seals', 'Audit seals prove the trail was not changed; only the retention sweep, as a bypass member, removes seals older than every entry it could cover')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

CREATE TABLE IF NOT EXISTS "audit_exports"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "requested_by_id" VARCHAR(100) NOT NULL,
    "format" VARCHAR(10) NOT NULL,
    "filters" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "range_from" BIGINT NOT NULL,
    "range_to" BIGINT NOT NULL,
    "include_security" BOOLEAN NOT NULL DEFAULT FALSE,
    "status" VARCHAR(20) NOT NULL DEFAULT 'Pending',
    "row_count" BIGINT NOT NULL DEFAULT 0,
    "byte_size" BIGINT NOT NULL DEFAULT 0,
    "sha256" VARCHAR(64),
    "artifact_key" VARCHAR(500),
    "artifact_expires_at" BIGINT,
    "workflow_id" VARCHAR(255),
    "error_message" TEXT,
    "started_at" BIGINT,
    "completed_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_audit_exports" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_audit_exports_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_exports_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_exports_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_audit_exports_format" CHECK ("format" IN ('CSV', 'JSONL')),
    CONSTRAINT "ck_audit_exports_status" CHECK ("status" IN ('Pending', 'Running', 'Succeeded', 'Failed', 'Expired')),
    CONSTRAINT "ck_audit_exports_range" CHECK ("range_to" > "range_from"),
    CONSTRAINT "ck_audit_exports_filters" CHECK (jsonb_typeof("filters") = 'object')
);

CREATE INDEX IF NOT EXISTS "idx_audit_exports_tenant_created" ON "audit_exports"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_audit_exports_expiry" ON "audit_exports"("artifact_expires_at") WHERE "artifact_key" IS NOT NULL;

COMMENT ON TABLE "audit_exports" IS 'Requests to write a range of the audit trail to a CSV or JSON Lines file, and the file each produced';

CREATE TABLE IF NOT EXISTS "audit_alert_policies"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "kind" VARCHAR(50) NOT NULL,
    "enabled" BOOLEAN NOT NULL DEFAULT TRUE,
    "threshold" INTEGER NOT NULL DEFAULT 1,
    "window_minutes" INTEGER NOT NULL DEFAULT 10,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_audit_alert_policies" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_audit_alert_policies_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_alert_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_audit_alert_policies_kind" CHECK ("kind" IN (
        'RoleChanged', 'APIKeyChanged', 'MFAReset', 'IdentityConfigChanged', 'CrossTenantRefused',
        'MalwareRefused', 'LegalHoldChanged', 'BulkDelete', 'BulkExport'
    )),
    CONSTRAINT "ck_audit_alert_policies_threshold" CHECK ("threshold" BETWEEN 1 AND 100000),
    CONSTRAINT "ck_audit_alert_policies_window" CHECK ("window_minutes" BETWEEN 1 AND 1440)
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_audit_alert_policies_tenant_kind" ON "audit_alert_policies"("organization_id", "business_unit_id", "kind");

COMMENT ON TABLE "audit_alert_policies" IS 'An organization''s choices for each security alert raised from the audit trail; a kind with no row uses its default';

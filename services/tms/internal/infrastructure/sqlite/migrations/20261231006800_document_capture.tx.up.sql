-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006800_document_capture.tx.up.sql

CREATE TABLE IF NOT EXISTS "capture_devices"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "machine_name" TEXT NOT NULL,
    "windows_user" TEXT,
    "agent_version" TEXT NOT NULL,
    "architecture" TEXT NOT NULL,
    "os_version" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "refresh_token_hash" TEXT NOT NULL,
    "previous_refresh_hash" TEXT,
    "access_token_hash" TEXT NOT NULL,
    "access_token_expires_at" INTEGER NOT NULL,
    "sources" TEXT NOT NULL DEFAULT '[]',
    "last_seen_at" INTEGER,
    "last_ip" TEXT,
    "revoked_at" INTEGER,
    "revoked_by_id" TEXT,
    "revoked_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_capture_devices" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_devices_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_devices_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_devices_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_devices_architecture" CHECK ("architecture" IN ('x64')),
    CONSTRAINT "ck_capture_devices_status" CHECK ("status" IN ('Active', 'Revoked'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_devices_access_token" ON "capture_devices" ("access_token_hash");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_devices_refresh_token" ON "capture_devices" ("refresh_token_hash");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_devices_previous_refresh" ON "capture_devices" ("previous_refresh_hash")WHERE "previous_refresh_hash" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_devices_user" ON "capture_devices" ("organization_id", "business_unit_id", "user_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "capture_pairings"(
    "id" TEXT NOT NULL,
    "device_code_hash" TEXT NOT NULL,
    "user_code" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "machine_name" TEXT NOT NULL,
    "windows_user" TEXT,
    "agent_version" TEXT NOT NULL,
    "architecture" TEXT NOT NULL,
    "os_version" TEXT,
    "client_ip" TEXT,
    "organization_id" TEXT,
    "business_unit_id" TEXT,
    "approved_by_id" TEXT,
    "device_name" TEXT,
    "device_id" TEXT,
    "last_polled_at" INTEGER,
    "expires_at" INTEGER NOT NULL,
    "decided_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_capture_pairings" PRIMARY KEY ("id"),
    CONSTRAINT "fk_capture_pairings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_pairings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_pairings_architecture" CHECK ("architecture" IN ('x64')),
    CONSTRAINT "ck_capture_pairings_status" CHECK ("status" IN ('Pending', 'Approved', 'Denied', 'Consumed', 'Expired'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_pairings_device_code" ON "capture_pairings" ("device_code_hash");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_pairings_user_code_open" ON "capture_pairings" ("user_code")WHERE "status" IN ('Pending', 'Approved');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_pairings_expiry" ON "capture_pairings" ("expires_at")WHERE "status" IN ('Pending', 'Approved');

--bun:split

CREATE TABLE IF NOT EXISTS "capture_profiles"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "dpi" INTEGER NOT NULL DEFAULT 300,
    "pixel_type" TEXT NOT NULL DEFAULT 'BlackWhite',
    "duplex" INTEGER NOT NULL DEFAULT 1,
    "use_feeder" INTEGER NOT NULL DEFAULT 1,
    "discard_blank_pages" INTEGER NOT NULL DEFAULT 1,
    "jpeg_quality" INTEGER NOT NULL DEFAULT 80,
    "show_driver_ui" INTEGER NOT NULL DEFAULT 0,
    "separator_strategies" TEXT NOT NULL DEFAULT '[]',
    "fixed_page_count" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_capture_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_profiles_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_profiles_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_profiles_status" CHECK ("status" IN ('Active', 'Inactive')),
    CONSTRAINT "ck_capture_profiles_pixel_type" CHECK ("pixel_type" IN ('BlackWhite', 'Grayscale', 'Color')),
    CONSTRAINT "ck_capture_profiles_dpi" CHECK ("dpi" BETWEEN 100 AND 600),
    CONSTRAINT "ck_capture_profiles_jpeg_quality" CHECK ("jpeg_quality" BETWEEN 30 AND 95)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_profiles_name" ON "capture_profiles" ("organization_id", "business_unit_id", lower("name"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_profiles_default" ON "capture_profiles" ("organization_id", "business_unit_id")WHERE "is_default";

--bun:split

CREATE TABLE IF NOT EXISTS "capture_requests"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "device_id" TEXT NOT NULL,
    "mode" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "target_type" TEXT NOT NULL,
    "target_id" TEXT NOT NULL,
    "document_type_id" TEXT,
    "profile_id" TEXT,
    "source_name" TEXT,
    "batch_id" TEXT,
    "failure_code" TEXT,
    "failure_message" TEXT,
    "expires_at" INTEGER NOT NULL,
    "delivered_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_capture_requests" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_requests_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_requests_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_requests_device" FOREIGN KEY ("device_id", "business_unit_id", "organization_id") REFERENCES "capture_devices"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_requests_profile" FOREIGN KEY ("profile_id", "business_unit_id", "organization_id") REFERENCES "capture_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_capture_requests_mode" CHECK ("mode" IN ('Scan', 'Print')),
    CONSTRAINT "ck_capture_requests_status" CHECK ("status" IN ('Pending', 'Delivered', 'InProgress', 'Completed', 'Canceled', 'Expired', 'Failed')),
    CONSTRAINT "ck_capture_requests_failure_code" CHECK ("failure_code" IS NULL OR "failure_code" IN ('SOURCE_UNAVAILABLE', 'SOURCE_BUSY', 'PAPER_JAM', 'FEEDER_EMPTY', 'CANCELED_BY_USER', 'DRIVER_ERROR', 'UPLOAD_FAILED', 'NOT_DELIVERED', 'INTERNAL'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_requests_device_open" ON "capture_requests" ("organization_id", "business_unit_id", "device_id", "created_at")WHERE "status" IN ('Pending', 'Delivered', 'InProgress');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_requests_expiry" ON "capture_requests" ("expires_at")WHERE "status" IN ('Pending', 'Delivered');

--bun:split

CREATE TABLE IF NOT EXISTS "capture_batches"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "device_id" TEXT NOT NULL,
    "request_id" TEXT,
    "profile_id" TEXT,
    "client_key" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Receiving',
    "source_name" TEXT,
    "job_name" TEXT,
    "settings" TEXT NOT NULL DEFAULT '{}',
    "target_type" TEXT,
    "target_id" TEXT,
    "document_type_id" TEXT,
    "expected_page_count" INTEGER NOT NULL DEFAULT 0,
    "received_page_count" INTEGER NOT NULL DEFAULT 0,
    "item_count" INTEGER NOT NULL DEFAULT 0,
    "filed_item_count" INTEGER NOT NULL DEFAULT 0,
    "manifest_digest" TEXT,
    "failure_message" TEXT,
    "sealed_at" INTEGER,
    "processed_at" INTEGER,
    "retain_until" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_capture_batches" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_batches_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_batches_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_batches_device" FOREIGN KEY ("device_id", "business_unit_id", "organization_id") REFERENCES "capture_devices"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_batches_source" CHECK ("source" IN ('Scan', 'Print')),
    CONSTRAINT "ck_capture_batches_status" CHECK ("status" IN ('Receiving', 'Sealed', 'Processing', 'Ready', 'PartiallyFiled', 'Filed', 'Discarded', 'Expired', 'Failed')),
    CONSTRAINT "ck_capture_batches_pages" CHECK ("expected_page_count" BETWEEN 0 AND 1000 AND "received_page_count" BETWEEN 0 AND 1000)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_batches_client_key" ON "capture_batches" ("organization_id", "business_unit_id", "device_id", "client_key");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_batches_queue" ON "capture_batches" ("organization_id", "business_unit_id", "status", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_batches_user" ON "capture_batches" ("organization_id", "business_unit_id", "user_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_batches_retention" ON "capture_batches" ("retain_until")WHERE "status" NOT IN ('Receiving', 'Sealed', 'Processing');

--bun:split

CREATE TABLE IF NOT EXISTS "capture_pages"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "batch_id" TEXT NOT NULL,
    "sequence" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Received',
    "storage_path" TEXT NOT NULL,
    "checksum_sha256" TEXT NOT NULL,
    "byte_size" INTEGER NOT NULL,
    "content_type" TEXT NOT NULL,
    "width_px" INTEGER NOT NULL DEFAULT 0,
    "height_px" INTEGER NOT NULL DEFAULT 0,
    "dpi" INTEGER NOT NULL DEFAULT 0,
    "rotation" INTEGER NOT NULL DEFAULT 0,
    "thumbnail_path" TEXT,
    "blank_score" REAL,
    "is_separator" INTEGER NOT NULL DEFAULT 0,
    "markers" TEXT NOT NULL DEFAULT '{}',
    "failure_message" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_capture_pages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_pages_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_pages_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_pages_batch" FOREIGN KEY ("batch_id", "business_unit_id", "organization_id") REFERENCES "capture_batches"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_pages_status" CHECK ("status" IN ('Received', 'Processed', 'Failed')),
    CONSTRAINT "ck_capture_pages_rotation" CHECK ("rotation" IN (0, 90, 180, 270)),
    CONSTRAINT "ck_capture_pages_sequence" CHECK ("sequence" BETWEEN 1 AND 1000)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_pages_sequence" ON "capture_pages" ("batch_id", "business_unit_id", "organization_id", "sequence");

--bun:split

CREATE TABLE IF NOT EXISTS "capture_items"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "batch_id" TEXT NOT NULL,
    "position" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Proposed',
    "page_ids" TEXT NOT NULL,
    "suggested_type" TEXT,
    "suggested_id" TEXT,
    "suggested_doc_type_id" TEXT,
    "suggestion_source" TEXT,
    "suggestion_confidence" REAL,
    "suggestion_reason" TEXT,
    "cover_sheet_id" TEXT,
    "detected_kind" TEXT,
    "filed_type" TEXT,
    "filed_id" TEXT,
    "filed_doc_type_id" TEXT,
    "document_id" TEXT,
    "upload_session_id" TEXT,
    "filed_by_id" TEXT,
    "filed_at" INTEGER,
    "failure_message" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_capture_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_items_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_items_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_items_batch" FOREIGN KEY ("batch_id", "business_unit_id", "organization_id") REFERENCES "capture_batches"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_items_status" CHECK ("status" IN ('Proposed', 'Filing', 'Filed', 'Discarded', 'Failed')),
    CONSTRAINT "ck_capture_items_suggestion_source" CHECK ("suggestion_source" IS NULL OR "suggestion_source" IN ('CoverSheet', 'Request', 'Classifier', 'Person'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_capture_items_batch" ON "capture_items" ("batch_id", "business_unit_id", "organization_id", "position");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_items_document" ON "capture_items" ("organization_id", "business_unit_id", "document_id")WHERE "document_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "capture_cover_sheets"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "target_type" TEXT,
    "target_id" TEXT,
    "document_type_id" TEXT,
    "issued_by_id" TEXT NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "last_used_at" INTEGER,
    "use_count" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_capture_cover_sheets" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_cover_sheets_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_cover_sheets_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_cover_sheets_token" ON "capture_cover_sheets" ("organization_id", "business_unit_id", "token_hash");

--bun:split

ALTER TABLE "document_controls" ADD COLUMN "enable_capture" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "document_controls" ADD COLUMN "capture_auto_file_cover_sheets" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "document_controls" ADD COLUMN "capture_retention_days" INTEGER NOT NULL DEFAULT 30;

--bun:split

ALTER TABLE "document_controls" ADD COLUMN "capture_min_agent_version" TEXT;

--bun:split

ALTER TABLE "document_controls" ADD COLUMN "capture_allow_auto_update" INTEGER NOT NULL DEFAULT 1;

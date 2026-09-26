-- A paired Trenova Capture install, acting for one person.
--
-- Both tokens are stored hashed: a leaked row is something somebody can read,
-- not a credential they can present. The replaced refresh token is kept so a
-- second presentation of it can be recognised as the credential having been
-- copied, which revokes the device.
CREATE TABLE IF NOT EXISTS "capture_devices"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "machine_name" varchar(255) NOT NULL,
    "windows_user" varchar(255),
    "agent_version" varchar(50) NOT NULL,
    "architecture" varchar(10) NOT NULL,
    "os_version" varchar(100),
    "status" varchar(20) NOT NULL DEFAULT 'Active',
    "refresh_token_hash" varchar(128) NOT NULL,
    "previous_refresh_hash" varchar(128),
    "access_token_hash" varchar(128) NOT NULL,
    "access_token_expires_at" bigint NOT NULL,
    "sources" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "last_seen_at" bigint,
    "last_ip" varchar(64),
    "revoked_at" bigint,
    "revoked_by_id" varchar(100),
    "revoked_reason" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_capture_devices" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_devices_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_devices_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_devices_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_devices_architecture" CHECK ("architecture" IN ('x64')),
    CONSTRAINT "ck_capture_devices_status" CHECK ("status" IN ('Active', 'Revoked'))
);

--bun:split
-- Every request from a device resolves it by its access token, before anything
-- else is known about the caller, so this has to be one indexed read across
-- every tenant.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_devices_access_token" ON "capture_devices"("access_token_hash");

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_devices_refresh_token" ON "capture_devices"("refresh_token_hash");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_capture_devices_previous_refresh" ON "capture_devices"("previous_refresh_hash") WHERE "previous_refresh_hash" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_capture_devices_user" ON "capture_devices"("organization_id", "business_unit_id", "user_id", "status");

--bun:split
-- One device authorization grant. It has no tenant until a signed-in person
-- approves it: the machine asking for a code must not be able to choose which
-- organization it joins.
CREATE TABLE IF NOT EXISTS "capture_pairings"(
    "id" varchar(100) NOT NULL,
    "device_code_hash" varchar(128) NOT NULL,
    "user_code" varchar(16) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "machine_name" varchar(255) NOT NULL,
    "windows_user" varchar(255),
    "agent_version" varchar(50) NOT NULL,
    "architecture" varchar(10) NOT NULL,
    "os_version" varchar(100),
    "client_ip" varchar(64),
    "organization_id" varchar(100),
    "business_unit_id" varchar(100),
    "approved_by_id" varchar(100),
    "device_name" varchar(100),
    "device_id" varchar(100),
    "last_polled_at" bigint,
    "expires_at" bigint NOT NULL,
    "decided_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_capture_pairings" PRIMARY KEY ("id"),
    CONSTRAINT "fk_capture_pairings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_pairings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_pairings_architecture" CHECK ("architecture" IN ('x64')),
    CONSTRAINT "ck_capture_pairings_status" CHECK ("status" IN ('Pending', 'Approved', 'Denied', 'Consumed', 'Expired'))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_pairings_device_code" ON "capture_pairings"("device_code_hash");

--bun:split
-- A user code only has to be unique among the codes somebody could still
-- type, which is what keeps it short enough to type.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_pairings_user_code_open" ON "capture_pairings"("user_code") WHERE "status" IN ('Pending', 'Approved');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_capture_pairings_expiry" ON "capture_pairings"("expires_at") WHERE "status" IN ('Pending', 'Approved');

--bun:split
CREATE TABLE IF NOT EXISTS "capture_profiles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" varchar(500),
    "status" varchar(20) NOT NULL DEFAULT 'Active',
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "dpi" integer NOT NULL DEFAULT 300,
    "pixel_type" varchar(20) NOT NULL DEFAULT 'BlackWhite',
    "duplex" boolean NOT NULL DEFAULT TRUE,
    "use_feeder" boolean NOT NULL DEFAULT TRUE,
    "discard_blank_pages" boolean NOT NULL DEFAULT TRUE,
    "jpeg_quality" integer NOT NULL DEFAULT 80,
    "show_driver_ui" boolean NOT NULL DEFAULT FALSE,
    "separator_strategies" text[] NOT NULL DEFAULT '{}',
    "fixed_page_count" integer NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_capture_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_profiles_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_profiles_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_profiles_status" CHECK ("status" IN ('Active', 'Inactive')),
    CONSTRAINT "ck_capture_profiles_pixel_type" CHECK ("pixel_type" IN ('BlackWhite', 'Grayscale', 'Color')),
    CONSTRAINT "ck_capture_profiles_dpi" CHECK ("dpi" BETWEEN 100 AND 600),
    CONSTRAINT "ck_capture_profiles_jpeg_quality" CHECK ("jpeg_quality" BETWEEN 30 AND 95),
    CONSTRAINT "ck_capture_profiles_separator_strategies" CHECK ("separator_strategies" <@ ARRAY['PatchCode', 'CoverSheet', 'BlankPage', 'FixedPageCount']::text[])
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_profiles_name" ON "capture_profiles"("organization_id", "business_unit_id", lower("name"));

--bun:split
-- There is one default, or none. Two would leave the web app to pick, and it
-- would pick differently on different screens.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_profiles_default" ON "capture_profiles"("organization_id", "business_unit_id") WHERE "is_default";

--bun:split
CREATE TABLE IF NOT EXISTS "capture_requests"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "device_id" varchar(100) NOT NULL,
    "mode" varchar(10) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "target_type" varchar(50) NOT NULL,
    "target_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100),
    "profile_id" varchar(100),
    "source_name" varchar(255),
    "batch_id" varchar(100),
    "failure_code" varchar(40),
    "failure_message" varchar(500),
    "expires_at" bigint NOT NULL,
    "delivered_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_capture_requests" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_requests_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_requests_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_requests_device" FOREIGN KEY ("device_id", "business_unit_id", "organization_id") REFERENCES "capture_devices"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_requests_profile" FOREIGN KEY ("profile_id", "business_unit_id", "organization_id") REFERENCES "capture_profiles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("profile_id"),
    CONSTRAINT "ck_capture_requests_mode" CHECK ("mode" IN ('Scan', 'Print')),
    CONSTRAINT "ck_capture_requests_status" CHECK ("status" IN ('Pending', 'Delivered', 'InProgress', 'Completed', 'Canceled', 'Expired', 'Failed')),
    CONSTRAINT "ck_capture_requests_failure_code" CHECK ("failure_code" IS NULL OR "failure_code" IN ('SOURCE_UNAVAILABLE', 'SOURCE_BUSY', 'PAPER_JAM', 'FEEDER_EMPTY', 'CANCELED_BY_USER', 'DRIVER_ERROR', 'UPLOAD_FAILED', 'NOT_DELIVERED', 'INTERNAL'))
);

--bun:split
-- A device asks for its open requests every time its stream says something
-- changed and every time it reconnects.
CREATE INDEX IF NOT EXISTS "idx_capture_requests_device_open" ON "capture_requests"("organization_id", "business_unit_id", "device_id", "created_at") WHERE "status" IN ('Pending', 'Delivered', 'InProgress');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_capture_requests_expiry" ON "capture_requests"("expires_at") WHERE "status" IN ('Pending', 'Delivered');

--bun:split
CREATE TABLE IF NOT EXISTS "capture_batches"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "device_id" varchar(100) NOT NULL,
    "request_id" varchar(100),
    "profile_id" varchar(100),
    "client_key" varchar(100) NOT NULL,
    "source" varchar(10) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Receiving',
    "source_name" varchar(255),
    "job_name" varchar(255),
    "settings" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "target_type" varchar(50),
    "target_id" varchar(100),
    "document_type_id" varchar(100),
    "expected_page_count" integer NOT NULL DEFAULT 0,
    "received_page_count" integer NOT NULL DEFAULT 0,
    "item_count" integer NOT NULL DEFAULT 0,
    "filed_item_count" integer NOT NULL DEFAULT 0,
    "manifest_digest" varchar(64),
    "failure_message" varchar(500),
    "sealed_at" bigint,
    "processed_at" bigint,
    "retain_until" bigint NOT NULL,
    "retention_reminded_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_capture_batches" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_batches_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_batches_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_batches_device" FOREIGN KEY ("device_id", "business_unit_id", "organization_id") REFERENCES "capture_devices"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_batches_source" CHECK ("source" IN ('Scan', 'Print')),
    CONSTRAINT "ck_capture_batches_status" CHECK ("status" IN ('Receiving', 'Sealed', 'Processing', 'Ready', 'PartiallyFiled', 'Filed', 'Discarded', 'Expired', 'Failed')),
    CONSTRAINT "ck_capture_batches_pages" CHECK ("expected_page_count" BETWEEN 0 AND 1000 AND "received_page_count" BETWEEN 0 AND 1000)
);

--bun:split
-- The device names each batch before it opens it, so a retried open after a
-- dropped response finds the batch it already made instead of starting a
-- second one with half the pages.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_batches_client_key" ON "capture_batches"("organization_id", "business_unit_id", "device_id", "client_key");

--bun:split
-- The intake queue reads one tenant's open batches, newest first.
CREATE INDEX IF NOT EXISTS "idx_capture_batches_queue" ON "capture_batches"("organization_id", "business_unit_id", "status", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_capture_batches_user" ON "capture_batches"("organization_id", "business_unit_id", "user_id", "created_at" DESC);

--bun:split
-- Every batch leaves storage once its retention passes, filed or not: a filed
-- document has its own copy, and the pages behind it are only a working set.
CREATE INDEX IF NOT EXISTS "idx_capture_batches_retention" ON "capture_batches"("retain_until") WHERE "status" NOT IN ('Receiving', 'Sealed', 'Processing');

--bun:split
-- A stack still waiting on a person is its owner's to file; they are told a
-- week before its pages go, once.
CREATE INDEX IF NOT EXISTS "idx_capture_batches_retention_reminder" ON "capture_batches"("retain_until") WHERE "retention_reminded_at" IS NULL AND "status" IN ('Ready', 'PartiallyFiled');

--bun:split
CREATE TABLE IF NOT EXISTS "capture_pages"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "batch_id" varchar(100) NOT NULL,
    "sequence" integer NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Received',
    "storage_path" varchar(512) NOT NULL,
    "checksum_sha256" varchar(64) NOT NULL,
    "byte_size" bigint NOT NULL,
    "content_type" varchar(100) NOT NULL,
    "width_px" integer NOT NULL DEFAULT 0,
    "height_px" integer NOT NULL DEFAULT 0,
    "dpi" integer NOT NULL DEFAULT 0,
    "rotation" integer NOT NULL DEFAULT 0,
    "thumbnail_path" varchar(512),
    "blank_score" numeric(6, 5),
    "is_separator" boolean NOT NULL DEFAULT FALSE,
    "markers" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "failure_message" varchar(500),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_capture_pages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_pages_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_pages_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_pages_batch" FOREIGN KEY ("batch_id", "business_unit_id", "organization_id") REFERENCES "capture_batches"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_pages_status" CHECK ("status" IN ('Received', 'Processed', 'Failed')),
    CONSTRAINT "ck_capture_pages_rotation" CHECK ("rotation" IN (0, 90, 180, 270)),
    CONSTRAINT "ck_capture_pages_sequence" CHECK ("sequence" BETWEEN 1 AND 1000)
);

--bun:split
-- A retried upload of the same page lands on the page it already made.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_pages_sequence" ON "capture_pages"("batch_id", "business_unit_id", "organization_id", "sequence");

--bun:split
CREATE TABLE IF NOT EXISTS "capture_items"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "batch_id" varchar(100) NOT NULL,
    "position" integer NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Proposed',
    "page_ids" text[] NOT NULL,
    "suggested_type" varchar(50),
    "suggested_id" varchar(100),
    "suggested_doc_type_id" varchar(100),
    "suggestion_source" varchar(20),
    "suggestion_confidence" numeric(4, 3),
    "suggestion_reason" varchar(500),
    "cover_sheet_id" varchar(100),
    "detected_kind" varchar(50),
    "filed_type" varchar(50),
    "filed_id" varchar(100),
    "filed_doc_type_id" varchar(100),
    "document_id" varchar(100),
    "upload_session_id" varchar(100),
    "filed_by_id" varchar(100),
    "filed_at" bigint,
    "failure_message" varchar(500),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_capture_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_items_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_items_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_items_batch" FOREIGN KEY ("batch_id", "business_unit_id", "organization_id") REFERENCES "capture_batches"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_capture_items_status" CHECK ("status" IN ('Proposed', 'Filing', 'Filed', 'Discarded', 'Failed')),
    CONSTRAINT "ck_capture_items_suggestion_source" CHECK ("suggestion_source" IS NULL OR "suggestion_source" IN ('CoverSheet', 'Request', 'Classifier', 'Person')),
    CONSTRAINT "ck_capture_items_pages" CHECK (cardinality("page_ids") BETWEEN 1 AND 1000)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_capture_items_batch" ON "capture_items"("batch_id", "business_unit_id", "organization_id", "position");

--bun:split
-- Filing is idempotent on the item: the workflow that files it may run twice,
-- and a second run must find the document the first one made.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_items_document" ON "capture_items"("organization_id", "business_unit_id", "document_id") WHERE "document_id" IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS "capture_cover_sheets"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "token_hash" varchar(128) NOT NULL,
    "target_type" varchar(50),
    "target_id" varchar(100),
    "document_type_id" varchar(100),
    "issued_by_id" varchar(100) NOT NULL,
    "expires_at" bigint NOT NULL,
    "last_used_at" bigint,
    "use_count" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_capture_cover_sheets" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_capture_cover_sheets_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_capture_cover_sheets_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- A sheet resolves only inside the tenant whose stack it was scanned in, so
-- the lookup carries the tenant and a sheet copied into another organization's
-- stack finds nothing.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_capture_cover_sheets_token" ON "capture_cover_sheets"("organization_id", "business_unit_id", "token_hash");

--bun:split
ALTER TABLE "document_controls"
    ADD COLUMN IF NOT EXISTS "enable_capture" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "capture_auto_file_cover_sheets" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "capture_retention_days" integer NOT NULL DEFAULT 30,
    ADD COLUMN IF NOT EXISTS "capture_min_agent_version" varchar(20),
    ADD COLUMN IF NOT EXISTS "capture_allow_auto_update" boolean NOT NULL DEFAULT TRUE;

--bun:split
ALTER TABLE "document_controls"
    ADD CONSTRAINT "ck_document_controls_capture_retention_days" CHECK ("capture_retention_days" BETWEEN 1 AND 365);

COMMENT ON TABLE "capture_devices" IS 'Paired Trenova Capture installs; each acts for one person and can do no more than they can';

COMMENT ON TABLE "capture_batches" IS 'One acquisition from a scanner or the Trenova printer, kept until every page is filed or discarded';

COMMENT ON COLUMN "capture_cover_sheets"."token_hash" IS 'SHA-256 of the random token in the sheet''s QR code; the sheet names no record on paper';

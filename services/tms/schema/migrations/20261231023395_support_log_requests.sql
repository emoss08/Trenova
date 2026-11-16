-- An administrator of a managed instance asks for the instance's logs to be sent to Trenova
-- support for a period. The operator reads the requests that wait, collects the logs into a
-- bundle kept in the instance's own object storage for 30 days, and reports back. Who asked is
-- the asking organization's data, so the row belongs to that organization; the bounds hold for
-- the whole instance, because the logs and the operator are shared: one request in flight at a
-- time (the partial unique index below, which row-level security does not narrow) and at most a
-- few a day. A request the operator never picks up or never finishes is failed by a sweep.
CREATE TABLE IF NOT EXISTS "support_log_requests"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "requested_by_id" varchar(100) NOT NULL,
    "requester_name" varchar(255) NOT NULL,
    "requester_email" varchar(254) NOT NULL,
    "period" varchar(10) NOT NULL,
    "since" bigint NOT NULL,
    "until" bigint NOT NULL,
    "status" varchar(20) NOT NULL,
    "bundle_id" varchar(100),
    "failure_message" text,
    "collecting_at" bigint,
    "finished_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_support_log_requests" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_support_log_requests_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_log_requests_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_log_requests_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "ck_support_log_requests_period" CHECK ("period" IN ('hour', 'day', 'week')),
    CONSTRAINT "ck_support_log_requests_status" CHECK ("status" IN ('requested', 'collecting', 'sent', 'failed')),
    CONSTRAINT "ck_support_log_requests_window" CHECK ("until" > "since" AND "until" - "since" <= 7 * 86400),
    CONSTRAINT "ck_support_log_requests_requester" CHECK (length(btrim("requester_name")) > 0 AND length(btrim("requester_email")) > 0),
    CONSTRAINT "ck_support_log_requests_bundle" CHECK ("bundle_id" IS NULL OR "bundle_id" ~ '^[A-Za-z0-9_-]{1,100}$'),
    CONSTRAINT "ck_support_log_requests_sent" CHECK ("status" <> 'sent' OR "bundle_id" IS NOT NULL),
    CONSTRAINT "ck_support_log_requests_failure" CHECK (
        ("status" = 'failed' AND length(btrim("failure_message")) BETWEEN 1 AND 500)
        OR ("status" <> 'failed' AND "failure_message" IS NULL)
    ),
    CONSTRAINT "ck_support_log_requests_collecting" CHECK ("status" <> 'collecting' OR "collecting_at" IS NOT NULL),
    CONSTRAINT "ck_support_log_requests_finished" CHECK (("status" IN ('sent', 'failed')) = ("finished_at" IS NOT NULL))
);

COMMENT ON TABLE "support_log_requests" IS 'Requests by an administrator of a managed instance to send the instance''s logs for a period to Trenova support';

COMMENT ON COLUMN "support_log_requests"."requester_name" IS 'The name of the person who asked, as it was when they asked; the operator and Trenova staff are shown it';

COMMENT ON COLUMN "support_log_requests"."requester_email" IS 'The email address of the person who asked, as it was when they asked; the operator and Trenova staff are shown it';

COMMENT ON COLUMN "support_log_requests"."bundle_id" IS 'The id the control plane gave the log bundle when the operator registered it';

COMMENT ON COLUMN "support_log_requests"."failure_message" IS 'One sentence saying why the logs could not be sent, shown to the administrators';

CREATE UNIQUE INDEX IF NOT EXISTS "uq_support_log_requests_in_flight" ON "support_log_requests"((1))
WHERE "status" IN ('requested', 'collecting');

CREATE INDEX IF NOT EXISTS "idx_support_log_requests_tenant_created" ON "support_log_requests"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_support_log_requests_created" ON "support_log_requests"("created_at");

CREATE INDEX IF NOT EXISTS "idx_support_log_requests_in_flight_created" ON "support_log_requests"("created_at", "id")
WHERE "status" IN ('requested', 'collecting');

CREATE INDEX IF NOT EXISTS "idx_support_log_requests_collecting" ON "support_log_requests"("collecting_at")
WHERE "status" = 'collecting';

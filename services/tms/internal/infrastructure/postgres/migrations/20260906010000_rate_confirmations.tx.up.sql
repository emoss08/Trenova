CREATE TABLE IF NOT EXISTS "rate_confirmations"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "carrier_assignment_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "revision" integer NOT NULL DEFAULT 1,
    "status" varchar(50) NOT NULL DEFAULT 'Generated',
    "document_id" varchar(100),
    "sent_at" bigint,
    "sent_to_emails" text,
    "confirmed_at" bigint,
    "confirmed_by_name" varchar(255),
    "voided_at" bigint,
    "void_reason" text,
    "payload_snapshot" jsonb,
    "generated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_rate_confirmations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_confirmations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmations_carrier_assignment" FOREIGN KEY ("carrier_assignment_id", "organization_id", "business_unit_id") REFERENCES "carrier_assignments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmations_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_confirmations_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmations_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX "idx_rate_confirmations_active_assignment" ON "rate_confirmations"("carrier_assignment_id", "organization_id", "business_unit_id")
WHERE
    "status" <> 'Voided';

--bun:split
CREATE UNIQUE INDEX "idx_rate_confirmations_assignment_revision" ON "rate_confirmations"("carrier_assignment_id", "revision", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_rate_confirmations_shipment" ON "rate_confirmations"("shipment_id", "organization_id");

--bun:split
CREATE INDEX "idx_rate_confirmations_carrier" ON "rate_confirmations"("carrier_id", "organization_id");

--bun:split
COMMENT ON TABLE "rate_confirmations" IS 'Outbound carrier rate confirmations with revisioning and a frozen payload snapshot';

--bun:split
-- The rendered rate confirmation is filed as a shipment document, which needs a
-- document type to file under. Seeding covers new organizations only, so existing
-- ones are backfilled here the same way DETNOTICE was.
INSERT INTO document_types (
    id,
    business_unit_id,
    organization_id,
    code,
    name,
    description,
    color,
    document_classification,
    document_category,
    is_system
)
SELECT
    'dt_' || substr(md5(o.id || 'RATECON'), 1, 26),
    o.business_unit_id,
    o.id,
    'RATECON',
    'Rate Confirmation',
    'Carrier rate confirmation rendered as a PDF and filed against the shipment',
    '#0ea5e9',
    'Public',
    'Shipment',
    TRUE
FROM organizations o
ON CONFLICT DO NOTHING;

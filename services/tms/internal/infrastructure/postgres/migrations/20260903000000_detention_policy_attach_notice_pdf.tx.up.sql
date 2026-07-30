-- Attaching the notice PDF is a per-policy choice.
--
-- Some customers accept an emailed notice as sufficient evidence; others require
-- a document they can file against the claim. The flag defaults to FALSE so every
-- existing policy keeps sending exactly what it sends today.
ALTER TABLE "detention_policies"
    ADD COLUMN IF NOT EXISTS "attach_notice_pdf" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN "detention_policies"."attach_notice_pdf" IS 'Attach a rendered PDF of the notice to the customer email and file it as a shipment document';

--bun:split
-- The notice PDF is stored as a shipment document, which needs a document type
-- to file it under. Seeding only covers new organizations, so every existing one
-- is backfilled here; without it the first policy to turn the flag on would fail
-- to file its notice.
--
-- The id is derived from the organization id so re-running this against a
-- restored database cannot produce a second row, and the ON CONFLICT guards an
-- organization that already authored a DETNOTICE type by hand.
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
    'dt_' || substr(md5(o.id || 'DETNOTICE'), 1, 26),
    o.business_unit_id,
    o.id,
    'DETNOTICE',
    'Detention Notice',
    'Customer detention notice rendered as a PDF and filed against the shipment',
    '#f97316',
    'Public',
    'Shipment',
    TRUE
FROM organizations o
ON CONFLICT DO NOTHING;

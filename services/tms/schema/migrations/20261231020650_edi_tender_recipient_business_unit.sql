-- Internal EDI runs between organizations of one business unit: a tender's recipient, its changes,
-- the shipment link and the connection all live in the source's business unit, and keel scopes
-- edi_tender_recipients and edi_tender_changes by business_unit_id. Their source and recipient
-- business unit columns repeated business_unit_id, and the policies admitted a recipient row
-- stored in another business unit, which keel's tenant predicate hides and no writer produces.
-- The columns go, the source shipment key and the recipient key read business_unit_id, and the
-- policies read the business unit the way keel does, so a reader sees exactly what the policy
-- admits.

SELECT trenova_rls.apply_policy(
    'public.edi_tender_recipients',
    $expr$business_unit_id = (SELECT trenova_rls.bu_id())
        AND (SELECT trenova_rls.org_id()) IN (source_organization_id, recipient_organization_id)$expr$
);

SELECT trenova_rls.apply_policy(
    'public.edi_tender_changes',
    $expr$business_unit_id = (SELECT trenova_rls.bu_id())
        AND (
            source_organization_id = (SELECT trenova_rls.org_id())
            OR EXISTS (
                SELECT 1
                FROM public.edi_tender_recipients r
                WHERE r.id = edi_tender_changes.recipient_id
                    AND r.business_unit_id = edi_tender_changes.business_unit_id
                    AND r.recipient_organization_id = (SELECT trenova_rls.org_id())
            )
        )$expr$
);

ALTER TABLE "edi_tender_recipients"
    DROP CONSTRAINT "fk_edi_tender_recipients_source_shipment";

DROP INDEX "idx_edi_tender_recipients_unique_recipient";

DROP INDEX "idx_edi_tender_recipients_source";

ALTER TABLE "edi_tender_recipients"
    DROP COLUMN "source_business_unit_id",
    DROP COLUMN "recipient_business_unit_id";

ALTER TABLE "edi_tender_recipients"
    ADD CONSTRAINT "fk_edi_tender_recipients_source_shipment" FOREIGN KEY ("source_shipment_id", "business_unit_id", "source_organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE;

CREATE UNIQUE INDEX "idx_edi_tender_recipients_unique_recipient"
    ON "edi_tender_recipients"(
        "source_shipment_id",
        "business_unit_id",
        "source_organization_id",
        "recipient_kind",
        COALESCE("recipient_organization_id", ''),
        COALESCE("edi_partner_id", ''),
        COALESCE("partner_document_profile_id", '')
    );

CREATE INDEX "idx_edi_tender_recipients_source"
    ON "edi_tender_recipients"("business_unit_id", "source_organization_id", "source_shipment_id", "status");

DROP INDEX "idx_edi_tender_changes_source";

ALTER TABLE "edi_tender_changes"
    DROP COLUMN "source_business_unit_id";

CREATE INDEX "idx_edi_tender_changes_source"
    ON "edi_tender_changes"("business_unit_id", "source_organization_id", "source_shipment_id", "created_at" DESC);

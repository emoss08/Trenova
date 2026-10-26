-- Internal EDI runs between organizations of one business unit: a tender's recipient, its changes,
-- the shipment link and the connection all live in the source's business unit, and keel scopes
-- edi_tender_recipients and edi_tender_changes by business_unit_id. The policies below admitted a
-- recipient row stored in another business unit, which keel's tenant predicate hides and no writer
-- produces. The constraints make the single business unit a rule of the table and the policies
-- read it the way keel does, so a reader sees exactly what the policy admits.

ALTER TABLE "edi_tender_recipients"
    ADD CONSTRAINT "ck_edi_tender_recipients_one_business_unit" CHECK (
        "source_business_unit_id" = "business_unit_id"
        AND ("recipient_business_unit_id" IS NULL OR "recipient_business_unit_id" = "business_unit_id")
    );

ALTER TABLE "edi_tender_changes"
    ADD CONSTRAINT "ck_edi_tender_changes_one_business_unit" CHECK (
        "source_business_unit_id" = "business_unit_id"
    );

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

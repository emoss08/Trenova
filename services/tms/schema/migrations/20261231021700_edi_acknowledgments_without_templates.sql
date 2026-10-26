-- Trenova writes 997 and 999 acknowledgments itself, from the reading of what it received
-- (pkg/x12, the trenova-edi crate), so an acknowledgment profile has no template and the
-- starter acknowledgment templates go.
ALTER TABLE "edi_partner_document_profiles"
    ALTER COLUMN "template_id" DROP NOT NULL;

UPDATE "edi_partner_document_profiles"
SET "template_id" = NULL,
    "template_version_id" = NULL,
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
WHERE "transaction_set" IN ('997', '999');

UPDATE "edi_messages"
SET "template_id" = NULL,
    "template_version_id" = NULL
WHERE "transaction_set" IN ('997', '999')
    AND "template_id" IS NOT NULL;

DELETE FROM "edi_templates"
WHERE "transaction_set" IN ('997', '999');

ALTER TABLE "edi_partner_document_profiles"
    ADD CONSTRAINT "ck_edi_partner_document_profiles_template" CHECK (
        ("transaction_set" IN ('997', '999')) = ("template_id" IS NULL)
    );

-- The X12 dictionaries live in the crate's guides now; these tables were never read.
DROP TABLE IF EXISTS
    "edi_transaction_element_definitions",
    "edi_code_list_definitions",
    "edi_transaction_segment_definitions",
    "edi_transaction_loop_definitions";

DELETE FROM trenova_rls.global_tables
WHERE "table_name" IN (
    'edi_transaction_element_definitions',
    'edi_code_list_definitions',
    'edi_transaction_segment_definitions',
    'edi_transaction_loop_definitions'
);

-- The starter 210 put the net amount due in B306 and the currency in B311, where X12 has the
-- invoice date and the SCAC, and wrote N2 charges with a decimal point. A starter segment
-- nobody changed is corrected; one a person changed is theirs and stays as it is.

-- B3: date in B306, amount in implied cents in B307, SCAC in B311.
UPDATE "edi_template_segments" AS "s"
SET "elements" = jsonb_set(jsonb_set(jsonb_set("s"."elements", '{3}', '{"position":6,"name":"Date","source":"transform","value":"","fieldPath":"","partnerSettingPath":"","mappingSourcePath":"","runtimeKey":"","repeatPath":"","baseSource":{"source":"fieldPath","fieldPath":"invoice.invoiceDate"},"transformPipeline":[{"operation":"format_date"}],"starlarkFunction":"","starlarkScript":"","default":"","condition":"","validation":{"required":true,"maxLength":0,"minLength":0,"code":"required","message":"Date is required"},"implementationGuideNote":""}'::jsonb), '{4}', '{"position":7,"name":"Net Amount Due","source":"transform","value":"","fieldPath":"","partnerSettingPath":"","mappingSourcePath":"","runtimeKey":"","repeatPath":"","baseSource":{"source":"fieldPath","fieldPath":"invoice.totalAmount"},"transformPipeline":[{"operation":"format_decimal","arguments":{"implied":true,"places":2}}],"starlarkFunction":"","starlarkScript":"","default":"","condition":"","validation":{"required":true,"maxLength":0,"minLength":0,"code":"required","message":"Net Amount Due is required"},"implementationGuideNote":""}'::jsonb), '{5}', '{"position":11,"name":"Standard Carrier Alpha Code","source":"partnerSetting","value":"","fieldPath":"","partnerSettingPath":"carrier.scac","mappingSourcePath":"","runtimeKey":"","repeatPath":"","starlarkFunction":"","starlarkScript":"","default":"","condition":"","validation":{"required":false,"maxLength":0,"minLength":0,"code":"required","message":"Standard Carrier Alpha Code is required"},"implementationGuideNote":""}'::jsonb),
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
FROM "edi_template_versions" AS "v"
JOIN "edi_templates" AS "t"
    ON "t"."id" = "v"."template_id"
    AND "t"."business_unit_id" = "v"."business_unit_id"
    AND "t"."organization_id" = "v"."organization_id"
WHERE "v"."id" = "s"."template_version_id"
    AND "v"."business_unit_id" = "s"."business_unit_id"
    AND "v"."organization_id" = "s"."organization_id"
    AND "t"."transaction_set" = '210'
    AND "t"."direction" = 'Outbound'
    AND "s"."segment_id" = 'B3'
    AND jsonb_array_length("s"."elements") = 6
    AND "s"."elements" -> 3 ->> 'position' = '6'
    AND "s"."elements" -> 3 ->> 'source' = 'fieldPath'
    AND "s"."elements" -> 3 ->> 'fieldPath' = 'invoice.totalAmount'
    AND "s"."elements" -> 4 ->> 'position' = '7'
    AND "s"."elements" -> 4 ->> 'source' = 'constant'
    AND "s"."elements" -> 4 ->> 'value' = ''
    AND "s"."elements" -> 5 ->> 'position' = '11'
    AND "s"."elements" -> 5 ->> 'source' = 'fieldPath'
    AND "s"."elements" -> 5 ->> 'fieldPath' = 'invoice.currencyCode';

-- G62: the invoice date as CCYYMMDD.
UPDATE "edi_template_segments" AS "s"
SET "elements" = jsonb_set("s"."elements", '{1}', '{"position":2,"name":"Invoice Date","source":"transform","value":"","fieldPath":"","partnerSettingPath":"","mappingSourcePath":"","runtimeKey":"","repeatPath":"","baseSource":{"source":"fieldPath","fieldPath":"invoice.invoiceDate"},"transformPipeline":[{"operation":"format_date"}],"starlarkFunction":"","starlarkScript":"","default":"","condition":"","validation":{"required":false,"maxLength":0,"minLength":0,"code":"required","message":"Invoice Date is required"},"implementationGuideNote":""}'::jsonb),
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
FROM "edi_template_versions" AS "v"
JOIN "edi_templates" AS "t"
    ON "t"."id" = "v"."template_id"
    AND "t"."business_unit_id" = "v"."business_unit_id"
    AND "t"."organization_id" = "v"."organization_id"
WHERE "v"."id" = "s"."template_version_id"
    AND "v"."business_unit_id" = "s"."business_unit_id"
    AND "v"."organization_id" = "s"."organization_id"
    AND "t"."transaction_set" = '210'
    AND "t"."direction" = 'Outbound'
    AND "s"."segment_id" = 'G62'
    AND jsonb_array_length("s"."elements") = 2
    AND "s"."elements" -> 1 ->> 'position' = '2'
    AND "s"."elements" -> 1 ->> 'source' = 'fieldPath'
    AND "s"."elements" -> 1 ->> 'fieldPath' = 'invoice.invoiceDate';

-- L1: the charge in implied cents.
UPDATE "edi_template_segments" AS "s"
SET "elements" = jsonb_set("s"."elements", '{1}', '{"position":4,"name":"Charge","source":"transform","value":"","fieldPath":"","partnerSettingPath":"","mappingSourcePath":"","runtimeKey":"","repeatPath":"","baseSource":{"source":"repeat","repeatPath":"amount"},"transformPipeline":[{"operation":"format_decimal","arguments":{"places":2,"implied":true}}],"starlarkFunction":"","starlarkScript":"","default":"","condition":"","validation":{"required":false,"maxLength":0,"minLength":0,"code":"required","message":"Charge is required"},"implementationGuideNote":""}'::jsonb),
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
FROM "edi_template_versions" AS "v"
JOIN "edi_templates" AS "t"
    ON "t"."id" = "v"."template_id"
    AND "t"."business_unit_id" = "v"."business_unit_id"
    AND "t"."organization_id" = "v"."organization_id"
WHERE "v"."id" = "s"."template_version_id"
    AND "v"."business_unit_id" = "s"."business_unit_id"
    AND "v"."organization_id" = "s"."organization_id"
    AND "t"."transaction_set" = '210'
    AND "t"."direction" = 'Outbound'
    AND "s"."segment_id" = 'L1'
    AND jsonb_array_length("s"."elements") = 3
    AND "s"."elements" -> 1 ->> 'position' = '4'
    AND "s"."elements" -> 1 ->> 'source' = 'repeat'
    AND "s"."elements" -> 1 ->> 'repeatPath' = 'amount';

-- L3: the charge in implied cents.
UPDATE "edi_template_segments" AS "s"
SET "elements" = jsonb_set("s"."elements", '{0}', '{"position":5,"name":"Charge","source":"transform","value":"","fieldPath":"","partnerSettingPath":"","mappingSourcePath":"","runtimeKey":"","repeatPath":"","baseSource":{"source":"fieldPath","fieldPath":"invoice.totalAmount"},"transformPipeline":[{"operation":"format_decimal","arguments":{"places":2,"implied":true}}],"starlarkFunction":"","starlarkScript":"","default":"","condition":"","validation":{"required":true,"maxLength":0,"minLength":0,"code":"required","message":"Charge is required"},"implementationGuideNote":""}'::jsonb),
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
FROM "edi_template_versions" AS "v"
JOIN "edi_templates" AS "t"
    ON "t"."id" = "v"."template_id"
    AND "t"."business_unit_id" = "v"."business_unit_id"
    AND "t"."organization_id" = "v"."organization_id"
WHERE "v"."id" = "s"."template_version_id"
    AND "v"."business_unit_id" = "s"."business_unit_id"
    AND "v"."organization_id" = "s"."organization_id"
    AND "t"."transaction_set" = '210'
    AND "t"."direction" = 'Outbound'
    AND "s"."segment_id" = 'L3'
    AND jsonb_array_length("s"."elements") = 1
    AND "s"."elements" -> 0 ->> 'position' = '5'
    AND "s"."elements" -> 0 ->> 'source' = 'fieldPath'
    AND "s"."elements" -> 0 ->> 'fieldPath' = 'invoice.totalAmount';

-- The starter 204's L305 charge, in implied cents.
UPDATE "edi_template_segments" AS "s"
SET "elements" = jsonb_set("s"."elements", '{2}', '{"position":5,"name":"Charge","source":"transform","value":"","fieldPath":"","partnerSettingPath":"","mappingSourcePath":"","runtimeKey":"","repeatPath":"","baseSource":{"source":"fieldPath","fieldPath":"totalChargeAmount"},"transformPipeline":[{"operation":"format_decimal","arguments":{"places":2,"implied":true}}],"starlarkFunction":"","starlarkScript":"","default":"","condition":"","validation":{"required":false,"maxLength":0,"minLength":0,"code":"required","message":"Charge is required"},"implementationGuideNote":""}'::jsonb),
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
FROM "edi_template_versions" AS "v"
JOIN "edi_templates" AS "t"
    ON "t"."id" = "v"."template_id"
    AND "t"."business_unit_id" = "v"."business_unit_id"
    AND "t"."organization_id" = "v"."organization_id"
WHERE "v"."id" = "s"."template_version_id"
    AND "v"."business_unit_id" = "s"."business_unit_id"
    AND "v"."organization_id" = "s"."organization_id"
    AND "t"."transaction_set" = '204'
    AND "t"."direction" = 'Outbound'
    AND "s"."segment_id" = 'L3'
    AND jsonb_array_length("s"."elements") = 3
    AND "s"."elements" -> 2 ->> 'position' = '5'
    AND "s"."elements" -> 2 ->> 'source' = 'fieldPath'
    AND "s"."elements" -> 2 ->> 'fieldPath' = 'totalChargeAmount';

-- The starter 204 template wrote B2 one element early: the SCAC in B201, the
-- shipment identification number in B202 and the method of payment in B204.
-- X12 puts them in B202, B204 and B206. Only a B2 that is still exactly the
-- starter's is moved; one a person changed is theirs and stays as it is.
UPDATE "edi_template_segments" AS "s"
SET "elements" = jsonb_build_array(
        jsonb_set("s"."elements" -> 0, '{position}', '2'::jsonb),
        jsonb_set("s"."elements" -> 1, '{position}', '4'::jsonb),
        jsonb_set("s"."elements" -> 2, '{position}', '6'::jsonb)
    ),
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
    AND "s"."segment_id" = 'B2'
    AND jsonb_array_length("s"."elements") = 3
    AND "s"."elements" -> 0 ->> 'position' = '1'
    AND "s"."elements" -> 0 ->> 'source' = 'partnerSetting'
    AND "s"."elements" -> 0 ->> 'partnerSettingPath' = 'carrier.scac'
    AND "s"."elements" -> 1 ->> 'position' = '2'
    AND "s"."elements" -> 1 ->> 'source' = 'fieldPath'
    AND "s"."elements" -> 1 ->> 'fieldPath' = 'shipmentId'
    AND "s"."elements" -> 2 ->> 'position' = '4'
    AND "s"."elements" -> 2 ->> 'source' = 'fieldPath'
    AND "s"."elements" -> 2 ->> 'fieldPath' = 'ratingDetail.paymentMethod';

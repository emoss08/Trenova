DO $$
DECLARE
    duplicate_pairs text;
BEGIN
    SELECT string_agg(
        format('(%s, %s): profiles %s', local_id, partner_id, profile_ids),
        '; '
    )
    INTO duplicate_pairs
    FROM (
        SELECT
            TRIM("config"->>'localAS2Id') AS local_id,
            TRIM("config"->>'partnerAS2Id') AS partner_id,
            string_agg("id", ', ' ORDER BY "id") AS profile_ids
        FROM "edi_communication_profiles"
        WHERE "method" = 'AS2'
          AND "status" = 'Active'
          AND "edi_partner_id" IS NOT NULL
          AND COALESCE(TRIM("config"->>'localAS2Id'), '') <> ''
          AND COALESCE(TRIM("config"->>'partnerAS2Id'), '') <> ''
        GROUP BY 1, 2
        HAVING count(*) > 1
    ) AS duplicates;

    IF duplicate_pairs IS NOT NULL THEN
        RAISE EXCEPTION 'Active AS2 profiles share an AS2 identifier pair, so inbound messages cannot be routed to one tenant: %. Deactivate or re-identify all but one profile per pair, then rerun the migration.', duplicate_pairs;
    END IF;
END
$$;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_edi_communication_profiles_active_as2_identifiers"
    ON "edi_communication_profiles"(
        TRIM("config"->>'localAS2Id'),
        TRIM("config"->>'partnerAS2Id')
    )
    WHERE "method" = 'AS2'
      AND "status" = 'Active'
      AND "edi_partner_id" IS NOT NULL
      AND COALESCE(TRIM("config"->>'localAS2Id'), '') <> ''
      AND COALESCE(TRIM("config"->>'partnerAS2Id'), '') <> '';

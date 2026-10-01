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

--keel:notx
-- The inbound address resolves one profile before its tenant is known.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_edi_communication_profiles_as2_inbound_token" ON "edi_communication_profiles"("as2_inbound_token");

--keel:split
-- The address names the tenant, so an AS2 identifier pair only has to be unique inside one: two
-- tenants may trade with the same partner under the same identifiers.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_edi_communication_profiles_tenant_as2_identifiers" ON "edi_communication_profiles"(
    "organization_id",
    "business_unit_id",
    TRIM("config"->>'localAS2Id'),
    TRIM("config"->>'partnerAS2Id')
)
WHERE
    "method" = 'AS2'
    AND "status" = 'Active'
    AND "edi_partner_id" IS NOT NULL
    AND COALESCE(TRIM("config"->>'localAS2Id'), '') <> ''
    AND COALESCE(TRIM("config"->>'partnerAS2Id'), '') <> '';

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "uq_edi_communication_profiles_active_as2_identifiers";

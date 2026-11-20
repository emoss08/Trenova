-- Detention recalculation reads a shipment's candidate policies once: the active policies scoped
-- to its customer or to none, and to one of its stops' facilities or to none. The read spells that
-- scope as four exact arms, and this index answers each arm, tenant first, so a tenant with
-- thousands of facility or customer terms reads only the ones that can apply rather than every
-- active policy through idx_detention_policies_resolution.
CREATE INDEX IF NOT EXISTS "idx_detention_policies_resolution_scope" ON "detention_policies"("organization_id", "business_unit_id", "customer_id", "location_id")
WHERE
    "status" = 'Active';

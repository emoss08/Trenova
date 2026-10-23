CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_accessorial_charges_code" ON "accessorial_charges"("organization_id", "business_unit_id", upper("code"));

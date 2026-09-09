--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "tractors"
    ADD COLUMN IF NOT EXISTS "fuel_type" ifta_fuel_type_enum NOT NULL DEFAULT 'Diesel',
    ADD COLUMN IF NOT EXISTS "ifta_qualified" boolean NOT NULL DEFAULT TRUE;

--bun:split
COMMENT ON COLUMN "tractors"."fuel_type" IS 'The fuel this tractor burns. IFTA computes fleet MPG per fuel type, so every mile the tractor drives is attributed to this fuel type on the return.';

--bun:split
COMMENT ON COLUMN "tractors"."ifta_qualified" IS 'Whether the tractor is an IFTA qualified motor vehicle. Miles and fuel on a non-qualified tractor are excluded from the return and reported as non-qualified activity.';

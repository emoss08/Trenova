DELETE FROM "accessorial_charges"
WHERE "code" IN ('PERMIT', 'ESCORT', 'TARP', 'SECURE');

--bun:split
DELETE FROM "hold_reasons"
WHERE "code" = 'PERMIT_REQUIRED';

--bun:split
DROP TABLE IF EXISTS "permit_requirements";

--bun:split
DROP TABLE IF EXISTS "permits";

--bun:split
DROP TYPE IF EXISTS "permit_requirement_status_enum";

--bun:split
DROP TYPE IF EXISTS "permit_status_enum";

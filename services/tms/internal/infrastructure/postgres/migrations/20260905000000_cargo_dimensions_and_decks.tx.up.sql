CREATE TYPE "equipment_deck_type_enum" AS ENUM(
    'Flatbed',
    'StepDeck',
    'DoubleDrop',
    'RGN',
    'Lowboy',
    'Conestoga'
);

--bun:split
ALTER TABLE "equipment_types"
    ADD COLUMN IF NOT EXISTS "deck_type" equipment_deck_type_enum,
    ADD COLUMN IF NOT EXISTS "deck_height_inches" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "well_length_feet" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "axle_count" smallint;

--bun:split
ALTER TABLE "equipment_types"
    ADD CONSTRAINT "chk_equipment_types_deck_height" CHECK ("deck_height_inches" IS NULL OR ("deck_height_inches" > 0 AND "deck_height_inches" <= 120)),
    ADD CONSTRAINT "chk_equipment_types_well_length" CHECK ("well_length_feet" IS NULL OR "well_length_feet" > 0),
    ADD CONSTRAINT "chk_equipment_types_well_requires_deck" CHECK ("well_length_feet" IS NULL OR "deck_type" IN ('DoubleDrop', 'RGN', 'Lowboy')),
    ADD CONSTRAINT "chk_equipment_types_axle_count" CHECK ("axle_count" IS NULL OR ("axle_count" >= 1 AND "axle_count" <= 20));

--bun:split
COMMENT ON COLUMN equipment_types.deck_height_inches IS 'Loaded deck height above ground; overall load height is this plus cargo height, and that sum is what decides whether a load clears without a height permit';

--bun:split
ALTER TABLE "commodities"
    ADD COLUMN IF NOT EXISTS "length_per_unit" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "width_per_unit" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "height_per_unit" numeric(10, 2);

--bun:split
ALTER TABLE "commodities"
    ADD CONSTRAINT "chk_commodities_dimensions_positive" CHECK (("length_per_unit" IS NULL OR "length_per_unit" > 0) AND ("width_per_unit" IS NULL OR "width_per_unit" > 0) AND ("height_per_unit" IS NULL OR "height_per_unit" > 0));

--bun:split
COMMENT ON COLUMN commodities.length_per_unit IS 'Default per-unit dimensions in feet; a shipment commodity line may override them for a specific configuration';

--bun:split
ALTER TABLE "shipment_commodities"
    ADD COLUMN IF NOT EXISTS "length_feet" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "width_feet" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "height_feet" numeric(10, 2);

--bun:split
ALTER TABLE "shipment_commodities"
    ADD CONSTRAINT "chk_shipment_commodities_dimensions_positive" CHECK (("length_feet" IS NULL OR "length_feet" > 0) AND ("width_feet" IS NULL OR "width_feet" > 0) AND ("height_feet" IS NULL OR "height_feet" > 0));

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "envelope_length_feet" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "envelope_width_feet" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "envelope_height_feet" numeric(10, 2),
    ADD COLUMN IF NOT EXISTS "envelope_overall_height_feet" numeric(10, 2);

--bun:split
ALTER TABLE "shipments"
    ADD CONSTRAINT "chk_shipments_envelope_positive" CHECK (("envelope_length_feet" IS NULL OR "envelope_length_feet" > 0) AND ("envelope_width_feet" IS NULL OR "envelope_width_feet" > 0) AND ("envelope_height_feet" IS NULL OR "envelope_height_feet" > 0) AND ("envelope_overall_height_feet" IS NULL OR "envelope_overall_height_feet" > 0));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_envelope_oversize" ON "shipments"("organization_id", "business_unit_id", "envelope_width_feet" DESC, "envelope_overall_height_feet" DESC)
WHERE
    "envelope_width_feet" IS NOT NULL;

--bun:split
COMMENT ON COLUMN shipments.envelope_width_feet IS 'Derived from the cargo lines on every save, never entered directly; the travel envelope is what jurisdiction permit thresholds are evaluated against';

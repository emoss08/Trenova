CREATE TYPE "fuel_type_enum" AS ENUM('Diesel', 'Gasoline');

--bun:split
ALTER TABLE "fuel_indices"
    ADD COLUMN IF NOT EXISTS "fuel_type" fuel_type_enum NOT NULL DEFAULT 'Diesel',
    ADD COLUMN IF NOT EXISTS "region" varchar(100);

--bun:split
UPDATE "fuel_indices"
SET "region" = CASE "eia_series_id"
    WHEN 'EMD_EPD2D_PTE_NUS_DPG' THEN 'US'
    WHEN 'EMD_EPD2D_PTE_R10_DPG' THEN 'PADD 1'
    WHEN 'EMD_EPD2D_PTE_R1X_DPG' THEN 'PADD 1A'
    WHEN 'EMD_EPD2D_PTE_R1Y_DPG' THEN 'PADD 1B'
    WHEN 'EMD_EPD2D_PTE_R1Z_DPG' THEN 'PADD 1C'
    WHEN 'EMD_EPD2D_PTE_R20_DPG' THEN 'PADD 2'
    WHEN 'EMD_EPD2D_PTE_R30_DPG' THEN 'PADD 3'
    WHEN 'EMD_EPD2D_PTE_R40_DPG' THEN 'PADD 4'
    WHEN 'EMD_EPD2D_PTE_R50_DPG' THEN 'PADD 5'
    WHEN 'EMD_EPD2D_PTE_R5XCA_DPG' THEN 'PADD 5 (excl. CA)'
    WHEN 'EMD_EPD2D_PTE_SCA_DPG' THEN 'California'
    ELSE "region"
END
WHERE "source" = 'EIA' AND "region" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_indices_region" ON "fuel_indices"("organization_id", "business_unit_id", "region");

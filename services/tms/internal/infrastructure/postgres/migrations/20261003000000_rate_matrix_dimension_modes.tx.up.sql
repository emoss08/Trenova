--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "rate_matrix_key_normalization_enum" AS ENUM(
    'None',
    'Trim',
    'Upper',
    'Zip3'
);

--bun:split
CREATE TYPE "rate_matrix_range_overflow_enum" AS ENUM(
    'Error',
    'ClampToTopBand',
    'Nearest'
);

--bun:split
ALTER TABLE "rate_matrix_dimensions"
    ADD COLUMN IF NOT EXISTS "key_normalization" rate_matrix_key_normalization_enum NOT NULL DEFAULT 'None',
    ADD COLUMN IF NOT EXISTS "range_overflow" rate_matrix_range_overflow_enum NOT NULL DEFAULT 'Error';

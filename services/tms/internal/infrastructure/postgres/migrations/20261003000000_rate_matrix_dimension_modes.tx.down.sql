--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "rate_matrix_dimensions"
    DROP COLUMN IF EXISTS "key_normalization",
    DROP COLUMN IF EXISTS "range_overflow";

--bun:split
DROP TYPE IF EXISTS "rate_matrix_range_overflow_enum";

--bun:split
DROP TYPE IF EXISTS "rate_matrix_key_normalization_enum";

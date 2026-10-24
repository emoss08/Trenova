--keel:notx
-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it. A ZipRange axis
-- holds inclusive postal-code ranges, such as 30000 to 31999.
ALTER TYPE "rate_matrix_dimension_kind_enum" ADD VALUE IF NOT EXISTS 'ZipRange';

-- A bare-number axis, for matrices addressed from a formula's lookup() call.
-- The expression supplies the quantity itself, so unlike WeightBreak or
-- Distance there is no shipment fact behind it — which is exactly what a
-- migrated range rate table needs, because a table never said what its key
-- meant. Added in its own migration: a value appended to an enum cannot be
-- used by the transaction that added it.
ALTER TYPE "rate_matrix_dimension_kind_enum" ADD VALUE IF NOT EXISTS 'Quantity';

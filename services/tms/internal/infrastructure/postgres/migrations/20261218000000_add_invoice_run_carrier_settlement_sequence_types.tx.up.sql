ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'carrier_settlement';

--bun:split
ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'invoice_run';

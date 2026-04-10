ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'credit_memo';

--bun:split
ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'debit_memo';

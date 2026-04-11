DELETE FROM "sequence_configs"
WHERE "sequence_type" IN ('journal_batch', 'journal_entry', 'manual_journal_request');

--bun:split
DELETE FROM "sequences"
WHERE "sequence_type" IN ('journal_batch', 'journal_entry', 'manual_journal_request');

--bun:split
ALTER TYPE "sequence_type_enum" RENAME TO "sequence_type_enum_old";

--bun:split
CREATE TYPE "sequence_type_enum" AS ENUM(
    'pro_number',
    'consolidation',
    'invoice',
    'work_order',
    'credit_memo',
    'debit_memo'
);

--bun:split
ALTER TABLE "sequences"
    ALTER COLUMN "sequence_type" TYPE "sequence_type_enum"
    USING "sequence_type"::text::"sequence_type_enum";

--bun:split
ALTER TABLE "sequence_configs"
    ALTER COLUMN "sequence_type" TYPE "sequence_type_enum"
    USING "sequence_type"::text::"sequence_type_enum";

--bun:split
DROP TYPE "sequence_type_enum_old";

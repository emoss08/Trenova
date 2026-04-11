ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'journal_batch';

--bun:split
ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'journal_entry';

--bun:split
ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'manual_journal_request';

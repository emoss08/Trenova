--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- New enum values live in their own migration: Postgres refuses to use a value
-- added by ALTER TYPE inside the same transaction.
ALTER TYPE "journal_entry_type_enum"
    ADD VALUE IF NOT EXISTS 'Opening';

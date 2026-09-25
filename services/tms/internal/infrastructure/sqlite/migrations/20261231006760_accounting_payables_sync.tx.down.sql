-- Hand-written: the narrower checks are not restored. Records and mappings
-- written under the wider ones would fail them, and a rollback that deletes
-- sync history is worse than checks that allow types the code no longer sends.
-- Source: 20261231006760_accounting_payables_sync.tx.down.sql

SELECT 1;

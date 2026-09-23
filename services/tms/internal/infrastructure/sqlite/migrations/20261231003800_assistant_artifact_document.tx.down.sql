-- Hand-written: the narrower kind check is not restored. Rows written under
-- the wider one would fail it; the application stops producing documents
-- once the code that writes them is rolled back.
-- Source: 20261231003800_assistant_artifact_document.tx.down.sql

SELECT 1;

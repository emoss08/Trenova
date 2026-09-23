-- Hand-written: the narrower kind check is not restored. Rows written under
-- the wider one would fail it; the application stops producing navigation
-- artifacts once the code that writes them is rolled back.
-- Source: 20261231004100_assistant_artifact_navigation.tx.down.sql

SELECT 1;

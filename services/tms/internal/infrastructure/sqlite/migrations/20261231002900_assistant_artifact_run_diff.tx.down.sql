-- The CHECK is not put back. Restoring it would reject every run_diff
-- artifact written while it was gone, which is a migration that fails on data
-- it created itself.
SELECT 1;

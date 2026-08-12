DROP TABLE IF EXISTS "rate_confirmations";

--bun:split
DELETE FROM document_types dt
WHERE dt.id = 'dt_' || substr(md5(dt.organization_id || 'RATECON'), 1, 26)
    AND dt.code = 'RATECON'
    AND dt.is_system = TRUE;

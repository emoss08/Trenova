UPDATE "carrier_settlement_controls"
SET "auto_accept_within_tolerance" = FALSE,
    "version" = "version" + 1,
    "updated_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT
WHERE "auto_accept_within_tolerance"
    AND NOT "auto_match_inbound_invoices";

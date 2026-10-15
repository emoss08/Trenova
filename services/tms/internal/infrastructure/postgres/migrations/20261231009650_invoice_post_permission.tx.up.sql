-- Posting an invoice to the ledger was checked as invoice update, so anyone who
-- could edit an invoice could post it. It is now its own operation, invoice
-- post. Every role that could post before keeps posting: a grant holding update,
-- or an operation that brings update with it, gains post. An administrator then
-- removes it from the roles that should prepare invoices without posting them.
UPDATE "resource_permissions"
SET "operations" = array_append("operations", 'post'),
    "updated_at" = EXTRACT(EPOCH FROM current_timestamp)::bigint
WHERE "resource" = 'invoice'
  AND "operations" && ARRAY['update', 'approve', 'reject', 'submit', 'cancel', 'manage']::text[]
  AND NOT ('post' = ANY("operations"));

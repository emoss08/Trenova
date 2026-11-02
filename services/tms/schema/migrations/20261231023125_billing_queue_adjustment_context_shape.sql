-- A queue item's adjustment context becomes two typed parts: the rebill a credit-and-rebill
-- prices (its replacement lines, subtotals, total and accounting date) and the memo an item
-- bills (its kind and who raised it). The flat keys they replace go, with the adjustment kind
-- and the source, group and adjustment IDs, which nothing read and the item's own columns hold.
-- Only adjustment and memo items carry a context, so the update reads that handful of rows.
UPDATE "billing_queue_items"
SET "adjustment_context" = jsonb_strip_nulls(jsonb_build_object(
    'rebill', CASE
        WHEN "adjustment_context" ? 'replacementLines' OR "adjustment_context" ? 'totalAmount' THEN
            jsonb_build_object(
                'replacementLines', COALESCE("adjustment_context" -> 'replacementLines', '[]'::jsonb),
                'subtotalAmount', COALESCE("adjustment_context" -> 'subtotalAmount', '"0"'::jsonb),
                'otherAmount', COALESCE("adjustment_context" -> 'otherAmount', '"0"'::jsonb),
                'totalAmount', COALESCE("adjustment_context" -> 'totalAmount', '"0"'::jsonb),
                'accountingDate', COALESCE("adjustment_context" -> 'accountingDate', '0'::jsonb))
    END,
    'memo', CASE
        WHEN "adjustment_context" ? 'memoKind' THEN
            jsonb_build_object(
                'kind', "adjustment_context" -> 'memoKind',
                'raisedById', "adjustment_context" -> 'raisedById')
    END))
WHERE
    "adjustment_context" <> '{}'::jsonb;

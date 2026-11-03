-- A customer update's comment said what was emailed under "tool", which named the agent's
-- email tool for one kind and a name no tool has (notify_shipment_delay) for the delay notice
-- a dispatcher sends from the board. It is now "update", the kind of update, beside the
-- comment's origin under "source": email for the status email an agent writes, delay_notice
-- for the board's delay notice.
UPDATE "shipment_comments"
SET "metadata" = ("metadata" - 'tool') || jsonb_build_object(
    'update',
    CASE "metadata" ->> 'tool'
        WHEN 'customer_send_email' THEN 'email'
        WHEN 'notify_shipment_delay' THEN 'delay_notice'
        ELSE "metadata" ->> 'tool'
    END)
WHERE "metadata" ? 'tool';
